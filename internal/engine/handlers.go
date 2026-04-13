// Package engine provides game engine logic for the Fated game.
package engine

import (
	"github.com/b1tAction/Fated/internal/core"
	"github.com/b1tAction/Fated/pkg/event"
)

// ========== Default Handler ==========

// executeDefaultBuffAction executes default Buff value effects.
// Used for Buffs without custom handlers, modifies values based on HPPerTurn/LPPerTurn.
func executeDefaultBuffAction(def *core.BuffDefinition, player *core.Player) {
	// Modify HP
	if def.HPPerTurn != 0 {
		if def.HPPerTurn > 0 {
			player.Heal(def.HPPerTurn)
		} else {
			player.ApplyDamage(-def.HPPerTurn) // Negative HPPerTurn means damage
		}
	}

	// Modify LP
	if def.LPPerTurn != 0 {
		player.ModifyLP(def.LPPerTurn)
	}
}

// ========== Buff Action Creation ==========

// createBuffAction creates an Action closure when Buff triggers.
// Uses GlobalRegistry to get custom handler if available.
func createBuffAction(buff *core.Buff, def *core.BuffDefinition, phase event.Phase, player *core.Player) func(ctx *event.Context) {
	return func(ctx *event.Context) {
		// Get Player from Context (ensure correct type)
		p, ok := ctx.Player.(*core.Player)
		if !ok {
			return
		}

		// Check if has custom handler via GlobalRegistry
		handler := core.GetBuffHandler(buff.Type)
		if handler != nil {
			// Call custom handler
			handler(phase, ctx)
		} else {
			// Execute default value effect
			executeDefaultBuffAction(def, p)
		}
	}
}