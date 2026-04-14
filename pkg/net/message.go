package net

import (
	"encoding/json"
	"time"
)

// Message is the base message structure for all client-server communication.
// All messages share this common envelope with an opcode and timestamp.
type Message struct {
	// OpCode identifies the message type.
	OpCode OpCode `json:"op_code"`

	// Timestamp is the message creation time in milliseconds.
	Timestamp int64 `json:"timestamp"`

	// Data contains the message payload as raw JSON.
	// The actual structure depends on OpCode and should be parsed separately.
	Data json.RawMessage `json:"data"`
}

// NewMessage creates a new message with the given opcode and data.
// The data is marshaled to JSON and stored in the Data field.
func NewMessage(op OpCode, data interface{}) (*Message, error) {
	bytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	return &Message{
		OpCode:    op,
		Timestamp: time.Now().UnixMilli(),
		Data:      bytes,
	}, nil
}

// ParseData unmarshals the Data field into the provided target structure.
// The target should match the expected structure for this message's OpCode.
func (m *Message) ParseData(target interface{}) error {
	if len(m.Data) == 0 {
		return nil
	}
	return json.Unmarshal(m.Data, target)
}

// MustNewMessage creates a new message, panicking on error.
// Use only when you're certain marshaling will succeed.
func MustNewMessage(op OpCode, data interface{}) *Message {
	msg, err := NewMessage(op, data)
	if err != nil {
		panic(err)
	}
	return msg
}