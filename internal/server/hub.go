package server

import (
	"crypto/rand"
	"log"
	"math/big"
	"net/http"
	"sync"
	"time"

	"github.com/PlasmolysisMango/Gonopoly/pkg/protocol"
	"nhooyr.io/websocket"
)

// Hub 是服务端的中心管理器，负责管理所有连接和房间。
type Hub struct {
	rooms      map[string]*Room
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
	storage    Storage
}

// NewHub 创建一个新的 Hub。storage 为 nil 时禁用持久化。
func NewHub(storage Storage) *Hub {
	h := &Hub{
		rooms:      make(map[string]*Room),
		clients:    make(map[*Client]bool),
		register:   make(chan *Client, 16),
		unregister: make(chan *Client, 16),
		storage:    storage,
	}
	if storage != nil {
		h.loadSavedRooms()
	}
	return h
}

// Run 主循环，处理客户端注册与注销。
func (h *Hub) Run() {
	for {
		select {
		case c := <-h.register:
			h.mu.Lock()
			h.clients[c] = true
			h.mu.Unlock()
			log.Printf("client registered: %p", c)
		case c := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[c]; ok {
				delete(h.clients, c)
				close(c.Send)
			}
			h.mu.Unlock()
			h.LeaveRoomOrSpectate(c)
			log.Printf("client unregistered: %p", c)
		}
	}
}

// HandleWebSocket 处理 WebSocket 升级与连接生命周期。
func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		log.Printf("websocket accept error: %v", err)
		return
	}

	client := &Client{
		Hub:       h,
		Conn:      conn,
		Send:      make(chan []byte, 64),
		Connected: true,
	}

	h.register <- client

	go client.WritePump()
	client.ReadPump()
	h.unregister <- client
}

// CreateRoom 处理创建房间请求。
func (h *Hub) CreateRoom(client *Client, msg protocol.C2SCreateRoom) {
	if msg.PlayerName == "" {
		client.SendError("invalid_name", "玩家名不能为空")
		return
	}

	if client.RoomID != "" {
		h.LeaveRoom(client)
	}

	roomID := h.generateRoomID()
	room := NewRoom(roomID)

	h.mu.Lock()
	h.rooms[roomID] = room
	h.mu.Unlock()

	client.PlayerName = msg.PlayerName
	client.CharName = msg.CharName
	client.RoomID = roomID
	client.Token = generateToken()

	if err := room.AddClient(client); err != nil {
		h.mu.Lock()
		delete(h.rooms, roomID)
		h.mu.Unlock()
		client.SendError("create_failed", err.Error())
		return
	}

	room.BroadcastRoomInfo()
	log.Printf("room %s created by %s", roomID, msg.PlayerName)
}

// JoinRoom 处理加入房间请求。
func (h *Hub) JoinRoom(client *Client, msg protocol.C2SJoinRoom) {
	if msg.PlayerName == "" {
		client.SendError("invalid_name", "玩家名不能为空")
		return
	}

	h.mu.RLock()
	room, ok := h.rooms[msg.RoomID]
	h.mu.RUnlock()
	if !ok {
		client.SendError("room_not_found", "房间不存在")
		return
	}

	if client.RoomID != "" && client.RoomID != msg.RoomID {
		h.LeaveRoom(client)
	}

	client.PlayerName = msg.PlayerName
	client.CharName = msg.CharName
	client.RoomID = msg.RoomID
	if client.Token == "" {
		client.Token = generateToken()
	}

	if err := room.AddClient(client); err != nil {
		client.SendError("join_failed", err.Error())
		return
	}

	joined, _ := protocol.NewMessage(protocol.TypePlayerJoined, protocol.S2CPlayerJoined{
		Name:     client.PlayerName,
		CharName: client.CharName,
	})
	room.BroadcastExcept(joined, client.PlayerName)

	room.BroadcastRoomInfo()
	log.Printf("player %s joined room %s", msg.PlayerName, msg.RoomID)
}

// LeaveRoomOrSpectate 根据客户端身份（玩家/观众）清理。
func (h *Hub) LeaveRoomOrSpectate(client *Client) {
	if client == nil || client.RoomID == "" {
		return
	}
	h.mu.RLock()
	room, ok := h.rooms[client.RoomID]
	h.mu.RUnlock()
	if ok && room.IsSpectator(client.PlayerName) {
		room.RemoveSpectator(client.PlayerName)
		log.Printf("spectator %s left room %s", client.PlayerName, client.RoomID)
		client.RoomID = ""
		return
	}
	h.LeaveRoom(client)
}

