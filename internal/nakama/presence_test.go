// Package nakama provides Nakama match handler implementation for Paradiced.
package nakama

import (
	"testing"

	"github.com/b1tAction/paradiced/internal/engine/hsm"
	"github.com/b1tAction/paradiced/pkg/constants"
	"github.com/b1tAction/paradiced/pkg/id"
	"github.com/b1tAction/paradiced/pkg/util"
)

func TestHandlePresenceJoin(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)
	mockDispatcher := NewMockDispatcherAdapter()
	handler.WithDispatcher(mockDispatcher)

	// Player joins
	err := handler.HandlePresenceJoin(id.TestUUID(1), nil)
	if err != nil {
		t.Fatalf("HandlePresenceJoin error: %v", err)
	}

	// Verify player is added
	if len(handler.players) != 1 {
		t.Errorf("players count = %d, want 1", len(handler.players))
	}

	if handler.players[id.TestUUID(1)] == nil {
		t.Error("user-001 should be in players map")
	}

	if len(handler.playerList) != 1 {
		t.Errorf("playerList count = %d, want 1", len(handler.playerList))
	}
}

func TestHandlePresenceJoinWithFaction(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)

	// Player joins with faction metadata
	metadata := util.NewMetadata()
	metadata.SetString("faction", "zhu_que")
	err := handler.HandlePresenceJoin(id.TestUUID(1), metadata)
	if err != nil {
		t.Fatalf("HandlePresenceJoin error: %v", err)
	}

	// Verify player faction
	player := handler.GetPlayer(id.TestUUID(1))
	if player == nil {
		t.Fatal("player should exist")
	}

	// Faction should be ZhuQue from metadata
	if player.GetFaction() != constants.FactionZhuQue {
		t.Errorf("player faction = %v, want ZhuQue", player.GetFaction())
	}
}

func TestHandlePresenceJoinDuplicate(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)

	// First join
	err := handler.HandlePresenceJoin(id.TestUUID(1), nil)
	if err != nil {
		t.Fatalf("first HandlePresenceJoin error: %v", err)
	}

	// Second join (duplicate)
	err = handler.HandlePresenceJoin(id.TestUUID(1), nil)
	if err != nil {
		t.Fatalf("duplicate HandlePresenceJoin should not error, got: %v", err)
	}

	// Should still have only 1 player
	if len(handler.players) != 1 {
		t.Errorf("players count = %d, want 1 (no duplicate)", len(handler.players))
	}
}

func TestHandlePresenceJoinMatchFull(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100) // max 4 players

	// Add 4 players
	handler.HandlePresenceJoin(id.TestUUID(1), nil)
	handler.HandlePresenceJoin(id.TestUUID(2), nil)
	handler.HandlePresenceJoin(id.TestUUID(3), nil)
	handler.HandlePresenceJoin(id.TestUUID(4), nil)

	// Try to add 5th player
	err := handler.HandlePresenceJoin("user-005", nil)
	if err == nil {
		t.Error("HandlePresenceJoin should fail when match is full")
	}

	// Should still have only 4 players
	if len(handler.players) != 4 {
		t.Errorf("players count = %d, want 4", len(handler.players))
	}
}

func TestHandlePresenceLeave(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)
	mockDispatcher := NewMockDispatcherAdapter()
	handler.WithDispatcher(mockDispatcher)

	// Add player first
	handler.HandlePresenceJoin(id.TestUUID(1), nil)
	handler.HandlePresenceJoin(id.TestUUID(2), nil)

	// Non-host player leaves
	err := handler.HandlePresenceLeave(id.TestUUID(2))
	if err != nil {
		t.Fatalf("HandlePresenceLeave error: %v", err)
	}

	// In waiting room, non-host leave should remove from room immediately.
	if len(handler.players) != 1 {
		t.Errorf("players count = %d, want 1 (leaver removed from waiting room)", len(handler.players))
	}

	if handler.players[id.TestUUID(2)] != nil {
		t.Error("user-002 should be removed from players map")
	}

	if handler.disconnected[id.TestUUID(2)] {
		t.Error("user-002 should not be kept in disconnected map in waiting room")
	}

	if len(handler.playerList) != 1 {
		t.Errorf("playerList count = %d, want 1", len(handler.playerList))
	}
}

func TestHandlePresenceLeaveNonExisting(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)

	// Remove non-existing player
	err := handler.HandlePresenceLeave("user-999")
	if err != nil {
		t.Fatalf("HandlePresenceLeave of non-existing player should not error, got: %v", err)
	}
}

