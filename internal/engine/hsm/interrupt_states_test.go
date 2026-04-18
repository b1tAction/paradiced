package hsm

import (
	"testing"
	"time"

	"github.com/b1tAction/paradiced/internal/core"
	"github.com/b1tAction/paradiced/internal/engine"
	"github.com/b1tAction/paradiced/internal/event"
	"github.com/b1tAction/paradiced/pkg/id"
)

// ========== WaitDecisionState Tests ==========

func TestWaitDecisionState_Enter(t *testing.T) {
	game := engine.NewGame(id.NewGameID(), 0)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	game.AddPlayer(player)

	decision := event.NewDecision("Choose an option", []event.Option{
		{ID: "opt1", Label: "Option 1"},
		{ID: "opt2", Label: "Option 2"},
	})

	state := NewWaitDecisionState().WithDecision(decision)
	ctx := NewStateContext().
		WithGame(game).
		WithPlayer(player)

	state.Enter(ctx)

	if ctx.Success != true {
		t.Errorf("Enter should succeed, got error: %v", ctx.Error)
	}
	if state.completed != false {
		t.Error("completed should be false initially")
	}
	if state.choice != -1 {
		t.Error("choice should be -1 (no choice made yet)")
	}
}

func TestWaitDecisionState_Enter_NoDecision(t *testing.T) {
	state := NewWaitDecisionState()
	ctx := NewStateContext()

	state.Enter(ctx)

	if ctx.Success != false {
		t.Error("Enter should fail without decision")
	}
	if ctx.Error == nil {
		t.Error("Error should be set")
	}
}

func TestWaitDecisionState_Enter_ContextDecision(t *testing.T) {
	game := engine.NewGame(id.NewGameID(), 0)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	game.AddPlayer(player)

	decision := event.NewDecision("Choose an option", []event.Option{
		{ID: "opt1", Label: "Option 1"},
	})

	state := NewWaitDecisionState()
	ctx := NewStateContext().
		WithGame(game).
		WithPlayer(player).
		WithDecision(decision)

	state.Enter(ctx)

	if ctx.Success != true {
		t.Errorf("Enter should succeed, got error: %v", ctx.Error)
	}
	if state.decision != decision {
		t.Error("decision should be set from context")
	}
}

func TestWaitDecisionState_Enter_CustomTimeout(t *testing.T) {
	decision := event.NewDecision("Choose", []event.Option{
		{ID: "opt1", Label: "Option 1"},
	})

	state := NewWaitDecisionState().WithDecision(decision)
	ctx := NewStateContext().WithTimeout(10 * time.Second)

	state.Enter(ctx)

	if state.timeout != 10*time.Second {
		t.Errorf("timeout should be 10s, got %v", state.timeout)
	}
}

func TestWaitDecisionState_Update_Waiting(t *testing.T) {
	decision := event.NewDecision("Choose", []event.Option{
		{ID: "opt1", Label: "Option 1"},
	})

	state := NewWaitDecisionState().WithDecision(decision)
	state.startTime = time.Now()
	ctx := NewStateContext()

	state.Enter(ctx)
	nextID := state.Update(ctx)

	// Should return StateNone while waiting
	if nextID != StateNone {
		t.Errorf("Update should return StateNone while waiting, got %s", nextID.String())
	}
}

func TestWaitDecisionState_Update_Completed(t *testing.T) {
	decision := event.NewDecision("Choose", []event.Option{
		{ID: "opt1", Label: "Option 1"},
	})

	state := &WaitDecisionState{
		BaseInterruptState: BaseInterruptState{id: StateWaitDecision},
		decision:           decision,
		completed:          true,
	}
	ctx := NewStateContext()

	nextID := state.Update(ctx)

	// Should return StateNone to signal PopInterrupt
	if nextID != StateNone {
		t.Errorf("Update should return StateNone for completed, got %s", nextID.String())
	}
}

func TestWaitDecisionState_OnUserChoice(t *testing.T) {
	decision := event.NewDecision("Choose", []event.Option{
		{ID: "opt1", Label: "Option 1"},
		{ID: "opt2", Label: "Option 2"},
	})

	state := NewWaitDecisionState().WithDecision(decision)
	ctx := NewStateContext()

	state.OnUserChoice(ctx, 1)

	if state.choice != 1 {
		t.Errorf("choice should be 1, got %d", state.choice)
	}
}

