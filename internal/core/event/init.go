package event

import (
	"github.com/b1tAction/Fated/pkg/constants"
)

// GlobalEventRegistry is the global Event definition registry.
// Initialized at package load time with all Event definitions.
var GlobalEventRegistry *EventRegistry

// init initializes the global Event registry and registers all Event definitions.
func init() {
	GlobalEventRegistry = NewEventRegistry()
	registerAllEvents()
}

// registerAllEvents registers all Event definitions.
func registerAllEvents() {
	// ========== Good Events (Evaluation > 65) ==========

	GlobalEventRegistry.RegisterEvent(&EventDefinition{
		Type:        constants.EventTypeHerb,
		Eval:        constants.EvaluationMildGood,
		EnglishName: "Herb",
		Name:        "采集到草药",
		Desc:        "在路边发现了草药，恢复了体力",
		HPChange:    1,
	})

	GlobalEventRegistry.RegisterEvent(&EventDefinition{
		Type:        constants.EventTypeMilkTea,
		Eval:        constants.EvaluationGood,
		EnglishName: "MilkTea",
		Name:        "捡到奶茶",
		Desc:        "捡到了一杯奶茶，一口就吃到了猪猪欸",
		LPChange:    1,
	})

	GlobalEventRegistry.RegisterEvent(&EventDefinition{
		Type:          constants.EventTypeRelic,
		Eval:          constants.EvaluationVeryGood,
		EnglishName:   "Relic",
		Name:          "捡到勇士的圣遗物",
		Desc:          "发现了古老圣遗物，获得一次道具抽奖机会",
		SpecialEffect: constants.SpecialDrawItem,
	})

	GlobalEventRegistry.RegisterEvent(&EventDefinition{
		Type:        constants.EventTypeDivineBless,
		Eval:        constants.EvaluationExcellent,
		EnglishName: "DivineBless",
		Name:        "受到天使眷顾",
		Desc:        "天使的祝福降临，获得神眷Buff",
		BuffType:    constants.BuffTypeDivine,
	})

	// ========== Neutral Events (Evaluation 41~65) ==========

	GlobalEventRegistry.RegisterEvent(&EventDefinition{
		Type:          constants.EventTypeExchange,
		Eval:          constants.EvaluationNeutral,
		EnglishName:   "Exchange",
		Name:          "交换",
		Desc:          "命运之手将你与另一位玩家交换位置",
		SpecialEffect: constants.SpecialSwapPosition,
	})

	GlobalEventRegistry.RegisterEvent(&EventDefinition{
		Type:        constants.EventTypeHiddenBuff,
		Eval:        constants.EvaluationGood,
		EnglishName: "HiddenBuff",
		Name:        "麻了",
		Desc:        "身体麻木，获得隐匿Buff",
		BuffType:    constants.BuffTypeHidden,
	})

	GlobalEventRegistry.RegisterEvent(&EventDefinition{
		Type:          constants.EventTypeTasteTest,
		Eval:          constants.EvaluationMixed,
		EnglishName:   "TasteTest",
		Name:          "这是什么？尝一口",
		Desc:          "发现神秘物质，尝试后获得随机效果",
		SpecialEffect: constants.SpecialRandomBuff,
	})

	// ========== Bad Events (Evaluation ≤ 40) ==========

	GlobalEventRegistry.RegisterEvent(&EventDefinition{
		Type:        constants.EventTypeMosquito,
		Eval:        constants.EvaluationMildBad,
		EnglishName: "Mosquito",
		Name:        "被蚊虫叮咬",
		Desc:        "丛林中的蚊虫叮咬了你",
		HPChange:    -1,
	})

	GlobalEventRegistry.RegisterEvent(&EventDefinition{
		Type:        constants.EventTypeGhostHit,
		Eval:        constants.EvaluationMildBad,
		EnglishName: "GhostHit",
		Name:        "偶遇孤魂野鬼",
		Desc:        "被野鬼打了一闷棍",
		HPChange:    -1,
	})

	GlobalEventRegistry.RegisterEvent(&EventDefinition{
		Type:        constants.EventTypeDogPoop,
		Eval:        constants.EvaluationMildBad,
		EnglishName: "DogPoop",
		Name:        "踩到了狗屎",
		Desc:        "运气糟糕的一天",
		LPChange:    -1,
	})

	GlobalEventRegistry.RegisterEvent(&EventDefinition{
		Type:          constants.EventTypeThief,
		Eval:          constants.EvaluationBad,
		EnglishName:   "Thief",
		Name:          "啊？！贼",
		Desc:          "遭遇盗贼，随机丢失一个道具",
		SpecialEffect: constants.SpecialLoseItem,
	})

	GlobalEventRegistry.RegisterEvent(&EventDefinition{
		Type:        constants.EventTypeCurseBuddha,
		Eval:        constants.EvaluationBad,
		EnglishName: "CurseBuddha",
		Name:        "虔诚拜三拜",
		Desc:        "拜路边的野佛，获得诅咒Buff",
		BuffType:    constants.BuffTypeCurse,
	})

	GlobalEventRegistry.RegisterEvent(&EventDefinition{
		Type:        constants.EventTypeLostWay,
		Eval:        constants.EvaluationMildBad,
		EnglishName: "LostWay",
		Name:        "迷途",
		Desc:        "迷失方向，获得迷途Buff",
		BuffType:    constants.BuffTypeLost,
	})

	GlobalEventRegistry.RegisterEvent(&EventDefinition{
		Type:        constants.EventTypeThunder,
		Eval:        constants.EvaluationVeryBad,
		EnglishName: "Thunder",
		Name:        "雷劫",
		Desc:        "天雷降临，HP归零",
		HPChange:    -999,
	})
}