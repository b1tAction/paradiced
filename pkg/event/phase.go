package event

// Phase defines trigger timing in the game.
// Used for Buff, Item, Faction passive effects trigger phases.
type Phase int

const (
	PhaseBeforeTurn Phase = iota // Before turn starts (Divine神眷/Curse诅咒 LP±1, Fire离火 every 4 turns)
	PhaseOnMove                  // During movement (Lost迷途 reverse direction)
	PhaseOnLand                  // After landing (AnyDoor, landing events)
	PhasePreEvent                // Before event triggers (Exorcism辟邪, XuanWu, shield items)
	PhasePreDamage               // Before damage (Hidden隐匿, shields)
	PhaseAfterTurn               // After turn ends (Rain甘霖/Corrupt腐化 HP±1, TickDuration)
	PhaseAnyTime                 // Any time usable (items active use)
	// Event-driven phases - Buff lifecycle events
	PhaseOnBuffApplied           // Triggered when any Buff is applied to a player
	PhaseOnBuffRemoved           // Triggered when any Buff is removed/expired from a player
)

// String returns the string representation of Phase.
func (p Phase) String() string {
	names := map[Phase]string{
		PhaseBeforeTurn:     "BeforeTurn",
		PhaseOnMove:         "OnMove",
		PhaseOnLand:         "OnLand",
		PhasePreEvent:       "PreEvent",
		PhasePreDamage:      "PreDamage",
		PhaseAfterTurn:      "AfterTurn",
		PhaseAnyTime:        "AnyTime",
		PhaseOnBuffApplied:  "OnBuffApplied",
		PhaseOnBuffRemoved:  "OnBuffRemoved",
	}
	if name, ok := names[p]; ok {
		return name
	}
	return "Unknown"
}

// IsValid checks if Phase is valid.
func (p Phase) IsValid() bool {
	return p >= PhaseBeforeTurn && p <= PhaseOnBuffRemoved
}

// NeedsSubscription determines if the Phase needs EventBus subscription.
// AnyTime type doesn't subscribe, requires manual trigger by player!!!
// PhaseOnBuffApplied and PhaseOnBuffRemoved need subscription to listen Buff lifecycle events.
func (p Phase) NeedsSubscription() bool {
	return p != PhaseAnyTime
}