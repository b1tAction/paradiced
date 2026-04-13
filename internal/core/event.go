package core

// ========== Event Type Definitions ==========

type EventType int

const (
	EventTypeNone EventType = iota

	// Good events (Evaluation > 65)
	EventTypeHerb         // Herb: HP+1
	EventTypeMilkTea      // MilkTea: LP+1
	EventTypeRelic        // Relic: item draw
	EventTypeDivineBless  // DivineBless: Divine buff

	// Neutral events (Evaluation 41~65)
	EventTypeExchange     // Exchange: swap position with random player
	EventTypeHiddenBuff   // HiddenBuff: Hidden buff
	EventTypeTasteTest    // TasteTest: random buff (Corrupt/Rain)

	// Bad events (Evaluation ≤ 40)
	EventTypeMosquito     // Mosquito: HP-1
	EventTypeGhostHit     // GhostHit: HP-1
	EventTypeDogPoop      // DogPoop: LP-1
	EventTypeThief        // Thief: random item loss
	EventTypeCurseBuddha  // CurseBuddha: Curse buff
	EventTypeLostWay      // LostWay: Lost buff
	EventTypeThunder      // Thunder: HP to 0
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

// GetEvaluation returns the event's evaluation score.
func (et EventType) GetEvaluation() Evaluation {
	evalMap := map[EventType]Evaluation{
		// Good events
		EventTypeHerb:         EvaluationMildGood,  // Herb: mild good
		EventTypeMilkTea:      EvaluationGood,      // MilkTea: good
		EventTypeRelic:        EvaluationVeryGood,  // Relic: very good
		EventTypeDivineBless:  EvaluationExcellent, // DivineBless: excellent

		// Neutral events
		EventTypeExchange:     EvaluationNeutral,  // Exchange: neutral
		EventTypeHiddenBuff:   EvaluationGood,     // HiddenBuff: good
		EventTypeTasteTest:    EvaluationMixed,    // TasteTest: mixed

		// Bad events
		EventTypeMosquito:     EvaluationMildBad,  // Mosquito: mild bad
		EventTypeGhostHit:     EvaluationMildBad,  // GhostHit: mild bad
		EventTypeDogPoop:      EvaluationMildBad,  // DogPoop: mild bad
		EventTypeThief:        EvaluationBad,      // Thief: bad
		EventTypeCurseBuddha:  EvaluationBad,      // CurseBuddha: bad
		EventTypeLostWay:      EvaluationMildBad,  // LostWay: mild bad
		EventTypeThunder:      EvaluationVeryBad,  // Thunder: very bad
	}
	if eval, ok := evalMap[et]; ok {
		return eval
	}
	return EvaluationNeutral
}

// GetCategory returns the event's category (based on Evaluation).
func (et EventType) GetCategory() string {
	return et.GetEvaluation().GetCategory()
}

// ========== Event Definition ==========

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

// ========== Event Registry ==========

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

// GetEventsByEvaluationRange returns Events within the specified Evaluation range.
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

// GetEventsByCategory returns Events by category.
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

// GetAllEventDefinitions returns all Event definitions.
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