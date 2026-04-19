package nakama

import (
	"errors"

	"github.com/b1tAction/paradiced/pkg/constants"
	pkgerrors "github.com/b1tAction/paradiced/pkg/errors"
	"github.com/b1tAction/paradiced/internal/engine/hsm"
)

// ErrorCodeForError maps internal error types to client-facing error codes.
// This function is the central place for error code mapping.
//
// Mapping rules:
// - ValidationError         -> ErrInvalidParameter (1001)
// - ActionExecutionError    -> ErrInternal (3001)
// - HSMError (player nil)   -> ErrPlayerNotFound (2001)
// - HSMError (invalid)      -> ErrInvalidState (1002)
// - StateError (player nil) -> ErrPlayerNotFound (2001)
// - InternalError           -> ErrInternal (3001)
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

	// HSMError -> depends on message
	var hsmErr *pkgerrors.HSMError
	if errors.As(err, &hsmErr) {
		switch hsmErr.Message {
		case "player is nil":
			return constants.ErrPlayerNotFound
		case "invalid state", "state execution failed":
			return constants.ErrInvalidState
		default:
			return constants.ErrInternal
		}
	}

	// StateError -> depends on message
	var stateErr *hsm.StateError
	if errors.As(err, &stateErr) {
		switch stateErr.Message {
		case "player is nil":
			return constants.ErrPlayerNotFound
		default:
			return constants.ErrInvalidState
		}
	}

	// InternalError -> 3001
	var internalErr *pkgerrors.InternalError
	if errors.As(err, &internalErr) {
		return constants.ErrInternal
	}

	// Default
	return constants.ErrInternal
}
