// Package constants provides unified enum type definitions.
package constants

// MiniGameMode defines the mini-game execution mode.
type MiniGameMode int

// MiniGameMode constants.
const (
	MiniGameModeFrontend MiniGameMode = 0 // Client submits game_data, server calculates rank
	MiniGameModeRPC      MiniGameMode = 1 // MiniGame Service RPC directly reports rankings
)

// ParseMiniGameMode converts a string to MiniGameMode.
// Returns MiniGameModeFrontend for unrecognized values.
func ParseMiniGameMode(s string) MiniGameMode {
	switch s {
	case "frontend":
		return MiniGameModeFrontend
	case "online":
		return MiniGameModeRPC
	default:
		return MiniGameModeFrontend
	}
}

// MiniGameType defines the mini-game type identifier.
type MiniGameType string

// MiniGameType constants - snake_case values for JSON serialization.
const (
	MiniGameTypeNone         MiniGameType = "none"          // Sentinel value for unknown types
	MiniGameTypeDiceRace     MiniGameType = "dice_race"     // Dice race - rank by score (dice sum) descending
	MiniGameTypeCoinFlip     MiniGameType = "coin_flip"     // Coin flip - rank by success count descending (not yet implemented)
	MiniGameTypeCountSeconds MiniGameType = "count_seconds" // Count seconds - rank by deviation from 5.0 ascending
	MiniGameTypeMathCalc     MiniGameType = "math_calc"     // Math calculation - rank by accuracy descending, then time_ms ascending
	MiniGameTypeRainbowMemory MiniGameType = "rainbow_memory" // Rainbow memory - rank by accuracy descending, then time_ms ascending
	MiniGameTypeVernier      MiniGameType = "vernier"       // Vernier - rank by deviation ascending
	MiniGameTypeDilemmaRace  MiniGameType = "dilemma_race"  // Dilemma race (博弈论) - Online: requires MiniGame Service
	MiniGameTypeTrustDilemma MiniGameType = "trust_dilemma" // Trust dilemma (经典博弈) - Online: requires MiniGame Service
)

// IsValid checks if MiniGameType corresponds to a known type constant.
// All defined types return true, regardless of availability status.
// Use MiniGameDefinition.Available to check if a type is in the selection pool.
func (mt MiniGameType) IsValid() bool {
	switch mt {
	case MiniGameTypeDiceRace, MiniGameTypeCoinFlip, MiniGameTypeCountSeconds,
		MiniGameTypeMathCalc, MiniGameTypeRainbowMemory, MiniGameTypeVernier,
		MiniGameTypeDilemmaRace, MiniGameTypeTrustDilemma:
		return true
	default:
		return false
	}
}

// IsOnline checks if the mini-game type requires an external online game service.
// Online types use MiniGameModeRPC and need an OnlineMiniGameProvider.
// This is a behavioral check tied to code execution paths (hardcoded).
// YAML mode field should match this for consistency (validated at load time).
func (mt MiniGameType) IsOnline() bool {
	switch mt {
	case MiniGameTypeDilemmaRace, MiniGameTypeTrustDilemma:
		return true
	default:
		return false
	}
}

// ParseMiniGameType converts a string to MiniGameType.
// Returns MiniGameTypeNone if the string is not a valid mini-game type.
func ParseMiniGameType(s string) MiniGameType {
	switch s {
	case "dice_race":
		return MiniGameTypeDiceRace
	case "coin_flip":
		return MiniGameTypeCoinFlip
	case "count_seconds":
		return MiniGameTypeCountSeconds
	case "math_calc":
		return MiniGameTypeMathCalc
	case "rainbow_memory":
		return MiniGameTypeRainbowMemory
	case "vernier":
		return MiniGameTypeVernier
	case "dilemma_race":
		return MiniGameTypeDilemmaRace
	case "trust_dilemma":
		return MiniGameTypeTrustDilemma
	default:
		return MiniGameTypeNone
	}
}

// AllMiniGameTypes is the pool of currently available mini-game types for random selection.
// Initialized with hardcoded defaults; overridden by pkg/resource init from YAML definitions.
// Only types with Available=true are included.
var AllMiniGameTypes = []MiniGameType{
	MiniGameTypeDilemmaRace,
	MiniGameTypeTrustDilemma,
}

// AllOnlineMiniGameTypes is the pool of mini-game types that require an online service.
// Initialized with hardcoded defaults; overridden by pkg/resource init from YAML definitions.
// Only types with Available=true and Mode=MiniGameModeRPC are included.
var AllOnlineMiniGameTypes = []MiniGameType{
	MiniGameTypeDilemmaRace,
	MiniGameTypeTrustDilemma,
}

// ========== Mini Game Definition (Static Metadata) ==========

// MiniGameDefinition contains static metadata for mini-game display and configuration.
// Game logic (rank calculation, online provider) stays in engine layer.
type MiniGameDefinition struct {
	Type        MiniGameType `json:"type"`
	Mode        MiniGameMode `json:"mode"`           // 0=Frontend, 1=RPC/Online
	Available   bool         `json:"available"`       // Whether in random selection pool
	EnglishName string       `json:"english_name"`    // English identifier
	Name        string       `json:"name"`            // Chinese display name
	Desc        string       `json:"desc"`            // Description
}

// IsOnline checks if the mini-game requires an external online service (Mode == MiniGameModeRPC).
func (d *MiniGameDefinition) IsOnline() bool {
	return d.Mode == MiniGameModeRPC
}