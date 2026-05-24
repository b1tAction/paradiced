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
		if gameType == constants.MiniGameTypeNone {
			t.Errorf("SelectMiniGameType returned MiniGameTypeNone (pool should not be empty)")
		}
		if !gameType.IsValid() {
			t.Errorf("SelectMiniGameType returned invalid type: %s", gameType)
		}
		results[gameType] = true
	}

	// With current pool size and 100 selections, at least 1 valid type should appear
	if len(results) < 1 {
		t.Errorf("Expected at least 1 game type to appear in 100 selections, got %d types", len(results))
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

func TestSelectMiniGameTypeEmptyPool(t *testing.T) {
	// Temporarily empty pool and restore
	origPool := constants.AllMiniGameTypes
	constants.AllMiniGameTypes = []constants.MiniGameType{}
	defer func() { constants.AllMiniGameTypes = origPool }()

	rng := rand.New(rand.NewSource(42))
	result := SelectMiniGameType(rng)
	if result != constants.MiniGameTypeNone {
		t.Errorf("Expected MiniGameTypeNone when pool empty, got %s", result)
	}
}

func TestSelectMiniGameTypeWithProviderEmptyPool(t *testing.T) {
	// When all available types are online and no provider is available,
	// the filtered pool should be empty and return MiniGameTypeNone.
	origPool := constants.AllMiniGameTypes
	constants.AllMiniGameTypes = []constants.MiniGameType{constants.MiniGameTypeDilemmaRace}
	defer func() { constants.AllMiniGameTypes = origPool }()

	rng := rand.New(rand.NewSource(42))
	result := SelectMiniGameTypeWithProvider(rng, false)
	if result != constants.MiniGameTypeNone {
		t.Errorf("Expected MiniGameTypeNone when all pool types are online and hasProvider=false, got %s", result)
	}

	// With provider, same pool should return dilemma_race
	resultWithProvider := SelectMiniGameTypeWithProvider(rng, true)
	if resultWithProvider != constants.MiniGameTypeDilemmaRace {
		t.Errorf("Expected dilemma_race with provider, got %s", resultWithProvider)
	}
}

func TestHasAvailableMiniGames(t *testing.T) {
	origPool := constants.AllMiniGameTypes
	defer func() { constants.AllMiniGameTypes = origPool }()

	// Current pool (has at least dilemma_race) should have available games
	if !HasAvailableMiniGames(true) {
		t.Error("HasAvailableMiniGames(true) should be true with current pool")
	}

	// Empty pool - no games available
	constants.AllMiniGameTypes = []constants.MiniGameType{}
	if HasAvailableMiniGames(true) {
		t.Error("HasAvailableMiniGames(true) should be false with empty pool")
	}
	if HasAvailableMiniGames(false) {
		t.Error("HasAvailableMiniGames(false) should be false with empty pool")
	}

	// Only online types - available with provider, not available without
	constants.AllMiniGameTypes = []constants.MiniGameType{constants.MiniGameTypeDilemmaRace}
	if !HasAvailableMiniGames(true) {
		t.Error("HasAvailableMiniGames(true) should be true with online-only pool")
	}
	if HasAvailableMiniGames(false) {
		t.Error("HasAvailableMiniGames(false) should be false with online-only pool")
	}
}

func TestFrontendMiniGamePool(t *testing.T) {
	origPool := constants.AllMiniGameTypes
	defer func() { constants.AllMiniGameTypes = origPool }()

	// Online-only pool: FrontendMiniGamePool should be empty
	constants.AllMiniGameTypes = []constants.MiniGameType{constants.MiniGameTypeDilemmaRace}
	frontendPool := FrontendMiniGamePool()
	if len(frontendPool) != 0 {
		t.Errorf("FrontendMiniGamePool should be empty with online-only pool, got %d types", len(frontendPool))
	}

	// Mixed pool: should contain only non-online types
	constants.AllMiniGameTypes = []constants.MiniGameType{constants.MiniGameTypeDiceRace, constants.MiniGameTypeDilemmaRace}
	frontendPool = FrontendMiniGamePool()
	if len(frontendPool) != 1 {
		t.Errorf("FrontendMiniGamePool should have 1 type with mixed pool, got %d types", len(frontendPool))
	}
	if frontendPool[0] != constants.MiniGameTypeDiceRace {
		t.Errorf("FrontendMiniGamePool should contain dice_race, got %s", frontendPool[0])
	}

	// Frontend-only pool: should contain all types
	constants.AllMiniGameTypes = []constants.MiniGameType{constants.MiniGameTypeDiceRace, constants.MiniGameTypeCountSeconds}
	frontendPool = FrontendMiniGamePool()
	if len(frontendPool) != 2 {
		t.Errorf("FrontendMiniGamePool should have 2 types with frontend-only pool, got %d types", len(frontendPool))
	}

	// Empty pool: should be empty
	constants.AllMiniGameTypes = []constants.MiniGameType{}
	frontendPool = FrontendMiniGamePool()
	if len(frontendPool) != 0 {
		t.Errorf("FrontendMiniGamePool should be empty with empty pool, got %d types", len(frontendPool))
	}
}

func TestSelectFromPool(t *testing.T) {
	rng := rand.New(rand.NewSource(42))

	// Empty pool should return MiniGameTypeNone
	result := SelectFromPool(rng, []constants.MiniGameType{})
	if result != constants.MiniGameTypeNone {
		t.Errorf("SelectFromPool with empty pool should return MiniGameTypeNone, got %s", result)
	}

	// Single-element pool should always return that element
	pool := []constants.MiniGameType{constants.MiniGameTypeDiceRace}
	for i := 0; i < 20; i++ {
		result = SelectFromPool(rng, pool)
		if result != constants.MiniGameTypeDiceRace {
			t.Errorf("SelectFromPool with single-element pool should return dice_race, got %s", result)
		}
	}

	// Multi-element pool deterministic selection
	rng1 := rand.New(rand.NewSource(99))
	rng2 := rand.New(rand.NewSource(99))
	multiPool := []constants.MiniGameType{constants.MiniGameTypeDiceRace, constants.MiniGameTypeCountSeconds, constants.MiniGameTypeMathCalc}
	for i := 0; i < 50; i++ {
		result1 := SelectFromPool(rng1, multiPool)
		result2 := SelectFromPool(rng2, multiPool)
		if result1 != result2 {
			t.Errorf("Same seed produced different SelectFromPool results at iteration %d: %s vs %s", i, result1, result2)
		}
	}
}