// Package hsm provides Hierarchical State Machine implementation for game flow management.
package hsm

import (
	"fmt"
)

// ========== HSM Error Types ==========

// HSMError represents an error that occurred during HSM execution.
// It wraps the original error with state context for better debugging.
type HSMError struct {
	// StateID is the state where the error occurred
	StateID StateID
	// Layer is the state layer (1=Global, 2=Turn, 3=Interrupt)
	Layer int
	// Phase is the current phase when error occurred
	Phase string
	// Err is the underlying error
	Err error
	// Message is an optional additional context
	Message string
}

// Error implements the error interface.
func (e *HSMError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("HSM error in state %s (layer %d, phase %s): %s - %v",
			e.StateID.String(), e.Layer, e.Phase, e.Message, e.Err)
	}
	return fmt.Sprintf("HSM error in state %s (layer %d, phase %s): %v",
		e.StateID.String(), e.Layer, e.Phase, e.Err)
}

// Unwrap returns the underlying error for errors.Is/As.
func (e *HSMError) Unwrap() error {
	return e.Err
}

// Note: StateError and NewStateError are defined in turn_states.go for backward compatibility.

// NewHSMError creates a new HSMError with context.
func NewHSMError(stateID StateID, layer int, phase string, err error, message string) *HSMError {
	return &HSMError{
		StateID: stateID,
		Layer:   layer,
		Phase:   phase,
		Err:     err,
		Message: message,
	}
}

// WrapError wraps an error with HSM context if it's not already wrapped.
// Returns nil if err is nil.
func WrapError(err error, stateID StateID, layer int, phase string, message string) error {
	if err == nil {
		return nil
	}
	// Don't wrap if already an HSMError
	if _, ok := err.(*HSMError); ok {
		return err
	}
	return NewHSMError(stateID, layer, phase, err, message)
}

// IsHSMError checks if an error is an HSMError.
func IsHSMError(err error) bool {
	_, ok := err.(*HSMError)
	return ok
}
