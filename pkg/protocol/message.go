package protocol

import "encoding/json"

// Message 是所有网络消息的统一包装格式。
// Type 字段标识消息类型，Payload 为对应的具体消息体（已序列化的 JSON）。
type Message struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}
