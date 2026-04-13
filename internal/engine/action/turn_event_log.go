package action

import "encoding/json"

// TurnEventLog collects events during a turn for client playback.
// The client uses this log to play animations in sequence.
type TurnEventLog struct {
	entries []TurnEventLogEntry
}

// NewTurnEventLog creates a new empty event log.
func NewTurnEventLog() *TurnEventLog {
	return &TurnEventLog{
		entries: make([]TurnEventLogEntry, 0),
	}
}

// AddEntry adds an event entry to the log.
func (l *TurnEventLog) AddEntry(entry TurnEventLogEntry) {
	l.entries = append(l.entries, entry)
}

// Entries returns all log entries.
func (l *TurnEventLog) Entries() []TurnEventLogEntry {
	return l.entries
}

// ToJSON serializes the log for client transmission.
func (l *TurnEventLog) ToJSON() ([]byte, error) {
	return json.Marshal(l.entries)
}

// Clear resets the log for a new turn.
func (l *TurnEventLog) Clear() {
	l.entries = make([]TurnEventLogEntry, 0)
}

// Len returns the number of entries in the log.
func (l *TurnEventLog) Len() int {
	return len(l.entries)
}