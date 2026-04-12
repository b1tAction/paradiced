package core

// ========== Event 类型定义 ==========

type EventType int

const (
	EventTypeNone EventType = iota

	// 良性事件 (Evaluation > 65)
	EventTypeHerb         // 采集到草药：HP+1
	EventTypeMilkTea      // 捡到奶茶：LP+1
	EventTypeRelic        // 捡到勇士的圣遗物：道具抽奖
	EventTypeDivineBless  // 受到天使眷顾：获得神眷Buff

	// 中性事件 (Evaluation 41~65)
	EventTypeExchange     // 交换：与随机玩家交换位置
	EventTypeHiddenBuff   // 麻了：获得隐匿Buff
	EventTypeTasteTest    // 这是什么？尝一口：获得腐化/甘霖Buff

	// 恶性事件 (Evaluation ≤ 40)
	EventTypeMosquito     // 被蚊虫叮咬：HP-1
	EventTypeGhostHit     // 偶遇孤魂野鬼：HP-1
	EventTypeDogPoop      // 踩到了狗屎：LP-1
	EventTypeThief        // 啊？！贼：随机丢失道具
	EventTypeCurseBuddha  // 虔诚拜三拜：获得诅咒Buff
	EventTypeLostWay      // 迷途：获得迷途Buff
	EventTypeThunder      // 雷劫：HP归零
)

func (et EventType) String() string {
	names := map[EventType]string{
		EventTypeNone:        "None",
		EventTypeHerb:        "Herb",
		EventTypeMilkTea:     "MilkTea",
		EventTypeRelic:       "Relic",
		EventTypeDivineBless: "DivineBless",
		EventTypeExchange:    "Exchange",
		EventTypeHiddenBuff:  "HiddenBuff",
		EventTypeTasteTest:   "TasteTest",
		EventTypeMosquito:    "Mosquito",
		EventTypeGhostHit:    "GhostHit",
		EventTypeDogPoop:     "DogPoop",
		EventTypeThief:       "Thief",
		EventTypeCurseBuddha: "CurseBuddha",
		EventTypeLostWay:     "LostWay",
		EventTypeThunder:     "Thunder",
	}
	if name, ok := names[et]; ok {
		return name
	}
	return "Unknown"
}

func (et EventType) IsValid() bool {
	return et > EventTypeNone && et <= EventTypeThunder
}

// GetEvaluation 获取事件的评分
func (et EventType) GetEvaluation() Evaluation {
	evalMap := map[EventType]Evaluation{
		// 良性事件
		EventTypeHerb:         EvaluationMildGood,  // 草药：轻良
		EventTypeMilkTea:      EvaluationGood,      // 奶茶：较良
		EventTypeRelic:        EvaluationVeryGood,  // 圣遗物：极良
		EventTypeDivineBless:  EvaluationExcellent, // 神眷：最佳

		// 中性事件
		EventTypeExchange:     EvaluationNeutral, // 交换：标准中性
		EventTypeHiddenBuff:   EvaluationGood,    // 隐匿：较良
		EventTypeTasteTest:    EvaluationMixed,   // 品尝：混合效果

		// 恶性事件
		EventTypeMosquito:     EvaluationMildBad, // 蚊虫：轻恶
		EventTypeGhostHit:     EvaluationMildBad, // 野鬼：轻恶
		EventTypeDogPoop:      EvaluationMildBad, // 狗屎：轻恶
		EventTypeThief:        EvaluationBad,     // 盗贼：较恶
		EventTypeCurseBuddha:  EvaluationBad,     // 野佛：较恶
		EventTypeLostWay:      EvaluationMildBad, // 迷途：轻恶
		EventTypeThunder:      EvaluationVeryBad, // 雷劫：极恶
	}
	if eval, ok := evalMap[et]; ok {
		return eval
	}
	return EvaluationNeutral
}

// GetCategory 获取事件的类别（基于 Evaluation）
func (et EventType) GetCategory() string {
	return et.GetEvaluation().GetCategory()
}

// ========== Event 静态定义 ==========

type EventDefinition struct {
	Type       EventType  `json:"type"`
	Eval       Evaluation `json:"evaluation"`
	Name       string     `json:"name"`
	Desc       string     `json:"desc"`
	HPChange   int        `json:"hp_change"`
	LPChange   int        `json:"lp_change"`
	BuffType   BuffType   `json:"buff_type"`
	ItemAction string     `json:"item_action"`
}

