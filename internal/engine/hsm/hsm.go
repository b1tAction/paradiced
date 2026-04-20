package hsm

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/b1tAction/paradiced/internal/core"
	"github.com/b1tAction/paradiced/internal/engine"
	"github.com/b1tAction/paradiced/internal/event"
	"github.com/b1tAction/paradiced/internal/gamemap"
	"github.com/b1tAction/paradiced/pkg/id"
)

// HSMConfig holds configuration for the HSM.
type HSMConfig struct {
	// MainActionTimeout is the timeout for MainAction state (default: 45s).
	// In dev mode (PD_DEV=true), defaults to 10s.
	MainActionTimeout time.Duration
}

// DefaultHSMConfig returns the default HSM configuration.
func DefaultHSMConfig() *HSMConfig {
	// Check if running in dev mode
	isDev, _ := strconv.ParseBool(os.Getenv("PD_DEV"))

	mainActionTimeout := 45 * time.Second
	if isDev {
		mainActionTimeout = 10 * time.Second
	}

	return &HSMConfig{
		MainActionTimeout: mainActionTimeout,
	}
}

// HSM is the main Hierarchical State Machine structure.
// It manages three layers of states: Global (Layer 1), Turn (Layer 2), and Interrupt (Layer 3).
type HSM struct {
	// ========== Layer 1: Global State ==========
	globalState   State
	globalStateID StateID

	// ========== Layer 2: Turn State ==========
	turnState   State
	turnStateID StateID
	turnPlayer  *core.Player // Current player in turn (direct type)

	// ========== Layer 3: Interrupt Stack ==========
	stack        *StateStack
	waitingState State           // Current WaitDecision state (if active)
	decision     *event.Decision // Current pending decision

	// ========== State Registry ==========
	states  map[StateID]State // All registered states
	factory StateFactory      // State factory for creating instances

	// ========== Core References ==========
	game      *engine.Game       // Game instance (single source of truth)
	bus       *event.EventBus    // EventBus (derived from game)
	mapEngine *gamemap.MapEngine // MapEngine (set externally)

	// ========== Configuration ==========
	config *HSMConfig // HSM configuration

	// ========== Round/Turn State (moved from GameState) ==========
	round int // Current round number
	turn  int // Current turn (player index)

	// ========== Timing ==========
	lastUpdate     time.Time // Last update timestamp
	stateEnterTime time.Time // Time when current state was entered

	// ========== Flow Control ==========
	running bool // HSM is running
	paused  bool // HSM is paused (e.g., waiting for decision)
}

// NewHSM creates a new HSM instance.
func NewHSM(game *engine.Game) *HSM {
	return &HSM{
		globalStateID: StateNone,
		turnStateID:   StateNone,
		stack:         NewStateStack(),
		states:        make(map[StateID]State),
		game:          game,
		bus:           game.Bus,
		config:        DefaultHSMConfig(), // Use default config
		round:         1,                  // Start at round 1
		turn:          0,                  // Start at turn 0 (first player)
		running:       false,
		paused:        false,
	}
}

// WithConfig sets the HSM configuration.
func (hsm *HSM) WithConfig(config *HSMConfig) *HSM {
	hsm.config = config
	return hsm
}

// ========== State Registration ==========

// RegisterState registers a state instance with the HSM.
func (hsm *HSM) RegisterState(state State) error {
	if state == nil {
		return errors.New("state is nil")
	}
	id := state.ID()
	if !id.IsValid() {
		return errors.New("invalid state ID: " + id.String())
	}
	if _, exists := hsm.states[id]; exists {
		return errors.New("state already registered: " + id.String())
	}
	hsm.states[id] = state
	return nil
}

// RegisterStates registers multiple state instances.
func (hsm *HSM) RegisterStates(states []State) error {
	for _, state := range states {
		if err := hsm.RegisterState(state); err != nil {
			return err
		}
	}
	return nil
}

// SetFactory sets the state factory for creating state instances.
func (hsm *HSM) SetFactory(factory StateFactory) {
	hsm.factory = factory
}

// GetState retrieves a registered state by ID.
func (hsm *HSM) GetState(id StateID) State {
	if state, ok := hsm.states[id]; ok {
		return state
	}
	// Try factory if state not registered
	if hsm.factory != nil {
		return hsm.factory.CreateState(id)
	}
	return nil
}

// ========== State Access ==========

