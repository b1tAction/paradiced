package hsm

import (
	"sort"
	"testing"
	"time"

	"github.com/b1tAction/paradiced/internal/core"
	"github.com/b1tAction/paradiced/internal/engine"
	"github.com/b1tAction/paradiced/internal/engine/minigame"
	"github.com/b1tAction/paradiced/pkg/constants"
	"github.com/b1tAction/paradiced/pkg/id"
	pkgnet "github.com/b1tAction/paradiced/pkg/net"
	"github.com/b1tAction/paradiced/pkg/rng"
)

func TestStateMatchInit(t *testing.T) {
	state := NewMatchInitState()

	if state.ID() != StateMatchInit {
		t.Errorf("MatchInitState.ID() = %s, want StateMatchInit", state.ID().String())
	}

	game := engine.NewGame(id.NewGameID(), 0)
	ctx := NewStateContext().WithHSM(NewHSM(game))
	state.Enter(ctx)
	if ctx.Error != nil {
		t.Errorf("Enter should succeed, got error: %v", ctx.Error)
	}
	if !ctx.GetBoolOrDefault(KeyInitialized, false) {
		t.Error("Enter should set initialized metadata")
	}

	nextID := state.Update(ctx)
	if nextID != StateWaitingForHost {
		t.Errorf("Update should return StateWaitingForHost, got %s", nextID.String())
	}

	if !state.CanTransitionTo(StateRoundMiniGame) {
		t.Error("MatchInit should be able to transition to RoundMiniGame")
	}
}

// setupMiniGamePoolWithFrontendType temporarily replaces AllMiniGameTypes
// with a frontend type (DiceRace) so mini-game tests work without an online provider.
// Returns a cleanup function that restores the original pool.
func setupMiniGamePoolWithFrontendType() func() {
	origPool := constants.AllMiniGameTypes
	constants.AllMiniGameTypes = []constants.MiniGameType{constants.MiniGameTypeDiceRace}
	return func() { constants.AllMiniGameTypes = origPool }
}

func TestStateRoundMiniGame(t *testing.T) {
	origPool := setupMiniGamePoolWithFrontendType()
	defer origPool()

	state := NewRoundMiniGameState()

	if state.ID() != StateRoundMiniGame {
		t.Errorf("RoundMiniGameState.ID() = %s, want StateRoundMiniGame", state.ID().String())
	}

	game := engine.NewGame(id.NewGameID(), 0)
	game.AddPlayer(core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()}))
	game.AddPlayer(core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()}))
	game.AddPlayer(core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()}))

	ctx := NewStateContext().WithHSM(NewHSM(game))
	state.Enter(ctx)

	if state.totalPlayers != 3 {
		t.Errorf("totalPlayers should be 3, got %d", state.totalPlayers)
	}
	if !ctx.GetBoolOrDefault(KeyMiniGameStarted, false) {
		t.Error("mini_game_started should be true")
	}

	nextID := state.Update(ctx)
	if nextID != StateNone {
		t.Errorf("Update should return StateNone while waiting, got %s", nextID.String())
	}

	state.OnMiniGameResult(ctx, "p1", 1)
	state.OnMiniGameResult(ctx, "p2", 2)
	state.OnMiniGameResult(ctx, "p3", 3)

	if state.resultsReceived != 3 {
		t.Errorf("resultsReceived should be 3, got %d", state.resultsReceived)
	}

	nextID = state.Update(ctx)
	if nextID != StateRoundPrep {
		t.Errorf("Update should return StateRoundPrep after all results, got %s", nextID.String())
	}

	state.Exit(ctx)
	if ctx.GetBoolOrDefault(KeyMiniGameStarted, true) {
		t.Error("mini_game_started should be false after exit")
	}
}

func TestStateRoundMiniGame_ExitBroadcastsResult(t *testing.T) {
	origPool := setupMiniGamePoolWithFrontendType()
	defer origPool()

	state := NewRoundMiniGameState()

	game := engine.NewGame(id.NewGameID(), 0)
	p1 := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	p2 := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	game.AddPlayer(p1)
	game.AddPlayer(p2)

	ctx := NewStateContext().WithHSM(NewHSM(game))
	mockBroadcast := pkgnet.NewMockBroadcastAdapter()
	ctx.Broadcast = mockBroadcast

	state.Enter(ctx)
	state.OnMiniGameResult(ctx, p1.ID.UUID(), 2)
	state.OnMiniGameResult(ctx, p2.ID.UUID(), 1)
	state.Exit(ctx)

	if len(mockBroadcast.MiniGameResults) != 1 {
		t.Fatalf("MiniGameResults broadcast count = %d, want 1", len(mockBroadcast.MiniGameResults))
	}

	result := mockBroadcast.MiniGameResults[0]
	if result == nil {
		t.Fatal("MiniGameResult should not be nil")
	}
	if len(result.Rankings) != 2 {
		t.Fatalf("rankings length = %d, want 2", len(result.Rankings))
	}

	if result.Rankings[0].PlayerID != p2.ID.UUID() || result.Rankings[0].Rank != 1 {
		t.Errorf("rankings[0] = (%s, %d), want (%s, 1)", result.Rankings[0].PlayerID, result.Rankings[0].Rank, p2.ID.UUID())
	}
	if result.Rankings[1].PlayerID != p1.ID.UUID() || result.Rankings[1].Rank != 2 {
		t.Errorf("rankings[1] = (%s, %d), want (%s, 2)", result.Rankings[1].PlayerID, result.Rankings[1].Rank, p1.ID.UUID())
	}
}

