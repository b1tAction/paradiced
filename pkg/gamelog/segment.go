package gamelog

import "time"

// TurnSegment represents a single player's turn log entries.
// Used for client playback - each segment contains all events for one turn.
type TurnSegment struct {
	// Round is the current game round number.
	Round int `json:"round"`
	// Turn is the current turn index (0-3 for 4 players).
	Turn int `json:"turn"`
	// PlayerID is the player who took this turn.
	PlayerID string `json:"player_id"`
	// Entries is the list of log entries for this turn.
	Entries []LogEntry `json:"entries"`
	// StartTime is when this turn started.
	StartTime time.Time `json:"start_time"`
	// EndTime is when this turn ended (set when EndTurn is called).
	EndTime time.Time `json:"end_time,omitempty"`
}

// NewTurnSegment creates a new turn segment.
func NewTurnSegment(round, turn int, playerID string) *TurnSegment {
	return &TurnSegment{
		Round:     round,
		Turn:      turn,
		PlayerID:  playerID,
		Entries:   make([]LogEntry, 0),
		StartTime: time.Now(),
	}
}

// AddEntry adds a log entry to this segment.
func (s *TurnSegment) AddEntry(entry LogEntry) {
	s.Entries = append(s.Entries, entry)
}

// End marks the turn as ended and sets EndTime.
func (s *TurnSegment) End() {
	s.EndTime = time.Now()
}

// Len returns the number of entries in this segment.
func (s *TurnSegment) Len() int {
	return len(s.Entries)
}