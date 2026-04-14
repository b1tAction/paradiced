// Package buff provides Buff related data structures and registry.
// This package is independently usable via Direct Import.
package buff

import (
	"fmt"
	"time"

	"github.com/b1tAction/Fated/internal/core/types"
	"github.com/b1tAction/Fated/pkg/event"
)

// ========== Buff Type Definitions ==========

type BuffType int

const (
	BuffTypeNone BuffType = iota
	// Negative Buffs
	BuffTypeCurse    // Curse诅咒
	BuffTypeLost     // Lost迷途
	BuffTypeCorrupt  // Corrupt腐化
	BuffTypePoison   // Poison毒瘴
	// Neutral Buff
	BuffTypeHidden   // Hidden隐匿
	// Positive Buffs
	BuffTypeDivine   // Divine神眷
	BuffTypeRain     // Rain甘霖
	BuffTypeExorcism // Exorcism辟邪
	BuffTypeFire     // Fire离火
)

// IsValid checks if the Buff type is valid.
func (bt BuffType) IsValid() bool {
	return bt > BuffTypeNone && bt <= BuffTypeFire
}

// String returns the Buff type name from GlobalBuffRegistry.
func (bt BuffType) String() string {
	return GlobalBuffRegistry.GetBuffString(bt)
}

// IsPositive checks if the Buff is positive (based on effect, not evaluation).
// Hidden is neutral evaluation but positive effect (immunity).
func (bt BuffType) IsPositive() bool {
	// Positive buffs: Divine, Hidden, Rain, Exorcism, Fire
	positiveBuffs := []BuffType{BuffTypeDivine, BuffTypeHidden, BuffTypeRain, BuffTypeExorcism, BuffTypeFire}
	for _, pos := range positiveBuffs {
		if bt == pos {
			return true
		}
	}
	return false
}

// ========== Buff Instance ==========

type Buff struct {
	Type            BuffType `json:"type"`
	ID              string   `json:"id"`               // Buff instance ID
	Duration        int      `json:"duration"`
	Charge          int      `json:"charge"`
	SubscriptionIDs []string `json:"subscription_ids"` // EventBus subscription IDs (managed by engine package)
}