// LeaveRoom 将客户端从其所在房间中移除。
// 如果游戏已开始，保留玩家位置并启动断线计时器。
func (h *Hub) LeaveRoom(client *Client) {
	if client == nil || client.RoomID == "" {
		return
	}
	roomID := client.RoomID

	h.mu.RLock()
	room, ok := h.rooms[roomID]
	h.mu.RUnlock()
	if !ok {
		client.RoomID = ""
		return
	}

	playerName := client.PlayerName
	room.RemoveClient(playerName)

	left, _ := protocol.NewMessage(protocol.TypePlayerLeft, protocol.S2CPlayerLeft{
		Name:          playerName,
		Reconnectable: room.Started,
	})
	room.Broadcast(left)

	if room.Started {
		// 游戏已开始：启动断线计时器，并检查是否需要自动处理当前回合
		room.StartDisconnectTimer(playerName)
		room.BroadcastRoomInfo()
		// 如果轮到该断线玩家，自动处理其回合
		room.CheckActivePlayerOnline()
	} else if room.IsEmpty() {
		h.mu.Lock()
		delete(h.rooms, roomID)
		h.mu.Unlock()
		h.deleteRoomStorage(roomID)
		log.Printf("room %s removed (empty)", roomID)
	} else {
		room.BroadcastRoomInfo()
	}

	client.RoomID = ""
}

// HandleMessage 由 Client.ReadPump 调用，根据消息类型路由到对应处理函数。
func (h *Hub) HandleMessage(client *Client, raw []byte) {
	msg, err := protocol.ParseMessage(raw)
	if err != nil {
		client.SendError("invalid_message", err.Error())
		return
	}

	switch msg.Type {
	case protocol.TypeCreateRoom:
		payload, err := protocol.ParsePayload[protocol.C2SCreateRoom](msg)
		if err != nil {
			client.SendError("invalid_payload", err.Error())
			return
		}
		h.CreateRoom(client, payload)

	case protocol.TypeJoinRoom:
		payload, err := protocol.ParsePayload[protocol.C2SJoinRoom](msg)
		if err != nil {
			client.SendError("invalid_payload", err.Error())
			return
		}
		h.JoinRoom(client, payload)

	case protocol.TypeLeaveRoom:
		h.LeaveRoomOrSpectate(client)

	case protocol.TypeReady:
		room := h.getClientRoom(client)
		if room == nil {
			client.SendError("not_in_room", "尚未加入房间")
			return
		}
		room.SetReady(client.PlayerName, true)
		room.BroadcastRoomInfo()
		if room.AllReady() {
			room.StartGame()
		}

	case protocol.TypeChat:
		payload, err := protocol.ParsePayload[protocol.C2SChat](msg)
		if err != nil {
			client.SendError("invalid_payload", err.Error())
			return
		}
		room := h.getClientRoom(client)
		if room == nil {
			return
		}
		chatMsg := room.AddChatMessage(client.PlayerName, payload.Content)
		out, _ := protocol.NewMessage(protocol.TypeChatMsg, chatMsg)
		room.BroadcastAll(out)

	case protocol.TypeReconnect:
		payload, err := protocol.ParsePayload[protocol.C2SReconnect](msg)
		if err != nil {
			client.SendError("invalid_payload", err.Error())
			return
		}
		h.handleReconnect(client, payload)

	case protocol.TypeListRooms:
		h.ListRooms(client)

	case protocol.TypeSpectate:
		payload, err := protocol.ParsePayload[protocol.C2SSpectate](msg)
		if err != nil {
			client.SendError("invalid_payload", err.Error())
			return
		}
		h.SpectateRoom(client, payload)

	// 游戏操作类消息：路由给 ServerGame 处理
	case protocol.TypeRollDice,
		protocol.TypeBuyProperty,
		protocol.TypeBuild,
		protocol.TypeUseSkill,
		protocol.TypeMortgage,
		protocol.TypeRedeem,
		protocol.TypeTrade,
		protocol.TypeEndTurn:
		room := h.getClientRoom(client)
		if room == nil {
			client.SendError("not_in_room", "尚未加入房间")
			return
		}
		if room.Game == nil {
			client.SendError("not_started", "游戏尚未开始")
			return
		}
		if room.IsSpectator(client.PlayerName) {
			client.SendError("spectator_forbidden", "观战中不能执行游戏操作")
			return
		}
		room.HandleGameAction(client.PlayerName, msg.Type, msg.Payload)
		h.saveRoomAsync(room)

	default:
		client.SendError("unknown_type", "未知消息类型: "+msg.Type)
	}
}

