// Package rng provides random draw engine for the Paradiced game.
// This package has no internal dependencies - provides weighted random draw.
package rng

import "math/rand"

// ========== Boss Attack Calculation ==========

// BossAttackResult represents the result of a Boss attack decision.
type BossAttackResult struct {
	AttackType string // "normal", "crit", or "skill"
	SkillType  string // Skill type if AttackType is "skill" (empty otherwise)
	Target     string // Target player ID (empty for AOE skills)
	Damage     int    // Damage amount (0 for skills that apply buffs)
}

// Boss attack probability constants.
// Lower average LP → higher Boss crit/skill probability.
// Lower Boss HP → higher Boss skill probability (Boss fights harder when wounded).
const (
	bossCritSkillBaseRate = 0.25  // Base rate when avgLP = lpMax and bossHP = maxHP
	bossCritSkillLpFactor = 0.05 // Additional rate per LP below lpMax
	bossCritSkillHpFactor = 0.30 // Additional rate per HP ratio lost (0~0.30 based on bossHP/maxHP)
)

// CalcBossCritSkillProb calculates the probability of Boss using crit or skill.
// Formula: baseRate + lpFactor * (lpMax - avgLP) + hpFactor * (maxHP - currentHP) / maxHP
// avgLP=8, bossHP=50 → 25%, avgLP=4, bossHP=25 → 60%, avgLP=0, bossHP=0 → 95%
func CalcBossCritSkillProb(avgLP float64, bossCurrentHP int, bossMaxHP int) float64 {
	return bossCritSkillBaseRate +
		bossCritSkillLpFactor*(float64(lpMax)-avgLP) +
		bossCritSkillHpFactor*float64(bossMaxHP-bossCurrentHP)/float64(bossMaxHP)
}

// CalcBossAttackType determines Boss attack type based on average LP and Boss HP.
// Returns BossAttackResult with attack_type: "normal", "crit", or "skill".
// When critOrSkill triggers, 30% chance is crit, 70% chance is skill.
func CalcBossAttackType(r *rand.Rand, avgLP float64, bossCurrentHP int, bossMaxHP int, skillPool []*EvaluatedItem) BossAttackResult {
	prob := CalcBossCritSkillProb(avgLP, bossCurrentHP, bossMaxHP)

	// Normal attack is the default
	if r.Float64() >= prob {
		return BossAttackResult{
			AttackType: "normal",
			Damage:     1,
		}
	}

	// 30% chance crit, 70% chance skill
	if r.Float64() < 0.3 || len(skillPool) == 0 {
		return BossAttackResult{
			AttackType: "crit",
			Damage:     2,
		}
	}

	// Draw a random skill (equal weight)
	engine := NewDrawEngine(r)
	// Convert []*EvaluatedItem to []EvaluatedItem for DrawFromPool
	poolItems := make([]EvaluatedItem, len(skillPool))
	for i, item := range skillPool {
		poolItems[i] = *item
	}
	result := engine.DrawFromPool(&EvaluatedItemPool{Items: poolItems}, PoolTypeNeutral, 0)
	return BossAttackResult{
		AttackType: "skill",
		SkillType:  result,
		Damage:     0, // Damage depends on skill handler
	}
}

// Boss target selection constants.
const (
	bossTargetBaseWeight = 1.0
	bossTargetLpFactor   = 0.3 // Weight increase per LP below lpMax
)

// SelectBossTarget selects a target player for Boss attack.
// LP越低越容易被攻击 (weighted selection).
// Returns the selected player's UUID.
func SelectBossTarget(r *rand.Rand, players []BossTargetCandidate) string {
	if len(players) == 0 {
		return ""
	}
	if len(players) == 1 {
		return players[0].PlayerID
	}

	totalWeight := 0.0
	weights := make([]float64, len(players))
	for i, p := range players {
		weight := bossTargetBaseWeight + bossTargetLpFactor*(float64(lpMax)-float64(p.LP))
		weights[i] = weight
		totalWeight += weight
	}

	roll := r.Float64() * totalWeight
	cumulative := 0.0
	for i, w := range weights {
		cumulative += w
		if roll < cumulative {
			return players[i].PlayerID
		}
	}

	return players[len(players)-1].PlayerID
}

// BossTargetCandidate represents a candidate player for Boss target selection.
type BossTargetCandidate struct {
	PlayerID string
	LP       int
}

// ========== Player Crit Calculation ==========

// Player crit rate constants based on dice quality.
const (
	critRateGoldDice   = 0.30 // 30% crit rate for gold dice
	critRateSilverDice = 0.20 // 20% crit rate for silver dice
	critRateCopperDice = 0.10 // 10% crit rate for copper dice
	critRateWoodDice   = 0.05 // 5% crit rate for wood/normal dice
)

// CalcPlayerCritRate returns the crit rate based on dice type.
func CalcPlayerCritRate(diceType DiceType) float64 {
	switch diceType {
	case DiceTypeGold:
		return critRateGoldDice
	case DiceTypeSilver:
		return critRateSilverDice
	case DiceTypeCopper:
		return critRateCopperDice
	case DiceTypeWood:
		return critRateWoodDice
	default:
		return critRateWoodDice
	}
}

// CalcPlayerCrit determines if the player's attack is a critical hit.
// Returns true if the attack crits, based on dice type probability.
func CalcPlayerCrit(r *rand.Rand, diceType DiceType) bool {
	return r.Float64() < CalcPlayerCritRate(diceType)
}