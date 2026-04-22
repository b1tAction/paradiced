package hsm

import (
	"fmt"
	"time"

	"github.com/b1tAction/paradiced/internal/core"
	"github.com/b1tAction/paradiced/internal/engine"
	engineaction "github.com/b1tAction/paradiced/internal/engine/action"
	"github.com/b1tAction/paradiced/internal/event"
	"github.com/b1tAction/paradiced/internal/gamemap"
	"github.com/b1tAction/paradiced/pkg/constants"
	"github.com/b1tAction/paradiced/pkg/errors"
	"github.com/b1tAction/paradiced/pkg/rng"
)

// ========== Turn States (Layer 2) ==========

// BaseTurnState provides common functionality for turn layer states.
type BaseTurnState struct {
	id StateID
}

// ID returns the state identifier.
func (s *BaseTurnState) ID() StateID {
	return s.id
}

// CanTransitionTo defines valid transition rules for turn states.
// Turn states follow a linear flow: Upkeep -> MainAction -> Moving -> Landed -> Event -> End
func (s *BaseTurnState) CanTransitionTo(target StateID) bool {
	// Turn states can transition to any turn state (for jumps like SkipTurn)
	// or to WaitDecision (interrupt)
	return target.IsTurnState() || target == StateWaitDecision
}

// ========== TurnUpkeepState ==========

// TurnUpkeepState handles turn preparation and PhaseBeforeTurn trigger.
// Checks SkipTurn/IsDead, triggers BeforeTurn phase effects.
type TurnUpkeepState struct {
	BaseTurnState
	skipTurn  bool
	isDead    bool
	decisions []*event.Decision
	actionCtx *engineaction.ActionContext
}

// NewTurnUpkeepState creates a new TurnUpkeep state.
func NewTurnUpkeepState() *TurnUpkeepState {
	return &TurnUpkeepState{
		BaseTurnState: BaseTurnState{id: StateTurnUpkeep},
		skipTurn:      false,
		isDead:        false,
		decisions:     make([]*event.Decision, 0),
	}
}

func (s *TurnUpkeepState) Enter(ctx *StateContext) {
	player := ctx.Player
	if player == nil {
		ctx.Error = errors.WrapHSMError(
			errors.NewInternalError("HSM", "TurnUpkeep", nil),
			"TurnUpkeep", 2, "Enter", "player is nil")
		return
	}

	// Start turn log segment
	game := ctx.GetGame()
	if game != nil && game.Log != nil {
		game.Log.StartTurn(ctx.GetRound(), ctx.GetTurn(), player.ID.UUID())
	}

	// Create ActionContext for executing Actions
	mapEngine := ctx.GetMapEngine()
	s.actionCtx = newActionContextWithPoolsNoPlayer(
		game,
		game.Bus,
		mapEngine,
		game.Draw,
	)

	// Step 1: Check IsDead -> Respawn at checkpoint using RespawnAction
	if player.IsDead {
		if mapEngine != nil {
			checkpoint := mapEngine.GetLastCheckpoint(player.Position)
			respawnAction := engineaction.NewRespawnAction(player, checkpoint, "DeathRespawn")
			if err := s.actionCtx.ExecuteAction(respawnAction); err != nil {
				ctx.Error = errors.WrapHSMError(
					err, "TurnUpkeep", 2, "Enter", "respawn action failed")
				return
			}
		}
		s.isDead = true
	}

	// Step 2: Check SkipTurn flag
	if player.SkipTurn {
		player.SkipTurn = false // Clear flag
		s.skipTurn = true
		ctx.SetSkipTurn(true)
		// Broadcast StateSync even when skipping
		s.broadcastStateSync(ctx)
		return // Skip all BeforeTurn effects
	}

	// Step 3: Trigger PhaseBeforeTurn
	// HSM publishes this phase - Buff handlers respond and may return Actions
	triggerCtx := event.NewContext(player)
	triggerCtx.Set("action_context", s.actionCtx)
	triggerCtx.Set("current_player", player)

	// Publish PhaseBeforeTurn to trigger Buff effects
	s.decisions = ctx.GetBus().Publish(constants.PhaseBeforeTurn, player.ID.UUID(), triggerCtx)

	// Check for handler errors
	if triggerCtx.HasError() {
		ctx.Error = errors.WrapHSMError(
			triggerCtx.FirstError(), "TurnUpkeep", 2, "Enter", "PhaseBeforeTurn handler failed")
		return
	}

	// Step 4: Bridge and execute derived Actions from handlers
	if err := runDerived(triggerCtx); err != nil {
		ctx.Error = errors.WrapHSMError(
			err, "TurnUpkeep", 2, "Enter", "derived action execution failed")
		return
	}

	// Step 5: Check if any decisions need user input
	if len(s.decisions) > 0 {
		// Will be handled in Update - push WaitDecision if needed
		ctx.Decisions = s.decisions
		ctx.Set(KeyPendingCtx, triggerCtx)
	}

	// Broadcast StateSync after all BeforeTurn effects
	s.broadcastStateSync(ctx)
}