// GetGlobalStateID returns current global state ID.
func (hsm *HSM) GetGlobalStateID() StateID {
	return hsm.globalStateID
}

// GetGlobalState returns current global state instance.
func (hsm *HSM) GetGlobalState() State {
	return hsm.globalState
}

// GetTurnStateID returns current turn state ID (Layer 2).
func (hsm *HSM) GetTurnStateID() StateID {
	return hsm.turnStateID
}

// GetTurnState returns current turn state instance.
func (hsm *HSM) GetTurnState() State {
	return hsm.turnState
}

// GetTurnPlayer returns current player in turn (direct type).
func (hsm *HSM) GetTurnPlayer() *core.Player {
	return hsm.turnPlayer
}

// GetGame returns the game instance (direct type).
func (hsm *HSM) GetGame() *engine.Game {
	return hsm.game
}

// GetBus returns the EventBus (direct type).
func (hsm *HSM) GetBus() *event.EventBus {
	return hsm.bus
}

// GetMapEngine returns the MapEngine (direct type).
func (hsm *HSM) GetMapEngine() *gamemap.MapEngine {
	return hsm.mapEngine
}

// GetRound returns the current round number.
func (hsm *HSM) GetRound() int {
	return hsm.round
}

// GetTurn returns the current turn (player index).
func (hsm *HSM) GetTurn() int {
	return hsm.turn
}

// IncrementRound increments the round number.
func (hsm *HSM) IncrementRound() {
	hsm.round++
}

// NextTurn advances to the next player turn.
// Returns true if round wrapped around (all players completed).
func (hsm *HSM) NextTurn() bool {
	hsm.turn++
	if hsm.turn >= len(hsm.game.Players) {
		hsm.turn = 0
		hsm.round++
		return true // Round wrapped
	}
	return false
}

// SetMapEngine sets the MapEngine (direct type).
func (hsm *HSM) SetMapEngine(engine *gamemap.MapEngine) {
	hsm.mapEngine = engine
}

// GetCurrentStateID returns the currently active state ID.
// Returns the interrupt state if active, otherwise turn state, otherwise global state.
func (hsm *HSM) GetCurrentStateID() StateID {
	if hsm.paused && hsm.waitingState != nil {
		return hsm.waitingState.ID()
	}
	if hsm.turnState != nil {
		return hsm.turnStateID
	}
	return hsm.globalStateID
}

// GetStack returns the interrupt state stack.
func (hsm *HSM) GetStack() *StateStack {
	return hsm.stack
}

// GetCurrentDecision returns current pending decision (if waiting).
func (hsm *HSM) GetCurrentDecision() *event.Decision {
	return hsm.decision
}

// ========== State Transitions ==========

// TransitionTo transitions to a new state.
// Handles Enter/Exit lifecycle and validates transition rules.
func (hsm *HSM) TransitionTo(targetID StateID, ctx *StateContext) error {
	if !targetID.IsValid() {
		return errors.New("invalid target state ID: " + targetID.String())
	}

	target := hsm.GetState(targetID)
	if target == nil {
		return errors.New("state not found: " + targetID.String())
	}

	// Log state transition
	if hsm.game != nil && hsm.game.Log != nil {
		fromID := hsm.GetCurrentStateID()
		playerID := ""
		if hsm.turnPlayer != nil {
			playerID = hsm.turnPlayer.ID.UUID()
		}
		hsm.game.Log.LogStateTransition(fromID.String(), targetID.String(), playerID)
	}

	// Determine layer and handle transition
	switch targetID.Layer() {
	case 1:
		return hsm.transitionGlobal(target, ctx)
	case 2:
		return hsm.transitionTurn(target, ctx)
	case 3:
		return hsm.transitionInterrupt(target, ctx)
	default:
		return errors.New("unknown state layer: " + targetID.String())
	}
}

