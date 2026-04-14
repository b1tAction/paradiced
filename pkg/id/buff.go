package id

// BuffID represents a unique buff instance identifier.
type BuffID struct{ ID }

const buffPrefix = "buff"

// NewBuffID generates a new BuffID with UUID v7.
func NewBuffID() BuffID {
	return BuffID{NewID(buffPrefix)}
}

// ParseBuffID parses a UUID string into BuffID.
func ParseBuffID(s string) (BuffID, error) {
	id, err := ParseID(buffPrefix, s)
	return BuffID{id}, err
}

// MustParseBuffID parses or panics (for testing).
func MustParseBuffID(s string) BuffID {
	return BuffID{MustParseID(buffPrefix, s)}
}

// ZeroBuffID returns an empty BuffID.
func ZeroBuffID() BuffID {
	return BuffID{ZeroID(buffPrefix)}
}

// UnmarshalJSON parses a pure UUID string into BuffID.
func (b *BuffID) UnmarshalJSON(data []byte) error {
	if err := b.ID.UnmarshalJSON(data); err != nil {
		return err
	}
	b.ID.prefix = buffPrefix
	return nil
}