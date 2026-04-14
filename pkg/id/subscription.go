package id

// SubscriptionID represents a unique EventBus subscription identifier.
type SubscriptionID struct{ ID }

const subPrefix = "sub"

// NewSubscriptionID generates a new SubscriptionID with UUID v7.
func NewSubscriptionID() SubscriptionID {
	return SubscriptionID{NewID(subPrefix)}
}

// ParseSubscriptionID parses a UUID string into SubscriptionID.
func ParseSubscriptionID(s string) (SubscriptionID, error) {
	id, err := ParseID(subPrefix, s)
	return SubscriptionID{id}, err
}

// MustParseSubscriptionID parses or panics (for testing).
func MustParseSubscriptionID(s string) SubscriptionID {
	return SubscriptionID{MustParseID(subPrefix, s)}
}

// ZeroSubscriptionID returns an empty SubscriptionID.
func ZeroSubscriptionID() SubscriptionID {
	return SubscriptionID{ZeroID(subPrefix)}
}

// UnmarshalJSON parses a pure UUID string into SubscriptionID.
func (s *SubscriptionID) UnmarshalJSON(data []byte) error {
	if err := s.ID.UnmarshalJSON(data); err != nil {
		return err
	}
	s.ID.prefix = subPrefix
	return nil
}