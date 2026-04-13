package core

import (
	"github.com/b1tAction/Fated/pkg/event"
)

// EventHandler is a highly customized Buff/Item/Event effect handler function.
// Parameters:
//   - phase: currently triggered Phase
//   - ctx: event context, containing Player, Data etc.
type EventHandler func(phase event.Phase, ctx *event.Context)

// DefinitionRegistry is the unified registry for all game definitions.
// Provides single-source-of-truth for Buff, Event, Item definitions.
type DefinitionRegistry struct {
	// Definition storage
	buffDefs  map[BuffType]*BuffDefinition
	eventDefs map[EventType]*EventDefinition
	itemDefs  map[ItemType]*ItemDefinition

	// Handler storage
	buffHandlers map[BuffType]EventHandler

	// Cached mappings (auto-generated from definitions)
	buffStrings    map[BuffType]string // English identifier
	buffNames      map[BuffType]string // Chinese name
	buffEvals      map[BuffType]Evaluation
	eventStrings   map[EventType]string // English identifier
	eventNames     map[EventType]string // Chinese name
	eventEvals     map[EventType]Evaluation
	itemStrings    map[ItemType]string // English identifier
	itemNames      map[ItemType]string // Chinese name
	itemEvals      map[ItemType]Evaluation

	// Category lists (auto-generated)
	goodBuffs    []BuffType
	badBuffs     []BuffType
	neutralBuffs []BuffType

	goodEvents    []EventType
	badEvents     []EventType
	neutralEvents []EventType

	goodItems    []ItemType
	neutralItems []ItemType
	badItems     []ItemType
}

// NewDefinitionRegistry creates a new definition registry.
func NewDefinitionRegistry() *DefinitionRegistry {
	return &DefinitionRegistry{
		buffDefs:      make(map[BuffType]*BuffDefinition),
		eventDefs:     make(map[EventType]*EventDefinition),
		itemDefs:      make(map[ItemType]*ItemDefinition),
		buffHandlers:  make(map[BuffType]EventHandler),
		buffStrings:   make(map[BuffType]string),
		buffNames:     make(map[BuffType]string),
		buffEvals:     make(map[BuffType]Evaluation),
		eventStrings:  make(map[EventType]string),
		eventNames:    make(map[EventType]string),
		eventEvals:    make(map[EventType]Evaluation),
		itemStrings:   make(map[ItemType]string),
		itemNames:     make(map[ItemType]string),
		itemEvals:     make(map[ItemType]Evaluation),
		goodBuffs:     make([]BuffType, 0),
		badBuffs:      make([]BuffType, 0),
		neutralBuffs:  make([]BuffType, 0),
		goodEvents:    make([]EventType, 0),
		badEvents:     make([]EventType, 0),
		neutralEvents: make([]EventType, 0),
		goodItems:     make([]ItemType, 0),
		neutralItems:  make([]ItemType, 0),
		badItems:      make([]ItemType, 0),
	}
}

// ========== Buff Registration ==========

// RegisterBuff registers a Buff definition with optional handler.
// Automatically generates String, Evaluation mappings and category classification.
func (r *DefinitionRegistry) RegisterBuff(def *BuffDefinition, handler EventHandler) {
	if def == nil || def.Type == BuffTypeNone {
		return
	}

	// Store definition
	r.buffDefs[def.Type] = def

	// Auto-generate mappings
	// Use EnglishName for String(), Name for display
	r.buffStrings[def.Type] = def.EnglishName
	r.buffNames[def.Type] = def.Name
	r.buffEvals[def.Type] = def.Eval

	// Auto-classify by Evaluation
	if def.Eval.IsGood() {
		r.goodBuffs = append(r.goodBuffs, def.Type)
	} else if def.Eval.IsBad() {
		r.badBuffs = append(r.badBuffs, def.Type)
	} else {
		r.neutralBuffs = append(r.neutralBuffs, def.Type)
	}

	// Store handler (optional)
	if handler != nil {
		r.buffHandlers[def.Type] = handler
	}
}

// GetBuffDefinition returns the Buff definition by type.
func (r *DefinitionRegistry) GetBuffDefinition(bt BuffType) *BuffDefinition {
	if def, ok := r.buffDefs[bt]; ok {
		return def
	}
	return nil
}

