package buff

import (
	"github.com/b1tAction/Fated/internal/core/types"
	"github.com/b1tAction/Fated/pkg/event"
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
		Type:        BuffTypeCurse,
		Eval:        types.EvaluationBad,
		EnglishName: "Curse",
		Name:        "诅咒",
		Desc:        "接下来3回合LP-1",
		Duration:    3,
		LPPerTurn:   -1,
		Phases:      []event.Phase{event.PhaseBeforeTurn},
		Priority:    50,
	}, nil)

	GlobalBuffRegistry.RegisterBuff(&BuffDefinition{
		Type:          BuffTypeLost,
		Eval:          types.EvaluationMildBad,
		EnglishName:   "Lost",
		Name:          "迷途",
		Desc:          "下1回合朝反方向移动",
		Duration:      1,
		SpecialEffect: types.SpecialReverse,
		Phases:        []event.Phase{event.PhaseOnMove},
		Priority:      100,
	}, nil)

	GlobalBuffRegistry.RegisterBuff(&BuffDefinition{
		Type:        BuffTypeCorrupt,
		Eval:        types.EvaluationBad,
		EnglishName: "Corrupt",
		Name:        "腐化",
		Desc:        "接下来4回合每2回合HP-1",
		Duration:    4,
		HPPerTurn:   -1,
		Phases:      []event.Phase{event.PhaseAfterTurn},
		Priority:    50,
	}, nil)

	GlobalBuffRegistry.RegisterBuff(&BuffDefinition{
		Type:          BuffTypePoison,
		Eval:          types.EvaluationVeryBad,
		EnglishName:   "Poison",
		Name:          "毒瘴",
		Desc:          "接下来3回合每回合受一次恶性随机事件影响",
		Duration:      3,
		SpecialEffect: types.SpecialBadEvent,
		Phases:        []event.Phase{event.PhaseBeforeTurn},
		Priority:      30,
	}, nil)

	// ========== Neutral Buff ==========

	GlobalBuffRegistry.RegisterBuff(&BuffDefinition{
		Type:          BuffTypeHidden,
		Eval:          types.EvaluationNeutral,
		EnglishName:   "Hidden",
		Name:          "隐匿",
		Desc:          "接下来3回合免疫任意事件、BUFF或道具的影响",
		Duration:      3,
		SpecialEffect: types.SpecialImmune,
		Phases:        []event.Phase{event.PhasePreDamage},
		Priority:      100,
	}, nil)

	// ========== Positive Buffs ==========

	GlobalBuffRegistry.RegisterBuff(&BuffDefinition{
		Type:        BuffTypeDivine,
		Eval:        types.EvaluationVeryGood,
		EnglishName: "Divine",
		Name:        "神眷",
		Desc:        "接下来3回合LP+1",
		Duration:    3,
		LPPerTurn:   1,
		Phases:      []event.Phase{event.PhaseBeforeTurn},
		Priority:    50,
	}, nil)

	GlobalBuffRegistry.RegisterBuff(&BuffDefinition{
		Type:        BuffTypeRain,
		Eval:        types.EvaluationGood,
		EnglishName: "Rain",
		Name:        "甘霖",
		Desc:        "接下来4回合每2回合HP+1",
		Duration:    4,
		HPPerTurn:   1,
		Phases:      []event.Phase{event.PhaseAfterTurn},
		Priority:    50,
	}, nil)

	GlobalBuffRegistry.RegisterBuff(&BuffDefinition{
		Type:          BuffTypeExorcism,
		Eval:          types.EvaluationMildGood,
		EnglishName:   "Exorcism",
		Name:          "辟邪",
		Desc:          "接下来5回合无视毒瘴buff",
		Duration:      5,
		SpecialEffect: types.SpecialImmunePoison,
		Phases:        []event.Phase{event.PhasePreEvent},
		Priority:      80,
	}, nil)

	GlobalBuffRegistry.RegisterBuff(&BuffDefinition{
		Type:          BuffTypeFire,
		Eval:          types.EvaluationGood,
		EnglishName:   "Fire",
		Name:          "离火",
		Desc:          "朱雀阵营增益，每4回合LP+1",
		Duration:      -1,
		SpecialEffect: types.SpecialZhuQuePassive,
		Phases:        []event.Phase{event.PhaseBeforeTurn},
		Priority:      10,
	}, handleZhuQueFire)
}

// ========== Custom Buff Handlers ==========

// handleZhuQueFire is the custom handler for ZhuQue Fire buff.
// Effect: LP+1 every 4 turns.
func handleZhuQueFire(phase event.Phase, ctx *event.Context) {
	// Get Player interface from Context
	player, ok := ctx.Player.(Player)
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

// Player interface defines the methods needed for Buff handlers.
// This allows Buff handlers to work with Player type from core package.
type Player interface {
	IncrementFireCounter() int
	SetFireCounter(count int)
	ModifyLP(amount int)
}