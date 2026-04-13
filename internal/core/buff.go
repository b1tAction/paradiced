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

func (bt BuffType) String() string {
	names := map[BuffType]string{
		BuffTypeNone:     "None",
		BuffTypeCurse:    "Curse",
		BuffTypeLost:     "Lost",
		BuffTypeCorrupt:  "Corrupt",
		BuffTypePoison:   "Poison",
		BuffTypeDivine:   "Divine",
		BuffTypeHidden:   "Hidden",
		BuffTypeRain:     "Rain",
		BuffTypeExorcism: "Exorcism",
		BuffTypeFire:     "Fire",
	}
	if name, ok := names[bt]; ok {
		return name
	}
	return "Unknown"
}

// IsPositive checks if the Buff is positive.
func (bt BuffType) IsPositive() bool {
	return bt == BuffTypeDivine || bt == BuffTypeHidden ||
		bt == BuffTypeRain || bt == BuffTypeExorcism || bt == BuffTypeFire
}

// GetEvaluation returns the Buff's evaluation score.
func (bt BuffType) GetEvaluation() Evaluation {
	evalMap := map[BuffType]Evaluation{
		BuffTypeCurse:    EvaluationBad,      // Curse: bad
		BuffTypeLost:     EvaluationMildBad,  // Lost: mild bad
		BuffTypeCorrupt:  EvaluationBad,      // Corrupt: bad
		BuffTypePoison:   EvaluationVeryBad,  // Poison: very bad
		BuffTypeDivine:   EvaluationVeryGood, // Divine: very good
		BuffTypeHidden:   EvaluationNeutral,  // Hidden: neutral
		BuffTypeRain:     EvaluationGood,     // Rain: good
		BuffTypeExorcism: EvaluationMildGood, // Exorcism: mild good
		BuffTypeFire:     EvaluationGood,     // Fire: good
	}
	if eval, ok := evalMap[bt]; ok {
		return eval
	}
	return EvaluationNeutral
}

// ========== Buff Instance ==========

type Buff struct {
	Type            BuffType  `json:"type"`
	ID              string    `json:"id"`               // Buff instance ID
	Duration        int       `json:"duration"`
	Charge          int       `json:"charge"`
	SubscriptionIDs []string  `json:"subscription_ids"` // EventBus subscription IDs (managed by engine package, supports multi-phase subscriptions)
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
	Type        BuffType      `json:"type"`
	Eval        Evaluation    `json:"evaluation"`    // Evaluation score
	Name        string        `json:"name"`
	Desc        string        `json:"desc"`
	Duration    int           `json:"duration"`
	HPPerTurn   int           `json:"hp_per_turn"`
	LPPerTurn   int           `json:"lp_per_turn"`
	Special     string        `json:"special"`
	Phases      []event.Phase `json:"phases"`        // Trigger phases list (supports multi-phase)
	Priority    int           `json:"priority"`      // Execution priority
	NeedConfirm bool          `json:"need_confirm"`  // Whether user confirmation is needed (default false)
}

// GetPhases returns the Buff's trigger phase list.
// Backward compatible: if Phases is empty, returns default Phase (won't happen, all Buffs have definitions).
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

