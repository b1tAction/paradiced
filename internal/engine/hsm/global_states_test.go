package hsm

import (
	"testing"

	"github.com/b1tAction/Fated/internal/core"
	"github.com/b1tAction/Fated/internal/engine"
)

func TestStateMatchInit(t *testing.T) {
	state := NewMatchInitState()

	if state.ID() != StateMatchInit {
		t.Errorf("MatchInitState.ID() = %s, want StateMatchInit", state.ID().String())
	}

	game := engine.NewGame("game-1", 0)
	ctx := NewStateContext().WithGame(game)
	state.Enter(ctx)
	if !ctx.Success {
		t.Error("Enter should set Success = true")
	}
	if !ctx.GetBoolOrDefault("initialized", false) {
		t.Error("Enter should set initialized metadata")
	}

	nextID := state.Update(ctx)
	if nextID != StateRoundMiniGame {
		t.Errorf("Update should return StateRoundMiniGame, got %s", nextID.String())
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

	game := engine.NewGame("game-1", 0)
	game.AddPlayer(core.NewPlayer(core.PlayerConfig{UserID: "p1"}))
	game.AddPlayer(core.NewPlayer(core.PlayerConfig{UserID: "p2"}))
	game.AddPlayer(core.NewPlayer(core.PlayerConfig{UserID: "p3"}))

	ctx := NewStateContext().WithGame(game)
	state.Enter(ctx)

	if state.totalPlayers != 3 {
		t.Errorf("totalPlayers should be 3, got %d", state.totalPlayers)
	}
	if !ctx.GetBoolOrDefault("mini_game_started", false) {
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
	if ctx.GetBoolOrDefault("mini_game_started", true) {
		t.Error("mini_game_started should be false after exit")
	}
}

func TestStateRoundPrep(t *testing.T) {
	state := NewRoundPrepState()

	if state.ID() != StateRoundPrep {
		t.Errorf("RoundPrepState.ID() = %s, want StateRoundPrep", state.ID().String())
	}

	game := engine.NewGame("game-1", 0)
	game.AddPlayer(core.NewPlayer(core.PlayerConfig{UserID: "p1"}))
	game.AddPlayer(core.NewPlayer(core.PlayerConfig{UserID: "p2"}))
	game.AddPlayer(core.NewPlayer(core.PlayerConfig{UserID: "p3"}))
	game.AddPlayer(core.NewPlayer(core.PlayerConfig{UserID: "p4"}))

	ctx := NewStateContext().WithGame(game)
	ctx.SetMiniGameRank("p1", 1)
	ctx.SetMiniGameRank("p2", 2)
	ctx.SetMiniGameRank("p3", 3)
	ctx.SetMiniGameRank("p4", 4)

	state.Enter(ctx)

	if getDiceType(1) != "gold" {
		t.Error("Rank 1 should get gold dice")
	}
	if getDiceType(2) != "silver" {
		t.Error("Rank 2 should get silver dice")
	}
	if getDiceType(3) != "copper" {
		t.Error("Rank 3 should get copper dice")
	}
	if getDiceType(4) != "wood" {
		t.Error("Rank 4 should get wood dice")
	}

	if game.State.Round != 2 {
		t.Errorf("Round should be incremented to 2 (started at 1), got %d", game.State.Round)
	}

	// Verify dice types set correctly
	if ctx.GetDiceType("p1") != "gold" {
		t.Error("p1 should have gold dice")
	}
	if ctx.GetDiceType("p2") != "silver" {
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

	game := engine.NewGame("game-1", 0)
	game.AddPlayer(core.NewPlayer(core.PlayerConfig{UserID: "p1"}))
	game.AddPlayer(core.NewPlayer(core.PlayerConfig{UserID: "p2"}))

	ctx := NewStateContext().WithGame(game)

	state.Enter(ctx)

	if !ctx.GetBoolOrDefault("turn_loop_active", false) {
		t.Error("turn_loop_active should be true")
	}
	if state.currentPlayerIndex != 0 {
		t.Errorf("currentPlayerIndex should be 0, got %d", state.currentPlayerIndex)
	}

	nextID := state.Update(ctx)
	if nextID != StateNone {
		t.Errorf("Update should return StateNone while in loop, got %s", nextID.String())
	}

	// Start first player turn
	nextID = state.StartPlayerTurn(ctx)
	if nextID != StateTurnUpkeep {
		t.Errorf("StartPlayerTurn should return TurnUpkeep, got %s", nextID.String())
	}

	state.OnTurnComplete(ctx)
	if state.turnsCompleted != 1 {
		t.Errorf("turnsCompleted should be 1, got %d", state.turnsCompleted)
	}
	if state.currentPlayerIndex != 1 {
		t.Errorf("currentPlayerIndex should be 1, got %d", state.currentPlayerIndex)
	}

	// Test reached end marker via Metadata
	ctx.SetReachedEnd(true)
	state.OnTurnComplete(ctx)
	if !state.reachedEnd {
		t.Error("reachedEnd should be true when ctx.HasReachedEnd()")
	}

	nextID = state.Update(ctx)
	if nextID != StateBossBattle {
		t.Errorf("Update should return BossBattle when reached, got %s", nextID.String())
	}

	if !state.CanTransitionTo(StateBossBattle) {
		t.Error("TurnLoop should transition to BossBattle")
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

func TestStateBossBattle(t *testing.T) {
	state := NewBossBattleState()

	if state.ID() != StateBossBattle {
		t.Errorf("BossBattleState.ID() = %s, want StateBossBattle", state.ID().String())
	}

	game := engine.NewGame("game-1", 0)
	game.AddPlayer(core.NewPlayer(core.PlayerConfig{UserID: "p1"}))

	ctx := NewStateContext().WithGame(game)
	ctx.SetString(KeyBossTrigger, "p1")

	state.Enter(ctx)

	if state.triggerPlayer == nil {
		t.Error("triggerPlayer should be set")
	}
	if !ctx.GetBoolOrDefault("boss_battle_active", false) {
		t.Error("boss_battle_active should be true")
	}

	nextID := state.Update(ctx)
	if nextID != StateNone {
		t.Errorf("Update should return StateNone while in battle, got %s", nextID.String())
	}

	state.OnBossDefeated()
	if !state.bossDefeated {
		t.Error("bossDefeated should be true")
	}

	nextID = state.Update(ctx)
	if nextID != StateGameOver {
		t.Errorf("Update should return GameOver when defeated, got %s", nextID.String())
	}

	state.Exit(NewStateContext())
	if state.triggerPlayer != nil {
		t.Error("triggerPlayer should be nil after exit")
	}
}

func TestStateGameOver(t *testing.T) {
	state := NewGameOverState()

	if state.ID() != StateGameOver {
		t.Errorf("GameOverState.ID() = %s, want StateGameOver", state.ID().String())
	}

	game := engine.NewGame("game-1", 0)
	game.AddPlayer(core.NewPlayer(core.PlayerConfig{UserID: "winner"}))

	ctx := NewStateContext().WithGame(game)
	ctx.SetString(KeyWinner, "winner")

	state.Enter(ctx)

	if state.winner == nil {
		t.Error("winner should be set")
	}
	if !ctx.Success {
		t.Error("Success should be true")
	}
	if !ctx.GetBoolOrDefault("game_over", false) {
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
		StateTurnLoop, StateBossBattle, StateGameOver,
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
	game := engine.NewGame("game-1", 0)
	hsm := NewHSM(game)

	err := RegisterGlobalStates(hsm)
	if err != nil {
		t.Errorf("RegisterGlobalStates failed: %v", err)
	}

	for _, id := range []StateID{StateMatchInit, StateRoundMiniGame, StateRoundPrep, StateTurnLoop, StateBossBattle, StateGameOver} {
		if hsm.GetState(id) == nil {
			t.Errorf("State %s should be registered", id.String())
		}
	}
}

func TestGetDiceType(t *testing.T) {
	tests := []struct {
		rank     int
		expected string
	}{
		{1, "gold"},
		{2, "silver"},
		{3, "copper"},
		{4, "wood"},
		{5, "wood"},
	}

	for _, tt := range tests {
		result := getDiceType(tt.rank)
		if result != tt.expected {
			t.Errorf("getDiceType(%d) = %s, want %s", tt.rank, result, tt.expected)
		}
	}
}

func TestHSMGlobalStateFlow(t *testing.T) {
	game := engine.NewGame("game-1", 0)
	game.AddPlayer(core.NewPlayer(core.PlayerConfig{UserID: "p1"}))
	game.AddPlayer(core.NewPlayer(core.PlayerConfig{UserID: "p2"}))

	hsm := NewHSM(game)

	RegisterGlobalStates(hsm)

	ctx := NewStateContext().WithGame(game)
	err := hsm.Start(StateMatchInit, ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Stop HSM with proper context
	hsm.Stop(NewStateContext().WithGame(game))
	if hsm.IsRunning() {
		t.Error("HSM should not be running after Stop")
	}
}

func TestTurnLoopAllPlayersComplete(t *testing.T) {
	state := NewTurnLoopState()
	game := engine.NewGame("game-1", 0)
	game.AddPlayer(core.NewPlayer(core.PlayerConfig{UserID: "p1"}))
	game.AddPlayer(core.NewPlayer(core.PlayerConfig{UserID: "p2"}))

	ctx := NewStateContext().WithGame(game)

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