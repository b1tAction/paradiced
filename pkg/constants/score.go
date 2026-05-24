// Package constants provides unified enum type definitions.
package constants

// ScoreCategory defines scoring source category.
// Used to classify score points by their origin for ranking and UI rendering.
type ScoreCategory string

// ScoreCategory constants - snake_case values for JSON serialization.
const (
	ScoreCategoryMiniGame    ScoreCategory = "mini_game"    // Mini-game ranking score
	ScoreCategoryBoss        ScoreCategory = "boss"         // Boss battle score (damage + kill shot)
	ScoreCategoryItem        ScoreCategory = "item"         // Item acquisition/usage score
	ScoreCategoryAchievement ScoreCategory = "achievement" // Achievement bonus score
)

// IsValid checks if ScoreCategory is valid.
func (sc ScoreCategory) IsValid() bool {
	switch sc {
	case ScoreCategoryMiniGame, ScoreCategoryBoss, ScoreCategoryItem, ScoreCategoryAchievement:
		return true
	default:
		return false
	}
}

// ========== Score Point Constants ==========

// ScorePoint constants define fixed point values for each scoring event.
const (
	// Mini-game ranking scores (per round)
	ScoreMiniGameRank1 = 10 // 1st place
	ScoreMiniGameRank2 = 7  // 2nd place
	ScoreMiniGameRank3 = 4  // 3rd place
	ScoreMiniGameRank4 = 1  // 4th place

	// Boss battle scores
	ScoreBossKillShot    = 15 // Boss kill shot bonus (defeating the boss)
	ScoreBossDamagePerPt = 1  // 1 score per damage point dealt to boss
	ScoreBossCritBonus   = 2  // Critical hit bonus on boss

	// Item scores
	ScoreItemAcquired = 3 // Item acquired (added to inventory)
	ScoreItemUsed     = 2 // Item used/consumed
	ScoreDiceUpgrade  = 5 // Dice upgrade special bonus
)

// MiniGameRankToScore converts a mini-game ranking to score points.
// Formula: 10 - (rank - 1) * 3, minimum 1 point.
// For N-player games, rank must be between 1 and totalPlayers.
func MiniGameRankToScore(rank int, totalPlayers int) int {
	if rank < 1 || rank > totalPlayers || totalPlayers < 2 {
		return 1 // Fallback minimum score
	}
	score := 10 - (rank - 1) * 3
	if score < 1 {
		return 1
	}
	return score
}

// ========== ScoreReason ==========

// ScoreReason explains why a player received score points.
// Used in GameOver protocol for UI rendering of detailed score breakdown.
type ScoreReason struct {
	Category string `json:"category"` // ScoreCategory as string: "mini_game", "boss", "item", "achievement"
	Reason   string `json:"reason"`   // Human-readable reason (Chinese): "小游戏第1名", "Boss伤害8点", "获得道具", "K头"
	Points   int    `json:"points"`   // Points awarded for this reason
	Round    int    `json:"round"`    // Round number (0 if not round-specific)
}