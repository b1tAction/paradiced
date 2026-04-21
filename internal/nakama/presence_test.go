// Package nakama provides Nakama match handler implementation for Paradiced.
package nakama

import (
	"testing"

	"github.com/b1tAction/paradiced/pkg/constants"
	"github.com/b1tAction/paradiced/pkg/util"
)

func TestHandlePresenceJoin(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)
	mockDispatcher := NewMockDispatcherAdapter()
	handler.WithDispatcher(mockDispatcher)

	// Player joins
	err := handler.HandlePresenceJoin("user-001", nil)
	if err != nil {
		t.Fatalf("HandlePresenceJoin error: %v", err)
	}

	// Verify player is added
	if len(handler.players) != 1 {
		t.Errorf("players count = %d, want 1", len(handler.players))
	}

	if handler.players["user-001"] == nil {
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
	err := handler.HandlePresenceJoin("user-001", metadata)
	if err != nil {
		t.Fatalf("HandlePresenceJoin error: %v", err)
	}

	// Verify player faction
	player := handler.GetPlayer("user-001")
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
	err := handler.HandlePresenceJoin("user-001", nil)
	if err != nil {
		t.Fatalf("first HandlePresenceJoin error: %v", err)
	}

	// Second join (duplicate)
	err = handler.HandlePresenceJoin("user-001", nil)
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
	handler.HandlePresenceJoin("user-001", nil)
	handler.HandlePresenceJoin("user-002", nil)
	handler.HandlePresenceJoin("user-003", nil)
	handler.HandlePresenceJoin("user-004", nil)

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
	handler.HandlePresenceJoin("user-001", nil)
	handler.HandlePresenceJoin("user-002", nil)

	// Player leaves
	err := handler.HandlePresenceLeave("user-001")
	if err != nil {
		t.Fatalf("HandlePresenceLeave error: %v", err)
	}

	// Verify player is marked as disconnected (not removed)
	if len(handler.players) != 2 {
		t.Errorf("players count = %d, want 2 (disconnected players are kept for rejoin)", len(handler.players))
	}

	if handler.players["user-001"] == nil {
		t.Error("user-001 should still be in players map (for rejoin)")
	}

	if !handler.disconnected["user-001"] {
		t.Error("user-001 should be marked as disconnected")
	}

	// playerList should remain unchanged (for turn order)
	if len(handler.playerList) != 2 {
		t.Errorf("playerList count = %d, want 2", len(handler.playerList))
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
	handler.HandlePresenceJoin("user-001", nil)
	handler.HandlePresenceJoin("user-002", nil)
	handler.HandlePresenceJoin("user-003", nil)
	handler.HandlePresenceJoin("user-004", nil)

	// Verify match is initialized
	if handler.hsm == nil || !handler.hsm.IsRunning() {
		t.Fatal("match should be running after 4 players join")
	}

	// Player disconnects
	err := handler.HandlePresenceLeave("user-001")
	if err != nil {
		t.Fatalf("HandlePresenceLeave error: %v", err)
	}

	// Verify player is marked disconnected
	if !handler.disconnected["user-001"] {
		t.Error("user-001 should be marked as disconnected")
	}

	// Player reconnects
	err = handler.HandlePresenceJoin("user-001", nil)
	if err != nil {
		t.Fatalf("HandlePresenceJoin reconnect error: %v", err)
	}

	// Verify player is no longer disconnected
	if handler.disconnected["user-001"] {
		t.Error("user-001 should not be marked as disconnected after rejoin")
	}

	// Verify match is still running
	if !handler.hsm.IsRunning() {
		t.Error("match should still be running after reconnect")
	}
}

func TestIsPlayerConnected(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)

	// Add player
	handler.HandlePresenceJoin("user-001", nil)

	// Player should be connected
	if !handler.IsPlayerConnected("user-001") {
		t.Error("user-001 should be connected after join")
	}

	// Mark player as disconnected directly (without calling HandlePresenceLeave which needs HSM)
	handler.disconnected["user-001"] = true

	// Player should be disconnected
	if handler.IsPlayerConnected("user-001") {
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
	handler.HandlePresenceJoin("user-001", nil)
	handler.HandlePresenceJoin("user-002", nil)
	handler.HandlePresenceJoin("user-003", nil)

	// All should be connected
	connected := handler.GetConnectedPlayers()
	if len(connected) != 3 {
		t.Errorf("connected players count = %d, want 3", len(connected))
	}

	// Disconnect one
	handler.HandlePresenceLeave("user-001")

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
	handler.HandlePresenceJoin("user-001", nil)
	handler.MatchInit()

	// Verify match is running
	if handler.hsm == nil || !handler.hsm.IsRunning() {
		t.Fatal("match should be running after initialization")
	}

	// Last player leaves
	err := handler.HandlePresenceLeave("user-001")
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

func TestMatchFullPlayerJoin(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)
	mockDispatcher := NewMockDispatcherAdapter()
	handler.WithDispatcher(mockDispatcher)

	// 4 players join sequentially
	factions := []string{"qing_long", "zhu_que", "bai_hu", "xuan_wu"}
	for i := 0; i < 4; i++ {
		userID := "user-" + string(rune('0'+'1'+i))
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

	// Verify all 4 players are in game
	if len(game.Players) != 4 {
		t.Errorf("game.Players count = %d, want 4", len(game.Players))
	}
}

// Note: Faction assignment order testing is no longer relevant.
// Factions are now set during addPlayer via PlayerConfig, not reassigned by assignFactions.
// The assignFactions() method is deprecated and does nothing.
// For testing InitializePlayerFactionBuffs, see engine/game_test.go.

func TestFactionSetDuringAddPlayer(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)

	// Add players with specific factions
	handler.addPlayer("user-001", constants.FactionQingLong, "user-001")
	handler.addPlayer("user-002", constants.FactionZhuQue, "user-002")
	handler.addPlayer("user-003", constants.FactionBaiHu, "user-003")
	handler.addPlayer("user-004", constants.FactionXuanWu, "user-004")

	// Verify factions are set correctly
	if handler.players["user-001"].GetFaction() != constants.FactionQingLong {
		t.Error("user-001 should be QingLong")
	}
	if handler.players["user-002"].GetFaction() != constants.FactionZhuQue {
		t.Error("user-002 should be ZhuQue")
	}
	if handler.players["user-003"].GetFaction() != constants.FactionBaiHu {
		t.Error("user-003 should be BaiHu")
	}
	if handler.players["user-004"].GetFaction() != constants.FactionXuanWu {
		t.Error("user-004 should be XuanWu")
	}

	// Note: Fire buff is NOT added here - it's added by InitializePlayerFactionBuffs
	// during match initialization (see engine/game_test.go)
}