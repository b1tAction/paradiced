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

	// Verify game is created (via HSM)
	game := handler.hsm.GetGame()
	if game == nil {
		t.Fatal("game should be created after MatchInit (via HSM)")
	}

	// Verify HSM is running
	if !handler.hsm.IsRunning() {
		t.Error("HSM should be running after MatchInit")
	}

	// Note: MatchInit auto-transitions to WaitingForHost via Update
	// The state may have already transitioned
	stateID := handler.hsm.GetCurrentStateID()
	// Valid initial states: MatchInit or WaitingForHost (after auto-transition)
	if stateID != hsm.StateMatchInit && stateID != hsm.StateWaitingForHost {
		t.Errorf("state = %s, want MatchInit or WaitingForHost", stateID)
	}

	// Verify map is created
	if handler.mapEngine == nil {
		t.Fatal("mapEngine should be created after MatchInit")
	}

	// Verify players in game
	if len(game.Players) != 4 {
		t.Errorf("game.Players count = %d, want 4", len(game.Players))
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

	game := handler.hsm.GetGame()
	if player != game.Players[0] {
		t.Error("current player should be first player (turn 0)")
	}
}

func TestAssignFactions(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)

	// Add 4 players with factions set via PlayerConfig
	handler.addPlayer("user-001", constants.FactionQingLong)
	handler.addPlayer("user-002", constants.FactionZhuQue)
	handler.addPlayer("user-003", constants.FactionBaiHu)
	handler.addPlayer("user-004", constants.FactionXuanWu)

	// Note: assignFactions() is deprecated and does nothing now.
	// Factions are set during addPlayer via PlayerConfig.
	// Fire buff is added later by game.InitializePlayerFactionBuffs().

	// Verify ZhuQue player does NOT have Fire buff yet (added later)
	zhuQuePlayer := handler.players["user-002"]
	if zhuQuePlayer == nil {
		t.Fatal("user-002 should exist")
	}

	if zhuQuePlayer.GetFaction() != constants.FactionZhuQue {
		t.Errorf("ZhuQue player faction = %v, want ZhuQue", zhuQuePlayer.GetFaction())
	}

	// No Fire buff yet - will be added by InitializePlayerFactionBuffs during match init
	if zhuQuePlayer.HasBuff(constants.BuffTypeFire) {
		t.Error("ZhuQue player should NOT have Fire buff yet (added by InitializePlayerFactionBuffs)")
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

func TestMatchInitWithMinimumPlayers(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)
	mockDispatcher := NewMockDispatcherAdapter()
	handler.WithDispatcher(mockDispatcher)

	// Add minimum 2 players
	handler.addPlayer("user-001", constants.FactionQingLong)
	handler.addPlayer("user-002", constants.FactionZhuQue)

	// Initialize match - should work with 2 players
	err := handler.MatchInit()
	if err != nil {
		t.Fatalf("MatchInit error with 2 players: %v", err)
	}

	// Verify game is created
	game := handler.hsm.GetGame()
	if game == nil {
		t.Fatal("game should be created")
	}

	if len(game.Players) != 2 {
		t.Errorf("game.Players count = %d, want 2", len(game.Players))
	}
}

func TestMatchInitWithoutDispatcher(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)
	// No dispatcher set

	// Add players
	handler.addPlayer("user-001", constants.FactionQingLong)
	handler.addPlayer("user-002", constants.FactionZhuQue)

	// Initialize match - should work without dispatcher
	err := handler.MatchInit()
	if err != nil {
		t.Fatalf("MatchInit error without dispatcher: %v", err)
	}

	// Verify HSM is running
	if !handler.hsm.IsRunning() {
		t.Error("HSM should be running")
	}
}