func TestPlayerReconnect(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)
	mockDispatcher := NewMockDispatcherAdapter()
	handler.WithDispatcher(mockDispatcher)

	// Add players and initialize match
	handler.HandlePresenceJoin(id.TestUUID(1), nil)
	handler.HandlePresenceJoin(id.TestUUID(2), nil)
	handler.HandlePresenceJoin(id.TestUUID(3), nil)
	handler.HandlePresenceJoin(id.TestUUID(4), nil)

	// Verify match is initialized
	if handler.hsm == nil || !handler.hsm.IsRunning() {
		t.Fatal("match should be running after 4 players join")
	}

	// Non-host player disconnects
	err := handler.HandlePresenceLeave(id.TestUUID(2))
	if err != nil {
		t.Fatalf("HandlePresenceLeave error: %v", err)
	}

	// Waiting room leave removes player completely.
	if handler.players[id.TestUUID(2)] != nil {
		t.Error("user-002 should be removed from players map after leave in waiting room")
	}

	// Player joins again
	err = handler.HandlePresenceJoin(id.TestUUID(2), nil)
	if err != nil {
		t.Fatalf("HandlePresenceJoin rejoin error: %v", err)
	}

	if handler.players[id.TestUUID(2)] == nil {
		t.Error("user-002 should be added back after rejoin")
	}

	// Verify match is still running
	if !handler.hsm.IsRunning() {
		t.Error("match should still be running after reconnect")
	}
}

func TestIsPlayerConnected(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)

	// Add player
	handler.HandlePresenceJoin(id.TestUUID(1), nil)

	// Player should be connected
	if !handler.IsPlayerConnected(id.TestUUID(1)) {
		t.Error("user-001 should be connected after join")
	}

	// Mark player as disconnected directly (without calling HandlePresenceLeave which needs HSM)
	handler.disconnected[id.TestUUID(1)] = true

	// Player should be disconnected
	if handler.IsPlayerConnected(id.TestUUID(1)) {
		t.Error("user-001 should be disconnected after leave")
	}

	// Unknown player should return false
	if handler.IsPlayerConnected("user-999") {
		t.Error("unknown player should return false")
	}
}

func TestGetConnectedPlayers(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)

	// Add players
	handler.HandlePresenceJoin(id.TestUUID(1), nil)
	handler.HandlePresenceJoin(id.TestUUID(2), nil)
	handler.HandlePresenceJoin(id.TestUUID(3), nil)

	// All should be connected
	connected := handler.GetConnectedPlayers()
	if len(connected) != 3 {
		t.Errorf("connected players count = %d, want 3", len(connected))
	}

	// Disconnect one (non-host)
	handler.HandlePresenceLeave(id.TestUUID(2))

	// Should have 2 connected
	connected = handler.GetConnectedPlayers()
	if len(connected) != 2 {
		t.Errorf("connected players count = %d, want 2", len(connected))
	}
}

func TestHandlePresenceLeaveAllPlayers(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)
	mockDispatcher := NewMockDispatcherAdapter()
	handler.WithDispatcher(mockDispatcher)

	// Add player and initialize match
	handler.HandlePresenceJoin(id.TestUUID(1), nil)
	handler.MatchInit()

	// Verify match is running
	if handler.hsm == nil || !handler.hsm.IsRunning() {
		t.Fatal("match should be running after initialization")
	}

	// Last player leaves
	err := handler.HandlePresenceLeave(id.TestUUID(1))
	if err != nil {
		t.Fatalf("HandlePresenceLeave error: %v", err)
	}

	// Match should be stopped when all players are disconnected
	if handler.hsm.IsRunning() {
		t.Error("match should be stopped after all players disconnect")
	}

	// Players should be cleared after MatchStop
	if len(handler.players) != 0 {
		t.Errorf("players count = %d, want 0", len(handler.players))
	}
}

func TestHandlePresenceLeaveHostTerminatesMatch(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)
	mockDispatcher := NewMockDispatcherAdapter()
	handler.WithDispatcher(mockDispatcher)

	// First join becomes host.
	handler.HandlePresenceJoin(id.TestUUID(1), nil)
	handler.HandlePresenceJoin(id.TestUUID(2), nil)

	if handler.hostUserID != id.TestUUID(1) {
		t.Fatalf("hostUserID = %s, want user-001", handler.hostUserID)
	}

	err := handler.HandlePresenceLeave(id.TestUUID(1))
	if err != nil {
		t.Fatalf("HandlePresenceLeave(host) error: %v", err)
	}

	if len(handler.players) != 0 {
		t.Errorf("players count = %d, want 0 after host leave termination", len(handler.players))
	}
	if len(handler.playerList) != 0 {
		t.Errorf("playerList count = %d, want 0 after host leave termination", len(handler.playerList))
	}
}

