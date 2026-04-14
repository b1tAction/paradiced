package id

// DecisionID represents a unique decision request identifier.
type DecisionID struct{ ID }

const decPrefix = "dec"

// NewDecisionID generates a new DecisionID with UUID v7.
func NewDecisionID() DecisionID {
	return DecisionID{NewID(decPrefix)}
}

// ParseDecisionID parses a UUID string into DecisionID.
func ParseDecisionID(s string) (DecisionID, error) {
	id, err := ParseID(decPrefix, s)
	return DecisionID{id}, err
}

// MustParseDecisionID parses or panics (for testing).
func MustParseDecisionID(s string) DecisionID {
	return DecisionID{MustParseID(decPrefix, s)}
}

// ZeroDecisionID returns an empty DecisionID.
func ZeroDecisionID() DecisionID {
	return DecisionID{ZeroID(decPrefix)}
}

// UnmarshalJSON parses a pure UUID string into DecisionID.
func (d *DecisionID) UnmarshalJSON(data []byte) error {
	if err := d.ID.UnmarshalJSON(data); err != nil {
		return err
	}
	d.ID.prefix = decPrefix
	return nil
}