func TestStateRoundPrep(t *testing.T) {
	state := NewRoundPrepState()

	if state.ID() != StateRoundPrep {
		t.Errorf("RoundPrepState.ID() = %s, want StateRoundPrep", state.ID().String())
	}

	game := engine.NewGame(id.NewGameID(), 0)
	p1 := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	p2 := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	p3 := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	p4 := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	game.AddPlayer(p1)
	game.AddPlayer(p2)
	game.AddPlayer(p3)
	game.AddPlayer(p4)

	ctx := NewStateContext().WithHSM(NewHSM(game))
	ctx.SetMiniGameRank(p1.ID.UUID(), 1)
	ctx.SetMiniGameRank(p2.ID.UUID(), 2)
	ctx.SetMiniGameRank(p3.ID.UUID(), 3)
	ctx.SetMiniGameRank(p4.ID.UUID(), 4)

	state.Enter(ctx)

	if rng.RankToDiceType(1) != rng.DiceTypeGold {
		t.Error("Rank 1 should get gold dice")
	}
	if rng.RankToDiceType(2) != rng.DiceTypeSilver {
		t.Error("Rank 2 should get silver dice")
	}
	if rng.RankToDiceType(3) != rng.DiceTypeCopper {
		t.Error("Rank 3 should get copper dice")
	}
	if rng.RankToDiceType(4) != rng.DiceTypeWood {
		t.Error("Rank 4 should get wood dice")
	}

	if ctx.GetRound() != 1 {
		t.Errorf("Round should be 1 (first round, no increment in RoundPrep anymore), got %d", ctx.GetRound())
	}

	// Verify dice types set correctly
	if ctx.GetDiceType(p1.ID.UUID()) != rng.DiceTypeGold {
		t.Error("p1 should have gold dice")
	}
	if ctx.GetDiceType(p2.ID.UUID()) != rng.DiceTypeSilver {
		t.Error("p2 should have silver dice")
	}

	nextID := state.Update(ctx)
	if nextID != StateTurnLoop {
		t.Errorf("Update should return StateTurnLoop, got %s", nextID.String())
	}
}

func TestStateTurnLoop(t *testing.T) {
	state := NewTurnLoopState()

	if state.ID() != StateTurnLoop {
		t.Errorf("TurnLoopState.ID() = %s, want StateTurnLoop", state.ID().String())
	}

	game := engine.NewGame(id.NewGameID(), 0)
	game.AddPlayer(core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()}))
	game.AddPlayer(core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()}))

	hsmInst := NewHSM(game)
	ctx := NewStateContext().WithHSM(hsmInst)

	state.Enter(ctx)

	if !ctx.GetBoolOrDefault(KeyTurnLoopActive, false) {
		t.Error("turn_loop_active should be true")
	}
	if state.currentPlayerIndex != 0 {
		t.Errorf("currentPlayerIndex should be 0, got %d", state.currentPlayerIndex)
	}
	if !state.pendingTurnStart {
		t.Error("pendingTurnStart should be true after Enter")
	}

	// First Update should auto-start first player turn
	nextID := state.Update(ctx)
	if nextID != StateTurnUpkeep {
		t.Errorf("First Update should return TurnUpkeep, got %s", nextID.String())
	}
	if state.pendingTurnStart {
		t.Error("pendingTurnStart should be false after Update")
	}

	// After first turn started, Update should return StateNone
	nextID = state.Update(ctx)
	if nextID != StateNone {
		t.Errorf("Update should return StateNone after first turn started, got %s", nextID.String())
	}

	state.OnTurnComplete(ctx)
	if state.turnsCompleted != 1 {
		t.Errorf("turnsCompleted should be 1, got %d", state.turnsCompleted)
	}
	if state.currentPlayerIndex != 1 {
		t.Errorf("currentPlayerIndex should be 1, got %d", state.currentPlayerIndex)
	}

	// Test reached end marker via Metadata (now uses BossDefeated in RoundData)
	winnerPlayer := game.Players[0]
	game.RoundData.SetBool(KeyBossDefeated, true)
	game.RoundData.SetString(KeyBossDefeatedBy, winnerPlayer.ID.UUID())
	state.OnTurnComplete(ctx)

	nextID = state.Update(ctx)
	if nextID != StateGameOver {
		t.Errorf("Update should return GameOver when Boss defeated, got %s", nextID.String())
	}

	if !state.CanTransitionTo(StateGameOver) {
		t.Error("TurnLoop should transition to GameOver")
	}
	if !state.CanTransitionTo(StateRoundEndWait) {
		t.Error("TurnLoop should transition to RoundEndWait")
	}
	if !state.CanTransitionTo(StateTurnUpkeep) {
		t.Error("TurnLoop should transition to TurnUpkeep")
	}

	state.Exit(NewStateContext())
	if state.turnsCompleted != 0 {
		t.Error("turnsCompleted should be reset")
	}
}

