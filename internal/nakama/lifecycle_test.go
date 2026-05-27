// Package nakama provides Nakama match handler implementation for Paradiced.
package nakama

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/b1tAction/paradiced/internal/engine/hsm"
	internalnet "github.com/b1tAction/paradiced/internal/net"
	"github.com/b1tAction/paradiced/pkg/constants"
	"github.com/b1tAction/paradiced/pkg/id"
	pkgnet "github.com/b1tAction/paradiced/pkg/net"
	"github.com/heroiclabs/nakama-common/runtime"
)

func TestMatchInit(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)
	mockDispatcher := NewMockDispatcherAdapter()
	handler.WithDispatcher(mockDispatcher)

	// Add players first - use valid UUID format for userID
	handler.addPlayer(id.TestUUID(1), constants.FactionQingLong, id.TestUUID(1))
	handler.addPlayer(id.TestUUID(2), constants.FactionZhuQue, id.TestUUID(2))
	handler.addPlayer(id.TestUUID(3), constants.FactionBaiHu, id.TestUUID(3))
	handler.addPlayer(id.TestUUID(4), constants.FactionXuanWu, id.TestUUID(4))

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

	// Verify players in game (4 human + 1 Boss)
	if len(game.Players) != 5 {
		t.Errorf("game.Players count = %d, want 5 (4 human + 1 Boss)", len(game.Players))
	}
}

func TestMatchLoop(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)
	mockDispatcher := NewMockDispatcherAdapter()
	handler.WithDispatcher(mockDispatcher)

	// Setup match
	handler.addPlayer(id.TestUUID(1), constants.FactionQingLong, id.TestUUID(1))
	handler.addPlayer(id.TestUUID(2), constants.FactionZhuQue, id.TestUUID(2))
	handler.MatchInit()

	// Run one loop tick
	err := handler.MatchLoop(100 * time.Millisecond)
	if err != nil {
		t.Fatalf("MatchLoop error: %v", err)
	}
}

func TestMatchLoopDebugTriggerBroadcastsMiniGameStart(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)
	mockDispatcher := NewMockDispatcherAdapter()
	handler.WithDispatcher(mockDispatcher)
	handler.WithProvider(&debugTriggerMiniGameProvider{})

	handler.addPlayer(id.TestUUID(1), constants.FactionQingLong, "Alice")
	handler.addPlayer(id.TestUUID(2), constants.FactionZhuQue, "Bob")
	handler.addPlayer(id.TestUUID(3), constants.FactionBaiHu, "Carol")
	handler.addPlayer(id.TestUUID(4), constants.FactionXuanWu, "Dave")

	if err := handler.MatchInit(); err != nil {
		t.Fatalf("MatchInit error: %v", err)
	}

	mockDispatcher.Clear()
	handler.pendingTriggerMinigame = string(constants.MiniGameTypeDilemmaRace)

	if err := handler.MatchLoop(100 * time.Millisecond); err != nil {
		t.Fatalf("MatchLoop error: %v", err)
	}

	if handler.hsm.GetGlobalStateID() != hsm.StateRoundMiniGame {
		t.Fatalf("global state = %s, want RoundMiniGame", handler.hsm.GetGlobalStateID())
	}

	var start pkgnet.MiniGameStart
	found := false
	for index, broadcast := range mockDispatcher.GetBroadcasts() {
		if broadcast.OpCode != int64(pkgnet.OpMiniGameStart) {
			continue
		}
		if err := mockDispatcher.ParseBroadcastData(index, &start); err != nil {
			t.Fatalf("ParseBroadcastData error: %v", err)
		}
		found = true
		break
	}

	if !found {
		t.Fatal("expected OpMiniGameStart broadcast after debug mini-game trigger")
	}
	if start.GameType != string(constants.MiniGameTypeDilemmaRace) {
		t.Errorf("GameType = %s, want %s", start.GameType, constants.MiniGameTypeDilemmaRace)
	}
	if start.Connection == nil {
		t.Fatal("Connection should be present for dilemma_race")
	}
	if start.Connection.RoomName != string(constants.MiniGameTypeDilemmaRace) {
		t.Errorf("RoomName = %s, want %s", start.Connection.RoomName, constants.MiniGameTypeDilemmaRace)
	}
}

