// Package id provides typed identifiers for game entities.
// Each ID type wraps a UUID v7 with an internal prefix for debugging.
// JSON serialization outputs pure UUID for protocol compatibility.
package id

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// ID is the base structure for all typed identifiers.
// It contains a prefix for internal debugging and a UUID v7 for unique identification.
type ID struct {
	prefix string
	uuid   uuid.UUID
}

// NewID creates a new ID with the given prefix and a fresh UUID v7.
func NewID(prefix string) ID {
	return ID{
		prefix: prefix,
		uuid:   uuid.Must(uuid.NewV7()),
	}
}

// ParseID parses a UUID string into an ID with the given prefix.
// The input can be a pure UUID or a prefixed format (prefix-uuid).
// If prefixed format is provided, the prefix in the string is ignored.
func ParseID(prefix, s string) (ID, error) {
	// Handle prefixed format: extract UUID part
	var uuidStr string
	if strings.Contains(s, "-") && len(s) > 36 {
		// Format: prefix-uuid, extract last 36 chars
		parts := strings.SplitN(s, "-", 2)
		if len(parts) == 2 && len(parts[1]) == 36 {
			uuidStr = parts[1]
		} else {
			uuidStr = s // fallback: try parsing whole string
		}
	} else {
		uuidStr = s
	}

	u, err := uuid.Parse(uuidStr)
	if err != nil {
		return ID{}, err
	}
	return ID{prefix: prefix, uuid: u}, nil
}

// MustParseID parses a UUID string or panics.
func MustParseID(prefix, s string) ID {
	id, err := ParseID(prefix, s)
	if err != nil {
		panic(err)
	}
	return id
}

// String returns the prefixed format for debugging: "prefix-uuid".
func (id ID) String() string {
	if id.uuid == uuid.Nil {
		return id.prefix + "-nil"
	}
	return id.prefix + "-" + id.uuid.String()
}

// UUID returns the pure UUID string for protocol transmission.
func (id ID) UUID() string {
	if id.uuid == uuid.Nil {
		return ""
	}
	return id.uuid.String()
}

// MarshalJSON outputs the pure UUID for JSON serialization.
func (id ID) MarshalJSON() ([]byte, error) {
	if id.uuid == uuid.Nil {
		return json.Marshal("")
	}
	return json.Marshal(id.uuid.String())
}

// UnmarshalJSON parses a pure UUID string.
// The prefix is set by the specific ID type, not from the input.
func (id *ID) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	if s == "" {
		id.uuid = uuid.Nil
		return nil
	}
	u, err := uuid.Parse(s)
	if err != nil {
		return err
	}
	id.uuid = u
	// Note: prefix is set by the specific ID type's UnmarshalJSON wrapper
	return nil
}

// IsZero checks if the ID is empty/nil.
func (id ID) IsZero() bool {
	return id.uuid == uuid.Nil
}

// IsValid checks if the ID has a valid UUID.
func (id ID) IsValid() bool {
	return !id.IsZero()
}

// Equal checks if two IDs have the same UUID.
func (id ID) Equal(other ID) bool {
	return id.uuid == other.uuid
}

// ZeroID returns a zero/empty ID for the given prefix.
func ZeroID(prefix string) ID {
	return ID{prefix: prefix, uuid: uuid.Nil}
}

// TestUUID generates a valid UUID string for testing purposes.
// Format: 00000000-0000-0000-0000-000000000XXX where XXX is the index.
// This is useful for tests that need valid UUID format without external dependencies.
func TestUUID(index int) string {
	return fmt.Sprintf("00000000-0000-0000-0000-%012d", index)
}