// Package engine provides game engine logic for the Paradiced game.
package engine

import "github.com/b1tAction/paradiced/pkg/util"

// RoundData key constants for round-level persistent data.
// These keys are stored in Game.RoundData and cleared each round.
// Internal-only, not exposed to pkg/constants.
const (
	// Mini-game results: map[string]int (playerID -> rank)
	KeyMiniGameRanks = "mini_game_ranks"

	// Dice type assignments: map[string]int (playerID -> DiceType value)
	KeyDiceTypes = "dice_types"
)

// GetMiniGameRanks retrieves the mini-game ranks map from RoundData.
// Returns a new empty map if not found.
func GetMiniGameRanks(roundData *util.Metadata) map[string]int {
	return util.GetMapOrDefault(roundData, KeyMiniGameRanks, make(map[string]int))
}

// SetMiniGameRanks stores the mini-game ranks map in RoundData.
func SetMiniGameRanks(roundData *util.Metadata, ranks map[string]int) {
	util.SetMap(roundData, KeyMiniGameRanks, ranks)
}

// GetDiceTypes retrieves the dice types map from RoundData.
// Returns a new empty map if not found.
func GetDiceTypes(roundData *util.Metadata) map[string]int {
	return util.GetMapOrDefault(roundData, KeyDiceTypes, make(map[string]int))
}

// SetDiceTypes stores the dice types map in RoundData.
func SetDiceTypes(roundData *util.Metadata, types map[string]int) {
	util.SetMap(roundData, KeyDiceTypes, types)
}