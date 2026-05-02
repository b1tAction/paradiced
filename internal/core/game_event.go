// Package core provides core data structures for the Paradiced game.
// This package has no internal dependencies - only pure data types.
package core

import (
	"github.com/b1tAction/paradiced/pkg/constants"
	"github.com/b1tAction/paradiced/pkg/id"
)

// ========== GameEvent Instance ==========

// GameEvent represents a random event drawn during gameplay.
// Note: Named GameEvent to avoid conflict with internal/event package.
type GameEvent struct {
	Type       constants.EventType   `json:"type"`
	ID         id.EventID            `json:"id"` // Event instance ID (UUID v7)
	TargetID   string                `json:"target_id"` // Target player ID (UUID string)
}

// NewGameEvent creates a new GameEvent instance with auto-generated UUID v7 ID.
func NewGameEvent(eventType constants.EventType) *GameEvent {
	return &GameEvent{
		Type:     eventType,
		ID:       id.NewEventID(),
	}
}