// transitionGlobal handles Layer 1 (Global) state transitions.
func (hsm *HSM) transitionGlobal(target State, ctx *StateContext) error {
	// Exit current global state if exists
	if hsm.globalState != nil {
		if !hsm.globalState.CanTransitionTo(target.ID()) {
			return fmt.Errorf("invalid transition: %s -> %s", hsm.globalStateID.String(), target.ID().String())
		}
		hsm.globalState.Exit(ctx)
	}

	// Clear turn state when changing global state
	if hsm.turnState != nil {
		hsm.turnState.Exit(ctx)
		hsm.turnState = nil
		hsm.turnStateID = StateNone
		hsm.turnPlayer = nil
	}

	// Enter new global state
	hsm.globalState = target
	hsm.globalStateID = target.ID()
	hsm.stateEnterTime = time.Now()

	if ctx == nil {
		ctx = NewStateContext().WithHSM(hsm)
	}
	ctx.StartTime = hsm.stateEnterTime
	target.Enter(ctx)

	// Check if state wants immediate transition (auto-proceed)
	nextID := target.Update(ctx)
	if nextID != StateNone && nextID != target.ID() {
		return hsm.TransitionTo(nextID, ctx)
	}

	return nil
}

// transitionTurn handles Layer 2 (Turn) state transitions.
func (hsm *HSM) transitionTurn(target State, ctx *StateContext) error {
	// Must be in TurnLoop global state
	if hsm.globalStateID != StateTurnLoop {
		return errors.New("turn state transition requires TurnLoop global state")
	}

	// Exit current turn state if exists
	if hsm.turnState != nil {
		if !hsm.turnState.CanTransitionTo(target.ID()) {
			return fmt.Errorf("invalid transition: %s -> %s", hsm.turnStateID.String(), target.ID().String())
		}
		hsm.turnState.Exit(ctx)
	}

	// Enter new turn state
	hsm.turnState = target
	hsm.turnStateID = target.ID()
	hsm.stateEnterTime = time.Now()

	if ctx == nil {
		ctx = NewStateContext().WithHSM(hsm)
	}
	// Always set Player from HSM's turn player
	ctx.WithPlayer(hsm.turnPlayer)
	ctx.StartTime = hsm.stateEnterTime

	fmt.Printf("[hsm] transitionTurn: target=%s, dice_steps=%d\n", target.ID().String(), ctx.GetDiceSteps())

	target.Enter(ctx)

	// Check for decisions requiring user input
	if len(ctx.Decisions) > 0 {
		return hsm.PushInterrupt(target, ctx)
	}

	// Check if state wants immediate transition
	nextID := target.Update(ctx)
	if nextID != StateNone && nextID != target.ID() {
		return hsm.TransitionTo(nextID, ctx)
	}

	return nil
}

// transitionInterrupt handles Layer 3 (Interrupt) state transitions.
func (hsm *HSM) transitionInterrupt(target State, ctx *StateContext) error {
	// Must have state to push onto stack
	if hsm.stack.IsEmpty() && hsm.turnState == nil && hsm.globalState == nil {
		return errors.New("interrupt state requires parent state on stack")
	}

	// Enter interrupt state (WaitDecision)
	hsm.waitingState = target
	hsm.decision = ctx.GetDecision()
	hsm.paused = true
	hsm.stateEnterTime = time.Now()

	ctx.StartTime = hsm.stateEnterTime
	target.Enter(ctx)

	return nil
}

// PushInterrupt pushes current state onto stack and enters WaitDecision.
func (hsm *HSM) PushInterrupt(currentState State, ctx *StateContext) error {
	// Push current state onto stack
	if err := hsm.stack.Push(currentState, ctx); err != nil {
		return err
	}

	// Transition to WaitDecision
	waitState := hsm.GetState(StateWaitDecision)
	if waitState == nil {
		return errors.New("WaitDecision state not registered")
	}

	waitCtx := NewStateContext().
		WithHSM(hsm).
		WithPlayer(hsm.turnPlayer).
		WithDecisions(ctx.Decisions).
		WithTimeout(ctx.Timeout).
		WithStack(hsm.stack)

	return hsm.transitionInterrupt(waitState, waitCtx)
}

// PopInterrupt pops state from stack and resumes execution.
func (hsm *HSM) PopInterrupt(ctx *StateContext) error {
	// Pop from stack
	entry, err := hsm.stack.Pop()
	if err != nil {
		return err
	}

	// Exit WaitDecision
	if hsm.waitingState != nil {
		hsm.waitingState.Exit(ctx)
	}

	// Clear waiting state
	hsm.waitingState = nil
	hsm.decision = nil
	hsm.paused = false

	// Restore previous state
	switch entry.StateID.Layer() {
	case 1:
		hsm.globalState = entry.State
		hsm.globalStateID = entry.StateID
	case 2:
		hsm.turnState = entry.State
		hsm.turnStateID = entry.StateID
	}

	// Continue execution with restored context
	entry.Context.StartTime = time.Now()
	nextID := entry.State.Update(entry.Context)
	if nextID != StateNone && nextID != entry.StateID {
		return hsm.TransitionTo(nextID, entry.Context)
	}

	return nil
}

