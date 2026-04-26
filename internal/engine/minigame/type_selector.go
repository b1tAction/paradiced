// Package minigame provides mini-game type selection and rank calculation.
package minigame

import (
	"math/rand"

	"github.com/b1tAction/paradiced/pkg/constants"
)

// SelectMiniGameType randomly picks a mini-game type from the pool.
// Uses provided RNG for deterministic selection (consistent with game replay).
func SelectMiniGameType(rng *rand.Rand) constants.MiniGameType {
	pool := constants.AllMiniGameTypes
	if len(pool) == 0 {
		return constants.MiniGameTypeDiceRace // fallback
	}
	idx := rng.Intn(len(pool))
	return pool[idx]
}