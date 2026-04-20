package engine

import (
	"github.com/b1tAction/paradiced/internal/core"
	engineaction "github.com/b1tAction/paradiced/internal/engine/action"
	"github.com/b1tAction/paradiced/internal/event"
	"github.com/b1tAction/paradiced/pkg/constants"
)

// EffectHandler defines the handler signature for Buff/Item/Event effects.
// Defined in engine layer to maintain correct dependency direction.
type EffectHandler func(phase constants.Phase, ctx *event.Context)

// BuffHandlerConfig contains effect logic and execution configuration.
type BuffHandlerConfig struct {
	Phases      []constants.Phase `json:"phases"`
	Priority    int               `json:"priority"`
	NeedConfirm bool              `json:"need_confirm"`
	Handler     EffectHandler     `json:"-"`
}

// GetPhases returns the Buff's trigger phase list.
func (c *BuffHandlerConfig) GetPhases() []constants.Phase {
	return c.Phases
}

// HasPhase checks if the Buff triggers at the specified Phase.
func (c *BuffHandlerConfig) HasPhase(phase constants.Phase) bool {
	for _, p := range c.Phases {
		if p == phase {
			return true
		}
	}
	return false
}

// BuffRegistry is the registry for Buff definitions and handler configs.
type BuffRegistry struct {
	defs    map[constants.BuffType]*core.BuffDefinition
	configs map[constants.BuffType]*BuffHandlerConfig
	names   map[constants.BuffType]string

	goodBuffs    []constants.BuffType
	badBuffs     []constants.BuffType
	neutralBuffs []constants.BuffType
}

// GlobalBuffRegistry is the global Buff registry.
var GlobalBuffRegistry *BuffRegistry

func init() {
	GlobalBuffRegistry = NewBuffRegistry()
	registerAllBuffs()
}

// NewBuffRegistry creates a new Buff registry.
func NewBuffRegistry() *BuffRegistry {
	return &BuffRegistry{
		defs:         make(map[constants.BuffType]*core.BuffDefinition),
		configs:      make(map[constants.BuffType]*BuffHandlerConfig),
		names:        make(map[constants.BuffType]string),
		goodBuffs:    make([]constants.BuffType, 0),
		badBuffs:     make([]constants.BuffType, 0),
		neutralBuffs: make([]constants.BuffType, 0),
	}
}

// RegisterBuff registers a Buff definition with handler config.
func (r *BuffRegistry) RegisterBuff(def *core.BuffDefinition, config *BuffHandlerConfig) {
	if def == nil || def.Type == constants.BuffTypeNone {
		return
	}

	r.defs[def.Type] = def
	r.names[def.Type] = def.Name

	if def.Eval.IsGood() {
		r.goodBuffs = append(r.goodBuffs, def.Type)
	} else if def.Eval.IsBad() {
		r.badBuffs = append(r.badBuffs, def.Type)
	} else {
		r.neutralBuffs = append(r.neutralBuffs, def.Type)
	}

	if config != nil {
		r.configs[def.Type] = config
	}
}

// GetBuffDefinition returns the Buff definition by type.
func GetBuffDefinition(bt constants.BuffType) *core.BuffDefinition {
	return GlobalBuffRegistry.GetBuffDefinition(bt)
}

func (r *BuffRegistry) GetBuffDefinition(bt constants.BuffType) *core.BuffDefinition {
	if def, ok := r.defs[bt]; ok {
		return def
	}
	return nil
}

// GetBuffHandlerConfig returns the Buff's handler config.
func GetBuffHandlerConfig(bt constants.BuffType) *BuffHandlerConfig {
	return GlobalBuffRegistry.GetBuffHandlerConfig(bt)
}

func (r *BuffRegistry) GetBuffHandlerConfig(bt constants.BuffType) *BuffHandlerConfig {
	if config, ok := r.configs[bt]; ok {
		return config
	}
	return nil
}

// GetBuffName returns the Buff Chinese display name.
func GetBuffName(bt constants.BuffType) string {
	return GlobalBuffRegistry.GetBuffName(bt)
}

func (r *BuffRegistry) GetBuffName(bt constants.BuffType) string {
	if name, ok := r.names[bt]; ok {
		return name
	}
	return ""
}

// GetAllBuffTypes returns all registered Buff types.
func GetAllBuffTypes() []constants.BuffType {
	return GlobalBuffRegistry.GetAllBuffTypes()
}

func (r *BuffRegistry) GetAllBuffTypes() []constants.BuffType {
	result := make([]constants.BuffType, 0, len(r.defs))
	for bt := range r.defs {
		result = append(result, bt)
	}
	return result
}

// HasBuffHandler checks if a Buff type has a registered handler.
func HasBuffHandler(bt constants.BuffType) bool {
	return GlobalBuffRegistry.HasBuffHandler(bt)
}

func (r *BuffRegistry) HasBuffHandler(bt constants.BuffType) bool {
	config, ok := r.configs[bt]
	return ok && config != nil && config.Handler != nil
}

// GetBuffTypesByCategory returns Buff types by category.
func GetBuffTypesByCategory(category string) []constants.BuffType {
	return GlobalBuffRegistry.GetBuffTypesByCategory(category)
}

func (r *BuffRegistry) GetBuffTypesByCategory(category string) []constants.BuffType {
	switch category {
	case "Good":
		return r.goodBuffs
	case "Bad":
		return r.badBuffs
	case "Neutral":
		return r.neutralBuffs
	}
	return r.GetAllBuffTypes()
}