// ========== Lifecycle ==========

// Start begins the HSM execution, entering initial state.
func (hsm *HSM) Start(initialStateID StateID, ctx *StateContext) error {
	if hsm.running {
		return errors.New("HSM is already running")
	}

	hsm.running = true
	return hsm.TransitionTo(initialStateID, ctx)
}

// Stop halts the HSM execution.
func (hsm *HSM) Stop(ctx *StateContext) {
	if !hsm.running {
		return
	}

	// Exit all active states
	if hsm.waitingState != nil {
		hsm.waitingState.Exit(ctx)
	}
	if hsm.turnState != nil {
		hsm.turnState.Exit(ctx)
	}
	if hsm.globalState != nil {
		hsm.globalState.Exit(ctx)
	}

	// Clear stack
	hsm.stack.Clear()

	hsm.running = false
	hsm.paused = false
}

// Update advances the HSM by one tick.
// Returns the next state ID if a transition occurred.
func (hsm *HSM) Update(ctx *StateContext) (StateID, error) {
	if !hsm.running {
		return StateNone, errors.New("HSM is not running")
	}
	if hsm.paused {
		// Still in waiting state, check timeout
		return hsm.updateWaiting(ctx)
	}

	hsm.lastUpdate = time.Now()

	// Determine which state to update
	if hsm.turnState != nil {
		nextID := hsm.turnState.Update(ctx)
		if nextID != StateNone && nextID != hsm.turnStateID {
			return nextID, hsm.TransitionTo(nextID, ctx)
		}
	} else if hsm.globalState != nil {
		nextID := hsm.globalState.Update(ctx)
		if nextID != StateNone && nextID != hsm.globalStateID {
			return nextID, hsm.TransitionTo(nextID, ctx)
		}
	}

	return StateNone, nil
}

// updateWaiting handles updates while in waiting state.
func (hsm *HSM) updateWaiting(ctx *StateContext) (StateID, error) {
	if hsm.waitingState == nil {
		return StateNone, errors.New("no waiting state active")
	}

	nextID := hsm.waitingState.Update(ctx)
	if nextID == StateNone {
		// Still waiting
		return StateNone, nil
	}

	// Decision handled or timeout, pop interrupt
	return nextID, hsm.PopInterrupt(ctx)
}

// OnUserChoice handles user input for pending decision.
func (hsm *HSM) OnUserChoice(choice int, ctx *StateContext) error {
	if hsm.decision == nil {
		return errors.New("no pending decision")
	}

	// Execute decision action
	if ctx == nil {
		ctx = NewStateContext().WithHSM(hsm).WithPlayer(hsm.turnPlayer)
	}

	execCtx := hsm.resolveDecisionExecutionContext(ctx)
	hsm.decision.Execute(choice, execCtx)
	runDerived(execCtx)
	hsm.clearPendingDecisionEventContext(ctx)

	// Pop interrupt and resume
	return hsm.PopInterrupt(ctx)
}

func (hsm *HSM) resolveDecisionExecutionContext(ctx *StateContext) *event.Context {
	if pending := getPendingCtx(ctx); pending != nil {
		return pending
	}

	entry := hsm.stack.Peek()
	if entry != nil {
		if pending := getPendingCtx(entry.Context); pending != nil {
			return pending
		}
	}

	return event.NewContext(hsm.turnPlayer)
}

func (hsm *HSM) clearPendingDecisionEventContext(ctx *StateContext) {
	if ctx != nil {
		ctx.Delete(KeyPendingCtx)
	}

	entry := hsm.stack.Peek()
	if entry != nil && entry.Context != nil {
		entry.Context.Delete(KeyPendingCtx)
	}
}

// ========== Turn Player Management ==========

// SetTurnPlayer sets the current player for turn state (direct type).
func (hsm *HSM) SetTurnPlayer(player *core.Player) {
	hsm.turnPlayer = player
}

// NextTurnPlayer advances to the next player in turn queue.
// Returns the new turn player.
func (hsm *HSM) NextTurnPlayer() *core.Player {
	// Use HSM's own turn management
	hsm.NextTurn()
	// Get new current player
	if hsm.turn < len(hsm.game.Players) {
		hsm.turnPlayer = hsm.game.Players[hsm.turn]
	} else {
		hsm.turnPlayer = nil
	}
	return hsm.turnPlayer
}

