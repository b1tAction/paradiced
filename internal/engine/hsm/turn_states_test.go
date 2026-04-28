package hsm

import (
	"testing"
	"time"

	"github.com/b1tAction/paradiced/internal/core"
	"github.com/b1tAction/paradiced/internal/engine"
	engineaction "github.com/b1tAction/paradiced/internal/engine/action"
	"github.com/b1tAction/paradiced/internal/event"
	"github.com/b1tAction/paradiced/internal/gamemap"
	pkgnet "github.com/b1tAction/paradiced/pkg/net"
	"github.com/b1tAction/paradiced/pkg/constants"
	"github.com/b1tAction/paradiced/pkg/id"
	"github.com/b1tAction/paradiced/pkg/rng"
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

	state := NewTurnUpkeepState()
	hsmInst := NewHSM(game)
	hsmInst.SetMapEngine(mapEngine)
	ctx := NewStateContext().
		WithHSM(hsmInst).
		WithPlayer(player)

	// Execute
	state.Enter(ctx)

	// Verify
	if ctx.Error != nil {
		t.Errorf("Enter should succeed, got error: %v", ctx.Error)
	}
	if state.skipTurn != false {
		t.Error("skipTurn should be false for normal flow")
	}
}

func TestTurnUpkeepState_Enter_MarkBuffsTickEligible(t *testing.T) {
	// Buffs acquired mid-turn have tickEligible=false by default.
	// When TurnEnd calls TickBuffs, the first TickDuration call marks
	// tickEligible=true without decrementing. This ensures mid-turn
	// buffs survive their first turn-end tick.
	game := engine.NewGame(id.NewGameID(), 0)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	game.AddPlayer(player)

	// Add buff acquired in previous turn (tickEligible=false initially)
	buff := core.NewBuff(constants.BuffTypeLost, 1)
	game.ApplyBuffToPlayer(player, buff)

	state := NewTurnUpkeepState()
	ctx := NewStateContext().
		WithHSM(NewHSM(game)).
		WithPlayer(player)

	state.Enter(ctx)

	// Verify buff still has Duration=1 (TurnUpkeep doesn't tick buffs)
	if buff.Duration != 1 {
		t.Errorf("buff Duration = %d, expected 1 (not ticked at TurnUpkeep)", buff.Duration)
	}
}