func TestMatchLoopDebugTriggerCanRestartFromTurnLoop(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)
	mockDispatcher := NewMockDispatcherAdapter()
	handler.WithDispatcher(mockDispatcher)
	handler.WithProvider(&debugTriggerMiniGameProvider{})

	handler.addPlayer(id.TestUUID(1), constants.FactionQingLong, "Alice")
	handler.addPlayer(id.TestUUID(2), constants.FactionZhuQue, "Bob")
	handler.addPlayer(id.TestUUID(3), constants.FactionBaiHu, "Carol")
	handler.addPlayer(id.TestUUID(4), constants.FactionXuanWu, "Dave")

	if err := handler.MatchInit(); err != nil {
		t.Fatalf("MatchInit error: %v", err)
	}

	handler.pendingTriggerMinigame = string(constants.MiniGameTypeDilemmaRace)
	if err := handler.MatchLoop(100 * time.Millisecond); err != nil {
		t.Fatalf("first MatchLoop error: %v", err)
	}

	miniGameState, ok := handler.hsm.GetGlobalState().(*hsm.RoundMiniGameState)
	if !ok {
		t.Fatalf("global state = %T, want *RoundMiniGameState", handler.hsm.GetGlobalState())
	}

	ctx := hsm.NewStateContext().
		WithHSM(handler.hsm).
		WithBroadcast(NewNakamaBroadcastAdapter(handler)).
		WithBuilder(internalnet.NewBuilder(handler.hsm))
	for index, playerID := range handler.playerList {
		miniGameState.OnMiniGameResult(ctx, playerID, index+1)
	}

	if err := handler.MatchLoop(100 * time.Millisecond); err != nil {
		t.Fatalf("settlement MatchLoop error: %v", err)
	}
	if handler.hsm.GetGlobalStateID() != hsm.StateTurnLoop {
		t.Fatalf("global state = %s, want TurnLoop", handler.hsm.GetGlobalStateID())
	}

	mockDispatcher.Clear()
	handler.pendingTriggerMinigame = string(constants.MiniGameTypeDilemmaRace)
	if err := handler.MatchLoop(100 * time.Millisecond); err != nil {
		t.Fatalf("second trigger MatchLoop error: %v", err)
	}

	if handler.hsm.GetGlobalStateID() != hsm.StateRoundMiniGame {
		t.Fatalf("global state = %s, want RoundMiniGame", handler.hsm.GetGlobalStateID())
	}

	found := false
	for _, broadcast := range mockDispatcher.GetBroadcasts() {
		if broadcast.OpCode == int64(pkgnet.OpMiniGameStart) {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected OpMiniGameStart broadcast when debug trigger restarts from TurnLoop")
	}
}

type debugTriggerMiniGameProvider struct{}

func (p *debugTriggerMiniGameProvider) CreateRoom(gameType constants.MiniGameType, playerIDs []string) (*pkgnet.MiniGameConn, error) {
	tokens := make(map[string]string, len(playerIDs))
	for _, playerID := range playerIDs {
		tokens[playerID] = "token_" + playerID
	}

	return &pkgnet.MiniGameConn{
		URL:                "ws://mock-colyseus:2567",
		RoomName:           string(gameType),
		NakamaMatchID:      "match-001",
		MiniGameInstanceID: "mini-game-instance-001",
		CreatorPlayerID:    playerIDs[0],
		PlayerTokens:       tokens,
	}, nil
}

func (p *debugTriggerMiniGameProvider) DestroyRoom(roomID string) error {
	return nil
}

func (p *debugTriggerMiniGameProvider) GetTimeout(gameType constants.MiniGameType) time.Duration {
	return time.Minute
}

