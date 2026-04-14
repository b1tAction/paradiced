package buff

import (
	"github.com/b1tAction/paradiced/pkg/constants"
	"github.com/b1tAction/paradiced/pkg/event"
	"github.com/b1tAction/paradiced/pkg/protocol"
)

// GlobalBuffRegistry is the global Buff definition registry.
// Initialized at package load time with all Buff definitions.
var GlobalBuffRegistry *BuffRegistry

// init initializes the global Buff registry and registers all Buff definitions.
func init() {
	GlobalBuffRegistry = NewBuffRegistry()
	registerAllBuffs()
}

// registerAllBuffs registers all Buff definitions with their handler configs.
func registerAllBuffs() {
	// ========== Negative Buffs ==========

	// Curse: LP-1 per turn for 3 turns
	GlobalBuffRegistry.RegisterBuff(&BuffDefinition{
		Type:        constants.BuffTypeCurse,
		Eval:        constants.EvaluationBad,
		EnglishName: "Curse",
		Name:        "诅咒",
		Desc:        "接下来3回合LP-1",
		Duration:    3,
	}, &BuffHandlerConfig{
		Phases:      []constants.Phase{constants.PhaseBeforeTurn},
		Priority:    50,
		NeedConfirm: false,
		Handler:     createModifyLPHandler(-1),
	})

	// Lost: Reverse movement direction for 1 turn
	GlobalBuffRegistry.RegisterBuff(&BuffDefinition{
		Type:        constants.BuffTypeLost,
		Eval:        constants.EvaluationMildBad,
		EnglishName: "Lost",
		Name:        "迷途",
		Desc:        "下1回合朝反方向移动",
		Duration:    1,
	}, &BuffHandlerConfig{
		Phases:      []constants.Phase{constants.PhasePreMove},
		Priority:    100,
		NeedConfirm: false,
		Handler:     handleLostReverse,
	})

	// Corrupt: HP-1 every 2 turns for 4 turns
	GlobalBuffRegistry.RegisterBuff(&BuffDefinition{
		Type:        constants.BuffTypeCorrupt,
		Eval:        constants.EvaluationBad,
		EnglishName: "Corrupt",
		Name:        "腐化",
		Desc:        "接下来4回合每2回合HP-1",
		Duration:    4,
	}, &BuffHandlerConfig{
		Phases:      []constants.Phase{constants.PhaseAfterTurn},
		Priority:    50,
		NeedConfirm: false,
		Handler:     createEveryNTurnsHandler(2, createModifyHPHandler(-1)),
	})

	// Poison: Bad event each turn for 3 turns
	GlobalBuffRegistry.RegisterBuff(&BuffDefinition{
		Type:        constants.BuffTypePoison,
		Eval:        constants.EvaluationVeryBad,
		EnglishName: "Poison",
		Name:        "毒瘴",
		Desc:        "接下来3回合每回合受一次恶性随机事件影响",
		Duration:    3,
	}, &BuffHandlerConfig{
		Phases:      []constants.Phase{constants.PhaseBeforeTurn},
		Priority:    30,
		NeedConfirm: false,
		Handler:     handlePoisonBadEvent,
	})

	// ========== Neutral Buff ==========

	// Hidden: Immunity to damage/events for 3 turns
	GlobalBuffRegistry.RegisterBuff(&BuffDefinition{
		Type:        constants.BuffTypeHidden,
		Eval:        constants.EvaluationNeutral,
		EnglishName: "Hidden",
		Name:        "隐匿",
		Desc:        "接下来3回合免疫任意事件、BUFF或道具的影响",
		Duration:    3,
	}, &BuffHandlerConfig{
		Phases:      []constants.Phase{constants.PhasePreDamage},
		Priority:    100,
		NeedConfirm: false,
		Handler:     handleHiddenImmune,
	})

	// ========== Positive Buffs ==========

	// Divine: LP+1 per turn for 3 turns
	GlobalBuffRegistry.RegisterBuff(&BuffDefinition{
		Type:        constants.BuffTypeDivine,
		Eval:        constants.EvaluationVeryGood,
		EnglishName: "Divine",
		Name:        "神眷",
		Desc:        "接下来3回合LP+1",
		Duration:    3,
	}, &BuffHandlerConfig{
		Phases:      []constants.Phase{constants.PhaseBeforeTurn},
		Priority:    50,
		NeedConfirm: false,
		Handler:     createModifyLPHandler(1),
	})

	// Rain: HP+1 every 2 turns for 4 turns
	GlobalBuffRegistry.RegisterBuff(&BuffDefinition{
		Type:        constants.BuffTypeRain,
		Eval:        constants.EvaluationGood,
		EnglishName: "Rain",
		Name:        "甘霖",
		Desc:        "接下来4回合每2回合HP+1",
		Duration:    4,
	}, &BuffHandlerConfig{
		Phases:      []constants.Phase{constants.PhaseAfterTurn},
		Priority:    50,
		NeedConfirm: false,
		Handler:     createEveryNTurnsHandler(2, createModifyHPHandler(1)),
	})

	// Exorcism: Immune to poison buff for 5 turns
	GlobalBuffRegistry.RegisterBuff(&BuffDefinition{
		Type:        constants.BuffTypeExorcism,
		Eval:        constants.EvaluationMildGood,
		EnglishName: "Exorcism",
		Name:        "辟邪",
		Desc:        "接下来5回合无视毒瘴buff",
		Duration:    5,
	}, &BuffHandlerConfig{
		Phases:      []constants.Phase{constants.PhasePreEvent},
		Priority:    80,
		NeedConfirm: false,
		Handler:     handleExorcismImmunePoison,
	})

	// Fire: ZhuQue passive, LP+1 every 4 turns (permanent)
	GlobalBuffRegistry.RegisterBuff(&BuffDefinition{
		Type:        constants.BuffTypeFire,
		Eval:        constants.EvaluationGood,
		EnglishName: "Fire",
		Name:        "离火",
		Desc:        "朱雀阵营增益，每4回合LP+1",
		Duration:    -1,
	}, &BuffHandlerConfig{
		Phases:      []constants.Phase{constants.PhaseBeforeTurn},
		Priority:    10,
		NeedConfirm: false,
		Handler:     handleZhuQueFire,
	})
}

