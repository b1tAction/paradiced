// Package constants provides unified enum type definitions.
package constants

// BuffType defines Buff type identifiers.
type BuffType string

// BuffType constants - snake_case values for JSON serialization.
const (
	BuffTypeNone BuffType = "none"

	// Negative Buffs
	BuffTypeCurse   BuffType = "curse"   // Curse诅咒: LP-1 per turn
	BuffTypeLost    BuffType = "lost"    // Lost迷途: reverse movement
	BuffTypeCorrupt BuffType = "corrupt" // Corrupt腐化: HP-1 every 2 turns
	BuffTypePoison  BuffType = "poison"  // Poison毒瘴: bad event each turn

	// Neutral Buff
	BuffTypeHidden BuffType = "hidden" // Hidden隐匿: immunity

	// Positive Buffs
	BuffTypeDivine   BuffType = "divine"   // Divine神眷: LP+1 per turn
	BuffTypeRain     BuffType = "rain"     // Rain甘霖: HP+1 every 2 turns
	BuffTypeExorcism BuffType = "exorcism" // Exorcism辟邪: immune to poison
	BuffTypeFire     BuffType = "fire"     // Fire离火: ZhuQue passive
)

// IsValid checks if BuffType is valid.
func (bt BuffType) IsValid() bool {
	switch bt {
	case BuffTypeCurse, BuffTypeLost, BuffTypeCorrupt, BuffTypePoison,
		BuffTypeHidden, BuffTypeDivine, BuffTypeRain, BuffTypeExorcism, BuffTypeFire:
		return true
	default:
		return false
	}
}

// IsPositive checks if the Buff is positive (based on effect, not evaluation).
func (bt BuffType) IsPositive() bool {
	return bt == BuffTypeDivine || bt == BuffTypeHidden ||
		bt == BuffTypeRain || bt == BuffTypeExorcism || bt == BuffTypeFire
}

// IsNegative checks if the Buff is negative.
func (bt BuffType) IsNegative() bool {
	return bt == BuffTypeCurse || bt == BuffTypeLost ||
		bt == BuffTypeCorrupt || bt == BuffTypePoison
}