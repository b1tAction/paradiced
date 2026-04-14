package id

// PlayerID represents a unique player identifier.
type PlayerID struct{ ID }

const playerPrefix = "player"

// NewPlayerID generates a new PlayerID with UUID v7.
func NewPlayerID() PlayerID {
	return PlayerID{NewID(playerPrefix)}
}

// ParsePlayerID parses a UUID string into PlayerID.
// Accepts pure UUID or prefixed format "player-uuid".
func ParsePlayerID(s string) (PlayerID, error) {
	id, err := ParseID(playerPrefix, s)
	return PlayerID{id}, err
}

// MustParsePlayerID parses or panics (for testing).
func MustParsePlayerID(s string) PlayerID {
	return PlayerID{MustParseID(playerPrefix, s)}
}

// ZeroPlayerID returns an empty PlayerID.
func ZeroPlayerID() PlayerID {
	return PlayerID{ZeroID(playerPrefix)}
}

// UnmarshalJSON parses a pure UUID string into PlayerID.
func (p *PlayerID) UnmarshalJSON(data []byte) error {
	if err := p.ID.UnmarshalJSON(data); err != nil {
		return err
	}
	p.ID.prefix = playerPrefix
	return nil
}