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
	BuffTypeSinking  BuffType = "sinking"  // Sinking沉沦: share negative actions with linked player
	BuffTypeFearless BuffType = "fearless" // Fearless无畏: HP locked at 1 for 3 turns

	// Neutral Buff
	BuffTypeHidden BuffType = "hidden" // Hidden隐匿: immunity

	// Boss Buff (given by Boss skills, not drawn from lottery pool)
	BuffTypeThorns BuffType = "thorns" // Thorns反刺: reflect 30% damage to attacking player

	// Positive Buffs
	BuffTypeDivine      BuffType = "divine"      // Divine神眷: LP+1 per turn
	BuffTypeRain        BuffType = "rain"        // Rain甘霖: HP+1 every 2 turns
	BuffTypeExorcism    BuffType = "exorcism"    // Exorcism辟邪: immune to poison
	BuffTypeFire        BuffType = "fire"        // Fire离火: ZhuQue passive
	BuffTypeEternal     BuffType = "eternal"     // Eternal永恒: share positive actions with linked player
	BuffTypeGoldenBody  BuffType = "golden_body" // GoldenBody金身: damage halved
	BuffTypeWrath       BuffType = "wrath"       // Wrath嗔怒: outgoing damage +1

	// Faction Buffs (faction passive skills, not drawn from lottery pool)
	BuffTypeDominance BuffType = "dominance" // Dominance威势: QingLong faction, double beneficial actions
	BuffTypeRobLuck   BuffType = "rob_luck"  // RobLuck劫运: BaiHu faction, redirect good actions to self
	BuffTypeSuppress  BuffType = "suppress"  // Suppress鎮厄: XuanWu faction, immunity to bad events/buffs

	// Hidden Buffs (internal mechanism, not visible to player/client)
	BuffTypeDeathMark       BuffType = "death_mark"        // DeathMark死亡标记: blocks actions after death
	BuffTypeSavior          BuffType = "savior"            // Savior庇护: block one fatal damage (from NamedBlade item)
	BuffTypeSageProtection  BuffType = "sage_protection"   // SageProtection贤者庇护: respawn in-place (from SageProtection item)
)

// IsValid checks if BuffType is valid.
func (bt BuffType) IsValid() bool {
	switch bt {
	case BuffTypeCurse, BuffTypeLost, BuffTypeCorrupt, BuffTypePoison,
		BuffTypeSinking, BuffTypeFearless,
		BuffTypeHidden, BuffTypeThorns, BuffTypeDivine, BuffTypeRain,
		BuffTypeExorcism, BuffTypeFire, BuffTypeEternal,
		BuffTypeGoldenBody, BuffTypeWrath,
		BuffTypeDeathMark, BuffTypeSavior, BuffTypeSageProtection,
		BuffTypeDominance, BuffTypeRobLuck, BuffTypeSuppress:
		return true
	default:
		return false
	}
}

// IsPositive checks if the Buff is positive (based on effect, not evaluation).
func (bt BuffType) IsPositive() bool {
	return bt == BuffTypeDivine || bt == BuffTypeHidden ||
		bt == BuffTypeRain || bt == BuffTypeExorcism || bt == BuffTypeFire ||
		bt == BuffTypeEternal || bt == BuffTypeGoldenBody || bt == BuffTypeWrath ||
		bt == BuffTypeDominance || bt == BuffTypeSuppress
}

func (bt BuffType) IsNegative() bool {
	return bt == BuffTypeCurse || bt == BuffTypeLost ||
		bt == BuffTypeCorrupt || bt == BuffTypePoison || bt == BuffTypeRobLuck ||
		bt == BuffTypeSinking || bt == BuffTypeFearless
}

// IsHidden checks if the Buff is hidden (internal mechanism, not visible to player).
func (bt BuffType) IsHidden() bool {
	return bt == BuffTypeDeathMark || bt == BuffTypeRobLuck ||
		bt == BuffTypeSavior || bt == BuffTypeSageProtection
}

// IsBoss checks if the Buff is given by Boss skills or game mechanics (not drawn from lottery pool).
func (bt BuffType) IsBoss() bool {
	return bt == BuffTypeThorns || bt == BuffTypeDeathMark
}

// IsFaction checks if the Buff is a faction passive skill (not drawn from lottery pool).
func (bt BuffType) IsFaction() bool {
	return bt == BuffTypeFire || bt == BuffTypeDominance || bt == BuffTypeRobLuck || bt == BuffTypeSuppress
}

// IsItemOnly checks if the Buff can only be triggered by items (not drawn from lottery pool).
func (bt BuffType) IsItemOnly() bool {
	return bt == BuffTypeSinking || bt == BuffTypeEternal ||
		bt == BuffTypeSavior || bt == BuffTypeSageProtection
}

// IsIsolated checks if the Buff should not merge with same-type instances.
// Isolated buffs (Savior, SageProtection) are independent per instance -
// each gets its own EventBus subscription, metadata, and lifecycle.
func (bt BuffType) IsIsolated() bool {
	return bt == BuffTypeSavior || bt == BuffTypeSageProtection
}

// IsDraw checks if the Buff should participate in lottery pool draws.
// Returns false for Boss buffs, hidden buffs, faction passive buffs, and item-only buffs.
func (bt BuffType) IsDraw() bool {
	return !bt.IsBoss() && !bt.IsHidden() && !bt.IsFaction() && !bt.IsItemOnly()
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
	case "dominance":
		return BuffTypeDominance
	case "rob_luck":
		return BuffTypeRobLuck
	case "suppress":
		return BuffTypeSuppress
	case "sinking":
		return BuffTypeSinking
	case "eternal":
		return BuffTypeEternal
	case "fearless":
		return BuffTypeFearless
	case "golden_body":
		return BuffTypeGoldenBody
	case "wrath":
		return BuffTypeWrath
	case "savior":
		return BuffTypeSavior
	case "sage_protection":
		return BuffTypeSageProtection
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