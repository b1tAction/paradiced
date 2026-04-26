package errors

import (
	"testing"
)

func TestInternalError(t *testing.T) {
	baseErr := &InternalError{
		Component: "HSM",
		Operation: "Update",
		Err:       ErrTest,
	}

	if baseErr.Error() != "[HSM] Update failed: test error" {
		t.Errorf("unexpected error message: %s", baseErr.Error())
	}

	// Test with context
	withContext := baseErr.WithContext("player_id", "player-001")
	if withContext.Context["player_id"] != "player-001" {
		t.Errorf("expected context player_id to be 'player-001', got %v", withContext.Context["player_id"])
	}
}

func TestInternalErrorWithContext(t *testing.T) {
	err := NewInternalError("EventBus", "Publish", ErrTest).
		WithContext("phase", "BeforeTurn").
		WithContext("buff_type", "divine")

	if err.Component != "EventBus" {
		t.Errorf("expected Component 'EventBus', got %s", err.Component)
	}
	if err.Context["phase"] != "BeforeTurn" {
		t.Errorf("expected phase 'BeforeTurn', got %v", err.Context["phase"])
	}
	if err.Context["buff_type"] != "divine" {
		t.Errorf("expected buff_type 'divine', got %v", err.Context["buff_type"])
	}
}

func TestValidationError(t *testing.T) {
	err := NewValidationError("player_id", "", "must be non-empty")

	expected := "validation error: field 'player_id' - must be non-empty (got: )"
	if err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}

	// Test with value
	errWithValue := NewValidationError("dice_type", "invalid", "must be gold/silver/copper/wood")
	expectedWithValue := "validation error: field 'dice_type' - must be gold/silver/copper/wood (got: invalid)"
	if errWithValue.Error() != expectedWithValue {
		t.Errorf("expected error %q, got %q", expectedWithValue, errWithValue.Error())
	}
}

func TestStateExecutionError(t *testing.T) {
	err := NewStateExecutionError("TurnUpkeep", "Enter", ErrTest)

	if err.Error() != "state TurnUpkeep Enter phase error: test error" {
		t.Errorf("unexpected error message: %s", err.Error())
	}

	// Test Unwrap
	unwrapped := err.Unwrap()
	if unwrapped != ErrTest {
		t.Errorf("expected unwrapped error to be ErrTest, got %v", unwrapped)
	}
}

func TestActionExecutionError(t *testing.T) {
	err := NewActionExecutionError("damage", "player-001", "target is dead", ErrTest)

	if err.Error() != "action damage on target player-001 failed (target is dead): test error" {
		t.Errorf("unexpected error message: %s", err.Error())
	}

	// Test Unwrap
	unwrapped := err.Unwrap()
	if unwrapped != ErrTest {
		t.Errorf("expected unwrapped error to be ErrTest, got %v", unwrapped)
	}

	// Test without underlying error
	errNoErr := NewActionExecutionError("heal", "player-001", "target not found", nil)
	if errNoErr.Error() != "action heal on target player-001 failed (target not found)" {
		t.Errorf("unexpected error message: %s", errNoErr.Error())
	}

	// Test Unwrap returns nil when no underlying error
	if errNoErr.Unwrap() != nil {
		t.Errorf("expected Unwrap to return nil, got %v", errNoErr.Unwrap())
	}
}

func TestWrap(t *testing.T) {
	// Test wrapping nil error
	nilErr := Wrap(nil, "HSM", "Update")
	if nilErr != nil {
		t.Errorf("expected nil when wrapping nil error, got %v", nilErr)
	}

	// Test wrapping actual error
	wrapped := Wrap(ErrTest, "HSM", "Update")
	if wrapped == nil {
		t.Fatal("expected non-nil wrapped error")
	}
	if wrapped.Component != "HSM" {
		t.Errorf("expected Component 'HSM', got %s", wrapped.Component)
	}
	if wrapped.Operation != "Update" {
		t.Errorf("expected Operation 'Update', got %s", wrapped.Operation)
	}
	if wrapped.Unwrap() != ErrTest {
		t.Errorf("expected Unwrap to return ErrTest, got %v", wrapped.Unwrap())
	}
}

func TestWrapf(t *testing.T) {
	// Test wrapping with format
	wrapped := Wrapf(ErrTest, "EventBus", "Publish", "phase %s", "BeforeTurn")
	if wrapped == nil {
		t.Fatal("expected non-nil wrapped error")
	}
	if wrapped.Context["message"] != "phase BeforeTurn" {
		t.Errorf("expected context message 'phase BeforeTurn', got %v", wrapped.Context["message"])
	}

	// Test wrapping without format message (still need empty format string)
	wrappedNoFormat := Wrapf(ErrTest, "HSM", "Transition", "")
	if wrappedNoFormat == nil {
		t.Fatal("expected non-nil wrapped error")
	}
	if wrappedNoFormat.Context != nil {
		t.Errorf("expected nil context when no format provided, got %v", wrappedNoFormat.Context)
	}
}