// broadcastStateSync broadcasts current game state to clients.
func (s *TurnUpkeepState) broadcastStateSync(ctx *StateContext) {
	game := ctx.GetGame()
	if ctx.Broadcast == nil || game == nil {
		return
	}
	// Use Builder if available
	if ctx.Builder != nil {
		stateSync := ctx.Builder.BuildStateSync()
		ctx.Broadcast.BroadcastStateSync(stateSync)
	}
}

func (s *TurnUpkeepState) Update(ctx *StateContext) StateID {
	// Check if SkipTurn was set -> jump to TurnEnd
	if s.skipTurn {
		return StateTurnEnd
	}

	// Check if decisions need processing
	if len(s.decisions) > 0 {
		// Decisions should be handled by HSM.PushInterrupt
		// Return StateNone to wait for decision processing
		return StateNone
	}

	// Normal flow -> MainAction
	return StateMainAction
}

func (s *TurnUpkeepState) Exit(ctx *StateContext) {
	s.skipTurn = false
	s.isDead = false
	s.decisions = make([]*event.Decision, 0)
	if s.actionCtx != nil {
		s.actionCtx.Clear()
	}
}

// ========== MainActionState ==========

// MainActionState waits for player's main action (RollDice/UseItem/UseSkill).
// This is the primary user input phase.
type MainActionState struct {
	BaseTurnState
	waitingForAction bool
	diceRolled       bool
	diceSteps        int
	timeout          time.Duration
	startTime        time.Time
	actionCtx        *engineaction.ActionContext
}

// NewMainActionState creates a new MainAction state.
func NewMainActionState(timeout time.Duration) *MainActionState {
	return &MainActionState{
		BaseTurnState:    BaseTurnState{id: StateMainAction},
		waitingForAction: true,
		diceRolled:       false,
		timeout:          timeout,
	}
}

// NewMainActionStateDefault creates a new MainAction state with default timeout (45s).
// Deprecated: Use NewMainActionState with explicit timeout.
func NewMainActionStateDefault() *MainActionState {
	return NewMainActionState(45 * time.Second)
}

func (s *MainActionState) Enter(ctx *StateContext) {
	player := ctx.Player
	if player == nil {
		ctx.Error = errors.WrapHSMError(
			errors.NewInternalError("HSM", "MainAction", nil),
			"MainAction", 2, "Enter", "player is nil")
		return
	}

	// Initialize action context
	game := ctx.GetGame()
	mapEngine := ctx.GetMapEngine()
	s.actionCtx = newActionContextWithPoolsNoPlayer(
		game,
		game.Bus,
		mapEngine,
		game.Draw,
	)

	s.startTime = time.Now()
	s.waitingForAction = true
	s.diceRolled = false

	// Broadcast StateSync
	s.broadcastStateSync(ctx)

	// Send Available to current player
	s.sendAvailable(ctx)

	// Build available actions:
	// 1. PhaseItemUsed items (active items)
	// 2. Faction skills (QingLong/XuanWu charge check)
	// Note: Actual item/skill execution is handled by OnUseItem/OnUseSkill
}

// broadcastStateSync broadcasts current game state.
func (s *MainActionState) broadcastStateSync(ctx *StateContext) {
	game := ctx.GetGame()
	if ctx.Broadcast == nil || game == nil {
		return
	}
	// Use Builder if available
	if ctx.Builder != nil {
		stateSync := ctx.Builder.BuildStateSync()
		ctx.Broadcast.BroadcastStateSync(stateSync)
	}
}

// sendAvailable sends available actions to current player.
func (s *MainActionState) sendAvailable(ctx *StateContext) {
	game := ctx.GetGame()
	if ctx.Broadcast == nil || game == nil || ctx.Player == nil {
		return
	}
	// Use Builder if available
	if ctx.Builder != nil {
		// Set dice type from context
		diceType := ctx.GetDiceType(ctx.Player.ID.UUID())
		ctx.Builder.SetDiceType(diceType.String())
		available := ctx.Builder.BuildAvailable()
		ctx.Broadcast.SendAvailable(ctx.Player.ID.UUID(), available)
	}
}

func (s *MainActionState) Update(ctx *StateContext) StateID {
	// Check if dice was rolled
	if s.diceRolled {
		fmt.Printf("[hsm] MainActionState.Update: diceRolled=%v, diceSteps=%d\n", s.diceRolled, s.diceSteps)
		ctx.SetInt(KeyDiceSteps, s.diceSteps)
		return StateTurnMoving
	}

	// Check timeout
	if time.Since(s.startTime) > s.timeout {
		// Auto roll dice (default action)
		steps := s.defaultDiceRoll(ctx)
		s.OnRollDice(ctx, steps)
		ctx.SetInt(KeyDiceSteps, steps)
		return StateTurnMoving
	}

	// Continue waiting
	return StateNone
}

func (s *MainActionState) Exit(ctx *StateContext) {
	s.waitingForAction = false
	s.diceRolled = false
	if s.actionCtx != nil {
		s.actionCtx.Clear()
	}
}

// OnRollDice handles dice roll input.
func (s *MainActionState) OnRollDice(ctx *StateContext, steps int) {
	s.diceSteps = steps
	s.diceRolled = true
	s.waitingForAction = false
}

