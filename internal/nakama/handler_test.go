// Package nakama provides Nakama match handler implementation for Paradiced.
package nakama

import (
	"testing"

	"github.com/b1tAction/paradiced/pkg/constants"
	"github.com/b1tAction/paradiced/pkg/id"
)

func TestNakamaMatchHandlerNew(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)

	if handler == nil {
		t.Fatal("NewNakamaMatchHandler should return non-nil handler")
	}

	if handler.GetMatchID() != "match-001" {
		t.Errorf("matchID = %s, want match-001", handler.GetMatchID())
	}

	if handler.maxPlayers != 4 {
		t.Errorf("maxPlayers = %d, want 4", handler.maxPlayers)
	}

	if handler.mapLength != 100 {
		t.Errorf("mapLength = %d, want 100", handler.mapLength)
	}

	if handler.randomSeed != 12345 {
		t.Errorf("randomSeed = %d, want 12345", handler.randomSeed)
	}

	if len(handler.players) != 0 {
		t.Error("players should be empty initially")
	}

	if len(handler.playerList) != 0 {
		t.Error("playerList should be empty initially")
	}

	if handler.dispatcher != nil {
		t.Error("dispatcher should be nil initially")
	}
}

func TestNakamaMatchHandlerDefaultValues(t *testing.T) {
	// Test with zero values
	handler := NewNakamaMatchHandler("match-002", 0, 0, 0)

	if handler.maxPlayers != 4 {
		t.Errorf("maxPlayers default should be 4, got %d", handler.maxPlayers)
	}

	if handler.mapLength != 100 {
		t.Errorf("mapLength default should be 100, got %d", handler.mapLength)
	}
}

func TestNakamaMatchHandlerWithDispatcher(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)

	mockDispatcher := NewMockDispatcherAdapter()
	handler.WithDispatcher(mockDispatcher)

	if handler.GetDispatcher() != mockDispatcher {
		t.Error("dispatcher should be set correctly")
	}
}

func TestNakamaMatchHandlerAddPlayer(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)

	// Add player
	player := handler.addPlayer(id.TestUUID(1), constants.FactionQingLong, id.TestUUID(1))

	if player == nil {
		t.Fatal("addPlayer should return non-nil player")
	}

	if len(handler.players) != 1 {
		t.Errorf("players count = %d, want 1", len(handler.players))
	}

	if handler.players[id.TestUUID(1)] != player {
		t.Error("player should be stored in players map")
	}

	if len(handler.playerList) != 1 {
		t.Errorf("playerList count = %d, want 1", len(handler.playerList))
	}

	if handler.playerList[0] != id.TestUUID(1) {
		t.Error("playerList should contain user-001")
	}
}

func TestNakamaMatchHandlerGetPlayer(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)

	// Add player
	handler.addPlayer(id.TestUUID(1), constants.FactionQingLong, id.TestUUID(1))

	// Get existing player
	player := handler.GetPlayer(id.TestUUID(1))
	if player == nil {
		t.Fatal("GetPlayer should return existing player")
	}

	// Get non-existing player
	nonExisting := handler.GetPlayer("user-999")
	if nonExisting != nil {
		t.Error("GetPlayer should return nil for non-existing player")
	}
}

func TestNakamaMatchHandlerGetRoundTurn(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)

	// Before initialization
	round := handler.GetRound()
	turn := handler.GetTurn()
	if round != 0 {
		t.Errorf("GetRound should return 0 before HSM init, got %d", round)
	}
	if turn != 0 {
		t.Errorf("GetTurn should return 0 before HSM init, got %d", turn)
	}

	// Add player first
	handler.addPlayer(id.TestUUID(1), constants.FactionQingLong, id.TestUUID(1))

	// Initialize game
	err := handler.initializeGame()
	if err != nil {
		t.Fatalf("initializeGame failed: %v", err)
	}

	// After initialization
	round = handler.GetRound()
	turn = handler.GetTurn()
	// HSM starts with round=1, turn=0
	if round != 1 {
		t.Errorf("Round should be 1 after init, got %d", round)
	}

	if handler.GetTurn() < 0 {
		t.Errorf("Turn should be >= 0, got %d", handler.GetTurn())
	}
}

func TestNakamaMatchHandlerInitializeGame(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)

	// Add players
	handler.addPlayer(id.TestUUID(1), constants.FactionQingLong, id.TestUUID(1))
	handler.addPlayer(id.TestUUID(2), constants.FactionZhuQue, id.TestUUID(2))
	handler.addPlayer(id.TestUUID(3), constants.FactionBaiHu, id.TestUUID(3))
	handler.addPlayer(id.TestUUID(4), constants.FactionXuanWu, id.TestUUID(4))

	// Initialize game
	err := handler.initializeGame()
	if err != nil {
		t.Fatalf("initializeGame failed: %v", err)
	}

	// Verify game is created (via HSM)
	game := handler.hsm.GetGame()
	if game == nil {
		t.Fatal("game should be created after initialization (via HSM)")
	}

	// Verify HSM is created
	if handler.hsm == nil {
		t.Fatal("hsm should be created after initialization")
	}

	// Verify mapEngine is created
	if handler.mapEngine == nil {
		t.Fatal("mapEngine should be created after initialization")
	}

	// Verify players added to game (4 human players + 1 Boss)
	if len(game.Players) != 5 {
		t.Errorf("game.Players count = %d, want 5 (4 human + 1 Boss)", len(game.Players))
	}
}

func TestNakamaMatchHandlerGetMatchID(t *testing.T) {
	handler := NewNakamaMatchHandler("test-match-id", 0, 4, 100)

	if handler.GetMatchID() != "test-match-id" {
		t.Errorf("GetMatchID = %s, want test-match-id", handler.GetMatchID())
	}
}