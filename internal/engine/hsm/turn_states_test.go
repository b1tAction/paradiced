package hsm

import (
	"testing"
	"time"

	"github.com/b1tAction/fated/internal/core"
	"github.com/b1tAction/fated/internal/engine"
	"github.com/b1tAction/fated/internal/gamemap"
	"github.com/b1tAction/fated/pkg/id"
	"github.com/b1tAction/fated/pkg/rng"
)

// ========== TurnUpkeepState Tests ==========

func TestTurnUpkeepState_Enter_NormalFlow(t *testing.T) {
	// Setup
	game := engine.NewGame(id.NewGameID(), 0)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player.HP = 5
	player.LP = 3
	player.Position = 10
	game.AddPlayer(player)

	mapEngine := gamemap.NewMapEngine(100)
	mapAdapter := NewMapEngineWrapper(mapEngine)

	state := NewTurnUpkeepState()
	ctx := NewStateContext().
		WithGame(game).
		WithPlayer(player).
		WithMapEngine(mapAdapter).
		WithBus(NewEventBusWrapper(game.Bus))

	// Execute
	state.Enter(ctx)

	// Verify
	if ctx.Success != true {
		t.Errorf("Enter should succeed, got error: %v", ctx.Error)
	}
	if state.skipTurn != false {
		t.Error("skipTurn should be false for normal flow")
	}
	if state.isDead != false {
		t.Error("isDead should be false for alive player")
	}
}

func TestTurnUpkeepState_Enter_SkipTurn(t *testing.T) {
	// Setup
	game := engine.NewGame(id.NewGameID(), 0)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player.SkipTurn = true
	game.AddPlayer(player)

	state := NewTurnUpkeepState()
	ctx := NewStateContext().
		WithGame(game).
		WithPlayer(player).
		WithBus(NewEventBusWrapper(game.Bus))

	// Execute
	state.Enter(ctx)

	// Verify
	if state.skipTurn != true {
		t.Error("skipTurn should be true")
	}
	if ctx.IsSkipTurn() != true {
		t.Error("StateContext should have skip_turn marker")
	}
	if player.SkipTurn != false {
		t.Error("player.SkipTurn should be cleared")
	}
}

func TestTurnUpkeepState_Enter_DeadPlayer(t *testing.T) {
	// Setup
	game := engine.NewGame(id.NewGameID(), 0)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player.IsDead = true
	player.Position = 50
	game.AddPlayer(player)

	// Setup map with checkpoint
	mapEngine := gamemap.NewMapEngine(100)
	configs := map[int]gamemap.CellType{30: gamemap.CellTypeCheckpoint}
	mapEngine.GenerateLinearMap(configs)
	mapAdapter := NewMapEngineWrapper(mapEngine)

	state := NewTurnUpkeepState()
	ctx := NewStateContext().
		WithGame(game).
		WithPlayer(player).
		WithMapEngine(mapAdapter).
		WithBus(NewEventBusWrapper(game.Bus))

	// Execute
	state.Enter(ctx)

	// Verify
	if state.isDead != true {
		t.Error("isDead should be true")
	}
	// Player should respawn at checkpoint
	if player.Position != 30 {
		t.Errorf("Player should respawn at checkpoint 30, got %d", player.Position)
	}
}

func TestTurnUpkeepState_Update_SkipTurn(t *testing.T) {
	state := &TurnUpkeepState{
		BaseTurnState: BaseTurnState{id: StateTurnUpkeep},
		skipTurn:      true,
	}
	ctx := NewStateContext()

	nextID := state.Update(ctx)

	if nextID != StateTurnEnd {
		t.Errorf("Update should return StateTurnEnd for skipTurn, got %s", nextID.String())
	}
}

func TestTurnUpkeepState_Update_NormalFlow(t *testing.T) {
	state := NewTurnUpkeepState()
	ctx := NewStateContext()

	nextID := state.Update(ctx)

	if nextID != StateMainAction {
		t.Errorf("Update should return StateMainAction, got %s", nextID.String())
	}
}