func TestHandlePresenceLeaveHostDuringGameDoesNotTerminate(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)
	mockDispatcher := NewMockDispatcherAdapter()
	handler.WithDispatcher(mockDispatcher)

	// Add 4 players and start the game flow.
	handler.HandlePresenceJoin(id.TestUUID(1), nil) // host
	handler.HandlePresenceJoin(id.TestUUID(2), nil)
	handler.HandlePresenceJoin(id.TestUUID(3), nil)
	handler.HandlePresenceJoin(id.TestUUID(4), nil)

	if handler.hsm == nil || !handler.hsm.IsRunning() {
		t.Fatal("match should be running")
	}

	// Move out of waiting room state to simulate an active game.
	ctx := hsm.NewStateContext().WithHSM(handler.hsm)
	if err := handler.hsm.TransitionTo(hsm.StateRoundMiniGame, ctx); err != nil {
		t.Fatalf("failed to transition out of waiting state: %v", err)
	}

	err := handler.HandlePresenceLeave(id.TestUUID(1))
	if err != nil {
		t.Fatalf("HandlePresenceLeave(host during game) error: %v", err)
	}

	// Should not terminate the whole room.
	if len(handler.players) == 0 {
		t.Fatal("players map should not be cleared when host leaves during game")
	}
	if handler.players[id.TestUUID(1)] == nil {
		t.Fatal("host should remain in players map as disconnected")
	}
	if !handler.disconnected[id.TestUUID(1)] {
		t.Fatal("host should be marked disconnected")
	}
}

func TestMatchFullPlayerJoin(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)
	mockDispatcher := NewMockDispatcherAdapter()
	handler.WithDispatcher(mockDispatcher)

	// 4 players join sequentially
	factions := []string{"qing_long", "zhu_que", "bai_hu", "xuan_wu"}
	for i := 0; i < 4; i++ {
		userID := id.TestUUID(i + 1)
		metadata := util.NewMetadata()
		metadata.SetString("faction", factions[i])
		err := handler.HandlePresenceJoin(userID, metadata)
		if err != nil {
			t.Fatalf("HandlePresenceJoin[%d] error: %v", i, err)
		}
	}

	// When match is full, MatchInit should be called automatically
	// Verify game is initialized (via HSM)
	game := handler.hsm.GetGame()
	if game == nil {
		t.Error("game should be initialized when match is full (via HSM)")
	}

	if handler.hsm == nil {
		t.Error("hsm should be initialized when match is full")
	}

	// Verify all 4 human players + Boss are in game
	if len(game.Players) != 5 {
		t.Errorf("game.Players count = %d, want 5 (4 human + 1 Boss)", len(game.Players))
	}
}

// Note: Faction assignment order testing is no longer relevant.
// Factions are now set during addPlayer via PlayerConfig, not reassigned by assignFactions.
// The assignFactions() method is deprecated and does nothing.
// For testing InitializePlayerFactionBuffs, see engine/game_test.go.

func TestFactionSetDuringAddPlayer(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)

	// Add players with specific factions
	handler.addPlayer(id.TestUUID(1), constants.FactionQingLong, id.TestUUID(1))
	handler.addPlayer(id.TestUUID(2), constants.FactionZhuQue, id.TestUUID(2))
	handler.addPlayer(id.TestUUID(3), constants.FactionBaiHu, id.TestUUID(3))
	handler.addPlayer(id.TestUUID(4), constants.FactionXuanWu, id.TestUUID(4))

	// Verify factions are set correctly
	if handler.players[id.TestUUID(1)].GetFaction() != constants.FactionQingLong {
		t.Error("user-001 should be QingLong")
	}
	if handler.players[id.TestUUID(2)].GetFaction() != constants.FactionZhuQue {
		t.Error("user-002 should be ZhuQue")
	}
	if handler.players[id.TestUUID(3)].GetFaction() != constants.FactionBaiHu {
		t.Error("user-003 should be BaiHu")
	}
	if handler.players[id.TestUUID(4)].GetFaction() != constants.FactionXuanWu {
		t.Error("user-004 should be XuanWu")
	}

	// Note: Fire buff is NOT added here - it's added by InitializePlayerFactionBuffs
	// during match initialization (see engine/game_test.go)
}
