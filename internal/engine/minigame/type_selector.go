// Package minigame provides mini-game type selection, rank calculation, and online provider.
package minigame

import (
	"math/rand"

	"github.com/b1tAction/paradiced/pkg/constants"
)

// SelectMiniGameType randomly picks a mini-game type from the pool.
// Uses provided RNG for deterministic selection (consistent with game replay).
// Returns MiniGameTypeNone when no types are available.
func SelectMiniGameType(rng *rand.Rand) constants.MiniGameType {
	pool := constants.AllMiniGameTypes
	if len(pool) == 0 {
		return constants.MiniGameTypeNone
	}
	idx := rng.Intn(len(pool))
	return pool[idx]
}

// SelectMiniGameTypeWithProvider randomly picks a mini-game type from the pool.
// When an OnlineMiniGameProvider is available, all types (including online types)
// are eligible. When provider is nil, online types are excluded from the pool.
// Uses provided RNG for deterministic selection.
// Returns MiniGameTypeNone when no eligible types are available.
func SelectMiniGameTypeWithProvider(rng *rand.Rand, hasProvider bool) constants.MiniGameType {
	pool := miniGamePool(hasProvider)
	if len(pool) == 0 {
		return constants.MiniGameTypeNone
	}
	idx := rng.Intn(len(pool))
	return pool[idx]
}

// HasAvailableMiniGames checks if there are any eligible mini-game types
// given the provider availability.
func HasAvailableMiniGames(hasProvider bool) bool {
	return len(miniGamePool(hasProvider)) > 0
}

// miniGamePool constructs the mini-game type pool based on provider availability.
// Online types are only included when hasProvider is true.
func miniGamePool(hasProvider bool) []constants.MiniGameType {
	if hasProvider {
		return constants.AllMiniGameTypes // All types including online
	}
	// Filter out online types when no provider is available
	pool := make([]constants.MiniGameType, 0, len(constants.AllMiniGameTypes))
	for _, mt := range constants.AllMiniGameTypes {
		if !mt.IsOnline() {
			pool = append(pool, mt)
		}
	}
	return pool
}

// FrontendMiniGamePool returns available mini-game types that are NOT online.
// These types can run in frontend-driven mode (client submits game_data).
// Used as fallback pool when online room creation fails.
func FrontendMiniGamePool() []constants.MiniGameType {
	pool := make([]constants.MiniGameType, 0, len(constants.AllMiniGameTypes))
	for _, mt := range constants.AllMiniGameTypes {
		if !mt.IsOnline() {
			pool = append(pool, mt)
		}
	}
	return pool
}

// SelectFromPool randomly picks a mini-game type from the given pool.
// Uses provided RNG for deterministic selection.
// Returns MiniGameTypeNone when the pool is empty.
func SelectFromPool(rng *rand.Rand, pool []constants.MiniGameType) constants.MiniGameType {
	if len(pool) == 0 {
		return constants.MiniGameTypeNone
	}
	idx := rng.Intn(len(pool))
	return pool[idx]
}