// ========== MainActionState Tests ==========

func TestMainActionState_Enter(t *testing.T) {
	game := engine.NewGame(id.NewGameID(), 0)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	game.AddPlayer(player)

	state := NewMainActionState()
	ctx := NewStateContext().
		WithGame(game).
		WithPlayer(player).
		WithBus(NewEventBusWrapper(game.Bus))

	state.Enter(ctx)

	if state.waitingForAction != true {
		t.Error("waitingForAction should be true after Enter")
	}
	if state.diceRolled != false {
		t.Error("diceRolled should be false initially")
	}
}

func TestMainActionState_OnRollDice(t *testing.T) {
	state := NewMainActionState()
	ctx := NewStateContext()

	state.OnRollDice(ctx, 6)

	if state.diceSteps != 6 {
		t.Errorf("diceSteps should be 6, got %d", state.diceSteps)
	}
	if state.diceRolled != true {
		t.Error("diceRolled should be true after OnRollDice")
	}
	if state.waitingForAction != false {
		t.Error("waitingForAction should be false after dice roll")
	}
}

func TestMainActionState_Update_DiceRolled(t *testing.T) {
	state := &MainActionState{
		BaseTurnState: BaseTurnState{id: StateMainAction},
		diceRolled:    true,
		diceSteps:     5,
	}
	ctx := NewStateContext()

	nextID := state.Update(ctx)

	if nextID != StateTurnMoving {
		t.Errorf("Update should return StateTurnMoving after dice rolled, got %s", nextID.String())
	}
	if ctx.GetDiceSteps() != 5 {
		t.Errorf("StateContext should have dice_steps=5, got %d", ctx.GetDiceSteps())
	}
}

func TestMainActionState_Update_Waiting(t *testing.T) {
	state := &MainActionState{
		BaseTurnState:    BaseTurnState{id: StateMainAction},
		waitingForAction: true,
		diceRolled:       false,
		startTime:        time.Now(),
		timeout:          45 * time.Second,
	}
	ctx := NewStateContext()

	nextID := state.Update(ctx)

	if nextID != StateNone {
		t.Errorf("Update should return StateNone while waiting, got %s", nextID.String())
	}
}

func TestMainActionState_defaultDiceRoll(t *testing.T) {
	state := NewMainActionState()
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	ctx := NewStateContext().WithPlayer(player)

	tests := []struct {
		diceType rng.DiceType
		expected int
	}{
		{rng.DiceTypeGold, 6},
		{rng.DiceTypeSilver, 4},
		{rng.DiceTypeCopper, 3},
		{rng.DiceTypeWood, 2},
		{rng.DiceTypeNone, 2},
	}

	for _, tt := range tests {
		ctx.SetDiceType(player.ID.UUID(), tt.diceType)
		result := state.defaultDiceRoll(ctx)
		if result != tt.expected {
			t.Errorf("defaultDiceRoll for %s should return %d, got %d", tt.diceType.String(), tt.expected, result)
		}
	}
}

// ========== TurnMovingState Tests ==========

func TestTurnMovingState_Enter_FellDown(t *testing.T) {
	game := engine.NewGame(id.NewGameID(), 0)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player.HP = 5
	player.Position = 20
	game.AddPlayer(player)

	// Setup map with Fragile cell
	mapEngine := gamemap.NewMapEngine(100)
	configs := map[int]gamemap.CellType{25: gamemap.CellTypeFragile}
	mapEngine.GenerateLinearMap(configs)
	mapAdapter := NewMapEngineWrapper(mapEngine)

	state := NewTurnMovingState()
	ctx := NewStateContext().
		WithGame(game).
		WithPlayer(player).
		WithMapEngine(mapAdapter).
		WithBus(NewEventBusWrapper(game.Bus))
	ctx.SetInt(KeyDiceSteps, 5)

	state.Enter(ctx)

	// Note: Fragile handling is in MapEngine.CalculatePath
	// This test verifies the state setup
	if state.pathResult.TargetIndex < 0 {
		t.Error("TargetIndex should be valid after movement")
	}
}

