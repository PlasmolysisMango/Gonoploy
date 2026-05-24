package protocol

import (
	"encoding/json"
	"fmt"
)

// MarshalPayload 将任意 payload 序列化为 json.RawMessage。
// 当 payload 为 nil 时返回 JSON null。
func MarshalPayload(payload interface{}) (json.RawMessage, error) {
	if payload == nil {
		return json.RawMessage("null"), nil
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}
	return data, nil
}

// NewMessage 将消息类型与 payload 打包为可直接发送的 JSON 字节流。
func NewMessage(msgType string, payload interface{}) ([]byte, error) {
	raw, err := MarshalPayload(payload)
	if err != nil {
		return nil, err
	}
	msg := Message{
		Type:    msgType,
		Payload: raw,
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("marshal message: %w", err)
	}
	return data, nil
}

// ParseMessage 将原始 JSON 字节流解析为 Message 包装结构。
func ParseMessage(data []byte) (Message, error) {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return Message{}, fmt.Errorf("parse message: %w", err)
	}
	return msg, nil
}

// ParsePayload 使用泛型从 Message 中解析具体类型的 payload。
func ParsePayload[T any](msg Message) (T, error) {
	var out T
	if len(msg.Payload) == 0 || string(msg.Payload) == "null" {
		return out, nil
	}
	if err := json.Unmarshal(msg.Payload, &out); err != nil {
		return out, fmt.Errorf("parse payload: %w", err)
	}
	return out, nil
}
