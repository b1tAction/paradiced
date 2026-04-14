package event

import (
	"github.com/b1tAction/paradiced/pkg/constants"
	"github.com/b1tAction/paradiced/pkg/event"
	"github.com/b1tAction/paradiced/pkg/protocol"
)

// GlobalEventRegistry is the global Event definition registry.
// Initialized at package load time with all Event definitions.
var GlobalEventRegistry *EventRegistry

// init initializes the global Event registry and registers all Event definitions.
func init() {
	GlobalEventRegistry = NewEventRegistry()
	registerAllEvents()
}

// registerAllEvents registers all Event definitions with their handler configs.
func registerAllEvents() {
	// ========== Good Events (Evaluation > 65) ==========

	// Herb: HP+1
	GlobalEventRegistry.RegisterEvent(&EventDefinition{
		Type:        constants.EventTypeHerb,
		Eval:        constants.EvaluationMildGood,
		EnglishName: "Herb",
		Name:        "采集到草药",
		Desc:        "在路边发现了草药，恢复了体力",
	}, &EventHandlerConfig{
		Handler: createModifyHPHandler(1),
	})

	// MilkTea: LP+1
	GlobalEventRegistry.RegisterEvent(&EventDefinition{
		Type:        constants.EventTypeMilkTea,
		Eval:        constants.EvaluationGood,
		EnglishName: "MilkTea",
		Name:        "捡到奶茶",
		Desc:        "捡到了一杯奶茶，一口就吃到了猪猪欸",
	}, &EventHandlerConfig{
		Handler: createModifyLPHandler(1),
	})

	// Relic: Draw item
	GlobalEventRegistry.RegisterEvent(&EventDefinition{
		Type:        constants.EventTypeRelic,
		Eval:        constants.EvaluationVeryGood,
		EnglishName: "Relic",
		Name:        "捡到勇士的圣遗物",
		Desc:        "发现了古老圣遗物，获得一次道具抽奖机会",
	}, &EventHandlerConfig{
		Handler: handleDrawItem,
	})

	// DivineBless: Give Divine buff
	GlobalEventRegistry.RegisterEvent(&EventDefinition{
		Type:        constants.EventTypeDivineBless,
		Eval:        constants.EvaluationExcellent,
		EnglishName: "DivineBless",
		Name:        "受到天使眷顾",
		Desc:        "天使的祝福降临，获得神眷Buff",
	}, &EventHandlerConfig{
		Handler: createGiveBuffHandler(constants.BuffTypeDivine, 3),
	})

	// ========== Neutral Events (Evaluation 41~65) ==========

	// Exchange: Swap position with another player
	GlobalEventRegistry.RegisterEvent(&EventDefinition{
		Type:        constants.EventTypeExchange,
		Eval:        constants.EvaluationNeutral,
		EnglishName: "Exchange",
		Name:        "交换",
		Desc:        "命运之手将你与另一位玩家交换位置",
	}, &EventHandlerConfig{
		Handler: handleSwapPosition,
	})

	// HiddenBuff: Give Hidden buff
	GlobalEventRegistry.RegisterEvent(&EventDefinition{
		Type:        constants.EventTypeHiddenBuff,
		Eval:        constants.EvaluationGood,
		EnglishName: "HiddenBuff",
		Name:        "麻了",
		Desc:        "身体麻木，获得隐匿Buff",
	}, &EventHandlerConfig{
		Handler: createGiveBuffHandler(constants.BuffTypeHidden, 3),
	})

	// TasteTest: Random buff effect
	GlobalEventRegistry.RegisterEvent(&EventDefinition{
		Type:        constants.EventTypeTasteTest,
		Eval:        constants.EvaluationMixed,
		EnglishName: "TasteTest",
		Name:        "这是什么？尝一口",
		Desc:        "发现神秘物质，尝试后获得随机效果",
	}, &EventHandlerConfig{
		Handler: handleRandomBuff,
	})

	// ========== Bad Events (Evaluation ≤ 40) ==========

	// Mosquito: HP-1
	GlobalEventRegistry.RegisterEvent(&EventDefinition{
		Type:        constants.EventTypeMosquito,
		Eval:        constants.EvaluationMildBad,
		EnglishName: "Mosquito",
		Name:        "被蚊虫叮咬",
		Desc:        "丛林中的蚊虫叮咬了你",
	}, &EventHandlerConfig{
		Handler: createModifyHPHandler(-1),
	})

	// GhostHit: HP-1
	GlobalEventRegistry.RegisterEvent(&EventDefinition{
		Type:        constants.EventTypeGhostHit,
		Eval:        constants.EvaluationMildBad,
		EnglishName: "GhostHit",
		Name:        "偶遇孤魂野鬼",
		Desc:        "被野鬼打了一闷棍",
	}, &EventHandlerConfig{
		Handler: createModifyHPHandler(-1),
	})

	// DogPoop: LP-1
	GlobalEventRegistry.RegisterEvent(&EventDefinition{
		Type:        constants.EventTypeDogPoop,
		Eval:        constants.EvaluationMildBad,
		EnglishName: "DogPoop",
		Name:        "踩到了狗屎",
		Desc:        "运气糟糕的一天",
	}, &EventHandlerConfig{
		Handler: createModifyLPHandler(-1),
	})

	// Thief: Lose random item
	GlobalEventRegistry.RegisterEvent(&EventDefinition{
		Type:        constants.EventTypeThief,
		Eval:        constants.EvaluationBad,
		EnglishName: "Thief",
		Name:        "啊？！贼",
		Desc:        "遭遇盗贼，随机丢失一个道具",
	}, &EventHandlerConfig{
		Handler: handleLoseItem,
	})

	// CurseBuddha: Give Curse buff
	GlobalEventRegistry.RegisterEvent(&EventDefinition{
		Type:        constants.EventTypeCurseBuddha,
		Eval:        constants.EvaluationBad,
		EnglishName: "CurseBuddha",
		Name:        "虔诚拜三拜",
		Desc:        "拜路边的野佛，获得诅咒Buff",
	}, &EventHandlerConfig{
		Handler: createGiveBuffHandler(constants.BuffTypeCurse, 3),
	})

	// LostWay: Give Lost buff
	GlobalEventRegistry.RegisterEvent(&EventDefinition{
		Type:        constants.EventTypeLostWay,
		Eval:        constants.EvaluationMildBad,
		EnglishName: "LostWay",
		Name:        "迷途",
		Desc:        "迷失方向，获得迷途Buff",
	}, &EventHandlerConfig{
		Handler: createGiveBuffHandler(constants.BuffTypeLost, 1),
	})

	// Thunder: HP to 0 (death)
	GlobalEventRegistry.RegisterEvent(&EventDefinition{
		Type:        constants.EventTypeThunder,
		Eval:        constants.EvaluationVeryBad,
		EnglishName: "Thunder",
		Name:        "雷劫",
		Desc:        "天雷降临，HP归零",
	}, &EventHandlerConfig{
		Handler: handleThunderDeath,
	})
}