func TestStateGameOver(t *testing.T) {
	state := NewGameOverState()

	if state.ID() != StateGameOver {
		t.Errorf("GameOverState.ID() = %s, want StateGameOver", state.ID().String())
	}

	game := engine.NewGame(id.NewGameID(), 0)
	winnerPlayer := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	game.AddPlayer(winnerPlayer)

	ctx := NewStateContext().WithHSM(NewHSM(game))
	ctx.SetString(KeyWinner, winnerPlayer.ID.UUID())

	state.Enter(ctx)

	if state.winner == nil {
		t.Error("winner should be set")
	}
	if ctx.Error != nil {
		t.Errorf("Enter should succeed, got error: %v", ctx.Error)
	}
	if !ctx.GetBoolOrDefault(KeyGameOver, false) {
		t.Error("game_over should be true")
	}

	nextID := state.Update(ctx)
	if nextID != StateNone {
		t.Errorf("Update should return StateNone (terminal), got %s", nextID.String())
	}

	if state.CanTransitionTo(StateMatchInit) {
		t.Error("GameOver should not be able to transition (terminal state)")
	}
}

func TestGlobalStateFactory(t *testing.T) {
	factory := &GlobalStateFactory{}

	globalIDs := []StateID{
		StateMatchInit, StateRoundMiniGame, StateRoundPrep,
		StateTurnLoop, StateGameOver,
	}

	for _, id := range globalIDs {
		state := factory.CreateState(id)
		if state == nil {
			t.Errorf("Factory should create state for %s", id.String())
		}
		if state.ID() != id {
			t.Errorf("Created state ID = %s, want %s", state.ID().String(), id.String())
		}
	}

	state := factory.CreateState(StateTurnUpkeep)
	if state != nil {
		t.Error("Factory should return nil for non-global state")
	}
}

func TestRegisterGlobalStates(t *testing.T) {
	game := engine.NewGame(id.NewGameID(), 0)
	hsm := NewHSM(game)

	err := RegisterGlobalStates(hsm)
	if err != nil {
		t.Errorf("RegisterGlobalStates failed: %v", err)
	}

	for _, id := range []StateID{StateMatchInit, StateRoundMiniGame, StateRoundPrep, StateTurnLoop, StateGameOver} {
		if hsm.GetState(id) == nil {
			t.Errorf("State %s should be registered", id.String())
		}
	}
}

func TestDiceTypeFromRank(t *testing.T) {
	tests := []struct {
		rank     int
		expected rng.DiceType
	}{
		{1, rng.DiceTypeGold},
		{2, rng.DiceTypeSilver},
		{3, rng.DiceTypeCopper},
		{4, rng.DiceTypeWood},
		{5, rng.DiceTypeWood},
	}

	for _, tt := range tests {
		result := rng.RankToDiceType(tt.rank)
		if result != tt.expected {
			t.Errorf("RankToDiceType(%d) = %s, want %s", tt.rank, result.String(), tt.expected.String())
		}
	}
}

func TestHSMGlobalStateFlow(t *testing.T) {
	game := engine.NewGame(id.NewGameID(), 0)
	game.AddPlayer(core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()}))
	game.AddPlayer(core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()}))

	hsm := NewHSM(game)

	RegisterGlobalStates(hsm)

	ctx := NewStateContext().WithHSM(NewHSM(game))
	err := hsm.Start(StateMatchInit, ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Stop HSM with proper context
	hsm.Stop(NewStateContext().WithHSM(NewHSM(game)))
	if hsm.IsRunning() {
		t.Error("HSM should not be running after Stop")
	}
}

