package nakama

import (
	"errors"
	"testing"

	"github.com/b1tAction/paradiced/internal/engine/hsm"
	pkgerrors "github.com/b1tAction/paradiced/pkg/errors"
	"github.com/b1tAction/paradiced/pkg/constants"
)

// ========== ErrorCodeForError Tests ==========

func TestErrorCodeForErrorNil(t *testing.T) {
	code := ErrorCodeForError(nil)
	if code != constants.ErrInternal {
		t.Errorf("nil error should return ErrInternal, got %d", code)
	}
}

func TestErrorCodeForValidationError(t *testing.T) {
	err := pkgerrors.NewValidationError("field", "value", "invalid value")
	code := ErrorCodeForError(err)
	if code != constants.ErrInvalidParameter {
		t.Errorf("ValidationError should return ErrInvalidParameter (1001), got %d", code)
	}
}

func TestErrorCodeForActionExecutionError(t *testing.T) {
	err := pkgerrors.NewActionExecutionError("damage", "player-001", "failed to apply damage", nil)
	code := ErrorCodeForError(err)
	if code != constants.ErrInternal {
		t.Errorf("ActionExecutionError should return ErrInternal (3001), got %d", code)
	}
}

func TestErrorCodeForHSMErrorPlayerNil(t *testing.T) {
	err := pkgerrors.NewHSMError("TurnUpkeep", 2, "Enter", errors.New("underlying error"), "player is nil")
	code := ErrorCodeForError(err)
	if code != constants.ErrPlayerNotFound {
		t.Errorf("HSMError with 'player is nil' should return ErrPlayerNotFound (2001), got %d", code)
	}
}

func TestErrorCodeForHSMErrorInvalidState(t *testing.T) {
	err := pkgerrors.NewHSMError("TurnUpkeep", 2, "Enter", errors.New("underlying error"), "invalid state")
	code := ErrorCodeForError(err)
	if code != constants.ErrInvalidState {
		t.Errorf("HSMError with 'invalid state' should return ErrInvalidState (1002), got %d", code)
	}
}

func TestErrorCodeForHSMErrorStateExecutionFailed(t *testing.T) {
	err := pkgerrors.NewHSMError("TurnUpkeep", 2, "Enter", errors.New("underlying error"), "state execution failed")
	code := ErrorCodeForError(err)
	if code != constants.ErrInvalidState {
		t.Errorf("HSMError with 'state execution failed' should return ErrInvalidState (1002), got %d", code)
	}
}

func TestErrorCodeForHSMErrorOther(t *testing.T) {
	err := pkgerrors.NewHSMError("TurnUpkeep", 2, "Enter", errors.New("underlying error"), "some other message")
	code := ErrorCodeForError(err)
	if code != constants.ErrInternal {
		t.Errorf("HSMError with other message should return ErrInternal (3001), got %d", code)
	}
}

func TestErrorCodeForStateErrorPlayerNil(t *testing.T) {
	err := hsm.NewStateError(hsm.StateTurnUpkeep, "player is nil")
	code := ErrorCodeForError(err)
	if code != constants.ErrPlayerNotFound {
		t.Errorf("StateError with 'player is nil' should return ErrPlayerNotFound (2001), got %d", code)
	}
}

func TestErrorCodeForStateErrorOther(t *testing.T) {
	err := hsm.NewStateError(hsm.StateTurnUpkeep, "some other message")
	code := ErrorCodeForError(err)
	if code != constants.ErrInvalidState {
		t.Errorf("StateError with other message should return ErrInvalidState (1002), got %d", code)
	}
}

func TestErrorCodeForInternalError(t *testing.T) {
	err := pkgerrors.NewInternalError("HSM", "ExecuteAction", errors.New("underlying error"))
	code := ErrorCodeForError(err)
	if code != constants.ErrInternal {
		t.Errorf("InternalError should return ErrInternal (3001), got %d", code)
	}
}

func TestErrorCodeForGenericError(t *testing.T) {
	err := errors.New("generic error")
	code := ErrorCodeForError(err)
	if code != constants.ErrInternal {
		t.Errorf("generic error should return ErrInternal (3001), got %d", code)
	}
}

func TestErrorCodeForWrappedValidationError(t *testing.T) {
	// Wrap returns InternalError which wraps the ValidationError
	// errors.As will recursively find ValidationError through Unwrap()
	innerErr := pkgerrors.NewValidationError("field", "value", "invalid")
	wrappedErr := pkgerrors.Wrap(innerErr, "Handler", "Validate")
	// errors.As finds ValidationError first through Unwrap chain
	code := ErrorCodeForError(wrappedErr)
	if code != constants.ErrInvalidParameter {
		t.Errorf("wrapped ValidationError should return ErrInvalidParameter (1001), got %d", code)
	}
}

func TestErrorCodeForInternalErrorWrappingOther(t *testing.T) {
	// InternalError wrapping a generic error should return ErrInternal
	genericErr := errors.New("generic error")
	wrappedErr := pkgerrors.Wrap(genericErr, "Handler", "Process")
	code := ErrorCodeForError(wrappedErr)
	if code != constants.ErrInternal {
		t.Errorf("InternalError wrapping generic error should return ErrInternal (3001), got %d", code)
	}
}