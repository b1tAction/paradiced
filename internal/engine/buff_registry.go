package engine

import (
	"fmt"

	"github.com/b1tAction/paradiced/internal/core"
	engineaction "github.com/b1tAction/paradiced/internal/engine/action"
	"github.com/b1tAction/paradiced/internal/event"
	"github.com/b1tAction/paradiced/pkg/constants"
)

// EffectHandler defines the handler signature for Buff/Item/Event effects.
// Returns error to propagate failures up to HSM layer.
// Defined in engine layer to maintain correct dependency direction.
type EffectHandler func(phase constants.Phase, ctx *event.Context) error

// StepsModifier is an interface for state objects that hold movement steps.
// Used by 迷途 handler to reverse movement direction without importing HSM package.
// TurnMovingState implements this interface.
type StepsModifier interface {
	GetSteps() int
	SetSteps(steps int)
}

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

	// Hidden buffs are not added to lottery pools
	if !def.Hidden {
		if def.Eval.IsGood() {
			r.goodBuffs = append(r.goodBuffs, def.Type)
		} else if def.Eval.IsBad() {
			r.badBuffs = append(r.badBuffs, def.Type)
		} else {
			r.neutralBuffs = append(r.neutralBuffs, def.Type)
		}
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

// IsHidden checks if a Buff type is hidden (not visible to player/client).
func IsHidden(bt constants.BuffType) bool {
	return GlobalBuffRegistry.IsHidden(bt)
}

func (r *BuffRegistry) IsHidden(bt constants.BuffType) bool {
	def, ok := r.defs[bt]
	if !ok {
		return false
	}
	return def.Hidden
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
	// Curse: LP-1 when applied, LP+1 revert when removed
	GlobalBuffRegistry.RegisterBuff(&core.BuffDefinition{
		Type:        constants.BuffTypeCurse,
		Eval:        constants.EvaluationBad,
		EnglishName: "Curse",
		Name:        "诅咒",
		Desc:        "LP-1 until removed",
		Duration:    3,
	}, &BuffHandlerConfig{
		Phases:      []constants.Phase{constants.PhasePostBuffApplied, constants.PhasePreBuffRemoved},
		Priority:    50,
		NeedConfirm: false,
		Handler:     handleCurseEffect,
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
		Handler:     createEveryNTurnsHandler(2, constants.BuffTypeCorrupt, createModifyHPHandler(-1)),
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

	// Divine: LP+1 when applied, LP-1 revert when removed
	GlobalBuffRegistry.RegisterBuff(&core.BuffDefinition{
		Type:        constants.BuffTypeDivine,
		Eval:        constants.EvaluationVeryGood,
		EnglishName: "Divine",
		Name:        "神眷",
		Desc:        "LP+1 until removed",
		Duration:    3,
	}, &BuffHandlerConfig{
		Phases:      []constants.Phase{constants.PhasePostBuffApplied, constants.PhasePreBuffRemoved},
		Priority:    50,
		NeedConfirm: false,
		Handler:     handleDivineEffect,
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
		Handler:     createEveryNTurnsHandler(2, constants.BuffTypeRain, createModifyHPHandler(1)),
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

	// DeathMark: Hidden buff that blocks all subsequent actions after death.
	// Not visible to client, not drawn in lottery pools.
	GlobalBuffRegistry.RegisterBuff(&core.BuffDefinition{
		Type:        constants.BuffTypeDeathMark,
		Eval:        constants.EvaluationVeryBad,
		EnglishName: "DeathMark",
		Name:        "死亡标记",
		Desc:        "死亡后阻止后续行动",
		Duration:    1,
		Hidden:      true,
	}, &BuffHandlerConfig{
		Phases:      []constants.Phase{constants.PhasePreAction},
		Priority:    999,
		NeedConfirm: false,
		Handler:     handleDeathMarkBlock,
	})
}

// ========== Buff Handler Helpers ==========

func createModifyLPHandler(amount int) EffectHandler {
	return func(phase constants.Phase, ctx *event.Context) error {
		if ctx == nil {
			return fmt.Errorf("handler: event context is nil")
		}
		if ctx.Player == nil {
			return fmt.Errorf("handler: player is nil in event context")
		}

		// Must use Action system - no direct modification
		actionCtx, err := getActionCtxFromEventCtx(ctx)
		if err != nil {
			return err
		}
		_ = actionCtx // ActionContext used for derived action processing

		ctx.AddDerivedAction(engineaction.NewModifyLPAction(ctx.Player, amount, "Buff_Effect"))
		return nil
	}
}

func createModifyHPHandler(amount int) EffectHandler {
	return func(phase constants.Phase, ctx *event.Context) error {
		if ctx == nil {
			return fmt.Errorf("handler: event context is nil")
		}
		if ctx.Player == nil {
			return fmt.Errorf("handler: player is nil in event context")
		}
		if amount == 0 {
			return nil // No change needed
		}

		// Must use Action system - no direct modification
		actionCtx, err := getActionCtxFromEventCtx(ctx)
		if err != nil {
			return err
		}
		_ = actionCtx // ActionContext used for derived action processing

		if amount > 0 {
			ctx.AddDerivedAction(engineaction.NewHealAction(ctx.Player, amount, "Buff_Effect"))
		} else {
			ctx.AddDerivedAction(engineaction.NewDamageAction(ctx.Player, -amount, "Buff_Effect"))
		}
		return nil
	}
}

func createEveryNTurnsHandler(everyN int, buffType constants.BuffType, innerHandler EffectHandler) EffectHandler {
	return func(phase constants.Phase, ctx *event.Context) error {
		if ctx == nil {
			return nil
		}
		if ctx.Player == nil {
			return fmt.Errorf("handler: player is nil in event context")
		}

		// Read counter from Buff.Metadata (persists across turns)
		buff := ctx.Player.GetBuff(buffType)
		if buff == nil {
			return nil // Buff not active, skip
		}
		counter := buff.GetIntOrDefault("buff_turn_counter", 0)
		counter++
		buff.SetInt("buff_turn_counter", counter)

		if counter >= everyN {
			if err := innerHandler(phase, ctx); err != nil {
				return err
			}
			buff.SetInt("buff_turn_counter", 0) // Reset counter
		}
		return nil
	}
}

// ========== Custom Buff Handlers ==========

func handleZhuQueFire(phase constants.Phase, ctx *event.Context) error {
	if phase != constants.PhaseBeforeTurn {
		return nil
	}
	if ctx == nil {
		return fmt.Errorf("handler: event context is nil")
	}
	if ctx.Player == nil {
		return fmt.Errorf("handler: player is nil in event context")
	}

	newCount := ctx.Player.IncrementFireCounter()
	if newCount >= 4 {
		// Must use Action system - no direct modification
		actionCtx, err := getActionCtxFromEventCtx(ctx)
		if err != nil {
			return err
		}
		_ = actionCtx // ActionContext used for derived action processing

		ctx.AddDerivedAction(engineaction.NewModifyLPAction(ctx.Player, 1, "Buff_Fire"))
		ctx.Player.SetFireCounter(0)
	}
	return nil
}

func handleLostReverse(phase constants.Phase, ctx *event.Context) error {
	if phase != constants.PhasePreMove {
		return nil
	}
	if ctx == nil {
		return fmt.Errorf("handler: event context is nil")
	}

	// Get TurnMovingState from context (passed as StepsModifier interface)
	raw, ok := ctx.Get("current_state")
	if !ok {
		return nil // No current_state in context (e.g. test scenario without HSM state)
	}

	movingState, ok := raw.(StepsModifier)
	if !ok || movingState == nil {
		return nil // Not a StepsModifier, skip
	}

	// Reverse steps (prevent double-flip: if Steps < 0, it's already reversed)
	if movingState.GetSteps() > 0 {
		movingState.SetSteps(-movingState.GetSteps())
	}

	ctx.SetBool("reverse_movement", true) // For logging/debugging
	return nil
}

func handleHiddenImmune(phase constants.Phase, ctx *event.Context) error {
	if phase != constants.PhasePreBuffApplied {
		return nil
	}
	if ctx == nil {
		return fmt.Errorf("handler: event context is nil")
	}
	ctx.SetBool("action_blocked", true)
	ctx.SetString("blocked_by", "Buff_Hidden")
	return nil
}

func handlePoisonBadEvent(phase constants.Phase, ctx *event.Context) error {
	if ctx == nil {
		return fmt.Errorf("handler: event context is nil")
	}
	if phase != constants.PhaseBeforeTurn {
		return nil
	}
	ctx.SetBool("draw_bad_event", true)
	return nil
}

func handleExorcismImmunePoison(phase constants.Phase, ctx *event.Context) error {
	if ctx == nil {
		return fmt.Errorf("handler: event context is nil")
	}
	if phase != constants.PhasePreEvent {
		return nil
	}
	ctx.SetBool("block_poison_effect", true)
	return nil
}

func handleDeathMarkBlock(phase constants.Phase, ctx *event.Context) error {
	if ctx == nil {
		return fmt.Errorf("handler: event context is nil")
	}
	if phase != constants.PhasePreAction {
		return nil
	}

	// Get the current action from context
	raw, ok := ctx.Get("current_action")
	if !ok {
		// No action info, block by default
		ctx.SetBool("action_blocked", true)
		return nil
	}
	action, ok := raw.(engineaction.Action)
	if !ok {
		ctx.SetBool("action_blocked", true)
		return nil
	}

	// RespawnAction must execute even for dead players
	if _, isRespawn := action.(*engineaction.RespawnAction); isRespawn {
		return nil // Don't block respawn
	}

	// RemoveBuffAction for DeathMark itself must execute
	// (removing own DeathMark should not block itself)
	if removeAction, isRemove := action.(*engineaction.RemoveBuffAction); isRemove {
		if removeAction.BuffType == constants.BuffTypeDeathMark {
			return nil // Don't block DeathMark removal
		}
	}

	// Block all other actions for dead players
	ctx.SetBool("action_blocked", true)
	return nil
}

func handleDivineEffect(phase constants.Phase, ctx *event.Context) error {
	if ctx == nil {
		return fmt.Errorf("handler: event context is nil")
	}
	if ctx.Player == nil {
		return fmt.Errorf("handler: player is nil in event context")
	}
	actionCtx, err := getActionCtxFromEventCtx(ctx)
	if err != nil {
		return err
	}
	_ = actionCtx // ActionContext used for derived action processing

	switch phase {
	case constants.PhasePostBuffApplied:
		// Only react when Divine buff is applied → LP+1
		if raw, ok := ctx.Get("applied_buff_type"); ok {
			if constants.BuffType(raw.(string)) == constants.BuffTypeDivine {
				ctx.AddDerivedAction(engineaction.NewModifyLPAction(ctx.Player, 1, "Buff_Divine"))
			}
		}
	case constants.PhasePreBuffRemoved:
		// Only react when Divine buff is removed → LP-1 revert
		if raw, ok := ctx.Get("removed_buff_type"); ok {
			if constants.BuffType(raw.(string)) == constants.BuffTypeDivine {
				ctx.AddDerivedAction(engineaction.NewModifyLPAction(ctx.Player, -1, "Buff_Divine_Removal"))
			}
		}
	}
	return nil
}

func handleCurseEffect(phase constants.Phase, ctx *event.Context) error {
	if ctx == nil {
		return fmt.Errorf("handler: event context is nil")
	}
	if ctx.Player == nil {
		return fmt.Errorf("handler: player is nil in event context")
	}
	actionCtx, err := getActionCtxFromEventCtx(ctx)
	if err != nil {
		return err
	}
	_ = actionCtx // ActionContext used for derived action processing

	switch phase {
	case constants.PhasePostBuffApplied:
		// Only react when Curse buff is applied → LP-1
		if raw, ok := ctx.Get("applied_buff_type"); ok {
			if constants.BuffType(raw.(string)) == constants.BuffTypeCurse {
				ctx.AddDerivedAction(engineaction.NewModifyLPAction(ctx.Player, -1, "Buff_Curse"))
			}
		}
	case constants.PhasePreBuffRemoved:
		// Only react when Curse buff is removed → LP+1 revert
		if raw, ok := ctx.Get("removed_buff_type"); ok {
			if constants.BuffType(raw.(string)) == constants.BuffTypeCurse {
				ctx.AddDerivedAction(engineaction.NewModifyLPAction(ctx.Player, 1, "Buff_Curse_Removal"))
			}
		}
	}
	return nil
}

func getActionCtxFromEventCtx(ctx *event.Context) (*engineaction.ActionContext, error) {
	if ctx == nil {
		return nil, fmt.Errorf("handler: event context is nil")
	}

	raw, ok := ctx.Get("action_context")
	if !ok {
		return nil, fmt.Errorf("handler: action_context not found in event context")
	}

	actionCtx, ok := raw.(*engineaction.ActionContext)
	if !ok {
		return nil, fmt.Errorf("handler: action_context is not ActionContext type")
	}
	if actionCtx == nil {
		return nil, fmt.Errorf("handler: action_context is nil")
	}
	return actionCtx, nil
}