func TestTurnMovingState_Update_FellDown(t *testing.T) {
	state := &TurnMovingState{
		BaseTurnState: BaseTurnState{id: StateTurnMoving},
		fellDown:      true,
	}
	ctx := NewStateContext()

	nextID := state.Update(ctx)

	if nextID != StateTurnEnd {
		t.Errorf("Update should return StateTurnEnd for fellDown, got %s", nextID.String())
	}
	if ctx.IsFellDown() != true {
		t.Error("StateContext should have fell_down marker")
	}
}

func TestTurnMovingState_Update_ReachedEnd(t *testing.T) {
	state := &TurnMovingState{
		BaseTurnState: BaseTurnState{id: StateTurnMoving},
		reachedEnd:    true,
	}
	ctx := NewStateContext()

	nextID := state.Update(ctx)

	// ReachedEnd should go to TurnLanded first, then TurnLoop handles BossBattle
	if nextID != StateTurnLanded {
		t.Errorf("Update should return StateTurnLanded for reachedEnd, got %s", nextID.String())
	}
}

func TestTurnMovingState_Update_NormalFlow(t *testing.T) {
	state := &TurnMovingState{
		BaseTurnState: BaseTurnState{id: StateTurnMoving},
		fellDown:      false,
		reachedEnd:    false,
	}
	ctx := NewStateContext()

	nextID := state.Update(ctx)

	if nextID != StateTurnLanded {
		t.Errorf("Update should return StateTurnLanded, got %s", nextID.String())
	}
}

// ========== TurnLandedState Tests ==========

func TestTurnLandedState_Enter(t *testing.T) {
	game := engine.NewGame(id.NewGameID(), 0)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player.Position = 30
	game.AddPlayer(player)

	mapEngine := gamemap.NewMapEngine(100)
	configs := map[int]gamemap.CellType{30: gamemap.CellTypeCheckpoint}
	mapEngine.GenerateLinearMap(configs)
	mapAdapter := NewMapEngineWrapper(mapEngine)

	state := NewTurnLandedState()
	ctx := NewStateContext().
		WithGame(game).
		WithPlayer(player).
		WithMapEngine(mapAdapter).
		WithBus(NewEventBusWrapper(game.Bus))

	state.Enter(ctx)

	if ctx.Success != true {
		t.Errorf("Enter should succeed, got error: %v", ctx.Error)
	}
	if state.cellType != gamemap.CellTypeCheckpoint {
		t.Errorf("cellType should be Checkpoint, got %d", state.cellType)
	}
}

func TestTurnLandedState_Update(t *testing.T) {
	state := NewTurnLandedState()
	ctx := NewStateContext()

	nextID := state.Update(ctx)

	if nextID != StateTurnEvent {
		t.Errorf("Update should return StateTurnEvent, got %s", nextID.String())
	}
}

// ========== TurnEventState Tests ==========

func TestTurnEventState_Enter(t *testing.T) {
	game := engine.NewGame(id.NewGameID(), 0)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player.Position = 20
	game.AddPlayer(player)

	state := NewTurnEventState()
	ctx := NewStateContext().
		WithGame(game).
		WithPlayer(player).
		WithBus(NewEventBusWrapper(game.Bus))

	state.Enter(ctx)

	// Note: DrawEventAction requires event pool, this test verifies state setup
	if state.eventDrawn != true {
		t.Error("eventDrawn should be true after Enter")
	}
}

func TestTurnEventState_Update(t *testing.T) {
	state := NewTurnEventState()
	ctx := NewStateContext()

	nextID := state.Update(ctx)

	if nextID != StateTurnEnd {
		t.Errorf("Update should return StateTurnEnd, got %s", nextID.String())
	}
}

// ========== TurnEndState Tests ==========

