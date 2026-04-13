package engine

import (
	"errors"
	"time"

	"github.com/b1tAction/Fated/internal/core"
	"github.com/b1tAction/Fated/internal/gamemap"
	"github.com/b1tAction/Fated/pkg/event"
)

// TurnStep represents a step in the turn flow.
type TurnStep int

const (
	StepInit TurnStep = iota
	StepUpcheck       // Check frozen/stunned, decide skip turn
	StepBeforeTurn    // Trigger BeforeTurn phase
	StepMainAction    // User chooses items/skills
	StepOnMove        // Roll dice, calculate path
	StepOnLand        // Landing events
	StepPreEvent      // Event immunity check
	StepAfterTurn     // Tick buffs, post-turn effects
	StepComplete
)

// String returns the string representation of TurnStep.
func (ts TurnStep) String() string {
	names := map[TurnStep]string{
		StepInit:       "Init",
		StepUpcheck:    "Upcheck",
		StepBeforeTurn: "BeforeTurn",
		StepMainAction: "MainAction",
		StepOnMove:     "OnMove",
		StepOnLand:     "OnLand",
		StepPreEvent:   "PreEvent",
		StepAfterTurn:  "AfterTurn",
		StepComplete:   "Complete",
	}
	if name, ok := names[ts]; ok {
		return name
	}
	return "Unknown"
}

// StepResult represents the result of executing a step.
type StepResult struct {
	Step          TurnStep
	Success       bool
	Decisions     []*event.Decision // Decisions that need user input
	PathResult    *gamemap.PathResult // Movement result (for OnMove)
	TriggeredEvent *core.EventDefinition // Event triggered (for OnLand)
	PlayerUpdated bool               // Whether player state changed
	Error         error
}

// UserResponse represents user's response to a decision.
type UserResponse struct {
	DecisionID string
	Choice     int
	TimedOut   bool
}

// TurnFlow is the turn flow controller.
type TurnFlow struct {
	Game          *Game
	StateMachine  *StateMachine
	MapEngine     *gamemap.MapEngine
	CurrentStep   TurnStep
	CurrentPlayer *core.Player
	Interrupted   bool
	SavedSnapshot *FlowSnapshot
	Decisions     []*event.Decision // Current pending decisions
	DiceSteps     int               // Dice roll result
}

// NewTurnFlow creates a new turn flow controller.
func NewTurnFlow(game *Game, mapEngine *gamemap.MapEngine) *TurnFlow {
	return &TurnFlow{
		Game:         game,
		StateMachine: NewStateMachine(game),
		MapEngine:    mapEngine,
		CurrentStep:  StepInit,
		Interrupted:  false,
		Decisions:    make([]*event.Decision, 0),
	}
}

// ExecuteTurn executes a complete turn for the current player.
// Returns pending decisions if user input is needed.
func (tf *TurnFlow) ExecuteTurn(player *core.Player) ([]*event.Decision, error) {
	tf.CurrentPlayer = player
	tf.CurrentStep = StepInit
	tf.Decisions = make([]*event.Decision, 0)

	// Execute steps until we need user input or complete
	for tf.CurrentStep < StepComplete {
		result := tf.ExecuteStep(tf.CurrentStep, player)

		if result.Error != nil {
			return nil, result.Error
		}

		if len(result.Decisions) > 0 {
			// Need user input, pause and return decisions
			tf.Decisions = result.Decisions
			return result.Decisions, nil
		}

		tf.CurrentStep++
	}

	// Turn complete, advance to next player
	tf.Game.NextTurn()
	tf.CurrentStep = StepInit
	return nil, nil
}

// ExecuteStep executes a single step in the turn flow.
func (tf *TurnFlow) ExecuteStep(step TurnStep, player *core.Player) *StepResult {
	result := &StepResult{
		Step:      step,
		Success:   false,
		Decisions: make([]*event.Decision, 0),
	}

	switch step {
	case StepUpcheck:
		return tf.executeUpcheck(player)
	case StepBeforeTurn:
		return tf.executeBeforeTurn(player)
	case StepMainAction:
		return tf.executeMainAction(player)
	case StepOnMove:
		return tf.executeOnMove(player)
	case StepOnLand:
		return tf.executeOnLand(player)
	case StepPreEvent:
		return tf.executePreEvent(player)
	case StepAfterTurn:
		return tf.executeAfterTurn(player)
	default:
		result.Error = errors.New("unknown step")
		return result
	}
}

// executeUpcheck checks if player can act this turn.
func (tf *TurnFlow) executeUpcheck(player *core.Player) *StepResult {
	result := &StepResult{Step: StepUpcheck, Success: true}

	// Check if player is dead
	if player.IsDead {
		// Respawn at checkpoint
		checkpoint := tf.MapEngine.GetLastCheckpoint(player.Position)
		player.Respawn(checkpoint)
		result.PlayerUpdated = true
	}

	// Check SkipTurn flag (frozen/stunned)
	if player.SkipTurn {
		player.SkipTurn = false // Reset after skipping
		// Skip to AfterTurn
		tf.CurrentStep = StepAfterTurn - 1
	}

	// Check if player can act
	if !player.CanAct() {
		// Cannot act, skip to AfterTurn
		tf.CurrentStep = StepAfterTurn - 1
	}

	return result
}

