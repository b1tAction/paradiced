package event

import (
	"github.com/b1tAction/Fated/internal/core/buff"
	"github.com/b1tAction/Fated/internal/core/types"
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
		Type:        EventTypeHerb,
		Eval:        types.EvaluationMildGood,
		EnglishName: "Herb",
		Name:        "采集到草药",
		Desc:        "在路边发现了草药，恢复了体力",
		HPChange:    1,
	})

	GlobalEventRegistry.RegisterEvent(&EventDefinition{
		Type:        EventTypeMilkTea,
		Eval:        types.EvaluationGood,
		EnglishName: "MilkTea",
		Name:        "捡到奶茶",
		Desc:        "捡到了一杯奶茶，一口就吃到了猪猪欸",
		LPChange:    1,
	})

	GlobalEventRegistry.RegisterEvent(&EventDefinition{
		Type:          EventTypeRelic,
		Eval:          types.EvaluationVeryGood,
		EnglishName:   "Relic",
		Name:          "捡到勇士的圣遗物",
		Desc:          "发现了古老圣遗物，获得一次道具抽奖机会",
		SpecialEffect: types.SpecialDrawItem,
	})

	GlobalEventRegistry.RegisterEvent(&EventDefinition{
		Type:        EventTypeDivineBless,
		Eval:        types.EvaluationExcellent,
		EnglishName: "DivineBless",
		Name:        "受到天使眷顾",
		Desc:        "天使的祝福降临，获得神眷Buff",
		BuffType:    buff.BuffTypeDivine,
	})

	// ========== Neutral Events (Evaluation 41~65) ==========

	GlobalEventRegistry.RegisterEvent(&EventDefinition{
		Type:          EventTypeExchange,
		Eval:          types.EvaluationNeutral,
		EnglishName:   "Exchange",
		Name:          "交换",
		Desc:          "命运之手将你与另一位玩家交换位置",
		SpecialEffect: types.SpecialSwapPosition,
	})

	GlobalEventRegistry.RegisterEvent(&EventDefinition{
		Type:        EventTypeHiddenBuff,
		Eval:        types.EvaluationGood,
		EnglishName: "HiddenBuff",
		Name:        "麻了",
		Desc:        "身体麻木，获得隐匿Buff",
		BuffType:    buff.BuffTypeHidden,
	})

	GlobalEventRegistry.RegisterEvent(&EventDefinition{
		Type:          EventTypeTasteTest,
		Eval:          types.EvaluationMixed,
		EnglishName:   "TasteTest",
		Name:          "这是什么？尝一口",
		Desc:          "发现神秘物质，尝试后获得随机效果",
		SpecialEffect: types.SpecialRandomBuff,
	})

	// ========== Bad Events (Evaluation ≤ 40) ==========

	GlobalEventRegistry.RegisterEvent(&EventDefinition{
		Type:        EventTypeMosquito,
		Eval:        types.EvaluationMildBad,
		EnglishName: "Mosquito",
		Name:        "被蚊虫叮咬",
		Desc:        "丛林中的蚊虫叮咬了你",
		HPChange:    -1,
	})

	GlobalEventRegistry.RegisterEvent(&EventDefinition{
		Type:        EventTypeGhostHit,
		Eval:        types.EvaluationMildBad,
		EnglishName: "GhostHit",
		Name:        "偶遇孤魂野鬼",
		Desc:        "被野鬼打了一闷棍",
		HPChange:    -1,
	})

	GlobalEventRegistry.RegisterEvent(&EventDefinition{
		Type:        EventTypeDogPoop,
		Eval:        types.EvaluationMildBad,
		EnglishName: "DogPoop",
		Name:        "踩到了狗屎",
		Desc:        "运气糟糕的一天",
		LPChange:    -1,
	})

	GlobalEventRegistry.RegisterEvent(&EventDefinition{
		Type:          EventTypeThief,
		Eval:          types.EvaluationBad,
		EnglishName:   "Thief",
		Name:          "啊？！贼",
		Desc:          "遭遇盗贼，随机丢失一个道具",
		SpecialEffect: types.SpecialLoseItem,
	})

	GlobalEventRegistry.RegisterEvent(&EventDefinition{
		Type:        EventTypeCurseBuddha,
		Eval:        types.EvaluationBad,
		EnglishName: "CurseBuddha",
		Name:        "虔诚拜三拜",
		Desc:        "拜路边的野佛，获得诅咒Buff",
		BuffType:    buff.BuffTypeCurse,
	})

	GlobalEventRegistry.RegisterEvent(&EventDefinition{
		Type:        EventTypeLostWay,
		Eval:        types.EvaluationMildBad,
		EnglishName: "LostWay",
		Name:        "迷途",
		Desc:        "迷失方向，获得迷途Buff",
		BuffType:    buff.BuffTypeLost,
	})

	GlobalEventRegistry.RegisterEvent(&EventDefinition{
		Type:        EventTypeThunder,
		Eval:        types.EvaluationVeryBad,
		EnglishName: "Thunder",
		Name:        "雷劫",
		Desc:        "天雷降临，HP归零",
		HPChange:    -999,
	})
}