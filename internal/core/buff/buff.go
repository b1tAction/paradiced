// Package buff provides Buff related data structures and registry.
// This package is independently usable via Direct Import.
package buff

import (
	"github.com/b1tAction/paradiced/pkg/constants"
	"github.com/b1tAction/paradiced/pkg/handler"
	"github.com/b1tAction/paradiced/pkg/id"
)

// ========== Buff Instance ==========

type Buff struct {
	Type            constants.BuffType `json:"type"`
	ID              id.BuffID          `json:"id"` // Buff instance ID (UUID v7)
	Duration        int                `json:"duration"`
	Charge          int                `json:"charge"`
	SubscriptionIDs []string           `json:"subscription_ids"` // EventBus subscription IDs (UUID strings)
}

func NewBuff(buffType constants.BuffType, duration int) *Buff {
	return &Buff{
		Type:            buffType,
		ID:              id.NewBuffID(),
		Duration:        duration,
		Charge:          0,
		SubscriptionIDs: make([]string, 0),
	}
}

func (b *Buff) IsActive() bool {
	return b.Duration > 0 || b.Duration == -1 || b.Charge > 0
}

func (b *Buff) TickDuration() bool {
	if b.Duration > 0 {
		b.Duration--
	}
	return b.IsActive()
}

// ========== Buff Definition ==========

type BuffDefinition struct {
	Type          constants.BuffType      `json:"type"`
	Eval          constants.Evaluation    `json:"evaluation"`   // Evaluation score
	EnglishName   string                  `json:"english_name"` // English identifier (snake_case)
	Name          string                  `json:"name"`         // Chinese display name
	Desc          string                  `json:"desc"`
	Duration      int                     `json:"duration"`
	HPPerTurn     int                     `json:"hp_per_turn"`
	LPPerTurn     int                     `json:"lp_per_turn"`
	SpecialEffect constants.SpecialEffect `json:"special_effect"` // Special effect type
	Phases        []constants.Phase       `json:"phases"`         // Trigger phases list
	Priority      int                     `json:"priority"`       // Execution priority
	NeedConfirm   bool                    `json:"need_confirm"`   // Whether user confirmation needed
}

// GetPhases returns the Buff's trigger phase list.
func (def *BuffDefinition) GetPhases() []constants.Phase {
	return def.Phases
}

// HasPhase checks if the Buff triggers at the specified Phase.
func (def *BuffDefinition) HasPhase(phase constants.Phase) bool {
	for _, p := range def.Phases {
		if p == phase {
			return true
		}
	}
	return false
}

// ========== Buff Registry ==========

// EffectHandler alias for handler.EffectHandler (unified signature).
type EffectHandler = handler.EffectHandler

// BuffRegistry is the registry for Buff definitions.
type BuffRegistry struct {
	defs     map[constants.BuffType]*BuffDefinition
	handlers map[constants.BuffType]EffectHandler
	names    map[constants.BuffType]string // Chinese name

	// Category lists (auto-generated)
	goodBuffs    []constants.BuffType
	badBuffs     []constants.BuffType
	neutralBuffs []constants.BuffType
}

// NewBuffRegistry creates a new Buff registry.
func NewBuffRegistry() *BuffRegistry {
	return &BuffRegistry{
		defs:         make(map[constants.BuffType]*BuffDefinition),
		handlers:     make(map[constants.BuffType]EffectHandler),
		names:        make(map[constants.BuffType]string),
		goodBuffs:    make([]constants.BuffType, 0),
		badBuffs:     make([]constants.BuffType, 0),
		neutralBuffs: make([]constants.BuffType, 0),
	}
}

// RegisterBuff registers a Buff definition with optional handler.
func (r *BuffRegistry) RegisterBuff(def *BuffDefinition, handler EffectHandler) {
	if def == nil || def.Type == constants.BuffTypeNone {
		return
	}

	r.defs[def.Type] = def
	r.names[def.Type] = def.Name

	// Auto-classify by Evaluation
	if def.Eval.IsGood() {
		r.goodBuffs = append(r.goodBuffs, def.Type)
	} else if def.Eval.IsBad() {
		r.badBuffs = append(r.badBuffs, def.Type)
	} else {
		r.neutralBuffs = append(r.neutralBuffs, def.Type)
	}

	if handler != nil {
		r.handlers[def.Type] = handler
	}
}