func (et EventType) GetEventDefinition() *EventDefinition {
	eval := et.GetEvaluation()
	definitions := map[EventType]*EventDefinition{
		EventTypeHerb: {
			Type:     EventTypeHerb,
			Eval:     eval,
			Name:     "采集到草药",
			Desc:     "在路边发现了草药，恢复了体力",
			HPChange: 1,
		},
		EventTypeMilkTea: {
			Type:     EventTypeMilkTea,
			Eval:     eval,
			Name:     "捡到奶茶",
			Desc:     "捡到了一杯奶茶，一口就吃到了猪猪欸",
			LPChange: 1,
		},
		EventTypeRelic: {
			Type:       EventTypeRelic,
			Eval:       eval,
			Name:       "捡到勇士的圣遗物",
			Desc:       "发现了古老圣遗物，获得一次道具抽奖机会",
			ItemAction: "draw",
		},
		EventTypeDivineBless: {
			Type:     EventTypeDivineBless,
			Eval:     eval,
			Name:     "受到天使眷顾",
			Desc:     "天使的祝福降临，获得神眷Buff",
			BuffType: BuffTypeDivine,
		},
		EventTypeExchange: {
			Type:  EventTypeExchange,
			Eval:  eval,
			Name:  "交换",
			Desc:  "命运之手将你与另一位玩家交换位置",
		},
		EventTypeHiddenBuff: {
			Type:     EventTypeHiddenBuff,
			Eval:     eval,
			Name:     "麻了",
			Desc:     "身体麻木，获得隐匿Buff",
			BuffType: BuffTypeHidden,
		},
		EventTypeTasteTest: {
			Type:  EventTypeTasteTest,
			Eval:  eval,
			Name:  "这是什么？尝一口",
			Desc:  "发现神秘物质，尝试后获得随机效果",
		},
		EventTypeMosquito: {
			Type:     EventTypeMosquito,
			Eval:     eval,
			Name:     "被蚊虫叮咬",
			Desc:     "丛林中的蚊虫叮咬了你",
			HPChange: -1,
		},
		EventTypeGhostHit: {
			Type:     EventTypeGhostHit,
			Eval:     eval,
			Name:     "偶遇孤魂野鬼",
			Desc:     "被野鬼打了一闷棍",
			HPChange: -1,
		},
		EventTypeDogPoop: {
			Type:     EventTypeDogPoop,
			Eval:     eval,
			Name:     "踩到了狗屎",
			Desc:     "运气糟糕的一天",
			LPChange: -1,
		},
		EventTypeThief: {
			Type:       EventTypeThief,
			Eval:       eval,
			Name:       "啊？！贼",
			Desc:       "遭遇盗贼，随机丢失一个道具",
			ItemAction: "lose",
		},
		EventTypeCurseBuddha: {
			Type:     EventTypeCurseBuddha,
			Eval:     eval,
			Name:     "虔诚拜三拜",
			Desc:     "拜路边的野佛，获得诅咒Buff",
			BuffType: BuffTypeCurse,
		},
		EventTypeLostWay: {
			Type:     EventTypeLostWay,
			Eval:     eval,
			Name:     "迷途",
			Desc:     "迷失方向，获得迷途Buff",
			BuffType: BuffTypeLost,
		},
		EventTypeThunder: {
			Type:     EventTypeThunder,
			Eval:     eval,
			Name:     "雷劫",
			Desc:     "天雷降临，HP归零",
			HPChange: -999,
		},
	}
	if def, ok := definitions[et]; ok {
		return def
	}
	return nil
}

// ========== Event 注册表 ==========

type EventRegistry struct {
	AllEvents     []EventType `json:"all_events"`
	GoodEvents    []EventType `json:"good_events"`
	NeutralEvents []EventType `json:"neutral_events"`
	BadEvents     []EventType `json:"bad_events"`
}

func NewEventRegistry() *EventRegistry {
	all := []EventType{
		EventTypeHerb, EventTypeMilkTea, EventTypeRelic, EventTypeDivineBless,
		EventTypeExchange, EventTypeHiddenBuff, EventTypeTasteTest,
		EventTypeMosquito, EventTypeGhostHit, EventTypeDogPoop,
		EventTypeThief, EventTypeCurseBuddha, EventTypeLostWay, EventTypeThunder,
	}

	var good, neutral, bad []EventType
	for _, et := range all {
		eval := et.GetEvaluation()
		if eval.IsGood() {
			good = append(good, et)
		} else if eval.IsNeutral() {
			neutral = append(neutral, et)
		} else {
			bad = append(bad, et)
		}
	}

	return &EventRegistry{
		AllEvents:     all,
		GoodEvents:    good,
		NeutralEvents: neutral,
		BadEvents:     bad,
	}
}

// GetEventsByEvaluationRange 按 Evaluation 范围获取事件列表
func (er *EventRegistry) GetEventsByEvaluationRange(minEval, maxEval Evaluation) []EventType {
	var result []EventType
	for _, et := range er.AllEvents {
		eval := et.GetEvaluation()
		if eval >= minEval && eval <= maxEval {
			result = append(result, et)
		}
	}
	return result
}

// GetEventsByCategory 按类别获取事件列表
func (er *EventRegistry) GetEventsByCategory(category string) []EventType {
	switch category {
	case "Good":
		return er.GoodEvents
	case "Neutral":
		return er.NeutralEvents
	case "Bad":
		return er.BadEvents
	}
	return er.AllEvents
}

// GetAllEventDefinitions 获取所有事件定义
func (er *EventRegistry) GetAllEventDefinitions() []*EventDefinition {
	defs := make([]*EventDefinition, 0, len(er.AllEvents))
	for _, et := range er.AllEvents {
		def := et.GetEventDefinition()
		if def != nil {
			defs = append(defs, def)
		}
	}
	return defs
}