func TestTurnLoopAllPlayersComplete(t *testing.T) {
	state := NewTurnLoopState()
	game := engine.NewGame(id.NewGameID(), 0)
	game.AddPlayer(core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()}))
	game.AddPlayer(core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()}))

	ctx := NewStateContext().WithHSM(NewHSM(game))

	state.Enter(ctx)

	// Complete both players
	state.StartPlayerTurn(ctx)
	state.OnTurnComplete(ctx)

	state.StartPlayerTurn(ctx)
	state.OnTurnComplete(ctx)

	// After all players complete, wait for clients before next round
	nextID := state.StartPlayerTurn(ctx)
	if nextID != StateRoundEndWait {
		t.Errorf("After all players, should return RoundEndWait, got %s", nextID.String())
	}
}

// ========== RoundEndWaitState Tests ==========

func TestRoundEndWaitState_Enter(t *testing.T) {
	game := engine.NewGame(id.NewGameID(), 0)
	p1 := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	p2 := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	game.AddPlayer(p1)
	game.AddPlayer(p2)

	hsmInst := NewHSM(game)
	ctx := NewStateContext().WithHSM(hsmInst)

	state := NewRoundEndWaitState()
	state.Enter(ctx)

	if state.totalPlayers != 2 {
		t.Errorf("totalPlayers should be 2 (non-Boss), got %d", state.totalPlayers)
	}
	if state.readyReceived != 0 {
		t.Errorf("readyReceived should be 0 initially, got %d", state.readyReceived)
	}
	if !ctx.GetBoolOrDefault(KeyRoundEndWaiting, false) {
		t.Error("KeyRoundEndWaiting should be set after Enter")
	}
}

func TestRoundEndWaitState_Update_NoReady(t *testing.T) {
	game := engine.NewGame(id.NewGameID(), 0)
	p1 := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	game.AddPlayer(p1)

	hsmInst := NewHSM(game)
	ctx := NewStateContext().WithHSM(hsmInst)

	state := NewRoundEndWaitState()
	state.Enter(ctx)

	nextID := state.Update(ctx)
	if nextID != StateNone {
		t.Errorf("Update should return StateNone when no clients ready, got %s", nextID.String())
	}
}

func TestRoundEndWaitState_Update_AllReady(t *testing.T) {
	game := engine.NewGame(id.NewGameID(), 0)
	p1 := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	p2 := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	game.AddPlayer(p1)
	game.AddPlayer(p2)

	hsmInst := NewHSM(game)
	ctx := NewStateContext().WithHSM(hsmInst)

	state := NewRoundEndWaitState()
	state.Enter(ctx)

	// Signal all players ready
	state.OnRoundReady(ctx, p1.ID.UUID())
	state.OnRoundReady(ctx, p2.ID.UUID())

	nextID := state.Update(ctx)
	if nextID != StateRoundMiniGame {
		t.Errorf("Update should return StateRoundMiniGame when all ready, got %s", nextID.String())
	}
}

func TestRoundEndWaitState_Exit(t *testing.T) {
	game := engine.NewGame(id.NewGameID(), 0)
	p1 := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	game.AddPlayer(p1)

	hsmInst := NewHSM(game)
	ctx := NewStateContext().WithHSM(hsmInst)

	state := NewRoundEndWaitState()
	state.Enter(ctx)
	state.Exit(ctx)

	if ctx.GetBoolOrDefault(KeyRoundEndWaiting, false) {
		t.Error("KeyRoundEndWaiting should be cleared after Exit")
	}
}

func TestRoundEndWaitState_CanTransitionTo(t *testing.T) {
	state := NewRoundEndWaitState()

	if !state.CanTransitionTo(StateRoundMiniGame) {
		t.Error("RoundEndWait should transition to RoundMiniGame")
	}
	if !state.CanTransitionTo(StateGameOver) {
		t.Error("RoundEndWait should transition to GameOver")
	}
	if state.CanTransitionTo(StateTurnUpkeep) {
		t.Error("RoundEndWait should NOT transition to TurnUpkeep")
	}
}

// ========== RoundMiniGameState MiniGameDataSubmit Tests ==========

func TestRoundMiniGameState_GameTypeRandomSelection(t *testing.T) {
	origPool := setupMiniGamePoolWithFrontendType()
	defer origPool()

	state := NewRoundMiniGameState()

	game := engine.NewGame(id.NewGameID(), 42) // Fixed seed for deterministic test
	p1 := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	p2 := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	game.AddPlayer(p1)
	game.AddPlayer(p2)

	ctx := NewStateContext().WithHSM(NewHSM(game))
	mockBroadcast := pkgnet.NewMockBroadcastAdapter()
	ctx.Broadcast = mockBroadcast

	state.Enter(ctx)

	// game_type should be a valid type (not hardcoded "dice_race")
	gameType := state.GetGameType()
	if !gameType.IsValid() {
		t.Errorf("GetGameType() = %s, want valid MiniGameType", gameType)
	}

	// MiniGameStart broadcast should contain the selected game_type
	if len(mockBroadcast.MiniGameStarts) != 1 {
		t.Fatalf("MiniGameStarts count = %d, want 1", len(mockBroadcast.MiniGameStarts))
	}
	start := mockBroadcast.MiniGameStarts[0]
	if start.GameType != string(gameType) {
		t.Errorf("MiniGameStart.GameType = %s, want %s", start.GameType, string(gameType))
	}
}

