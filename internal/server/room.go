package server

import (
	"errors"
	"sync"
	"time"

	"github.com/PlasmolysisMango/Gonopoly/pkg/protocol"
)

// MaxPlayersPerRoom 单个房间最多容纳的玩家数。
const MaxPlayersPerRoom = 4

// ChatMessage 聊天记录条目。
type ChatMessage struct {
	From      string `json:"from"`
	Content   string `json:"content"`
	Timestamp int64  `json:"timestamp"`
}

// Room 表示一个游戏房间。
type Room struct {
	ID        string
	Clients   map[string]*Client // playerName -> client
	Players   []protocol.RoomPlayer
	Started   bool
	CreatedAt time.Time
	Game      *ServerGame // 游戏开始后由 StartGame 设置

	ChatHistory []ChatMessage // 最近100条聊天记录

	Spectators map[string]*Client // 观众列表 (playerName -> client)

	tokens map[string]string
	dtm    *disconnectTimerManager // 断线计时器管理
	mu     sync.RWMutex
}

func NewRoom(id string) *Room {
	return &Room{
		ID:         id,
		Clients:    make(map[string]*Client),
		Players:    make([]protocol.RoomPlayer, 0, MaxPlayersPerRoom),
		CreatedAt:  time.Now(),
		Spectators: make(map[string]*Client),
		tokens:     make(map[string]string),
		dtm:        newDisconnectTimerManager(),
	}
}

func (r *Room) AddClient(client *Client) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.Started {
		return errors.New("游戏已开始，无法加入")
	}
	if _, exists := r.Clients[client.PlayerName]; exists {
		return errors.New("玩家名已被占用")
	}
	if len(r.Players) >= MaxPlayersPerRoom {
		return errors.New("房间已满")
	}

	r.Clients[client.PlayerName] = client
	r.Players = append(r.Players, protocol.RoomPlayer{
		Name:     client.PlayerName,
		CharName: client.CharName,
		Ready:    false,
		Online:   true,
	})
	if client.Token != "" {
		r.tokens[client.PlayerName] = client.Token
	}
	return nil
}

func (r *Room) RemoveClient(playerName string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.Clients[playerName]; !ok {
		return
	}
	delete(r.Clients, playerName)

	if r.Started {
		for i := range r.Players {
			if r.Players[i].Name == playerName {
				r.Players[i].Online = false
				break
			}
		}
		return
	}

	for i := range r.Players {
		if r.Players[i].Name == playerName {
			r.Players = append(r.Players[:i], r.Players[i+1:]...)
			break
		}
	}
	delete(r.tokens, playerName)
}

func (r *Room) Reconnect(client *Client) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	found := false
	for i := range r.Players {
		if r.Players[i].Name == client.PlayerName {
			r.Players[i].Online = true
			found = true
			break
		}
	}
	if !found {
		return errors.New("玩家不在房间中")
	}
	r.Clients[client.PlayerName] = client
	r.tokens[client.PlayerName] = client.Token
	return nil
}

func (r *Room) SetReady(playerName string, ready bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range r.Players {
		if r.Players[i].Name == playerName {
			r.Players[i].Ready = ready
			return
		}
	}
}

func (r *Room) ValidateToken(playerName, token string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	want, ok := r.tokens[playerName]
	return ok && want != "" && want == token
}

func (r *Room) IsEmpty() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.Clients) == 0
}

func (r *Room) Broadcast(data []byte) {
	r.mu.RLock()
	clients := make([]*Client, 0, len(r.Clients))
	for _, c := range r.Clients {
		clients = append(clients, c)
	}
	r.mu.RUnlock()

	for _, c := range clients {
		c.SendBytes(data)
	}
}

// AddSpectator 将一个客户端加入观众列表。
// 调用前上层需设置 client.PlayerName。
// 要求游戏已开始，且名字不与现有玩家/观众冲突。
func (r *Room) AddSpectator(client *Client) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.Started {
		return errors.New("游戏尚未开始，无法观战")
	}
	if client.PlayerName == "" {
		return errors.New("观众名字不能为空")
	}
	if _, exists := r.Clients[client.PlayerName]; exists {
		return errors.New("名字已被玩家占用")
	}
	if _, exists := r.Spectators[client.PlayerName]; exists {
		return errors.New("观众名字已被占用")
	}
	r.Spectators[client.PlayerName] = client
	return nil
}

