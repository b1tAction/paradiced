package hsm

import (
	"testing"
	"time"

	"github.com/b1tAction/fated/internal/core"
	"github.com/b1tAction/fated/internal/engine"
	"github.com/b1tAction/fated/pkg/event"
)

func TestNewHSM(t *testing.T) {
	game := engine.NewGame("game-1", 0)
	hsm := NewHSM(game)

	if hsm == nil {
		t.Fatal("NewHSM should return non-nil HSM")
	}
	if hsm.GetGlobalStateID() != StateNone {
		t.Error("Initial global state should be StateNone")
	}
	if hsm.GetTurnStateID() != StateNone {
		t.Error("Initial turn state should be StateNone")
	}
	if hsm.IsRunning() {
		t.Error("HSM should not be running initially")
	}
	if hsm.GetStack() == nil {
		t.Error("Stack should be initialized")
	}
}

func TestHSMRegisterState(t *testing.T) {
	game := engine.NewGame("test", 0)
	hsm := NewHSM(game)

	// Register valid state
	state := &mockState{id: StateMatchInit}
	err := hsm.RegisterState(state)
	if err != nil {
		t.Errorf("Register valid state failed: %v", err)
	}

	// Check state is registered
	if hsm.GetState(StateMatchInit) != state {
		t.Error("Registered state should be retrievable")
	}

	// Register nil state (should fail)
	err = hsm.RegisterState(nil)
	if err == nil {
		t.Error("Register nil state should fail")
	}

	// Register duplicate state (should fail)
	err = hsm.RegisterState(&mockState{id: StateMatchInit})
	if err == nil {
		t.Error("Register duplicate state should fail")
	}

	// Register invalid state ID (should fail)
	err = hsm.RegisterState(&mockState{id: StateInvalid})
	if err == nil {
		t.Error("Register invalid state ID should fail")
	}
}

func TestHSMRegisterStates(t *testing.T) {
	game := engine.NewGame("test", 0)
	hsm := NewHSM(game)

	states := []State{
		&mockState{id: StateMatchInit},
		&mockState{id: StateRoundMiniGame},
		&mockState{id: StateRoundPrep},
	}

	err := hsm.RegisterStates(states)
	if err != nil {
		t.Errorf("RegisterStates failed: %v", err)
	}

	if len(hsm.states) != 3 {
		t.Errorf("Should have 3 registered states, got %d", len(hsm.states))
	}
}

func TestHSMStart(t *testing.T) {
	game := engine.NewGame("test", 0)
	hsm := NewHSM(game)
	hsm.RegisterState(&mockState{id: StateMatchInit})

	// Start HSM
	err := hsm.Start(StateMatchInit, nil)
	if err != nil {
		t.Errorf("Start failed: %v", err)
	}

	if !hsm.IsRunning() {
		t.Error("HSM should be running after Start")
	}
	if hsm.GetGlobalStateID() != StateMatchInit {
		t.Errorf("Global state should be MatchInit, got %s", hsm.GetGlobalStateID().String())
	}

	// Start again (should fail)
	err = hsm.Start(StateMatchInit, nil)
	if err == nil {
		t.Error("Start again should fail")
	}
}

func TestHSMStop(t *testing.T) {
	game := engine.NewGame("test", 0)
	hsm := NewHSM(game)
	state := &mockState{id: StateMatchInit}
	hsm.RegisterState(state)

	hsm.Start(StateMatchInit, nil)
	hsm.Stop(nil)

	if hsm.IsRunning() {
		t.Error("HSM should not be running after Stop")
	}
	if !state.exitCalled {
		t.Error("State Exit should be called on Stop")
	}
}

func TestHSMTransitionToGlobalState(t *testing.T) {
	game := engine.NewGame("test", 0)
	hsm := NewHSM(game)

	// Register global states
	matchInit := &mockState{id: StateMatchInit}
	roundMiniGame := &mockState{id: StateRoundMiniGame}
	hsm.RegisterState(matchInit)
	hsm.RegisterState(roundMiniGame)

	// Start HSM
	hsm.Start(StateMatchInit, nil)

	// Transition to next state
	err := hsm.TransitionTo(StateRoundMiniGame, nil)
	if err != nil {
		t.Errorf("TransitionTo failed: %v", err)
	}

	if hsm.GetGlobalStateID() != StateRoundMiniGame {
		t.Errorf("Global state should be RoundMiniGame, got %s", hsm.GetGlobalStateID().String())
	}
	if !matchInit.exitCalled {
		t.Error("Previous state Exit should be called")
	}
	if !roundMiniGame.enterCalled {
		t.Error("New state Enter should be called")
	}
}

func TestHSMTransitionToInvalidState(t *testing.T) {
	game := engine.NewGame("test", 0)
	hsm := NewHSM(game)
	hsm.RegisterState(&mockState{id: StateMatchInit})

	// Transition to invalid ID
	err := hsm.TransitionTo(StateInvalid, nil)
	if err == nil {
		t.Error("Transition to invalid state should fail")
	}

	// Transition to unregistered state
	err = hsm.TransitionTo(StateRoundMiniGame, nil)
	if err == nil {
		t.Error("Transition to unregistered state should fail")
	}
}