func TestRoundMiniGameState_GameTypeDeterministic(t *testing.T) {
	// Same seed should produce same game_type
	game1 := engine.NewGame(id.NewGameID(), 12345)
	game2 := engine.NewGame(id.NewGameID(), 12345)

	type1 := minigame.SelectMiniGameType(game1.RNG)
	type2 := minigame.SelectMiniGameType(game2.RNG)

	if type1 != type2 {
		t.Errorf("Same seed produced different game types: %s vs %s", type1, type2)
	}
}

func TestRoundMiniGameState_OnMiniGameDataSubmit(t *testing.T) {
	origPool := setupMiniGamePoolWithFrontendType()
	defer origPool()

	state := NewRoundMiniGameState()

	// Use seed 42 which deterministically selects dice_race game type,
	// so "score" key sorting works as expected
	game := engine.NewGame(id.NewGameID(), 42)
	p1 := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	p2 := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	game.AddPlayer(p1)
	game.AddPlayer(p2)

	ctx := NewStateContext().WithHSM(NewHSM(game))
	state.Enter(ctx)

	gameType := state.GetGameType()

	// Submit data for p1 (first player)
	allReceived := state.OnMiniGameDataSubmit(ctx, p1.ID.UUID(), gameType, map[string]interface{}{
		"score": 200,
		"time":  2.0,
	})
	if allReceived {
		t.Error("OnMiniGameDataSubmit should return false when not all players submitted")
	}
	if state.resultsReceived != 1 {
		t.Errorf("resultsReceived = %d, want 1", state.resultsReceived)
	}

	// Submit data for p2 (second player) - should trigger rank calculation
	allReceived = state.OnMiniGameDataSubmit(ctx, p2.ID.UUID(), gameType, map[string]interface{}{
		"score": 100,
		"time":  5.0,
	})
	if !allReceived {
		t.Error("OnMiniGameDataSubmit should return true when all players submitted")
	}
	if state.resultsReceived != 2 {
		t.Errorf("resultsReceived = %d, want 2", state.resultsReceived)
	}

	// Verify ranks were calculated and stored in context
	rank1 := ctx.GetMiniGameRank(p1.ID.UUID())
	rank2 := ctx.GetMiniGameRank(p2.ID.UUID())
	if rank1 == 0 || rank2 == 0 {
		t.Errorf("Ranks should be non-zero: p1=%d, p2=%d", rank1, rank2)
	}
	// p1 has higher score (200 vs 100) or lower time (2.0 vs 5.0) → rank 1
	if rank1 != 1 {
		t.Errorf("p1 rank = %d, want 1 (better performance)", rank1)
	}
	if rank2 != 2 {
		t.Errorf("p2 rank = %d, want 2", rank2)
	}

	// State should transition to RoundPrep
	nextID := state.Update(ctx)
	if nextID != StateRoundPrep {
		t.Errorf("Update should return StateRoundPrep after all data, got %s", nextID.String())
	}
}

func TestRoundMiniGameState_OnMiniGameDataSubmit_GameTypeMismatch(t *testing.T) {
	origPool := setupMiniGamePoolWithFrontendType()
	defer origPool()

	state := NewRoundMiniGameState()

	game := engine.NewGame(id.NewGameID(), 0)
	p1 := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	p2 := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	game.AddPlayer(p1)
	game.AddPlayer(p2)

	ctx := NewStateContext().WithHSM(NewHSM(game))
	state.Enter(ctx)

	// Submit with wrong game_type - should return false
	wrongType := constants.MiniGameType("wrong_type")
	allReceived := state.OnMiniGameDataSubmit(ctx, p1.ID.UUID(), wrongType, map[string]interface{}{
		"score": 100,
	})
	if allReceived {
		t.Error("OnMiniGameDataSubmit should return false for mismatched game_type")
	}
	// resultsReceived should NOT increment on mismatch
	if state.resultsReceived != 0 {
		t.Errorf("resultsReceived = %d, want 0 (mismatched game_type should not count)", state.resultsReceived)
	}
}

