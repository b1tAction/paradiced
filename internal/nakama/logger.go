// Package nakama provides Nakama match handler implementation for Paradiced.
package nakama

import (
	"encoding/json"

	"github.com/b1tAction/paradiced/pkg/constants"
)

// Logger provides structured logging for request/response tracking.
type Logger struct {
	handler *NakamaMatchHandler
}

// NewLogger creates a new Logger for a handler.
func NewLogger(handler *NakamaMatchHandler) *Logger {
	return &Logger{handler: handler}
}

// logRequest logs the start of request processing.
func (l *Logger) logRequest(opCode string, sender string, data interface{}) {
	if l.handler.logger == nil {
		return
	}

	dataStr := "nil"
	if data != nil {
		if b, err := json.Marshal(data); err == nil {
			dataStr = string(b)
		}
	}

	l.handler.logger.Debug("[REQ] Processing request",
		"op_code", opCode,
		"sender", sender,
		"data", dataStr)
}

// logResponse logs a successful response.
func (l *Logger) logResponse(opCode string, sender string, result string) {
	if l.handler.logger == nil {
		return
	}

	l.handler.logger.Debug("[RES] Request completed",
		"op_code", opCode,
		"sender", sender,
		"result", result)
}

// logReject logs a rejected request with error details.
func (l *Logger) logReject(opCode string, sender string, errCode constants.ErrorCode, reason string, message string) {
	if l.handler.logger == nil {
		return
	}

	l.handler.logger.Warn("[REJ] Request rejected",
		"op_code", opCode,
		"sender", sender,
		"error_code", errCode,
		"reason", reason,
		"message", message)
}

// logError logs an error during request processing.
func (l *Logger) logError(opCode string, sender string, err error) {
	if l.handler.logger == nil {
		return
	}

	l.handler.logger.Error("[ERR] Request processing failed",
		"op_code", opCode,
		"sender", sender,
		"error", err)
}

// logState logs state transition information.
func (l *Logger) logState(sender string, currentState string, expectedState string) {
	if l.handler.logger == nil {
		return
	}

	l.handler.logger.Debug("[STATE] State check",
		"sender", sender,
		"current_state", currentState,
		"expected_state", expectedState)
}

// logValidation logs validation results.
func (l *Logger) logValidation(sender string, checkName string, passed bool, details ...interface{}) {
	if l.handler.logger == nil {
		return
	}

	level := "debug"
	if !passed {
		level = "warn"
	}

	keysAndValues := []interface{}{
		"sender", sender,
		"check", checkName,
		"passed", passed,
	}
	keysAndValues = append(keysAndValues, details...)

	if level == "debug" {
		l.handler.logger.Debug("[VAL] Validation result", keysAndValues...)
	} else {
		l.handler.logger.Warn("[VAL] Validation failed", keysAndValues...)
	}
}

// logPlayer logs player-related information.
func (l *Logger) logPlayer(sender string, action string, playerID string, isCurrent bool) {
	if l.handler.logger == nil {
		return
	}

	l.handler.logger.Debug("[PLAYER] Player action",
		"sender", sender,
		"action", action,
		"player_id", playerID,
		"is_current", isCurrent)
}