func TestErrorAs(t *testing.T) {
	// Test errors.As with ValidationError
	var validationErr *ValidationError
	err := NewValidationError("field", "value", "reason")
	if !As(err, &validationErr) {
		t.Error("expected errors.As to return true for ValidationError")
	}
	if validationErr.Field != "field" {
		t.Errorf("expected Field 'field', got %s", validationErr.Field)
	}

	// Test errors.As with InternalError
	var internalErr *InternalError
	err2 := NewInternalError("HSM", "Update", ErrTest)
	if !As(err2, &internalErr) {
		t.Error("expected errors.As to return true for InternalError")
	}
	if internalErr.Component != "HSM" {
		t.Errorf("expected Component 'HSM', got %s", internalErr.Component)
	}
}

func TestHSMError(t *testing.T) {
	// Test basic HSMError
	err := NewHSMError("TurnUpkeep", 2, "Enter", ErrTest, "phase execution failed")

	if err.Error() != "HSM error in state TurnUpkeep (layer 2, phase Enter): phase execution failed - test error" {
		t.Errorf("unexpected error message: %s", err.Error())
	}

	// Test without message
	errNoMsg := NewHSMError("TurnMoving", 2, "Enter", ErrTest, "")
	if errNoMsg.Error() != "HSM error in state TurnMoving (layer 2, phase Enter): test error" {
		t.Errorf("unexpected error message: %s", errNoMsg.Error())
	}

	// Test Unwrap
	unwrapped := err.Unwrap()
	if unwrapped != ErrTest {
		t.Errorf("expected Unwrap to return ErrTest, got %v", unwrapped)
	}

	// Test errors.As
	var hsmErr *HSMError
	if !As(err, &hsmErr) {
		t.Error("expected errors.As to return true for HSMError")
	}
	if hsmErr.StateID != "TurnUpkeep" {
		t.Errorf("expected StateID 'TurnUpkeep', got %s", hsmErr.StateID)
	}
	if hsmErr.Layer != 2 {
		t.Errorf("expected Layer 2, got %d", hsmErr.Layer)
	}
	if hsmErr.Phase != "Enter" {
		t.Errorf("expected Phase 'Enter', got %s", hsmErr.Phase)
	}
}

func TestWrapHSMError(t *testing.T) {
	// Test wrapping nil error
	nilErr := WrapHSMError(nil, "TurnUpkeep", 2, "Enter", "")
	if nilErr != nil {
		t.Errorf("expected nil when wrapping nil error, got %v", nilErr)
	}

	// Test wrapping actual error
	wrapped := WrapHSMError(ErrTest, "TurnMoving", 2, "Enter", "move failed")
	if wrapped == nil {
		t.Fatal("expected non-nil wrapped error")
	}
	hsmErr, ok := wrapped.(*HSMError)
	if !ok {
		t.Fatal("expected wrapped error to be *HSMError")
	}
	if hsmErr.StateID != "TurnMoving" {
		t.Errorf("expected StateID 'TurnMoving', got %s", hsmErr.StateID)
	}
	if hsmErr.Layer != 2 {
		t.Errorf("expected Layer 2, got %d", hsmErr.Layer)
	}
	if hsmErr.Phase != "Enter" {
		t.Errorf("expected Phase 'Enter', got %s", hsmErr.Phase)
	}

	// Test don't wrap already HSMError
	alreadyHSM := NewHSMError("Test", 1, "Enter", ErrTest, "")
	notWrapped := WrapHSMError(alreadyHSM, "Other", 2, "Exit", "")
	if notWrapped != alreadyHSM {
		t.Error("expected WrapHSMError to return same error when already HSMError")
	}
}

func TestIsHSMError(t *testing.T) {
	// HSMError should be detected
	hsmErr := NewHSMError("TurnUpkeep", 2, "Enter", ErrTest, "")
	if !IsHSMError(hsmErr) {
		t.Error("IsHSMError should return true for HSMError")
	}

	// Other error types should not be detected
	internalErr := NewInternalError("HSM", "Update", ErrTest)
	if IsHSMError(internalErr) {
		t.Error("IsHSMError should return false for InternalError")
	}

	// Nil should not be detected
	if IsHSMError(nil) {
		t.Error("IsHSMError should return false for nil")
	}

	// Wrapped HSMError should be detected
	wrapped := WrapHSMError(ErrTest, "Test", 1, "Enter", "")
	if !IsHSMError(wrapped) {
		t.Error("IsHSMError should return true for wrapped HSMError")
	}
}

// ErrTest is a test error for use in tests.
var ErrTest = &testError{"test error"}

type testError struct {
	msg string
}

func (e *testError) Error() string { return e.msg }

// As is a helper for testing errors.As compatibility.
func As(err error, target interface{}) bool {
	return errorsAs(err, target)
}

// errorsAs is a simple implementation of errors.As for testing.
func errorsAs(err error, target interface{}) bool {
	if err == nil {
		return false
	}
	// For this test, just check if types match and set the target
	switch t := target.(type) {
	case **ValidationError:
		if v, ok := err.(*ValidationError); ok {
			*t = v
			return true
		}
		return false
	case **InternalError:
		if v, ok := err.(*InternalError); ok {
			*t = v
			return true
		}
		return false
	case **StateExecutionError:
		if v, ok := err.(*StateExecutionError); ok {
			*t = v
			return true
		}
		return false
	case **ActionExecutionError:
		if v, ok := err.(*ActionExecutionError); ok {
			*t = v
			return true
		}
		return false
	case **HSMError:
		if v, ok := err.(*HSMError); ok {
			*t = v
			return true
		}
		return false
	}
	return false
}
