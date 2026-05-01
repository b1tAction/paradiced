package constants

import "testing"

func TestMiniGameTypeIsValid(t *testing.T) {
	tests := []struct {
		name    string
		mgType  MiniGameType
		isValid bool
	}{
		{"dice_race is valid", MiniGameTypeDiceRace, true},
		{"coin_flip is not yet implemented", MiniGameTypeCoinFlip, false},
		{"count_seconds is valid", MiniGameTypeCountSeconds, true},
		{"math_calc is valid", MiniGameTypeMathCalc, true},
		{"rainbow_memory is valid", MiniGameTypeRainbowMemory, true},
		{"vernier is valid", MiniGameTypeVernier, true},
		{"invalid type", MiniGameType("invalid"), false},
		{"empty type", MiniGameType(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.mgType.IsValid() != tt.isValid {
				t.Errorf("MiniGameType(%s).IsValid() = %v, want %v", tt.mgType, tt.mgType.IsValid(), tt.isValid)
			}
		})
	}
}

func TestAllMiniGameTypes(t *testing.T) {
	if len(AllMiniGameTypes) == 0 {
		t.Error("AllMiniGameTypes should not be empty")
	}
	for _, mgType := range AllMiniGameTypes {
		if !mgType.IsValid() {
			t.Errorf("AllMiniGameTypes contains invalid type: %s", mgType)
		}
	}
}

func TestMiniGameModeConstants(t *testing.T) {
	if MiniGameModeFrontend != MiniGameMode(0) {
		t.Errorf("MiniGameModeFrontend = %d, want 0", MiniGameModeFrontend)
	}
	if MiniGameModeRPC != MiniGameMode(1) {
		t.Errorf("MiniGameModeRPC = %d, want 1", MiniGameModeRPC)
	}
}