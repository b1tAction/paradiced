// Package constants provides unified enum type definitions.
package constants

// MiniGameMode defines the mini-game execution mode.
type MiniGameMode int

// MiniGameMode constants.
const (
	MiniGameModeFrontend MiniGameMode = 0 // Client submits game_data, server calculates rank
	MiniGameModeRPC      MiniGameMode = 1 // MiniGame Service RPC directly reports rankings
)

// MiniGameType defines the mini-game type identifier.
type MiniGameType string

// MiniGameType constants - snake_case values for JSON serialization.
const (
	MiniGameTypeDiceRace     MiniGameType = "dice_race"     // Dice race - rank by score (dice sum) descending
	MiniGameTypeCoinFlip     MiniGameType = "coin_flip"     // Coin flip - rank by success count descending (未实现，暂不可用)
	MiniGameTypeCountSeconds MiniGameType = "count_seconds" // Count seconds - rank by deviation from 5.0 ascending
)

// IsValid checks if MiniGameType is valid and currently available.
// Unimplemented types (e.g. coin_flip) return false.
func (mt MiniGameType) IsValid() bool {
	switch mt {
	case MiniGameTypeDiceRace, MiniGameTypeCountSeconds:
		return true
	default:
		return false
	}
}

// AllMiniGameTypes is the pool of currently available mini-game types for random selection.
// Only implemented types are included.
var AllMiniGameTypes = []MiniGameType{
	MiniGameTypeDiceRace,
	MiniGameTypeCountSeconds,
}