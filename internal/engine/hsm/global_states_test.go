package hsm

import (
	"testing"

	"github.com/b1tAction/paradiced/internal/core"
	"github.com/b1tAction/paradiced/internal/engine"
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

func TestStateRoundMiniGame(t *testing.T) {
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
	if !state.CanTransitionTo(StateRoundMiniGame) {
		t.Error("TurnLoop should transition to RoundMiniGame")
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

	// After all players complete, next round
	nextID := state.StartPlayerTurn(ctx)
	if nextID != StateRoundMiniGame {
		t.Errorf("After all players, should return RoundMiniGame, got %s", nextID.String())
	}
}
