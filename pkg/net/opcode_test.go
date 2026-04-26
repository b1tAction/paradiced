// Package net provides network message protocol definitions for client-server communication.
package net

import (
	"testing"
)

func TestOpCodeString(t *testing.T) {
	tests := []struct {
		op   OpCode
		want string
	}{
		{OpStateSync, "state_sync"},
		{OpDecisionRequest, "decision_request"},
		{OpAvailable, "available"},
		{OpMiniGameStart, "mini_game_start"},
		{OpMiniGameResult, "mini_game_result"},
		{OpGameOver, "game_over"},
		{OpFullSync, "full_sync"},
		{OpRollDice, "roll_dice"},
		{OpUseItem, "use_item"},
		{OpUseSkill, "use_skill"},
		{OpUserChoice, "user_choice"},
	}

	for _, tt := range tests {
		if got := tt.op.String(); got != tt.want {
			t.Errorf("OpCode(%d).String() = %s, want %s", tt.op, got, tt.want)
		}
	}
}

func TestOpCodeUnknown(t *testing.T) {
	op := OpCode(999)
	if got := op.String(); got != "unknown" {
		t.Errorf("OpCode(999).String() = %s, want unknown", got)
	}
}

func TestOpCodeIsServerToClient(t *testing.T) {
	// Server -> Client: 1-99
	serverToClient := []OpCode{
		OpStateSync, OpDecisionRequest, OpAvailable,
		OpMiniGameStart, OpMiniGameResult, OpGameOver, OpFullSync,
	}
	for _, op := range serverToClient {
		if !op.IsServerToClient() {
			t.Errorf("OpCode(%s).IsServerToClient() should be true", op.String())
		}
	}

	// Client -> Server: 100+
	clientToServer := []OpCode{OpRollDice, OpUseItem, OpUseSkill, OpUserChoice}
	for _, op := range clientToServer {
		if op.IsServerToClient() {
			t.Errorf("OpCode(%s).IsServerToClient() should be false", op.String())
		}
	}
}

func TestOpCodeIsClientToServer(t *testing.T) {
	// Client -> Server: 100+
	clientToServer := []OpCode{OpRollDice, OpUseItem, OpUseSkill, OpUserChoice}
	for _, op := range clientToServer {
		if !op.IsClientToServer() {
			t.Errorf("OpCode(%s).IsClientToServer() should be true", op.String())
		}
	}

	// Server -> Client: 1-99
	serverToClient := []OpCode{OpStateSync, OpGameOver}
	for _, op := range serverToClient {
		if op.IsClientToServer() {
			t.Errorf("OpCode(%s).IsClientToServer() should be false", op.String())
		}
	}
}

func TestOpCodeBoundaries(t *testing.T) {
	// Min server-to-client
	if !OpCode(1).IsServerToClient() {
		t.Error("OpCode(1).IsServerToClient() should be true")
	}
	// Max server-to-client
	if !OpCode(99).IsServerToClient() {
		t.Error("OpCode(99).IsServerToClient() should be true")
	}
	// First client-to-server
	if OpCode(100).IsServerToClient() {
		t.Error("OpCode(100).IsServerToClient() should be false")
	}
	if !OpCode(100).IsClientToServer() {
		t.Error("OpCode(100).IsClientToServer() should be true")
	}
}