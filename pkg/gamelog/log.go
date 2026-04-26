package gamelog

import (
	"encoding/json"
	"sync"
)

// GameLog is the unified game event log for client playback.
// It maintains a list of turn segments, each containing events for one turn.
type GameLog struct {
	mutex              sync.RWMutex
	segments           []*TurnSegment
	current            *TurnSegment // Current turn segment (nil if no turn active)
	lastBroadcastIndex int          // Tracks which entries have been broadcast via StateSync
}

// NewGameLog creates a new empty game log.
func NewGameLog() *GameLog {
	return &GameLog{
		segments: make([]*TurnSegment, 0),
		current:  nil,
	}
}

// StartTurn starts a new turn segment.
// All subsequent AddEntry calls will add to this segment until EndTurn is called.
func (l *GameLog) StartTurn(round, turn int, playerID string) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	// End current turn if any (should not happen in normal flow)
	if l.current != nil {
		l.current.End()
	}

	// Create new segment
	l.current = NewTurnSegment(round, turn, playerID)
	l.segments = append(l.segments, l.current)
	// Reset broadcast index for new turn
	l.lastBroadcastIndex = 0
}

// EndTurn ends the current turn segment.
// After this call, AddEntry will not add to any segment until StartTurn is called again.
func (l *GameLog) EndTurn() {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if l.current != nil {
		l.current.End()
		l.current = nil
	}
}

// AddEntry adds a log entry to the current turn segment.
// If no turn is active (StartTurn not called), the entry is discarded.
func (l *GameLog) AddEntry(entry LogEntry) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if l.current != nil {
		l.current.AddEntry(entry)
	}
}

// GetCurrentTurnEntries returns entries for the current turn.
// Returns nil if no turn is active.
func (l *GameLog) GetCurrentTurnEntries() []LogEntry {
	l.mutex.RLock()
	defer l.mutex.RUnlock()

	if l.current == nil {
		return nil
	}
	return l.current.Entries
}

// GetTurnSegments returns all turn segments.
func (l *GameLog) GetTurnSegments() []*TurnSegment {
	l.mutex.RLock()
	defer l.mutex.RUnlock()

	// Return copy to avoid race conditions
	result := make([]*TurnSegment, len(l.segments))
	copy(result, l.segments)
	return result
}

// GetSegment returns the segment for a specific round/turn.
// Returns nil if not found.
func (l *GameLog) GetSegment(round, turn int) *TurnSegment {
	l.mutex.RLock()
	defer l.mutex.RUnlock()

	for _, seg := range l.segments {
		if seg.Round == round && seg.Turn == turn {
			return seg
		}
	}
	return nil
}

// ToJSON serializes all segments for client transmission.
func (l *GameLog) ToJSON() ([]byte, error) {
	l.mutex.RLock()
	defer l.mutex.RUnlock()

	return json.Marshal(map[string]interface{}{
		"segments": l.segments,
	})
}

// Clear removes all segments and resets the log.
func (l *GameLog) Clear() {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	l.segments = make([]*TurnSegment, 0)
	l.current = nil
}

// LogStateTransition records a state transition (shortcut method).
func (l *GameLog) LogStateTransition(from, to string, playerID string) {
	entry := NewStateEntry(from, to, playerID)
	l.AddEntry(entry)
}

// Len returns the total number of entries across all segments.
func (l *GameLog) Len() int {
	l.mutex.RLock()
	defer l.mutex.RUnlock()

	total := 0
	for _, seg := range l.segments {
		total += seg.Len()
	}
	return total
}

// IsTurnActive returns true if a turn segment is currently active.
func (l *GameLog) IsTurnActive() bool {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	return l.current != nil
}

// GetNewEntries returns entries added since the last broadcast.
// Used by StateSync to include incremental LogEntry data.
func (l *GameLog) GetNewEntries() []LogEntry {
	l.mutex.RLock()
	defer l.mutex.RUnlock()

	if l.current == nil {
		return nil
	}
	if l.lastBroadcastIndex >= len(l.current.Entries) {
		return nil
	}
	result := make([]LogEntry, len(l.current.Entries)-l.lastBroadcastIndex)
	copy(result, l.current.Entries[l.lastBroadcastIndex:])
	return result
}

// MarkBroadcasted marks all current entries as broadcasted.
// Should be called after GetNewEntries to update the tracking index.
func (l *GameLog) MarkBroadcasted() {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if l.current != nil {
		l.lastBroadcastIndex = len(l.current.Entries)
	}
}

// GetAllCurrentEntries returns all entries for the current turn.
// Used by FullSync to send complete turn data to reconnecting players.
func (l *GameLog) GetAllCurrentEntries() []LogEntry {
	l.mutex.RLock()
	defer l.mutex.RUnlock()

	if l.current == nil {
		return nil
	}
	result := make([]LogEntry, len(l.current.Entries))
	copy(result, l.current.Entries)
	return result
}