// OnUseItem handles item usage input.
func (s *MainActionState) OnUseItem(ctx *StateContext, itemID string) {
	// Publish PhaseItemUsed to trigger item handler
	player := ctx.Player
	triggerCtx := event.NewContext(player)
	triggerCtx.Set("item_id", itemID)
	triggerCtx.Set("action_context", s.actionCtx)

	ctx.GetBus().Publish(constants.PhaseItemUsed, player.ID.UUID(), triggerCtx)

	// Check for handler errors
	if triggerCtx.HasError() {
		ctx.Error = errors.WrapHSMError(
			triggerCtx.FirstError(), "MainAction", 2, "OnUseItem", "PhaseItemUsed handler failed")
		return
	}

	// Bridge and process derived actions
	if err := runDerived(triggerCtx); err != nil {
		ctx.Error = errors.WrapHSMError(
			err, "MainAction", 2, "OnUseItem", "derived action execution failed")
		return
	}
}

// defaultDiceRoll returns default dice steps based on dice type.
// Default values for timeout auto-roll scenarios.
func (s *MainActionState) defaultDiceRoll(ctx *StateContext) int {
	// Prefer current turn player from HSM; fallback to context player.
	var turnPlayer *core.Player
	if ctx.HSM != nil {
		turnPlayer = ctx.HSM.GetTurnPlayer()
	}
	if turnPlayer == nil {
		turnPlayer = ctx.Player
	}
	if turnPlayer == nil {
		return 2 // Default to wood dice steps
	}

	// Get dice type from context (assigned in RoundPrep)
	diceType := ctx.GetDiceType(turnPlayer.ID.UUID())
	switch diceType {
	case rng.DiceTypeGold:
		return 6 // Gold dice: weighted toward high numbers
	case rng.DiceTypeSilver:
		return 4 // Silver dice: moderate weights
	case rng.DiceTypeCopper:
		return 3 // Copper dice: slight high bias
	default:
		return 2 // Wood dice: uniform distribution
	}
}

// ========== TurnMovingState ==========

// TurnMovingState handles movement calculation and path processing.
// HSM layer does path pre-scanning via CalculatePath, then executes pure MoveAction.
// Steps field stores dice steps, modified by 迷途 handler during PhasePreMove.
// Data flow: State → ActionContext.Metadata → MoveAction.Execute()
type TurnMovingState struct {
	BaseTurnState
	Steps          int    // Movement steps (from dice, may be modified by 迷途)
	fellDown       bool   // Player fell from Fragile cell
	reachedEnd     bool   // Player reached Boss cell (end of map)
	hasCheckpoint  bool   // CheckPoint detected in path (Enter→Update transition flag)
	checkpointPos  int    // CheckPoint position (Enter→Update transition data)
	remainingSteps int    // Remaining steps after CheckPoint (Enter→Update transition data)
	pathResult     PathResultData
	actionCtx      *engineaction.ActionContext
}

// PathResultData stores path calculation results.
type PathResultData struct {
	StartIndex     int
	TargetIndex    int
	Path           []int
	BrokenFragiles []int
}

// NewTurnMovingState creates a new TurnMoving state.
func NewTurnMovingState() *TurnMovingState {
	return &TurnMovingState{
		BaseTurnState: BaseTurnState{id: StateTurnMoving},
	}
}

// GetSteps returns current movement steps (implements engine.StepsModifier).
func (s *TurnMovingState) GetSteps() int {
	return s.Steps
}

// SetSteps sets movement steps (implements engine.StepsModifier).
// Used by 迷途 handler to reverse movement direction.
func (s *TurnMovingState) SetSteps(steps int) {
	s.Steps = steps
}

