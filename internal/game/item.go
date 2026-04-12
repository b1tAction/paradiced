package game

// ========== Item 类型定义 ==========

type ItemType int

const (
	ItemTypeNone ItemType = iota
	ItemTypeReverseClock // 反方向的钟
	ItemTypeAnyDoor      // 任意门
	ItemTypeDiceSwap     // 骰子交换
	ItemTypeDiceUpgrade  // 骰子升级卡
)

func (it ItemType) String() string {
	names := map[ItemType]string{
		ItemTypeNone:         "None",
		ItemTypeReverseClock: "ReverseClock",
		ItemTypeAnyDoor:      "AnyDoor",
		ItemTypeDiceSwap:     "DiceSwap",
		ItemTypeDiceUpgrade:  "DiceUpgrade",
	}
	if name, ok := names[it]; ok {
		return name
	}
	return "Unknown"
}

func (it ItemType) IsValid() bool {
	return it > ItemTypeNone && it <= ItemTypeDiceUpgrade
}

// GetEvaluation 获取道具的评分
func (it ItemType) GetEvaluation() Evaluation {
	evalMap := map[ItemType]Evaluation{
		ItemTypeReverseClock: EvaluationBad,     // 反方向的钟：较恶（对他人负面）
		ItemTypeAnyDoor:      EvaluationNeutral, // 任意门：中性
		ItemTypeDiceSwap:     EvaluationNeutral, // 骰子交换：中性
		ItemTypeDiceUpgrade:  EvaluationGood,    // 骰子升级：较良
	}
	if eval, ok := evalMap[it]; ok {
		return eval
	}
	return EvaluationNeutral
}

// GetCategory 获取道具的类别（基于 Evaluation）
func (it ItemType) GetCategory() string {
	return it.GetEvaluation().GetCategory()
}

// ========== Item 实例 ==========

type Item struct {
	Type     ItemType `json:"type"`
	ID       string   `json:"id"`
	Usable   bool     `json:"usable"`
	TargetID string   `json:"target_id"`
}

func NewItem(itemType ItemType, id string) *Item {
	return &Item{
		Type:   itemType,
		ID:     id,
		Usable: true,
	}
}

// ========== Item 静态定义 ==========

type ItemDefinition struct {
	Type        ItemType   `json:"type"`
	Eval        Evaluation `json:"evaluation"`
	Name        string     `json:"name"`
	Desc        string     `json:"desc"`
	TargetSelf  bool       `json:"target_self"`
	TargetOther bool       `json:"target_other"`
	BuffType    BuffType   `json:"buff_type"`
	Range       int        `json:"range"`
}

func (it ItemType) GetItemDefinition() *ItemDefinition {
	eval := it.GetEvaluation()
	definitions := map[ItemType]*ItemDefinition{
		ItemTypeReverseClock: {
			Type:        ItemTypeReverseClock,
			Eval:        eval,
			Name:        "反方向的钟",
			Desc:        "给予指定玩家迷途Buff",
			TargetSelf:  false,
			TargetOther: true,
			BuffType:    BuffTypeLost,
		},
		ItemTypeAnyDoor: {
			Type:        ItemTypeAnyDoor,
			Eval:        eval,
			Name:        "任意门",
			Desc:        "去到30格内指定玩家身边",
			TargetSelf:  false,
			TargetOther: true,
			Range:       30,
		},
		ItemTypeDiceSwap: {
			Type:        ItemTypeDiceSwap,
			Eval:        eval,
			Name:        "骰子交换",
			Desc:        "与指定玩家交换骰子等级",
			TargetSelf:  false,
			TargetOther: true,
		},
		ItemTypeDiceUpgrade: {
			Type:        ItemTypeDiceUpgrade,
			Eval:        eval,
			Name:        "骰子升级卡",
			Desc:        "将当前骰子升级为更高等级",
			TargetSelf:  true,
			TargetOther: false,
		},
	}
	if def, ok := definitions[it]; ok {
		return def
	}
	return nil
}

// ========== Item 注册表 ==========

type ItemRegistry struct {
	AllItems     []ItemType `json:"all_items"`
	GoodItems    []ItemType `json:"good_items"`
	NeutralItems []ItemType `json:"neutral_items"`
	BadItems     []ItemType `json:"bad_items"`
}

func NewItemRegistry() *ItemRegistry {
	all := []ItemType{
		ItemTypeReverseClock, ItemTypeAnyDoor, ItemTypeDiceSwap, ItemTypeDiceUpgrade,
	}

	var good, neutral, bad []ItemType
	for _, it := range all {
		eval := it.GetEvaluation()
		if eval.IsGood() {
			good = append(good, it)
		} else if eval.IsNeutral() {
			neutral = append(neutral, it)
		} else {
			bad = append(bad, it)
		}
	}

	return &ItemRegistry{
		AllItems:     all,
		GoodItems:    good,
		NeutralItems: neutral,
		BadItems:     bad,
	}
}

// GetItemsByEvaluationRange 按 Evaluation 范围获取道具列表
func (ir *ItemRegistry) GetItemsByEvaluationRange(minEval, maxEval Evaluation) []ItemType {
	var result []ItemType
	for _, it := range ir.AllItems {
		eval := it.GetEvaluation()
		if eval >= minEval && eval <= maxEval {
			result = append(result, it)
		}
	}
	return result
}

// GetItemsByCategory 按类别获取道具列表（兼容旧版）
func (ir *ItemRegistry) GetItemsByCategory(category string) []ItemType {
	switch category {
	case "Good":
		return ir.GoodItems
	case "Neutral":
		return ir.NeutralItems
	case "Bad":
		return ir.BadItems
	}
	return ir.AllItems
}

// GetAllItemDefinitions 获取所有道具定义
func (ir *ItemRegistry) GetAllItemDefinitions() []*ItemDefinition {
	defs := make([]*ItemDefinition, 0, len(ir.AllItems))
	for _, it := range ir.AllItems {
		def := it.GetItemDefinition()
		if def != nil {
			defs = append(defs, def)
		}
	}
	return defs
}
