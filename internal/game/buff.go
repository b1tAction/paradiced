package game

import (
	"fmt"
	"time"

	"github.com/b1tAction/Fated/pkg/event"
)

// ========== Evaluation 属性评分系统 ==========

// Evaluation 属性评分（0~100）
// 越低越坏，越高越好
// 0~40: 恶性（Bad）
// 41~65: 中性（Neutral）
// 66~100: 良性（Good）
type Evaluation int

const (
	// Evaluation 范围常量
	EvaluationMin    Evaluation = 0   // 最低评分
	EvaluationMax    Evaluation = 100 // 最高评分

	// 分类边界
	EvaluationBadThreshold     Evaluation = 40  // 恶性上限（≤40）
	EvaluationNeutralThreshold Evaluation = 65  // 中性上限（≤65）
	// Evaluation > 65 为良性
)

// 预定义的 Evaluation 常量（常用评分）
const (
	// 恶性评分（0~40）
	EvaluationVeryBad   Evaluation = 10  // 极恶（如雷劫）
	EvaluationBad       Evaluation = 25  // 较恶（如诅咒）
	EvaluationMildBad   Evaluation = 35  // 轻恶（如蚊虫叮咬）

	// 中性评分（41~65）
	EvaluationNeutral   Evaluation = 50  // 标准中性（如交换）
	EvaluationMixed     Evaluation = 55  // 混合效果（如尝一口）

	// 良性评分（66~100）
	EvaluationMildGood  Evaluation = 70  // 轻良（如草药）
	EvaluationGood      Evaluation = 80  // 较良（如奶茶）
	EvaluationVeryGood  Evaluation = 90  // 极良（如神眷）
	EvaluationExcellent Evaluation = 100 // 最佳
)

// IsValid 检查评分是否在有效范围内
func (e Evaluation) IsValid() bool {
	return e >= EvaluationMin && e <= EvaluationMax
}

// GetCategory 获取评分类别
func (e Evaluation) GetCategory() string {
	if e <= EvaluationBadThreshold {
		return "Bad"
	} else if e <= EvaluationNeutralThreshold {
		return "Neutral"
	}
	return "Good"
}

// IsGood 判断是否为良性
func (e Evaluation) IsGood() bool {
	return e > EvaluationNeutralThreshold
}

// IsNeutral 判断是否为中性
func (e Evaluation) IsNeutral() bool {
	return e > EvaluationBadThreshold && e <= EvaluationNeutralThreshold
}

// IsBad 判断是否为恶性
func (e Evaluation) IsBad() bool {
	return e <= EvaluationBadThreshold
}

// String 返回评分描述
func (e Evaluation) String() string {
	return fmt.Sprintf("Evaluation(%d): %s", e, e.GetCategory())
}

// Compare 比较两个评分
// 返回 1 表示当前评分更好，-1 表示更差，0 表示相同
func (e Evaluation) Compare(other Evaluation) int {
	if e > other {
		return 1
	} else if e < other {
		return -1
	}
	return 0
}

// ========== 旧版兼容（EventAttribute） ==========

// EventAttribute 旧版属性分类（已废弃，建议使用 Evaluation）
// 保留用于向后兼容
type EventAttribute int

const (
	AttributeGood     EventAttribute = iota // 良性
	AttributeNeutral                        // 中性
	AttributeBad                            // 恶性
)

// ToEvaluation 将旧版 EventAttribute 转换为 Evaluation
func (ea EventAttribute) ToEvaluation() Evaluation {
	switch ea {
	case AttributeGood:
		return EvaluationGood
	case AttributeNeutral:
		return EvaluationNeutral
	case AttributeBad:
		return EvaluationBad
	}
	return EvaluationNeutral
}

// ========== Buff 类型定义 ==========

type BuffType int