func TestRoundMiniGameState_OnMiniGameDataSubmit_RPCMode(t *testing.T) {
	// Create a mock online provider that returns a connection
	mockProvider := &mockOnlineProvider{}
	state := NewRoundMiniGameState().WithProvider(mockProvider)

	game := engine.NewGame(id.NewGameID(), 0)
	p1 := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	game.AddPlayer(p1)

	// Force an online game type by setting gameType directly
	// (Enter() will select based on provider availability)
	state.gameType = constants.MiniGameTypeDilemmaRace
	state.mode = constants.MiniGameModeRPC

	ctx := NewStateContext().WithHSM(NewHSM(game))

	// In RPC mode, OnMiniGameDataSubmit should return false (not applicable)
	allReceived := state.OnMiniGameDataSubmit(ctx, p1.ID.UUID(), state.GetGameType(), map[string]interface{}{
		"score": 100,
	})
	if allReceived {
		t.Error("OnMiniGameDataSubmit should return false in RPC mode")
	}
}

func TestRoundMiniGameState_WithCustomRankCalculator(t *testing.T) {
	origPool := setupMiniGamePoolWithFrontendType()
	defer origPool()

	// Create a mock RankCalculator that reverses rankings
	mockCalc := &mockRankCalculator{}
	state := NewRoundMiniGameState().WithRankCalculator(mockCalc)

	game := engine.NewGame(id.NewGameID(), 0)
	p1 := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	p2 := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	game.AddPlayer(p1)
	game.AddPlayer(p2)

	ctx := NewStateContext().WithHSM(NewHSM(game))
	state.Enter(ctx)

	// Submit data for both players
	state.OnMiniGameDataSubmit(ctx, p1.ID.UUID(), state.GetGameType(), map[string]interface{}{"score": 100})
	state.OnMiniGameDataSubmit(ctx, p2.ID.UUID(), state.GetGameType(), map[string]interface{}{"score": 200})

	// Mock calculator should have been called and ranks assigned
	// The mock reverses rankings by sorting playerIDs alphabetically and assigning
	// rank N to the first and rank 1 to the last. Since UUIDs are random,
	// we just verify that both players got different ranks (not both 1).
	rank1 := ctx.GetMiniGameRank(p1.ID.UUID())
	rank2 := ctx.GetMiniGameRank(p2.ID.UUID())

	if rank1 == 0 || rank2 == 0 {
		t.Errorf("ranks should be assigned: p1=%d, p2=%d", rank1, rank2)
	}
	if rank1 == rank2 {
		t.Errorf("mock should produce different ranks: p1=%d, p2=%d", rank1, rank2)
	}
	// Verify mock reverses: one player should have rank 1, the other rank 2
	if rank1+rank2 != 3 { // 1+2=3
		t.Errorf("ranks should be 1 and 2 (sum=3): p1=%d, p2=%d (sum=%d)", rank1, rank2, rank1+rank2)
	}
}

// mockRankCalculator reverses rankings for testing (deterministic: sorts playerIDs for stable order)
type mockRankCalculator struct{}

func (m *mockRankCalculator) Calculate(gameType constants.MiniGameType, submissions map[string]map[string]interface{}) map[string]int {
	ranks := make(map[string]int, len(submissions))
	// Sort playerIDs to ensure deterministic ordering
	playerIDs := make([]string, 0, len(submissions))
	for playerID := range submissions {
		playerIDs = append(playerIDs, playerID)
	}
	sort.Strings(playerIDs)
	// Reverse order: last playerID gets rank 1, first gets rank N
	for i, playerID := range playerIDs {
		ranks[playerID] = len(playerIDs) - i
	}
	return ranks
}

// mockOnlineProvider implements protocol.OnlineMiniGameProvider for testing.
type mockOnlineProvider struct {
	createRoomErr  error
	destroyRoomErr error
}

func (m *mockOnlineProvider) CreateRoom(gameType constants.MiniGameType, playerIDs []string) (*pkgnet.MiniGameConn, error) {
	if m.createRoomErr != nil {
		return nil, m.createRoomErr
	}
	tokens := make(map[string]string)
	for _, pid := range playerIDs {
		tokens[pid] = "mock_token_" + pid
	}
	return &pkgnet.MiniGameConn{
		URL:          "ws://mock-colyseus:2567",
		RoomID:       "mock_room_id",
		PlayerTokens: tokens,
	}, nil
}

func (m *mockOnlineProvider) DestroyRoom(roomID string) error {
	return m.destroyRoomErr
}

func (m *mockOnlineProvider) GetTimeout(gameType constants.MiniGameType) time.Duration {
	return 60 * time.Second
}