// ========== Status Checks ==========

// IsRunning checks if HSM is running.
func (hsm *HSM) IsRunning() bool {
	return hsm.running
}

// IsPaused checks if HSM is paused (waiting for decision).
func (hsm *HSM) IsPaused() bool {
	return hsm.paused
}

// IsWaiting checks if HSM has pending decision.
func (hsm *HSM) IsWaiting() bool {
	return hsm.decision != nil
}

// IsInTurn checks if HSM is in turn loop with active turn state.
func (hsm *HSM) IsInTurn() bool {
	return hsm.globalStateID == StateTurnLoop && hsm.turnStateID != StateNone
}

// ========== Turn State Input Handling ==========

// OnRollDice handles dice roll input during MainActionState.
func (hsm *HSM) OnRollDice(steps int, ctx *StateContext) error {
	// Must be in MainAction state
	if hsm.turnStateID != StateMainAction {
		return errors.New("OnRollDice requires MainAction state")
	}

	// Create context if not provided
	if ctx == nil {
		ctx = NewStateContext().WithHSM(hsm).WithPlayer(hsm.turnPlayer)
	}

	// Call OnRollDice on the turn state if it implements RollDiceHandler
	if handler, ok := hsm.turnState.(RollDiceHandler); ok {
		handler.OnRollDice(ctx, steps)
	} else {
		return errors.New("current turn state does not handle RollDice")
	}

	// Trigger update to check for state transition
	nextID := hsm.turnState.Update(ctx)
	fmt.Printf("[hsm] OnRollDice: diceRolled=%v, steps=%d, nextID=%s, currentTurnStateID=%s\n",
		hsm.turnState.(*MainActionState).diceRolled, steps, nextID.String(), hsm.turnStateID.String())
	if nextID != StateNone && nextID != hsm.turnStateID {
		fmt.Printf("[hsm] OnRollDice: transitioning to %s\n", nextID.String())
		return hsm.TransitionTo(nextID, ctx)
	}

	return nil
}

// OnUseItem handles item usage input during MainActionState.
func (hsm *HSM) OnUseItem(itemID string, ctx *StateContext) error {
	// Must be in MainAction state
	if hsm.turnStateID != StateMainAction {
		return errors.New("OnUseItem requires MainAction state")
	}

	// Create context if not provided
	if ctx == nil {
		ctx = NewStateContext().WithHSM(hsm).WithPlayer(hsm.turnPlayer)
	}

	// Call OnUseItem on the turn state if it implements UseItemHandler
	if handler, ok := hsm.turnState.(UseItemHandler); ok {
		handler.OnUseItem(ctx, itemID)
	} else {
		return errors.New("current turn state does not handle UseItem")
	}

	// Trigger update to check for state transition
	nextID := hsm.turnState.Update(ctx)
	if nextID != StateNone && nextID != hsm.turnStateID {
		return hsm.TransitionTo(nextID, ctx)
	}

	return nil
}

// RollDiceHandler is an interface for states that handle dice roll input.
type RollDiceHandler interface {
	OnRollDice(ctx *StateContext, steps int)
}

// UseItemHandler is an interface for states that handle item usage input.
type UseItemHandler interface {
	OnUseItem(ctx *StateContext, itemID string)
}

// ========== Mini-Game Result Handler ==========

// OnMiniGameResult handles mini-game result submission.
// Must be called when in RoundMiniGame state.
func (hsm *HSM) OnMiniGameResult(playerID string, rank int, ctx *StateContext) error {
	// Must be in RoundMiniGame state
	if hsm.globalStateID != StateRoundMiniGame {
		return errors.New("OnMiniGameResult requires RoundMiniGame state")
	}

	// Get the global state
	globalState := hsm.GetGlobalState()
	if globalState == nil {
		return errors.New("no global state active")
	}

	// Type assertion to RoundMiniGameState
	miniGameState, ok := globalState.(*RoundMiniGameState)
	if !ok {
		return errors.New("current global state is not RoundMiniGameState")
	}

	// Create context if not provided
	if ctx == nil {
		ctx = NewStateContext().WithHSM(hsm)
	}

	// Call the state's OnMiniGameResult method
	miniGameState.OnMiniGameResult(ctx, playerID, rank)

	// Trigger update to check for state transition
	nextID := globalState.Update(ctx)
	if nextID != StateNone && nextID != hsm.globalStateID {
		return hsm.TransitionTo(nextID, ctx)
	}

	return nil
}

