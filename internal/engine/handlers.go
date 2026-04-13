// Package engine provides game engine logic for the Fated game.
package engine

import (
	"github.com/b1tAction/Fated/internal/core"
	"github.com/b1tAction/Fated/pkg/event"
)

// EventHandler is a highly customized Buff effect handler function.
// Through strategy pattern, each Buff can have its own dedicated handling logic.
// Parameters:
//   - phase: currently triggered Phase
//   - ctx: event context, containing Player, Data etc.
type EventHandler func(phase event.Phase, ctx *event.Context)

// BuffHandlers is the Buff handling strategy registry.
// Maps BuffType to its customized EventHandler.
// If Buff has no registered custom handler, default value handler is used.
var BuffHandlers = map[core.BuffType]EventHandler{
	core.BuffTypeFire: handleZhuQueFire, // ZhuQue Fire: LP+1 every 4 turns
	// More custom handlers can be registered here
	// For example:
	// core.BuffTypeHidden: handleHiddenImmunity,    // Hidden: damage immunity
	// core.BuffTypeLost: handleLostReverse,         // Lost: reverse movement
}

// HasCustomHandler checks if Buff has a custom handler.
func HasCustomHandler(buffType core.BuffType) bool {
	_, ok := BuffHandlers[buffType]
	return ok
}

// GetHandler returns Buff's custom handler.
func GetHandler(buffType core.BuffType) EventHandler {
	if handler, ok := BuffHandlers[buffType]; ok {
		return handler
	}
	return nil
}

// ========== Custom Handler Implementations ==========

// handleZhuQueFire is the custom handler for ZhuQue Fire Buff.
// Effect: LP+1 every 4 turns.
// This is Fire Buff's special logic, needs counter to track turns.
func handleZhuQueFire(phase event.Phase, ctx *event.Context) {
	player, ok := ctx.Player.(*core.Player)
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

// ========== Register New Handlers ==========

// RegisterBuffHandler registers a new Buff handler.
// Allows external extension of BuffHandlers registry.
func RegisterBuffHandler(buffType core.BuffType, handler EventHandler) {
	BuffHandlers[buffType] = handler
}