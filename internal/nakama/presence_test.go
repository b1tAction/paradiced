// Package nakama provides Nakama match handler implementation for Paradiced.
package nakama

import (
	"testing"

	"github.com/b1tAction/paradiced/pkg/constants"
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
	metadata := map[string]string{"faction": "zhu_que"}
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

	// Verify player is removed
	if len(handler.players) != 1 {
		t.Errorf("players count = %d, want 1", len(handler.players))
	}

	if handler.players["user-001"] != nil {
		t.Error("user-001 should be removed from players map")
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

	// Match should be stopped
	if handler.hsm.IsRunning() {
		t.Error("match should be stopped after all players leave")
	}

	// Players should be cleared
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
		metadata := map[string]string{"faction": factions[i]}
		err := handler.HandlePresenceJoin(userID, metadata)
		if err != nil {
			t.Fatalf("HandlePresenceJoin[%d] error: %v", i, err)
		}
	}

	// When match is full, MatchInit should be called automatically
	// Verify game is initialized
	if handler.game == nil {
		t.Error("game should be initialized when match is full")
	}

	if handler.hsm == nil {
		t.Error("hsm should be initialized when match is full")
	}

	// Verify all 4 players are in game
	if len(handler.game.Players) != 4 {
		t.Errorf("game.Players count = %d, want 4", len(handler.game.Players))
	}
}

func TestFactionAssignmentOrder(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)

	// Add 4 players in order
	handler.addPlayer("user-001", constants.FactionQingLong)
	handler.addPlayer("user-002", constants.FactionQingLong) // Will be reassigned
	handler.addPlayer("user-003", constants.FactionQingLong)
	handler.addPlayer("user-004", constants.FactionQingLong)

	// Assign factions (should assign by join order)
	handler.assignFactions()

	// Verify ZhuQue (index 1) has Fire buff
	zhuQuePlayer := handler.players["user-002"]
	hasFireBuff := false
	for _, buff := range zhuQuePlayer.ActiveBuffs {
		if string(buff.Type) == "fire" {
			hasFireBuff = true
			break
		}
	}

	if !hasFireBuff {
		t.Error("Second player (ZhuQue position) should have Fire buff")
	}
}