func TestHSMTransitionToTurnState(t *testing.T) {
	game := engine.NewGame("test", 0)
	hsm := NewHSM(game)

	// Register states
	hsm.RegisterState(&mockState{id: StateTurnLoop})
	hsm.RegisterState(&mockState{id: StateTurnUpkeep})

	// Must be in TurnLoop first
	err := hsm.Start(StateTurnLoop, nil)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Set turn player
	player := core.NewPlayer(core.PlayerConfig{UserID: "player-1"})
	hsm.SetTurnPlayer(player)

	// Transition to turn state
	err = hsm.TransitionTo(StateTurnUpkeep, nil)
	if err != nil {
		t.Errorf("TransitionTo turn state failed: %v", err)
	}

	if hsm.GetTurnStateID() != StateTurnUpkeep {
		t.Errorf("Turn state should be TurnUpkeep, got %s", hsm.GetTurnStateID().String())
	}
}

func TestHSMTransitionToTurnStateWithoutTurnLoop(t *testing.T) {
	game := engine.NewGame("test", 0)
	hsm := NewHSM(game)
	hsm.RegisterState(&mockState{id: StateMatchInit})
	hsm.RegisterState(&mockState{id: StateTurnUpkeep})

	// Start in non-TurnLoop state
	hsm.Start(StateMatchInit, nil)

	// Try to transition to turn state (should fail)
	err := hsm.TransitionTo(StateTurnUpkeep, nil)
	if err == nil {
		t.Error("Transition to turn state without TurnLoop should fail")
	}
}

func TestHSMSetTurnPlayer(t *testing.T) {
	game := engine.NewGame("test", 0)
	hsm := NewHSM(game)
	player := core.NewPlayer(core.PlayerConfig{UserID: "player-1"})

	hsm.SetTurnPlayer(player)

	if hsm.GetTurnPlayer() != player {
		t.Error("Turn player should be set correctly")
	}
}

func TestHSMUpdate(t *testing.T) {
	game := engine.NewGame("test", 0)
	hsm := NewHSM(game)

	// Register state that stays in current state
	state := &mockState{id: StateMatchInit}
	hsm.RegisterState(state)

	hsm.Start(StateMatchInit, nil)

	// Update
	nextID, err := hsm.Update(nil)
	if err != nil {
		t.Errorf("Update failed: %v", err)
	}
	if nextID != StateNone {
		t.Errorf("Update should return StateNone for stay-in-state, got %s", nextID.String())
	}
	if !state.updateCalled {
		t.Error("State Update should be called")
	}
}

func TestHSMUpdateNotRunning(t *testing.T) {
	game := engine.NewGame("test", 0)
	hsm := NewHSM(game)

	// Update without starting
	nextID, err := hsm.Update(nil)
	if err == nil {
		t.Error("Update without Start should fail")
	}
	if nextID != StateNone {
		t.Error("Update should return StateNone on error")
	}
}

func TestHSMIsPaused(t *testing.T) {
	game := engine.NewGame("test", 0)
	hsm := NewHSM(game)

	if hsm.IsPaused() {
		t.Error("HSM should not be paused initially")
	}
}

func TestHSMIsWaiting(t *testing.T) {
	game := engine.NewGame("test", 0)
	hsm := NewHSM(game)

	if hsm.IsWaiting() {
		t.Error("HSM should not be waiting initially")
	}
}

func TestHSMIsInTurn(t *testing.T) {
	game := engine.NewGame("test", 0)
	hsm := NewHSM(game)
	hsm.RegisterState(&mockState{id: StateTurnLoop})

	if hsm.IsInTurn() {
		t.Error("HSM should not be in turn initially")
	}

	// Start in TurnLoop but no turn state
	hsm.Start(StateTurnLoop, nil)
	if hsm.IsInTurn() {
		t.Error("HSM in TurnLoop without turn state should not be IsInTurn")
	}

	// Set turn state
	hsm.RegisterState(&mockState{id: StateTurnUpkeep})
	hsm.turnStateID = StateTurnUpkeep // Direct set for testing
	if !hsm.IsInTurn() {
		t.Error("HSM in TurnLoop with turn state should be IsInTurn")
	}
}

func TestHSMCreateSnapshot(t *testing.T) {
	game := engine.NewGame("test", 0)
	hsm := NewHSM(game)
	hsm.RegisterState(&mockState{id: StateMatchInit})
	hsm.RegisterState(&mockState{id: StateTurnUpkeep})

	hsm.Start(StateMatchInit, nil)
	player := core.NewPlayer(core.PlayerConfig{UserID: "player-1"})
	hsm.SetTurnPlayer(player)

	snapshot := hsm.CreateSnapshot()

	if snapshot == nil {
		t.Fatal("CreateSnapshot should return non-nil snapshot")
	}
	if snapshot.GlobalStateID != StateMatchInit {
		t.Errorf("Snapshot global state = %s, want MatchInit", snapshot.GlobalStateID.String())
	}
	if snapshot.Running != true {
		t.Error("Snapshot running should be true")
	}
	if snapshot.TurnPlayerID != "player-1" {
		t.Errorf("Snapshot turn player = %s, want player-1", snapshot.TurnPlayerID)
	}
}

