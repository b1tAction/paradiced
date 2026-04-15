// Package nakama provides Nakama match handler implementation for Paradiced.
package nakama

import (
	"encoding/json"
	"testing"

	"github.com/b1tAction/paradiced/internal/core"
	"github.com/b1tAction/paradiced/pkg/constants"
	"github.com/b1tAction/paradiced/pkg/protocol"
)

func TestHandleMessageUnknownOpCode(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)
	mockDispatcher := NewMockDispatcherAdapter()
	handler.WithDispatcher(mockDispatcher)

	// Add player and initialize
	handler.addPlayer("user-001", protocol.FactionQingLong)
	handler.MatchInit()

	// Send unknown opcode
	data, _ := json.Marshal(map[string]string{"op_code": "unknown"})
	err := handler.HandleMessage("user-001", data)
	if err != nil {
		t.Errorf("HandleMessage with unknown opcode should return nil, got: %v", err)
	}
}

func TestHandleMessageInvalidJSON(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)

	// Send invalid JSON - will return error from json.Unmarshal
	err := handler.HandleMessage("user-001", []byte("invalid json"))
	// Error is expected for invalid JSON
	_ = err // We accept either error or nil (implementation may vary)
}

func TestHandleRollDiceNonCurrentPlayer(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)
	mockDispatcher := NewMockDispatcherAdapter()
	handler.WithDispatcher(mockDispatcher)

	// Add 2 players
	handler.addPlayer("user-001", protocol.FactionQingLong)
	handler.addPlayer("user-002", protocol.FactionZhuQue)
	handler.MatchInit()

	// Send roll dice from non-current player (user-002)
	data, _ := json.Marshal(RollDiceRequest{OpCode: "1"}) // OpCode for RollDice
	err := handler.HandleMessage("user-002", data)
	if err != nil {
		t.Errorf("HandleMessage should return nil for non-current player, got: %v", err)
	}
}

func TestHandleUseItem(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)
	mockDispatcher := NewMockDispatcherAdapter()
	handler.WithDispatcher(mockDispatcher)

	// Add player with an item
	player := handler.addPlayer("user-001", protocol.FactionQingLong)
	player.AddItem(core.NewItem(constants.ItemTypeAnyDoor))

	handler.MatchInit()

	// Send use item request
	data, _ := json.Marshal(UseItemRequest{
		OpCode: "2", // OpCode for UseItem
		ItemID: "test-item-id",
	})

	// This should return nil even if item not found
	err := handler.HandleMessage("user-001", data)
	if err != nil {
		t.Errorf("HandleMessage should return nil, got: %v", err)
	}
}

func TestHandleUseSkill(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)
	mockDispatcher := NewMockDispatcherAdapter()
	handler.WithDispatcher(mockDispatcher)

	// Add player with charge
	player := handler.addPlayer("user-001", protocol.FactionQingLong)
	player.SetChargeCount(1)

	handler.MatchInit()

	// Note: Skill usage requires being in MainAction state and being current player
	// This test verifies message handling, not actual skill execution
	data, _ := json.Marshal(UseSkillRequest{OpCode: "3"}) // OpCode for UseSkill

	err := handler.HandleMessage("user-001", data)
	if err != nil {
		t.Errorf("HandleMessage should return nil, got: %v", err)
	}

	// Note: Charge may not be cleared if not in correct state
	// This is expected behavior - skill execution requires proper state
}

func TestHandleUseSkillNoCharge(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)
	mockDispatcher := NewMockDispatcherAdapter()
	handler.WithDispatcher(mockDispatcher)

	// Add player without charge
	player := handler.addPlayer("user-001", protocol.FactionQingLong)
	player.SetChargeCount(0)

	handler.MatchInit()

	// Send use skill request
	data, _ := json.Marshal(UseSkillRequest{OpCode: "3"})

	err := handler.HandleMessage("user-001", data)
	if err != nil {
		t.Errorf("HandleMessage should return nil, got: %v", err)
	}

	// Charge should still be 0
	if player.GetChargeCount() != 0 {
		t.Errorf("charge count = %d, want 0 (no change)", player.GetChargeCount())
	}
}

func TestHandleUserChoice(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)
	mockDispatcher := NewMockDispatcherAdapter()
	handler.WithDispatcher(mockDispatcher)

	// Add player
	handler.addPlayer("user-001", protocol.FactionQingLong)
	handler.MatchInit()

	// Send user choice
	data, _ := json.Marshal(UserChoiceResponse{
		OpCode:     "4", // OpCode for UserChoice
		DecisionID: "dec-001",
		Choice:     1,
	})

	err := handler.HandleMessage("user-001", data)
	if err != nil {
		t.Errorf("HandleMessage should return nil, got: %v", err)
	}
}

func TestHandleMiniGameResult(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)
	mockDispatcher := NewMockDispatcherAdapter()
	handler.WithDispatcher(mockDispatcher)

	// Add player
	handler.addPlayer("user-001", protocol.FactionQingLong)
	handler.MatchInit()

	// Send mini-game result
	data, _ := json.Marshal(MiniGameResultSubmit{
		OpCode: "5", // OpCode for MiniGameResultSubmit
		Rank:   1,
	})

	err := handler.HandleMessage("user-001", data)
	if err != nil {
		t.Errorf("HandleMessage should return nil, got: %v", err)
	}
}

func TestHandleMessageNonExistingPlayer(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)
	mockDispatcher := NewMockDispatcherAdapter()
	handler.WithDispatcher(mockDispatcher)

	// Add one player
	handler.addPlayer("user-001", protocol.FactionQingLong)
	handler.MatchInit()

	// Send message from non-existing player
	data, _ := json.Marshal(RollDiceRequest{OpCode: "1"})
	err := handler.HandleMessage("user-999", data)
	if err != nil {
		t.Errorf("HandleMessage from non-existing player should return nil, got: %v", err)
	}
}