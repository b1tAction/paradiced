package engine

import (
	"github.com/b1tAction/paradiced/pkg/constants"
)

// ScoreRegistry manages score definitions and provides score calculation helpers.
// This is a lightweight registry that centralizes score point constants and formulas.
type ScoreRegistry struct {
	// ScoreCategory definitions are stored as constants in pkg/constants/score.go
	// No runtime registration needed — all values are compile-time constants.
}

// GlobalScoreRegistry is the global score registry instance.
var GlobalScoreRegistry = NewScoreRegistry()

// NewScoreRegistry creates a new ScoreRegistry.
func NewScoreRegistry() *ScoreRegistry {
	return &ScoreRegistry{}
}

// MiniGameRankToScore converts a mini-game ranking to score points.
// Delegates to constants.MiniGameRankToScore.
func (sr *ScoreRegistry) MiniGameRankToScore(rank int, totalPlayers int) int {
	return constants.MiniGameRankToScore(rank, totalPlayers)
}

// BossDamageToScore converts boss damage amount to boss score points.
// Formula: damage * ScoreBossDamagePerPt
func (sr *ScoreRegistry) BossDamageToScore(damage int) int {
	return damage * constants.ScoreBossDamagePerPt
}

// BossCritBonusScore returns the bonus score for a critical hit on boss.
func (sr *ScoreRegistry) BossCritBonusScore() int {
	return constants.ScoreBossCritBonus
}

// BossKillShotScore returns the score for defeating the boss.
func (sr *ScoreRegistry) BossKillShotScore() int {
	return constants.ScoreBossKillShot
}

// ItemAcquiredScore returns the score for acquiring an item.
func (sr *ScoreRegistry) ItemAcquiredScore() int {
	return constants.ScoreItemAcquired
}

// ItemUsedScore returns the score for using/consuming an item.
func (sr *ScoreRegistry) ItemUsedScore() int {
	return constants.ScoreItemUsed
}

// DiceUpgradeScore returns the score for upgrading dice type.
func (sr *ScoreRegistry) DiceUpgradeScore() int {
	return constants.ScoreDiceUpgrade
}

// AchievementScore returns the score points for a given achievement type.
// Returns 0 for unknown achievement types.
// Note: Uses GlobalAchievementRegistry directly (same package, no cycle issue).
func (sr *ScoreRegistry) AchievementScore(achievementType constants.AchievementType) int {
	if GlobalAchievementRegistry == nil {
		return 0
	}
	def := GlobalAchievementRegistry.GetDefinition(achievementType)
	if def != nil {
		return def.Points
	}
	return 0
}