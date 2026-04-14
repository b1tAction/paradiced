// Package constants provides unified enum type definitions.
package constants

// SpecialEffect defines special effect types for Buffs/Items/Events.
type SpecialEffect string

// SpecialEffect constants - snake_case values for JSON serialization.
const (
	SpecialNone SpecialEffect = "none"

	// Buff special effects
	SpecialImmune        SpecialEffect = "immune"         // Hidden隐匿: immunity
	SpecialReverse       SpecialEffect = "reverse"        // Lost迷途: reverse movement
	SpecialImmunePoison  SpecialEffect = "immune_poison"  // Exorcism辟邪: immune to poison
	SpecialBadEvent      SpecialEffect = "bad_event"      // Poison毒瘴: bad event each turn
	SpecialZhuQuePassive SpecialEffect = "zhu_que_passive" // Fire离火: ZhuQue passive

	// Item special effects
	SpecialTeleport    SpecialEffect = "teleport"    // AnyDoor任意门: teleport
	SpecialDiceSwap    SpecialEffect = "dice_swap"   // DiceSwap骰子交换: swap dice
	SpecialDiceUpgrade SpecialEffect = "dice_upgrade" // DiceUpgrade骰子升级: upgrade dice
	SpecialGiveLost    SpecialEffect = "give_lost"   // ReverseClock反方向的钟: give Lost

	// Event special effects
	SpecialDrawItem     SpecialEffect = "draw_item"    // Relic圣遗物: draw item
	SpecialLoseItem     SpecialEffect = "lose_item"    // Thief盗贼: lose item
	SpecialSwapPosition SpecialEffect = "swap_position" // Exchange交换: swap position
	SpecialRandomBuff   SpecialEffect = "random_buff"  // TasteTest尝一口: random buff
)

// IsValid checks if SpecialEffect is valid.
func (se SpecialEffect) IsValid() bool {
	return se != SpecialNone && se != ""
}

// IsBuffEffect checks if the special effect is Buff-related.
func (se SpecialEffect) IsBuffEffect() bool {
	return se == SpecialImmune || se == SpecialReverse ||
		se == SpecialImmunePoison || se == SpecialBadEvent ||
		se == SpecialZhuQuePassive
}

// IsItemEffect checks if the special effect is Item-related.
func (se SpecialEffect) IsItemEffect() bool {
	return se == SpecialTeleport || se == SpecialDiceSwap ||
		se == SpecialDiceUpgrade || se == SpecialGiveLost
}

// IsEventEffect checks if the special effect is Event-related.
func (se SpecialEffect) IsEventEffect() bool {
	return se == SpecialDrawItem || se == SpecialLoseItem ||
		se == SpecialSwapPosition || se == SpecialRandomBuff
}