// ========== Handler Helper Functions ==========

// createModifyHPHandler creates a handler that signals HP modification intent.
// Actual HP change will be executed by engine via Action system.
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
		// Direct LP modification (no Action needed)
		player.ModifyLP(amount)
	}
}

// createEveryNTurnsHandler wraps a handler to execute only every N turns.
// Uses context to track turn count.
func createEveryNTurnsHandler(everyN int, innerHandler EffectHandler) EffectHandler {
	return func(phase constants.Phase, ctx *event.Context) {
		// Get turn counter from context
		counter, _ := ctx.GetInt("buff_turn_counter")
		counter++
		ctx.SetInt("buff_turn_counter", counter)

		// Execute inner handler only when counter reaches N
		if counter >= everyN {
			innerHandler(phase, ctx)
			ctx.SetInt("buff_turn_counter", 0) // Reset counter
		}
	}
}

// ========== Custom Buff Handlers ==========

// handleZhuQueFire is the custom handler for ZhuQue Fire buff.
// Effect: LP+1 every 4 turns.
func handleZhuQueFire(phase constants.Phase, ctx *event.Context) {
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

// handleLostReverse reverses movement direction.
func handleLostReverse(phase constants.Phase, ctx *event.Context) {
	if phase != constants.PhasePreMove {
		return
	}
	// Signal reverse movement intent
	ctx.SetBool("reverse_movement", true)
}

// handleHiddenImmune blocks damage and events.
func handleHiddenImmune(phase constants.Phase, ctx *event.Context) {
	if phase != constants.PhasePreDamage {
		return
	}
	// Block the damage action
	ctx.SetBool("action_blocked", true)
	ctx.SetString("blocked_by", "Buff_Hidden")
}

// handlePoisonBadEvent triggers a bad event each turn.
func handlePoisonBadEvent(phase constants.Phase, ctx *event.Context) {
	if phase != constants.PhaseBeforeTurn {
		return
	}
	// Signal that a bad event should be drawn
	ctx.SetBool("draw_bad_event", true)
}

// handleExorcismImmunePoison cancels poison buff effect.
func handleExorcismImmunePoison(phase constants.Phase, ctx *event.Context) {
	if phase != constants.PhasePreEvent {
		return
	}
	// Block poison-related events
	ctx.SetBool("block_poison_effect", true)
}