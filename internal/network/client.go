// Package network 实现客户端与服务端的 WebSocket 通信层。
//
// 该层职责：
//  1. 建立/维护到服务器的长连接（基于 nhooyr.io/websocket，兼容 WASM）。
//  2. 提供非阻塞的发送队列与接收队列，便于在 Ebiten 主循环中轮询。
//  3. 在网络断开时自动尝试重连，并在重连成功后重新发送 C2SReconnect。
//
// 本包不依赖 Ebiten，以保持网络层的纯净，便于复用与测试。
package network

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"nhooyr.io/websocket"

	"github.com/PlasmolysisMango/Gonopoly/pkg/protocol"
)

// ConnectionState 表示客户端与服务端的连接生命周期状态。
type ConnectionState int

const (
	// StateDisconnected 当前未连接（初始或已主动关闭）。
	StateDisconnected ConnectionState = iota
	// StateConnecting 正在执行首次连接握手。
	StateConnecting
	// StateConnected 已建立可用的 WebSocket 连接。
	StateConnected
	// StateReconnecting 连接中断后正在尝试自动重连。
	StateReconnecting
)

// String 返回连接状态的可读名称，便于日志与调试。
func (s ConnectionState) String() string {
	switch s {
	case StateDisconnected:
		return "disconnected"
	case StateConnecting:
		return "connecting"
	case StateConnected:
		return "connected"
	case StateReconnecting:
		return "reconnecting"
	default:
		return "unknown"
	}
}

// 网络层相关的可调参数。
const (
	sendBufferSize       = 256             // 发送队列缓冲
	recvBufferSize       = 256             // 接收队列缓冲（避免读 goroutine 阻塞）
	writeTimeout         = 10 * time.Second // 单次写超时
	readMessageLimit     = 1 << 20         // 单条消息上限 1 MiB
	reconnectInterval    = 3 * time.Second // 重连尝试间隔
	maxReconnectAttempts = 100             // 最多重连 100 次（约 5 分钟）
)

// NetClient 是客户端的 WebSocket 通信封装。
//
// 典型用法：
//
//	nc := NewNetClient("ws://host:port/ws")
//	if err := nc.Connect(); err != nil { ... }
//	nc.Send(protocol.TypeJoinRoom, &protocol.C2SJoinRoom{...})
//	for {
//	    if msg, ok := nc.Poll(); ok { ... }
//	}
type NetClient struct {
	serverURL string

	// conn 受 mu 保护，断线/重连时会被替换。
	conn  *websocket.Conn
	state ConnectionState

	send chan []byte           // 待发送的原始 JSON 字节
	recv chan protocol.Message // 已解析的接收消息

	// 重连所需信息，加入房间后由上层通过 SetReconnectInfo 设置。
	roomID     string
	playerName string
	token      string

	// 生命周期控制。
	ctx        context.Context
	cancel     context.CancelFunc
	pumpCtx    context.Context // 当前一次连接的 read/write pump 上下文
	pumpCancel context.CancelFunc

	mu sync.RWMutex

	// 回调，从内部 goroutine 触发，调用方需自行保证线程安全。
	onDisconnect func()
	onReconnect  func()
}

// NewNetClient 构造一个尚未建立连接的客户端实例。
func NewNetClient(serverURL string) *NetClient {
	ctx, cancel := context.WithCancel(context.Background())
	return &NetClient{
		serverURL: serverURL,
		send:      make(chan []byte, sendBufferSize),
		recv:      make(chan protocol.Message, recvBufferSize),
		state:     StateDisconnected,
		ctx:       ctx,
		cancel:    cancel,
	}
}

