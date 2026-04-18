// Package core provides core data structures for the Paradiced game.
// This package has no internal dependencies - only pure data types.
package core

import (
	"github.com/b1tAction/paradiced/pkg/constants"
	"github.com/b1tAction/paradiced/pkg/id"
)

// ========== Buff Instance ==========

// Buff represents a buff instance attached to a player.
type Buff struct {
	Type            constants.BuffType `json:"type"`
	ID              id.BuffID          `json:"id"` // Buff instance ID (UUID v7)
	Duration        int                `json:"duration"`    // Remaining duration (-1 for permanent)
	Charge          int                `json:"charge"`      // Charge count (for faction skills)
	SubscriptionIDs []string           `json:"subscription_ids"` // EventBus subscription IDs (UUID strings)
}

// NewBuff creates a new Buff instance with auto-generated UUID v7 ID.
func NewBuff(buffType constants.BuffType, duration int) *Buff {
	return &Buff{
		Type:            buffType,
		ID:              id.NewBuffID(),
		Duration:        duration,
		Charge:          0,
		SubscriptionIDs: make([]string, 0),
	}
}

// NewBuffWithID creates a new Buff instance with a specific ID.
// Used for testing and special cases where ID needs to be controlled.
func NewBuffWithID(buffType constants.BuffType, buffID id.BuffID, duration int) *Buff {
	return &Buff{
		Type:            buffType,
		ID:              buffID,
		Duration:        duration,
		Charge:          0,
		SubscriptionIDs: make([]string, 0),
	}
}

// IsActive checks if the buff is still active.
func (b *Buff) IsActive() bool {
	return b.Duration > 0 || b.Duration == -1 || b.Charge > 0
}

// TickDuration decrements duration and returns whether buff is still active.
func (b *Buff) TickDuration() bool {
	if b.Duration > 0 {
		b.Duration--
	}
	return b.IsActive()
}

// ========== Buff Definition (Static Metadata) ==========

// BuffDefinition contains static metadata for Buff display and classification.
// Effect logic is managed by engine layer's BuffHandlerConfig.
type BuffDefinition struct {
	Type        constants.BuffType   `json:"type"`
	Eval        constants.Evaluation `json:"evaluation"`    // Evaluation score for random draw
	EnglishName string               `json:"english_name"`  // English identifier (snake_case)
	Name        string               `json:"name"`          // Chinese display name
	Desc        string               `json:"desc"`          // Description text
	Duration    int                  `json:"duration"`      // Default duration (-1 for permanent)
}

// IsPositive checks if the buff is beneficial.
func (d *BuffDefinition) IsPositive() bool {
	return d.Eval.IsGood()
}

// IsNegative checks if the buff is harmful.
func (d *BuffDefinition) IsNegative() bool {
	return d.Eval.IsBad()
}