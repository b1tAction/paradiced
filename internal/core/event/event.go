// Package event provides Event related data structures and registry.
// This package is independently usable via Direct Import.
package event

import (
	"github.com/b1tAction/paradiced/pkg/constants"
)

// ========== Event Definition ==========

type EventDefinition struct {
	Type          constants.EventType     `json:"type"`
	Eval          constants.Evaluation    `json:"evaluation"`
	EnglishName   string                  `json:"english_name"` // English identifier (for String())
	Name          string                  `json:"name"`         // Chinese display name
	Desc          string                  `json:"desc"`
	HPChange      int                     `json:"hp_change"`
	LPChange      int                     `json:"lp_change"`
	BuffType      constants.BuffType      `json:"buff_type"`
	SpecialEffect constants.SpecialEffect `json:"special_effect"` // Special effect type
}

// ========== Event Registry ==========

// EventRegistry is the registry for Event definitions.
type EventRegistry struct {
	defs    map[constants.EventType]*EventDefinition
	strings map[constants.EventType]string // English identifier
	names   map[constants.EventType]string // Chinese name
	evals   map[constants.EventType]constants.Evaluation

	// Category lists (auto-generated)
	goodEvents    []constants.EventType
	badEvents     []constants.EventType
	neutralEvents []constants.EventType
}

// NewEventRegistry creates a new Event registry.
func NewEventRegistry() *EventRegistry {
	return &EventRegistry{
		defs:          make(map[constants.EventType]*EventDefinition),
		strings:       make(map[constants.EventType]string),
		names:         make(map[constants.EventType]string),
		evals:         make(map[constants.EventType]constants.Evaluation),
		goodEvents:    make([]constants.EventType, 0),
		badEvents:     make([]constants.EventType, 0),
		neutralEvents: make([]constants.EventType, 0),
	}
}

// RegisterEvent registers an Event definition.
func (r *EventRegistry) RegisterEvent(def *EventDefinition) {
	if def == nil || def.Type == constants.EventTypeNone {
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
func (r *EventRegistry) GetEventDefinition(et constants.EventType) *EventDefinition {
	if def, ok := r.defs[et]; ok {
		return def
	}
	return nil
}

// GetEventString returns the Event English identifier.
func (r *EventRegistry) GetEventString(et constants.EventType) string {
	if name, ok := r.strings[et]; ok {
		return name
	}
	return "Unknown"
}

// GetEventName returns the Event Chinese display name.
func (r *EventRegistry) GetEventName(et constants.EventType) string {
	if name, ok := r.names[et]; ok {
		return name
	}
	return "未知"
}

// GetEventEvaluation returns the Event evaluation score.
func (r *EventRegistry) GetEventEvaluation(et constants.EventType) constants.Evaluation {
	if eval, ok := r.evals[et]; ok {
		return eval
	}
	return constants.EvaluationNeutral
}

// GetAllEventTypes returns all registered Event types.
func (r *EventRegistry) GetAllEventTypes() []constants.EventType {
	result := make([]constants.EventType, 0, len(r.defs))
	for et := range r.defs {
		result = append(result, et)
	}
	return result
}

// GetEventTypesByCategory returns Event types by category.
func (r *EventRegistry) GetEventTypesByCategory(category string) []constants.EventType {
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
func (r *EventRegistry) GetEventTypesByEvaluationRange(minEval, maxEval constants.Evaluation) []constants.EventType {
	var result []constants.EventType
	for et, eval := range r.evals {
		if eval >= minEval && eval <= maxEval {
			result = append(result, et)
		}
	}
	return result
}

// ========== Global Registry Access Functions ==========

// GetEventDefinition returns the Event definition from GlobalEventRegistry.
func GetEventDefinition(et constants.EventType) *EventDefinition {
	return GlobalEventRegistry.GetEventDefinition(et)
}

// GetEventString returns the Event name string from GlobalEventRegistry.
func GetEventString(et constants.EventType) string {
	return GlobalEventRegistry.GetEventString(et)
}

// GetEventEvaluation returns the Event evaluation score from GlobalEventRegistry.
func GetEventEvaluation(et constants.EventType) constants.Evaluation {
	return GlobalEventRegistry.GetEventEvaluation(et)
}

// GetAllEventTypes returns all registered Event types.
func GetAllEventTypes() []constants.EventType {
	return GlobalEventRegistry.GetAllEventTypes()
}

// GetEventTypesByCategory returns Event types by category.
func GetEventTypesByCategory(category string) []constants.EventType {
	return GlobalEventRegistry.GetEventTypesByCategory(category)
}

// GetAllEventDefinitions returns all Event definitions.
func GetAllEventDefinitions() []*EventDefinition {
	return GlobalEventRegistry.GetAllEventDefinitions()
}

// GetEventName returns the Event Chinese display name from GlobalEventRegistry.
func GetEventName(et constants.EventType) string {
	return GlobalEventRegistry.GetEventName(et)
}
