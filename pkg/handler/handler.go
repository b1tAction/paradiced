// Package handler provides unified effect handler types for Buff/Item/Event/Faction.
package handler

import (
	"github.com/b1tAction/fated/pkg/constants"
	"github.com/b1tAction/fated/pkg/event"
)

// EffectHandler is a unified handler function for Buff/Item/Event/Faction effects.
// Handlers use ctx.AddDerivedAction() to generate new actions.
// All effect sources share this signature for consistency.
//
// Usage:
//   - Buff handlers: registered via BuffRegistry.RegisterBuff(def, handler)
//   - Item handlers: registered via ItemRegistry.RegisterItem(def, handler)
//   - Event handlers: registered via EventRegistry.RegisterEvent(def, handler)
//   - Faction handlers: managed separately for faction passives
//
// Example:
//
//	func UndyingHandler(phase constants.Phase, ctx *event.Context) {
//	    if phase != constants.PhasePreRespawn {
//	        return
//	    }
//	    ctx.SetBool("action_blocked", true)
//	    player := ctx.Player.(*core.Player)
//	    ctx.AddDerivedAction(NewHealAction(player, player.MaxHP, "Buff_Undying"))
//	    ctx.AddDerivedAction(NewRemoveBuffAction(player, BuffTypeUndying, "Buff_Undying"))
//	}
type EffectHandler func(phase constants.Phase, ctx *event.Context)
