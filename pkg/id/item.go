package id

// ItemID represents a unique item instance identifier.
type ItemID struct{ ID }

const itemPrefix = "item"

// NewItemID generates a new ItemID with UUID v7.
func NewItemID() ItemID {
	return ItemID{NewID(itemPrefix)}
}

// ParseItemID parses a UUID string into ItemID.
func ParseItemID(s string) (ItemID, error) {
	id, err := ParseID(itemPrefix, s)
	return ItemID{id}, err
}

// MustParseItemID parses or panics (for testing).
func MustParseItemID(s string) ItemID {
	return ItemID{MustParseID(itemPrefix, s)}
}

// ZeroItemID returns an empty ItemID.
func ZeroItemID() ItemID {
	return ItemID{ZeroID(itemPrefix)}
}

// UnmarshalJSON parses a pure UUID string into ItemID.
func (i *ItemID) UnmarshalJSON(data []byte) error {
	if err := i.ID.UnmarshalJSON(data); err != nil {
		return err
	}
	i.ID.prefix = itemPrefix
	return nil
}