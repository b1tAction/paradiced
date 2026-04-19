// Package constants provides unified enum type definitions.
package constants

// ErrorCode represents a standardized error code for client communication.
// Error codes are structured as follows:
// - 1xxx: Validation errors (player, state, input)
// - 2xxx: Game logic errors (action rejected, invalid timing)
// - 3xxx: System errors (internal, HSM, network)
// - 4xxx: Not found errors (player, item, buff)
type ErrorCode int

const (
	// ErrOK indicates no error.
	ErrOK ErrorCode = 0

	// === Validation Errors (1xxx) ===

	// ErrInvalidParameter indicates an invalid parameter was provided.
	ErrInvalidParameter ErrorCode = 1001

	// ErrInvalidState indicates the game state does not allow this action.
	ErrInvalidState ErrorCode = 1002

	// ErrInvalidTiming indicates the action timing is incorrect.
	ErrInvalidTiming ErrorCode = 1003

	// ErrNotCurrentTurn indicates the player is not the current turn player.
	ErrNotCurrentTurn ErrorCode = 1004

	// ErrConditionNotMet indicates required conditions are not met.
	ErrConditionNotMet ErrorCode = 1005

	// === Game Logic Errors (2xxx) ===

	// ErrActionRejected indicates the action was rejected by game logic.
	ErrActionRejected ErrorCode = 2001

	// ErrCooldownActive indicates a cooldown is still active.
	ErrCooldownActive ErrorCode = 2002

	// === System Errors (3xxx) ===

	// ErrInternal indicates an internal server error.
	ErrInternal ErrorCode = 3001

	// ErrHSMError indicates a state machine error.
	ErrHSMError ErrorCode = 3002

	// ErrDispatchFailed indicates message dispatch failed.
	ErrDispatchFailed ErrorCode = 3003

	// === Not Found Errors (4xxx) ===

	// ErrPlayerNotFound indicates the player was not found.
	ErrPlayerNotFound ErrorCode = 4001

	// ErrItemNotFound indicates the item was not found.
	ErrItemNotFound ErrorCode = 4002

	// ErrBuffNotFound indicates the buff was not found.
	ErrBuffNotFound ErrorCode = 4003

	// ErrMatchNotFound indicates the match was not found.
	ErrMatchNotFound ErrorCode = 4004
)

// ErrorCodeDetails provides human-readable details for each error code.
var ErrorCodeDetails = map[ErrorCode]ErrorDetail{
	ErrOK: {
		Code:    ErrOK,
		Message: "Success",
	},
	ErrInvalidParameter: {
		Code:    ErrInvalidParameter,
		Message: "Invalid parameter",
	},
	ErrInvalidState: {
		Code:    ErrInvalidState,
		Message: "Invalid game state for this action",
	},
	ErrInvalidTiming: {
		Code:    ErrInvalidTiming,
		Message: "Invalid timing for this action",
	},
	ErrNotCurrentTurn: {
		Code:    ErrNotCurrentTurn,
		Message: "Not your turn",
	},
	ErrConditionNotMet: {
		Code:    ErrConditionNotMet,
		Message: "Required conditions not met",
	},
	ErrActionRejected: {
		Code:    ErrActionRejected,
		Message: "Action rejected by game rules",
	},
	ErrCooldownActive: {
		Code:    ErrCooldownActive,
		Message: "Cooldown still active",
	},
	ErrInternal: {
		Code:    ErrInternal,
		Message: "Internal server error",
	},
	ErrHSMError: {
		Code:    ErrHSMError,
		Message: "State machine error",
	},
	ErrDispatchFailed: {
		Code:    ErrDispatchFailed,
		Message: "Message dispatch failed",
	},
	ErrPlayerNotFound: {
		Code:    ErrPlayerNotFound,
		Message: "Player not found",
	},
	ErrItemNotFound: {
		Code:    ErrItemNotFound,
		Message: "Item not found",
	},
	ErrBuffNotFound: {
		Code:    ErrBuffNotFound,
		Message: "Buff not found",
	},
	ErrMatchNotFound: {
		Code:    ErrMatchNotFound,
		Message: "Match not found",
	},
}

// ErrorDetail contains detailed information about an error code.
type ErrorDetail struct {
	// Code is the numeric error code.
	Code ErrorCode `json:"code"`

	// Message is a human-readable error message.
	Message string `json:"message"`
}

// GetErrorDetail returns the ErrorDetail for a given ErrorCode.
func GetErrorDetail(code ErrorCode) ErrorDetail {
	if detail, ok := ErrorCodeDetails[code]; ok {
		return detail
	}
	return ErrorDetail{
		Code:    ErrInternal,
		Message: "Unknown error code",
	}
}

// ToReason converts an ErrorCode to a string reason for ActionRejected.
func (e ErrorCode) ToReason() string {
	switch e {
	case ErrPlayerNotFound:
		return "player_not_found"
	case ErrItemNotFound:
		return "item_not_found"
	case ErrBuffNotFound:
		return "buff_not_found"
	case ErrNotCurrentTurn:
		return "not_current_player"
	case ErrInvalidState:
		return "invalid_state"
	case ErrInvalidTiming:
		return "invalid_timing"
	case ErrConditionNotMet:
		return "condition_not_met"
	case ErrActionRejected:
		return "action_rejected"
	default:
		return "unknown_error"
	}
}