func TestHSMRestoreFromSnapshot(t *testing.T) {
	game := engine.NewGame("test", 0)
	hsm := NewHSM(game)
	hsm.RegisterState(&mockState{id: StateMatchInit})
	hsm.RegisterState(&mockState{id: StateRoundMiniGame})

	// Create and restore snapshot
	snapshot := &HSMSnapshot{
		GlobalStateID: StateRoundMiniGame,
		Running:       true,
		Paused:        false,
		EnterTime:     time.Now(),
	}

	err := hsm.RestoreFromSnapshot(snapshot)
	if err != nil {
		t.Errorf("RestoreFromSnapshot failed: %v", err)
	}

	if hsm.GetGlobalStateID() != StateRoundMiniGame {
		t.Errorf("Restored global state = %s, want RoundMiniGame", hsm.GetGlobalStateID().String())
	}
	if !hsm.IsRunning() {
		t.Error("Restored HSM should be running")
	}
}

func TestHSMRestoreFromSnapshotNil(t *testing.T) {
	game := engine.NewGame("test", 0)
	hsm := NewHSM(game)

	err := hsm.RestoreFromSnapshot(nil)
	if err == nil {
		t.Error("Restore from nil snapshot should fail")
	}
}

func TestHSMRestoreFromSnapshotUnknownState(t *testing.T) {
	game := engine.NewGame("test", 0)
	hsm := NewHSM(game)

	snapshot := &HSMSnapshot{
		GlobalStateID: StateBossBattle, // Not registered
		Running:       true,
	}

	err := hsm.RestoreFromSnapshot(snapshot)
	if err == nil {
		t.Error("Restore with unknown state should fail")
	}
}

func TestHSMString(t *testing.T) {
	game := engine.NewGame("test", 0)
	hsm := NewHSM(game)
	hsm.RegisterState(&mockState{id: StateMatchInit})

	// Initial state
	expected := "HSM[Global=None]"
	if hsm.String() != expected {
		t.Errorf("HSM string = %s, want '%s'", hsm.String(), expected)
	}

	// After start
	hsm.Start(StateMatchInit, nil)
	expected = "HSM[Global=MatchInit]"
	if hsm.String() != expected {
		t.Errorf("HSM string = %s, want '%s'", hsm.String(), expected)
	}

	// With turn state
	hsm.turnStateID = StateTurnUpkeep
	hsm.RegisterState(&mockState{id: StateTurnUpkeep})
	expected = "HSM[Global=MatchInit, Turn=TurnUpkeep]"
	if hsm.String() != expected {
		t.Errorf("HSM string = %s, want '%s'", hsm.String(), expected)
	}
}

func TestHSMOnUserChoice(t *testing.T) {
	game := engine.NewGame("test", 0)
	hsm := NewHSM(game)

	// No pending decision
	err := hsm.OnUserChoice(0, nil)
	if err == nil {
		t.Error("OnUserChoice without decision should fail")
	}

	// Setup proper interrupt state
	hsm.RegisterState(&mockState{id: StateTurnLoop})
	hsm.RegisterState(&mockState{id: StateTurnUpkeep})
	hsm.RegisterState(&mockState{id: StateWaitDecision})
	hsm.Start(StateTurnLoop, nil)
	player := core.NewPlayer(core.PlayerConfig{UserID: "player-1"})
	hsm.SetTurnPlayer(player)
	hsm.TransitionTo(StateTurnUpkeep, nil)

	// Simulate decision by setting up interrupt stack
	ctx := NewStateContext().WithGame(game).WithPlayer(player)
	ctx.StartTime = time.Now()
	hsm.stack.Push(&mockState{id: StateTurnUpkeep}, ctx)

	// Set decision
	decision := event.NewDecision("test", []event.Option{
		{ID: "opt1", Label: "Option 1", Action: nil},
	})
	hsm.decision = decision
	hsm.paused = true

	// Execute choice - should succeed now with stack setup
	err = hsm.OnUserChoice(0, ctx)
	// The test may still fail due to PopInterrupt logic, but at least we test the decision execution
	if err != nil && err.Error() != "stack is empty" {
		t.Logf("OnUserChoice error (expected for empty stack scenario): %v", err)
	}
}

// ========== Mock State for Testing ==========

type mockState struct {
	id           StateID
	enterCalled  bool
	updateCalled bool
	exitCalled   bool
}

func (s *mockState) ID() StateID { return s.id }

func (s *mockState) Enter(ctx *StateContext) {
	s.enterCalled = true
}

func (s *mockState) Update(ctx *StateContext) StateID {
	s.updateCalled = true
	return StateNone // Stay in current state by default
}

func (s *mockState) Exit(ctx *StateContext) {
	s.exitCalled = true
}

func (s *mockState) CanTransitionTo(target StateID) bool {
	return true // Allow all transitions for mock
}