func TestTurnEndState_Enter(t *testing.T) {
	game := engine.NewGame(id.NewGameID(), 0)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player.HP = 5
	player.LP = 3
	player.Position = 20
	game.AddPlayer(player)

	state := NewTurnEndState()
	ctx := NewStateContext().
		WithGame(game).
		WithPlayer(player).
		WithBus(NewEventBusWrapper(game.Bus))

	state.Enter(ctx)

	if ctx.Success != true {
		t.Errorf("Enter should succeed, got error: %v", ctx.Error)
	}
}

func TestTurnEndState_Enter_TickBuffs(t *testing.T) {
	game := engine.NewGame(id.NewGameID(), 0)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	game.AddPlayer(player)

	// Add buff with 1 duration (will expire)
	buff1 := core.NewBuff(core.BuffTypeCurse, 1)
	game.ApplyBuffToPlayer(player, buff1)

	// Add buff with 3 duration (won't expire)
	buff2 := core.NewBuff(core.BuffTypeDivine, 3)
	game.ApplyBuffToPlayer(player, buff2)

	state := NewTurnEndState()
	ctx := NewStateContext().
		WithGame(game).
		WithPlayer(player).
		WithBus(NewEventBusWrapper(game.Bus))

	state.Enter(ctx)

	// Verify buff1 expired and unsubscribed
	if len(player.ActiveBuffs) != 1 {
		t.Errorf("Player should have 1 buff remaining, got %d", len(player.ActiveBuffs))
	}
	if player.ActiveBuffs[0].Type != core.BuffTypeDivine {
		t.Errorf("Remaining buff should be Divine, got %s", string(player.ActiveBuffs[0].Type))
	}
}

func TestTurnEndState_Enter_FactionCharging(t *testing.T) {
	game := engine.NewGame(id.NewGameID(), 0)
	player := core.NewPlayer(core.PlayerConfig{
		ID:      id.NewPlayerID(),
		Faction: core.FactionQingLong,
	})
	game.AddPlayer(player)

	state := NewTurnEndState()
	ctx := NewStateContext().
		WithGame(game).
		WithPlayer(player).
		WithBus(NewEventBusWrapper(game.Bus))

	state.Enter(ctx)

	// QingLong charges every turn (max 1)
	if player.GetChargeCount() != 1 {
		t.Errorf("QingLong charge count should be 1, got %d", player.GetChargeCount())
	}
}

func TestTurnEndState_Update(t *testing.T) {
	state := NewTurnEndState()
	ctx := NewStateContext()

	nextID := state.Update(ctx)

	if nextID != StateNone {
		t.Errorf("Update should return StateNone to signal completion, got %s", nextID.String())
	}
}

// ========== State Factory Tests ==========

func TestTurnStateFactory_CreateState(t *testing.T) {
	factory := &TurnStateFactory{}

	tests := []StateID{
		StateTurnUpkeep,
		StateMainAction,
		StateTurnMoving,
		StateTurnLanded,
		StateTurnEvent,
		StateTurnEnd,
	}

	for _, id := range tests {
		state := factory.CreateState(id)
		if state == nil {
			t.Errorf("CreateState should create state for %s", id.String())
		}
		if state.ID() != id {
			t.Errorf("State ID should be %s, got %s", id.String(), state.ID().String())
		}
	}

	// Invalid ID should return nil
	state := factory.CreateState(StateNone)
	if state != nil {
		t.Error("CreateState should return nil for StateNone")
	}
}

// ========== State Error Tests ==========

func TestStateError(t *testing.T) {
	err := NewStateError(StateTurnUpkeep, "player is nil")

	if err.StateID != StateTurnUpkeep {
		t.Errorf("StateError ID should be TurnUpkeep, got %s", err.StateID.String())
	}
	if err.Message != "player is nil" {
		t.Errorf("StateError message should be 'player is nil', got %s", err.Message)
	}
	expectedStr := "TurnUpkeep: player is nil"
	if err.Error() != expectedStr {
		t.Errorf("Error string should be '%s', got '%s'", expectedStr, err.Error())
	}
}