func (s *TurnMovingState) Enter(ctx *StateContext) {
	player := ctx.Player
	if player == nil {
		ctx.Error = errors.WrapHSMError(
			errors.NewInternalError("HSM", "TurnMoving", nil),
			"TurnMoving", 2, "Enter", "player is nil")
		return
	}

	// Get dice steps (from dice or remaining steps from TurnCheckpoint re-entry)
	s.Steps = ctx.GetDiceSteps()
	fmt.Printf("[hsm] TurnMovingState.Enter: dice_steps=%d, player_position=%d\n", s.Steps, player.Position)
	if s.Steps == 0 {
		// Zero steps, skip movement entirely
		fmt.Printf("[hsm] TurnMovingState.Enter: zero steps, going to TurnLanded\n")
		return
	}

	// Create ActionContext
	game := ctx.GetGame()
	mapEngine := ctx.GetMapEngine()
	s.actionCtx = newActionContextWithPools(
		game,
		game.Bus,
		mapEngine,
		game.Draw,
		player,
	)

	// Step 1: HSM publishes PhasePreMove (迷途 handler modifies s.Steps directly)
	triggerCtx := event.NewContext(player)
	triggerCtx.Set("action_context", s.actionCtx)
	triggerCtx.Set("current_state", s)
	triggerCtx.Set("current_player", player)

	decisions := ctx.GetBus().Publish(constants.PhasePreMove, player.ID.UUID(), triggerCtx)

	// Check for handler errors
	if triggerCtx.HasError() {
		ctx.Error = errors.WrapHSMError(
			triggerCtx.FirstError(), "TurnMoving", 2, "Enter", "PhasePreMove handler failed")
		return
	}

	// Step 2: Path pre-scanning via CalculatePath (using modified s.Steps)
	startPos := player.Position
	pathResult, err := mapEngine.CalculatePath(startPos, s.Steps)
	if err != nil {
		ctx.Error = errors.WrapHSMError(
			err, "TurnMoving", 2, "Enter", "path calculation failed")
		return
	}

	// Step 3: FellDown handling (HSM layer)
	if pathResult.FellDown {
		s.fellDown = true
		// Write path data to actionCtx.Metadata for MoveAction to read
		s.actionCtx.SetInt("target_pos", pathResult.TargetIndex)
		s.actionCtx.Set("path", pathResult.Path)
		moveAction := engineaction.NewMoveAction(player, pathResult.TargetIndex-startPos, "DiceRollFellDown")
		s.actionCtx.ExecuteAction(moveAction)
		fellDownAction := engineaction.NewFellDownAction(player, pathResult.TargetIndex, 1, "FragileCell")
		s.actionCtx.ExecuteAction(fellDownAction)
		// FellDown -> TurnEnd, no further processing
		return
	}

	// Step 4: CheckPoint check (only when Steps > 0, i.e. forward movement)
	if s.Steps > 0 {
		checkpointPos := findFirstCheckpointInPath(pathResult.Path, mapEngine)
		if checkpointPos != -1 {
			// Split movement at CheckPoint
			remainingSteps := pathResult.OriginalTarget - checkpointPos
			firstSegSteps := checkpointPos - startPos

			// Calculate first segment path
			firstSegPathResult, firstSegErr := mapEngine.CalculatePath(startPos, firstSegSteps)
			if firstSegErr != nil {
				ctx.Error = errors.WrapHSMError(
					firstSegErr, "TurnMoving", 2, "Enter", "first segment path calculation failed")
				return
			}

			// Write first segment path data to actionCtx.Metadata
			s.actionCtx.SetInt("target_pos", checkpointPos)
			s.actionCtx.Set("path", firstSegPathResult.Path)

			moveAction := engineaction.NewMoveAction(player, firstSegSteps, "DiceRollCheckpoint")
			s.actionCtx.ExecuteAction(moveAction)

			s.hasCheckpoint = true
			s.checkpointPos = checkpointPos
			s.remainingSteps = remainingSteps
			// Store remaining steps for TurnCheckpoint → TurnMoving re-entry
			ctx.SetInt(KeyDiceSteps, remainingSteps)
			return
		}
	}

	// Normal full movement (no CheckPoint, or reverse movement)
	s.actionCtx.SetInt("target_pos", pathResult.TargetIndex)
	s.actionCtx.Set("path", pathResult.Path)

	moveAction := engineaction.NewMoveAction(player, s.Steps, "DiceRoll")
	s.actionCtx.ExecuteAction(moveAction)

	s.reachedEnd = pathResult.ReachedEnd
	if s.reachedEnd {
		ctx.SetReachedEnd(true)
	}

	s.pathResult = PathResultData{
		StartIndex:     startPos,
		TargetIndex:    pathResult.TargetIndex,
		Path:           pathResult.Path,
		BrokenFragiles: pathResult.BrokenFragiles,
	}

	// Handle Fog activation
	for _, pos := range s.pathResult.Path {
		if mapEngine != nil {
			cell, _ := mapEngine.GetCell(pos)
			if cell != nil && cell.CellType == constants.CellTypeFog {
				mapEngine.ActivateFog(pos)
			}
		}
	}

	// Store decisions if any
	if len(decisions) > 0 {
		ctx.Decisions = decisions
		ctx.Set(KeyPendingCtx, triggerCtx)
	}
}

func (s *TurnMovingState) Update(ctx *StateContext) StateID {
	// FellDown -> TurnEnd (FellDownAction already executed in Enter)
	if s.fellDown {
		ctx.SetFellDown(true)
		return StateTurnEnd
	}

	// CheckPoint detected -> TurnCheckpoint (then re-enter TurnMoving with remaining steps)
	if s.hasCheckpoint {
		return StateTurnCheckpoint
	}

	// Normal flow -> TurnLanded
	return StateTurnLanded
}

func (s *TurnMovingState) Exit(ctx *StateContext) {
	s.Steps = 0
	s.fellDown = false
	s.reachedEnd = false
	s.hasCheckpoint = false
	s.remainingSteps = 0
	s.checkpointPos = 0
	s.pathResult = PathResultData{}
	if s.actionCtx != nil {
		s.actionCtx.Clear()
	}
}

// ========== TurnCheckpointState ==========

// TurnCheckpointState handles CheckPoint landing effects (DrawItem etc.).
// After processing, re-enters TurnMoving with remaining steps as dice_steps.
type TurnCheckpointState struct {
	BaseTurnState
	actionCtx *engineaction.ActionContext
}

// NewTurnCheckpointState creates a new TurnCheckpoint state.
func NewTurnCheckpointState() *TurnCheckpointState {
	return &TurnCheckpointState{
		BaseTurnState: BaseTurnState{id: StateTurnCheckpoint},
	}
}

