package gamelog

import (
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
)

// LogLevel for game debug logging.
type LogLevel int

const (
	LogLevelDebug LogLevel = 0
	LogLevelInfo  LogLevel = 1
	LogLevelWarn  LogLevel = 2
	LogLevelError LogLevel = 3
)

// ParseLogLevel parses a level string ("debug", "info", "warn", "error")
// into a LogLevel. Returns LogLevelInfo for unrecognized values.
func ParseLogLevel(s string) LogLevel {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return LogLevelDebug
	case "info":
		return LogLevelInfo
	case "warn", "warning":
		return LogLevelWarn
	case "error", "err":
		return LogLevelError
	default:
		return LogLevelInfo
	}
}

// GameLogger provides structured debug logging for the game engine.
// It is nil-safe: all methods on a nil *GameLogger silently return without panicking.
// Thread-safe via mutex on writer access.
// By default, outputs to stdout via log.Printf.
// In Nakama production, a writer bridging to runtime.Logger is injected.
// Log level is controlled via PD_LOG_LEVEL environment variable (default: "info").
type GameLogger struct {
	level  LogLevel
	writer func(msg string, keysAndValues ...interface{})
	mu     sync.RWMutex
}

// NewGameLogger creates a GameLogger with level from PD_LOG_LEVEL env var
// (default: info) and stdout writer via log.Printf.
func NewGameLogger() *GameLogger {
	level := ParseLogLevel(os.Getenv("PD_LOG_LEVEL"))
	return &GameLogger{
		level:  level,
		writer: defaultWriter,
	}
}

// defaultWriter formats structured log messages to stdout via log.Printf.
func defaultWriter(msg string, keysAndValues ...interface{}) {
	if len(keysAndValues) == 0 {
		log.Printf("[paradiced] %s", msg)
		return
	}
	// Format key-value pairs as k=v
	pairs := make([]string, 0, len(keysAndValues)/2)
	for i := 0; i+1 < len(keysAndValues); i += 2 {
		pairs = append(pairs, fmt.Sprintf("%s=%v", keysAndValues[i], keysAndValues[i+1]))
	}
	// Handle odd trailing value
	if len(keysAndValues)%2 == 1 {
		pairs = append(pairs, fmt.Sprintf("extra=%v", keysAndValues[len(keysAndValues)-1]))
	}
	log.Printf("[paradiced] %s %s", msg, strings.Join(pairs, " "))
}

// WithLevel sets the minimum log level. Returns the same pointer for chaining.
func (l *GameLogger) WithLevel(level LogLevel) *GameLogger {
	if l == nil {
		return nil
	}
	l.mu.Lock()
	l.level = level
	l.mu.Unlock()
	return l
}

// WithWriter sets a custom writer function (e.g., Nakama runtime.Logger bridge).
// Returns the same pointer for chaining.
func (l *GameLogger) WithWriter(writer func(msg string, keysAndValues ...interface{})) *GameLogger {
	if l == nil {
		return nil
	}
	l.mu.Lock()
	l.writer = writer
	l.mu.Unlock()
	return l
}

// Debug logs a message at debug level. Nil-safe.
func (l *GameLogger) Debug(msg string, keysAndValues ...interface{}) {
	if l == nil {
		return
	}
	l.mu.RLock()
	level := l.level
	writer := l.writer
	l.mu.RUnlock()
	if level <= LogLevelDebug {
		writer("DEBUG "+msg, keysAndValues...)
	}
}

// Info logs a message at info level. Nil-safe.
func (l *GameLogger) Info(msg string, keysAndValues ...interface{}) {
	if l == nil {
		return
	}
	l.mu.RLock()
	level := l.level
	writer := l.writer
	l.mu.RUnlock()
	if level <= LogLevelInfo {
		writer("INFO "+msg, keysAndValues...)
	}
}

// Warn logs a message at warn level. Nil-safe.
func (l *GameLogger) Warn(msg string, keysAndValues ...interface{}) {
	if l == nil {
		return
	}
	l.mu.RLock()
	level := l.level
	writer := l.writer
	l.mu.RUnlock()
	if level <= LogLevelWarn {
		writer("WARN "+msg, keysAndValues...)
	}
}

// Error logs a message at error level. Nil-safe.
func (l *GameLogger) Error(msg string, keysAndValues ...interface{}) {
	if l == nil {
		return
	}
	l.mu.RLock()
	level := l.level
	writer := l.writer
	l.mu.RUnlock()
	if level <= LogLevelError {
		writer("ERROR "+msg, keysAndValues...)
	}
}