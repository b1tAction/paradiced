// Package engine provides game engine logic for the Fated game.
package engine

import (
	"github.com/b1tAction/Fated/internal/core"
	"github.com/b1tAction/Fated/internal/core/buff"
	engineaction "github.com/b1tAction/Fated/internal/engine/action"
	"github.com/b1tAction/Fated/pkg/action"
	"github.com/b1tAction/Fated/pkg/event"
)

// ========== Default Handler ==========

// executeDefaultBuffAction executes default Buff value effects.
// Used for Buffs without custom handlers, generates Actions based on HPPerTurn/LPPerTurn.
func executeDefaultBuffAction(def *core.BuffDefinition, player *core.Player, actionCtx *engineaction.ActionContext) {
	// Generate HP modification Action
	if def.HPPerTurn != 0 {
		if def.HPPerTurn > 0 {
			actionCtx.PushDerivedAction(engineaction.NewHealAction(player, def.HPPerTurn, "Buff_"+def.EnglishName))
		} else {
			actionCtx.PushDerivedAction(engineaction.NewDamageAction(player, -def.HPPerTurn, "Buff_"+def.EnglishName))
		}
	}

	// Generate LP modification Action
	if def.LPPerTurn != 0 {
		actionCtx.PushDerivedAction(engineaction.NewModifyLPAction(player, def.LPPerTurn, "Buff_"+def.EnglishName))
	}
}

// ========== Buff Action Creation ==========

// createBuffAction creates an Action closure when Buff triggers.
// Uses GlobalRegistry to get custom handler if available.
// The closure expects ActionContext to be passed via event.Context (ctx.Set("action_context", ctx)).
func createBuffAction(buffInstance *core.Buff, def *core.BuffDefinition, phase event.Phase, player *core.Player) func(ctx *event.Context) action.Action {
	return func(ctx *event.Context) action.Action {
		// Get Player from Context (ensure correct type)
		p, ok := ctx.Player.(*core.Player)
		if !ok {
			return nil
		}

		// Get ActionContext from event.Context (set by caller before Publish)
		actionCtxVal, ok := ctx.Get("action_context")
		actionCtx, ok2 := actionCtxVal.(*engineaction.ActionContext)
		if !ok || !ok2 || actionCtx == nil {
			// No ActionContext available - use old direct modification approach
			if def.HPPerTurn != 0 {
				if def.HPPerTurn > 0 {
					p.Heal(def.HPPerTurn)
				} else {
					p.ApplyDamage(-def.HPPerTurn)
				}
			}
			if def.LPPerTurn != 0 {
				p.ModifyLP(def.LPPerTurn)
			}
			return nil
		}

		// Check if has custom handler via GlobalRegistry
		handler := buff.GetBuffHandler(buffInstance.Type)
		if handler != nil {
			// Call custom handler - returns Action or nil
			derivedAction := handler(phase, ctx)

			// If handler returned an Action, return it
			if derivedAction != nil {
				return derivedAction
			}

			// Otherwise, check for Actions set in context
			derivedVal, hasDerived := ctx.Get("derived_action")
			if hasDerived {
				if act, ok := derivedVal.(action.Action); ok && act != nil {
					return act
				}
			}
		} else {
			// Execute default value effect - generates Actions
			executeDefaultBuffAction(def, p, actionCtx)
		}

		return nil
	}
}