// SpectateRoom 处理观战请求：加入已开始的房间作为观众。
func (h *Hub) SpectateRoom(client *Client, msg protocol.C2SSpectate) {
	if msg.PlayerName == "" {
		client.SendError("invalid_name", "观众名字不能为空")
		return
	}

	h.mu.RLock()
	room, ok := h.rooms[msg.RoomID]
	h.mu.RUnlock()
	if !ok {
		client.SendError("room_not_found", "房间不存在")
		return
	}

	if client.RoomID != "" && client.RoomID != msg.RoomID {
		h.LeaveRoomOrSpectate(client)
	}

	client.PlayerName = msg.PlayerName
	client.RoomID = msg.RoomID

	if err := room.AddSpectator(client); err != nil {
		client.SendError("spectate_failed", err.Error())
		return
	}

	// 发送观战成功确认
	okMsg, _ := protocol.NewMessage(protocol.TypeSpectateOK, protocol.S2CSpectateOK{RoomID: room.ID})
	client.SendBytes(okMsg)

	// 发送完整游戏状态供观战者查看
	room.mu.RLock()
	game := room.Game
	room.mu.RUnlock()
	if game != nil {
		if stateData := game.FullStateMessage(); stateData != nil {
			client.SendBytes(stateData)
		}
	}

	// 同步房间信息（成员状态等）
	info := room.GetRoomInfo()
	if data, err := protocol.NewMessage(protocol.TypeRoomInfo, info); err == nil {
		client.SendBytes(data)
	}

	// 发送聊天历史
	for _, chatMsg := range room.GetChatHistory() {
		if chatData, err := protocol.NewMessage(protocol.TypeChatMsg, chatMsg); err == nil {
			client.SendBytes(chatData)
		}
	}

	log.Printf("spectator %s joined room %s", msg.PlayerName, msg.RoomID)
}

// handleReconnect 处理玩家重连。
func (h *Hub) handleReconnect(client *Client, msg protocol.C2SReconnect) {
	h.mu.RLock()
	room, ok := h.rooms[msg.RoomID]
	h.mu.RUnlock()
	if !ok {
		client.SendError("room_not_found", "房间不存在")
		return
	}

	if !room.ValidateToken(msg.PlayerName, msg.Token) {
		client.SendError("invalid_token", "重连令牌无效")
		return
	}

	client.PlayerName = msg.PlayerName
	client.RoomID = msg.RoomID
	client.Token = msg.Token

	if err := room.Reconnect(client); err != nil {
		client.SendError("reconnect_failed", err.Error())
		return
	}

	// 取消断线计时器
	room.CancelDisconnectTimer(msg.PlayerName)

	// 发送重连成功确认
	out, _ := protocol.NewMessage(protocol.TypeReconnectOK, protocol.S2CReconnectOK{Token: client.Token})
	client.SendBytes(out)

	// 发送完整游戏状态给重连的客户端
	room.mu.RLock()
	game := room.Game
	room.mu.RUnlock()
	if game != nil {
		if stateData := game.FullStateMessage(); stateData != nil {
			client.SendBytes(stateData)
		}
		// 如果轮到该重连玩家的回合，发送 TurnUpdate 让他知道
		if idx, exists := game.nameIndex[msg.PlayerName]; exists && game.ActiveIdx() == idx {
			turnData, _ := protocol.NewMessage(protocol.TypeTurnUpdate, protocol.S2CTurnUpdate{
				ActiveIdx: game.ActiveIdx(),
				Phase:     game.Phase(),
			})
			client.SendBytes(turnData)
		}
	}

	// 发送聊天历史
	for _, chatMsg := range room.GetChatHistory() {
		chatData, _ := protocol.NewMessage(protocol.TypeChatMsg, chatMsg)
		client.SendBytes(chatData)
	}

	// 广播通知其他玩家该玩家已回来
	joined, _ := protocol.NewMessage(protocol.TypePlayerJoined, protocol.S2CPlayerJoined{
		Name:     client.PlayerName,
		CharName: client.CharName,
	})
	room.BroadcastExcept(joined, client.PlayerName)

	room.BroadcastRoomInfo()
	log.Printf("player %s reconnected to room %s", msg.PlayerName, msg.RoomID)
}

