package nakama

import (
	"errors"

	"github.com/b1tAction/paradiced/internal/engine/hsm"
	"github.com/b1tAction/paradiced/pkg/constants"
	pkgerrors "github.com/b1tAction/paradiced/pkg/errors"
)

// ErrorCodeForError maps internal error types to client-facing error codes.
// This function is the central place for error code mapping.
//
// Mapping strategy: type-based classification with recursive unwrapping.
// HSMError and InternalError wrap underlying errors; we unwrap them
// to classify based on the root cause type, not message strings.
//
// Mapping rules:
// - ValidationError         -> ErrInvalidParameter (1001)
// - ActionExecutionError    -> ErrInternal (3001)
// - HSMError                -> unwrap Err, classify root cause
//   - root ValidationError  -> ErrInvalidParameter
//   - root InternalError    -> classify by context
//   - no Err (message only) -> fallback message-based
// - StateError              -> classify by Message (StateError has no Err)
// - InternalError           -> unwrap Err, classify root cause
// - Default                 -> ErrInternal (3001)
func ErrorCodeForError(err error) constants.ErrorCode {
	if err == nil {
		return constants.ErrInternal
	}

	// ValidationError -> 1001
	var validationErr *pkgerrors.ValidationError
	if errors.As(err, &validationErr) {
		return constants.ErrInvalidParameter
	}

	// ActionExecutionError -> 3001
	var actionErr *pkgerrors.ActionExecutionError
	if errors.As(err, &actionErr) {
		return constants.ErrInternal
	}

	// InternalError -> unwrap and classify root cause
	var internalErr *pkgerrors.InternalError
	if errors.As(err, &internalErr) {
		if internalErr.Err != nil {
			// Recursively classify the wrapped error
			return ErrorCodeForError(internalErr.Err)
		}
		return constants.ErrInternal
	}

	// HSMError -> unwrap and classify underlying error
	var hsmErr *pkgerrors.HSMError
	if errors.As(err, &hsmErr) {
		if hsmErr.Err != nil {
			// Classify based on the wrapped error type
			return ErrorCodeForError(hsmErr.Err)
		}
		// HSMError with no underlying error - use message-based fallback
		// (only for cases where Message is the sole error indicator)
		switch hsmErr.Message {
		case "player is nil":
			return constants.ErrPlayerNotFound
		default:
			return constants.ErrInvalidState
		}
	}

	// StateError -> classify based on message
	// StateError has no Err field, so message-based classification
	// is the only option here.
	var stateErr *hsm.StateError
	if errors.As(err, &stateErr) {
		switch stateErr.Message {
		case "player is nil":
			return constants.ErrPlayerNotFound
		default:
			return constants.ErrInvalidState
		}
	}

	// Default
	return constants.ErrInternal
}