package core

import (
	"github.com/b1tAction/Fated/pkg/event"
)

// GlobalRegistry is the global definition registry.
// Initialized at package load time with all game definitions.
var GlobalRegistry *DefinitionRegistry

// init initializes the global registry and registers all definitions.
func init() {
	GlobalRegistry = NewDefinitionRegistry()

	// Register all Buffs
	registerAllBuffs()

	// Register all Events
	registerAllEvents()

	// Register all Items
	registerAllItems()
}

// registerAllBuffs registers all Buff definitions with their handlers.
func registerAllBuffs() {
	// ========== Negative Buffs ==========

	GlobalRegistry.RegisterBuff(&BuffDefinition{
		Type:        BuffTypeCurse,
		Eval:        EvaluationBad,
		EnglishName: "Curse",
		Name:        "诅咒",
		Desc:        "接下来3回合LP-1",
		Duration:    3,
		LPPerTurn:   -1,
		Phases:      []event.Phase{event.PhaseBeforeTurn},
		Priority:    50,
	}, nil)

	GlobalRegistry.RegisterBuff(&BuffDefinition{
		Type:          BuffTypeLost,
		Eval:          EvaluationMildBad,
		EnglishName:   "Lost",
		Name:          "迷途",
		Desc:          "下1回合朝反方向移动",
		Duration:      1,
		SpecialEffect: SpecialReverse,
		Phases:        []event.Phase{event.PhaseOnMove},
		Priority:      100,
	}, nil)

	GlobalRegistry.RegisterBuff(&BuffDefinition{
		Type:        BuffTypeCorrupt,
		Eval:        EvaluationBad,
		EnglishName: "Corrupt",
		Name:        "腐化",
		Desc:        "接下来4回合每2回合HP-1",
		Duration:    4,
		HPPerTurn:   -1,
		Phases:      []event.Phase{event.PhaseAfterTurn},
		Priority:    50,
	}, nil)

	GlobalRegistry.RegisterBuff(&BuffDefinition{
		Type:          BuffTypePoison,
		Eval:          EvaluationVeryBad,
		EnglishName:   "Poison",
		Name:          "毒瘴",
		Desc:          "接下来3回合每回合受一次恶性随机事件影响",
		Duration:      3,
		SpecialEffect: SpecialBadEvent,
		Phases:        []event.Phase{event.PhaseBeforeTurn},
		Priority:      30,
	}, nil)

	// ========== Neutral Buff ==========

	GlobalRegistry.RegisterBuff(&BuffDefinition{
		Type:          BuffTypeHidden,
		Eval:          EvaluationNeutral,
		EnglishName:   "Hidden",
		Name:          "隐匿",
		Desc:          "接下来3回合免疫任意事件、BUFF或道具的影响",
		Duration:      3,
		SpecialEffect: SpecialImmune,
		Phases:        []event.Phase{event.PhasePreDamage},
		Priority:      100,
	}, nil)

	// ========== Positive Buffs ==========

	GlobalRegistry.RegisterBuff(&BuffDefinition{
		Type:        BuffTypeDivine,
		Eval:        EvaluationVeryGood,
		EnglishName: "Divine",
		Name:        "神眷",
		Desc:        "接下来3回合LP+1",
		Duration:    3,
		LPPerTurn:   1,
		Phases:      []event.Phase{event.PhaseBeforeTurn},
		Priority:    50,
	}, nil)

	GlobalRegistry.RegisterBuff(&BuffDefinition{
		Type:        BuffTypeRain,
		Eval:        EvaluationGood,
		EnglishName: "Rain",
		Name:        "甘霖",
		Desc:        "接下来4回合每2回合HP+1",
		Duration:    4,
		HPPerTurn:   1,
		Phases:      []event.Phase{event.PhaseAfterTurn},
		Priority:    50,
	}, nil)

	GlobalRegistry.RegisterBuff(&BuffDefinition{
		Type:          BuffTypeExorcism,
		Eval:          EvaluationMildGood,
		EnglishName:   "Exorcism",
		Name:          "辟邪",
		Desc:          "接下来5回合无视毒瘴buff",
		Duration:      5,
		SpecialEffect: SpecialImmunePoison,
		Phases:        []event.Phase{event.PhasePreEvent},
		Priority:      80,
	}, nil)

	GlobalRegistry.RegisterBuff(&BuffDefinition{
		Type:          BuffTypeFire,
		Eval:          EvaluationGood,
		EnglishName:   "Fire",
		Name:          "离火",
		Desc:          "朱雀阵营增益，每4回合LP+1",
		Duration:      -1,
		SpecialEffect: SpecialZhuQuePassive,
		Phases:        []event.Phase{event.PhaseBeforeTurn},
		Priority:      10,
	}, handleZhuQueFire)
}

// ========== Custom Buff Handlers ==========

// handleZhuQueFire is the custom handler for ZhuQue Fire Buff.
// Effect: LP+1 every 4 turns.
func handleZhuQueFire(phase event.Phase, ctx *event.Context) {
	player, ok := ctx.Player.(*Player)
	if !ok {
		return
	}

	// Only execute in BeforeTurn Phase
	if phase != event.PhaseBeforeTurn {
		return
	}

	// Increment Fire counter (using Metadata method)
	newCount := player.IncrementFireCounter()

	// Add 1 LP every 4 turns
	if newCount >= 4 {
		player.ModifyLP(1)
		player.SetFireCounter(0)
	}
}