func (s *TurnCheckpointState) Enter(ctx *StateContext) {
	player := ctx.Player
	if player == nil {
		ctx.Error = errors.WrapHSMError(
			errors.NewInternalError("HSM", "TurnCheckpoint", nil),
			"TurnCheckpoint", 2, "Enter", "player is nil")
		return
	}

	// Create ActionContext for DrawItem
	game := ctx.GetGame()
	mapEngine := ctx.GetMapEngine()
	s.actionCtx = newActionContextWithPools(
		game,
		game.Bus,
		mapEngine,
		game.Draw,
		player,
	)

	// Execute DrawItemAction at CheckPoint (auto draw, no interception)
	drawItemAction := engineaction.NewDrawItemAction(player, "CheckpointTreasure")
	s.actionCtx.ExecuteAction(drawItemAction)

	// No TurnSync broadcast here, LogEntry recorded to GameLog
	// TurnSync will be broadcast at turn end (TurnEndState)
}

func (s *TurnCheckpointState) Update(ctx *StateContext) StateID {
	// Re-enter TurnMoving with remaining steps
	// remaining steps were already written to ctx(KeyDiceSteps) by TurnMoving.Update()
	return StateTurnMoving
}

func (s *TurnCheckpointState) Exit(ctx *StateContext) {
	if s.actionCtx != nil {
		s.actionCtx.Clear()
	}
}

// ========== Helper Functions ==========

// findFirstCheckpointInPath scans the path for the first CheckPoint cell.
// Skips the first element (start position) since player is already there.
// Returns position index or -1 if none found.
func findFirstCheckpointInPath(path []int, mapEngine *gamemap.MapEngine) int {
	for i, pos := range path {
		if i == 0 {
			continue // Skip start position (player already at this cell)
		}
		cell, err := mapEngine.GetCell(pos)
		if err == nil && cell != nil && cell.CellType == constants.CellTypeCheckpoint {
			return pos
		}
	}
	return -1
}

// newActionContextWithPools creates an ActionContext with pool data from Game.
func newActionContextWithPools(game *engine.Game, bus *event.EventBus, mapEngine *gamemap.MapEngine, drawEngine *rng.DrawEngine, player *core.Player) *engineaction.ActionContext {
	ctx := engineaction.NewActionContextWithPlayer(game, bus, mapEngine, drawEngine, player)
	ctx.SetPools(game.EventPool, game.ItemPool)
	return ctx
}

// newActionContextWithPoolsNoPlayer creates an ActionContext with pool data from Game (no current player).
func newActionContextWithPoolsNoPlayer(game *engine.Game, bus *event.EventBus, mapEngine *gamemap.MapEngine, drawEngine *rng.DrawEngine) *engineaction.ActionContext {
	ctx := engineaction.NewActionContext(game, bus, mapEngine, drawEngine)
	ctx.SetPools(game.EventPool, game.ItemPool)
	return ctx
}

// runEventEffect looks up the EventRegistry handler for the drawn event type
// and bridges DerivedActions from the handler to ActionContext.
func runEventEffect(drawnType constants.EventType, player *core.Player, actionCtx *engineaction.ActionContext) error {
	if drawnType == constants.EventTypeNone || !drawnType.IsValid() {
		return nil // No event drawn or invalid type
	}

	// Look up EventRegistry handler
	handlerConfig := engine.GetEventHandlerConfig(drawnType)
	if handlerConfig == nil || handlerConfig.Handler == nil {
		return nil // No handler registered for this event type
	}

	// Create event.Context and call handler
	triggerCtx := event.NewContext(player)
	triggerCtx.Set("action_context", actionCtx)
	triggerCtx.Set("current_player", player)

	// Call the event effect handler
	if err := handlerConfig.Handler(constants.PhaseAnyTime, triggerCtx); err != nil {
		return err
	}

	// Bridge DerivedActions to ActionContext
	for _, derived := range triggerCtx.GetDerivedActions() {
		if act, ok := derived.(engineaction.Action); ok {
			actionCtx.PushDerivedAction(act)
		}
	}

	// Process the derived actions
	return actionCtx.ProcessQueue()
}

// ========== TurnLandedState ==========

// TurnLandedState handles landing effects based on cell type behavior matrix.
// PhaseOnLand is triggered first, then cell-type-specific actions:
// - CellTypeEvent: DrawEventAction with bound event ID
// - CellTypeNormal/Fog/Fragile: Draw based on cell.DrawType (Event/Item/None)
// - CellTypeCheckpoint: Already processed in TurnCheckpoint, no action here
// - CellTypeBoss: Already handled in TurnMoving (reachedEnd flag)
type TurnLandedState struct {
	BaseTurnState
	cellType   constants.CellType
	cell       *gamemap.MapCell // Landing cell data (for EventID access)
	decisions  []*event.Decision
	actionCtx  *engineaction.ActionContext
	skipEvent  bool             // Skip TurnDraw (CellTypeCheckpoint/Boss don't need random event)
	// Cell draw configuration
	drawType   constants.DrawType
	probGood   float64
	probNeutral float64
	probBad    float64
}