// Connect 同步执行一次首次连接握手。
// 握手成功后会启动后台的 readPump/writePump goroutine。
// 该方法是同步阻塞的，调用方应在合适的时机（如登录前）调用。
func (nc *NetClient) Connect() error {
	nc.mu.Lock()
	if nc.state == StateConnected || nc.state == StateConnecting {
		nc.mu.Unlock()
		return errors.New("network: already connected or connecting")
	}
	nc.state = StateConnecting
	url := nc.serverURL
	nc.mu.Unlock()

	dialCtx, dialCancel := context.WithTimeout(nc.ctx, writeTimeout)
	defer dialCancel()

	conn, _, err := websocket.Dial(dialCtx, url, nil)
	if err != nil {
		nc.mu.Lock()
		nc.state = StateDisconnected
		nc.mu.Unlock()
		return fmt.Errorf("network: dial %s: %w", url, err)
	}
	conn.SetReadLimit(readMessageLimit)

	nc.attachConn(conn)
	return nil
}

// attachConn 将一个已经握手成功的连接挂载到客户端，
// 并启动配套的读写 goroutine。
func (nc *NetClient) attachConn(conn *websocket.Conn) {
	pumpCtx, pumpCancel := context.WithCancel(nc.ctx)

	nc.mu.Lock()
	nc.conn = conn
	nc.state = StateConnected
	nc.pumpCtx = pumpCtx
	nc.pumpCancel = pumpCancel
	nc.mu.Unlock()

	go nc.readPump(pumpCtx, conn)
	go nc.writePump(pumpCtx, conn)
}

// Send 将消息序列化后放入发送队列。
// 该方法非阻塞：若发送队列已满，会返回 ErrSendQueueFull，
// 调用方可根据业务需要选择丢弃或稍后重试。
func (nc *NetClient) Send(msgType string, payload interface{}) error {
	data, err := protocol.NewMessage(msgType, payload)
	if err != nil {
		return fmt.Errorf("network: marshal %s: %w", msgType, err)
	}
	select {
	case nc.send <- data:
		return nil
	case <-nc.ctx.Done():
		return errors.New("network: client closed")
	default:
		return errors.New("network: send queue full")
	}
}

// Poll 非阻塞获取一条已接收的消息。
// 当无消息时返回 (nil, false)。
func (nc *NetClient) Poll() (*protocol.Message, bool) {
	select {
	case msg := <-nc.recv:
		return &msg, true
	default:
		return nil, false
	}
}

// State 返回当前连接状态，可被任意 goroutine 调用。
func (nc *NetClient) State() ConnectionState {
	nc.mu.RLock()
	defer nc.mu.RUnlock()
	return nc.state
}

// Close 主动关闭网络层并释放所有资源。Close 是幂等的。
func (nc *NetClient) Close() {
	nc.mu.Lock()
	if nc.state == StateDisconnected && nc.ctx.Err() != nil {
		nc.mu.Unlock()
		return
	}
	nc.state = StateDisconnected
	conn := nc.conn
	nc.conn = nil
	pumpCancel := nc.pumpCancel
	nc.pumpCancel = nil
	nc.mu.Unlock()

	if pumpCancel != nil {
		pumpCancel()
	}
	if conn != nil {
		_ = conn.Close(websocket.StatusNormalClosure, "client closing")
	}
	nc.cancel()
}

// SetReconnectInfo 在成功加入房间后由上层设置，
// 用于在自动重连后向服务器发送 C2SReconnect 恢复会话。
func (nc *NetClient) SetReconnectInfo(roomID, playerName, token string) {
	nc.mu.Lock()
	defer nc.mu.Unlock()
	nc.roomID = roomID
	nc.playerName = playerName
	nc.token = token
}

// OnDisconnect 注册断线回调，在每次连接断开后被触发。
func (nc *NetClient) OnDisconnect(fn func()) {
	nc.mu.Lock()
	defer nc.mu.Unlock()
	nc.onDisconnect = fn
}

// OnReconnect 注册重连成功回调，在自动重连成功并发送 C2SReconnect 后触发。
func (nc *NetClient) OnReconnect(fn func()) {
	nc.mu.Lock()
	defer nc.mu.Unlock()
	nc.onReconnect = fn
}

