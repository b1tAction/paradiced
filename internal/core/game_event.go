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
	Executed   bool                  `json:"executed"`  // Whether event has been executed
}

// NewGameEvent creates a new GameEvent instance with auto-generated UUID v7 ID.
func NewGameEvent(eventType constants.EventType) *GameEvent {
	return &GameEvent{
		Type:     eventType,
		ID:       id.NewEventID(),
		Executed: false,
	}
}

// ========== Event Definition (Static Metadata) ==========

// EventDefinition contains static metadata for Event display and classification.
// Effect logic is managed by engine layer's EventHandlerConfig.
type EventDefinition struct {
	Type        constants.EventType   `json:"type"`
	Eval        constants.Evaluation  `json:"evaluation"`    // Evaluation score for random draw
	EnglishName string                `json:"english_name"`  // English identifier (snake_case)
	Name        string                `json:"name"`          // Chinese display name
	Desc        string                `json:"desc"`          // Description text
}

// IsGood checks if the event is beneficial (Evaluation > 65).
func (d *EventDefinition) IsGood() bool {
	return d.Eval.IsGood()
}

// IsBad checks if the event is harmful (Evaluation <= 40).
func (d *EventDefinition) IsBad() bool {
	return d.Eval.IsBad()
}

// IsNeutral checks if the event is neutral (Evaluation 41-65).
func (d *EventDefinition) IsNeutral() bool {
	return !d.IsGood() && !d.IsBad()
}