func TestStateRoundMiniGame_ExitBroadcastsResultWithGameData(t *testing.T) {
	origPool := setupMiniGamePoolWithFrontendType()
	defer origPool()

	state := NewRoundMiniGameState()

	game := engine.NewGame(id.NewGameID(), 0)
	p1 := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	p2 := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	game.AddPlayer(p1)
	game.AddPlayer(p2)

	ctx := NewStateContext().WithHSM(NewHSM(game))
	mockBroadcast := pkgnet.NewMockBroadcastAdapter()
	ctx.Broadcast = mockBroadcast

	state.Enter(ctx)
	gameType := state.GetGameType()

	// Submit game_data for both players via frontend-driven mode
	p1Data := map[string]interface{}{
		"dice1": 3,
		"dice2": 5,
		"score": 8,
	}
	p2Data := map[string]interface{}{
		"dice1": 2,
		"dice2": 4,
		"score": 6,
	}
	state.OnMiniGameDataSubmit(ctx, p1.ID.UUID(), gameType, p1Data)
	state.OnMiniGameDataSubmit(ctx, p2.ID.UUID(), gameType, p2Data)

	state.Exit(ctx)

	if len(mockBroadcast.MiniGameResults) != 1 {
		t.Fatalf("MiniGameResults broadcast count = %d, want 1", len(mockBroadcast.MiniGameResults))
	}

	result := mockBroadcast.MiniGameResults[0]
	if result == nil {
		t.Fatal("MiniGameResult should not be nil")
	}
	if len(result.Rankings) != 2 {
		t.Fatalf("rankings length = %d, want 2", len(result.Rankings))
	}

	// Verify DisplayName is populated (currently = player UUID, TODO for actual name)
	for _, ranking := range result.Rankings {
		if ranking.DisplayName == "" {
			t.Errorf("RankingEntry.DisplayName should be non-empty for %s", ranking.PlayerID)
		}
		if ranking.PlayerID != ranking.DisplayName {
			t.Errorf("DisplayName should equal PlayerID (current TODO), got display_name=%s, player_id=%s", ranking.DisplayName, ranking.PlayerID)
		}
	}

	// Verify GameData is carried through for each player
	for _, ranking := range result.Rankings {
		if ranking.GameData == nil {
			t.Errorf("RankingEntry.GameData should not be nil for player %s (submitted data)", ranking.PlayerID)
		}
	}

	// Find rankings for each specific player
	var p1Ranking, p2Ranking *pkgnet.RankingEntry
	for i := range result.Rankings {
		if result.Rankings[i].PlayerID == p1.ID.UUID() {
			p1Ranking = &result.Rankings[i]
		}
		if result.Rankings[i].PlayerID == p2.ID.UUID() {
			p2Ranking = &result.Rankings[i]
		}
	}

	if p1Ranking == nil || p2Ranking == nil {
		t.Fatal("rankings for p1 and p2 should both exist")
	}

	// Verify p1's game_data contains submitted values
	score1, ok := p1Ranking.GameData["score"]
	if !ok {
		t.Error("p1 GameData should contain 'score' key")
	}
	// gameData is stored as map[string]interface{} directly from Go (not JSON),
	// so integer values remain as int, not float64
	if score1 != 8 {
		t.Errorf("p1 GameData['score'] = %v, want 8", score1)
	}

	// Verify p2's game_data contains submitted values
	score2, ok := p2Ranking.GameData["score"]
	if !ok {
		t.Error("p2 GameData should contain 'score' key")
	}
	if score2 != 6 {
		t.Errorf("p2 GameData['score'] = %v, want 6", score2)
	}
}

func TestStateRoundMiniGame_ExitBroadcastsResultWithDisplayNameOnly(t *testing.T) {
	origPool := setupMiniGamePoolWithFrontendType()
	defer origPool()

	// RPC mode: OnMiniGameResult sets rank directly, no gameData stored
	state := NewRoundMiniGameState()

	game := engine.NewGame(id.NewGameID(), 0)
	p1 := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	p2 := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	game.AddPlayer(p1)
	game.AddPlayer(p2)

	ctx := NewStateContext().WithHSM(NewHSM(game))
	mockBroadcast := pkgnet.NewMockBroadcastAdapter()
	ctx.Broadcast = mockBroadcast

	state.Enter(ctx)

	// Use RPC mode: directly set rank (no game_data stored)
	state.OnMiniGameResult(ctx, p1.ID.UUID(), 1)
	state.OnMiniGameResult(ctx, p2.ID.UUID(), 2)

	state.Exit(ctx)

	if len(mockBroadcast.MiniGameResults) != 1 {
		t.Fatalf("MiniGameResults broadcast count = %d, want 1", len(mockBroadcast.MiniGameResults))
	}

	result := mockBroadcast.MiniGameResults[0]
	if len(result.Rankings) != 2 {
		t.Fatalf("rankings length = %d, want 2", len(result.Rankings))
	}

	// DisplayName should be populated even without gameData
	for _, ranking := range result.Rankings {
		if ranking.DisplayName == "" {
			t.Errorf("RankingEntry.DisplayName should be non-empty for %s", ranking.PlayerID)
		}
	}

	// GameData should be nil since no game_data was submitted (RPC mode)
	for _, ranking := range result.Rankings {
		if ranking.GameData != nil {
			t.Errorf("RankingEntry.GameData should be nil for player %s (RPC mode, no data submitted)", ranking.PlayerID)
		}
	}
}

