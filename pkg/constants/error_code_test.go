// Package constants provides unified enum type definitions.
package constants

import (
	"testing"
)

func TestErrorCodeValue(t *testing.T) {
	tests := []struct {
		name     string
		code     ErrorCode
		expected int
	}{
		{"ErrOK", ErrOK, 0},
		{"ErrInvalidParameter", ErrInvalidParameter, 1001},
		{"ErrInvalidState", ErrInvalidState, 1002},
		{"ErrInvalidTiming", ErrInvalidTiming, 1003},
		{"ErrNotCurrentTurn", ErrNotCurrentTurn, 1004},
		{"ErrConditionNotMet", ErrConditionNotMet, 1005},
		{"ErrActionRejected", ErrActionRejected, 2001},
		{"ErrCooldownActive", ErrCooldownActive, 2002},
		{"ErrInternal", ErrInternal, 3001},
		{"ErrHSMError", ErrHSMError, 3002},
		{"ErrDispatchFailed", ErrDispatchFailed, 3003},
		{"ErrPlayerNotFound", ErrPlayerNotFound, 4001},
		{"ErrItemNotFound", ErrItemNotFound, 4002},
		{"ErrBuffNotFound", ErrBuffNotFound, 4003},
		{"ErrMatchNotFound", ErrMatchNotFound, 4004},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if int(tt.code) != tt.expected {
				t.Errorf("ErrorCode %s = %d, want %d", tt.name, tt.code, tt.expected)
			}
		})
	}
}

func TestErrorCodeCategory(t *testing.T) {
	tests := []struct {
		name     string
		code     ErrorCode
		category string
	}{
		{"Validation", ErrInvalidState, "1xxx"},
		{"Validation", ErrNotCurrentTurn, "1xxx"},
		{"Validation", ErrConditionNotMet, "1xxx"},
		{"GameLogic", ErrActionRejected, "2xxx"},
		{"GameLogic", ErrCooldownActive, "2xxx"},
		{"System", ErrInternal, "3xxx"},
		{"System", ErrHSMError, "3xxx"},
		{"NotFound", ErrPlayerNotFound, "4xxx"},
		{"NotFound", ErrItemNotFound, "4xxx"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			codeInt := int(tt.code)
			if codeInt < 1000 || codeInt >= 5000 {
				t.Errorf("ErrorCode %d outside expected range", codeInt)
			}
		})
	}
}

func TestGetErrorDetail(t *testing.T) {
	tests := []struct {
		name          string
		code          ErrorCode
		expectedMsg   string
		expectedReason string
	}{
		{"Success", ErrOK, "Success", "unknown_error"},
		{"PlayerNotFound", ErrPlayerNotFound, "Player not found", "player_not_found"},
		{"InvalidState", ErrInvalidState, "Invalid game state for this action", "invalid_state"},
		{"NotCurrentTurn", ErrNotCurrentTurn, "Not your turn", "not_current_player"},
		{"ItemNotFound", ErrItemNotFound, "Item not found", "item_not_found"},
		{"Internal", ErrInternal, "Internal server error", "unknown_error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detail := GetErrorDetail(tt.code)
			if detail.Message != tt.expectedMsg {
				t.Errorf("GetErrorDetail(%v).Message = %s, want %s", tt.code, detail.Message, tt.expectedMsg)
			}
			if detail.Code != tt.code {
				t.Errorf("GetErrorDetail(%v).Code = %v, want %v", tt.code, detail.Code, tt.code)
			}
		})
	}
}

func TestGetErrorDetailUnknown(t *testing.T) {
	detail := GetErrorDetail(9999)
	if detail.Code != ErrInternal {
		t.Errorf("GetErrorDetail(9999).Code = %v, want %v", detail.Code, ErrInternal)
	}
	if detail.Message != "Unknown error code" {
		t.Errorf("GetErrorDetail(9999).Message = %s, want Unknown error code", detail.Message)
	}
}

func TestErrorCodeToReason(t *testing.T) {
	tests := []struct {
		name     string
		code     ErrorCode
		expected string
	}{
		{"PlayerNotFound", ErrPlayerNotFound, "player_not_found"},
		{"ItemNotFound", ErrItemNotFound, "item_not_found"},
		{"BuffNotFound", ErrBuffNotFound, "buff_not_found"},
		{"NotCurrentTurn", ErrNotCurrentTurn, "not_current_player"},
		{"InvalidState", ErrInvalidState, "invalid_state"},
		{"InvalidTiming", ErrInvalidTiming, "invalid_timing"},
		{"ConditionNotMet", ErrConditionNotMet, "condition_not_met"},
		{"ActionRejected", ErrActionRejected, "action_rejected"},
		{"Unknown", ErrInternal, "unknown_error"},
		{"Unknown", ErrOK, "unknown_error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reason := tt.code.ToReason()
			if reason != tt.expected {
				t.Errorf("ErrorCode(%v).ToReason() = %s, want %s", tt.code, reason, tt.expected)
			}
		})
	}
}

func TestErrorCodeDetailsCompleteness(t *testing.T) {
	// Verify all defined error codes have details
	allCodes := []ErrorCode{
		ErrOK, ErrInvalidParameter, ErrInvalidState, ErrInvalidTiming,
		ErrNotCurrentTurn, ErrConditionNotMet, ErrActionRejected,
		ErrCooldownActive, ErrInternal, ErrHSMError, ErrDispatchFailed,
		ErrPlayerNotFound, ErrItemNotFound, ErrBuffNotFound, ErrMatchNotFound,
	}

	for _, code := range allCodes {
		detail, ok := ErrorCodeDetails[code]
		if !ok {
			t.Errorf("ErrorCodeDetails missing entry for %v", code)
		}
		if detail.Code != code {
			t.Errorf("ErrorCodeDetails[%v].Code = %v, want %v", code, detail.Code, code)
		}
		if detail.Message == "" {
			t.Errorf("ErrorCodeDetails[%v].Message is empty", code)
		}
	}
}

func TestErrorCodeToReasonMapping(t *testing.T) {
	// Verify ToReason returns non-empty strings for all codes
	for code := range ErrorCodeDetails {
		reason := code.ToReason()
		if reason == "" {
			t.Errorf("ErrorCode(%v).ToReason() returned empty string", code)
		}
	}
}