// NewTurnLandedState creates a new TurnLanded state.
func NewTurnLandedState() *TurnLandedState {
	return &TurnLandedState{
		BaseTurnState: BaseTurnState{id: StateTurnLanded},
		decisions:     make([]*event.Decision, 0),
	}
}

func (s *TurnLandedState) Enter(ctx *StateContext) {
	player := ctx.Player
	if player == nil {
		ctx.Error = errors.WrapHSMError(
			errors.NewInternalError("HSM", "TurnLanded", nil),
			"TurnLanded", 2, "Enter", "player is nil")
		return
	}

	// Create ActionContext
	game := ctx.GetGame()
	mapEngine := ctx.GetMapEngine()
	s.actionCtx = newActionContextWithPoolsNoPlayer(
		game,
		game.Bus,
		mapEngine,
		game.Draw,
	)

	// Get cell data at landing position
	if mapEngine != nil {
		cell, err := mapEngine.GetCell(player.Position)
		if err == nil && cell != nil {
			s.cellType = cell.CellType
			s.cell = cell
			// Capture cell draw configuration
			s.drawType = cell.DrawType
			s.probGood = cell.ProbGood
			s.probNeutral = cell.ProbNeutral
			s.probBad = cell.ProbBad
		}
	}

	// Trigger PhaseOnLand
	triggerCtx := event.NewContext(player)
	triggerCtx.Set("action_context", s.actionCtx)
	triggerCtx.Set("cell_type", s.cellType)
	triggerCtx.Set("position", player.Position)

	s.decisions = ctx.GetBus().Publish(constants.PhaseOnLand, player.ID.UUID(), triggerCtx)

	// Check for handler errors
	if triggerCtx.HasError() {
		ctx.Error = errors.WrapHSMError(
			triggerCtx.FirstError(), "TurnLanded", 2, "Enter", "PhaseOnLand handler failed")
		return
	}

	if len(s.decisions) > 0 {
		ctx.Set(KeyPendingCtx, triggerCtx)
	}

	// Bridge and process derived actions
	if err := runDerived(triggerCtx); err != nil {
		ctx.Error = errors.WrapHSMError(
			err, "TurnLanded", 2, "Enter", "derived action execution failed")
		return
	}

	// Handle cell type behavior matrix
	switch s.cellType {
	case constants.CellTypeEvent:
		// Event cell: trigger bound event (cell.EventID specifies which event type)
		if s.cell != nil && s.cell.EventID != "" {
			drawnType := constants.ParseEventType(s.cell.EventID)
			if drawnType.IsValid() {
				drawAction := engineaction.NewDrawEventAction(player, "CellEvent_"+s.cell.EventID)
				drawAction.DrawnType = drawnType // Set directly from cell binding
				drawAction.DrawnName = engine.GetEventName(drawnType)
				if err := s.actionCtx.ExecuteAction(drawAction); err != nil {
					ctx.Error = errors.WrapHSMError(
						err, "TurnLanded", 2, "Enter", "bound event action failed")
					return
				}
				// Execute the bound event's effect via EventRegistry handler
				if err := runEventEffect(drawnType, player, s.actionCtx); err != nil {
					ctx.Error = errors.WrapHSMError(
						err, "TurnLanded", 2, "Enter", "bound event effect failed")
					return
				}
				// Bound event already executed, skip random draw in TurnDraw
				s.skipEvent = true
			}
		}
	case constants.CellTypeCheckpoint:
		// Checkpoint already processed in TurnCheckpoint state
		// Skip random event draw in TurnDraw
		s.skipEvent = true
	case constants.CellTypeBoss:
		// Boss cell: already handled in TurnMoving (reachedEnd flag)
		// Skip random event draw in TurnDraw
		s.skipEvent = true
	case constants.CellTypeNormal, constants.CellTypeFog, constants.CellTypeFragile:
		// Random DrawEvent will be handled in TurnDraw state
		// No special action needed here
	}
}

func (s *TurnLandedState) Update(ctx *StateContext) StateID {
	// Check if decisions need processing
	if len(s.decisions) > 0 {
		ctx.Decisions = s.decisions
		return StateNone // Wait for decision
	}

	// Skip TurnDraw for CheckPoint/Boss cells or if DrawType is None (or not set)
	if s.skipEvent || s.drawType == constants.DrawTypeNone || !s.drawType.IsValid() {
		return StateTurnEnd
	}

	// Normal flow -> TurnDraw
	// Pass cell draw configuration to TurnDrawState
	if ctx.HSM != nil {
		if drawState, ok := ctx.HSM.states[StateTurnDraw].(*TurnDrawState); ok {
			drawState.drawType = s.drawType
			drawState.probGood = s.probGood
			drawState.probNeutral = s.probNeutral
			drawState.probBad = s.probBad
		}
	}

	return StateTurnDraw
}

func (s *TurnLandedState) Exit(ctx *StateContext) {
	s.cellType = constants.CellTypeNormal
	s.cell = nil
	s.skipEvent = false
	s.decisions = make([]*event.Decision, 0)
	if s.actionCtx != nil {
		s.actionCtx.Clear()
	}
}