// executeBeforeTurn triggers BeforeTurn phase.
func (tf *TurnFlow) executeBeforeTurn(player *core.Player) *StepResult {
	result := &StepResult{Step: StepBeforeTurn}

	// Trigger BeforeTurn phase
	pendingDecisions := tf.StateMachine.TriggerPhase(event.PhaseBeforeTurn, player)
	if len(pendingDecisions) > 0 {
		result.Decisions = pendingDecisions
		result.Success = false // Paused for user input
	} else {
		result.Success = true
		result.PlayerUpdated = true
	}

	return result
}

// executeMainAction handles user's main action selection.
func (tf *TurnFlow) executeMainAction(player *core.Player) *StepResult {
	result := &StepResult{Step: StepMainAction}

	// Check if player has usable items
	usableItems := tf.getUsableItems(player)
	if len(usableItems) == 0 {
		// No items, proceed to roll dice
		result.Success = true
		return result
	}

	// Create decision for item usage
	options := []event.Option{
		{ID: "roll", Label: "Roll Dice"},
	}
	for _, item := range usableItems {
		def := item.Type.GetItemDefinition()
		options = append(options, event.Option{
			ID:     item.ID,
			Label:  def.Name,
			Action: nil, // Action set during subscription
		})
	}

	decision := event.NewDecision("Choose action:", options)
	result.Decisions = []*event.Decision{decision}
	result.Success = false // Paused for user input

	return result
}

// getUsableItems returns items usable in current context.
func (tf *TurnFlow) getUsableItems(player *core.Player) []*core.Item {
	var usable []*core.Item
	for _, item := range player.Inventory {
		def := item.Type.GetItemDefinition()
		// Items usable in BeforeTurn or AnyTime
		if def.Phase == event.PhaseBeforeTurn || def.Phase == event.PhaseAnyTime {
			usable = append(usable, item)
		}
	}
	return usable
}

// executeOnMove handles dice roll and movement.
func (tf *TurnFlow) executeOnMove(player *core.Player) *StepResult {
	result := &StepResult{Step: StepOnMove}

	// Trigger OnMove phase (for Lost reverse direction)
	_ = tf.StateMachine.TriggerPhase(event.PhaseOnMove, player)
	// OnMove phase doesn't usually wait for user input

	// Calculate movement (dice steps should be set externally or via previous step)
	if tf.DiceSteps <= 0 {
		// Default dice roll (should be replaced with actual dice system)
		tf.DiceSteps = 6 // Placeholder
	}

	// Check for Lost buff (reverse movement)
	if player.HasBuff(core.BuffTypeLost) {
		// Reverse movement - move backwards
		targetPos := player.Position - tf.DiceSteps
		if targetPos < 0 {
			targetPos = 0
		}
		tf.DiceSteps = player.Position - targetPos
	}

	// Calculate path
	pathResult, err := tf.MapEngine.CalculatePath(player.Position, tf.DiceSteps)
	if err != nil {
		result.Error = err
		return result
	}

	result.PathResult = pathResult
	result.Success = true
	result.PlayerUpdated = true

	// Update player position
	player.Position = pathResult.TargetIndex

	// Handle falling
	if pathResult.FellDown {
		// Player fell, apply damage
		player.ApplyDamage(1)
	}

	// Check if reached end
	if pathResult.ReachedEnd {
		// Reached Boss cell - trigger win condition
		// TODO: Implement Boss battle
	}

	return result
}

// executeOnLand handles landing events.
func (tf *TurnFlow) executeOnLand(player *core.Player) *StepResult {
	result := &StepResult{Step: StepOnLand}

	// Trigger OnLand phase
	pendingDecisions := tf.StateMachine.TriggerPhase(event.PhaseOnLand, player)
	if len(pendingDecisions) > 0 {
		result.Decisions = pendingDecisions
		result.Success = false
		return result
	}

	// Get the cell at player's position
	cell, err := tf.MapEngine.GetCell(player.Position)
	if err != nil {
		result.Error = err
		return result
	}

	// Execute cell event if exists
	if cell.EventID != "" {
		// TODO: Implement event execution
		// This would involve looking up the event and executing its effects
	}

	result.Success = true
	return result
}

// executePreEvent handles event immunity check.
func (tf *TurnFlow) executePreEvent(player *core.Player) *StepResult {
	result := &StepResult{Step: StepPreEvent}

	// Trigger PreEvent phase
	_ = tf.StateMachine.TriggerPhase(event.PhasePreEvent, player)

	result.Success = true
	return result
}