func TestWaitDecisionState_OnUserChoice_InvalidIndex(t *testing.T) {
	decision := event.NewDecision("Choose", []event.Option{
		{ID: "opt1", Label: "Option 1"},
	})

	state := NewWaitDecisionState().WithDecision(decision).WithDefaultOption(0)
	ctx := NewStateContext()

	// Invalid index should use default
	state.OnUserChoice(ctx, -1)

	if state.choice != 0 {
		t.Errorf("choice should be 0 (default) for invalid index, got %d", state.choice)
	}
}

func TestWaitDecisionState_ExecuteOption(t *testing.T) {
	actionExecuted := false
	decision := event.NewDecision("Choose", []event.Option{
		{
			ID:     "opt1",
			Label:  "Option 1",
			Action: func(ctx *event.Context) { actionExecuted = true },
		},
	})

	game := engine.NewGame(id.NewGameID(), 0)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	game.AddPlayer(player)

	state := NewWaitDecisionState().WithDecision(decision)
	ctx := NewStateContext().WithGame(game).WithPlayer(player)

	state.Enter(ctx)
	state.executeOption(ctx, 0)

	if actionExecuted != true {
		t.Error("option action should be executed")
	}
	if state.completed != true {
		t.Error("completed should be true after executeOption")
	}
}

func TestWaitDecisionState_Exit(t *testing.T) {
	decision := event.NewDecision("Choose", []event.Option{
		{ID: "opt1", Label: "Option 1"},
	})

	state := NewWaitDecisionState().WithDecision(decision)
	state.completed = true
	state.choice = 1
	state.startTime = time.Now()

	ctx := NewStateContext()
	state.Exit(ctx)

	if state.decision != nil {
		t.Error("decision should be nil after Exit")
	}
	if state.completed != false {
		t.Error("completed should be false after Exit")
	}
	if state.choice != -1 {
		t.Error("choice should be -1 after Exit")
	}
}

func TestWaitDecisionState_WithMethods(t *testing.T) {
	decision := event.NewDecision("Choose", []event.Option{
		{ID: "opt1", Label: "Option 1"},
	})

	state := NewWaitDecisionState().
		WithDecision(decision).
		WithTimeout(15 * time.Second).
		WithDefaultOption(1)

	if state.decision != decision {
		t.Error("decision should be set")
	}
	if state.timeout != 15*time.Second {
		t.Errorf("timeout should be 15s, got %v", state.timeout)
	}
	if state.defaultOption != 1 {
		t.Errorf("defaultOption should be 1, got %d", state.defaultOption)
	}
}

// ========== Interrupt State Factory Tests ==========

func TestInterruptStateFactory_CreateState(t *testing.T) {
	factory := &InterruptStateFactory{}

	state := factory.CreateState(StateWaitDecision)
	if state == nil {
		t.Error("CreateState should create WaitDecisionState")
	}
	if state.ID() != StateWaitDecision {
		t.Errorf("State ID should be WaitDecision, got %s", state.ID().String())
	}

	// Invalid ID should return nil
	state = factory.CreateState(StateTurnUpkeep)
	if state != nil {
		t.Error("CreateState should return nil for non-interrupt ID")
	}
}

// ========== BaseInterruptState Tests ==========

func TestBaseInterruptState_ID(t *testing.T) {
	state := &BaseInterruptState{id: StateWaitDecision}
	if state.ID() != StateWaitDecision {
		t.Errorf("ID should be WaitDecision, got %s", state.ID().String())
	}
}

func TestBaseInterruptState_CanTransitionTo(t *testing.T) {
	state := &BaseInterruptState{id: StateWaitDecision}

	// Interrupt states should not allow direct transitions
	if state.CanTransitionTo(StateTurnUpkeep) != false {
		t.Error("Interrupt states should not allow transitions")
	}
	if state.CanTransitionTo(StateWaitDecision) != false {
		t.Error("Interrupt states should not allow transitions")
	}
}

// ========== RegisterInterruptStates Tests ==========

func TestRegisterInterruptStates(t *testing.T) {
	game := engine.NewGame(id.NewGameID(), 0)
	hsm := NewHSM(game)

	err := RegisterInterruptStates(hsm)
	if err != nil {
		t.Errorf("RegisterInterruptStates should succeed, got error: %v", err)
	}

	// Verify state is registered
	state := hsm.GetState(StateWaitDecision)
	if state == nil {
		t.Error("WaitDecisionState should be registered")
	}
}
