package hsm

import (
	"fmt"
	"time"

	"github.com/b1tAction/paradiced/internal/core"
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
	s.actionCtx = engineaction.NewActionContext(
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
	s.actionCtx = engineaction.NewActionContext(
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
// Uses Action system for MoveAction execution.
type TurnMovingState struct {
	BaseTurnState
	pathResult PathResultData
	fellDown   bool
	reachedEnd bool
	actionCtx  *engineaction.ActionContext
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
		fellDown:      false,
		reachedEnd:    false,
	}
}

func (s *TurnMovingState) Enter(ctx *StateContext) {
	player := ctx.Player
	if player == nil {
		ctx.Error = errors.WrapHSMError(
			errors.NewInternalError("HSM", "TurnMoving", nil),
			"TurnMoving", 2, "Enter", "player is nil")
		return
	}

	// Get dice steps
	steps := ctx.GetDiceSteps()
	fmt.Printf("[hsm] TurnMovingState.Enter: dice_steps=%d, player_position=%d\n", steps, player.Position)
	if steps <= 0 {
		// Invalid steps, set error and end turn
		fmt.Printf("[hsm] TurnMovingState.Enter: invalid dice steps, ending turn\n")
		ctx.Error = errors.WrapHSMError(
			errors.NewValidationError("dice_steps", steps, "must be positive"),
			"TurnMoving", 2, "Enter", "invalid dice steps")
		return
	}

	// Create ActionContext
	game := ctx.GetGame()
	mapEngine := ctx.GetMapEngine()
	s.actionCtx = engineaction.NewActionContext(
		game,
		game.Bus,
		mapEngine,
		game.Draw,
	)

	// Create and execute MoveAction
	// MoveAction's PreTriggerPhase = PhasePreMove (迷途 can intercept)
	moveAction := engineaction.NewMoveAction(player, steps, "DiceRoll")

	// Execute through ActionContext (handles PreTrigger, Execute, PostTrigger)
	if err := s.actionCtx.ExecuteAction(moveAction); err != nil {
		ctx.Error = errors.WrapHSMError(
			err, "TurnMoving", 2, "Enter", "move action execution failed")
		return
	}

	// Get path result from MoveAction
	s.pathResult.TargetIndex = moveAction.TargetPos
	s.pathResult.Path = moveAction.Path
	s.pathResult.StartIndex = player.Position - steps // Approximate

	// Check for Fragile fall (PathResult.Interrupted/FellDown)
	// This would be set by MapEngine during CalculatePath
	// For now, check if player position changed correctly
	if player.Position != s.pathResult.TargetIndex {
		// Position mismatch might indicate fall
		// In full implementation, check MapEngine.PathResult.FellDown
	}

	// Check for reaching end (Boss cell)
	if mapEngine != nil {
		mapLength := mapEngine.Length
		if player.Position >= mapLength-1 {
			s.reachedEnd = true
			ctx.SetReachedEnd(true)
		}
	}

	// Handle Fog activation (first player passing through)
	for _, pos := range s.pathResult.Path {
		if mapEngine != nil {
			cell, _ := mapEngine.GetCell(pos)
			if cell != nil && cell.CellType == gamemap.CellTypeFog {
				mapEngine.ActivateFog(pos)
			}
		}
	}

	// Note: Overtaken handling (白虎劫运) would be implemented
	// by checking players at positions in path and generating StealBuffAction
}

func (s *TurnMovingState) Update(ctx *StateContext) StateID {
	// Check if fell down -> skip to TurnEnd, use FellDownAction
	if s.fellDown {
		ctx.SetFellDown(true)
		// Use FellDownAction for falling
		if s.actionCtx != nil && ctx.Player != nil {
			fellDownAction := engineaction.NewFellDownAction(ctx.Player, ctx.Player.Position, 1, "FragileCell")
			if err := s.actionCtx.ExecuteAction(fellDownAction); err != nil {
				ctx.Error = errors.WrapHSMError(
					err, "TurnMoving", 2, "Update", "fell down action failed")
				return StateNone // Block state transition
			}
		}
		return StateTurnEnd
	}

	// Check if reached Boss -> notify TurnLoop
	if s.reachedEnd {
		// TurnLoop will handle transition to BossBattle
		return StateTurnLanded
	}

	// Normal flow -> TurnLanded
	return StateTurnLanded
}

func (s *TurnMovingState) Exit(ctx *StateContext) {
	s.fellDown = false
	s.reachedEnd = false
	s.pathResult = PathResultData{}
	if s.actionCtx != nil {
		s.actionCtx.Clear()
	}
}

// ========== TurnLandedState ==========

// TurnLandedState handles landing effects and PhaseOnLand trigger.
type TurnLandedState struct {
	BaseTurnState
	cellType  gamemap.CellType
	decisions []*event.Decision
	actionCtx *engineaction.ActionContext
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
	s.actionCtx = engineaction.NewActionContext(
		game,
		game.Bus,
		mapEngine,
		game.Draw,
	)

	// Get cell type at landing position
	if mapEngine != nil {
		cell, err := mapEngine.GetCell(player.Position)
		if err == nil && cell != nil {
			s.cellType = cell.CellType
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

	// Handle special cell types
	switch s.cellType {
	case gamemap.CellTypeCheckpoint:
		// Checkpoint: could trigger treasure refresh (implementation specific)
	case gamemap.CellTypeBoss:
		// Boss cell: mark as reached end (already handled in Moving)
	}
}

func (s *TurnLandedState) Update(ctx *StateContext) StateID {
	// Check if decisions need processing
	if len(s.decisions) > 0 {
		ctx.Decisions = s.decisions
		return StateNone // Wait for decision
	}

	// Normal flow -> TurnEvent
	return StateTurnEvent
}

func (s *TurnLandedState) Exit(ctx *StateContext) {
	s.cellType = gamemap.CellTypeNormal
	s.decisions = make([]*event.Decision, 0)
	if s.actionCtx != nil {
		s.actionCtx.Clear()
	}
}

// ========== TurnEventState ==========

// TurnEventState handles random event drawing and execution.
// Triggers PhasePreEvent for immunity checks.
type TurnEventState struct {
	BaseTurnState
	eventDrawn   bool
	eventBlocked bool
	decisions    []*event.Decision
	actionCtx    *engineaction.ActionContext
}

// NewTurnEventState creates a new TurnEvent state.
func NewTurnEventState() *TurnEventState {
	return &TurnEventState{
		BaseTurnState: BaseTurnState{id: StateTurnEvent},
		eventDrawn:    false,
		eventBlocked:  false,
		decisions:     make([]*event.Decision, 0),
	}
}

func (s *TurnEventState) Enter(ctx *StateContext) {
	player := ctx.Player
	if player == nil {
		ctx.Error = errors.WrapHSMError(
			errors.NewInternalError("HSM", "TurnEvent", nil),
			"TurnEvent", 2, "Enter", "player is nil")
		return
	}

	// Create ActionContext
	game := ctx.GetGame()
	mapEngine := ctx.GetMapEngine()
	s.actionCtx = engineaction.NewActionContext(
		game,
		game.Bus,
		mapEngine,
		game.Draw,
	)

	// Create DrawEventAction (PreTriggerPhase = PhasePreEvent)
	// This allows 辟邪/玄武 to intercept bad events
	drawAction := engineaction.NewDrawEventAction(player, "CellEvent")

	// Execute through ActionContext
	if err := s.actionCtx.ExecuteAction(drawAction); err != nil {
		ctx.Error = errors.WrapHSMError(
			err, "TurnEvent", 2, "Enter", "draw event action failed")
		return
	}

	// Check if event was blocked (set during PhasePreEvent interception)
	s.eventBlocked = s.actionCtx.GetBoolOrDefault("event_blocked", false)

	s.eventDrawn = true

	// Note: Actual event drawing and execution would be in DrawEventAction.Execute()
	// which would use RNG engine and event pool
}

func (s *TurnEventState) Update(ctx *StateContext) StateID {
	// Check if decisions need processing
	if len(s.decisions) > 0 {
		ctx.Decisions = s.decisions
		return StateNone
	}

	// Normal flow -> TurnEnd
	return StateTurnEnd
}

func (s *TurnEventState) Exit(ctx *StateContext) {
	s.eventDrawn = false
	s.eventBlocked = false
	s.decisions = make([]*event.Decision, 0)
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
	s.actionCtx = engineaction.NewActionContext(
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

	// End turn log segment
	if game != nil && game.Log != nil {
		game.Log.EndTurn()
	}

	// Broadcast TurnSync (all actions from this turn)
	s.broadcastTurnSync(ctx)

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
	case StateTurnLanded:
		return NewTurnLandedState()
	case StateTurnEvent:
		return NewTurnEventState()
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
		factory.CreateState(StateTurnLanded),
		factory.CreateState(StateTurnEvent),
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