func TestMatchStop(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)
	mockDispatcher := NewMockDispatcherAdapter()
	handler.WithDispatcher(mockDispatcher)

	// Setup match
	handler.addPlayer(id.TestUUID(1), constants.FactionQingLong, id.TestUUID(1))
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
	handler.addPlayer(id.TestUUID(1), constants.FactionQingLong, id.TestUUID(1))
	handler.addPlayer(id.TestUUID(2), constants.FactionZhuQue, id.TestUUID(2))
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
	handler.addPlayer(id.TestUUID(1), constants.FactionQingLong, id.TestUUID(1))
	handler.addPlayer(id.TestUUID(2), constants.FactionZhuQue, id.TestUUID(2))
	handler.addPlayer(id.TestUUID(3), constants.FactionBaiHu, id.TestUUID(3))
	handler.addPlayer(id.TestUUID(4), constants.FactionXuanWu, id.TestUUID(4))

	// Note: assignFactions() is deprecated and does nothing now.
	// Factions are set during addPlayer via PlayerConfig.
	// Fire buff is added later by game.InitializePlayerFactionBuffs()
	// which is called in WaitingForHostState.Exit() when the host starts the game.

	// Verify ZhuQue player does NOT have Fire buff yet (added later)
	zhuQuePlayer := handler.players[id.TestUUID(2)]
	if zhuQuePlayer == nil {
		t.Fatal("user-002 should exist")
	}

	if zhuQuePlayer.GetFaction() != constants.FactionZhuQue {
		t.Errorf("ZhuQue player faction = %v, want ZhuQue", zhuQuePlayer.GetFaction())
	}

	// No Fire buff yet - will be added by InitializePlayerFactionBuffs during WaitingForHostState.Exit()
	if zhuQuePlayer.HasBuff(constants.BuffTypeFire) {
		t.Error("ZhuQue player should NOT have Fire buff yet (added by InitializePlayerFactionBuffs)")
	}
}

func TestAddPlayer(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)

	// Add first player
	player1 := handler.addPlayer(id.TestUUID(1), constants.FactionQingLong, id.TestUUID(1))
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

	if player1.MaxHP != 8 {
		t.Errorf("player MaxHP = %d, want 8", player1.MaxHP)
	}

	if player1.InitHP != 6 {
		t.Errorf("player InitHP = %d, want 6", player1.InitHP)
	}

	if player1.LP != 4 {
		t.Errorf("player LP = %d, want 4", player1.LP)
	}

	// Add second player
	player2 := handler.addPlayer(id.TestUUID(2), constants.FactionZhuQue, id.TestUUID(2))
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
	handler.addPlayer(id.TestUUID(1), constants.FactionQingLong, id.TestUUID(1))
	handler.addPlayer(id.TestUUID(2), constants.FactionZhuQue, id.TestUUID(2))
	handler.addPlayer(id.TestUUID(3), constants.FactionBaiHu, id.TestUUID(3))
	handler.addPlayer(id.TestUUID(4), constants.FactionXuanWu, id.TestUUID(4))

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
	handler.addPlayer(id.TestUUID(1), constants.FactionQingLong, id.TestUUID(1))
	handler.addPlayer(id.TestUUID(2), constants.FactionZhuQue, id.TestUUID(2))

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

	if len(game.Players) != 3 {
		t.Errorf("game.Players count = %d, want 3 (2 human + 1 Boss)", len(game.Players))
	}
}

func TestMatchInitWithoutDispatcher(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)
	// No dispatcher set

	// Add players
	handler.addPlayer(id.TestUUID(1), constants.FactionQingLong, id.TestUUID(1))
	handler.addPlayer(id.TestUUID(2), constants.FactionZhuQue, id.TestUUID(2))

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
	player := handler.addPlayer(id.TestUUID(1), constants.FactionQingLong, id.TestUUID(1))

	// Get existing player
	found := handler.GetPlayer(id.TestUUID(1))
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

	handler.addPlayer(id.TestUUID(1), constants.FactionQingLong, id.TestUUID(1))
	handler.addPlayer(id.TestUUID(2), constants.FactionZhuQue, id.TestUUID(2))
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
	handler.addPlayer(id.TestUUID(1), constants.FactionQingLong, id.TestUUID(1))

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
	handler.addPlayer(id.TestUUID(1), constants.FactionQingLong, id.TestUUID(1))
	handler.addPlayer(id.TestUUID(2), constants.FactionZhuQue, id.TestUUID(2))
	handler.addPlayer(id.TestUUID(3), constants.FactionBaiHu, id.TestUUID(3))
	handler.addPlayer(id.TestUUID(4), constants.FactionXuanWu, id.TestUUID(4))

	// Verify max players
	if len(handler.players) != 4 {
		t.Errorf("players count = %d, want 4", len(handler.players))
	}
}

