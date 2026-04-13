package hsm

import (
	"time"

	"github.com/b1tAction/Fated/pkg/event"
)

// State defines the interface for all states in the HSM.
// Each state must implement Enter, Update, Exit methods for lifecycle management.
type State interface {
	// ID returns the unique identifier of the state.
	ID() StateID

	// Enter is called when transitioning into this state.
	// Use ctx to access game data, trigger phases, and setup state-specific logic.
	Enter(ctx *StateContext)

	// Update is called each tick or event while in this state.
	// Return next state ID to trigger transition, or StateNone to stay in current state.
	Update(ctx *StateContext) StateID

	// Exit is called when transitioning out of this state.
	// Cleanup state-specific resources and reset temporary flags.
	Exit(ctx *StateContext)

	// CanTransitionTo checks if transition to target state is valid.
	// Used for enforcing state flow rules and preventing illegal transitions.
	CanTransitionTo(target StateID) bool
}

// StateContext provides context data passed to state methods.
// Contains game reference, player data, phase info, and interrupt stack.
type StateContext struct {
	// Core references
	Game      GameAdapter       // Game adapter for accessing game state and EventBus
	Player    PlayerAdapter     // Current player (used in Layer 2 states)

	// Phase triggering
	Phase     event.Phase        // Current phase to trigger
	PhaseData interface{}        // Additional phase data (e.g., damage amount, dice steps)

	// Movement data
	DiceSteps int                // Dice roll result for movement calculation
	TargetPos int                // Target position after movement

	// Decision handling
	Decision  *event.Decision    // Pending decision requiring user input
	Decisions []*event.Decision  // List of pending decisions

	// Timing
	Timeout   time.Duration      // Timeout duration for waiting states
	StartTime time.Time          // State entry time

	// Stack reference (Layer 3)
	Stack     *StateStack        // Reference to interrupt stack

	// State result markers
	Success   bool               // Whether state execution succeeded
	Error     error              // Error if state execution failed
	SkipTurn  bool               // Mark to skip remaining turn
	FellDown  bool               // Mark for Fragile fall
	ReachedEnd bool              // Mark for reaching Boss cell

	// Additional metadata
	Metadata  map[string]interface{} // Extensible data container
}

// NewStateContext creates a new StateContext with default values.
func NewStateContext() *StateContext {
	return &StateContext{
		Metadata:  make(map[string]interface{}),
		Decisions: make([]*event.Decision, 0),
		Success:   true,
	}
}

// WithGame sets the game adapter.
func (ctx *StateContext) WithGame(game GameAdapter) *StateContext {
	ctx.Game = game
	return ctx
}

// WithPlayer sets the player adapter.
func (ctx *StateContext) WithPlayer(player PlayerAdapter) *StateContext {
	ctx.Player = player
	return ctx
}

// WithPhase sets the phase and optional data.
func (ctx *StateContext) WithPhase(phase event.Phase, data interface{}) *StateContext {
	ctx.Phase = phase
	ctx.PhaseData = data
	return ctx
}

// WithDiceSteps sets dice steps for movement.
func (ctx *StateContext) WithDiceSteps(steps int) *StateContext {
	ctx.DiceSteps = steps
	return ctx
}

// WithDecision sets a pending decision.
func (ctx *StateContext) WithDecision(decision *event.Decision) *StateContext {
	ctx.Decision = decision
	return ctx
}

// WithDecisions sets multiple pending decisions.
func (ctx *StateContext) WithDecisions(decisions []*event.Decision) *StateContext {
	ctx.Decisions = decisions
	return ctx
}

// WithTimeout sets timeout duration.
func (ctx *StateContext) WithTimeout(timeout time.Duration) *StateContext {
	ctx.Timeout = timeout
	return ctx
}

// WithStack sets the state stack reference.
func (ctx *StateContext) WithStack(stack *StateStack) *StateContext {
	ctx.Stack = stack
	return ctx
}

// SetMetadata sets a metadata key-value pair.
func (ctx *StateContext) SetMetadata(key string, value interface{}) {
	if ctx.Metadata == nil {
		ctx.Metadata = make(map[string]interface{})
	}
	ctx.Metadata[key] = value
}

// GetMetadata retrieves a metadata value by key.
func (ctx *StateContext) GetMetadata(key string) interface{} {
	if ctx.Metadata == nil {
		return nil
	}
	return ctx.Metadata[key]
}

// Clear resets the context to default values.
func (ctx *StateContext) Clear() {
	ctx.Phase = event.Phase(0)
	ctx.PhaseData = nil
	ctx.DiceSteps = 0
	ctx.TargetPos = 0
	ctx.Decision = nil
	ctx.Decisions = make([]*event.Decision, 0)
	ctx.Timeout = 0
	ctx.Success = true
	ctx.Error = nil
	ctx.SkipTurn = false
	ctx.FellDown = false
	ctx.ReachedEnd = false
	ctx.Metadata = make(map[string]interface{})
}