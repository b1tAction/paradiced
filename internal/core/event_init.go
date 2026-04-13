package core

// registerAllEvents registers all Event definitions.
// Called from init() in buff_init.go.
func registerAllEvents() {
	// ========== Good Events (Evaluation > 65) ==========

	GlobalRegistry.RegisterEvent(&EventDefinition{
		Type:        EventTypeHerb,
		Eval:        EvaluationMildGood,
		EnglishName: "Herb",
		Name:        "采集到草药",
		Desc:        "在路边发现了草药，恢复了体力",
		HPChange:    1,
	})

	GlobalRegistry.RegisterEvent(&EventDefinition{
		Type:        EventTypeMilkTea,
		Eval:        EvaluationGood,
		EnglishName: "MilkTea",
		Name:        "捡到奶茶",
		Desc:        "捡到了一杯奶茶，一口就吃到了猪猪欸",
		LPChange:    1,
	})

	GlobalRegistry.RegisterEvent(&EventDefinition{
		Type:          EventTypeRelic,
		Eval:          EvaluationVeryGood,
		EnglishName:   "Relic",
		Name:          "捡到勇士的圣遗物",
		Desc:          "发现了古老圣遗物，获得一次道具抽奖机会",
		SpecialEffect: SpecialDrawItem,
	})

	GlobalRegistry.RegisterEvent(&EventDefinition{
		Type:        EventTypeDivineBless,
		Eval:        EvaluationExcellent,
		EnglishName: "DivineBless",
		Name:        "受到天使眷顾",
		Desc:        "天使的祝福降临，获得神眷Buff",
		BuffType:    BuffTypeDivine,
	})

	// ========== Neutral Events (Evaluation 41~65) ==========

	GlobalRegistry.RegisterEvent(&EventDefinition{
		Type:          EventTypeExchange,
		Eval:          EvaluationNeutral,
		EnglishName:   "Exchange",
		Name:          "交换",
		Desc:          "命运之手将你与另一位玩家交换位置",
		SpecialEffect: SpecialSwapPosition,
	})

	GlobalRegistry.RegisterEvent(&EventDefinition{
		Type:        EventTypeHiddenBuff,
		Eval:        EvaluationGood,
		EnglishName: "HiddenBuff",
		Name:        "麻了",
		Desc:        "身体麻木，获得隐匿Buff",
		BuffType:    BuffTypeHidden,
	})

	GlobalRegistry.RegisterEvent(&EventDefinition{
		Type:          EventTypeTasteTest,
		Eval:          EvaluationMixed,
		EnglishName:   "TasteTest",
		Name:          "这是什么？尝一口",
		Desc:          "发现神秘物质，尝试后获得随机效果",
		SpecialEffect: SpecialRandomBuff,
	})

	// ========== Bad Events (Evaluation ≤ 40) ==========

	GlobalRegistry.RegisterEvent(&EventDefinition{
		Type:        EventTypeMosquito,
		Eval:        EvaluationMildBad,
		EnglishName: "Mosquito",
		Name:        "被蚊虫叮咬",
		Desc:        "丛林中的蚊虫叮咬了你",
		HPChange:    -1,
	})

	GlobalRegistry.RegisterEvent(&EventDefinition{
		Type:        EventTypeGhostHit,
		Eval:        EvaluationMildBad,
		EnglishName: "GhostHit",
		Name:        "偶遇孤魂野鬼",
		Desc:        "被野鬼打了一闷棍",
		HPChange:    -1,
	})

	GlobalRegistry.RegisterEvent(&EventDefinition{
		Type:        EventTypeDogPoop,
		Eval:        EvaluationMildBad,
		EnglishName: "DogPoop",
		Name:        "踩到了狗屎",
		Desc:        "运气糟糕的一天",
		LPChange:    -1,
	})

	GlobalRegistry.RegisterEvent(&EventDefinition{
		Type:          EventTypeThief,
		Eval:          EvaluationBad,
		EnglishName:   "Thief",
		Name:          "啊？！贼",
		Desc:          "遭遇盗贼，随机丢失一个道具",
		SpecialEffect: SpecialLoseItem,
	})

	GlobalRegistry.RegisterEvent(&EventDefinition{
		Type:        EventTypeCurseBuddha,
		Eval:        EvaluationBad,
		EnglishName: "CurseBuddha",
		Name:        "虔诚拜三拜",
		Desc:        "拜路边的野佛，获得诅咒Buff",
		BuffType:    BuffTypeCurse,
	})

	GlobalRegistry.RegisterEvent(&EventDefinition{
		Type:        EventTypeLostWay,
		Eval:        EvaluationMildBad,
		EnglishName: "LostWay",
		Name:        "迷途",
		Desc:        "迷失方向，获得迷途Buff",
		BuffType:    BuffTypeLost,
	})

	GlobalRegistry.RegisterEvent(&EventDefinition{
		Type:        EventTypeThunder,
		Eval:        EvaluationVeryBad,
		EnglishName: "Thunder",
		Name:        "雷劫",
		Desc:        "天雷降临，HP归零",
		HPChange:    -999,
	})
}