// executeAfterTurn handles post-turn effects.
func (tf *TurnFlow) executeAfterTurn(player *core.Player) *StepResult {
	result := &StepResult{Step: StepAfterTurn}

	// Trigger AfterTurn phase
	_ = tf.StateMachine.TriggerPhase(event.PhaseAfterTurn, player)

	// Tick buff durations
	expiredBuffs := player.TickBuffs()
	for _, buff := range expiredBuffs {
		// Remove expired buffs from EventBus
		tf.Game.UnsubscribeBuff(buff)
	}

	// Check death
	if player.IsDead {
		// Respawn at checkpoint
		checkpoint := tf.MapEngine.GetLastCheckpoint(player.Position)
		player.Respawn(checkpoint)
		result.PlayerUpdated = true
	}

	result.Success = true
	result.PlayerUpdated = true
	return result
}

// OnUserChoice handles user's choice for pending decisions.
func (tf *TurnFlow) OnUserChoice(choice int) error {
	if len(tf.Decisions) == 0 {
		return errors.New("no pending decisions")
	}

	current := tf.Decisions[0]
	ctx := event.NewContext(tf.CurrentPlayer)
	current.Execute(choice, ctx)

	tf.Decisions = tf.Decisions[1:]

	// If all decisions handled, continue flow
	if len(tf.Decisions) == 0 {
		tf.CurrentStep++
	}

	return nil
}

// OnUserChoiceWithID handles user's choice by decision ID.
func (tf *TurnFlow) OnUserChoiceWithID(decisionID string, choice int) error {
	for i, decision := range tf.Decisions {
		if decision.ID == decisionID {
			ctx := event.NewContext(tf.CurrentPlayer)
			decision.Execute(choice, ctx)
			tf.Decisions = append(tf.Decisions[:i], tf.Decisions[i+1:]...)

			if len(tf.Decisions) == 0 {
				tf.CurrentStep++
			}
			return nil
		}
	}
	return errors.New("decision not found")
}

// SetDiceSteps sets the dice roll result (for external dice system).
func (tf *TurnFlow) SetDiceSteps(steps int) {
	tf.DiceSteps = steps
}

// GetCurrentStep returns current step.
func (tf *TurnFlow) GetCurrentStep() TurnStep {
	return tf.CurrentStep
}

// IsWaiting returns whether flow is waiting for user input.
func (tf *TurnFlow) IsWaiting() bool {
	return len(tf.Decisions) > 0
}

// GetPendingDecisions returns pending decisions.
func (tf *TurnFlow) GetPendingDecisions() []*event.Decision {
	return tf.Decisions
}

// Interrupt interrupts the current flow.
func (tf *TurnFlow) Interrupt() error {
	if tf.IsWaiting() {
		tf.Interrupted = true
		tf.SavedSnapshot = tf.CreateSnapshot()
		return nil
	}
	return errors.New("cannot interrupt while not waiting")
}

// ResumeFromInterrupt resumes flow from a saved snapshot.
func (tf *TurnFlow) ResumeFromInterrupt(snapshot *FlowSnapshot) error {
	if snapshot == nil {
		return errors.New("snapshot is nil")
	}

	// Restore game state
	tf.Game.State.Round = snapshot.Round
	tf.Game.State.Turn = snapshot.Turn
	tf.CurrentStep = snapshot.CurrentStep

	// Find player
	player := tf.Game.GetPlayer(snapshot.PlayerID)
	if player == nil {
		return errors.New("player not found")
	}
	tf.CurrentPlayer = player

	// Restore pending decisions
	tf.Decisions = make([]*event.Decision, 0)
	for _, ds := range snapshot.WaitingDecisions {
		decision := event.NewDecision(ds.Prompt, make([]event.Option, 0))
		decision.ID = ds.DecisionID
		decision.Default = ds.DefaultChoice
		tf.Decisions = append(tf.Decisions, decision)
	}

	tf.Interrupted = false
	return nil
}

// CreateSnapshot creates a snapshot of current flow state.
func (tf *TurnFlow) CreateSnapshot() *FlowSnapshot {
	snapshot := &FlowSnapshot{
		GameID:       tf.Game.ID,
		Round:        tf.Game.State.Round,
		Turn:         tf.Game.State.Turn,
		CurrentStep:  tf.CurrentStep,
		PlayerID:     tf.CurrentPlayer.UserID,
		Timestamp:    time.Now(),
		WaitingDecisions: make([]*DecisionSnapshot, 0),
	}

	for _, decision := range tf.Decisions {
		ds := &DecisionSnapshot{
			DecisionID:   decision.ID,
			Prompt:       decision.Prompt,
			DefaultChoice: decision.Default,
		}
		snapshot.WaitingDecisions = append(snapshot.WaitingDecisions, ds)
	}

	return snapshot
}

// WaitForUserDecision waits for user input with timeout.
func (tf *TurnFlow) WaitForUserDecision(timeout time.Duration) (*UserResponse, error) {
	if len(tf.Decisions) == 0 {
		return nil, errors.New("no pending decisions")
	}

	// This is a placeholder - actual implementation would use
	// external event system or channel for user input
	// For now, return timeout response
	return &UserResponse{
		TimedOut: true,
		Choice:   tf.Decisions[0].Default,
	}, nil
}