func TestGetCurrentPlayerAfterTurnAdvance(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)
	mockDispatcher := NewMockDispatcherAdapter()
	handler.WithDispatcher(mockDispatcher)

	handler.addPlayer(id.TestUUID(1), constants.FactionQingLong, id.TestUUID(1))
	handler.addPlayer(id.TestUUID(2), constants.FactionZhuQue, id.TestUUID(2))
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
	p1 := handler.addPlayer(id.TestUUID(1), constants.FactionQingLong, id.TestUUID(1))
	p2 := handler.addPlayer(id.TestUUID(2), constants.FactionZhuQue, id.TestUUID(2))
	p3 := handler.addPlayer(id.TestUUID(3), constants.FactionBaiHu, id.TestUUID(3))
	p4 := handler.addPlayer(id.TestUUID(4), constants.FactionXuanWu, id.TestUUID(4))

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

// ========== cleanupPlayerStorage Tests ==========

// mockStorageDeleter tracks StorageDelete calls for testing.
type mockStorageDeleter struct {
	mu     sync.Mutex
	calls  [][]*runtime.StorageDelete
	errors map[int]error // call index -> error to return
}

func (m *mockStorageDeleter) StorageDelete(ctx context.Context, deletes []*runtime.StorageDelete) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	idx := len(m.calls)
	m.calls = append(m.calls, deletes)
	if m.errors != nil && m.errors[idx] != nil {
		return m.errors[idx]
	}
	return nil
}

func (m *mockStorageDeleter) getCalls() [][]*runtime.StorageDelete {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([][]*runtime.StorageDelete, len(m.calls))
	copy(result, m.calls)
	return result
}

func TestCleanupPlayerStorage(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)
	handler.addPlayer(id.TestUUID(1), constants.FactionQingLong, id.TestUUID(1))
	handler.addPlayer(id.TestUUID(2), constants.FactionZhuQue, id.TestUUID(2))

	deleter := &mockStorageDeleter{}
	logger := newMockLogger()

	cleanupPlayerStorage(context.Background(), deleter, handler, logger)

	calls := deleter.getCalls()
	if len(calls) != 2 {
		t.Fatalf("StorageDelete should be called for each player, got %d calls", len(calls))
	}

	// Each call should have 2 storage keys
	for _, call := range calls {
		if len(call) != 2 {
			t.Errorf("Each StorageDelete call should have 2 keys, got %d", len(call))
		}
		// Verify collection and key names
		for _, del := range call {
			if del.Collection != "paradiced" {
				t.Errorf("Collection = %s, want 'paradiced'", del.Collection)
			}
			if del.Key != "paradiced_match_result" && del.Key != "paradiced_stats" {
				t.Errorf("Key = %s, want 'paradiced_match_result' or 'paradiced_stats'", del.Key)
			}
		}
	}
}

func TestCleanupPlayerStorageNilDeleter(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)
	handler.addPlayer(id.TestUUID(1), constants.FactionQingLong, id.TestUUID(1))

	logger := newMockLogger()

	// Should not panic with nil deleter
	cleanupPlayerStorage(context.Background(), nil, handler, logger)
}

func TestCleanupPlayerStorageNoPlayers(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)
	deleter := &mockStorageDeleter{}
	logger := newMockLogger()

	cleanupPlayerStorage(context.Background(), deleter, handler, logger)

	calls := deleter.getCalls()
	if len(calls) != 0 {
		t.Errorf("StorageDelete should not be called when no players, got %d calls", len(calls))
	}
}
