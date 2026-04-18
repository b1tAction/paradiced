// Package engine provides game engine logic for the Paradiced game.
package engine

import (
	"github.com/b1tAction/paradiced/internal/core"
	engineaction "github.com/b1tAction/paradiced/internal/engine/action"
	"github.com/b1tAction/paradiced/internal/event"
	"github.com/b1tAction/paradiced/pkg/constants"
)

// ========== Buff Action Creation ==========

// createBuffAction creates an Action closure when Buff triggers.
// Uses BuffHandlerConfig.Handler for effect execution.
func createBuffAction(buffInstance *core.Buff, def *core.BuffDefinition, config *BuffHandlerConfig, phase constants.Phase, player *core.Player) func(ctx *event.Context) {
	return func(ctx *event.Context) {
		// Get ActionContext from event.Context (set by caller before Publish)
		actionCtxVal, ok := ctx.Get("action_context")
		actionCtx, ok2 := actionCtxVal.(*engineaction.ActionContext)
		if !ok || !ok2 || actionCtx == nil {
			// No ActionContext available - execute Handler directly
			if config.Handler != nil {
				config.Handler(phase, ctx)
			}
			return
		}

		// Execute Handler - Handler uses ctx.AddDerivedAction() for new Actions
		if config.Handler != nil {
			config.Handler(phase, ctx)
		}
	}
}

// ========== Item Action Creation ==========

// createItemAction creates an Action closure when Item triggers.
// Uses ItemHandlerConfig.Handler for effect execution.
func createItemAction(itemInstance *core.Item, def *core.ItemDefinition, config *ItemHandlerConfig, player *core.Player) func(ctx *event.Context) {
	return func(ctx *event.Context) {
		// Get ActionContext from event.Context
		actionCtxVal, ok := ctx.Get("action_context")
		actionCtx, ok2 := actionCtxVal.(*engineaction.ActionContext)
		if !ok || !ok2 || actionCtx == nil {
			// No ActionContext - execute Handler directly
			if config.Handler != nil {
				config.Handler(constants.PhaseItemUsed, ctx)
			}
			return
		}

		// Execute Handler
		if config.Handler != nil {
			config.Handler(constants.PhaseItemUsed, ctx)
		}
	}
}

// ========== Helper Action Creation Functions ==========

// These functions are used by Handlers to create Actions.
// They are called from Handler implementations via ActionContext.

// NewHealAction creates a HealAction.
func NewHealAction(target *core.Player, amount int, source string) *engineaction.HealAction {
	return engineaction.NewHealAction(target, amount, source)
}

// NewDamageAction creates a DamageAction.
func NewDamageAction(target *core.Player, amount int, source string) *engineaction.DamageAction {
	return engineaction.NewDamageAction(target, amount, source)
}

// NewModifyLPAction creates a ModifyLPAction.
func NewModifyLPAction(target *core.Player, amount int, source string) *engineaction.ModifyLPAction {
	return engineaction.NewModifyLPAction(target, amount, source)
}

// NewAddBuffAction creates an AddBuffAction.
func NewAddBuffAction(target *core.Player, buffType constants.BuffType, duration int, source string) *engineaction.AddBuffAction {
	return engineaction.NewAddBuffAction(target, buffType, duration, source)
}

// NewRemoveBuffAction creates a RemoveBuffAction.
func NewRemoveBuffAction(target *core.Player, buffType constants.BuffType, source string) *engineaction.RemoveBuffAction {
	return engineaction.NewRemoveBuffAction(target, buffType, source)
}

// NewMoveAction creates a MoveAction.
func NewMoveAction(target *core.Player, steps int, source string) *engineaction.MoveAction {
	return engineaction.NewMoveAction(target, steps, source)
}

// NewTeleportAction creates a TeleportAction.
func NewTeleportAction(target *core.Player, position int, source string) *engineaction.TeleportAction {
	return engineaction.NewTeleportAction(target, position, source)
}