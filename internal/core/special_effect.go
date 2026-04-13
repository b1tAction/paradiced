package core

// SpecialEffect represents special effect types for Buffs/Items/Events.
// Used to identify special behaviors beyond simple HP/LP changes.
type SpecialEffect int

const (
	SpecialNone SpecialEffect = iota

	// Buff special effects
	SpecialImmune        // Hidden隐匿: immunity to damage/events
	SpecialReverse       // Lost迷途: reverse movement direction
	SpecialImmunePoison  // Exorcism辟邪: immunity to poison buff
	SpecialBadEvent      // Poison毒瘴: bad event each turn
	SpecialZhuQuePassive // Fire离火: ZhuQue faction passive (LP+1 every 4 turns)

	// Item special effects (for future use)
	SpecialTeleport      // AnyDoor任意门: teleport to target
	SpecialDiceSwap      // DiceSwap骰子交换: swap dice with target
	SpecialDiceUpgrade   // DiceUpgrade骰子升级: upgrade dice level
	SpecialGiveLost      // ReverseClock反方向的钟: give Lost buff to target

	// Event special effects (for future use)
	SpecialDrawItem      // Relic圣遗物: draw random item
	SpecialLoseItem      // Thief盗贼: lose random item
	SpecialSwapPosition  // Exchange交换: swap position with random player
	SpecialRandomBuff    // TasteTest尝一口: random Corrupt/Rain buff
)

// String returns the special effect name.
func (se SpecialEffect) String() string {
	names := map[SpecialEffect]string{
		SpecialNone:          "None",
		SpecialImmune:        "Immune",
		SpecialReverse:       "Reverse",
		SpecialImmunePoison:  "ImmunePoison",
		SpecialBadEvent:      "BadEvent",
		SpecialZhuQuePassive: "ZhuQuePassive",
		SpecialTeleport:      "Teleport",
		SpecialDiceSwap:      "DiceSwap",
		SpecialDiceUpgrade:   "DiceUpgrade",
		SpecialGiveLost:      "GiveLost",
		SpecialDrawItem:      "DrawItem",
		SpecialLoseItem:      "LoseItem",
		SpecialSwapPosition:  "SwapPosition",
		SpecialRandomBuff:    "RandomBuff",
	}
	if name, ok := names[se]; ok {
		return name
	}
	return "Unknown"
}

// IsValid checks if the special effect is valid.
func (se SpecialEffect) IsValid() bool {
	return se >= SpecialNone && se <= SpecialRandomBuff
}

// IsBuffEffect checks if the special effect is Buff-related.
func (se SpecialEffect) IsBuffEffect() bool {
	return se >= SpecialImmune && se <= SpecialZhuQuePassive
}

// IsItemEffect checks if the special effect is Item-related.
func (se SpecialEffect) IsItemEffect() bool {
	return se >= SpecialTeleport && se <= SpecialGiveLost
}

// IsEventEffect checks if the special effect is Event-related.
func (se SpecialEffect) IsEventEffect() bool {
	return se >= SpecialDrawItem && se <= SpecialRandomBuff
}