// ========== GameOverState.Exit Tests ==========

func TestGameOverStateExit(t *testing.T) {
	state := NewGameOverState()
	state.winner = core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})

	state.Exit(nil)

	if state.winner != nil {
		t.Error("GameOverState.Exit should clear winner")
	}
}

// ========== GameOverState.Enter Stops HSM Tests ==========

func TestGameOverStateEnterStopsHSM(t *testing.T) {
	game := engine.NewGame(id.NewGameID(), 0)
	winnerPlayer := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	game.AddPlayer(winnerPlayer)

	hsmInst := NewHSM(game)
	RegisterGlobalStates(hsmInst)

	// Start HSM so it's running
	ctx := NewStateContext().WithHSM(hsmInst)
	hsmInst.Start(StateMatchInit, ctx)

	if !hsmInst.IsRunning() {
		t.Fatal("HSM should be running before GameOver")
	}

	// Transition to GameOver
	gameOverCtx := NewStateContext().WithHSM(hsmInst)
	gameOverCtx.SetString(KeyWinner, winnerPlayer.ID.UUID())
	hsmInst.TransitionTo(StateGameOver, gameOverCtx)

	// After GameOverState.Enter(), HSM should be stopped
	if hsmInst.IsRunning() {
		t.Error("HSM should not be running after GameOverState.Enter() stops it")
	}
}

// ========== RoundMiniGameState Skip Tests ==========

func TestRoundMiniGameState_SkipWhenNoAvailable(t *testing.T) {
	// When all available types are online and no provider exists,
	// MiniGameTypeNone should be returned and the mini-game phase should skip.
	origPool := constants.AllMiniGameTypes
	constants.AllMiniGameTypes = []constants.MiniGameType{constants.MiniGameTypeDilemmaRace}
	defer func() { constants.AllMiniGameTypes = origPool }()

	state := NewRoundMiniGameState()

	game := engine.NewGame(id.NewGameID(), 0)
	p1 := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	p2 := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	game.AddPlayer(p1)
	game.AddPlayer(p2)

	ctx := NewStateContext().WithHSM(NewHSM(game))
	mockBroadcast := pkgnet.NewMockBroadcastAdapter()
	ctx.Broadcast = mockBroadcast

	state.Enter(ctx)

	// gameType should be MiniGameTypeNone (no provider, all pool types are online)
	if state.gameType != constants.MiniGameTypeNone {
		t.Errorf("gameType = %s, want MiniGameTypeNone", state.gameType)
	}

	// mini_game_started should be false (skipped)
	if ctx.GetBoolOrDefault(KeyMiniGameStarted, false) {
		t.Error("mini_game_started should be false when no mini-game available")
	}

	// No MiniGameStart broadcast should have been sent
	if len(mockBroadcast.MiniGameStarts) != 0 {
		t.Errorf("MiniGameStarts count = %d, want 0 (skipped)", len(mockBroadcast.MiniGameStarts))
	}

	// Update should transition to RoundPrep immediately
	nextID := state.Update(ctx)
	if nextID != StateRoundPrep {
		t.Errorf("Update should return StateRoundPrep when no mini-game available, got %s", nextID.String())
	}
}

func TestRoundMiniGameState_OnlineWithProvider(t *testing.T) {
	// When provider is available and online type is in pool, mini-game should proceed normally.
	origPool := constants.AllMiniGameTypes
	constants.AllMiniGameTypes = []constants.MiniGameType{constants.MiniGameTypeDilemmaRace}
	defer func() { constants.AllMiniGameTypes = origPool }()

	mockProvider := &mockOnlineProvider{}
	state := NewRoundMiniGameState().WithProvider(mockProvider)

	game := engine.NewGame(id.NewGameID(), 0)
	p1 := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	p2 := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	game.AddPlayer(p1)
	game.AddPlayer(p2)

	ctx := NewStateContext().WithHSM(NewHSM(game))
	mockBroadcast := pkgnet.NewMockBroadcastAdapter()
	ctx.Broadcast = mockBroadcast

	state.Enter(ctx)

	// gameType should be dilemma_race (provider available, online type eligible)
	if state.gameType != constants.MiniGameTypeDilemmaRace {
		t.Errorf("gameType = %s, want dilemma_race", state.gameType)
	}

	// mini_game_started should be true
	if !ctx.GetBoolOrDefault(KeyMiniGameStarted, false) {
		t.Error("mini_game_started should be true when mini-game is available")
	}

	// MiniGameStart broadcast should have been sent
	if len(mockBroadcast.MiniGameStarts) != 1 {
		t.Errorf("MiniGameStarts count = %d, want 1", len(mockBroadcast.MiniGameStarts))
	}
}