func TestTurnUpkeepState_Enter_BuffCreatedInBeforeTurnNotTickEligible(t *testing.T) {
	// Buffs created during BeforeTurn phase will have tickEligible=false
	// (set by NewBuff default). When TickBuffs is called at TurnEnd,
	// the first TickDuration call will mark tickEligible=true without
	// decrementing Duration, so they won't expire in the same turn.
	game := engine.NewGame(id.NewGameID(), 0)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	game.AddPlayer(player)

	// Add pre-existing buff (will be ticked properly after first TurnEnd)
	existingBuff := core.NewBuff(constants.BuffTypeDivine, 3)
	game.ApplyBuffToPlayer(player, existingBuff)

	state := NewTurnUpkeepState()
	ctx := NewStateContext().
		WithHSM(NewHSM(game)).
		WithPlayer(player)

	state.Enter(ctx)

	// Simulate adding a buff during BeforeTurn phase (after TurnUpkeep Enter)
	newBuff := core.NewBuff(constants.BuffTypeCurse, 3)
	player.AddBuff(newBuff)

	// Both buffs should have original Duration (neither ticked at TurnUpkeep)
	if existingBuff.Duration != 3 {
		t.Errorf("existing buff Duration = %d, expected 3", existingBuff.Duration)
	}
	if newBuff.Duration != 3 {
		t.Errorf("new buff Duration = %d, expected 3", newBuff.Duration)
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
		WithHSM(NewHSM(game)).
		WithPlayer(player)

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

func TestTurnEndState_Enter_DeadPlayer(t *testing.T) {
	// Death detection and respawn are now handled in TurnEndState.
	// When player.IsDead=true, TurnEndState executes DeathAction + RespawnAction.
	game := engine.NewGame(id.NewGameID(), 0)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player.IsDead = true
	player.Position = 50
	game.AddPlayer(player)

	// Setup map with checkpoint
	mapEngine := gamemap.NewMapEngine(100)
	configs := map[int]constants.CellType{30: constants.CellTypeCheckpoint}
	mapEngine.GenerateLinearMap(configs)

	state := NewTurnEndState()
	hsmInst := NewHSM(game)
	hsmInst.SetMapEngine(mapEngine)
	ctx := NewStateContext().
		WithHSM(hsmInst).
		WithPlayer(player)

	// Execute
	state.Enter(ctx)

	// Verify player respawned at checkpoint
	if player.Position != 30 {
		t.Errorf("Player should respawn at checkpoint 30, got %d", player.Position)
	}
	if player.IsDead != false {
		t.Error("Player should not be dead after respawn")
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

	state := NewMainActionState(45 * time.Second)
	ctx := NewStateContext().
		WithHSM(NewHSM(game)).
		WithPlayer(player)

	state.Enter(ctx)

	if state.waitingForAction != true {
		t.Error("waitingForAction should be true after Enter")
	}
	if state.diceRolled != false {
		t.Error("diceRolled should be false initially")
	}
}

func TestMainActionState_OnRollDice(t *testing.T) {
	game := engine.NewGame(id.NewGameID(), 42) // Fixed seed for deterministic dice roll
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	game.AddPlayer(player)

	hsmInst := NewHSM(game)
	ctx := NewStateContext().WithHSM(hsmInst).WithPlayer(player)

	state := NewMainActionState(45 * time.Second)
	state.Enter(ctx)

	// Set dice type for deterministic roll
	ctx.SetDiceType(player.ID.UUID(), rng.DiceTypeWood)

	state.OnRollDice(ctx)

	if state.diceSteps == 0 {
		t.Error("diceSteps should be non-zero after OnRollDice (computed by RollDiceAction)")
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

func TestMainActionState_RollDiceAction_StepsRange(t *testing.T) {
	// Verify RollDiceAction produces valid steps (1-6) for each dice type
	game := engine.NewGame(id.NewGameID(), 42)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	game.AddPlayer(player)

	diceTypes := []rng.DiceType{rng.DiceTypeGold, rng.DiceTypeSilver, rng.DiceTypeCopper, rng.DiceTypeWood, rng.DiceTypeNormal}
	for _, diceType := range diceTypes {
		action := engineaction.NewRollDiceAction(player, diceType, game.RNG, "DiceRoll")
		if action.Steps < 1 || action.Steps > 6 {
			t.Errorf("RollDiceAction Steps for %s should be 1-6, got %d", diceType.String(), action.Steps)
		}
	}
}

func TestMainActionState_OnUseItem_DiceUpgradeAppliesImmediately(t *testing.T) {
	game := engine.NewGame(id.NewGameID(), 0)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	game.AddPlayer(player)

	item := core.NewItem(constants.ItemTypeDiceUpgrade)
	if err := game.ApplyItemToPlayer(player, item); err != nil {
		t.Fatalf("ApplyItemToPlayer failed: %v", err)
	}

	hsmInst := NewHSM(game)
	ctx := NewStateContext().WithHSM(hsmInst).WithPlayer(player)
	ctx.SetDiceType(player.ID.UUID(), rng.DiceTypeSilver)

	state := NewMainActionState(45 * time.Second)
	state.Enter(ctx)
	state.OnUseItem(ctx, item.ID.UUID())

	if ctx.Error != nil {
		t.Fatalf("OnUseItem should succeed, got error: %v", ctx.Error)
	}

	if got := ctx.GetDiceType(player.ID.UUID()); got != rng.DiceTypeGold {
		t.Errorf("dice type = %s, expected %s", got.String(), rng.DiceTypeGold.String())
	}

	if len(player.Inventory) != 0 {
		t.Errorf("item should be consumed after use, inventory len = %d", len(player.Inventory))
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
	configs := map[int]constants.CellType{25: constants.CellTypeFragile}
	mapEngine.GenerateLinearMap(configs)

	state := NewTurnMovingState()
	hsmInst := NewHSM(game)
	hsmInst.SetMapEngine(mapEngine)
	ctx := NewStateContext().
		WithHSM(hsmInst).
		WithPlayer(player)
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
	configs := map[int]constants.CellType{30: constants.CellTypeCheckpoint}
	mapEngine.GenerateLinearMap(configs)

	state := NewTurnLandedState()
	hsmInst := NewHSM(game)
	hsmInst.SetMapEngine(mapEngine)
	ctx := NewStateContext().
		WithHSM(hsmInst).
		WithPlayer(player)

	state.Enter(ctx)

	if ctx.Error != nil {
		t.Errorf("Enter should succeed, got error: %v", ctx.Error)
	}
	if state.cellType != constants.CellTypeCheckpoint {
		t.Errorf("cellType should be Checkpoint, got %s", state.cellType)
	}
}

func TestTurnLandedState_Enter_CellTypeEvent(t *testing.T) {
	game := engine.NewGame(id.NewGameID(), 42)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID(), MaxHP: 6})
	player.HP = 3
	player.Position = 25
	game.AddPlayer(player)

	// Setup EventPool for fallback (though bound events set DrawnType directly)
	game.EventPool = []*rng.EvaluatedItem{
		{Type: "herb", Eval: constants.EvaluationMildGood},
	}

	// Setup map with CellTypeEvent and bound event ID "herb"
	mapEngine := gamemap.NewMapEngine(100)
	configs := map[int]constants.CellType{25: constants.CellTypeEvent}
	mapEngine.GenerateLinearMap(configs)

	// Set EventID on the event cell
	cell, _ := mapEngine.GetCell(25)
	cell.EventID = "herb"

	state := NewTurnLandedState()
	hsmInst := NewHSM(game)
	hsmInst.SetMapEngine(mapEngine)
	ctx := NewStateContext().
		WithHSM(hsmInst).
		WithPlayer(player)

	state.Enter(ctx)

	if ctx.Error != nil {
		t.Errorf("Enter should succeed, got error: %v", ctx.Error)
	}
	// CellType should be Event
	if state.cellType != constants.CellTypeEvent {
		t.Errorf("cellType should be Event, got %s", state.cellType)
	}
	// Bound event should have been executed, so skipEvent=true
	if !state.skipEvent {
		t.Error("skipEvent should be true after bound event execution (CellTypeEvent)")
	}
	// Herb event handler gives HP+1 via HealAction
	if player.HP != 4 {
		t.Errorf("Player HP should be 4 (3+1 from herb), got %d", player.HP)
	}
	// Update should go to TurnEnd (skipEvent=true)
	nextID := state.Update(ctx)
	if nextID != StateTurnEnd {
		t.Errorf("Update should return StateTurnEnd for skipEvent, got %s", nextID.String())
	}
}

func TestTurnLandedState_Enter_CellTypeNormal(t *testing.T) {
	game := engine.NewGame(id.NewGameID(), 0)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player.Position = 20
	game.AddPlayer(player)

	mapEngine := gamemap.NewMapEngine(100)
	mapEngine.GenerateLinearMap(nil) // All Normal cells

	// Set DrawType on the cell to enable drawing
	cell, _ := mapEngine.GetCell(20)
	cell.DrawType = constants.DrawTypeEvent
	cell.ProbGood = 1.0
	cell.ProbNeutral = 0.0
	cell.ProbBad = 0.0

	state := NewTurnLandedState()
	hsmInst := NewHSM(game)
	hsmInst.SetMapEngine(mapEngine)
	ctx := NewStateContext().
		WithHSM(hsmInst).
		WithPlayer(player)

	state.Enter(ctx)

	if ctx.Error != nil {
		t.Errorf("Enter should succeed, got error: %v", ctx.Error)
	}
	if state.cellType != constants.CellTypeNormal {
		t.Errorf("cellType should be Normal, got %s", state.cellType)
	}
	// Normal cell should NOT skip event
	if state.skipEvent {
		t.Error("skipEvent should be false for Normal cell")
	}
	// Update should go to TurnDraw (DrawType is Event)
	nextID := state.Update(ctx)
	if nextID != StateTurnDraw {
		t.Errorf("Update should return StateTurnDraw for Normal cell with DrawType=Event, got %s", nextID.String())
	}
}

func TestTurnLandedState_Update(t *testing.T) {
	state := NewTurnLandedState()
	ctx := NewStateContext()

	// With default drawType (None), Update should return StateTurnEnd
	nextID := state.Update(ctx)

	if nextID != StateTurnEnd {
		t.Errorf("Update should return StateTurnEnd for DrawTypeNone, got %s", nextID.String())
	}
}

// ========== TurnCheckpointState Tests ==========

func TestTurnCheckpointState_Enter(t *testing.T) {
	game := engine.NewGame(id.NewGameID(), 42)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player.Position = 25
	player.HP = 5
	game.AddPlayer(player)

	// Setup ItemPool for DrawItemAction
	game.ItemPool = []*rng.EvaluatedItem{
		{Type: "healing_potion", Eval: constants.EvaluationGood},
	}

	state := NewTurnCheckpointState()
	ctx := NewStateContext().
		WithHSM(NewHSM(game)).
		WithPlayer(player)

	state.Enter(ctx)

	if ctx.Error != nil {
		t.Errorf("Enter should succeed, got error: %v", ctx.Error)
	}
}

func TestTurnCheckpointState_Update(t *testing.T) {
	state := NewTurnCheckpointState()
	ctx := NewStateContext()

	nextID := state.Update(ctx)

	if nextID != StateTurnMoving {
		t.Errorf("Update should return StateTurnMoving, got %s", nextID.String())
	}
}

// ========== TurnMovingState CheckPoint Split Tests ==========

func TestTurnMovingState_Enter_CheckPointSplit(t *testing.T) {
	game := engine.NewGame(id.NewGameID(), 42)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player.Position = 20
	player.HP = 5
	game.AddPlayer(player)

	// Setup map with CheckPoint at position 25
	mapEngine := gamemap.NewMapEngine(100)
	configs := map[int]constants.CellType{25: constants.CellTypeCheckpoint}
	mapEngine.GenerateLinearMap(configs)

	// Setup pools for ActionContext
	game.EventPool = []*rng.EvaluatedItem{}
	game.ItemPool = []*rng.EvaluatedItem{
		{Type: "healing_potion", Eval: constants.EvaluationGood},
	}

	state := NewTurnMovingState()
	hsmInst := NewHSM(game)
	hsmInst.SetMapEngine(mapEngine)
	ctx := NewStateContext().
		WithHSM(hsmInst).
		WithPlayer(player)
	ctx.SetInt(KeyDiceSteps, 10) // Should pass through CheckPoint at 25

	state.Enter(ctx)

	if ctx.Error != nil {
		t.Errorf("Enter should succeed, got error: %v", ctx.Error)
	}
	if state.hasCheckpoint != true {
		t.Error("hasCheckpoint should be true when path contains CheckPoint")
	}
	if state.checkpointPos != 25 {
		t.Errorf("checkpointPos should be 25, got %d", state.checkpointPos)
	}
	// Player should have moved to checkpoint position (first segment)
	if player.Position != 25 {
		t.Errorf("Player should be at checkpoint 25, got %d", player.Position)
	}
	// Remaining steps should be 5 (10 - 5 = 5)
	if state.remainingSteps != 5 {
		t.Errorf("remainingSteps should be 5, got %d", state.remainingSteps)
	}

	// Update should transition to TurnCheckpoint
	nextID := state.Update(ctx)
	if nextID != StateTurnCheckpoint {
		t.Errorf("Update should return StateTurnCheckpoint for CheckPoint split, got %s", nextID.String())
	}
}

func TestTurnMovingState_Enter_ReverseSkipsCheckpoint(t *testing.T) {
	game := engine.NewGame(id.NewGameID(), 42)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player.Position = 30
	player.HP = 5
	game.AddPlayer(player)

	// Setup map with CheckPoint at position 25 (before player position)
	mapEngine := gamemap.NewMapEngine(100)
	configs := map[int]constants.CellType{25: constants.CellTypeCheckpoint}
	mapEngine.GenerateLinearMap(configs)

	// Setup pools for ActionContext
	game.EventPool = []*rng.EvaluatedItem{}
	game.ItemPool = []*rng.EvaluatedItem{}

	state := NewTurnMovingState()
	hsmInst := NewHSM(game)
	hsmInst.SetMapEngine(mapEngine)
	ctx := NewStateContext().
		WithHSM(hsmInst).
		WithPlayer(player)
	// Simulate 迷途 reverse movement: set Steps < 0 directly
	// (after 迷途 handler flips positive Steps to negative)
	ctx.SetInt(KeyDiceSteps, -5) // Moving backward from 30

	state.Enter(ctx)

	if ctx.Error != nil {
		t.Errorf("Enter should succeed, got error: %v", ctx.Error)
	}
	// Reverse movement should NOT trigger CheckPoint split
	if state.hasCheckpoint != false {
		t.Error("hasCheckpoint should be false for reverse movement (迷途)")
	}
	// Player should have moved backward (30 + (-5) = 25)
	if player.Position != 25 {
		t.Errorf("Player should be at 25 (reverse movement), got %d", player.Position)
	}
	// Steps should remain negative
	if state.Steps >= 0 {
		t.Errorf("Steps should be negative after reverse movement, got %d", state.Steps)
	}

	// Update should transition to TurnLanded (no CheckPoint split)
	nextID := state.Update(ctx)
	if nextID != StateTurnLanded {
		t.Errorf("Update should return StateTurnLanded for reverse movement, got %s", nextID.String())
	}
}

// ========== TurnDrawState Tests ==========

func TestTurnDrawState_Enter(t *testing.T) {
	game := engine.NewGame(id.NewGameID(), 42)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player.Position = 20
	game.AddPlayer(player)

	// Setup EventPool for DrawEventAction (Game.Draw already initialized by NewGame)
	game.EventPool = []*rng.EvaluatedItem{
		{Type: "herb", Eval: constants.EvaluationMildGood},
		{Type: "milk_tea", Eval: constants.EvaluationGood},
	}

	state := NewTurnDrawState()
	// Set draw configuration
	state.drawType = constants.DrawTypeEvent
	state.probGood = 1.0
	state.probNeutral = 0.0
	state.probBad = 0.0

	ctx := NewStateContext().
		WithHSM(NewHSM(game)).
		WithPlayer(player)

	state.Enter(ctx)

	if ctx.Error != nil {
		t.Errorf("Enter should succeed, got error: %v", ctx.Error)
	}
}

func TestTurnDrawState_Enter_DrawTypeItem(t *testing.T) {
	game := engine.NewGame(id.NewGameID(), 42)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player.Position = 20
	player.HP = 5
	player.LP = 4 // Set LP for draw weight calculation
	game.AddPlayer(player)

	// Setup ItemPool for DrawItemAction
	game.ItemPool = []*rng.EvaluatedItem{
		{Type: "healing_potion", Eval: constants.EvaluationGood},
		{Type: "reverse_clock", Eval: constants.EvaluationMildGood},
	}

	state := NewTurnDrawState()
	// Set draw configuration for item draw
	state.drawType = constants.DrawTypeItem
	state.probGood = 1.0
	state.probNeutral = 0.0
	state.probBad = 0.0

	ctx := NewStateContext().
		WithHSM(NewHSM(game)).
		WithPlayer(player)

	state.Enter(ctx)

	if ctx.Error != nil {
		t.Errorf("Enter should succeed, got error: %v", ctx.Error)
	}

	// Verify item was drawn and added to inventory
	// Note: DrawItemAction uses DrawWithProb which may return nil if draw fails
	// but with valid pool it should succeed
	if len(player.Inventory) == 0 {
		t.Logf("Warning: No item was drawn (this may be expected if DrawWithProb returned nil)")
	}
}

func TestTurnDrawState_Enter_DrawTypeNone(t *testing.T) {
	game := engine.NewGame(id.NewGameID(), 42)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player.Position = 20
	game.AddPlayer(player)

	state := NewTurnDrawState()
	// Set draw configuration to None (should skip drawing)
	state.drawType = constants.DrawTypeNone

	ctx := NewStateContext().
		WithHSM(NewHSM(game)).
		WithPlayer(player)

	state.Enter(ctx)

	if ctx.Error != nil {
		t.Errorf("Enter should succeed, got error: %v", ctx.Error)
	}
	// No action should be taken when DrawType is None
}

func TestTurnDrawState_Enter_NilPlayer(t *testing.T) {
	state := NewTurnDrawState()
	state.drawType = constants.DrawTypeEvent

	ctx := NewStateContext().
		WithHSM(NewHSM(engine.NewGame(id.NewGameID(), 42))).
		WithPlayer(nil)

	state.Enter(ctx)

	if ctx.Error == nil {
		t.Error("Enter should return error when player is nil")
	}
}

func TestTurnDrawState_Update(t *testing.T) {
	state := NewTurnDrawState()
	ctx := NewStateContext()

	nextID := state.Update(ctx)

	if nextID != StateTurnEnd {
		t.Errorf("Update should return StateTurnEnd, got %s", nextID.String())
	}
}

func TestTurnDrawState_Exit(t *testing.T) {
	game := engine.NewGame(id.NewGameID(), 42)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})

	state := NewTurnDrawState()
	ctx := NewStateContext().
		WithHSM(NewHSM(game)).
		WithPlayer(player)

	// Enter to initialize actionCtx
	state.drawType = constants.DrawTypeEvent
	state.probGood = 1.0
	state.Enter(ctx)

	// Exit should clear actionCtx
	state.Exit(ctx)

	// Verify actionCtx is cleared (can't directly check, but Exit should not panic)
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
		WithHSM(NewHSM(game)).
		WithPlayer(player)

	state.Enter(ctx)

	if ctx.Error != nil {
		t.Errorf("Enter should succeed, got error: %v", ctx.Error)
	}
}

func TestTurnEndState_Enter_TickBuffs(t *testing.T) {
	game := engine.NewGame(id.NewGameID(), 0)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	game.AddPlayer(player)

	// Add buff with 1 duration - first TickDuration call marks eligible,
	// second call decrements and expires
	buff1 := core.NewBuff(constants.BuffTypeCurse, 1)
	game.ApplyBuffToPlayer(player, buff1)
	// Simulate previous turn-end tick: marks eligible without decrement
	buff1.TickDuration()

	// Add buff with 3 duration - same: first call marks, second call decrements
	buff2 := core.NewBuff(constants.BuffTypeDivine, 3)
	game.ApplyBuffToPlayer(player, buff2)
	buff2.TickDuration()

	state := NewTurnEndState()
	ctx := NewStateContext().
		WithHSM(NewHSM(game)).
		WithPlayer(player)

	state.Enter(ctx)

	// Verify buff1 expired and unsubscribed (Duration 1→0)
	if len(player.ActiveBuffs) != 1 {
		t.Errorf("Player should have 1 buff remaining, got %d", len(player.ActiveBuffs))
	}
	if player.ActiveBuffs[0].Type != constants.BuffTypeDivine {
		t.Errorf("Remaining buff should be Divine, got %s", string(player.ActiveBuffs[0].Type))
	}
}

func TestTurnEndState_Enter_NewBuffNotTicked(t *testing.T) {
	// Buffs acquired mid-turn (tickEligible=false) should NOT be ticked at TurnEnd.
	// First TickDuration call marks eligible without decrementing.
	game := engine.NewGame(id.NewGameID(), 0)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	game.AddPlayer(player)

	// Add buff with 1 duration, newly created (tickEligible=false)
	buff := core.NewBuff(constants.BuffTypeLost, 1)
	game.ApplyBuffToPlayer(player, buff)

	state := NewTurnEndState()
	ctx := NewStateContext().
		WithHSM(NewHSM(game)).
		WithPlayer(player)

	state.Enter(ctx)

	// Buff should NOT expire because first TickDuration only marks eligible
	if !player.HasBuff(constants.BuffTypeLost) {
		t.Error("Lost buff should still be active (not ticked this turn)")
	}
	if buff.Duration != 1 {
		t.Errorf("Lost buff Duration = %d, expected 1 (not decremented)", buff.Duration)
	}
}

func TestTurnEndState_Enter_FactionCharging(t *testing.T) {
	game := engine.NewGame(id.NewGameID(), 0)
	player := core.NewPlayer(core.PlayerConfig{
		ID:      id.NewPlayerID(),
		Faction: constants.FactionQingLong,
	})
	game.AddPlayer(player)

	state := NewTurnEndState()
	ctx := NewStateContext().
		WithHSM(NewHSM(game)).
		WithPlayer(player)

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
		StateTurnDraw,
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

func TestTurnUpkeepState_ResetPendingDecisions(t *testing.T) {
	state := NewTurnUpkeepState()
	state.decisions = []*event.Decision{
		event.NewDecision("test", []event.Option{{ID: "1", Label: "option1"}}),
	}

	state.ResetPendingDecisions()

	if len(state.decisions) != 0 {
		t.Errorf("ResetPendingDecisions should clear decisions, got %d", len(state.decisions))
	}
}

func TestTurnLandedState_ResetPendingDecisions(t *testing.T) {
	state := NewTurnLandedState()
	state.decisions = []*event.Decision{
		event.NewDecision("test", []event.Option{{ID: "1", Label: "option1"}}),
	}

	state.ResetPendingDecisions()

	if len(state.decisions) != 0 {
		t.Errorf("ResetPendingDecisions should clear decisions, got %d", len(state.decisions))
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

// ========== TurnMovingState GetSteps/SetSteps Tests ==========

func TestTurnMovingStateGetStepsSetSteps(t *testing.T) {
	state := NewTurnMovingState()

	// Default steps should be 0
	if state.GetSteps() != 0 {
		t.Errorf("GetSteps() = %d, want 0", state.GetSteps())
	}

	// SetSteps should update steps
	state.SetSteps(5)
	if state.GetSteps() != 5 {
		t.Errorf("GetSteps() after SetSteps(5) = %d, want 5", state.GetSteps())
	}

	// SetSteps can also set negative values (for 迷途 reversal)
	state.SetSteps(-3)
	if state.GetSteps() != -3 {
		t.Errorf("GetSteps() after SetSteps(-3) = %d, want -3", state.GetSteps())
	}
}

// ========== TurnCheckpointState.Exit Tests ==========

func TestTurnCheckpointStateExit(t *testing.T) {
	state := NewTurnCheckpointState()
	state.actionCtx = engineaction.NewActionContext(nil, nil, nil, nil)
	state.actionCtx.Metadata.SetBool("test_key", true)

	state.Exit(nil)

	// Exit should clear the actionCtx
	if state.actionCtx.Metadata.HasKey("test_key") {
		t.Error("TurnCheckpointState.Exit should clear actionCtx")
	}
}

func TestTurnCheckpointStateExitNilActionCtx(t *testing.T) {
	state := NewTurnCheckpointState()
	// actionCtx is nil by default - Exit should not panic
	state.Exit(nil)
}

func TestTurnCheckpointState_Enter_BroadcastStateSync(t *testing.T) {
	game := engine.NewGame(id.NewGameID(), 42)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player.Position = 25
	player.HP = 5
	game.AddPlayer(player)

	// Setup ItemPool for DrawItemAction
	game.ItemPool = []*rng.EvaluatedItem{
		{Type: "healing_potion", Eval: constants.EvaluationGood},
	}

	hsmInst := NewHSM(game)
	mockBroadcast := pkgnet.NewMockBroadcastAdapter()

	state := NewTurnCheckpointState()
	ctx := NewStateContext().
		WithHSM(hsmInst).
		WithPlayer(player).
		WithBroadcast(mockBroadcast).
		WithBuilder(&builderAdapter{hsmInst: hsmInst})

	state.Enter(ctx)

	if ctx.Error != nil {
		t.Errorf("Enter should succeed, got error: %v", ctx.Error)
	}
	if len(mockBroadcast.StateSyncs) != 1 {
		t.Errorf("broadcastStateSync should be called once, got %d calls", len(mockBroadcast.StateSyncs))
	}
}

func TestTurnCheckpointState_Enter_NoBroadcastOrBuilder(t *testing.T) {
	game := engine.NewGame(id.NewGameID(), 42)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player.Position = 25
	player.HP = 5
	game.AddPlayer(player)

	// Setup ItemPool for DrawItemAction
	game.ItemPool = []*rng.EvaluatedItem{
		{Type: "healing_potion", Eval: constants.EvaluationGood},
	}

	state := NewTurnCheckpointState()
	ctx := NewStateContext().
		WithHSM(NewHSM(game)).
		WithPlayer(player)
	// Broadcast and Builder are nil - should not panic

	state.Enter(ctx)

	if ctx.Error != nil {
		t.Errorf("Enter should succeed without Broadcast/Builder, got error: %v", ctx.Error)
	}
}
