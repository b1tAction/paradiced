// Package constants provides unified enum type definitions.
// All enums use string type with snake_case values for JSON compatibility.
package constants

// Phase defines trigger timing in the game.
// Used for Buff, Item, Faction passive effects trigger phases.
//
// Design principle: Who produces the timing, who publishes the Phase.
// - HSM publishes state timing Phases (BeforeTurn, OnLand, AfterTurn)
// - Action publishes action timing Phases (PreDamage, PreEvent, PreMove, etc.)
type Phase string

// Phase constants - snake_case values for JSON serialization.
const (
	// ========== HSM Published Phases (State Timing) ==========

	PhaseBeforeTurn Phase = "before_turn" // TurnUpkeep.Enter() - Before turn starts
	PhaseOnLand     Phase = "on_land"     // TurnLanded.Enter() - After landing
	PhaseAfterTurn  Phase = "after_turn"  // TurnEnd.Enter() - After turn ends

	// ========== Action Published Phases (Action Timing) ==========

	PhasePreDamage       Phase = "pre_damage"        // Before damage application
	PhasePreEvent        Phase = "pre_event"         // Before event triggers
	PhasePreMove         Phase = "pre_move"          // Before movement
	PhasePreRespawn      Phase = "pre_respawn"       // Before respawn (interceptable)
	PhasePreBuffApplied  Phase = "pre_buff_applied"  // Before buff applied
	PhasePostBuffApplied Phase = "post_buff_applied" // After buff applied
	PhasePreBuffRemoved  Phase = "pre_buff_removed"  // Before buff removed
	PhasePostBuffRemoved Phase = "post_buff_removed" // After buff removed
	PhasePreAction       Phase = "pre_action"        // Before any action execution (death mark interception)
	PhasePreDiceRoll     Phase = "pre_dice_roll"     // Before dice roll result (interceptable, Buff can modify Steps)

	// ========== Special Phases ==========

	PhaseAnyTime  Phase = "any_time"  // Any time usable (manual trigger)
	PhaseItemUsed Phase = "item_used" // Item actively used
)

// IsValid checks if Phase is valid.
func (p Phase) IsValid() bool {
	return p == PhaseBeforeTurn || p == PhaseOnLand || p == PhaseAfterTurn ||
		p == PhasePreDamage || p == PhasePreEvent || p == PhasePreMove ||
		p == PhasePreRespawn || p == PhasePreBuffApplied || p == PhasePostBuffApplied ||
		p == PhasePreBuffRemoved || p == PhasePostBuffRemoved ||
		p == PhasePreAction || p == PhasePreDiceRoll || p == PhaseAnyTime || p == PhaseItemUsed
}

// NeedsSubscription determines if the Phase needs EventBus subscription.
// AnyTime type doesn't subscribe, requires manual trigger by player.
func (p Phase) NeedsSubscription() bool {
	return p != PhaseAnyTime
}

// IsHSMPublished returns true if this Phase should be published by HSM states.
func (p Phase) IsHSMPublished() bool {
	return p == PhaseBeforeTurn || p == PhaseOnLand || p == PhaseAfterTurn
}

// IsActionPublished returns true if this Phase should be published by Action execution.
func (p Phase) IsActionPublished() bool {
	return p == PhasePreDamage || p == PhasePreEvent || p == PhasePreMove ||
		p == PhasePreRespawn || p == PhasePreBuffApplied || p == PhasePostBuffApplied ||
		p == PhasePreBuffRemoved || p == PhasePostBuffRemoved || p == PhasePreAction ||
		p == PhasePreDiceRoll
}