func NewBuff(buffType BuffType, duration int) *Buff {
	return &Buff{
		Type:            buffType,
		ID:              fmt.Sprintf("buff-%d", time.Now().UnixNano()),
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
	Type          BuffType             `json:"type"`
	Eval          types.Evaluation     `json:"evaluation"`     // Evaluation score
	EnglishName   string               `json:"english_name"`   // English identifier (for String())
	Name          string               `json:"name"`           // Chinese display name
	Desc          string               `json:"desc"`
	Duration      int                  `json:"duration"`
	HPPerTurn     int                  `json:"hp_per_turn"`
	LPPerTurn     int                  `json:"lp_per_turn"`
	SpecialEffect types.SpecialEffect  `json:"special_effect"` // Special effect type
	Phases        []event.Phase        `json:"phases"`         // Trigger phases list
	Priority      int                  `json:"priority"`       // Execution priority
	NeedConfirm   bool                 `json:"need_confirm"`   // Whether user confirmation needed
}

// GetPhases returns the Buff's trigger phase list.
func (def *BuffDefinition) GetPhases() []event.Phase {
	return def.Phases
}

// HasPhase checks if the Buff triggers at the specified Phase.
func (def *BuffDefinition) HasPhase(phase event.Phase) bool {
	for _, p := range def.Phases {
		if p == phase {
			return true
		}
	}
	return false
}

// ========== Buff Registry ==========

// EffectHandler is a handler function for Buff/Item/Event/Faction effects.
// Handlers should use ctx.AddDerivedAction() to generate new actions.
// Unified signature for all effect sources.
type EffectHandler func(phase event.Phase, ctx *event.Context)

// BuffRegistry is the registry for Buff definitions.
type BuffRegistry struct {
	defs     map[BuffType]*BuffDefinition
	handlers map[BuffType]EffectHandler
	strings  map[BuffType]string // English identifier
	names    map[BuffType]string // Chinese name
	evals    map[BuffType]types.Evaluation

	// Category lists (auto-generated)
	goodBuffs    []BuffType
	badBuffs     []BuffType
	neutralBuffs []BuffType
}

// NewBuffRegistry creates a new Buff registry.
func NewBuffRegistry() *BuffRegistry {
	return &BuffRegistry{
		defs:         make(map[BuffType]*BuffDefinition),
		handlers:     make(map[BuffType]EffectHandler),
		strings:      make(map[BuffType]string),
		names:        make(map[BuffType]string),
		evals:        make(map[BuffType]types.Evaluation),
		goodBuffs:    make([]BuffType, 0),
		badBuffs:     make([]BuffType, 0),
		neutralBuffs: make([]BuffType, 0),
	}
}

// RegisterBuff registers a Buff definition with optional handler.
func (r *BuffRegistry) RegisterBuff(def *BuffDefinition, handler EffectHandler) {
	if def == nil || def.Type == BuffTypeNone {
		return
	}

	r.defs[def.Type] = def
	r.strings[def.Type] = def.EnglishName
	r.names[def.Type] = def.Name
	r.evals[def.Type] = def.Eval

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
func (r *BuffRegistry) GetBuffDefinition(bt BuffType) *BuffDefinition {
	if def, ok := r.defs[bt]; ok {
		return def
	}
	return nil
}

// GetBuffString returns the Buff English identifier.
func (r *BuffRegistry) GetBuffString(bt BuffType) string {
	if name, ok := r.strings[bt]; ok {
		return name
	}
	return "Unknown"
}

// GetBuffName returns the Buff Chinese display name.
func (r *BuffRegistry) GetBuffName(bt BuffType) string {
	if name, ok := r.names[bt]; ok {
		return name
	}
	return "未知"
}

// GetBuffEvaluation returns the Buff evaluation score.
func (r *BuffRegistry) GetBuffEvaluation(bt BuffType) types.Evaluation {
	if eval, ok := r.evals[bt]; ok {
		return eval
	}
	return types.EvaluationNeutral
}

// GetBuffHandler returns the Buff's custom handler (nil if none).
func (r *BuffRegistry) GetBuffHandler(bt BuffType) EffectHandler {
	if handler, ok := r.handlers[bt]; ok {
		return handler
	}
	return nil
}

// HasBuffHandler checks if Buff has a custom handler.
func (r *BuffRegistry) HasBuffHandler(bt BuffType) bool {
	_, ok := r.handlers[bt]
	return ok
}

// GetAllBuffTypes returns all registered Buff types.
func (r *BuffRegistry) GetAllBuffTypes() []BuffType {
	result := make([]BuffType, 0, len(r.defs))
	for bt := range r.defs {
		result = append(result, bt)
	}
	return result
}

// GetBuffTypesByCategory returns Buff types by category.
func (r *BuffRegistry) GetBuffTypesByCategory(category string) []BuffType {
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
func (r *BuffRegistry) GetBuffTypesByEvaluationRange(minEval, maxEval types.Evaluation) []BuffType {
	var result []BuffType
	for bt, eval := range r.evals {
		if eval >= minEval && eval <= maxEval {
			result = append(result, bt)
		}
	}
	return result
}

// ========== Global Registry Access Functions ==========

// GetBuffDefinition returns the Buff definition from GlobalBuffRegistry.
func GetBuffDefinition(bt BuffType) *BuffDefinition {
	return GlobalBuffRegistry.GetBuffDefinition(bt)
}

// GetBuffString returns the Buff name string from GlobalBuffRegistry.
func GetBuffString(bt BuffType) string {
	return GlobalBuffRegistry.GetBuffString(bt)
}

// GetBuffEvaluation returns the Buff evaluation score from GlobalBuffRegistry.
func GetBuffEvaluation(bt BuffType) types.Evaluation {
	return GlobalBuffRegistry.GetBuffEvaluation(bt)
}

// GetBuffHandler returns the Buff's custom handler from GlobalBuffRegistry.
func GetBuffHandler(bt BuffType) EffectHandler {
	return GlobalBuffRegistry.GetBuffHandler(bt)
}

// HasBuffHandler checks if Buff has a custom handler.
func HasBuffHandler(bt BuffType) bool {
	return GlobalBuffRegistry.HasBuffHandler(bt)
}

// GetAllBuffTypes returns all registered Buff types.
func GetAllBuffTypes() []BuffType {
	return GlobalBuffRegistry.GetAllBuffTypes()
}

// GetBuffTypesByCategory returns Buff types by category.
func GetBuffTypesByCategory(category string) []BuffType {
	return GlobalBuffRegistry.GetBuffTypesByCategory(category)
}

// GetAllBuffDefinitions returns all Buff definitions.
func GetAllBuffDefinitions() []*BuffDefinition {
	return GlobalBuffRegistry.GetAllBuffDefinitions()
}