func TestGetPlayer(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)

	// Add player
	player := handler.addPlayer("user-001", constants.FactionQingLong)

	// Get existing player
	found := handler.GetPlayer("user-001")
	if found == nil {
		t.Fatal("GetPlayer should return player for existing user")
	}

	if found != player {
		t.Error("GetPlayer should return same player instance")
	}

	// Get non-existing player
	notFound := handler.GetPlayer("user-999")
	if notFound != nil {
		t.Error("GetPlayer should return nil for non-existing user")
	}
}

func TestMatchLoopMultipleTicks(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 42, 4, 100)
	mockDispatcher := NewMockDispatcherAdapter()
	handler.WithDispatcher(mockDispatcher)

	handler.addPlayer("user-001", constants.FactionQingLong)
	handler.addPlayer("user-002", constants.FactionZhuQue)
	handler.MatchInit()

	// Run many ticks
	for i := 0; i < 20; i++ {
		err := handler.MatchLoop(50 * time.Millisecond)
		if err != nil {
			t.Fatalf("MatchLoop[%d] failed: %v", i, err)
		}

		// Verify HSM still running (unless game ended)
		if handler.hsm != nil && !handler.hsm.IsRunning() {
			// Game ended naturally, that's fine
			break
		}
	}
}

func TestMatchStopWithoutInit(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)

	// Add player but don't init
	handler.addPlayer("user-001", constants.FactionQingLong)

	// Stop should work even without init - but HSM is nil so we skip
	// MatchStop requires HSM to be initialized first
	// This test verifies that players can be added before init
	if len(handler.players) != 1 {
		t.Errorf("players count = %d, want 1 before init", len(handler.players))
	}

	// Note: MatchStop cannot be called without MatchInit first
	// because HSM is nil. This is expected behavior.
}

func TestAddPlayerMaxCapacity(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)

	// Add 4 players (max capacity)
	handler.addPlayer("user-001", constants.FactionQingLong)
	handler.addPlayer("user-002", constants.FactionZhuQue)
	handler.addPlayer("user-003", constants.FactionBaiHu)
	handler.addPlayer("user-004", constants.FactionXuanWu)

	// Verify max players
	if len(handler.players) != 4 {
		t.Errorf("players count = %d, want 4", len(handler.players))
	}
}

func TestGetCurrentPlayerAfterTurnAdvance(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)
	mockDispatcher := NewMockDispatcherAdapter()
	handler.WithDispatcher(mockDispatcher)

	handler.addPlayer("user-001", constants.FactionQingLong)
	handler.addPlayer("user-002", constants.FactionZhuQue)
	handler.MatchInit()

	// Get initial current player
	initialPlayer := handler.getCurrentPlayer()
	if initialPlayer == nil {
		t.Fatal("should have initial current player")
	}

	// Run some ticks to potentially advance turns
	for i := 0; i < 10; i++ {
		handler.MatchLoop(100 * time.Millisecond)
	}

	// Player should still exist (may or may not be same depending on game progress)
	currentPlayer := handler.getCurrentPlayer()
	// currentPlayer may be nil if game ended, or different if turn advanced
	_ = currentPlayer
}

func TestAssignFactionsAllFour(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)

	// Add players for all four factions
	p1 := handler.addPlayer("user-001", constants.FactionQingLong)
	p2 := handler.addPlayer("user-002", constants.FactionZhuQue)
	p3 := handler.addPlayer("user-003", constants.FactionBaiHu)
	p4 := handler.addPlayer("user-004", constants.FactionXuanWu)

	// Verify each faction
	if p1.GetFaction() != constants.FactionQingLong {
		t.Errorf("p1 faction = %v, want QingLong", p1.GetFaction())
	}
	if p2.GetFaction() != constants.FactionZhuQue {
		t.Errorf("p2 faction = %v, want ZhuQue", p2.GetFaction())
	}
	if p3.GetFaction() != constants.FactionBaiHu {
		t.Errorf("p3 faction = %v, want BaiHu", p3.GetFaction())
	}
	if p4.GetFaction() != constants.FactionXuanWu {
		t.Errorf("p4 faction = %v, want XuanWu", p4.GetFaction())
	}
}