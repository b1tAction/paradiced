// Package minigame provides mini-game type selection and rank calculation.
package minigame

import (
	"sort"

	"github.com/b1tAction/paradiced/pkg/constants"
)

// RankCalculator calculates rankings from raw mini-game data submissions.
type RankCalculator interface {
	// Calculate computes player rankings from submitted game_data.
	// Returns: playerID -> rank (1 = best, N = worst)
	Calculate(gameType constants.MiniGameType, submissions map[string]map[string]interface{}) map[string]int
}

// DefaultRankCalculator sorts by game_type-specific key and assigns ranks.
type DefaultRankCalculator struct{}

// NewDefaultRankCalculator creates a new DefaultRankCalculator.
func NewDefaultRankCalculator() *DefaultRankCalculator {
	return &DefaultRankCalculator{}
}

// Calculate ranks players based on game_type-specific scoring rules.
// - count_seconds: sort by deviation |elapsed - 5.0| ascending (closer to 5s = better rank)
// - vernier: sort by deviation ascending (closer to 0 = better rank)
// - math_calc, rainbow_memory: sort by accuracy descending, then time_ms ascending
// - default: sort by "score" descending
func (c *DefaultRankCalculator) Calculate(gameType constants.MiniGameType, submissions map[string]map[string]interface{}) map[string]int {
	if len(submissions) == 0 {
		return map[string]int{}
	}

	// Build sortable player list
	type entry struct {
		playerID string
		accuracy float64 // for math_calc
		timeMs   float64 // for math_calc
		keyValue float64 // for others
	}

	entries := make([]entry, 0, len(submissions))
	for playerID, data := range submissions {
		if gameType == constants.MiniGameTypeMathCalc || gameType == constants.MiniGameTypeRainbowMemory {
			// multi-key: accuracy descending, then timeMs ascending
			acc := getFloatValue(data, "accuracy")
			t := getFloatValue(data, "time_ms")
			entries = append(entries, entry{playerID: playerID, accuracy: acc, timeMs: t})
		} else if gameType == constants.MiniGameTypeCountSeconds {
			// count_seconds: compute deviation from 5.0 seconds
			elapsed := getFloatValue(data, "elapsed")
			deviation := abs(elapsed - 5.0)
			entries = append(entries, entry{playerID: playerID, keyValue: deviation})
		} else if gameType == constants.MiniGameTypeVernier {
			// vernier: deviation from center (submitted as "deviation" key)
			dev := getFloatValue(data, "deviation")
			entries = append(entries, entry{playerID: playerID, keyValue: dev})
		} else {
			sortKey := getSortKey(gameType)
			val := getFloatValue(data, sortKey)
			entries = append(entries, entry{playerID: playerID, keyValue: val})
		}
	}

	// Sort based on game_type
	switch gameType {
	case constants.MiniGameTypeMathCalc, constants.MiniGameTypeRainbowMemory:
		// Accuracy descending, then timeMs ascending
		sort.SliceStable(entries, func(i, j int) bool {
			if entries[i].accuracy != entries[j].accuracy {
				return entries[i].accuracy > entries[j].accuracy
			}
			return entries[i].timeMs < entries[j].timeMs
		})
	case constants.MiniGameTypeCountSeconds, constants.MiniGameTypeVernier:
		// Lower deviation = better rank (ascending): closer to target is better
		sort.SliceStable(entries, func(i, j int) bool {
			return entries[i].keyValue < entries[j].keyValue
		})
	default:
		// Higher score = better rank (descending): dice_race, coin_flip, and others
		sort.SliceStable(entries, func(i, j int) bool {
			return entries[i].keyValue > entries[j].keyValue
		})
	}

	// Assign ranks 1..N
	ranks := make(map[string]int, len(entries))
	for i, e := range entries {
		ranks[e.playerID] = i + 1
	}

	return ranks
}

// abs returns the absolute value of a float64.
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// getSortKey returns the data key to use for sorting based on game_type.
// All non-special game types sort by "score" descending.
func getSortKey(gameType constants.MiniGameType) string {
	return "score"
}

// getFloatValue extracts a float64 value from game_data map.
// Supports int, float64, and numeric values. Returns 0 if key not found.
func getFloatValue(data map[string]interface{}, key string) float64 {
	val, ok := data[key]
	if !ok {
		return 0
	}
	switch v := val.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case int32:
		return float64(v)
	default:
		return 0
	}
}