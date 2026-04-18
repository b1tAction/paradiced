package hsm

import (
	"time"

	"github.com/b1tAction/paradiced/internal/event"
)

// ========== Interrupt States (Layer 3) ==========

// BaseInterruptState provides common functionality for interrupt layer states.
// Interrupt states can be pushed onto the StateStack and temporarily
// suspend the current turn state execution.
type BaseInterruptState struct {
	id StateID
}

// ID returns the state identifier.
func (s *BaseInterruptState) ID() StateID {
	return s.id
}

// CanTransitionTo defines valid transition rules for interrupt states.
// Interrupt states typically return to StateNone to signal PopInterrupt.
func (s *BaseInterruptState) CanTransitionTo(target StateID) bool {
	// Interrupt states don't transition to other states directly.
	// They signal completion via returning StateNone, which triggers Pop.
	return false
}

// ========== WaitDecisionState ==========

// WaitDecisionState handles user decision input with timeout support.
// This state is pushed onto the interrupt stack when a Phase triggers
// a decision that requires user confirmation.
type WaitDecisionState struct {
	BaseInterruptState
	decision      *event.Decision
	timeout       time.Duration
	startTime     time.Time
	defaultOption int
	completed     bool
	choice        int
}

// NewWaitDecisionState creates a new WaitDecision state.
func NewWaitDecisionState() *WaitDecisionState {
	return &WaitDecisionState{
		BaseInterruptState: BaseInterruptState{id: StateWaitDecision},
		timeout:            30 * time.Second, // Default timeout
		defaultOption:      0,                // Default to first option
		completed:          false,
	}
}

// WithDecision configures the decision to wait for.
func (s *WaitDecisionState) WithDecision(decision *event.Decision) *WaitDecisionState {
	s.decision = decision
	return s
}

// WithTimeout sets the timeout duration.
func (s *WaitDecisionState) WithTimeout(timeout time.Duration) *WaitDecisionState {
	s.timeout = timeout
	return s
}

// WithDefaultOption sets the default option index for timeout.
func (s *WaitDecisionState) WithDefaultOption(option int) *WaitDecisionState {
	s.defaultOption = option
	return s
}

func (s *WaitDecisionState) Enter(ctx *StateContext) {
	// Initialize decision from context if not already set
	if s.decision == nil {
		s.decision = ctx.GetDecision()
	}

	// Validate decision exists
	if s.decision == nil {
		ctx.Error = NewStateError(StateWaitDecision, "no decision to wait for")
		return
	}

	// Initialize timeout from context if specified
	if ctx.Timeout > 0 {
		s.timeout = ctx.Timeout
	}

	s.startTime = time.Now()
	s.completed = false
	s.choice = -1 // No choice made yet
}

func (s *WaitDecisionState) Update(ctx *StateContext) StateID {
	// Check if already completed
	if s.completed {
		return StateNone // Signal PopInterrupt
	}

	// Check timeout - execute default option
	if time.Since(s.startTime) > s.timeout {
		s.executeOption(ctx, s.defaultOption)
		return StateNone // Signal PopInterrupt
	}

	// Check if user choice was received (via OnUserChoice)
	if s.choice >= 0 {
		s.executeOption(ctx, s.choice)
		return StateNone // Signal PopInterrupt
	}

	// Continue waiting
	return StateNone
}

func (s *WaitDecisionState) Exit(ctx *StateContext) {
	s.decision = nil
	s.completed = false
	s.choice = -1
	s.startTime = time.Time{}
}

// OnUserChoice handles user choice input.
// Called by HSM when receiving user decision response.
func (s *WaitDecisionState) OnUserChoice(ctx *StateContext, choice int) {
	// Validate choice index
	if choice < 0 || choice >= len(s.decision.Options) {
		// Invalid choice, use default
		choice = s.defaultOption
	}

	s.choice = choice
}

// executeOption executes the selected decision option.
func (s *WaitDecisionState) executeOption(ctx *StateContext, optionIndex int) {
	if s.decision == nil || optionIndex >= len(s.decision.Options) {
		return
	}

	option := s.decision.Options[optionIndex]

	// Create execution context
	execCtx := event.NewContext(ctx.Player)

	// Execute option action if defined
	if option.Action != nil {
		option.Action(execCtx)
	}

	// Mark as completed
	s.completed = true

	// Signal that decision was processed
	ctx.SetBool("decision_processed", true)
	ctx.SetInt("decision_choice", optionIndex)
}

// ========== Interrupt State Factory ==========

// InterruptStateFactory creates interrupt layer states.
type InterruptStateFactory struct{}

// CreateState creates an interrupt state by ID.
func (f *InterruptStateFactory) CreateState(id StateID) State {
	switch id {
	case StateWaitDecision:
		return NewWaitDecisionState()
	default:
		return nil
	}
}

// RegisterInterruptStates registers all interrupt states with HSM.
func RegisterInterruptStates(hsm *HSM) error {
	factory := &InterruptStateFactory{}
	states := []State{
		factory.CreateState(StateWaitDecision),
	}
	return hsm.RegisterStates(states)
}