const (
	BuffTypeNone BuffType = iota
	// 负性 Buff
	BuffTypeCurse    // 诅咒
	BuffTypeLost     // 迷途
	BuffTypeCorrupt  // 腐化
	BuffTypePoison   // 毒瘴
	// 正性 Buff
	BuffTypeDivine   // 神眷
	BuffTypeHidden   // 隐匿
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
		BuffTypeCurse:    EvaluationBad,    // 诅咒：较恶
		BuffTypeLost:     EvaluationMildBad, // 迷途：轻恶
		BuffTypeCorrupt:  EvaluationBad,    // 腐化：较恶
		BuffTypePoison:   EvaluationVeryBad, // 毒瘴：极恶
		BuffTypeDivine:   EvaluationVeryGood, // 神眷：极良
		BuffTypeHidden:   EvaluationExcellent, // 隐匿：最佳（免疫）
		BuffTypeRain:     EvaluationGood,    // 甘霖：较良
		BuffTypeExorcism: EvaluationMildGood, // 辟邪：轻良
		BuffTypeFire:     EvaluationGood,    // 离火：较良
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
	SubscriptionID string   `json:"subscription_id"` // EventBus订阅ID
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
	Type        BuffType   `json:"type"`
	Eval        Evaluation `json:"evaluation"`    // 评分（替代旧版 Attribute）
	Name        string     `json:"name"`
	Desc        string     `json:"desc"`
	Duration    int        `json:"duration"`
	HPPerTurn   int        `json:"hp_per_turn"`
	LPPerTurn   int        `json:"lp_per_turn"`
	Special     string     `json:"special"`

	// Phase系统字段
	Phase       event.Phase `json:"phase"`        // 触发时机
	Priority    int        `json:"priority"`     // 执行优先级
	NeedConfirm bool       `json:"need_confirm"` // 是否需要用户确认（默认false）
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
			Phase:     event.PhaseBeforeTurn,  // 回合开始前执行
			Priority:  50,
		},
		BuffTypeDivine: {
			Type:      BuffTypeDivine,
			Eval:      eval,
			Name:      "神眷",
			Desc:      "接下来3回合LP+1",
			Duration:  3,
			LPPerTurn: 1,
			Phase:     event.PhaseBeforeTurn,  // 回合开始前执行
			Priority:  50,
		},
		BuffTypeHidden: {
			Type:      BuffTypeHidden,
			Eval:      eval,
			Name:      "隐匿",
			Desc:      "接下来3回合免疫任意事件、BUFF或道具的影响",
			Duration:  3,
			Special:   "immune",
			Phase:     event.PhasePreDamage,  // 受伤前自动免疫
			Priority:  100,  // 高优先级
		},
		BuffTypeLost: {
			Type:      BuffTypeLost,
			Eval:      eval,
			Name:      "迷途",
			Desc:      "下1回合朝反方向移动",
			Duration:  1,
			Special:   "reverse",
			Phase:     event.PhaseOnMove,  // 移动时反向
			Priority:  100,
		},
		BuffTypeCorrupt: {
			Type:      BuffTypeCorrupt,
			Eval:      eval,
			Name:      "腐化",
			Desc:      "接下来4回合每2回合HP-1",
			Duration:  4,
			HPPerTurn: -1,
			Phase:     event.PhaseAfterTurn,  // 回合结束后执行
			Priority:  50,
		},
		BuffTypeRain: {
			Type:      BuffTypeRain,
			Eval:      eval,
			Name:      "甘霖",
			Desc:      "接下来4回合每2回合HP+1",
			Duration:  4,
			HPPerTurn: 1,
			Phase:     event.PhaseAfterTurn,  // 回合结束后执行
			Priority:  50,
		},
		BuffTypeExorcism: {
			Type:      BuffTypeExorcism,
			Eval:      eval,
			Name:      "辟邪",
			Desc:      "接下来5回合无视毒瘴buff",
			Duration:  5,
			Special:   "immune_poison",
			Phase:     event.PhasePreEvent,  // 事件前免疫毒瘴
			Priority:  80,
		},
		BuffTypePoison: {
			Type:      BuffTypePoison,
			Eval:      eval,
			Name:      "毒瘴",
			Desc:      "接下来3回合每回合受一次恶性随机事件影响",
			Duration:  3,
			Special:   "bad_event_per_turn",
			Phase:     event.PhaseBeforeTurn,  // 回合开始前触发
			Priority:  30,  // 低优先级，在其他Buff效果之后
		},
		BuffTypeFire: {
			Type:      BuffTypeFire,
			Eval:      eval,
			Name:      "离火",
			Desc:      "朱雀阵营增益，每4回合LP+1",
			Duration:  -1,
			Special:   "zhuque_passive",
			Phase:     event.PhaseBeforeTurn,  // 回合开始前检查，内部判断是否生效
			Priority:  10,  // 最低优先级，在其他效果之后
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

// GetBuffsByCategory 按类别获取 Buff 列表（兼容旧版）
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