// ========== TurnDrawState ==========

// TurnDrawState handles drawing events or items based on cell configuration.
// Uses cell's DrawType to determine what to draw (Event/Item/None).
// Uses cell's ProbGood/ProbNeutral/ProbBad for weighted pool selection.
type TurnDrawState struct {
	BaseTurnState
	drawType    constants.DrawType
	probGood    float64
	probNeutral float64
	probBad     float64
	actionCtx   *engineaction.ActionContext
}

// NewTurnDrawState creates a new TurnDraw state.
func NewTurnDrawState() *TurnDrawState {
	return &TurnDrawState{
		BaseTurnState: BaseTurnState{id: StateTurnDraw},
	}
}

func (s *TurnDrawState) Enter(ctx *StateContext) {
	player := ctx.Player
	if player == nil {
		ctx.Error = errors.WrapHSMError(
			errors.NewInternalError("HSM", "TurnDraw", nil),
			"TurnDraw", 2, "Enter", "player is nil")
		return
	}

	// Skip if DrawType is None
	if s.drawType == constants.DrawTypeNone {
		return
	}

	// Create ActionContext
	game := ctx.GetGame()
	mapEngine := ctx.GetMapEngine()
	s.actionCtx = newActionContextWithPoolsNoPlayer(
		game,
		game.Bus,
		mapEngine,
		game.Draw,
	)

	// Set cell draw probabilities
	s.actionCtx.SetCellDraw(s.probGood, s.probNeutral, s.probBad)

	// Execute draw based on DrawType
	switch s.drawType {
	case constants.DrawTypeEvent:
		// Draw event (PreTriggerPhase = PhasePreEvent for immunity checks)
		drawAction := engineaction.NewDrawEventAction(player, "CellDraw")
		if err := s.actionCtx.ExecuteAction(drawAction); err != nil {
			ctx.Error = errors.WrapHSMError(
				err, "TurnDraw", 2, "Enter", "draw event action failed")
			return
		}
		// Execute the drawn event's effect via EventRegistry handler
		if drawAction.DrawnType.IsValid() {
			if err := runEventEffect(drawAction.DrawnType, player, s.actionCtx); err != nil {
				ctx.Error = errors.WrapHSMError(
					err, "TurnDraw", 2, "Enter", "event effect execution failed")
				return
			}
		}
	case constants.DrawTypeItem:
		// Draw item (no interception)
		drawAction := engineaction.NewDrawItemAction(player, "CellDraw")
		if err := s.actionCtx.ExecuteAction(drawAction); err != nil {
			ctx.Error = errors.WrapHSMError(
				err, "TurnDraw", 2, "Enter", "draw item action failed")
			return
		}
	}
}

func (s *TurnDrawState) Update(ctx *StateContext) StateID {
	// Normal flow -> TurnEnd
	return StateTurnEnd
}

func (s *TurnDrawState) Exit(ctx *StateContext) {
	if s.actionCtx != nil {
		s.actionCtx.Clear()
	}
}

// ========== TurnEndState ==========

// TurnEndState handles turn cleanup and PhaseAfterTurn trigger.
// Ticks Buff durations, checks death, handles faction charging.
type TurnEndState struct {
	BaseTurnState
	actionCtx *engineaction.ActionContext
}

// NewTurnEndState creates a new TurnEnd state.
func NewTurnEndState() *TurnEndState {
	return &TurnEndState{
		BaseTurnState: BaseTurnState{id: StateTurnEnd},
	}
}

func (s *TurnEndState) Enter(ctx *StateContext) {
	player := ctx.Player
	if player == nil {
		ctx.Error = errors.WrapHSMError(
			errors.NewInternalError("HSM", "TurnEnd", nil),
			"TurnEnd", 2, "Enter", "player is nil")
		return
	}

	// Create ActionContext
	game := ctx.GetGame()
	mapEngine := ctx.GetMapEngine()
	s.actionCtx = newActionContextWithPoolsNoPlayer(
		game,
		game.Bus,
		mapEngine,
		game.Draw,
	)

	// Trigger PhaseAfterTurn
	triggerCtx := event.NewContext(player)
	triggerCtx.Set("action_context", s.actionCtx)

	decisions := ctx.GetBus().Publish(constants.PhaseAfterTurn, player.ID.UUID(), triggerCtx)

	// Check for handler errors
	if triggerCtx.HasError() {
		ctx.Error = errors.WrapHSMError(
			triggerCtx.FirstError(), "TurnEnd", 2, "Enter", "PhaseAfterTurn handler failed")
		return
	}

	// Bridge and process derived actions (甘霖/腐化 effects)
	if err := runDerived(triggerCtx); err != nil {
		ctx.Error = errors.WrapHSMError(
			err, "TurnEnd", 2, "Enter", "derived action execution failed")
		return
	}

	// Tick Buff durations
	expiredBuffs := player.TickBuffs()

	// Handle expired buffs (unsubscribe from EventBus)
	for _, expired := range expiredBuffs {
		game.UnsubscribeBuff(expired)
	}

	// Check IsDead after AfterTurn effects - use RespawnAction
	if player.IsDead {
		if mapEngine != nil {
			checkpoint := mapEngine.GetLastCheckpoint(player.Position)
			respawnAction := engineaction.NewRespawnAction(player, checkpoint, "AfterTurnRespawn")
			if err := s.actionCtx.ExecuteAction(respawnAction); err != nil {
				ctx.Error = errors.WrapHSMError(
					err, "TurnEnd", 2, "Enter", "respawn action failed")
				return
			}
		}
	}

	// Faction charging logic
	s.handleFactionCharging(ctx, player)

	// Handle decisions if any
	if len(decisions) > 0 {
		ctx.Decisions = decisions
		ctx.Set(KeyPendingCtx, triggerCtx)
	}

	// Broadcast TurnSync (all actions from this turn) BEFORE EndTurn,
	// because EndTurn sets GameLog.current to nil, making GetCurrentTurnEntries return nil.
	s.broadcastTurnSync(ctx)

	// End turn log segment
	if game != nil && game.Log != nil {
		game.Log.EndTurn()
	}

	// Broadcast StateSync (final state after turn)
	s.broadcastStateSync(ctx)
}