func (bt BuffType) GetBuffDefinition() *BuffDefinition {
	eval := bt.GetEvaluation()
	definitions := map[BuffType]*BuffDefinition{
		BuffTypeCurse: {
			Type:      BuffTypeCurse,
			Eval:      eval,
			Name:      "诅咒",
			Desc:      "接下来3回合LP-1",
			Duration:  3,
			LPPerTurn: -1,
			Phases:    []event.Phase{event.PhaseBeforeTurn},
			Priority:  50,
		},
		BuffTypeDivine: {
			Type:      BuffTypeDivine,
			Eval:      eval,
			Name:      "神眷",
			Desc:      "接下来3回合LP+1",
			Duration:  3,
			LPPerTurn: 1,
			Phases:    []event.Phase{event.PhaseBeforeTurn},
			Priority:  50,
		},
		BuffTypeHidden: {
			Type:      BuffTypeHidden,
			Eval:      eval,
			Name:      "隐匿",
			Desc:      "接下来3回合免疫任意事件、BUFF或道具的影响",
			Duration:  3,
			Special:   "immune",
			Phases:    []event.Phase{event.PhasePreDamage},
			Priority:  100,
		},
		BuffTypeLost: {
			Type:      BuffTypeLost,
			Eval:      eval,
			Name:      "迷途",
			Desc:      "下1回合朝反方向移动",
			Duration:  1,
			Special:   "reverse",
			Phases:    []event.Phase{event.PhaseOnMove},
			Priority:  100,
		},
		BuffTypeCorrupt: {
			Type:      BuffTypeCorrupt,
			Eval:      eval,
			Name:      "腐化",
			Desc:      "接下来4回合每2回合HP-1",
			Duration:  4,
			HPPerTurn: -1,
			Phases:    []event.Phase{event.PhaseAfterTurn},
			Priority:  50,
		},
		BuffTypeRain: {
			Type:      BuffTypeRain,
			Eval:      eval,
			Name:      "甘霖",
			Desc:      "接下来4回合每2回合HP+1",
			Duration:  4,
			HPPerTurn: 1,
			Phases:    []event.Phase{event.PhaseAfterTurn},
			Priority:  50,
		},
		BuffTypeExorcism: {
			Type:      BuffTypeExorcism,
			Eval:      eval,
			Name:      "辟邪",
			Desc:      "接下来5回合无视毒瘴buff",
			Duration:  5,
			Special:   "immune_poison",
			Phases:    []event.Phase{event.PhasePreEvent},
			Priority:  80,
		},
		BuffTypePoison: {
			Type:      BuffTypePoison,
			Eval:      eval,
			Name:      "毒瘴",
			Desc:      "接下来3回合每回合受一次恶性随机事件影响",
			Duration:  3,
			Special:   "bad_event_per_turn",
			Phases:    []event.Phase{event.PhaseBeforeTurn},
			Priority:  30,
		},
		BuffTypeFire: {
			Type:      BuffTypeFire,
			Eval:      eval,
			Name:      "离火",
			Desc:      "朱雀阵营增益，每4回合LP+1",
			Duration:  -1,
			Special:   "zhuque_passive",
			Phases:    []event.Phase{event.PhaseBeforeTurn},
			Priority:  10,
		},
	}
	if def, ok := definitions[bt]; ok {
		return def
	}
	return nil
}

// ========== Buff Registry ==========

type BuffRegistry struct {
	AllBuffs  []BuffType `json:"all_buffs"`
	GoodBuffs []BuffType `json:"good_buffs"`
	BadBuffs  []BuffType `json:"bad_buffs"`
}

func NewBuffRegistry() *BuffRegistry {
	return &BuffRegistry{
		AllBuffs: []BuffType{
			BuffTypeCurse, BuffTypeDivine, BuffTypeHidden, BuffTypeLost,
			BuffTypeCorrupt, BuffTypeRain, BuffTypeExorcism, BuffTypePoison, BuffTypeFire,
		},
		GoodBuffs: []BuffType{
			BuffTypeDivine, BuffTypeHidden, BuffTypeRain, BuffTypeExorcism, BuffTypeFire,
		},
		BadBuffs: []BuffType{
			BuffTypeCurse, BuffTypeLost, BuffTypeCorrupt, BuffTypePoison,
		},
	}
}

// GetBuffsByEvaluationRange returns Buffs within the specified Evaluation range.
func (br *BuffRegistry) GetBuffsByEvaluationRange(minEval, maxEval Evaluation) []BuffType {
	var result []BuffType
	for _, bt := range br.AllBuffs {
		eval := bt.GetEvaluation()
		if eval >= minEval && eval <= maxEval {
			result = append(result, bt)
		}
	}
	return result
}

// GetBuffsByCategory returns Buffs by category.
func (br *BuffRegistry) GetBuffsByCategory(category string) []BuffType {
	switch category {
	case "Good":
		return br.GoodBuffs
	case "Bad":
		return br.BadBuffs
	}
	return br.AllBuffs
}

// GetAllBuffDefinitions returns all Buff definitions.
func (br *BuffRegistry) GetAllBuffDefinitions() []*BuffDefinition {
	defs := make([]*BuffDefinition, 0, len(br.AllBuffs))
	for _, bt := range br.AllBuffs {
		def := bt.GetBuffDefinition()
		if def != nil {
			defs = append(defs, def)
		}
	}
	return defs
}