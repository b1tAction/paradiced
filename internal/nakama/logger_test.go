// Package nakama provides Nakama match handler implementation for Paradiced.
package nakama

import (
	"testing"

	"github.com/b1tAction/paradiced/pkg/constants"
	"github.com/heroiclabs/nakama-common/runtime"
)

// mockLogger implements runtime.Logger for testing.
type mockLogger struct {
	debugMsgs   []string
	infoMsgs    []string
	warnMsgs    []string
	errorMsgs   []string
	lastArgs    []interface{}
	debugCalled bool
	infoCalled  bool
	warnCalled  bool
	errorCalled bool
}

func (m *mockLogger) Debug(format string, args ...interface{}) {
	m.debugCalled = true
	m.debugMsgs = append(m.debugMsgs, format)
	m.lastArgs = args
}

func (m *mockLogger) Info(format string, args ...interface{}) {
	m.infoCalled = true
	m.infoMsgs = append(m.infoMsgs, format)
	m.lastArgs = args
}

func (m *mockLogger) Warn(format string, args ...interface{}) {
	m.warnCalled = true
	m.warnMsgs = append(m.warnMsgs, format)
	m.lastArgs = args
}

func (m *mockLogger) Error(format string, args ...interface{}) {
	m.errorCalled = true
	m.errorMsgs = append(m.errorMsgs, format)
	m.lastArgs = args
}

func (m *mockLogger) Trace(format string, args ...interface{}) {}

func (m *mockLogger) WithField(key string, v interface{}) runtime.Logger {
	return m
}

func (m *mockLogger) WithFields(fields map[string]interface{}) runtime.Logger {
	return m
}

func (m *mockLogger) Fields() map[string]interface{} {
	return nil
}

func newMockLogger() *mockLogger {
	return &mockLogger{
		debugMsgs: make([]string, 0),
		infoMsgs:  make([]string, 0),
		warnMsgs:  make([]string, 0),
		errorMsgs: make([]string, 0),
	}
}

func TestNewLogger(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)
	logger := NewLogger(handler)

	if logger == nil {
		t.Fatal("NewLogger returned nil")
	}
	if logger.handler != handler {
		t.Error("Logger handler not set correctly")
	}
}

func TestLoggerWithNilLogger(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)
	logger := NewLogger(handler)

	// These should not panic with nil logger
	logger.logRequest("test_op", "user-001", nil)
	logger.logResponse("test_op", "user-001", "success")
	logger.logReject("test_op", "user-001", constants.ErrPlayerNotFound, "player_not_found", "Player not found")
	logger.logError("test_op", "user-001", nil)
	logger.logState("user-001", "main_action", "main_action")
	logger.logValidation("user-001", "player_check", true)
	logger.logPlayer("user-001", "roll_dice", "player-001", true)
}

func TestLoggerLogRequest(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)
	mockLog := newMockLogger()
	handler.WithLogger(mockLog)
	logger := NewLogger(handler)

	logger.logRequest("roll_dice", "user-001", nil)

	if !mockLog.debugCalled {
		t.Error("logRequest should call Debug")
	}
	if len(mockLog.debugMsgs) != 1 {
		t.Errorf("logRequest should call Debug once, got %d", len(mockLog.debugMsgs))
	}
}

func TestLoggerLogResponse(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)
	mockLog := newMockLogger()
	handler.WithLogger(mockLog)
	logger := NewLogger(handler)

	logger.logResponse("roll_dice", "user-001", "success")

	if !mockLog.debugCalled {
		t.Error("logResponse should call Debug")
	}
}

func TestLoggerLogReject(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)
	mockLog := newMockLogger()
	handler.WithLogger(mockLog)
	logger := NewLogger(handler)

	logger.logReject("roll_dice", "user-001", constants.ErrPlayerNotFound, "player_not_found", "Player not found")

	if !mockLog.warnCalled {
		t.Error("logReject should call Warn")
	}
}

func TestLoggerLogError(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)
	mockLog := newMockLogger()
	handler.WithLogger(mockLog)
	logger := NewLogger(handler)

	testErr := &mockError{"test error"}
	logger.logError("roll_dice", "user-001", testErr)

	if !mockLog.errorCalled {
		t.Error("logError should call Error")
	}
}

func TestLoggerLogState(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)
	mockLog := newMockLogger()
	handler.WithLogger(mockLog)
	logger := NewLogger(handler)

	logger.logState("user-001", "main_action", "main_action")

	if !mockLog.debugCalled {
		t.Error("logState should call Debug")
	}
}

func TestLoggerLogValidation(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)
	mockLog := newMockLogger()
	handler.WithLogger(mockLog)
	logger := NewLogger(handler)

	// Test passed validation
	logger.logValidation("user-001", "player_check", true)

	if !mockLog.debugCalled {
		t.Error("logValidation with passed=true should call Debug")
	}

	// Reset
	mockLog.debugCalled = false

	// Test failed validation
	logger.logValidation("user-001", "player_check", false)

	if !mockLog.warnCalled {
		t.Error("logValidation with passed=false should call Warn")
	}
}

func TestLoggerLogPlayer(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)
	mockLog := newMockLogger()
	handler.WithLogger(mockLog)
	logger := NewLogger(handler)

	logger.logPlayer("user-001", "roll_dice", "player-001", true)

	if !mockLog.debugCalled {
		t.Error("logPlayer should call Debug")
	}
}

func TestLoggerLogRequestWithData(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)
	mockLog := newMockLogger()
	handler.WithLogger(mockLog)
	logger := NewLogger(handler)

	data := map[string]interface{}{"key": "value"}
	logger.logRequest("use_item", "user-001", data)

	if !mockLog.debugCalled {
		t.Error("logRequest with data should call Debug")
	}
}

// mockError implements error interface for testing.
type mockError struct {
	msg string
}

func (e *mockError) Error() string {
	return e.msg
}
