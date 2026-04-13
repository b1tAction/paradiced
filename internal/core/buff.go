package core

import (
	"fmt"
	"time"

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

// String returns the Buff type name from GlobalRegistry.
func (bt BuffType) String() string {
	return GlobalRegistry.GetBuffString(bt)
}

// ========== Buff Instance ==========

type Buff struct {
	Type            BuffType `json:"type"`
	ID              string   `json:"id"`               // Buff instance ID
	Duration        int      `json:"duration"`
	Charge          int      `json:"charge"`
	SubscriptionIDs []string `json:"subscription_ids"` // EventBus subscription IDs (managed by engine package, supports multi-phase subscriptions)
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
	Type          BuffType      `json:"type"`
	Eval          Evaluation    `json:"evaluation"`     // Evaluation score
	EnglishName   string        `json:"english_name"`   // English identifier (for String())
	Name          string        `json:"name"`           // Chinese display name
	Desc          string        `json:"desc"`
	Duration      int           `json:"duration"`
	HPPerTurn     int           `json:"hp_per_turn"`
	LPPerTurn     int           `json:"lp_per_turn"`
	SpecialEffect SpecialEffect `json:"special_effect"` // Special effect type (enum, not string)
	Phases        []event.Phase `json:"phases"`         // Trigger phases list (supports multi-phase)
	Priority      int           `json:"priority"`       // Execution priority
	NeedConfirm   bool          `json:"need_confirm"`   // Whether user confirmation is needed (default false)
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

// ========== Global Registry Access Functions ==========

// GetBuffDefinition returns the Buff definition from GlobalRegistry.
func GetBuffDefinition(bt BuffType) *BuffDefinition {
	return GlobalRegistry.GetBuffDefinition(bt)
}

// GetBuffString returns the Buff name string from GlobalRegistry.
func GetBuffString(bt BuffType) string {
	return GlobalRegistry.GetBuffString(bt)
}

// GetBuffEvaluation returns the Buff evaluation score from GlobalRegistry.
func GetBuffEvaluation(bt BuffType) Evaluation {
	return GlobalRegistry.GetBuffEvaluation(bt)
}

// GetBuffHandler returns the Buff's custom handler from GlobalRegistry.
func GetBuffHandler(bt BuffType) EventHandler {
	return GlobalRegistry.GetBuffHandler(bt)
}

// HasBuffHandler checks if Buff has a custom handler.
func HasBuffHandler(bt BuffType) bool {
	return GlobalRegistry.HasBuffHandler(bt)
}

// GetAllBuffTypes returns all registered Buff types.
func GetAllBuffTypes() []BuffType {
	return GlobalRegistry.GetAllBuffTypes()
}

// GetBuffTypesByCategory returns Buff types by category.
func GetBuffTypesByCategory(category string) []BuffType {
	return GlobalRegistry.GetBuffTypesByCategory(category)
}

// GetAllBuffDefinitions returns all Buff definitions.
func GetAllBuffDefinitions() []*BuffDefinition {
	return GlobalRegistry.GetAllBuffDefinitions()
}

// IsPositive checks if the Buff is positive (based on effect, not evaluation).
// Hidden is neutral evaluation but positive effect (immunity).
func (bt BuffType) IsPositive() bool {
	// Positive buffs: Divine, Hidden, Rain, Exorcism, Fire
	// Hidden is neutral eval but positive effect
	positiveBuffs := []BuffType{BuffTypeDivine, BuffTypeHidden, BuffTypeRain, BuffTypeExorcism, BuffTypeFire}
	for _, pos := range positiveBuffs {
		if bt == pos {
			return true
		}
	}
	return false
}