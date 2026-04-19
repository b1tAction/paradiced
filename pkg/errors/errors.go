// Package errors provides unified error handling for the Paradiced project.
// This package defines common error types and wrapping utilities for consistent error handling.
//
// # Error Handling Strategy
//
// 1. Use specific error types for different error scenarios
// 2. Wrap low-level errors with context using Wrap/Wrapf
// 3. Return errors up the call chain - don't swallow them
// 4. Log errors at the boundary (Nakama layer)
//
// # Error Types
//
//   - [InternalError]: Internal server errors with component context
//   - [ValidationError]: Client input validation failures
//   - [StateExecutionError]: HSM state execution failures
//   - [ActionExecutionError]: Action execution failures
//
// # Example usage
//
//	// In service layer
//	func DoSomething(id string) error {
//	    if err := validateID(id); err != nil {
//	        return errors.NewValidationError("id", id, "must be non-empty")
//	    }
//	    result, err := repo.Find(id)
//	    if err != nil {
//	        return errors.Wrap(err, "repository", "find")
//	    }
//	    return nil
//	}
//
//	// In handler layer
//	func Handle(w http.ResponseWriter, r *http.Request) {
//	    err := DoSomething("123")
//	    if err != nil {
//	        var ve *errors.ValidationError
//	        if errors.As(err, &ve) {
//	            http.Error(w, ve.Error(), http.StatusBadRequest)
//	            return
//	        }
//	        // Log and return internal error
//	        logger.Error("handle failed", "error", err)
//	        http.Error(w, "internal error", http.StatusInternalServerError)
//	    }
//	}
package errors

import "fmt"

// InternalError represents an internal server error with context.
// Used for wrapping low-level errors with additional context.
type InternalError struct {
	// Component is the component where the error originated (e.g., "HSM", "Action", "EventBus")
	Component string
	// Operation is the operation being performed when error occurred
	Operation string
	// Err is the underlying error
	Err error
	// Context provides additional key-value context
	Context map[string]any
}

// Error implements the error interface.
func (e *InternalError) Error() string {
	if len(e.Context) > 0 {
		return fmt.Sprintf("[%s] %s failed: %v (context: %v)", e.Component, e.Operation, e.Err, e.Context)
	}
	return fmt.Sprintf("[%s] %s failed: %v", e.Component, e.Operation, e.Err)
}

// Unwrap returns the underlying error for errors.Is/As.
func (e *InternalError) Unwrap() error {
	return e.Err
}

// WithContext adds context to the error.
func (e *InternalError) WithContext(key string, value any) *InternalError {
	if e.Context == nil {
		e.Context = make(map[string]any)
	}
	e.Context[key] = value
	return e
}

// ValidationError represents a validation failure.
// Used for client input validation errors.
type ValidationError struct {
	// Field is the field that failed validation
	Field string
	// Value is the invalid value that was provided
	Value any
	// Reason explains why validation failed
	Reason string
}

// Error implements the error interface.
func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error: field '%s' - %s (got: %v)", e.Field, e.Reason, e.Value)
}

// StateExecutionError represents an error during state execution.
type StateExecutionError struct {
	// StateName is the name of the state where error occurred
	StateName string
	// Phase is the execution phase (Enter, Update, Exit)
	Phase string
	// Err is the underlying error
	Err error
}

// Error implements the error interface.
func (e *StateExecutionError) Error() string {
	return fmt.Sprintf("state %s %s phase error: %v", e.StateName, e.Phase, e.Err)
}

// Unwrap returns the underlying error.
func (e *StateExecutionError) Unwrap() error {
	return e.Err
}

// ActionExecutionError represents an error during action execution.
type ActionExecutionError struct {
	// ActionType is the type of action that failed
	ActionType string
	// TargetID is the target of the action
	TargetID string
	// Reason explains why the action failed
	Reason string
	// Err is the underlying error
	Err error
}

// Error implements the error interface.
func (e *ActionExecutionError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("action %s on target %s failed (%s): %v", e.ActionType, e.TargetID, e.Reason, e.Err)
	}
	return fmt.Sprintf("action %s on target %s failed (%s)", e.ActionType, e.TargetID, e.Reason)
}

// Unwrap returns the underlying error.
func (e *ActionExecutionError) Unwrap() error {
	return e.Err
}

// ========== Error Creation Helpers ==========

// NewInternalError creates a new InternalError.
func NewInternalError(component, operation string, err error) *InternalError {
	return &InternalError{
		Component: component,
		Operation: operation,
		Err:       err,
	}
}

// NewValidationError creates a new ValidationError.
func NewValidationError(field string, value any, reason string) *ValidationError {
	return &ValidationError{
		Field:  field,
		Value:  value,
		Reason: reason,
	}
}

// NewStateExecutionError creates a new StateExecutionError.
func NewStateExecutionError(stateName, phase string, err error) *StateExecutionError {
	return &StateExecutionError{
		StateName: stateName,
		Phase:     phase,
		Err:       err,
	}
}

// NewActionExecutionError creates a new ActionExecutionError.
func NewActionExecutionError(actionType, targetID, reason string, err error) *ActionExecutionError {
	return &ActionExecutionError{
		ActionType: actionType,
		TargetID:   targetID,
		Reason:     reason,
		Err:        err,
	}
}

// Wrap wraps an error with additional context.
// Returns nil if err is nil.
func Wrap(err error, component, operation string) *InternalError {
	if err == nil {
		return nil
	}
	return &InternalError{
		Component: component,
		Operation: operation,
		Err:       err,
	}
}

// Wrapf wraps an error with formatted message.
func Wrapf(err error, component, operation string, format string, args ...any) *InternalError {
	if err == nil {
		return nil
	}
	ctx := &InternalError{
		Component: component,
		Operation: operation,
		Err:       err,
	}
	if len(args) > 0 {
		ctx.Context = map[string]any{"message": fmt.Sprintf(format, args...)}
	}
	return ctx
}

// ========== HSM Error Types ==========

// HSMError represents an error that occurred during HSM execution.
// It wraps the original error with state context for better debugging.
type HSMError struct {
	// StateID is the state where the error occurred
	StateID string
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
			e.StateID, e.Layer, e.Phase, e.Message, e.Err)
	}
	return fmt.Sprintf("HSM error in state %s (layer %d, phase %s): %v",
		e.StateID, e.Layer, e.Phase, e.Err)
}

// Unwrap returns the underlying error for errors.Is/As.
func (e *HSMError) Unwrap() error {
	return e.Err
}

// NewHSMError creates a new HSMError with context.
func NewHSMError(stateID string, layer int, phase string, err error, message string) *HSMError {
	return &HSMError{
		StateID: stateID,
		Layer:   layer,
		Phase:   phase,
		Err:     err,
		Message: message,
	}
}

// WrapHSMError wraps an error with HSM context if it's not already wrapped.
// Returns nil if err is nil.
func WrapHSMError(err error, stateID string, layer int, phase string, message string) error {
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