// ListRooms 向请求者发送当前所有房间的摘要信息。
//
// 列表仅包含公开字段（ID、人数、开局状态、玩家名），
// 不附带令牌、坐标、地产等敏感信息。
func (h *Hub) ListRooms(client *Client) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	items := make([]protocol.RoomListItem, 0, len(h.rooms))
	for id, room := range h.rooms {
		room.mu.RLock()
		item := protocol.RoomListItem{
			RoomID:      id,
			PlayerCount: len(room.Players),
			MaxPlayers:  MaxPlayersPerRoom,
			Started:     room.Started,
		}
		for _, p := range room.Players {
			item.Players = append(item.Players, p.Name)
		}
		room.mu.RUnlock()
		items = append(items, item)
	}

	data, err := protocol.NewMessage(protocol.TypeRoomList, protocol.S2CRoomList{Rooms: items})
	if err != nil {
		client.SendError("list_failed", err.Error())
		return
	}
	client.SendBytes(data)
}

func (h *Hub) getClientRoom(client *Client) *Room {
	if client == nil || client.RoomID == "" {
		return nil
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.rooms[client.RoomID]
}

// generateRoomID 生成 6 位字母数字房间 ID。
func (h *Hub) generateRoomID() string {
	const charset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	for attempt := 0; attempt < 16; attempt++ {
		id := randomString(charset, 6)
		h.mu.RLock()
		_, exists := h.rooms[id]
		h.mu.RUnlock()
		if !exists {
			return id
		}
	}
	return randomString("ABCDEFGHJKLMNPQRSTUVWXYZ23456789", 8)
}

func randomString(charset string, n int) string {
	out := make([]byte, n)
	max := big.NewInt(int64(len(charset)))
	for i := 0; i < n; i++ {
		idx, err := rand.Int(rand.Reader, max)
		if err != nil {
			out[i] = charset[0]
			continue
		}
		out[i] = charset[idx.Int64()]
	}
	return string(out)
}

func generateToken() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	return randomString(charset, 24)
}

// ===== 持久化 =====

// loadSavedRooms 启动时恢复未完成的房间。
func (h *Hub) loadSavedRooms() {
	roomIDs, err := h.storage.ListRooms()
	if err != nil {
		log.Printf("[storage] list rooms error: %v", err)
		return
	}
	for _, id := range roomIDs {
		data, err := h.storage.LoadRoom(id)
		if err != nil {
			log.Printf("[storage] load room %s error: %v", id, err)
			continue
		}
		room := &Room{
			ID:        data.RoomID,
			Clients:   make(map[string]*Client),
			Players:   make([]protocol.RoomPlayer, 0, len(data.Players)),
			CreatedAt: data.CreatedAt,
			Started:   data.Started,
			tokens:    make(map[string]string),
		}
		for _, pm := range data.Players {
			room.Players = append(room.Players, protocol.RoomPlayer{
				Name:     pm.Name,
				CharName: pm.CharName,
				Ready:    true,
				Online:   false, // 标记所有玩家离线，等待重连
			})
			if pm.Token != "" {
				room.tokens[pm.Name] = pm.Token
			}
		}
		if data.Started {
			room.Game = RestoreFromState(data.GameState)
		}
		h.rooms[data.RoomID] = room
		log.Printf("[storage] restored room %s (%d players)", data.RoomID, len(data.Players))
	}
}

// saveRoomAsync 异步保存房间状态。
func (h *Hub) saveRoomAsync(room *Room) {
	if h.storage == nil || room == nil {
		return
	}
	// 游戏结束的房间不保存（即将被清理）
	if room.Game != nil && room.Game.State() == StateGameOver {
		h.deleteRoomStorage(room.ID)
		return
	}
	go func() {
		data := h.buildRoomSaveData(room)
		if err := h.storage.SaveRoom(room.ID, data); err != nil {
			log.Printf("[storage] save room %s error: %v", room.ID, err)
		}
	}()
}

// deleteRoomStorage 删除房间的持久化数据。
func (h *Hub) deleteRoomStorage(roomID string) {
	if h.storage == nil {
		return
	}
	go func() {
		if err := h.storage.DeleteRoom(roomID); err != nil {
			log.Printf("[storage] delete room %s error: %v", roomID, err)
		}
	}()
}

// buildRoomSaveData 从 Room 构建持久化数据结构。
func (h *Hub) buildRoomSaveData(room *Room) *RoomSaveData {
	room.mu.RLock()
	defer room.mu.RUnlock()

	data := &RoomSaveData{
		RoomID:    room.ID,
		CreatedAt: room.CreatedAt,
		Started:   room.Started,
	}

	for _, rp := range room.Players {
		meta := RoomPlayerMeta{
			Name:     rp.Name,
			CharName: rp.CharName,
			Online:   rp.Online,
			Token:    room.tokens[rp.Name],
		}
		if !rp.Online {
			meta.DisconnectAt = time.Now()
		}
		data.Players = append(data.Players, meta)
	}

	if room.Game != nil {
		data.GameState = room.Game.GetFullState()
	}

	return data
}
