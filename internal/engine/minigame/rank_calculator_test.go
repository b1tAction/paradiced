package minigame

import (
	"testing"

	"github.com/b1tAction/paradiced/pkg/constants"
)

func TestDefaultRankCalculator_DiceRace(t *testing.T) {
	calc := NewDefaultRankCalculator()

	submissions := map[string]map[string]interface{}{
		"p1": {"dice1": 3, "dice2": 4, "score": 7},
		"p2": {"dice1": 5, "dice2": 6, "score": 11},
		"p3": {"dice1": 2, "dice2": 1, "score": 3},
	}

	ranks := calc.Calculate(constants.MiniGameTypeDiceRace, submissions)

	// dice_race: higher score (dice sum) = better rank (descending)
	// p2 (score=11) → rank 1, p1 (score=7) → rank 2, p3 (score=3) → rank 3
	if ranks["p2"] != 1 {
		t.Errorf("p2 rank = %d, want 1 (highest score=11)", ranks["p2"])
	}
	if ranks["p1"] != 2 {
		t.Errorf("p1 rank = %d, want 2 (score=7)", ranks["p1"])
	}
	if ranks["p3"] != 3 {
		t.Errorf("p3 rank = %d, want 3 (lowest score=3)", ranks["p3"])
	}
}

func TestDefaultRankCalculator_CoinFlip(t *testing.T) {
	calc := NewDefaultRankCalculator()

	submissions := map[string]map[string]interface{}{
		"p1": {"score": 50},
		"p2": {"score": 100},
		"p3": {"score": 25},
	}

	ranks := calc.Calculate(constants.MiniGameTypeCoinFlip, submissions)

	// coin_flip: higher score = better rank (default descending)
	// p2 (100) → rank 1, p1 (50) → rank 2, p3 (25) → rank 3
	if ranks["p2"] != 1 {
		t.Errorf("p2 rank = %d, want 1 (highest score)", ranks["p2"])
	}
	if ranks["p1"] != 2 {
		t.Errorf("p1 rank = %d, want 2", ranks["p1"])
	}
	if ranks["p3"] != 3 {
		t.Errorf("p3 rank = %d, want 3 (lowest score)", ranks["p3"])
	}
}

func TestDefaultRankCalculator_EmptySubmissions(t *testing.T) {
	calc := NewDefaultRankCalculator()

	ranks := calc.Calculate(constants.MiniGameTypeDiceRace, map[string]map[string]interface{}{})
	if len(ranks) != 0 {
		t.Errorf("Expected empty ranks for empty submissions, got %d entries", len(ranks))
	}
}

func TestDefaultRankCalculator_MissingSortKey(t *testing.T) {
	calc := NewDefaultRankCalculator()

	// Submission missing "score" key for dice_race
	submissions := map[string]map[string]interface{}{
		"p1": {"dice1": 3, "dice2": 4}, // no "score" key → value defaults to 0
		"p2": {"dice1": 5, "dice2": 6, "score": 11},
	}

	ranks := calc.Calculate(constants.MiniGameTypeDiceRace, submissions)

	// p2 (score=11) → rank 1, p1 (score defaults to 0) → rank 2
	// Missing key defaults to 0, which is worst for descending sort
	if ranks["p2"] != 1 {
		t.Errorf("p2 rank = %d, want 1 (score=11)", ranks["p2"])
	}
	if ranks["p1"] != 2 {
		t.Errorf("p1 rank = %d, want 2 (score defaults to 0)", ranks["p1"])
	}
}

func TestDefaultRankCalculator_IntValues(t *testing.T) {
	calc := NewDefaultRankCalculator()

	// Test that int values are handled correctly
	submissions := map[string]map[string]interface{}{
		"p1": {"score": 42},     // int
		"p2": {"score": 100.0},  // float64
	}

	ranks := calc.Calculate(constants.MiniGameTypeCoinFlip, submissions)

	if ranks["p2"] != 1 {
		t.Errorf("p2 rank = %d, want 1 (score=100)", ranks["p2"])
	}
	if ranks["p1"] != 2 {
		t.Errorf("p1 rank = %d, want 2 (score=42)", ranks["p1"])
	}
}

