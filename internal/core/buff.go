// Package core provides core data structures for the Paradiced game.
// This package has no internal dependencies - only pure data types.
package core

import (
	"github.com/b1tAction/paradiced/pkg/constants"
	"github.com/b1tAction/paradiced/pkg/id"
	"github.com/b1tAction/paradiced/pkg/util"
)

// ========== Buff Instance ==========

// Buff represents a buff instance attached to a player.
type Buff struct {
	Type         constants.BuffType `json:"type"`
	ID           id.BuffID          `json:"id"`            // Buff instance ID (UUID v7)
	Duration     int                `json:"duration"`      // Remaining duration (-1 for permanent)
	tickEligible bool               // Whether buff should be ticked at next turn end
	*util.Metadata `json:"metadata"` // Per-buff state storage (e.g. everyNTurns counter)
}

// NewBuff creates a new Buff instance with auto-generated UUID v7 ID.
func NewBuff(buffType constants.BuffType, duration int) *Buff {
	return &Buff{
		Type:     buffType,
		ID:       id.NewBuffID(),
		Duration: duration,
		Metadata: util.NewMetadata(),
	}
}

// NewBuffWithID creates a new Buff instance with a specific ID.
// Used for testing and special cases where ID needs to be controlled.
func NewBuffWithID(buffType constants.BuffType, buffID id.BuffID, duration int) *Buff {
	return &Buff{
		Type:     buffType,
		ID:       buffID,
		Duration: duration,
		Metadata: util.NewMetadata(),
	}
}

// IsActive checks if the buff is still active.
func (b *Buff) IsActive() bool {
	return b.Duration > 0 || b.Duration == -1
}

// TickDuration decrements duration if tick-eligible, and returns whether buff is still active.
// First call marks the buff as tick-eligible (without decrementing), so buffs acquired
// mid-turn survive their first turn-end tick. Duration is only decremented on subsequent calls.
func (b *Buff) TickDuration() bool {
	if b.Duration > 0 {
		if b.tickEligible {
			b.Duration--
		} else {
			b.tickEligible = true
		}
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