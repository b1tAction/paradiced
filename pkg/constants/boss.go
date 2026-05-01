// Package constants provides unified enum type definitions.
package constants

// BossType defines boss entity type.
type BossType string

// BossType constants - snake_case values for JSON serialization.
const (
	BossTypeBeast BossType = "beast" // The main boss of the game
)

// IsValid checks if BossType is valid.
func (bt BossType) IsValid() bool {
	switch bt {
	case BossTypeBeast:
		return true
	default:
		return false
	}
}

// BossSkillType defines boss skill type.
type BossSkillType string

// BossSkillType constants.
const (
	BossSkillThunder BossSkillType = "thunder" // AOE damage 2 to all boss-cell players
	BossSkillCurse   BossSkillType = "curse"   // Add curse buff to all boss-cell players
	BossSkillLost    BossSkillType = "lost"    // Add lost buff to all boss-cell players
	BossSkillRest    BossSkillType = "rest"    // Boss heals 5 HP
	BossSkillThorns  BossSkillType = "thorns"  // Add thorns buff to all boss-cell players (reflect 30% damage)
)

// IsValid checks if BossSkillType is valid.
func (bst BossSkillType) IsValid() bool {
	switch bst {
	case BossSkillThunder, BossSkillCurse, BossSkillLost, BossSkillRest, BossSkillThorns:
		return true
	default:
		return false
	}
}

// ParseBossSkillType parses a string into BossSkillType.
func ParseBossSkillType(s string) BossSkillType {
	return BossSkillType(s)
}

// BossAttackType defines boss physical attack type (normal or critical).
// Skill effects are handled by BossSkillAction + derived DamageAction/AddBuffAction,
// not by BossAttackAction.
type BossAttackType string

// BossAttackType constants.
const (
	BossAttackNormal BossAttackType = "normal" // 1 damage to random player
	BossAttackCrit   BossAttackType = "crit"   // 2 damage to random player
)

// IsValid checks if BossAttackType is valid.
func (bat BossAttackType) IsValid() bool {
	switch bat {
	case BossAttackNormal, BossAttackCrit:
		return true
	default:
		return false
	}
}

// BossPlayerUUID is the fixed UUID for the Boss special player.
// Uses a distinctive hex pattern that won't collide with normal player UUIDs (v7).
// Must conform to standard UUID format (8-4-4-4-12 hex chars with dashes).
const BossPlayerUUID = "beeeeeef-beef-beef-beef-beeeeeeeeeef"