func TestDefaultRankCalculator_CountSeconds(t *testing.T) {
	calc := NewDefaultRankCalculator()

	submissions := map[string]map[string]interface{}{
		"p1": {"elapsed": 5.0},  // deviation = 0.0 (perfect)
		"p2": {"elapsed": 4.5},  // deviation = 0.5
		"p3": {"elapsed": 6.2},  // deviation = 1.2
	}

	ranks := calc.Calculate(constants.MiniGameTypeCountSeconds, submissions)

	// count_seconds: lower deviation = better rank
	// p1 (dev=0.0) → rank 1, p2 (dev=0.5) → rank 2, p3 (dev=1.2) → rank 3
	if ranks["p1"] != 1 {
		t.Errorf("p1 rank = %d, want 1 (deviation=0.0)", ranks["p1"])
	}
	if ranks["p2"] != 2 {
		t.Errorf("p2 rank = %d, want 2 (deviation=0.5)", ranks["p2"])
	}
	if ranks["p3"] != 3 {
		t.Errorf("p3 rank = %d, want 3 (deviation=1.2)", ranks["p3"])
	}
}

func TestDefaultRankCalculator_CountSeconds_OverestimateVsUnderestimate(t *testing.T) {
	calc := NewDefaultRankCalculator()

	// Same deviation from both sides of 5.0
	submissions := map[string]map[string]interface{}{
		"p1": {"elapsed": 4.0}, // deviation = 1.0 (underestimated)
		"p2": {"elapsed": 6.0}, // deviation = 1.0 (overestimated)
		"p3": {"elapsed": 3.0}, // deviation = 2.0
	}

	ranks := calc.Calculate(constants.MiniGameTypeCountSeconds, submissions)

	// p1 and p2 have same deviation (1.0), p3 has deviation 2.0
	// p1/p2 should share top ranks (1,2), p3 should be rank 3
	if ranks["p3"] != 3 {
		t.Errorf("p3 rank = %d, want 3 (deviation=2.0)", ranks["p3"])
	}
	// p1 and p2 should be rank 1 and 2 (stable sort, order depends on map iteration)
	totalTop := ranks["p1"] + ranks["p2"]
	if totalTop != 3 { // 1+2
		t.Errorf("p1+p2 rank sum = %d, want 3 (both deviation=1.0)", totalTop)
	}
}

func TestDefaultRankCalculator_CountSeconds_MissingElapsed(t *testing.T) {
	calc := NewDefaultRankCalculator()

	// Submission missing "elapsed" key
	submissions := map[string]map[string]interface{}{
		"p1": {"elapsed": 5.3}, // deviation = 0.3
		"p2": {"score": 100},   // no "elapsed" → defaults to 0, deviation = 5.0
	}

	ranks := calc.Calculate(constants.MiniGameTypeCountSeconds, submissions)

	// p1 (dev=0.3) → rank 1, p2 (elapsed defaults to 0, dev=5.0) → rank 2
	if ranks["p1"] != 1 {
		t.Errorf("p1 rank = %d, want 1 (deviation=0.3)", ranks["p1"])
	}
	if ranks["p2"] != 2 {
		t.Errorf("p2 rank = %d, want 2 (deviation=5.0)", ranks["p2"])
	}
}

func TestDefaultRankCalculator_StableSort(t *testing.T) {
	calc := NewDefaultRankCalculator()

	// Same score values - stable sort should preserve insertion order
	submissions := map[string]map[string]interface{}{
		"p1": {"score": 50.0},
		"p2": {"score": 50.0},
		"p3": {"score": 50.0},
	}

	ranks := calc.Calculate(constants.MiniGameTypeCoinFlip, submissions)

	// All same score, ranks should be 1, 2, 3 (stable, deterministic)
	total := 0
	for _, r := range ranks {
		total += r
	}
	if total != 6 { // 1+2+3
		t.Errorf("Expected total rank sum = 6 for 3 tied players, got %d", total)
	}
}

func TestGetFloatValueAllTypes(t *testing.T) {
	tests := []struct {
		name     string
		data     map[string]interface{}
		key      string
		expected float64
	}{
		{"missing key", map[string]interface{}{"other": 1}, "score", 0},
		{"float64", map[string]interface{}{"score": 42.5}, "score", 42.5},
		{"float32", map[string]interface{}{"score": float32(3.14)}, "score", float64(float32(3.14))},
		{"int", map[string]interface{}{"score": 7}, "score", 7.0},
		{"int64", map[string]interface{}{"score": int64(100)}, "score", 100.0},
		{"int32", map[string]interface{}{"score": int32(9)}, "score", 9.0},
		{"unsupported type", map[string]interface{}{"score": "hello"}, "score", 0},
	}

	for _, tt := range tests {
		result := getFloatValue(tt.data, tt.key)
		if result != tt.expected {
			t.Errorf("%s: getFloatValue = %v, want %v", tt.name, result, tt.expected)
		}
	}
}