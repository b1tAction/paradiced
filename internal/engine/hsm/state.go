package hsm

import (
	"time"

	"github.com/b1tAction/Fated/internal/core"
	"github.com/b1tAction/Fated/internal/engine"
	"github.com/b1tAction/Fated/pkg/event"
	"github.com/b1tAction/Fated/pkg/util"
)

// Context key constants for commonly used state markers.
// These keys are stored in the embedded Metadata for type-safe access.
const (
	KeySkipTurn    = "skip_turn"     // Mark to skip remaining turn
	KeyFellDown    = "fell_down"     // Mark for Fragile cell fall
	KeyReachedEnd  = "reached_end"   // Mark for reaching Boss cell
	KeyDiceSteps   = "dice_steps"    // Dice roll result for movement
	KeyTargetPos   = "target_pos"    // Target position after movement
	KeyDamage      = "damage"        // Damage amount
	KeyMiniGameRank = "mini_game_rank" // Mini-game ranking result
	KeyDiceType    = "dice_type"     // Dice type (gold/silver/copper/wood)
	KeyBossTrigger = "boss_trigger_player" // Player who triggered boss battle
	KeyWinner      = "winner_id"     // Winner player ID
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
// Embeds util.Metadata for extensible type-safe key-value storage.
type StateContext struct {
	// Embedded Metadata for extensible storage
	*util.Metadata

	// Core references - direct types for domain objects
	Game      *engine.Game       // Game instance (direct access)
	Player    *core.Player       // Current player (direct access, used in Layer 2 states)

	// Adapter interfaces - for cross-package isolation
	Bus       EventBusAdapter    // EventBus adapter (isolates pkg/event)
	MapEngine MapEngineAdapter   // MapEngine adapter (isolates internal/gamemap)

	// Phase triggering
	Phase     event.Phase        // Current phase to trigger
	PhaseData interface{}        // Additional phase data (e.g., damage amount, dice steps)

	// Decision handling
	Decision  *event.Decision    // Pending decision requiring user input
	Decisions []*event.Decision  // List of pending decisions

	// Timing
	Timeout   time.Duration      // Timeout duration for waiting states
	StartTime time.Time          // State entry time

	// Stack reference (Layer 3)
	Stack     *StateStack        // Reference to interrupt stack

	// Execution result
	Success   bool               // Whether state execution succeeded
	Error     error              // Error if state execution failed
}

// NewStateContext creates a new StateContext with default values.
func NewStateContext() *StateContext {
	return &StateContext{
		Metadata:  util.NewMetadata(),
		Decisions: make([]*event.Decision, 0),
		Success:   true,
	}
}

// ========== Game/Player Setup ==========

// WithGame sets the game instance and creates EventBus wrapper.
func (ctx *StateContext) WithGame(game *engine.Game) *StateContext {
	ctx.Game = game
	if game != nil && game.Bus != nil {
		ctx.Bus = NewEventBusWrapper(game.Bus)
	}
	return ctx
}

// WithPlayer sets the player (direct type).
func (ctx *StateContext) WithPlayer(player *core.Player) *StateContext {
	ctx.Player = player
	return ctx
}

// WithBus sets the EventBus adapter directly.
func (ctx *StateContext) WithBus(bus EventBusAdapter) *StateContext {
	ctx.Bus = bus
	return ctx
}

// WithMapEngine sets the MapEngine adapter.
func (ctx *StateContext) WithMapEngine(engine MapEngineAdapter) *StateContext {
	ctx.MapEngine = engine
	return ctx
}

// ========== Phase Setup ==========

// WithPhase sets the phase and optional data.
func (ctx *StateContext) WithPhase(phase event.Phase, data interface{}) *StateContext {
	ctx.Phase = phase
	ctx.PhaseData = data
	return ctx
}

// ========== Movement Data ==========

// WithDiceSteps sets dice steps for movement (stored in Metadata).
func (ctx *StateContext) WithDiceSteps(steps int) *StateContext {
	ctx.SetInt(KeyDiceSteps, steps)
	return ctx
}

// GetDiceSteps retrieves dice steps from Metadata.
func (ctx *StateContext) GetDiceSteps() int {
	return ctx.GetIntOrDefault(KeyDiceSteps, 0)
}

// WithTargetPos sets target position after movement (stored in Metadata).
func (ctx *StateContext) WithTargetPos(pos int) *StateContext {
	ctx.SetInt(KeyTargetPos, pos)
	return ctx
}

// GetTargetPos retrieves target position from Metadata.
func (ctx *StateContext) GetTargetPos() int {
	return ctx.GetIntOrDefault(KeyTargetPos, 0)
}

// ========== Decision Setup ==========

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

// ========== Timing Setup ==========

// WithTimeout sets timeout duration.
func (ctx *StateContext) WithTimeout(timeout time.Duration) *StateContext {
	ctx.Timeout = timeout
	return ctx
}

// ========== Stack Setup ==========

// WithStack sets the state stack reference.
func (ctx *StateContext) WithStack(stack *StateStack) *StateContext {
	ctx.Stack = stack
	return ctx
}

// ========== State Markers (stored in Metadata) ==========

// IsSkipTurn checks if turn should be skipped.
func (ctx *StateContext) IsSkipTurn() bool {
	return ctx.GetBoolOrDefault(KeySkipTurn, false)
}

// SetSkipTurn sets the skip turn marker.
func (ctx *StateContext) SetSkipTurn(skip bool) {
	ctx.SetBool(KeySkipTurn, skip)
}

// IsFellDown checks if player fell from Fragile cell.
func (ctx *StateContext) IsFellDown() bool {
	return ctx.GetBoolOrDefault(KeyFellDown, false)
}

// SetFellDown sets the fell down marker.
func (ctx *StateContext) SetFellDown(fell bool) {
	ctx.SetBool(KeyFellDown, fell)
}

// HasReachedEnd checks if player reached Boss cell.
func (ctx *StateContext) HasReachedEnd() bool {
	return ctx.GetBoolOrDefault(KeyReachedEnd, false)
}

// SetReachedEnd sets the reached end marker.
func (ctx *StateContext) SetReachedEnd(reached bool) {
	ctx.SetBool(KeyReachedEnd, reached)
}

// ========== Mini-Game Results ==========

// SetMiniGameRank sets player's mini-game ranking.
func (ctx *StateContext) SetMiniGameRank(playerID string, rank int) {
	ctx.SetInt("result_"+playerID, rank)
}

// GetMiniGameRank retrieves player's mini-game ranking.
func (ctx *StateContext) GetMiniGameRank(playerID string) int {
	return ctx.GetIntOrDefault("result_"+playerID, 0)
}

// SetDiceType sets player's dice type based on ranking.
func (ctx *StateContext) SetDiceType(playerID string, diceType string) {
	ctx.SetString("dice_"+playerID, diceType)
}

// GetDiceType retrieves player's dice type.
func (ctx *StateContext) GetDiceType(playerID string) string {
	return ctx.GetStringOrDefault("dice_"+playerID, "wood")
}

// ========== Lifecycle ==========

// Clear resets the context to default values.
func (ctx *StateContext) Clear() {
	ctx.Phase = event.Phase(0)
	ctx.PhaseData = nil
	ctx.Decision = nil
	ctx.Decisions = make([]*event.Decision, 0)
	ctx.Timeout = 0
	ctx.Success = true
	ctx.Error = nil
	ctx.Metadata.Clear()
}