// GetBuffDefinition returns the Buff definition by type.
func (r *BuffRegistry) GetBuffDefinition(bt constants.BuffType) *BuffDefinition {
	if def, ok := r.defs[bt]; ok {
		return def
	}
	return nil
}

// GetBuffString returns the Buff type as string (snake_case).
func (r *BuffRegistry) GetBuffString(bt constants.BuffType) string {
	return string(bt) // BuffType is already string with snake_case
}

// GetBuffName returns the Buff Chinese display name.
func (r *BuffRegistry) GetBuffName(bt constants.BuffType) string {
	if name, ok := r.names[bt]; ok {
		return name
	}
	return "未知"
}

// GetBuffEvaluation returns the Buff evaluation score.
func (r *BuffRegistry) GetBuffEvaluation(bt constants.BuffType) constants.Evaluation {
	if def, ok := r.defs[bt]; ok {
		return def.Eval
	}
	return constants.EvaluationNeutral
}

// GetBuffHandler returns the Buff's custom handler (nil if none).
func (r *BuffRegistry) GetBuffHandler(bt constants.BuffType) EffectHandler {
	if handler, ok := r.handlers[bt]; ok {
		return handler
	}
	return nil
}

// HasBuffHandler checks if Buff has a custom handler.
func (r *BuffRegistry) HasBuffHandler(bt constants.BuffType) bool {
	_, ok := r.handlers[bt]
	return ok
}

// GetAllBuffTypes returns all registered Buff types.
func (r *BuffRegistry) GetAllBuffTypes() []constants.BuffType {
	result := make([]constants.BuffType, 0, len(r.defs))
	for bt := range r.defs {
		result = append(result, bt)
	}
	return result
}

// GetBuffTypesByCategory returns Buff types by category.
func (r *BuffRegistry) GetBuffTypesByCategory(category string) []constants.BuffType {
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
func (r *BuffRegistry) GetAllBuffDefinitions() []*BuffDefinition {
	defs := make([]*BuffDefinition, 0, len(r.defs))
	for _, def := range r.defs {
		defs = append(defs, def)
	}
	return defs
}

// GetBuffTypesByEvaluationRange returns Buffs within the specified Evaluation range.
func (r *BuffRegistry) GetBuffTypesByEvaluationRange(minEval, maxEval constants.Evaluation) []constants.BuffType {
	var result []constants.BuffType
	for bt, def := range r.defs {
		if def.Eval >= minEval && def.Eval <= maxEval {
			result = append(result, bt)
		}
	}
	return result
}

// ========== Global Registry Access Functions ==========

// GetBuffDefinition returns the Buff definition from GlobalBuffRegistry.
func GetBuffDefinition(bt constants.BuffType) *BuffDefinition {
	return GlobalBuffRegistry.GetBuffDefinition(bt)
}

// GetBuffString returns the Buff name string from GlobalBuffRegistry.
func GetBuffString(bt constants.BuffType) string {
	return GlobalBuffRegistry.GetBuffString(bt)
}

// GetBuffEvaluation returns the Buff evaluation score from GlobalBuffRegistry.
func GetBuffEvaluation(bt constants.BuffType) constants.Evaluation {
	return GlobalBuffRegistry.GetBuffEvaluation(bt)
}

// GetBuffHandler returns the Buff's custom handler from GlobalBuffRegistry.
func GetBuffHandler(bt constants.BuffType) EffectHandler {
	return GlobalBuffRegistry.GetBuffHandler(bt)
}

// HasBuffHandler checks if Buff has a custom handler.
func HasBuffHandler(bt constants.BuffType) bool {
	return GlobalBuffRegistry.HasBuffHandler(bt)
}

// GetAllBuffTypes returns all registered Buff types.
func GetAllBuffTypes() []constants.BuffType {
	return GlobalBuffRegistry.GetAllBuffTypes()
}

// GetBuffTypesByCategory returns Buff types by category.
func GetBuffTypesByCategory(category string) []constants.BuffType {
	return GlobalBuffRegistry.GetBuffTypesByCategory(category)
}

// GetAllBuffDefinitions returns all Buff definitions.
func GetAllBuffDefinitions() []*BuffDefinition {
	return GlobalBuffRegistry.GetAllBuffDefinitions()
}

// GetBuffName returns the Buff Chinese display name from GlobalBuffRegistry.
func GetBuffName(bt constants.BuffType) string {
	return GlobalBuffRegistry.GetBuffName(bt)
}
