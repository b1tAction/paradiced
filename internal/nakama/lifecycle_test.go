// Package nakama provides Nakama match handler implementation for Paradiced.
package nakama

import (
	"testing"
	"time"

	"github.com/b1tAction/paradiced/internal/engine/hsm"
	"github.com/b1tAction/paradiced/pkg/constants"
)

func TestMatchInit(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)
	mockDispatcher := NewMockDispatcherAdapter()
	handler.WithDispatcher(mockDispatcher)

	// Add players first
	handler.addPlayer("user-001", constants.FactionQingLong)
	handler.addPlayer("user-002", constants.FactionZhuQue)
	handler.addPlayer("user-003", constants.FactionBaiHu)
	handler.addPlayer("user-004", constants.FactionXuanWu)

	// Assign factions
	handler.assignFactions()

	// Initialize match
	err := handler.MatchInit()
	if err != nil {
		t.Fatalf("MatchInit error: %v", err)
	}

	// Verify game is created
	if handler.game == nil {
		t.Fatal("game should be created after MatchInit")
	}

	// Verify HSM is running
	if !handler.hsm.IsRunning() {
		t.Error("HSM should be running after MatchInit")
	}

	// Note: MatchInit auto-transitions to RoundMiniGame via Update
	// The state may have already transitioned
	stateID := handler.hsm.GetCurrentStateID()
	// Valid initial states: MatchInit or RoundMiniGame (after auto-transition)
	if stateID != hsm.StateMatchInit && stateID != hsm.StateRoundMiniGame {
		t.Errorf("state = %s, want MatchInit or RoundMiniGame", stateID)
	}

	// Verify map is created
	if handler.mapEngine == nil {
		t.Fatal("mapEngine should be created after MatchInit")
	}

	// Verify players in game
	if len(handler.game.Players) != 4 {
		t.Errorf("game.Players count = %d, want 4", len(handler.game.Players))
	}
}

func TestMatchLoop(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)
	mockDispatcher := NewMockDispatcherAdapter()
	handler.WithDispatcher(mockDispatcher)

	// Setup match
	handler.addPlayer("user-001", constants.FactionQingLong)
	handler.addPlayer("user-002", constants.FactionZhuQue)
	handler.MatchInit()

	// Run one loop tick
	err := handler.MatchLoop(100 * time.Millisecond)
	if err != nil {
		t.Fatalf("MatchLoop error: %v", err)
	}
}

func TestMatchStop(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)
	mockDispatcher := NewMockDispatcherAdapter()
	handler.WithDispatcher(mockDispatcher)

	// Setup match
	handler.addPlayer("user-001", constants.FactionQingLong)
	handler.MatchInit()

	// Stop match
	err := handler.MatchStop()
	if err != nil {
		t.Fatalf("MatchStop error: %v", err)
	}

	// Verify HSM is stopped
	if handler.hsm.IsRunning() {
		t.Error("HSM should not be running after MatchStop")
	}

	// Verify players are cleared
	if len(handler.players) != 0 {
		t.Errorf("players count = %d, want 0", len(handler.players))
	}

	if len(handler.playerList) != 0 {
		t.Errorf("playerList count = %d, want 0", len(handler.playerList))
	}
}

func TestGetCurrentPlayer(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)

	// No players yet
	player := handler.getCurrentPlayer()
	if player != nil {
		t.Error("getCurrentPlayer should return nil with no players")
	}

	// Add players and initialize
	handler.addPlayer("user-001", constants.FactionQingLong)
	handler.addPlayer("user-002", constants.FactionZhuQue)
	handler.MatchInit()

	// Turn 0 should return first player
	player = handler.getCurrentPlayer()
	if player == nil {
		t.Fatal("getCurrentPlayer should return first player")
	}

	if player != handler.game.Players[0] {
		t.Error("current player should be first player (turn 0)")
	}
}

func TestAssignFactions(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)

	// Add 4 players
	handler.addPlayer("user-001", constants.FactionQingLong)
	handler.addPlayer("user-002", constants.FactionZhuQue)
	handler.addPlayer("user-003", constants.FactionBaiHu)
	handler.addPlayer("user-004", constants.FactionXuanWu)

	// Assign factions
	handler.assignFactions()

	// Verify ZhuQue player has Fire buff
	zhuQuePlayer := handler.players["user-002"]
	if zhuQuePlayer == nil {
		t.Fatal("user-002 should exist")
	}

	// Check for Fire buff
	hasFireBuff := false
	for _, buff := range zhuQuePlayer.ActiveBuffs {
		if string(buff.Type) == "fire" {
			hasFireBuff = true
			break
		}
	}

	if !hasFireBuff {
		t.Error("ZhuQue player should have Fire buff (离火 passive)")
	}
}

func TestAddPlayer(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)

	// Add first player
	player1 := handler.addPlayer("user-001", constants.FactionQingLong)
	if player1 == nil {
		t.Fatal("addPlayer should return non-nil player")
	}

	// Verify player properties
	if player1.GetFaction() != constants.FactionQingLong {
		t.Errorf("player faction = %v, want QingLong", player1.GetFaction())
	}

	if player1.HP != 6 {
		t.Errorf("player HP = %d, want 6", player1.HP)
	}

	if player1.LP != 8 {
		t.Errorf("player LP = %d, want 8", player1.LP)
	}

	// Add second player
	player2 := handler.addPlayer("user-002", constants.FactionZhuQue)
	if player2 == nil {
		t.Fatal("addPlayer should return non-nil player")
	}

	// Verify players are stored
	if len(handler.players) != 2 {
		t.Errorf("players count = %d, want 2", len(handler.players))
	}

	if len(handler.playerList) != 2 {
		t.Errorf("playerList count = %d, want 2", len(handler.playerList))
	}
}

func TestFullGameFlow(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 42, 4, 100) // Fixed seed for reproducibility
	mockDispatcher := NewMockDispatcherAdapter()
	handler.WithDispatcher(mockDispatcher)

	// Phase 1: Players join
	handler.addPlayer("user-001", constants.FactionQingLong)
	handler.addPlayer("user-002", constants.FactionZhuQue)
	handler.addPlayer("user-003", constants.FactionBaiHu)
	handler.addPlayer("user-004", constants.FactionXuanWu)

	// Phase 2: Match initialization
	handler.assignFactions()
	err := handler.MatchInit()
	if err != nil {
		t.Fatalf("MatchInit failed: %v", err)
	}

	// Verify initial state
	if !handler.hsm.IsRunning() {
		t.Fatal("HSM should be running")
	}

	// Phase 3: Run several ticks
	for i := 0; i < 5; i++ {
		err = handler.MatchLoop(100 * time.Millisecond)
		if err != nil {
			t.Fatalf("MatchLoop[%d] failed: %v", i, err)
		}
	}

	// Phase 4: Match termination
	err = handler.MatchStop()
	if err != nil {
		t.Fatalf("MatchStop failed: %v", err)
	}

	// Verify final state
	if handler.hsm.IsRunning() {
		t.Error("HSM should not be running after MatchStop")
	}

	if len(handler.players) != 0 {
		t.Error("players should be cleared after MatchStop")
	}
}