// ========== Buff Handler Registration ==========

func registerAllBuffs() {
	// Curse: LP-1 per turn for 3 turns
	GlobalBuffRegistry.RegisterBuff(&core.BuffDefinition{
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
	GlobalBuffRegistry.RegisterBuff(&core.BuffDefinition{
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
	GlobalBuffRegistry.RegisterBuff(&core.BuffDefinition{
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
	GlobalBuffRegistry.RegisterBuff(&core.BuffDefinition{
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

	// Hidden: Immunity to damage/events for 3 turns
	GlobalBuffRegistry.RegisterBuff(&core.BuffDefinition{
		Type:        constants.BuffTypeHidden,
		Eval:        constants.EvaluationNeutral,
		EnglishName: "Hidden",
		Name:        "隐匿",
		Desc:        "接下来3回合免疫任意事件、BUFF或道具的影响",
		Duration:    3,
	}, &BuffHandlerConfig{
		Phases:      []constants.Phase{constants.PhasePreBuffApplied},
		Priority:    100,
		NeedConfirm: false,
		Handler:     handleHiddenImmune,
	})

	// Divine: LP+1 per turn for 3 turns
	GlobalBuffRegistry.RegisterBuff(&core.BuffDefinition{
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
	GlobalBuffRegistry.RegisterBuff(&core.BuffDefinition{
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
	GlobalBuffRegistry.RegisterBuff(&core.BuffDefinition{
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
	GlobalBuffRegistry.RegisterBuff(&core.BuffDefinition{
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

// ========== Buff Handler Helpers ==========

func createModifyLPHandler(amount int) EffectHandler {
	return func(phase constants.Phase, ctx *event.Context) {
		if ctx == nil || ctx.Player == nil {
			return
		}

		// Must use Action system - no direct modification
		actionCtx := getActionCtxFromEventCtx(ctx)
		if actionCtx == nil {
			return
		}

		ctx.AddDerivedAction(engineaction.NewModifyLPAction(ctx.Player, amount, "Buff_Effect"))
	}
}

func createModifyHPHandler(amount int) EffectHandler {
	return func(phase constants.Phase, ctx *event.Context) {
		if ctx == nil || ctx.Player == nil || amount == 0 {
			return
		}

		// Must use Action system - no direct modification
		actionCtx := getActionCtxFromEventCtx(ctx)
		if actionCtx == nil {
			return
		}

		if amount > 0 {
			ctx.AddDerivedAction(engineaction.NewHealAction(ctx.Player, amount, "Buff_Effect"))
		} else {
			ctx.AddDerivedAction(engineaction.NewDamageAction(ctx.Player, -amount, "Buff_Effect"))
		}
	}
}

func createEveryNTurnsHandler(everyN int, innerHandler EffectHandler) EffectHandler {
	return func(phase constants.Phase, ctx *event.Context) {
		counter, _ := ctx.GetInt("buff_turn_counter")
		counter++
		ctx.SetInt("buff_turn_counter", counter)

		if counter >= everyN {
			innerHandler(phase, ctx)
			ctx.SetInt("buff_turn_counter", 0)
		}
	}
}

// ========== Custom Buff Handlers ==========

func handleZhuQueFire(phase constants.Phase, ctx *event.Context) {
	if phase != constants.PhaseBeforeTurn {
		return
	}
	if ctx == nil || ctx.Player == nil {
		return
	}

	newCount := ctx.Player.IncrementFireCounter()
	if newCount >= 4 {
		// Must use Action system - no direct modification
		actionCtx := getActionCtxFromEventCtx(ctx)
		if actionCtx == nil {
			return
		}

		ctx.AddDerivedAction(engineaction.NewModifyLPAction(ctx.Player, 1, "Buff_Fire"))
		ctx.Player.SetFireCounter(0)
	}
}

func handleLostReverse(phase constants.Phase, ctx *event.Context) {
	if phase != constants.PhasePreMove {
		return
	}
	ctx.SetBool("reverse_movement", true)

	raw, ok := ctx.Get("current_action")
	if !ok {
		return
	}

	moveAction, ok := raw.(*engineaction.MoveAction)
	if !ok || moveAction == nil {
		return
	}

	moveAction.Steps = -moveAction.Steps
}

func handleHiddenImmune(phase constants.Phase, ctx *event.Context) {
	if phase != constants.PhasePreBuffApplied {
		return
	}
	ctx.SetBool("action_blocked", true)
	ctx.SetString("blocked_by", "Buff_Hidden")
}

func handlePoisonBadEvent(phase constants.Phase, ctx *event.Context) {
	if phase != constants.PhaseBeforeTurn {
		return
	}
	ctx.SetBool("draw_bad_event", true)
}

func handleExorcismImmunePoison(phase constants.Phase, ctx *event.Context) {
	if phase != constants.PhasePreEvent {
		return
	}
	ctx.SetBool("block_poison_effect", true)
}

func getActionCtxFromEventCtx(ctx *event.Context) *engineaction.ActionContext {
	if ctx == nil {
		return nil
	}

	raw, ok := ctx.Get("action_context")
	if !ok {
		return nil
	}

	actionCtx, _ := raw.(*engineaction.ActionContext)
	return actionCtx
}
