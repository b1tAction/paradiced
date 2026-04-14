// Package gamelog provides unified game event logging for client playback.
package gamelog

import (
	"time"

	"github.com/b1tAction/fated/pkg/constants"
	"github.com/b1tAction/fated/pkg/util"
)

// EntryType identifies the type of log entry.
// Alias to constants.EntryType for unified enum system.
type EntryType = constants.EntryType

// EntryType constants - aliases to constants package.
const (
	EntryTypeAction   = constants.EntryTypeAction
	EntryTypeState    = constants.EntryTypeState
	EntryTypeMiniGame = constants.EntryTypeMiniGame
	EntryTypeBoss     = constants.EntryTypeBoss
	EntryTypeDecision = constants.EntryTypeDecision
)

// LogEntry represents a single game event for client playback.
// This replaces the old TurnEventLogEntry structure.
// Uses util.Metadata for type-safe metadata storage with JSON serialization support.
type LogEntry struct {
	// Timestamp is when the event occurred.
	Timestamp time.Time `json:"timestamp"`
	// Type is the entry type (action, state, etc).
	Type EntryType `json:"type"`
	// ActionType is the specific action type name (e.g., "Damage", "Move", "Respawn").
	ActionType string `json:"action_type,omitempty"`
	// Target is the player ID affected by this event.
	Target string `json:"target,omitempty"`
	// Delta is the change amount (negative for damage/LP loss, positive for heal/LP gain).
	Delta int `json:"delta,omitempty"`
	// Source is the source identifier (Buff ID, Item ID, Event ID).
	Source string `json:"source,omitempty"`
	// Metadata contains additional data with type-safe access and JSON serialization.
	Metadata *util.Metadata `json:"metadata,omitempty"`
}

// NewActionEntry creates a new LogEntry for an Action execution.
func NewActionEntry(actionType string, target string, delta int, source string) LogEntry {
	return LogEntry{
		Timestamp:  time.Now(),
		Type:       EntryTypeAction,
		ActionType: actionType,
		Target:     target,
		Delta:      delta,
		Source:     source,
		Metadata:   util.NewMetadata(),
	}
}

// NewActionEntryWithMetadata creates a new LogEntry with pre-populated metadata.
func NewActionEntryWithMetadata(actionType string, target string, delta int, source string, metadata *util.Metadata) LogEntry {
	return LogEntry{
		Timestamp:  time.Now(),
		Type:       EntryTypeAction,
		ActionType: actionType,
		Target:     target,
		Delta:      delta,
		Source:     source,
		Metadata:   metadata,
	}
}

// NewStateEntry creates a new LogEntry for a State transition.
func NewStateEntry(from, to string, playerID string) LogEntry {
	metadata := util.NewMetadata()
	metadata.SetString("from", from)
	metadata.SetString("to", to)

	return LogEntry{
		Timestamp: time.Now(),
		Type:      EntryTypeState,
		Target:    playerID,
		Metadata:  metadata,
	}
}
