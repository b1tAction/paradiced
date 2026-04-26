package minigame

import (
	"math/rand"
	"testing"

	"github.com/b1tAction/paradiced/pkg/constants"
)

func TestSelectMiniGameType(t *testing.T) {
	rng := rand.New(rand.NewSource(42))

	// Run multiple selections to verify randomness
	results := make(map[constants.MiniGameType]bool)
	for i := 0; i < 100; i++ {
		gameType := SelectMiniGameType(rng)
		if !gameType.IsValid() {
			t.Errorf("SelectMiniGameType returned invalid type: %s", gameType)
		}
		results[gameType] = true
	}

	// With 2 types and 100 selections, both should appear
	if len(results) < 2 {
		t.Errorf("Expected both game types to appear in 100 selections, got %d types", len(results))
	}
}

func TestSelectMiniGameTypeDeterministic(t *testing.T) {
	// Same seed should produce same sequence
	rng1 := rand.New(rand.NewSource(123))
	rng2 := rand.New(rand.NewSource(123))

	for i := 0; i < 50; i++ {
		result1 := SelectMiniGameType(rng1)
		result2 := SelectMiniGameType(rng2)
		if result1 != result2 {
			t.Errorf("Same seed produced different results at iteration %d: %s vs %s", i, result1, result2)
		}
	}
}

func TestSelectMiniGameTypeFallback(t *testing.T) {
	// Temporarily empty pool and restore
	origPool := constants.AllMiniGameTypes
	constants.AllMiniGameTypes = []constants.MiniGameType{}
	defer func() { constants.AllMiniGameTypes = origPool }()

	rng := rand.New(rand.NewSource(42))
	result := SelectMiniGameType(rng)
	if result != constants.MiniGameTypeDiceRace {
		t.Errorf("Expected fallback to dice_race when pool empty, got %s", result)
	}
}