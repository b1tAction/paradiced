package id

// MiniGameInstanceID represents a unique mini-game session instance identifier.
// Used as Colyseus filterBy key to ensure all players enter the same room.
type MiniGameInstanceID struct{ ID }

const minigameInstancePrefix = "mg_inst"

// NewMiniGameInstanceID generates a new MiniGameInstanceID with UUID v7.
func NewMiniGameInstanceID() MiniGameInstanceID {
	return MiniGameInstanceID{NewID(minigameInstancePrefix)}
}

// ParseMiniGameInstanceID parses a UUID string into MiniGameInstanceID.
func ParseMiniGameInstanceID(s string) (MiniGameInstanceID, error) {
	id, err := ParseID(minigameInstancePrefix, s)
	return MiniGameInstanceID{id}, err
}

// MustParseMiniGameInstanceID parses or panics (for testing).
func MustParseMiniGameInstanceID(s string) MiniGameInstanceID {
	return MiniGameInstanceID{MustParseID(minigameInstancePrefix, s)}
}

// ZeroMiniGameInstanceID returns an empty MiniGameInstanceID.
func ZeroMiniGameInstanceID() MiniGameInstanceID {
	return MiniGameInstanceID{ZeroID(minigameInstancePrefix)}
}

// UnmarshalJSON parses a pure UUID string into MiniGameInstanceID.
func (m *MiniGameInstanceID) UnmarshalJSON(data []byte) error {
	if err := m.ID.UnmarshalJSON(data); err != nil {
		return err
	}
	m.ID.prefix = minigameInstancePrefix
	return nil
}