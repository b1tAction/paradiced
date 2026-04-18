package hsm

import (
	"time"

	pkgnet "github.com/b1tAction/paradiced/pkg/net"

	"github.com/b1tAction/paradiced/internal/core"
	"github.com/b1tAction/paradiced/internal/engine"
	"github.com/b1tAction/paradiced/internal/event"
	"github.com/b1tAction/paradiced/internal/gamemap"
	"github.com/b1tAction/paradiced/pkg/constants"
	"github.com/b1tAction/paradiced/pkg/rng"
	"github.com/b1tAction/paradiced/pkg/util"
)

// Context key constants for commonly used state markers.
// These keys are stored in the embedded Metadata for type-safe access.
const (
	KeySkipTurn     = "skip_turn"           // Mark to skip remaining turn
	KeyFellDown     = "fell_down"           // Mark for Fragile cell fall
	KeyReachedEnd   = "reached_end"         // Mark for reaching Boss cell
	KeyDiceSteps    = "dice_steps"          // Dice roll result for movement
	KeyTargetPos    = "target_pos"          // Target position after movement
	KeyDamage       = "damage"              // Damage amount
	KeyMiniGameRank = "mini_game_rank"      // Mini-game ranking result prefix (used as "result_{playerID}")
	KeyDiceType     = "dice_type"           // Dice type prefix (used as "dice_{playerID}")
	KeyBossTrigger  = "boss_trigger_player" // Player who triggered boss battle
	KeyWinner       = "winner_id"           // Winner player ID

	// State flow markers
	KeyInitialized       = "initialized"         // Match initialized flag
	KeyMiniGameStarted   = "mini_game_started"   // Mini-game phase started
	KeyWaitingForResults = "waiting_for_results" // Waiting for mini-game results
	KeyTurnLoopActive    = "turn_loop_active"    // Turn loop active flag
	KeyBossBattleActive  = "boss_battle_active"  // Boss battle active flag
	KeyGameOver          = "game_over"           // Game over flag
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
// Uses HSM as single source of truth for game data access.
// Embeds util.Metadata for extensible type-safe key-value storage.
type StateContext struct {
	// Embedded Metadata for extensible storage
	*util.Metadata

	// ========== Single Source of Truth ==========
	// HSM reference provides access to Game, Bus, MapEngine via getter methods
	HSM *HSM

	// ========== Current Player ==========
	Player *core.Player // Current player (direct access, used in Layer 2 states)

	// ========== Broadcast Adapter ==========
	// Broadcast adapter for client communication (set by HSM)
	Broadcast pkgnet.BroadcastAdapter

	// ========== Phase Triggering ==========
	Phase     constants.Phase // Current phase to trigger
	PhaseData interface{}     // Additional phase data (e.g., damage amount, dice steps)

	// ========== Decision Handling ==========
	Decision  *event.Decision   // Pending decision requiring user input
	Decisions []*event.Decision // List of pending decisions

	// ========== Timing ==========
	Timeout   time.Duration // Timeout duration for waiting states
	StartTime time.Time     // State entry time

	// ========== Stack Reference (Layer 3) ==========
	Stack *StateStack // Reference to interrupt stack

	// ========== Execution Result ==========
	Success bool  // Whether state execution succeeded
	Error   error // Error if state execution failed
}

// NewStateContext creates a new StateContext with default values.
func NewStateContext() *StateContext {
	return &StateContext{
		Metadata:  util.NewMetadata(),
		Decisions: make([]*event.Decision, 0),
		Success:   true,
	}
}

// ========== HSM Setup ==========

// WithHSM sets the HSM reference (single source of truth).
func (ctx *StateContext) WithHSM(hsm *HSM) *StateContext {
	ctx.HSM = hsm
	return ctx
}

// ========== Convenience Access Methods ==========

// GetGame returns the game instance via HSM.
func (ctx *StateContext) GetGame() *engine.Game {
	if ctx.HSM == nil {
		return nil
	}
	return ctx.HSM.GetGame()
}

// GetBus returns the EventBus via HSM.
func (ctx *StateContext) GetBus() *event.EventBus {
	if ctx.HSM == nil {
		return nil
	}
	return ctx.HSM.GetBus()
}

// GetMapEngine returns the MapEngine via HSM.
func (ctx *StateContext) GetMapEngine() *gamemap.MapEngine {
	if ctx.HSM == nil {
		return nil
	}
	return ctx.HSM.GetMapEngine()
}

// GetRound returns the current round number via HSM.
func (ctx *StateContext) GetRound() int {
	if ctx.HSM == nil {
		return 0
	}
	return ctx.HSM.GetRound()
}

// GetTurn returns the current turn (player index) via HSM.
func (ctx *StateContext) GetTurn() int {
	if ctx.HSM == nil {
		return 0
	}
	return ctx.HSM.GetTurn()
}

// IncrementRound increments the round counter via HSM.
func (ctx *StateContext) IncrementRound() {
	if ctx.HSM != nil {
		ctx.HSM.IncrementRound()
	}
}

// SetTurn sets the current turn index via HSM.
// HSM is the single source of truth for turn state.
func (ctx *StateContext) SetTurn(turn int) {
	if ctx.HSM != nil {
		ctx.HSM.turn = turn
	}
}

// ========== Player Setup ==========

// WithPlayer sets the player (direct type).
func (ctx *StateContext) WithPlayer(player *core.Player) *StateContext {
	ctx.Player = player
	return ctx
}

// ========== Broadcast Setup ==========

// WithBroadcast sets the Broadcast adapter for client communication.
func (ctx *StateContext) WithBroadcast(adapter pkgnet.BroadcastAdapter) *StateContext {
	ctx.Broadcast = adapter
	return ctx
}

// ========== Phase Setup ==========

// WithPhase sets the phase and optional data.
func (ctx *StateContext) WithPhase(phase constants.Phase, data interface{}) *StateContext {
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
func (ctx *StateContext) SetDiceType(playerID string, diceType rng.DiceType) {
	ctx.SetInt("dice_"+playerID, int(diceType))
}

// GetDiceType retrieves player's dice type.
// Returns DiceTypeWood if player has no assigned dice.
func (ctx *StateContext) GetDiceType(playerID string) rng.DiceType {
	val := ctx.GetIntOrDefault("dice_"+playerID, int(rng.DiceTypeWood))
	return rng.DiceType(val)
}

// ========== Lifecycle ==========

// Clear resets the context to default values.
func (ctx *StateContext) Clear() {
	ctx.Phase = ""
	ctx.PhaseData = nil
	ctx.Decision = nil
	ctx.Decisions = make([]*event.Decision, 0)
	ctx.Timeout = 0
	ctx.Success = true
	ctx.Error = nil
	ctx.Metadata.Clear()
}