// broadcastTurnSync broadcasts all actions from current turn.
func (s *TurnEndState) broadcastTurnSync(ctx *StateContext) {
	game := ctx.GetGame()
	if ctx.Broadcast == nil || game == nil {
		return
	}
	// Use Builder if available
	if ctx.Builder != nil {
		turnSync := ctx.Builder.BuildTurnSync()
		ctx.Broadcast.BroadcastTurnSync(turnSync)
	}
}

// broadcastStateSync broadcasts final game state after turn.
func (s *TurnEndState) broadcastStateSync(ctx *StateContext) {
	game := ctx.GetGame()
	if ctx.Broadcast == nil || game == nil {
		return
	}
	// Use Builder if available
	if ctx.Builder != nil {
		stateSync := ctx.Builder.BuildStateSync()
		ctx.Broadcast.BroadcastStateSync(stateSync)
	}
}

func (s *TurnEndState) Update(ctx *StateContext) StateID {
	// Return to parent state (TurnLoop) -> NextTurn
	// Return StateNone to signal completion
	return StateNone
}

func (s *TurnEndState) Exit(ctx *StateContext) {
	if s.actionCtx != nil {
		s.actionCtx.Clear()
	}
}

// handleFactionCharging handles faction-specific charging mechanics.
func (s *TurnEndState) handleFactionCharging(ctx *StateContext, player *core.Player) {
	faction := player.GetFaction()

	switch faction {
	case constants.FactionQingLong:
		// 青龙行迹: charge every turn (max 1)
		current := player.GetChargeCount()
		if current < 1 {
			player.SetChargeCount(current + 1)
		}
	case constants.FactionXuanWu:
		// 玄武镇厄: charge every turn (max 1)
		current := player.GetChargeCount()
		if current < 1 {
			player.SetChargeCount(current + 1)
		}
	case constants.FactionZhuQue:
		// 朱雀离火: handled by Fire buff handler in PhaseBeforeTurn
		// No additional action needed here
	case constants.FactionBaiHu:
		// 白虎劫运: handled during movement (overtaken check)
		// No additional action needed here
	}
}

// ========== Turn State Factory ==========

// TurnStateFactory creates turn layer states.
type TurnStateFactory struct {
	config *HSMConfig
}

// CreateState creates a turn state by ID.
func (f *TurnStateFactory) CreateState(id StateID) State {
	config := f.config
	if config == nil {
		config = DefaultHSMConfig()
	}

	switch id {
	case StateTurnUpkeep:
		return NewTurnUpkeepState()
	case StateMainAction:
		timeout := config.MainActionTimeout
		if timeout == 0 {
			timeout = 45 * time.Second
		}
		return NewMainActionState(timeout)
	case StateTurnMoving:
		return NewTurnMovingState()
	case StateTurnCheckpoint:
		return NewTurnCheckpointState()
	case StateTurnLanded:
		return NewTurnLandedState()
	case StateTurnDraw:
		return NewTurnDrawState()
	case StateTurnEnd:
		return NewTurnEndState()
	default:
		return nil
	}
}

// RegisterTurnStates registers all turn states with HSM.
func RegisterTurnStates(hsm *HSM) error {
	config := hsm.config
	if config == nil {
		config = DefaultHSMConfig()
	}
	factory := &TurnStateFactory{config: config}
	states := []State{
		factory.CreateState(StateTurnUpkeep),
		factory.CreateState(StateMainAction),
		factory.CreateState(StateTurnMoving),
		factory.CreateState(StateTurnCheckpoint),
		factory.CreateState(StateTurnLanded),
		factory.CreateState(StateTurnDraw),
		factory.CreateState(StateTurnEnd),
	}
	return hsm.RegisterStates(states)
}

// ========== Helper Types ==========

// StateError represents an error during state execution.
type StateError struct {
	StateID StateID
	Message string
}

// NewStateError creates a new StateError.
func NewStateError(id StateID, msg string) *StateError {
	return &StateError{
		StateID: id,
		Message: msg,
	}
}

func (e *StateError) Error() string {
	return e.StateID.String() + ": " + e.Message
}
