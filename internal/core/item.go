// Package core provides core data structures for the Paradiced game.
// This package has no internal dependencies - only pure data types.
package core

import (
	"github.com/b1tAction/paradiced/pkg/constants"
	"github.com/b1tAction/paradiced/pkg/id"
)

// ========== Item Instance ==========

// Item represents an item instance in player's inventory.
type Item struct {
	Type     constants.ItemType `json:"type"`
	ID       id.ItemID          `json:"id"` // Item instance ID (UUID v7)
	Usable   bool               `json:"usable"`    // Whether item can be used
	TargetID string             `json:"target_id"` // Target player ID (UUID string for protocol)
}

// NewItem creates a new Item instance with auto-generated UUID v7 ID.
func NewItem(itemType constants.ItemType) *Item {
	return &Item{
		Type:   itemType,
		ID:     id.NewItemID(),
		Usable: true,
	}
}

// NewItemWithID creates a new Item instance with a specific ID.
// Used for testing and special cases where ID needs to be controlled.
func NewItemWithID(itemType constants.ItemType, itemID id.ItemID) *Item {
	return &Item{
		Type:   itemType,
		ID:     itemID,
		Usable: true,
	}
}

