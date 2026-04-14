// Package handler provides unified effect handler types for Buff/Item/Event/Faction.
package handler

import (
	"github.com/b1tAction/Fated/pkg/event"
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
//	func UndyingHandler(phase event.Phase, ctx *event.Context) {
//	    if phase != event.PhasePreRespawn {
//	        return
//	    }
//	    ctx.SetBool("action_blocked", true)
//	    player := ctx.Player.(*core.Player)
//	    ctx.AddDerivedAction(NewHealAction(player, player.MaxHP, "Buff_Undying"))
//	    ctx.AddDerivedAction(NewRemoveBuffAction(player, BuffTypeUndying, "Buff_Undying"))
//	}
type EffectHandler func(phase event.Phase, ctx *event.Context)