// readPump 持续从 WebSocket 读取消息并放入 recv 队列。
// 当读取出错（连接断开）时，触发重连流程并退出。
func (nc *NetClient) readPump(ctx context.Context, conn *websocket.Conn) {
	for {
		// Read 在 ctx 取消时会立即返回错误。
		_, data, err := conn.Read(ctx)
		if err != nil {
			nc.handleDisconnect(conn)
			return
		}
		msg, perr := protocol.ParseMessage(data)
		if perr != nil {
			// 单条消息解析失败不影响连接，继续读取下一条。
			continue
		}
		// recv 缓冲足够大；若仍然满载则丢弃最旧的一条以避免阻塞读循环。
		select {
		case nc.recv <- msg:
		default:
			select {
			case <-nc.recv:
			default:
			}
			select {
			case nc.recv <- msg:
			default:
			}
		}
	}
}

// writePump 持续从 send 队列取出字节并写入 WebSocket。
func (nc *NetClient) writePump(ctx context.Context, conn *websocket.Conn) {
	for {
		select {
		case <-ctx.Done():
			return
		case data, ok := <-nc.send:
			if !ok {
				return
			}
			writeCtx, cancel := context.WithTimeout(ctx, writeTimeout)
			err := conn.Write(writeCtx, websocket.MessageText, data)
			cancel()
			if err != nil {
				nc.handleDisconnect(conn)
				return
			}
		}
	}
}

// handleDisconnect 处理连接断开：清理当前连接，
// 若客户端尚未被 Close 且具备重连信息，则启动后台重连。
func (nc *NetClient) handleDisconnect(stale *websocket.Conn) {
	nc.mu.Lock()
	// 多个 goroutine 可能同时触发；仅由首次进入者处理。
	if nc.conn != stale {
		nc.mu.Unlock()
		return
	}
	if nc.state == StateDisconnected || nc.ctx.Err() != nil {
		nc.mu.Unlock()
		return
	}
	nc.state = StateReconnecting
	nc.conn = nil
	pumpCancel := nc.pumpCancel
	nc.pumpCancel = nil
	cb := nc.onDisconnect
	nc.mu.Unlock()

	if pumpCancel != nil {
		pumpCancel()
	}
	_ = stale.Close(websocket.StatusGoingAway, "connection lost")

	if cb != nil {
		go cb()
	}

	go nc.reconnectLoop()
}

// reconnectLoop 在断线后周期性地尝试重新连接。
// 每 reconnectInterval 重试一次，最多尝试 maxReconnectAttempts 次。
func (nc *NetClient) reconnectLoop() {
	for attempt := 0; attempt < maxReconnectAttempts; attempt++ {
		// 等待间隔或客户端关闭。
		select {
		case <-nc.ctx.Done():
			return
		case <-time.After(reconnectInterval):
		}

		nc.mu.RLock()
		if nc.state == StateDisconnected || nc.ctx.Err() != nil {
			nc.mu.RUnlock()
			return
		}
		url := nc.serverURL
		nc.mu.RUnlock()

		dialCtx, cancel := context.WithTimeout(nc.ctx, writeTimeout)
		conn, _, err := websocket.Dial(dialCtx, url, nil)
		cancel()
		if err != nil {
			continue
		}
		conn.SetReadLimit(readMessageLimit)
		nc.attachConn(conn)

		// 重连成功后立即尝试发送 C2SReconnect（若上层已经设置过重连信息）。
		nc.mu.RLock()
		roomID, playerName, token := nc.roomID, nc.playerName, nc.token
		cb := nc.onReconnect
		nc.mu.RUnlock()
		if roomID != "" && playerName != "" {
			_ = nc.Send(protocol.TypeReconnect, &protocol.C2SReconnect{
				RoomID:     roomID,
				PlayerName: playerName,
				Token:      token,
			})
		}
		if cb != nil {
			go cb()
		}
		return
	}

	// 超过最大重试次数，标记为彻底断开。
	nc.mu.Lock()
	nc.state = StateDisconnected
	nc.mu.Unlock()
}
