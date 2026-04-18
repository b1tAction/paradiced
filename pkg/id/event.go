package id

// EventID represents a unique event instance identifier.
type EventID struct{ ID }

const eventPrefix = "event"

// NewEventID generates a new EventID with UUID v7.
func NewEventID() EventID {
	return EventID{NewID(eventPrefix)}
}

// ParseEventID parses a UUID string into EventID.
func ParseEventID(s string) (EventID, error) {
	id, err := ParseID(eventPrefix, s)
	return EventID{id}, err
}

// MustParseEventID parses or panics (for testing).
func MustParseEventID(s string) EventID {
	return EventID{MustParseID(eventPrefix, s)}
}

// ZeroEventID returns an empty EventID.
func ZeroEventID() EventID {
	return EventID{ZeroID(eventPrefix)}
}

// UnmarshalJSON parses a pure UUID string into EventID.
func (e *EventID) UnmarshalJSON(data []byte) error {
	if err := e.ID.UnmarshalJSON(data); err != nil {
		return err
	}
	e.ID.prefix = eventPrefix
	return nil
}