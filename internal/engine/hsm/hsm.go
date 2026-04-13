package hsm

import (
	"errors"
	"fmt"
	"time"

	"github.com/b1tAction/Fated/pkg/event"
)

// HSM is the main Hierarchical State Machine structure.
// It manages three layers of states: Global (Layer 1), Turn (Layer 2), and Interrupt (Layer 3).
type HSM struct {
	// ========== Layer 1: Global State ==========
	globalState   State
	globalStateID StateID

	// ========== Layer 2: Turn State ==========
	turnState     State
	turnStateID   StateID
	turnPlayer    PlayerAdapter // Current player in turn

	// ========== Layer 3: Interrupt Stack ==========
	stack         *StateStack
	waitingState  State         // Current WaitDecision state (if active)
	decision      *event.Decision // Current pending decision

	// ========== State Registry ==========
	states       map[StateID]State // All registered states
	factory      StateFactory      // State factory for creating instances

	// ========== Game Reference ==========
	game         GameAdapter       // Game adapter for state operations

	// ========== Timing ==========
	lastUpdate   time.Time         // Last update timestamp
	stateEnterTime time.Time       // Time when current state was entered

	// ========== Flow Control ==========
	running      bool              // HSM is running
	paused       bool              // HSM is paused (e.g., waiting for decision)
}

// NewHSM creates a new HSM instance.
func NewHSM(game GameAdapter) *HSM {
	return &HSM{
		globalStateID: StateNone,
		turnStateID:   StateNone,
		stack:         NewStateStack(),
		states:        make(map[StateID]State),
		game:          game,
		running:       false,
		paused:        false,
	}
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

// GetTurnPlayer returns current player in turn.
func (hsm *HSM) GetTurnPlayer() PlayerAdapter {
	return hsm.turnPlayer
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
		ctx = NewStateContext().WithGame(hsm.game)
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
		ctx = NewStateContext().WithGame(hsm.game).WithPlayer(hsm.turnPlayer)
	}
	ctx.StartTime = hsm.stateEnterTime
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
	hsm.decision = ctx.Decision
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
		WithGame(hsm.game).
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
		ctx = NewStateContext().WithGame(hsm.game).WithPlayer(hsm.turnPlayer)
	}
	hsm.decision.Execute(choice, event.NewContext(nil))

	// Pop interrupt and resume
	return hsm.PopInterrupt(ctx)
}

// ========== Turn Player Management ==========

// SetTurnPlayer sets the current player for turn state.
func (hsm *HSM) SetTurnPlayer(player PlayerAdapter) {
	hsm.turnPlayer = player
}

// NextTurnPlayer advances to the next player in turn queue.
func (hsm *HSM) NextTurnPlayer() PlayerAdapter {
	hsm.game.NextTurn()
	hsm.turnPlayer = hsm.game.GetCurrentPlayer()
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
		hsm.turnPlayer = hsm.game.GetPlayer(snapshot.TurnPlayerID)
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

func getPlayerID(player PlayerAdapter) string {
	if player == nil {
		return ""
	}
	return player.GetUserID()
}

func snapshotDecision(decision *event.Decision) *DecisionSnapshot {
	if decision == nil {
		return nil
	}
	return &DecisionSnapshot{
		ID:      decision.ID,
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