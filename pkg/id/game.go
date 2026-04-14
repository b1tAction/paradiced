package id

// GameID represents a unique game/match instance identifier.
type GameID struct{ ID }

const gamePrefix = "game"

// NewGameID generates a new GameID with UUID v7.
func NewGameID() GameID {
	return GameID{NewID(gamePrefix)}
}

// ParseGameID parses a UUID string into GameID.
func ParseGameID(s string) (GameID, error) {
	id, err := ParseID(gamePrefix, s)
	return GameID{id}, err
}

// MustParseGameID parses or panics (for testing).
func MustParseGameID(s string) GameID {
	return GameID{MustParseID(gamePrefix, s)}
}

// ZeroGameID returns an empty GameID.
func ZeroGameID() GameID {
	return GameID{ZeroID(gamePrefix)}
}

// UnmarshalJSON parses a pure UUID string into GameID.
func (g *GameID) UnmarshalJSON(data []byte) error {
	if err := g.ID.UnmarshalJSON(data); err != nil {
		return err
	}
	g.ID.prefix = gamePrefix
	return nil
}