// RemoveSpectator 从观众列表中移除。
func (r *Room) RemoveSpectator(playerName string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.Spectators, playerName)
}

// IsSpectator 查询某名字是否在观众列表中。
func (r *Room) IsSpectator(playerName string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.Spectators[playerName]
	return ok
}

// BroadcastAll 广播给所有玩家与观众。
func (r *Room) BroadcastAll(data []byte) {
	r.mu.RLock()
	clients := make([]*Client, 0, len(r.Clients)+len(r.Spectators))
	for _, c := range r.Clients {
		clients = append(clients, c)
	}
	for _, c := range r.Spectators {
		clients = append(clients, c)
	}
	r.mu.RUnlock()

	for _, c := range clients {
		c.SendBytes(data)
	}
}

func (r *Room) BroadcastExcept(data []byte, except string) {
	r.mu.RLock()
	clients := make([]*Client, 0, len(r.Clients))
	for name, c := range r.Clients {
		if name == except {
			continue
		}
		clients = append(clients, c)
	}
	r.mu.RUnlock()

	for _, c := range clients {
		c.SendBytes(data)
	}
}

func (r *Room) SendTo(playerName string, data []byte) {
	r.mu.RLock()
	c, ok := r.Clients[playerName]
	r.mu.RUnlock()
	if !ok {
		return
	}
	c.SendBytes(data)
}

func (r *Room) GetRoomInfo() protocol.S2CRoomInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	players := make([]protocol.RoomPlayer, len(r.Players))
	copy(players, r.Players)

	return protocol.S2CRoomInfo{
		RoomID:  r.ID,
		Players: players,
		Started: r.Started,
	}
}

func (r *Room) BroadcastRoomInfo() {
	info := r.GetRoomInfo()
	data, err := protocol.NewMessage(protocol.TypeRoomInfo, info)
	if err != nil {
		return
	}
	r.BroadcastAll(data)
}

// AllReady 当且仅当房间内有至少 2 个玩家且全部 Ready 时返回 true。
func (r *Room) AllReady() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.Started || len(r.Players) < 2 {
		return false
	}
	for _, rp := range r.Players {
		if !rp.Ready {
			return false
		}
	}
	return true
}

// StartGame 创建 ServerGame 并广播初始状态。重复调用是幂等的。
func (r *Room) StartGame() {
	r.mu.Lock()
	if r.Started || len(r.Players) == 0 {
		r.mu.Unlock()
		return
	}
	players := make([]protocol.RoomPlayer, len(r.Players))
	copy(players, r.Players)
	game := NewServerGame(players)
	r.Game = game
	r.Started = true
	r.mu.Unlock()

	// 通知房间已开始
	r.BroadcastRoomInfo()
	// 全量状态同步
	if data := game.FullStateMessage(); data != nil {
		r.BroadcastAll(data)
	}
}

// AddChatMessage 将一条聊天消息加入历史并返回 S2CChatMsg 用于广播。
func (r *Room) AddChatMessage(from, content string) protocol.S2CChatMsg {
	msg := ChatMessage{
		From:      from,
		Content:   content,
		Timestamp: time.Now().Unix(),
	}
	r.mu.Lock()
	r.ChatHistory = append(r.ChatHistory, msg)
	if len(r.ChatHistory) > 100 {
		r.ChatHistory = r.ChatHistory[len(r.ChatHistory)-100:]
	}
	r.mu.Unlock()
	return protocol.S2CChatMsg{
		From:      msg.From,
		Content:   msg.Content,
		Timestamp: msg.Timestamp,
	}
}

// GetChatHistory 返回历史聊天消息（用于重连时发送）。
func (r *Room) GetChatHistory() []protocol.S2CChatMsg {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]protocol.S2CChatMsg, 0, len(r.ChatHistory))
	for _, m := range r.ChatHistory {
		result = append(result, protocol.S2CChatMsg{
			From:      m.From,
			Content:   m.Content,
			Timestamp: m.Timestamp,
		})
	}
	return result
}

// HandleGameAction 调用 ServerGame 处理玩家操作并广播返回的消息。
func (r *Room) HandleGameAction(playerName, msgType string, payload []byte) {
	r.mu.RLock()
	game := r.Game
	r.mu.RUnlock()
	if game == nil {
		return
	}
	msgs := game.HandleAction(playerName, msgType, payload)
	for _, m := range msgs {
		r.BroadcastAll(m)
	}
}
