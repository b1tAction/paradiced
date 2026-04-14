package hsm

import (
	"time"

	"github.com/b1tAction/Fated/internal/core"
	"github.com/b1tAction/Fated/internal/gamemap"
	engineaction "github.com/b1tAction/Fated/internal/engine/action"
	"github.com/b1tAction/Fated/pkg/event"
	"github.com/b1tAction/Fated/pkg/protocol"
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
	skipTurn    bool
	isDead      bool
	decisions   []*event.Decision
	actionCtx   *engineaction.ActionContext
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
		ctx.Success = false
		ctx.Error = NewStateError(StateTurnUpkeep, "player is nil")
		return
	}

	// Start turn log segment
	if ctx.Game != nil && ctx.Game.Log != nil {
		ctx.Game.Log.StartTurn(ctx.Game.State.Round, ctx.Game.State.Turn, player.UserID)
	}

	// Create ActionContext for executing Actions
	s.actionCtx = engineaction.NewActionContext(
		NewGameWrapper(ctx.Game),
		ctx.Game.Bus,
		NewProtocolMapEngineWrapper(ctx.MapEngine),
	)

	// Step 1: Check IsDead -> Respawn at checkpoint using RespawnAction
	if player.IsDead {
		if ctx.MapEngine != nil {
			checkpoint := ctx.MapEngine.GetLastCheckpoint(player.Position)
			respawnAction := engineaction.NewRespawnAction(player, checkpoint, "DeathRespawn")
			s.actionCtx.ExecuteAction(respawnAction)
		}
		s.isDead = true
	}

	// Step 2: Check SkipTurn flag
	if player.SkipTurn {
		player.SkipTurn = false // Clear flag
		s.skipTurn = true
		ctx.SetSkipTurn(true)
		return // Skip all BeforeTurn effects
	}

	// Step 3: Trigger PhaseBeforeTurn
	// HSM publishes this phase - Buff handlers respond and may return Actions
	triggerCtx := event.NewContext(player)
	triggerCtx.Set("action_context", s.actionCtx)
	triggerCtx.Set("current_player", player)

	// Publish PhaseBeforeTurn to trigger Buff effects
	s.decisions = ctx.Bus.Publish(event.PhaseBeforeTurn, player.UserID, triggerCtx)

	// Step 4: Execute any derived Actions from handlers
	s.actionCtx.ProcessQueue()

	// Step 5: Check if any decisions need user input
	if len(s.decisions) > 0 {
		// Will be handled in Update - push WaitDecision if needed
		ctx.Decisions = s.decisions
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
func NewMainActionState() *MainActionState {
	return &MainActionState{
		BaseTurnState:    BaseTurnState{id: StateMainAction},
		waitingForAction: true,
		diceRolled:       false,
		timeout:          45 * time.Second, // Default timeout
	}
}

func (s *MainActionState) Enter(ctx *StateContext) {
	player := ctx.Player
	if player == nil {
		ctx.Success = false
		ctx.Error = NewStateError(StateMainAction, "player is nil")
		return
	}

	// Initialize action context
	s.actionCtx = engineaction.NewActionContext(
		NewGameWrapper(ctx.Game),
		ctx.Game.Bus,
		NewProtocolMapEngineWrapper(ctx.MapEngine),
	)

	s.startTime = time.Now()
	s.waitingForAction = true
	s.diceRolled = false

	// Build available actions:
	// 1. PhaseItemUsed items (active items)
	// 2. Faction skills (QingLong/XuanWu charge check)
	// Note: Actual item/skill execution is handled by OnUseItem/OnUseSkill
}

func (s *MainActionState) Update(ctx *StateContext) StateID {
	// Check if dice was rolled
	if s.diceRolled {
		ctx.SetInt(KeyDiceSteps, s.diceSteps)
		return StateTurnMoving
	}

	// Check timeout
	if time.Since(s.startTime) > s.timeout {
		// Auto roll dice (default action)
		s.OnRollDice(ctx, s.defaultDiceRoll(ctx))
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

	ctx.Bus.Publish(event.PhaseItemUsed, player.UserID, triggerCtx)

	// Process derived actions
	s.actionCtx.ProcessQueue()
}

// defaultDiceRoll returns default dice steps based on dice type.
func (s *MainActionState) defaultDiceRoll(ctx *StateContext) int {
	// Get dice type from context (assigned in RoundPrep)
	diceType := ctx.GetDiceType(ctx.Player.UserID)
	switch diceType {
	case "gold":
		return 6 // Gold dice: 1-10, default to 6
	case "silver":
		return 4 // Silver dice: 1-7, default to 4
	case "copper":
		return 3 // Copper dice: 1-5, default to 3
	default:
		return 2 // Wood dice: 1-3, default to 2
	}
}

// ========== TurnMovingState ==========

// TurnMovingState handles movement calculation and path processing.
// Uses Action system for MoveAction execution.
type TurnMovingState struct {
	BaseTurnState
	pathResult  PathResultData
	fellDown    bool
	reachedEnd  bool
	actionCtx   *engineaction.ActionContext
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
		ctx.Success = false
		ctx.Error = NewStateError(StateTurnMoving, "player is nil")
		return
	}

	// Get dice steps
	steps := ctx.GetDiceSteps()
	if steps <= 0 {
		// Invalid steps, end turn
		return
	}

	// Create ActionContext
	s.actionCtx = engineaction.NewActionContext(
		NewGameWrapper(ctx.Game),
		ctx.Game.Bus,
		NewProtocolMapEngineWrapper(ctx.MapEngine),
	)

	// Create and execute MoveAction
	// MoveAction's PreTriggerPhase = PhasePreMove (迷途 can intercept)
	moveAction := engineaction.NewMoveAction(player, steps, "DiceRoll")

	// Execute through ActionContext (handles PreTrigger, Execute, PostTrigger)
	err := s.actionCtx.ExecuteAction(moveAction)
	if err != nil {
		ctx.Success = false
		ctx.Error = NewStateError(StateTurnMoving, err.Error())
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
	if ctx.MapEngine != nil {
		mapLength := ctx.MapEngine.GetLength()
		if player.Position >= mapLength-1 {
			s.reachedEnd = true
			ctx.SetReachedEnd(true)
		}
	}

	// Handle Fog activation (first player passing through)
	for _, pos := range s.pathResult.Path {
		if ctx.MapEngine != nil {
			cell, _ := ctx.MapEngine.GetCell(pos)
			if cell != nil && cell.CellType == gamemap.CellTypeFog {
				ctx.MapEngine.ActivateFog(pos)
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
		if s.actionCtx != nil {
			fellDownAction := engineaction.NewFellDownAction(ctx.Player, ctx.Player.Position, 1, "FragileCell")
			s.actionCtx.ExecuteAction(fellDownAction)
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
	cellType    CellType
	decisions   []*event.Decision
	actionCtx   *engineaction.ActionContext
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
		ctx.Success = false
		ctx.Error = NewStateError(StateTurnLanded, "player is nil")
		return
	}

	// Create ActionContext
	s.actionCtx = engineaction.NewActionContext(
		NewGameWrapper(ctx.Game),
		ctx.Game.Bus,
		NewProtocolMapEngineWrapper(ctx.MapEngine),
	)

	// Get cell type at landing position
	if ctx.MapEngine != nil {
		cell, err := ctx.MapEngine.GetCell(player.Position)
		if err == nil && cell != nil {
			s.cellType = cell.CellType
		}
	}

	// Trigger PhaseOnLand
	triggerCtx := event.NewContext(player)
	triggerCtx.Set("action_context", s.actionCtx)
	triggerCtx.Set("cell_type", s.cellType)
	triggerCtx.Set("position", player.Position)

	s.decisions = ctx.Bus.Publish(event.PhaseOnLand, player.UserID, triggerCtx)

	// Process derived actions
	s.actionCtx.ProcessQueue()

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
	eventDrawn    bool
	eventBlocked  bool
	decisions     []*event.Decision
	actionCtx     *engineaction.ActionContext
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
		ctx.Success = false
		ctx.Error = NewStateError(StateTurnEvent, "player is nil")
		return
	}

	// Create ActionContext
	s.actionCtx = engineaction.NewActionContext(
		NewGameWrapper(ctx.Game),
		ctx.Game.Bus,
		NewProtocolMapEngineWrapper(ctx.MapEngine),
	)

	// Create DrawEventAction (PreTriggerPhase = PhasePreEvent)
	// This allows 辟邪/玄武 to intercept bad events
	drawAction := engineaction.NewDrawEventAction(player, "CellEvent")

	// Execute through ActionContext
	err := s.actionCtx.ExecuteAction(drawAction)
	if err != nil {
		ctx.Success = false
		ctx.Error = NewStateError(StateTurnEvent, err.Error())
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
	actionCtx     *engineaction.ActionContext
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
		ctx.Success = false
		ctx.Error = NewStateError(StateTurnEnd, "player is nil")
		return
	}

	// Create ActionContext
	s.actionCtx = engineaction.NewActionContext(
		NewGameWrapper(ctx.Game),
		ctx.Game.Bus,
		NewProtocolMapEngineWrapper(ctx.MapEngine),
	)

	// Trigger PhaseAfterTurn
	triggerCtx := event.NewContext(player)
	triggerCtx.Set("action_context", s.actionCtx)

	decisions := ctx.Bus.Publish(event.PhaseAfterTurn, player.UserID, triggerCtx)

	// Process derived actions (甘霖/腐化 effects)
	s.actionCtx.ProcessQueue()

	// Tick Buff durations
	expiredBuffs := player.TickBuffs()

	// Handle expired buffs (unsubscribe from EventBus)
	for _, expired := range expiredBuffs {
		ctx.Game.UnsubscribeBuff(expired)
	}

	// Check IsDead after AfterTurn effects - use RespawnAction
	if player.IsDead {
		if ctx.MapEngine != nil {
			checkpoint := ctx.MapEngine.GetLastCheckpoint(player.Position)
			respawnAction := engineaction.NewRespawnAction(player, checkpoint, "AfterTurnRespawn")
			s.actionCtx.ExecuteAction(respawnAction)
		}
	}

	// Faction charging logic
	s.handleFactionCharging(ctx, player)

	// Handle decisions if any
	if len(decisions) > 0 {
		ctx.Decisions = decisions
	}

	// End turn log segment
	if ctx.Game != nil && ctx.Game.Log != nil {
		ctx.Game.Log.EndTurn()
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
	case protocol.FactionQingLong:
		// 青龙行迹: charge every turn (max 1)
		current := player.GetChargeCount()
		if current < 1 {
			player.SetChargeCount(current + 1)
		}
	case protocol.FactionXuanWu:
		// 玄武镇厄: charge every turn (max 1)
		current := player.GetChargeCount()
		if current < 1 {
			player.SetChargeCount(current + 1)
		}
	case protocol.FactionZhuQue:
		// 朱雀离火: handled by Fire buff handler in PhaseBeforeTurn
		// No additional action needed here
	case protocol.FactionBaiHu:
		// 白虎劫运: handled during movement (overtaken check)
		// No additional action needed here
	}
}

// ========== Turn State Factory ==========

// TurnStateFactory creates turn layer states.
type TurnStateFactory struct{}

// CreateState creates a turn state by ID.
func (f *TurnStateFactory) CreateState(id StateID) State {
	switch id {
	case StateTurnUpkeep:
		return NewTurnUpkeepState()
	case StateMainAction:
		return NewMainActionState()
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
	factory := &TurnStateFactory{}
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

// CellType is imported from gamemap package for convenience.
type CellType = gamemap.CellType