// GetBuffString returns the Buff English identifier.
func (r *DefinitionRegistry) GetBuffString(bt BuffType) string {
	if name, ok := r.buffStrings[bt]; ok {
		return name
	}
	return "Unknown"
}

// GetBuffName returns the Buff Chinese display name.
func (r *DefinitionRegistry) GetBuffName(bt BuffType) string {
	if name, ok := r.buffNames[bt]; ok {
		return name
	}
	return "未知"
}

// GetBuffEvaluation returns the Buff evaluation score.
func (r *DefinitionRegistry) GetBuffEvaluation(bt BuffType) Evaluation {
	if eval, ok := r.buffEvals[bt]; ok {
		return eval
	}
	return EvaluationNeutral
}

// GetBuffHandler returns the Buff's custom handler (nil if none).
func (r *DefinitionRegistry) GetBuffHandler(bt BuffType) EventHandler {
	if handler, ok := r.buffHandlers[bt]; ok {
		return handler
	}
	return nil
}

// HasBuffHandler checks if Buff has a custom handler.
func (r *DefinitionRegistry) HasBuffHandler(bt BuffType) bool {
	_, ok := r.buffHandlers[bt]
	return ok
}

// GetAllBuffTypes returns all registered Buff types.
func (r *DefinitionRegistry) GetAllBuffTypes() []BuffType {
	types := make([]BuffType, 0, len(r.buffDefs))
	for bt := range r.buffDefs {
		types = append(types, bt)
	}
	return types
}

// GetBuffTypesByCategory returns Buff types by category.
func (r *DefinitionRegistry) GetBuffTypesByCategory(category string) []BuffType {
	switch category {
	case "Good":
		return r.goodBuffs
	case "Bad":
		return r.badBuffs
	case "Neutral":
		return r.neutralBuffs
	}
	return r.GetAllBuffTypes()
}

// GetAllBuffDefinitions returns all Buff definitions.
func (r *DefinitionRegistry) GetAllBuffDefinitions() []*BuffDefinition {
	defs := make([]*BuffDefinition, 0, len(r.buffDefs))
	for _, def := range r.buffDefs {
		defs = append(defs, def)
	}
	return defs
}

// ========== Event Registration ==========

