// Package net provides network message protocol definitions for client-server communication.
package net

import (
	"encoding/json"
	"testing"
)

func TestNewMessage(t *testing.T) {
	stateSync := &StateSync{
		GlobalState: "turn_loop",
		Round:       1,
	}

	msg, err := NewMessage(OpStateSync, stateSync)
	if err != nil {
		t.Fatalf("NewMessage() error: %v", err)
	}
	if msg.OpCode != OpStateSync {
		t.Errorf("NewMessage().OpCode = %d, want %d", msg.OpCode, OpStateSync)
	}
	if msg.Timestamp <= 0 {
		t.Error("NewMessage().Timestamp should be > 0")
	}
	if len(msg.Data) == 0 {
		t.Error("NewMessage().Data should not be empty")
	}
}

func TestNewMessageWithNilData(t *testing.T) {
	msg, err := NewMessage(OpRollDice, nil)
	if err != nil {
		t.Fatalf("NewMessage() error: %v", err)
	}
	if msg.OpCode != OpRollDice {
		t.Errorf("NewMessage().OpCode = %d, want %d", msg.OpCode, OpRollDice)
	}
	// json.Marshal(nil) produces "null" not empty
	if string(msg.Data) != "null" {
		t.Errorf("NewMessage().Data = %s, want null", string(msg.Data))
	}
}

func TestParseData(t *testing.T) {
	original := &StateSync{
		GlobalState: "turn_loop",
		TurnState:   "main_action",
		TurnPlayer:  "player-abc123",
		Round:       1,
		Turn:        0,
	}
	msg, err := NewMessage(OpStateSync, original)
	if err != nil {
		t.Fatalf("NewMessage() error: %v", err)
	}

	var parsed StateSync
	err = msg.ParseData(&parsed)
	if err != nil {
		t.Fatalf("ParseData() error: %v", err)
	}
	if parsed.GlobalState != "turn_loop" {
		t.Errorf("parsed.GlobalState = %s, want turn_loop", parsed.GlobalState)
	}
	if parsed.TurnState != "main_action" {
		t.Errorf("parsed.TurnState = %s, want main_action", parsed.TurnState)
	}
	if parsed.TurnPlayer != "player-abc123" {
		t.Errorf("parsed.TurnPlayer = %s, want player-abc123", parsed.TurnPlayer)
	}
	if parsed.Round != 1 {
		t.Errorf("parsed.Round = %d, want 1", parsed.Round)
	}
}

func TestParseDataWithEmptyData(t *testing.T) {
	msg := &Message{
		OpCode:    OpRollDice,
		Timestamp: 12345,
		Data:      json.RawMessage{},
	}

	var data RollDice
	err := msg.ParseData(&data)
	if err != nil {
		t.Errorf("ParseData() error: %v", err)
	}
}

func TestMustNewMessage(t *testing.T) {
	stateSync := &StateSync{GlobalState: "turn_loop"}
	msg := MustNewMessage(OpStateSync, stateSync)
	if msg == nil {
		t.Fatal("MustNewMessage() returned nil")
	}
	if msg.OpCode != OpStateSync {
		t.Errorf("MustNewMessage().OpCode = %d, want %d", msg.OpCode, OpStateSync)
	}
}

func TestMessageJSONSerialization(t *testing.T) {
	stateSync := &StateSync{
		GlobalState: "turn_loop",
		Round:       1,
	}

	msg, err := NewMessage(OpStateSync, stateSync)
	if err != nil {
		t.Fatalf("NewMessage() error: %v", err)
	}

	jsonBytes, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	var parsed Message
	err = json.Unmarshal(jsonBytes, &parsed)
	if err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}
	if parsed.OpCode != OpStateSync {
		t.Errorf("parsed.OpCode = %d, want %d", parsed.OpCode, OpStateSync)
	}
	if parsed.Timestamp != msg.Timestamp {
		t.Errorf("parsed.Timestamp = %d, want %d", parsed.Timestamp, msg.Timestamp)
	}
}