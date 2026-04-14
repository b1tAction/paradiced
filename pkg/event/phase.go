package event

// Phase defines trigger timing in the game.
// Used for Buff, Item, Faction passive effects trigger phases.
//
// Design principle: Who produces the timing, who publishes the Phase.
// - HSM publishes state timing Phases (BeforeTurn, OnLand, AfterTurn)
// - Action publishes action timing Phases (PreDamage, PreEvent, PreMove, OnBuffApplied, OnBuffRemoved)
type Phase int

const (
	// ========== HSM Published Phases (State Timing) ==========
	// These Phases are published by HSM State.Enter() methods.

	PhaseBeforeTurn Phase = iota // TurnUpkeep.Enter() - Before turn starts (Divine神眷/Curse诅咒 LP±1, Fire离火 every 4 turns)
	PhaseOnLand                  // TurnLanded.Enter() - After landing (landing events, cell effects)
	PhaseAfterTurn               // TurnEnd.Enter() - After turn ends (Rain甘霖/Corrupt腐化 HP±1, TickDuration)

	// ========== Action Published Phases (Action Timing) ==========
	// These Phases are published by ActionContext.ExecuteAction().

	PhasePreDamage    // ActionDamage.Execute() - Before damage application (Hidden隐匿, shields)
	PhasePreEvent     // ActionDrawEvent.Execute() - Before event triggers (Exorcism辟邪, XuanWu)
	PhasePreMove      // ActionMove.Execute() - Before movement (Lost迷途 reverse direction)
	PhasePreRespawn   // ActionRespawn.Execute() - Before respawn (Undying不死 intercept)
	PhaseOnBuffApplied  // ActionAddBuff.Execute() - After buff applied (buff entry effects, chain reactions)
	PhaseOnBuffRemoved  // ActionRemoveBuff.Execute() - Before buff removed (buff death effects/亡语)

	// ========== Special Phases ==========

	PhaseAnyTime  // Any time usable (items active use) - manual trigger by player
	PhaseItemUsed // Item actively used by player - triggered by game.UseItem()
)

// String returns the string representation of Phase.
func (p Phase) String() string {
	names := map[Phase]string{
		PhaseBeforeTurn:    "BeforeTurn",
		PhaseOnLand:        "OnLand",
		PhaseAfterTurn:     "AfterTurn",
		PhasePreDamage:     "PreDamage",
		PhasePreEvent:      "PreEvent",
		PhasePreMove:       "PreMove",
		PhasePreRespawn:    "PreRespawn",
		PhaseOnBuffApplied: "OnBuffApplied",
		PhaseOnBuffRemoved: "OnBuffRemoved",
		PhaseAnyTime:       "AnyTime",
		PhaseItemUsed:      "ItemUsed",
	}
	if name, ok := names[p]; ok {
		return name
	}
	return "Unknown"
}

// IsValid checks if Phase is valid.
func (p Phase) IsValid() bool {
	return p >= PhaseBeforeTurn && p <= PhaseItemUsed
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
	return p == PhasePreDamage || p == PhasePreEvent || p == PhasePreMove || p == PhasePreRespawn ||
		p == PhaseOnBuffApplied || p == PhaseOnBuffRemoved
}