package engine

import (
	"fmt"
	"math"

	"github.com/b1tAction/paradiced/internal/core"
	engineaction "github.com/b1tAction/paradiced/internal/engine/action"
	"github.com/b1tAction/paradiced/internal/event"
	"github.com/b1tAction/paradiced/pkg/constants"
	"github.com/b1tAction/paradiced/pkg/id"
	"github.com/b1tAction/paradiced/pkg/resource"
	"github.com/b1tAction/paradiced/pkg/rng"
)

// logHandlerResult logs buff handler execution results via ActionContext's Game.DebugLog.
// Nil-safe: silently returns if ActionContext or Game is nil.
func logHandlerResult(handlerName string, eventCtx *event.Context, result string, extraKeysAndValues ...interface{}) {
	if eventCtx == nil {
		return
	}
	actionCtx, err := getActionCtxFromEventCtx(eventCtx)
	if err != nil || actionCtx == nil || actionCtx.Game == nil {
		return
	}
	kv := []interface{}{"handler", handlerName, "result", result, "player_id", ""}
	if eventCtx.Player != nil {
		kv[4] = eventCtx.Player.ID.UUID()
	}
	kv = append(kv, extraKeysAndValues...)
	actionCtx.Game.GetDebugLog().Info("BuffHandler", kv...)
}

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
	defs    map[constants.BuffType]*constants.BuffDefinition
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
		defs:         make(map[constants.BuffType]*constants.BuffDefinition),
		configs:      make(map[constants.BuffType]*BuffHandlerConfig),
		names:        make(map[constants.BuffType]string),
		goodBuffs:    make([]constants.BuffType, 0),
		badBuffs:     make([]constants.BuffType, 0),
		neutralBuffs: make([]constants.BuffType, 0),
	}
}