// RegisterEvent registers an Event definition.
// Automatically generates String, Evaluation mappings and category classification.
func (r *DefinitionRegistry) RegisterEvent(def *EventDefinition) {
	if def == nil || def.Type == EventTypeNone {
		return
	}

	// Store definition
	r.eventDefs[def.Type] = def

	// Auto-generate mappings
	r.eventStrings[def.Type] = def.EnglishName
	r.eventNames[def.Type] = def.Name
	r.eventEvals[def.Type] = def.Eval

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
func (r *DefinitionRegistry) GetEventDefinition(et EventType) *EventDefinition {
	if def, ok := r.eventDefs[et]; ok {
		return def
	}
	return nil
}

// GetEventString returns the Event English identifier.
func (r *DefinitionRegistry) GetEventString(et EventType) string {
	if name, ok := r.eventStrings[et]; ok {
		return name
	}
	return "Unknown"
}

// GetEventName returns the Event Chinese display name.
func (r *DefinitionRegistry) GetEventName(et EventType) string {
	if name, ok := r.eventNames[et]; ok {
		return name
	}
	return "未知"
}

// GetEventEvaluation returns the Event evaluation score.
func (r *DefinitionRegistry) GetEventEvaluation(et EventType) Evaluation {
	if eval, ok := r.eventEvals[et]; ok {
		return eval
	}
	return EvaluationNeutral
}

// GetAllEventTypes returns all registered Event types.
func (r *DefinitionRegistry) GetAllEventTypes() []EventType {
	types := make([]EventType, 0, len(r.eventDefs))
	for et := range r.eventDefs {
		types = append(types, et)
	}
	return types
}

// GetEventTypesByCategory returns Event types by category.
func (r *DefinitionRegistry) GetEventTypesByCategory(category string) []EventType {
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
func (r *DefinitionRegistry) GetAllEventDefinitions() []*EventDefinition {
	defs := make([]*EventDefinition, 0, len(r.eventDefs))
	for _, def := range r.eventDefs {
		defs = append(defs, def)
	}
	return defs
}

// ========== Item Registration ==========

// RegisterItem registers an Item definition.
// Automatically generates String, Evaluation mappings and category classification.
func (r *DefinitionRegistry) RegisterItem(def *ItemDefinition) {
	if def == nil || def.Type == ItemTypeNone {
		return
	}

	// Store definition
	r.itemDefs[def.Type] = def

	// Auto-generate mappings
	r.itemStrings[def.Type] = def.EnglishName
	r.itemNames[def.Type] = def.Name
	r.itemEvals[def.Type] = def.Eval

	// Auto-classify by Evaluation
	if def.Eval.IsGood() {
		r.goodItems = append(r.goodItems, def.Type)
	} else if def.Eval.IsBad() {
		r.badItems = append(r.badItems, def.Type)
	} else {
		r.neutralItems = append(r.neutralItems, def.Type)
	}
}

// GetItemDefinition returns the Item definition by type.
func (r *DefinitionRegistry) GetItemDefinition(it ItemType) *ItemDefinition {
	if def, ok := r.itemDefs[it]; ok {
		return def
	}
	return nil
}

// GetItemString returns the Item English identifier.
func (r *DefinitionRegistry) GetItemString(it ItemType) string {
	if name, ok := r.itemStrings[it]; ok {
		return name
	}
	return "Unknown"
}

// GetItemName returns the Item Chinese display name.
func (r *DefinitionRegistry) GetItemName(it ItemType) string {
	if name, ok := r.itemNames[it]; ok {
		return name
	}
	return "未知"
}

// GetItemEvaluation returns the Item evaluation score.
func (r *DefinitionRegistry) GetItemEvaluation(it ItemType) Evaluation {
	if eval, ok := r.itemEvals[it]; ok {
		return eval
	}
	return EvaluationNeutral
}

// GetAllItemTypes returns all registered Item types.
func (r *DefinitionRegistry) GetAllItemTypes() []ItemType {
	types := make([]ItemType, 0, len(r.itemDefs))
	for it := range r.itemDefs {
		types = append(types, it)
	}
	return types
}

// GetItemTypesByCategory returns Item types by category.
func (r *DefinitionRegistry) GetItemTypesByCategory(category string) []ItemType {
	switch category {
	case "Good":
		return r.goodItems
	case "Bad":
		return r.badItems
	case "Neutral":
		return r.neutralItems
	}
	return r.GetAllItemTypes()
}

// GetAllItemDefinitions returns all Item definitions.
func (r *DefinitionRegistry) GetAllItemDefinitions() []*ItemDefinition {
	defs := make([]*ItemDefinition, 0, len(r.itemDefs))
	for _, def := range r.itemDefs {
		defs = append(defs, def)
	}
	return defs
}

// ========== Evaluation Range Queries ==========

// GetBuffTypesByEvaluationRange returns Buffs within the specified Evaluation range.
func (r *DefinitionRegistry) GetBuffTypesByEvaluationRange(minEval, maxEval Evaluation) []BuffType {
	var result []BuffType
	for bt, eval := range r.buffEvals {
		if eval >= minEval && eval <= maxEval {
			result = append(result, bt)
		}
	}
	return result
}

// GetEventTypesByEvaluationRange returns Events within the specified Evaluation range.
func (r *DefinitionRegistry) GetEventTypesByEvaluationRange(minEval, maxEval Evaluation) []EventType {
	var result []EventType
	for et, eval := range r.eventEvals {
		if eval >= minEval && eval <= maxEval {
			result = append(result, et)
		}
	}
	return result
}

// GetItemTypesByEvaluationRange returns Items within the specified Evaluation range.
func (r *DefinitionRegistry) GetItemTypesByEvaluationRange(minEval, maxEval Evaluation) []ItemType {
	var result []ItemType
	for it, eval := range r.itemEvals {
		if eval >= minEval && eval <= maxEval {
			result = append(result, it)
		}
	}
	return result
}