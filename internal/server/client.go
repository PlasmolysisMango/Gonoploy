package server

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/PlasmolysisMango/Gonopoly/pkg/protocol"
	"nhooyr.io/websocket"
)

const (
	// readTimeout WebSocket 单次读取的超时时间。
	readTimeout = 60 * time.Second
	// writeTimeout WebSocket 单次写入的超时时间。
	writeTimeout = 10 * time.Second
	// pingInterval 心跳检测间隔。
	pingInterval = 30 * time.Second
	// maxMessageSize 单条消息最大字节数。
	maxMessageSize = 1 << 16 // 64KB
)

// Client 表示一个 WebSocket 连接对应的客户端。
type Client struct {
	Hub        *Hub
	Conn       *websocket.Conn
	Send       chan []byte
	PlayerName string
	CharName   string
	RoomID     string
	Connected  bool
	Token      string // 用于重连验证
}

// ReadPump 读取循环：解析 WebSocket 消息并路由到 Hub 处理。
func (c *Client) ReadPump() {
	defer func() {
		_ = c.Conn.Close(websocket.StatusNormalClosure, "")
		c.Connected = false
	}()

	c.Conn.SetReadLimit(maxMessageSize)

	for {
		ctx, cancel := context.WithTimeout(context.Background(), readTimeout)
		_, data, err := c.Conn.Read(ctx)
		cancel()
		if err != nil {
			if !isCloseError(err) {
				log.Printf("read error from %s: %v", c.PlayerName, err)
			}
			return
		}
		c.Hub.HandleMessage(c, data)
	}
}

// WritePump 写入循环：从 Send channel 取消息发送，并定期发送 ping 心跳。
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()

	for {
		select {
		case msg, ok := <-c.Send:
			if !ok {
				_ = c.Conn.Close(websocket.StatusNormalClosure, "channel closed")
				return
			}
			ctx, cancel := context.WithTimeout(context.Background(), writeTimeout)
			err := c.Conn.Write(ctx, websocket.MessageText, msg)
			cancel()
			if err != nil {
				log.Printf("write error to %s: %v", c.PlayerName, err)
				return
			}
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), writeTimeout)
			err := c.Conn.Ping(ctx)
			cancel()
			if err != nil {
				log.Printf("ping error to %s: %v", c.PlayerName, err)
				return
			}
		}
	}
}

// SendError 向客户端发送错误消息（不阻塞）。
func (c *Client) SendError(code, message string) {
	data, err := protocol.NewMessage(protocol.TypeError, protocol.S2CError{
		Code:    code,
		Message: message,
	})
	if err != nil {
		return
	}
	c.SendBytes(data)
}

// SendBytes 非阻塞地把数据写入 Send channel；channel 满时丢弃并记录日志。
func (c *Client) SendBytes(data []byte) {
	defer func() {
		// channel 已被 Hub 关闭时 send 会 panic，吞掉以保证健壮性。
		_ = recover()
	}()
	select {
	case c.Send <- data:
	default:
		log.Printf("send buffer full for %s, dropping message", c.PlayerName)
	}
}

// isCloseError 判断错误是否为正常的连接关闭。
func isCloseError(err error) bool {
	if err == nil {
		return true
	}
	var ce websocket.CloseError
	if errors.As(err, &ce) {
		switch ce.Code {
		case websocket.StatusNormalClosure, websocket.StatusGoingAway:
			return true
		}
	}
	return false
}

// nowUnixMilli 返回当前时间的毫秒时间戳。
func nowUnixMilli() int64 {
	return time.Now().UnixMilli()
}