// ========== Snapshot ==========

// CreateSnapshot creates a snapshot of current HSM state.
func (hsm *HSM) CreateSnapshot() *HSMSnapshot {
	return &HSMSnapshot{
		GlobalStateID:   hsm.globalStateID,
		TurnStateID:     hsm.turnStateID,
		TurnPlayerID:    getPlayerID(hsm.turnPlayer),
		InterruptStack:  hsm.stack.GetStackIDs(),
		CurrentDecision: snapshotDecision(hsm.decision),
		Running:         hsm.running,
		Paused:          hsm.paused,
		EnterTime:       hsm.stateEnterTime,
	}
}

// RestoreFromSnapshot restores HSM state from snapshot.
func (hsm *HSM) RestoreFromSnapshot(snapshot *HSMSnapshot) error {
	if snapshot == nil {
		return errors.New("snapshot is nil")
	}

	// Restore global state
	if snapshot.GlobalStateID != StateNone {
		state := hsm.GetState(snapshot.GlobalStateID)
		if state == nil {
			return errors.New("global state not found: " + snapshot.GlobalStateID.String())
		}
		hsm.globalState = state
		hsm.globalStateID = snapshot.GlobalStateID
	}

	// Restore turn state
	if snapshot.TurnStateID != StateNone {
		state := hsm.GetState(snapshot.TurnStateID)
		if state == nil {
			return errors.New("turn state not found: " + snapshot.TurnStateID.String())
		}
		hsm.turnState = state
		hsm.turnStateID = snapshot.TurnStateID
		// Parse TurnPlayerID string to id.PlayerID
		if snapshot.TurnPlayerID != "" {
			parsedID, err := id.ParsePlayerID(snapshot.TurnPlayerID)
			if err == nil {
				hsm.turnPlayer = hsm.game.GetPlayer(parsedID)
			}
		}
	}

	// Restore interrupt stack
	if len(snapshot.InterruptStack) > 0 {
		if err := hsm.stack.RestoreFromIDs(snapshot.InterruptStack, hsm.factory); err != nil {
			return err
		}
	}

	// Restore decision
	if snapshot.CurrentDecision != nil {
		hsm.decision = restoreDecision(snapshot.CurrentDecision)
	}

	hsm.running = snapshot.Running
	hsm.paused = snapshot.Paused
	hsm.stateEnterTime = snapshot.EnterTime

	return nil
}

// ========== Helper Functions ==========

func getPlayerID(player *core.Player) string {
	if player == nil {
		return ""
	}
	return player.ID.UUID()
}

func snapshotDecision(decision *event.Decision) *DecisionSnapshot {
	if decision == nil {
		return nil
	}
	return &DecisionSnapshot{
		ID:      decision.ID.UUID(),
		Prompt:  decision.Prompt,
		Default: decision.Default,
	}
}

func restoreDecision(snapshot *DecisionSnapshot) *event.Decision {
	if snapshot == nil {
		return nil
	}
	// Create decision with placeholder options (will be restored on context)
	return event.NewDecision(snapshot.Prompt, nil)
}

// ========== Snapshot Types ==========

// HSMSnapshot represents a snapshot of HSM state for persistence.
type HSMSnapshot struct {
	GlobalStateID   StateID
	TurnStateID     StateID
	TurnPlayerID    string
	InterruptStack  []StateID
	CurrentDecision *DecisionSnapshot
	Running         bool
	Paused          bool
	EnterTime       time.Time
}

// DecisionSnapshot represents a snapshot of pending decision.
type DecisionSnapshot struct {
	ID      string
	Prompt  string
	Default int
}

// String returns a human-readable representation of current HSM state.
func (hsm *HSM) String() string {
	result := fmt.Sprintf("HSM[Global=%s", hsm.globalStateID.String())
	if hsm.turnStateID != StateNone {
		result += fmt.Sprintf(", Turn=%s", hsm.turnStateID.String())
	}
	if hsm.paused {
		result += ", Paused"
	}
	if hsm.stack.Depth() > 0 {
		result += fmt.Sprintf(", Stack=%s", hsm.stack.String())
	}
	result += "]"
	return result
}
