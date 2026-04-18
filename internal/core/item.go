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
	Type           constants.ItemType `json:"type"`
	ID             id.ItemID          `json:"id"` // Item instance ID (UUID v7)
	Usable         bool               `json:"usable"`       // Whether item can be used
	TargetID       string             `json:"target_id"`    // Target player ID (UUID string for protocol)
	SubscriptionID string             `json:"subscription_id"` // EventBus subscription ID (managed by engine)
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

// ========== Item Definition (Static Metadata) ==========

// ItemDefinition contains static metadata for Item display and classification.
// Effect logic is managed by engine layer's ItemHandlerConfig.
type ItemDefinition struct {
	Type        constants.ItemType   `json:"type"`
	Eval        constants.Evaluation `json:"evaluation"`    // Evaluation score for random draw
	EnglishName string               `json:"english_name"`  // English identifier (snake_case)
	Name        string               `json:"name"`          // Chinese display name
	Desc        string               `json:"desc"`          // Description text
	TargetSelf  bool                 `json:"target_self"`   // Can target self
	TargetOther bool                 `json:"target_other"`  // Can target other player
	Range       int                  `json:"range"`         // Target range (0 = any distance)
}

// CanTarget checks if the item can target a specific target type.
func (d *ItemDefinition) CanTarget(targetSelf bool) bool {
	if targetSelf {
		return d.TargetSelf
	}
	return d.TargetOther
}

// IsInRange checks if target is within item's effective range.
func (d *ItemDefinition) IsInRange(distance int) bool {
	if d.Range == 0 {
		return true // 0 means any distance
	}
	return distance <= d.Range
}