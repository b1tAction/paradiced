// Package constants provides unified enum type definitions.
package constants

// AchievementType defines achievement identifiers.
// Each achievement can only be earned once per game per player.
type AchievementType string

// AchievementType constants - snake_case values for JSON serialization.
const (
	// Dice roll achievements (triggered via PhasePreAction + ActionDiceRoll)
	AchievementTripleOne AchievementType = "triple_one" // 三连一: 3 consecutive dice rolls of 1
	AchievementTripleSix AchievementType = "triple_six" // 三连六: 3 consecutive dice rolls of 6

	// Boss battle achievements (triggered via PhasePreAction + ActionBossDamage or HSM)
	AchievementBossKillShot  AchievementType = "boss_kill_shot"  // K头: dealt the fatal blow to Boss
	AchievementBossDamageTen AchievementType = "boss_damage_ten" // 勇者之路: cumulative boss damage >= 10

	// Item achievements (triggered via PhasePreAction + ActionAddItem)
	AchievementItemCollector AchievementType = "item_collector" // 道具收藏家: held 3+ items simultaneously

	// Game state achievements (triggered by HSM direct detection)
	AchievementSurvivor            AchievementType = "survivor"             // 生存大师: never died during the game
	AchievementLuckMaster          AchievementType = "luck_master"         // 幸运之星: LP reached maximum (8)
	AchievementFirstToBoss         AchievementType = "first_to_boss"       // 先行者: first player to reach Boss cell
	AchievementMiniGameWinnerThree AchievementType = "mini_game_winner_three" // 小游戏之王: won mini-game 3+ times
)

// IsValid checks if AchievementType is valid.
func (at AchievementType) IsValid() bool {
	switch at {
	case AchievementTripleOne, AchievementTripleSix,
		AchievementBossKillShot, AchievementBossDamageTen,
		AchievementItemCollector,
		AchievementSurvivor, AchievementLuckMaster,
		AchievementFirstToBoss, AchievementMiniGameWinnerThree:
		return true
	default:
		return false
	}
}

// ParseAchievementType converts a string to AchievementType.
// Returns AchievementTripleOne for unrecognized values (default, not "none").
func ParseAchievementType(s string) AchievementType {
	switch s {
	case "triple_one":
		return AchievementTripleOne
	case "triple_six":
		return AchievementTripleSix
	case "boss_kill_shot":
		return AchievementBossKillShot
	case "boss_damage_ten":
		return AchievementBossDamageTen
	case "item_collector":
		return AchievementItemCollector
	case "survivor":
		return AchievementSurvivor
	case "luck_master":
		return AchievementLuckMaster
	case "first_to_boss":
		return AchievementFirstToBoss
	case "mini_game_winner_three":
		return AchievementMiniGameWinnerThree
	default:
		return AchievementTripleOne // Default for unrecognized
	}
}

// ========== Achievement Definition (Static Metadata) ==========

// AchievementDefinition contains static metadata for achievement display.
// Trigger logic is managed by engine layer's AchievementHandlerConfig.
type AchievementDefinition struct {
	Type        AchievementType `json:"type"`        // Achievement type identifier
	Name        string          `json:"name"`         // Chinese display name (三连一, K头, etc.)
	Desc        string          `json:"desc"`          // Description text
	Points      int             `json:"points"`        // Bonus score points awarded when earned
	EnglishName string          `json:"english_name"`  // English identifier (triple_one, boss_kill_shot, etc.)
}