// RegisterBuff registers a Buff definition with handler config.
func (r *BuffRegistry) RegisterBuff(def *constants.BuffDefinition, config *BuffHandlerConfig) {
	if def == nil || def.Type == constants.BuffTypeNone {
		return
	}

	r.defs[def.Type] = def
	r.names[def.Type] = def.Name

	// Buffs that are not drawn from lottery pools (IsDraw=false) are excluded
	if def.Type.IsDraw() {
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
func GetBuffDefinition(bt constants.BuffType) *constants.BuffDefinition {
	return GlobalBuffRegistry.GetBuffDefinition(bt)
}

func (r *BuffRegistry) GetBuffDefinition(bt constants.BuffType) *constants.BuffDefinition {
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

// BuildBuffPool builds an []*rng.EvaluatedItem pool from all registered BuffDefinitions
// that can participate in lottery pool draws (IsDraw == true).
// This excludes Boss buffs (Thorns) and Hidden buffs (DeathMark).
func BuildBuffPool() []*rng.EvaluatedItem {
	return GlobalBuffRegistry.buildBuffPool()
}

func (r *BuffRegistry) buildBuffPool() []*rng.EvaluatedItem {
	pool := make([]*rng.EvaluatedItem, 0, len(r.defs))
	for _, def := range r.defs {
		if def.Type.IsDraw() {
			pool = append(pool, &rng.EvaluatedItem{
				Type: string(def.Type),
				Eval: def.Eval,
			})
		}
	}
	return pool
}

// IsHidden checks if a Buff type is hidden (not visible to player/client).
func IsHidden(bt constants.BuffType) bool {
	return bt.IsHidden()
}

func (r *BuffRegistry) IsHidden(bt constants.BuffType) bool {
	return bt.IsHidden()
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
	defs := resource.GlobalDefinitionSet

	// Curse: LP-1 when applied, LP+1 revert when removed
	GlobalBuffRegistry.RegisterBuff(defs.Buffs[constants.BuffTypeCurse], &BuffHandlerConfig{
		Phases:      []constants.Phase{constants.PhasePostBuffApplied, constants.PhasePreBuffRemoved},
		Priority:    50,
		NeedConfirm: false,
		Handler:     handleCurseEffect,
	})

	// Lost: Reverse movement direction for 1 turn
	GlobalBuffRegistry.RegisterBuff(defs.Buffs[constants.BuffTypeLost], &BuffHandlerConfig{
		Phases:      []constants.Phase{constants.PhasePreMove},
		Priority:    100,
		NeedConfirm: false,
		Handler:     handleLostReverse,
	})

	// Corrupt: HP-1 every 2 turns for 4 turns
	GlobalBuffRegistry.RegisterBuff(defs.Buffs[constants.BuffTypeCorrupt], &BuffHandlerConfig{
		Phases:      []constants.Phase{constants.PhaseAfterTurn},
		Priority:    50,
		NeedConfirm: false,
		Handler:     createEveryNTurnsHandler(2, constants.BuffTypeCorrupt, createModifyHPHandler(-1, constants.SourceBuffCorrupt)),
	})

	// Poison: Bad event each turn for 3 turns
	GlobalBuffRegistry.RegisterBuff(defs.Buffs[constants.BuffTypePoison], &BuffHandlerConfig{
		Phases:      []constants.Phase{constants.PhaseBeforeTurn},
		Priority:    30,
		NeedConfirm: false,
		Handler:     handlePoisonBadEvent,
	})

	// Hidden: Immunity to events/buffs for 3 turns (does NOT block damage).
	GlobalBuffRegistry.RegisterBuff(defs.Buffs[constants.BuffTypeHidden], &BuffHandlerConfig{
		Phases:      []constants.Phase{constants.PhasePreBuffApplied},
		Priority:    100,
		NeedConfirm: false,
		Handler:     handleHiddenImmune,
	})

	// Divine: LP+1 when applied, LP-1 revert when removed
	GlobalBuffRegistry.RegisterBuff(defs.Buffs[constants.BuffTypeDivine], &BuffHandlerConfig{
		Phases:      []constants.Phase{constants.PhasePostBuffApplied, constants.PhasePreBuffRemoved},
		Priority:    50,
		NeedConfirm: false,
		Handler:     handleDivineEffect,
	})

	// Rain: HP+1 every 2 turns for 4 turns
	GlobalBuffRegistry.RegisterBuff(defs.Buffs[constants.BuffTypeRain], &BuffHandlerConfig{
		Phases:      []constants.Phase{constants.PhaseAfterTurn},
		Priority:    50,
		NeedConfirm: false,
		Handler:     createEveryNTurnsHandler(2, constants.BuffTypeRain, createModifyHPHandler(1, constants.SourceBuffRain)),
	})

	// Exorcism: Immune to poison buff for 5 turns
	GlobalBuffRegistry.RegisterBuff(defs.Buffs[constants.BuffTypeExorcism], &BuffHandlerConfig{
		Phases:      []constants.Phase{constants.PhasePreEvent},
		Priority:    80,
		NeedConfirm: false,
		Handler:     handleExorcismImmunePoison,
	})

	// Fire: ZhuQue passive, LP+1 every 3 turns (permanent)
	GlobalBuffRegistry.RegisterBuff(defs.Buffs[constants.BuffTypeFire], &BuffHandlerConfig{
		Phases:      []constants.Phase{constants.PhaseBeforeTurn},
		Priority:    10,
		NeedConfirm: false,
		Handler:     handleZhuQueFire,
	})

	// Thorns: Boss skill buff — reflect 30% damage back to attacking player.
	// Not drawn from lottery pools (IsBoss=true). Buff is on BossPlayer.
	GlobalBuffRegistry.RegisterBuff(defs.Buffs[constants.BuffTypeThorns], &BuffHandlerConfig{
		Phases:      []constants.Phase{constants.PhasePreDamage},
		Priority:    50,
		NeedConfirm: false,
		Handler:     handleThornsReflect,
	})

	// DeathMark: Hidden buff that blocks all subsequent actions after death.
	// Not visible to client, not drawn in lottery pools.
	GlobalBuffRegistry.RegisterBuff(defs.Buffs[constants.BuffTypeDeathMark], &BuffHandlerConfig{
		Phases:      []constants.Phase{constants.PhasePreAction},
		Priority:    999,
		NeedConfirm: false,
		Handler:     handleDeathMarkBlock,
	})

	// Dominance: QingLong faction skill — double beneficial action effects for 1 turn
	GlobalBuffRegistry.RegisterBuff(defs.Buffs[constants.BuffTypeDominance], &BuffHandlerConfig{
		Phases:      []constants.Phase{constants.PhasePreAction},
		Priority:    50,
		NeedConfirm: false,
		Handler:     handleDominanceAmplify,
	})

	// RobLuck: BaiHu faction skill — redirect good actions to BaiHu player
	GlobalBuffRegistry.RegisterBuff(defs.Buffs[constants.BuffTypeRobLuck], &BuffHandlerConfig{
		Phases:      []constants.Phase{constants.PhasePreAction, constants.PhasePreBuffApplied},
		Priority:    80,
		NeedConfirm: false,
		Handler:     handleRobLuckRedirect,
	})

	// Suppress: XuanWu faction skill — immunity to bad events and negative buffs for 1 turn
	GlobalBuffRegistry.RegisterBuff(defs.Buffs[constants.BuffTypeSuppress], &BuffHandlerConfig{
		Phases:      []constants.Phase{constants.PhasePreEvent, constants.PhasePreBuffApplied},
		Priority:    90,
		NeedConfirm: false,
		Handler:     handleSuppressImmune,
	})

	// Sinking: share negative actions/buffs with linked player
	GlobalBuffRegistry.RegisterBuff(defs.Buffs[constants.BuffTypeSinking], &BuffHandlerConfig{
		Phases:      []constants.Phase{constants.PhasePreAction, constants.PhasePreBuffApplied},
		Priority:    60,
		NeedConfirm: false,
		Handler:     handleSinkingShare,
	})

	// Eternal: share positive actions/buffs with linked player
	GlobalBuffRegistry.RegisterBuff(defs.Buffs[constants.BuffTypeEternal], &BuffHandlerConfig{
		Phases:      []constants.Phase{constants.PhasePreAction, constants.PhasePreBuffApplied},
		Priority:    60,
		NeedConfirm: false,
		Handler:     handleEternalShare,
	})

	// Fearless: HP locked at 1 for duration, blocks damage/heal at PreAction, sets HP=1 at PostBuffApplied
	GlobalBuffRegistry.RegisterBuff(defs.Buffs[constants.BuffTypeFearless], &BuffHandlerConfig{
		Phases:      []constants.Phase{constants.PhasePreAction, constants.PhasePostBuffApplied},
		Priority:    200,
		NeedConfirm: false,
		Handler:     handleFearless,
	})

	// GoldenBody: damage reduced to floor(damage/2)+1
	GlobalBuffRegistry.RegisterBuff(defs.Buffs[constants.BuffTypeGoldenBody], &BuffHandlerConfig{
		Phases:      []constants.Phase{constants.PhasePreDamage},
		Priority:    70,
		NeedConfirm: false,
		Handler:     handleGoldenBodyReduce,
	})

	// Wrath: outgoing damage +1
	GlobalBuffRegistry.RegisterBuff(defs.Buffs[constants.BuffTypeWrath], &BuffHandlerConfig{
		Phases:      []constants.Phase{constants.PhasePreAction},
		Priority:    60,
		NeedConfirm: false,
		Handler:     handleWrathAmplify,
	})

	// Savior: block one fatal damage, then remove Savior buff
	GlobalBuffRegistry.RegisterBuff(defs.Buffs[constants.BuffTypeSavior], &BuffHandlerConfig{
		Phases:      []constants.Phase{constants.PhasePreDamage},
		Priority:    999,
		NeedConfirm: false,
		Handler:     handleSaviorBlock,
	})

	// SageProtection: respawn in-place (death location) instead of checkpoint
	GlobalBuffRegistry.RegisterBuff(defs.Buffs[constants.BuffTypeSageProtection], &BuffHandlerConfig{
		Phases:      []constants.Phase{constants.PhasePreRespawn},
		Priority:    50,
		NeedConfirm: false,
		Handler:     handleSageProtectionRespawn,
	})
}

// ========== Buff Handler Helpers ==========

func createModifyLPHandler(amount int, source constants.ActionSource) EffectHandler {
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

		ctx.AddDerivedAction(engineaction.NewModifyLPAction(ctx.Player, amount, string(source)))
		return nil
	}
}

func createModifyHPHandler(amount int, source constants.ActionSource) EffectHandler {
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
			ctx.AddDerivedAction(engineaction.NewHealAction(ctx.Player, amount, string(source)))
		} else {
			ctx.AddDerivedAction(engineaction.NewDamageAction(ctx.Player, -amount, string(source)))
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
	if newCount >= 3 {
		// Must use Action system - no direct modification
		actionCtx, err := getActionCtxFromEventCtx(ctx)
		if err != nil {
			return err
		}
		_ = actionCtx // ActionContext used for derived action processing

		ctx.AddDerivedAction(engineaction.NewModifyLPAction(ctx.Player, 1, string(constants.SourceBuffFire)))
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

	// Only block non-positive, non-Boss buffs.
	// IsBoss buffs (Thorns, DeathMark) are forced by game mechanics and bypass Hidden immunity.
	// Positive buffs (Divine, Rain, Exorcism) should also pass through.
	if raw, ok := ctx.Get("applied_buff_type"); ok {
		buffType := constants.BuffType(raw.(string))
		if buffType.IsPositive() || buffType.IsBoss() {
			return nil // Allow positive and Boss buffs through
		}
	}

	ctx.SetBool("action_blocked", true)
	ctx.SetString("blocked_by", string(constants.SourceBuffHidden))
	logHandlerResult("Hidden", ctx, "blocked_buff", "phase", phase)
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
		logHandlerResult("DeathMark", ctx, "blocked_no_action_info")
		return nil
	}
	action, ok := raw.(engineaction.Action)
	if !ok {
		ctx.SetBool("action_blocked", true)
		logHandlerResult("DeathMark", ctx, "blocked_invalid_action_type")
		return nil
	}

	// RespawnAction must execute even for dead players
	if _, isRespawn := action.(*engineaction.RespawnAction); isRespawn {
		logHandlerResult("DeathMark", ctx, "skip_respawn_allowed")
		return nil // Don't block respawn
	}

	// RemoveBuffAction for DeathMark itself must execute
	// (removing own DeathMark should not block itself)
	if removeAction, isRemove := action.(*engineaction.RemoveBuffAction); isRemove {
		if removeAction.BuffType == constants.BuffTypeDeathMark {
			logHandlerResult("DeathMark", ctx, "skip_death_mark_removal_allowed")
			return nil // Don't block DeathMark removal
		}
	}

	// Block all other actions for dead players
	ctx.SetBool("action_blocked", true)
	logHandlerResult("DeathMark", ctx, "blocked", "action_type", action.Type(), "player_id", ctx.Player.ID.UUID())
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
				ctx.AddDerivedAction(engineaction.NewModifyLPAction(ctx.Player, 1, string(constants.SourceBuffDivine)))
			}
		}
	case constants.PhasePreBuffRemoved:
		// Only react when Divine buff is removed → LP-1 revert
		if raw, ok := ctx.Get("removed_buff_type"); ok {
			if constants.BuffType(raw.(string)) == constants.BuffTypeDivine {
				ctx.AddDerivedAction(engineaction.NewModifyLPAction(ctx.Player, -1, string(constants.SourceBuffDivineRemoval)))
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
				ctx.AddDerivedAction(engineaction.NewModifyLPAction(ctx.Player, -1, string(constants.SourceBuffCurse)))
			}
		}
	case constants.PhasePreBuffRemoved:
		// Only react when Curse buff is removed → LP+1 revert
		if raw, ok := ctx.Get("removed_buff_type"); ok {
			if constants.BuffType(raw.(string)) == constants.BuffTypeCurse {
				ctx.AddDerivedAction(engineaction.NewModifyLPAction(ctx.Player, 1, string(constants.SourceBuffCurseRemoval)))
			}
		}
	}
	return nil
}

