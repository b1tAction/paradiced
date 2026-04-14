package buff

import (
	"github.com/b1tAction/Fated/pkg/constants"
	"github.com/b1tAction/Fated/pkg/event"
	"github.com/b1tAction/Fated/pkg/protocol"
)

// GlobalBuffRegistry is the global Buff definition registry.
// Initialized at package load time with all Buff definitions.
var GlobalBuffRegistry *BuffRegistry

// init initializes the global Buff registry and registers all Buff definitions.
func init() {
	GlobalBuffRegistry = NewBuffRegistry()
	registerAllBuffs()
}

// registerAllBuffs registers all Buff definitions with their handlers.
func registerAllBuffs() {
	// ========== Negative Buffs ==========

	GlobalBuffRegistry.RegisterBuff(&BuffDefinition{
		Type:        constants.BuffTypeCurse,
		Eval:        constants.EvaluationBad,
		EnglishName: "Curse",
		Name:        "诅咒",
		Desc:        "接下来3回合LP-1",
		Duration:    3,
		LPPerTurn:   -1,
		Phases:      []constants.Phase{constants.PhaseBeforeTurn},
		Priority:    50,
	}, nil)

	GlobalBuffRegistry.RegisterBuff(&BuffDefinition{
		Type:          constants.BuffTypeLost,
		Eval:          constants.EvaluationMildBad,
		EnglishName:   "Lost",
		Name:          "迷途",
		Desc:          "下1回合朝反方向移动",
		Duration:      1,
		SpecialEffect: constants.SpecialReverse,
		Phases:        []constants.Phase{constants.PhasePreMove},
		Priority:      100,
	}, nil)

	GlobalBuffRegistry.RegisterBuff(&BuffDefinition{
		Type:        constants.BuffTypeCorrupt,
		Eval:        constants.EvaluationBad,
		EnglishName: "Corrupt",
		Name:        "腐化",
		Desc:        "接下来4回合每2回合HP-1",
		Duration:    4,
		HPPerTurn:   -1,
		Phases:      []constants.Phase{constants.PhaseAfterTurn},
		Priority:    50,
	}, nil)

	GlobalBuffRegistry.RegisterBuff(&BuffDefinition{
		Type:          constants.BuffTypePoison,
		Eval:          constants.EvaluationVeryBad,
		EnglishName:   "Poison",
		Name:          "毒瘴",
		Desc:          "接下来3回合每回合受一次恶性随机事件影响",
		Duration:      3,
		SpecialEffect: constants.SpecialBadEvent,
		Phases:        []constants.Phase{constants.PhaseBeforeTurn},
		Priority:      30,
	}, nil)

	// ========== Neutral Buff ==========

	GlobalBuffRegistry.RegisterBuff(&BuffDefinition{
		Type:          constants.BuffTypeHidden,
		Eval:          constants.EvaluationNeutral,
		EnglishName:   "Hidden",
		Name:          "隐匿",
		Desc:          "接下来3回合免疫任意事件、BUFF或道具的影响",
		Duration:      3,
		SpecialEffect: constants.SpecialImmune,
		Phases:        []constants.Phase{constants.PhasePreDamage},
		Priority:      100,
	}, nil)

	// ========== Positive Buffs ==========

	GlobalBuffRegistry.RegisterBuff(&BuffDefinition{
		Type:        constants.BuffTypeDivine,
		Eval:        constants.EvaluationVeryGood,
		EnglishName: "Divine",
		Name:        "神眷",
		Desc:        "接下来3回合LP+1",
		Duration:    3,
		LPPerTurn:   1,
		Phases:      []constants.Phase{constants.PhaseBeforeTurn},
		Priority:    50,
	}, nil)

	GlobalBuffRegistry.RegisterBuff(&BuffDefinition{
		Type:        constants.BuffTypeRain,
		Eval:        constants.EvaluationGood,
		EnglishName: "Rain",
		Name:        "甘霖",
		Desc:        "接下来4回合每2回合HP+1",
		Duration:    4,
		HPPerTurn:   1,
		Phases:      []constants.Phase{constants.PhaseAfterTurn},
		Priority:    50,
	}, nil)

	GlobalBuffRegistry.RegisterBuff(&BuffDefinition{
		Type:          constants.BuffTypeExorcism,
		Eval:          constants.EvaluationMildGood,
		EnglishName:   "Exorcism",
		Name:          "辟邪",
		Desc:          "接下来5回合无视毒瘴buff",
		Duration:      5,
		SpecialEffect: constants.SpecialImmunePoison,
		Phases:        []constants.Phase{constants.PhasePreEvent},
		Priority:      80,
	}, nil)

	GlobalBuffRegistry.RegisterBuff(&BuffDefinition{
		Type:          constants.BuffTypeFire,
		Eval:          constants.EvaluationGood,
		EnglishName:   "Fire",
		Name:          "离火",
		Desc:          "朱雀阵营增益，每4回合LP+1",
		Duration:      -1,
		SpecialEffect: constants.SpecialZhuQuePassive,
		Phases:        []constants.Phase{constants.PhaseBeforeTurn},
		Priority:      10,
	}, handleZhuQueFire)
}

// ========== Custom Buff Handlers ==========

// handleZhuQueFire is the custom handler for ZhuQue Fire buff.
// Effect: LP+1 every 4 turns.
// Uses direct Player interface modification.
func handleZhuQueFire(phase constants.Phase, ctx *event.Context) {
	// Get Player interface from Context
	player, ok := ctx.Player.(protocol.PlayerLite)
	if !ok {
		return
	}

	// Only execute in BeforeTurn Phase
	if phase != constants.PhaseBeforeTurn {
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