// ========== Handler Helper Functions ==========

// createModifyHPHandler creates a handler that signals HP modification intent.
func createModifyHPHandler(amount int) EffectHandler {
	return func(phase constants.Phase, ctx *event.Context) {
		// Signal HP modification intent (actual Action in engine)
		ctx.SetInt("hp_change", amount)
	}
}

// createModifyLPHandler creates a handler that modifies LP directly.
func createModifyLPHandler(amount int) EffectHandler {
	return func(phase constants.Phase, ctx *event.Context) {
		player, ok := ctx.Player.(protocol.PlayerLite)
		if !ok {
			return
		}
		player.ModifyLP(amount)
	}
}

// createGiveBuffHandler creates a handler that signals buff addition intent.
func createGiveBuffHandler(buffType constants.BuffType, duration int) EffectHandler {
	return func(phase constants.Phase, ctx *event.Context) {
		// Signal buff addition intent (actual Action in engine)
		ctx.SetString("give_buff_type", string(buffType))
		ctx.SetInt("give_buff_duration", duration)
	}
}

// ========== Custom Event Handlers ==========

// handleDrawItem triggers an item draw.
func handleDrawItem(phase constants.Phase, ctx *event.Context) {
	ctx.SetBool("draw_item", true)
}

// handleSwapPosition swaps position with another player.
func handleSwapPosition(phase constants.Phase, ctx *event.Context) {
	ctx.SetBool("swap_position", true)
}

// handleRandomBuff applies a random buff effect.
func handleRandomBuff(phase constants.Phase, ctx *event.Context) {
	ctx.SetBool("random_buff", true)
}

// handleLoseItem loses a random item from inventory.
func handleLoseItem(phase constants.Phase, ctx *event.Context) {
	ctx.SetBool("lose_item", true)
}

// handleThunderDeath signals instant death intent.
func handleThunderDeath(phase constants.Phase, ctx *event.Context) {
	// Signal death intent
	ctx.SetBool("instant_death", true)
}