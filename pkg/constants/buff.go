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

	// Boss Buff (given by Boss skills, not drawn from lottery pool)
	BuffTypeThorns BuffType = "thorns" // Thorns反刺: reflect 30% damage to attacking player

	// Positive Buffs
	BuffTypeDivine   BuffType = "divine"   // Divine神眷: LP+1 per turn
	BuffTypeRain     BuffType = "rain"     // Rain甘霖: HP+1 every 2 turns
	BuffTypeExorcism BuffType = "exorcism" // Exorcism辟邪: immune to poison
	BuffTypeFire     BuffType = "fire"     // Fire离火: ZhuQue passive

	// Hidden Buffs (internal mechanism, not visible to player/client)
	BuffTypeDeathMark BuffType = "death_mark" // DeathMark死亡标记: blocks actions after death
)

// IsValid checks if BuffType is valid.
func (bt BuffType) IsValid() bool {
	switch bt {
	case BuffTypeCurse, BuffTypeLost, BuffTypeCorrupt, BuffTypePoison,
		BuffTypeHidden, BuffTypeThorns, BuffTypeDivine, BuffTypeRain,
		BuffTypeExorcism, BuffTypeFire, BuffTypeDeathMark:
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

func (bt BuffType) IsNegative() bool {
	return bt == BuffTypeCurse || bt == BuffTypeLost ||
		bt == BuffTypeCorrupt || bt == BuffTypePoison
}

// IsHidden checks if the Buff is hidden (internal mechanism, not visible to player).
func (bt BuffType) IsHidden() bool {
	return bt == BuffTypeDeathMark
}

// IsBoss checks if the Buff is given by Boss skills or game mechanics (not drawn from lottery pool).
func (bt BuffType) IsBoss() bool {
	return bt == BuffTypeThorns || bt == BuffTypeDeathMark
}

// IsFaction checks if the Buff is a faction passive skill (not drawn from lottery pool).
func (bt BuffType) IsFaction() bool {
	return bt == BuffTypeFire
}

// IsDraw checks if the Buff should participate in lottery pool draws.
// Returns false for Boss buffs, hidden buffs, and faction passive buffs.
func (bt BuffType) IsDraw() bool {
	return !bt.IsBoss() && !bt.IsHidden() && !bt.IsFaction()
}

// ParseBuffType converts a string to BuffType.
// Returns BuffTypeNone if the string is not a valid buff type.
func ParseBuffType(s string) BuffType {
	switch s {
	case "curse":
		return BuffTypeCurse
	case "lost":
		return BuffTypeLost
	case "corrupt":
		return BuffTypeCorrupt
	case "poison":
		return BuffTypePoison
	case "hidden":
		return BuffTypeHidden
	case "thorns":
		return BuffTypeThorns
	case "divine":
		return BuffTypeDivine
	case "rain":
		return BuffTypeRain
	case "exorcism":
		return BuffTypeExorcism
	case "fire":
		return BuffTypeFire
	case "death_mark":
		return BuffTypeDeathMark
	default:
		return BuffTypeNone
	}
}

// ========== Buff Definition (Static Metadata) ==========

// BuffDefinition contains static metadata for Buff display and classification.
// Effect logic is managed by engine layer's BuffHandlerConfig.
type BuffDefinition struct {
	Type        BuffType   `json:"type"`
	Eval        Evaluation `json:"evaluation"`    // Evaluation score for random draw
	EnglishName string     `json:"english_name"`  // English identifier (snake_case)
	Name        string     `json:"name"`          // Chinese display name
	Desc        string     `json:"desc"`          // Description text
	Duration    int        `json:"duration"`      // Default duration (-1 for permanent)
}

// IsPositive checks if the buff is beneficial.
func (d *BuffDefinition) IsPositive() bool {
	return d.Eval.IsGood()
}

// IsNegative checks if the buff is harmful.
func (d *BuffDefinition) IsNegative() bool {
	return d.Eval.IsBad()
}