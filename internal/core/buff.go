package core

import (
	"fmt"
	"time"

	"github.com/b1tAction/Fated/pkg/event"
)

// ========== Buff 类型定义 ==========

type BuffType int

const (
	BuffTypeNone BuffType = iota
	// 负性 Buff
	BuffTypeCurse    // 诅咒
	BuffTypeLost     // 迷途
	BuffTypeCorrupt  // 腐化
	BuffTypePoison   // 毒瘴
	// 中性 Buff
	BuffTypeHidden   // 隐匿
	// 正性 Buff
	BuffTypeDivine   // 神眷
	BuffTypeRain     // 甘霖
	BuffTypeExorcism // 辟邪
	BuffTypeFire     // 离火
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

// IsPositive 判断是否为正面 Buff
func (bt BuffType) IsPositive() bool {
	return bt == BuffTypeDivine || bt == BuffTypeHidden ||
		bt == BuffTypeRain || bt == BuffTypeExorcism || bt == BuffTypeFire
}

// GetEvaluation 获取 Buff 的评分
func (bt BuffType) GetEvaluation() Evaluation {
	evalMap := map[BuffType]Evaluation{
		BuffTypeCurse:    EvaluationBad,      // 诅咒：较恶
		BuffTypeLost:     EvaluationMildBad,  // 迷途：轻恶
		BuffTypeCorrupt:  EvaluationBad,      // 腐化：较恶
		BuffTypePoison:   EvaluationVeryBad,  // 毒瘴：极恶
		BuffTypeDivine:   EvaluationVeryGood, // 神眷：极良
		BuffTypeHidden:   EvaluationExcellent, // 隐匿：最佳（免疫）
		BuffTypeRain:     EvaluationGood,     // 甘霖：较良
		BuffTypeExorcism: EvaluationMildGood, // 辟邪：轻良
		BuffTypeFire:     EvaluationGood,     // 离火：较良
	}
	if eval, ok := evalMap[bt]; ok {
		return eval
	}
	return EvaluationNeutral
}

// ========== Buff 实例 ==========

type Buff struct {
	Type           BuffType `json:"type"`
	ID             string   `json:"id"`              // Buff实例ID
	Duration       int      `json:"duration"`
	Charge         int      `json:"charge"`
	SubscriptionID string   `json:"subscription_id"` // EventBus订阅ID（由 engine 包管理）
}

func NewBuff(buffType BuffType, duration int) *Buff {
	return &Buff{
		Type:     buffType,
		ID:       fmt.Sprintf("buff-%d", time.Now().UnixNano()),
		Duration: duration,
		Charge:   0,
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

// ========== Buff 静态定义 ==========

type BuffDefinition struct {
	Type        BuffType    `json:"type"`
	Eval        Evaluation  `json:"evaluation"`    // 评分
	Name        string      `json:"name"`
	Desc        string      `json:"desc"`
	Duration    int         `json:"duration"`
	HPPerTurn   int         `json:"hp_per_turn"`
	LPPerTurn   int         `json:"lp_per_turn"`
	Special     string      `json:"special"`
	Phase       event.Phase `json:"phase"`         // 触发时机
	Priority    int         `json:"priority"`      // 执行优先级
	NeedConfirm bool        `json:"need_confirm"`  // 是否需要用户确认（默认false）
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
			Phase:     event.PhaseBeforeTurn,
			Priority:  50,
		},
		BuffTypeDivine: {
			Type:      BuffTypeDivine,
			Eval:      eval,
			Name:      "神眷",
			Desc:      "接下来3回合LP+1",
			Duration:  3,
			LPPerTurn: 1,
			Phase:     event.PhaseBeforeTurn,
			Priority:  50,
		},
		BuffTypeHidden: {
			Type:      BuffTypeHidden,
			Eval:      eval,
			Name:      "隐匿",
			Desc:      "接下来3回合免疫任意事件、BUFF或道具的影响",
			Duration:  3,
			Special:   "immune",
			Phase:     event.PhasePreDamage,
			Priority:  100,
		},
		BuffTypeLost: {
			Type:      BuffTypeLost,
			Eval:      eval,
			Name:      "迷途",
			Desc:      "下1回合朝反方向移动",
			Duration:  1,
			Special:   "reverse",
			Phase:     event.PhaseOnMove,
			Priority:  100,
		},
		BuffTypeCorrupt: {
			Type:      BuffTypeCorrupt,
			Eval:      eval,
			Name:      "腐化",
			Desc:      "接下来4回合每2回合HP-1",
			Duration:  4,
			HPPerTurn: -1,
			Phase:     event.PhaseAfterTurn,
			Priority:  50,
		},
		BuffTypeRain: {
			Type:      BuffTypeRain,
			Eval:      eval,
			Name:      "甘霖",
			Desc:      "接下来4回合每2回合HP+1",
			Duration:  4,
			HPPerTurn: 1,
			Phase:     event.PhaseAfterTurn,
			Priority:  50,
		},
		BuffTypeExorcism: {
			Type:      BuffTypeExorcism,
			Eval:      eval,
			Name:      "辟邪",
			Desc:      "接下来5回合无视毒瘴buff",
			Duration:  5,
			Special:   "immune_poison",
			Phase:     event.PhasePreEvent,
			Priority:  80,
		},
		BuffTypePoison: {
			Type:      BuffTypePoison,
			Eval:      eval,
			Name:      "毒瘴",
			Desc:      "接下来3回合每回合受一次恶性随机事件影响",
			Duration:  3,
			Special:   "bad_event_per_turn",
			Phase:     event.PhaseBeforeTurn,
			Priority:  30,
		},
		BuffTypeFire: {
			Type:      BuffTypeFire,
			Eval:      eval,
			Name:      "离火",
			Desc:      "朱雀阵营增益，每4回合LP+1",
			Duration:  -1,
			Special:   "zhuque_passive",
			Phase:     event.PhaseBeforeTurn,
			Priority:  10,
		},
	}
	if def, ok := definitions[bt]; ok {
		return def
	}
	return nil
}

// ========== Buff 注册表 ==========

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

// GetBuffsByEvaluationRange 按 Evaluation 范围获取 Buff 列表
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

// GetBuffsByCategory 按类别获取 Buff 列表
func (br *BuffRegistry) GetBuffsByCategory(category string) []BuffType {
	switch category {
	case "Good":
		return br.GoodBuffs
	case "Bad":
		return br.BadBuffs
	}
	return br.AllBuffs
}

// GetAllBuffDefinitions 获取所有 Buff 定义
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