// thornsReflectRate is the damage reflect rate for the Thorns buff (30%).
const thornsReflectRate = 0.3

func handleThornsReflect(phase constants.Phase, ctx *event.Context) error {
	if phase != constants.PhasePreDamage {
		return nil
	}
	if ctx == nil || ctx.Player == nil {
		return nil
	}

	// Get BossDamageAction from context
	raw, ok := ctx.Get("current_action")
	if !ok {
		return nil
	}
	bossDamageAction, ok := raw.(*engineaction.BossDamageAction)
	if !ok || bossDamageAction == nil {
		return nil
	}

	// Only reflect if BossPlayer has Thorns buff
	if !ctx.Player.HasBuff(constants.BuffTypeThorns) {
		return nil
	}

	reflectDamage := int(math.Round(float64(bossDamageAction.Damage) * thornsReflectRate))
	if reflectDamage <= 0 {
		return nil
	}

	// Push derived PiercingDamageAction (buff reflect → attacking player).
	// Thorns reflect is a buff passive effect, not a Boss active attack,
	// so it derives DamageAction (not BossAttackAction). Piercing ensures
	// reflect bypasses PhasePreDamage interception (consistent with previous
	// BossAttackAction behavior which also skipped PreDamage interception).
	reflectAction := engineaction.NewPiercingDamageAction(
		bossDamageAction.SourcePlayer, // attacking player (target of reflect)
		reflectDamage,
		string(constants.SourceBuffThornsReflect),
	)
	ctx.AddDerivedAction(reflectAction)
	logHandlerResult("Thorns", ctx, "reflect_damage", "original_damage", bossDamageAction.Damage, "reflect_damage", reflectDamage, "attacker", bossDamageAction.SourcePlayer.ID.UUID())
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

// ========== Faction Skill Buff Handlers ==========

// handleDominanceAmplify doubles beneficial action effects.
// When Dominance buff is active on QingLong player:
// - BossDamageAction: push derived BossDamageAction with same damage amount (total = 2x)
// - HealAction targeting self: push derived HealAction with same amount (total = 2x)
// - ModifyLPAction (positive) targeting self: push derived ModifyLPAction with same amount (total = 2x)
// Does NOT block the original action — both original + derived execute for 2x total effect.
func handleDominanceAmplify(phase constants.Phase, ctx *event.Context) error {
	if phase != constants.PhasePreAction {
		return nil
	}
	if ctx == nil || ctx.Player == nil {
		return nil
	}

	raw, ok := ctx.Get("current_action")
	if !ok {
		return nil
	}
	action, ok := raw.(engineaction.Action)
	if !ok || action == nil {
		return nil
	}

	// Skip if this action was already amplified by Dominance (prevents infinite loops)
	if action.Source() == string(constants.SourceFactionQingLongDominance) {
		return nil
	}

	source := string(constants.SourceFactionQingLongDominance)

	switch a := action.(type) {
	case *engineaction.BossDamageAction:
		// Only amplify if QingLong player (Dominance holder) is the attacker
		if a.SourcePlayer != ctx.Player {
			return nil
		}
		ctx.AddDerivedAction(engineaction.NewBossDamageAction(
			a.SourcePlayer, a.TargetPlayer(), a.Damage, false, source,
		))
		logHandlerResult("Dominance", ctx, "amplified_boss_damage", "damage", a.Damage, "attacker", a.SourcePlayer.ID.UUID())

	case *engineaction.HealAction:
		// Only amplify heal targeting Dominance holder
		if a.TargetPlayer() != ctx.Player || a.Amount <= 0 {
			return nil
		}
		ctx.AddDerivedAction(engineaction.NewHealAction(ctx.Player, a.Amount, source))
		logHandlerResult("Dominance", ctx, "amplified_heal", "amount", a.Amount)

	case *engineaction.ModifyLPAction:
		// Only amplify positive LP targeting Dominance holder
		if a.TargetPlayer() != ctx.Player || a.Amount <= 0 {
			return nil
		}
		ctx.AddDerivedAction(engineaction.NewModifyLPAction(ctx.Player, a.Amount, source))
		logHandlerResult("Dominance", ctx, "amplified_modify_lp", "amount", a.Amount)
	}

	return nil
}

// handleRobLuckRedirect redirects beneficial actions from RobLuck target to BaiHu player.
// PhasePreAction: intercept HealAction/ModifyLPAction(positive)/AddItemAction targeting
// RobLuck-buffed player → block + push derived to BaiHu player.
// PhasePreBuffApplied: intercept positive buff applied to RobLuck-buffed player →
// block + push AddBuffAction to BaiHu player.
func handleRobLuckRedirect(phase constants.Phase, ctx *event.Context) error {
	if ctx == nil || ctx.Player == nil {
		return nil
	}

	// Get BaiHu source player from buff metadata
	buff := ctx.Player.GetBuff(constants.BuffTypeRobLuck)
	if buff == nil {
		return nil
	}
	sourcePlayerID := buff.GetStringOrDefault("rob_luck_source_player", "")
	if sourcePlayerID == "" {
		return nil
	}

	// Skip if this action was already redirected by RobLuck (prevents infinite loops)
	// Applies to both PhasePreAction and PhasePreBuffApplied
	if raw, ok := ctx.Get("current_action"); ok {
		action, ok := raw.(engineaction.Action)
		if ok && action != nil && action.Source() == string(constants.SourceFactionBaiHuRobLuck) {
			return nil
		}
	}

	// Find BaiHu player from game via ActionContext
	actionCtx, err := getActionCtxFromEventCtx(ctx)
	if err != nil {
		return err
	}
	parsedID, err := id.ParsePlayerID(sourcePlayerID)
	if err != nil {
		return nil // Invalid ID format, skip redirect
	}
	rawPlayer := actionCtx.Game.GetPlayerInterface(parsedID)
	baiHuPlayer, ok := rawPlayer.(*core.Player)
	if !ok || baiHuPlayer == nil {
		return nil // BaiHu player not found, skip redirect
	}

	source := string(constants.SourceFactionBaiHuRobLuck)

	switch phase {
	case constants.PhasePreAction:
		raw, ok := ctx.Get("current_action")
		if !ok {
			return nil
		}
		action, ok := raw.(engineaction.Action)
		if !ok || action == nil {
			return nil
		}

		switch a := action.(type) {
		case *engineaction.HealAction:
			if a.Amount > 0 && a.TargetPlayer() == ctx.Player {
				ctx.SetBool("action_blocked", true)
				ctx.SetString("blocked_by", source)
				ctx.AddDerivedAction(engineaction.NewHealAction(baiHuPlayer, a.Amount, source))
				logHandlerResult("RobLuck", ctx, "redirected_heal", "amount", a.Amount, "baihu_player", baiHuPlayer.ID.UUID())
			}

		case *engineaction.ModifyLPAction:
			if a.Amount > 0 && a.TargetPlayer() == ctx.Player {
				ctx.SetBool("action_blocked", true)
				ctx.SetString("blocked_by", source)
				ctx.AddDerivedAction(engineaction.NewModifyLPAction(baiHuPlayer, a.Amount, source))
				logHandlerResult("RobLuck", ctx, "redirected_modify_lp", "amount", a.Amount, "baihu_player", baiHuPlayer.ID.UUID())
			}

		case *engineaction.AddItemAction:
			if a.TargetPlayer() == ctx.Player {
				ctx.SetBool("action_blocked", true)
				ctx.SetString("blocked_by", source)
				ctx.AddDerivedAction(engineaction.NewAddItemAction(baiHuPlayer, a.ItemType, source))
				logHandlerResult("RobLuck", ctx, "redirected_add_item", "item_type", a.ItemType, "baihu_player", baiHuPlayer.ID.UUID())
			}
		}

	case constants.PhasePreBuffApplied:
		if raw, ok := ctx.Get("applied_buff_type"); ok {
			buffType := constants.BuffType(raw.(string))
			if buffType.IsPositive() {
				ctx.SetBool("action_blocked", true)
				ctx.SetString("blocked_by", source)
				ctx.AddDerivedAction(engineaction.NewAddBuffAction(baiHuPlayer, buffType, source))
				logHandlerResult("RobLuck", ctx, "redirected_buff", "buff_type", buffType, "baihu_player", baiHuPlayer.ID.UUID())
			}
		}
	}

	return nil
}

// handleSuppressImmune blocks bad events and negative buffs.
// PhasePreEvent: block events with bad evaluation.
// PhasePreBuffApplied: block negative buff types.
// Does NOT block damage (distinct from Hidden buff).
func handleSuppressImmune(phase constants.Phase, ctx *event.Context) error {
	if ctx == nil || ctx.Player == nil {
		return nil
	}

	source := string(constants.SourceFactionXuanWu) + "_suppress"

	switch phase {
	case constants.PhasePreEvent:
		// Block bad events (evaluation ≤ 40)
		if raw, ok := ctx.Get("event_evaluation"); ok {
			eval := constants.Evaluation(raw.(int))
			if eval.IsBad() {
				ctx.SetBool("action_blocked", true)
				ctx.SetString("blocked_by", source)
				logHandlerResult("Suppress", ctx, "blocked_bad_event", "phase", phase, "evaluation", eval)
			}
		}

	case constants.PhasePreBuffApplied:
		// Block negative buffs
		if raw, ok := ctx.Get("applied_buff_type"); ok {
			buffType := constants.BuffType(raw.(string))
			if buffType.IsNegative() {
				ctx.SetBool("action_blocked", true)
				ctx.SetString("blocked_by", source)
				logHandlerResult("Suppress", ctx, "blocked_negative_buff", "phase", phase, "buff_type", buffType)
			}
		}
	}

	return nil
}

// ========== New Buff Handlers ==========

// resolveLinkedPlayer finds a player by UUID from the game's player list.
// Used by Sinking and Eternal handlers to find their linked player.
func resolveLinkedPlayer(actionCtx *engineaction.ActionContext, linkedPlayerUUID string) *core.Player {
	if actionCtx == nil || actionCtx.Game == nil || linkedPlayerUUID == "" {
		return nil
	}
	game, ok := actionCtx.Game.(*Game)
	if !ok {
		return nil // Cannot access game players (e.g. mock in tests)
	}
	for _, p := range game.Players {
		if p.ID.UUID() == linkedPlayerUUID {
			return p
		}
	}
	return nil
}

// handleSinkingShare shares negative actions/buffs with linked player.
// PhasePreAction: share DamageAction and negative ModifyLPAction targeting Sinking holder with linked player.
// PhasePreBuffApplied: share negative buffs (excluding Boss and Hidden) targeting Sinking holder with linked player.
func handleSinkingShare(phase constants.Phase, ctx *event.Context) error {
	if ctx == nil || ctx.Player == nil {
		return nil
	}

	// Skip if this action was already shared by Sinking (prevents infinite loops)
	if raw, ok := ctx.Get("current_action"); ok {
		action, ok := raw.(engineaction.Action)
		if ok && action != nil && action.Source() == string(constants.SourceBuffSinking) {
			return nil
		}
	}

	// Resolve linked player from buff metadata
	linkedPlayerUUID := ""
	sinkingBuff := ctx.Player.GetBuff(constants.BuffTypeSinking)
	if sinkingBuff != nil {
		linkedPlayerUUID = sinkingBuff.Metadata.GetStringOrDefault("sinking_linked_player", "")
	}
	if linkedPlayerUUID == "" {
		return nil // No linked player, skip
	}

	actionCtx, err := getActionCtxFromEventCtx(ctx)
	if err != nil {
		return err
	}
	linkedPlayer := resolveLinkedPlayer(actionCtx, linkedPlayerUUID)
	if linkedPlayer == nil {
		return nil // Linked player not found
	}

	source := string(constants.SourceBuffSinking)

	switch phase {
	case constants.PhasePreAction:
		raw, ok := ctx.Get("current_action")
		if !ok {
			return nil
		}
		action, ok := raw.(engineaction.Action)
		if !ok || action == nil {
			return nil
		}

		switch a := action.(type) {
		case *engineaction.DamageAction:
			// Share damage with linked player if targeting Sinking holder
			if a.TargetPlayer() == ctx.Player {
				ctx.AddDerivedAction(engineaction.NewDamageAction(linkedPlayer, a.Amount, source))
				logHandlerResult("Sinking", ctx, "shared_damage", "amount", a.Amount, "linked_player", linkedPlayer.ID.UUID())
			}

		case *engineaction.ModifyLPAction:
			// Share negative LP modification with linked player if targeting Sinking holder
			if a.Amount < 0 && a.TargetPlayer() == ctx.Player {
				ctx.AddDerivedAction(engineaction.NewModifyLPAction(linkedPlayer, a.Amount, source))
				logHandlerResult("Sinking", ctx, "shared_negative_modify_lp", "amount", a.Amount, "linked_player", linkedPlayer.ID.UUID())
			}
		}

	case constants.PhasePreBuffApplied:
		if raw, ok := ctx.Get("applied_buff_type"); ok {
			buffType := constants.BuffType(raw.(string))
			// Share negative buffs (excluding Boss and Hidden)
			if buffType.IsNegative() && !buffType.IsBoss() && !buffType.IsHidden() {
				ctx.AddDerivedAction(engineaction.NewAddBuffAction(linkedPlayer, buffType, source))
				logHandlerResult("Sinking", ctx, "shared_buff", "buff_type", buffType, "linked_player", linkedPlayer.ID.UUID())
			}
		}
	}

	return nil
}

// handleEternalShare shares positive actions/buffs with linked player.
// PhasePreAction: share HealAction and positive ModifyLPAction targeting Eternal holder with linked player.
// PhasePreBuffApplied: share positive buffs (excluding Boss, Hidden, Faction) targeting Eternal holder with linked player.
func handleEternalShare(phase constants.Phase, ctx *event.Context) error {
	if ctx == nil || ctx.Player == nil {
		return nil
	}

	// Skip if this action was already shared by Eternal (prevents infinite loops)
	if raw, ok := ctx.Get("current_action"); ok {
		action, ok := raw.(engineaction.Action)
		if ok && action != nil && action.Source() == string(constants.SourceBuffEternal) {
			return nil
		}
	}

	// Resolve linked player from buff metadata
	linkedPlayerUUID := ""
	eternalBuff := ctx.Player.GetBuff(constants.BuffTypeEternal)
	if eternalBuff != nil {
		linkedPlayerUUID = eternalBuff.Metadata.GetStringOrDefault("eternal_linked_player", "")
	}
	if linkedPlayerUUID == "" {
		return nil // No linked player, skip
	}

	actionCtx, err := getActionCtxFromEventCtx(ctx)
	if err != nil {
		return err
	}
	linkedPlayer := resolveLinkedPlayer(actionCtx, linkedPlayerUUID)
	if linkedPlayer == nil {
		return nil // Linked player not found
	}

	source := string(constants.SourceBuffEternal)

	switch phase {
	case constants.PhasePreAction:
		raw, ok := ctx.Get("current_action")
		if !ok {
			return nil
		}
		action, ok := raw.(engineaction.Action)
		if !ok || action == nil {
			return nil
		}

		switch a := action.(type) {
		case *engineaction.HealAction:
			// Share healing with linked player if targeting Eternal holder
			if a.Amount > 0 && a.TargetPlayer() == ctx.Player {
				ctx.AddDerivedAction(engineaction.NewHealAction(linkedPlayer, a.Amount, source))
				logHandlerResult("Eternal", ctx, "shared_heal", "amount", a.Amount, "linked_player", linkedPlayer.ID.UUID())
			}

		case *engineaction.ModifyLPAction:
			if a.Amount > 0 && a.TargetPlayer() == ctx.Player {
				ctx.AddDerivedAction(engineaction.NewModifyLPAction(linkedPlayer, a.Amount, source))
				logHandlerResult("Eternal", ctx, "shared_positive_modify_lp", "amount", a.Amount, "linked_player", linkedPlayer.ID.UUID())
			}

		case *engineaction.AddItemAction:
			if a.TargetPlayer() == ctx.Player {
				ctx.AddDerivedAction(engineaction.NewAddItemAction(linkedPlayer, a.ItemType, source))
				logHandlerResult("Eternal", ctx, "shared_add_item", "item_type", a.ItemType, "linked_player", linkedPlayer.ID.UUID())
			}
		}

	case constants.PhasePreBuffApplied:
		if raw, ok := ctx.Get("applied_buff_type"); ok {
			buffType := constants.BuffType(raw.(string))
			// Share positive buffs (excluding Boss, Hidden, Faction)
			if buffType.IsPositive() && !buffType.IsBoss() && !buffType.IsHidden() && !buffType.IsFaction() {
				ctx.AddDerivedAction(engineaction.NewAddBuffAction(linkedPlayer, buffType, source))
				logHandlerResult("Eternal", ctx, "shared_buff", "buff_type", buffType, "linked_player", linkedPlayer.ID.UUID())
			}
		}
	}

	return nil
}

// handleFearless blocks damage/heal and maintains HP at 1.
// PhasePreAction: block DamageAction/HealAction targeting Fearless holder.
// Allow self-damage from Fearless (source=buff_fearless) to pass through for HP reduction.
// PhasePostBuffApplied: push PiercingDamageAction to reduce HP to 1 when Fearless is applied.
func handleFearless(phase constants.Phase, ctx *event.Context) error {
	if ctx == nil || ctx.Player == nil {
		return nil
	}

	switch phase {
	case constants.PhasePreAction:
		raw, ok := ctx.Get("current_action")
		if !ok {
			return nil
		}
		action, ok := raw.(engineaction.Action)
		if !ok || action == nil {
			return nil
		}

		switch a := action.(type) {
		case *engineaction.DamageAction:
			if a.TargetPlayer() == ctx.Player {
				// Allow Fearless's own HP-setting damage to pass through
				if a.Source() == string(constants.SourceBuffFearless) {
					return nil
				}
				ctx.SetBool("action_blocked", true)
				ctx.SetString("blocked_by", string(constants.SourceBuffFearless))
				logHandlerResult("Fearless", ctx, "blocked_damage", "amount", a.Amount)
			}

		case *engineaction.HealAction:
			if a.TargetPlayer() == ctx.Player {
				ctx.SetBool("action_blocked", true)
				ctx.SetString("blocked_by", string(constants.SourceBuffFearless))
				logHandlerResult("Fearless", ctx, "blocked_heal", "amount", a.Amount)
			}

		default:
			// Allow all other actions through (including RemoveBuffAction)
		}

	case constants.PhasePostBuffApplied:
		if raw, ok := ctx.Get("applied_buff_type"); ok {
			if constants.BuffType(raw.(string)) == constants.BuffTypeFearless {
				damageAmount := ctx.Player.HP - 1
				if damageAmount > 0 {
					ctx.AddDerivedAction(engineaction.NewPiercingDamageAction(ctx.Player, damageAmount, string(constants.SourceBuffFearless)))
				}
				logHandlerResult("Fearless", ctx, "reduce_hp_to_1", "damage_amount", damageAmount)
			}
		}
	}

	return nil
}

// handleGoldenBodyReduce reduces incoming non-piercing damage to floor(damage/2)+1.
// PhasePreDamage: modify DamageAction.Amount in-place.
// Minimum result is 1 (damage=1: 0+1=1, damage=3: 1+1=2, damage=5: 2+1=3).
func handleGoldenBodyReduce(phase constants.Phase, ctx *event.Context) error {
	if phase != constants.PhasePreDamage {
		return nil
	}
	if ctx == nil || ctx.Player == nil {
		return nil
	}

	raw, ok := ctx.Get("current_action")
	if !ok {
		return nil
	}
	damageAction, ok := raw.(*engineaction.DamageAction)
	if !ok || damageAction == nil {
		return nil
	}

	// Skip piercing damage (cannot be intercepted)
	if damageAction.IsPiercing {
		return nil
	}

	// Reduce damage: floor(damage/2) + 1
	originalAmount := damageAction.Amount
	damageAction.Amount = damageAction.Amount/2 + 1
	logHandlerResult("GoldenBody", ctx, "reduced_damage", "original_amount", originalAmount, "new_amount", damageAction.Amount)

	return nil
}

// handleWrathAmplify adds +1 to outgoing damage from Wrath holder.
// PhasePreAction: when Wrath holder is the source of DamageAction or BossDamageAction,
// push derived action with +1 additional damage.
func handleWrathAmplify(phase constants.Phase, ctx *event.Context) error {
	if phase != constants.PhasePreAction {
		return nil
	}
	if ctx == nil || ctx.Player == nil {
		return nil
	}

	raw, ok := ctx.Get("current_action")
	if !ok {
		return nil
	}
	action, ok := raw.(engineaction.Action)
	if !ok || action == nil {
		return nil
	}

	// Skip if this action was already amplified by Wrath (prevents infinite loops)
	if action.Source() == string(constants.SourceBuffWrath) {
		return nil
	}

	source := string(constants.SourceBuffWrath)

	switch a := action.(type) {
	case *engineaction.DamageAction:
		// Only amplify if Wrath holder is the source player of the damage
		if a.SourcePlayer != nil && a.SourcePlayer == ctx.Player {
			target := a.TargetPlayer()
			ctx.AddDerivedAction(engineaction.NewDamageAction(target, 1, source))
			logHandlerResult("Wrath", ctx, "amplified_damage", "target", target.ID.UUID())
		}

	case *engineaction.BossDamageAction:
		// Only amplify if Wrath holder is the attacker (SourcePlayer)
		if a.SourcePlayer == ctx.Player {
			ctx.AddDerivedAction(engineaction.NewBossDamageAction(
				a.SourcePlayer, a.TargetPlayer(), 1, false, source,
			))
			logHandlerResult("Wrath", ctx, "amplified_boss_damage", "attacker", a.SourcePlayer.ID.UUID())
		}
	}

	return nil
}

// handleSaviorBlock blocks one fatal damage and removes Savior buff.
// PhasePreDamage: if DamageAction would kill the player (HP - Amount <= 0),
// block the action and remove Savior buff via derived RemoveBuffAction.
func handleSaviorBlock(phase constants.Phase, ctx *event.Context) error {
	if phase != constants.PhasePreDamage {
		return nil
	}
	if ctx == nil || ctx.Player == nil {
		return nil
	}

	raw, ok := ctx.Get("current_action")
	if !ok {
		return nil
	}
	damageAction, ok := raw.(*engineaction.DamageAction)
	if !ok || damageAction == nil {
		return nil
	}

	// Check if damage would be fatal (HP - Amount <= 0)
	if ctx.Player.HP-damageAction.Amount <= 0 {
		ctx.SetBool("action_blocked", true)
		ctx.SetString("blocked_by", string(constants.SourceBuffSavior))
		ctx.AddDerivedAction(engineaction.NewRemoveBuffAction(ctx.Player, constants.BuffTypeSavior, string(constants.SourceBuffSavior)))
		logHandlerResult("Savior", ctx, "blocked_fatal_damage", "damage", damageAction.Amount, "player_hp", ctx.Player.HP)
	}

	return nil
}

// handleSageProtectionRespawn modifies respawn position to player's current location
// (death location instead of checkpoint) and removes SageProtection buff.
// PhasePreRespawn: modify RespawnAction.CheckpointPos to ctx.Player.Position,
// then remove SageProtection buff via derived RemoveBuffAction.
func handleSageProtectionRespawn(phase constants.Phase, ctx *event.Context) error {
	if phase != constants.PhasePreRespawn {
		return nil
	}
	if ctx == nil || ctx.Player == nil {
		return nil
	}

	raw, ok := ctx.Get("current_action")
	if !ok {
		return nil
	}
	respawnAction, ok := raw.(*engineaction.RespawnAction)
	if !ok || respawnAction == nil {
		return nil
	}

	// Modify respawn position to player's current position (death location)
	respawnAction.CheckpointPos = ctx.Player.Position
	ctx.AddDerivedAction(engineaction.NewRemoveBuffAction(ctx.Player, constants.BuffTypeSageProtection, string(constants.SourceBuffSageProtection)))
	logHandlerResult("SageProtection", ctx, "respawn_in_place", "position", ctx.Player.Position)

	return nil
}
