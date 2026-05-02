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

// MarkTickEligible marks the buff as eligible for duration ticking at TurnEnd.
// Called automatically by buff handler closures when handler executes successfully,
// ensuring mid-turn buff effects (e.g., Curse/Divine at PhasePostBuffApplied) are
// also decremented at this turn's end rather than getting an extra free turn.
func (b *Buff) MarkTickEligible() {
	b.tickEligible = true
}

// TickEligible returns whether the buff is eligible for duration ticking.
func (b *Buff) TickEligible() bool {
	return b.tickEligible
}

// TickDuration decrements duration and returns whether buff is still active.
// Buffs are marked tick-eligible at TurnUpkeep (BeforeTurn), so this always
// decrements when called at TurnEnd. Duration=-1 buffs (permanent) are never decremented.
func (b *Buff) TickDuration() bool {
	if b.Duration > 0 && b.tickEligible {
		b.Duration--
	}
	return b.IsActive()
}