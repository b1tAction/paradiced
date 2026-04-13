// Package event provides Event related data structures and registry.
// This package is independently usable via Direct Import.
package event

import (
	"github.com/b1tAction/Fated/internal/core/buff"
	"github.com/b1tAction/Fated/internal/core/types"
)

// ========== Event Type Definitions ==========

type EventType int

const (
	EventTypeNone EventType = iota

	// Good events (Evaluation > 65)
	EventTypeHerb         // Herb: HP+1
	EventTypeMilkTea      // MilkTea: LP+1
	EventTypeRelic        // Relic: item draw
	EventTypeDivineBless  // DivineBless: Divine buff

	// Neutral events (Evaluation 41~65)
	EventTypeExchange     // Exchange: swap position with random player
	EventTypeHiddenBuff   // HiddenBuff: Hidden buff
	EventTypeTasteTest    // TasteTest: random buff (Corrupt/Rain)

	// Bad events (Evaluation ≤ 40)
	EventTypeMosquito     // Mosquito: HP-1
	EventTypeGhostHit     // GhostHit: HP-1
	EventTypeDogPoop      // DogPoop: LP-1
	EventTypeThief        // Thief: random item loss
	EventTypeCurseBuddha  // CurseBuddha: Curse buff
	EventTypeLostWay      // LostWay: Lost buff
	EventTypeThunder      // Thunder: HP to 0
)

// IsValid checks if the Event type is valid.
func (et EventType) IsValid() bool {
	return et > EventTypeNone && et <= EventTypeThunder
}

// String returns the Event type name from GlobalEventRegistry.
func (et EventType) String() string {
	return GlobalEventRegistry.GetEventString(et)
}

// ========== Event Definition ==========

type EventDefinition struct {
	Type          EventType           `json:"type"`
	Eval          types.Evaluation    `json:"evaluation"`
	EnglishName   string              `json:"english_name"` // English identifier (for String())
	Name          string              `json:"name"`         // Chinese display name
	Desc          string              `json:"desc"`
	HPChange      int                 `json:"hp_change"`
	LPChange      int                 `json:"lp_change"`
	BuffType      buff.BuffType       `json:"buff_type"`
	SpecialEffect types.SpecialEffect `json:"special_effect"` // Special effect type
}

// ========== Event Registry ==========

// EventRegistry is the registry for Event definitions.
type EventRegistry struct {
	defs    map[EventType]*EventDefinition
	strings map[EventType]string // English identifier
	names   map[EventType]string // Chinese name
	evals   map[EventType]types.Evaluation

	// Category lists (auto-generated)
	goodEvents    []EventType
	badEvents     []EventType
	neutralEvents []EventType
}

// NewEventRegistry creates a new Event registry.
func NewEventRegistry() *EventRegistry {
	return &EventRegistry{
		defs:          make(map[EventType]*EventDefinition),
		strings:       make(map[EventType]string),
		names:         make(map[EventType]string),
		evals:         make(map[EventType]types.Evaluation),
		goodEvents:    make([]EventType, 0),
		badEvents:     make([]EventType, 0),
		neutralEvents: make([]EventType, 0),
	}
}

// RegisterEvent registers an Event definition.
func (r *EventRegistry) RegisterEvent(def *EventDefinition) {
	if def == nil || def.Type == EventTypeNone {
		return
	}

	r.defs[def.Type] = def
	r.strings[def.Type] = def.EnglishName
	r.names[def.Type] = def.Name
	r.evals[def.Type] = def.Eval

	// Auto-classify by Evaluation
	if def.Eval.IsGood() {
		r.goodEvents = append(r.goodEvents, def.Type)
	} else if def.Eval.IsBad() {
		r.badEvents = append(r.badEvents, def.Type)
	} else {
		r.neutralEvents = append(r.neutralEvents, def.Type)
	}
}

// GetEventDefinition returns the Event definition by type.
func (r *EventRegistry) GetEventDefinition(et EventType) *EventDefinition {
	if def, ok := r.defs[et]; ok {
		return def
	}
	return nil
}

// GetEventString returns the Event English identifier.
func (r *EventRegistry) GetEventString(et EventType) string {
	if name, ok := r.strings[et]; ok {
		return name
	}
	return "Unknown"
}

// GetEventName returns the Event Chinese display name.
func (r *EventRegistry) GetEventName(et EventType) string {
	if name, ok := r.names[et]; ok {
		return name
	}
	return "未知"
}

// GetEventEvaluation returns the Event evaluation score.
func (r *EventRegistry) GetEventEvaluation(et EventType) types.Evaluation {
	if eval, ok := r.evals[et]; ok {
		return eval
	}
	return types.EvaluationNeutral
}

// GetAllEventTypes returns all registered Event types.
func (r *EventRegistry) GetAllEventTypes() []EventType {
	result := make([]EventType, 0, len(r.defs))
	for et := range r.defs {
		result = append(result, et)
	}
	return result
}

// GetEventTypesByCategory returns Event types by category.
func (r *EventRegistry) GetEventTypesByCategory(category string) []EventType {
	switch category {
	case "Good":
		return r.goodEvents
	case "Bad":
		return r.badEvents
	case "Neutral":
		return r.neutralEvents
	}
	return r.GetAllEventTypes()
}

// GetAllEventDefinitions returns all Event definitions.
func (r *EventRegistry) GetAllEventDefinitions() []*EventDefinition {
	defs := make([]*EventDefinition, 0, len(r.defs))
	for _, def := range r.defs {
		defs = append(defs, def)
	}
	return defs
}

// GetEventTypesByEvaluationRange returns Events within the specified Evaluation range.
func (r *EventRegistry) GetEventTypesByEvaluationRange(minEval, maxEval types.Evaluation) []EventType {
	var result []EventType
	for et, eval := range r.evals {
		if eval >= minEval && eval <= maxEval {
			result = append(result, et)
		}
	}
	return result
}

// ========== Global Registry Access Functions ==========

// GetEventDefinition returns the Event definition from GlobalEventRegistry.
func GetEventDefinition(et EventType) *EventDefinition {
	return GlobalEventRegistry.GetEventDefinition(et)
}

// GetEventString returns the Event name string from GlobalEventRegistry.
func GetEventString(et EventType) string {
	return GlobalEventRegistry.GetEventString(et)
}

// GetEventEvaluation returns the Event evaluation score from GlobalEventRegistry.
func GetEventEvaluation(et EventType) types.Evaluation {
	return GlobalEventRegistry.GetEventEvaluation(et)
}

// GetAllEventTypes returns all registered Event types.
func GetAllEventTypes() []EventType {
	return GlobalEventRegistry.GetAllEventTypes()
}

// GetEventTypesByCategory returns Event types by category.
func GetEventTypesByCategory(category string) []EventType {
	return GlobalEventRegistry.GetEventTypesByCategory(category)
}

// GetAllEventDefinitions returns all Event definitions.
func GetAllEventDefinitions() []*EventDefinition {
	return GlobalEventRegistry.GetAllEventDefinitions()
}