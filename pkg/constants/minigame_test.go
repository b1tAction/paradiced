package constants

import "testing"

func TestMiniGameTypeIsValid(t *testing.T) {
	tests := []struct {
		name    string
		mgType  MiniGameType
		isValid bool
	}{
		{"dice_race is valid", MiniGameTypeDiceRace, true},
		{"coin_flip is valid (known type, not yet available)", MiniGameTypeCoinFlip, true},
		{"count_seconds is valid", MiniGameTypeCountSeconds, true},
		{"math_calc is valid", MiniGameTypeMathCalc, true},
		{"rainbow_memory is valid", MiniGameTypeRainbowMemory, true},
		{"vernier is valid", MiniGameTypeVernier, true},
		{"dilemma_race is valid", MiniGameTypeDilemmaRace, true},
		{"invalid type", MiniGameType("invalid"), false},
		{"empty type", MiniGameType(""), false},
		{"none is not valid", MiniGameTypeNone, false},
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

func TestMiniGameTypeIsOnline(t *testing.T) {
	tests := []struct {
		name     string
		mgType   MiniGameType
		isOnline bool
	}{
		{"dice_race is not online", MiniGameTypeDiceRace, false},
		{"coin_flip is not online", MiniGameTypeCoinFlip, false},
		{"count_seconds is not online", MiniGameTypeCountSeconds, false},
		{"math_calc is not online", MiniGameTypeMathCalc, false},
		{"rainbow_memory is not online", MiniGameTypeRainbowMemory, false},
		{"vernier is not online", MiniGameTypeVernier, false},
		{"dilemma_race is online", MiniGameTypeDilemmaRace, true},
		{"invalid type is not online", MiniGameType("invalid"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.mgType.IsOnline() != tt.isOnline {
				t.Errorf("MiniGameType(%s).IsOnline() = %v, want %v", tt.mgType, tt.mgType.IsOnline(), tt.isOnline)
			}
		})
	}
}

func TestAllOnlineMiniGameTypes(t *testing.T) {
	if len(AllOnlineMiniGameTypes) == 0 {
		t.Error("AllOnlineMiniGameTypes should not be empty")
	}
	for _, mgType := range AllOnlineMiniGameTypes {
		if !mgType.IsOnline() {
			t.Errorf("AllOnlineMiniGameTypes contains non-online type: %s", mgType)
		}
		if !mgType.IsValid() {
			t.Errorf("AllOnlineMiniGameTypes contains invalid type: %s", mgType)
		}
	}
}

func TestParseMiniGameType(t *testing.T) {
	tests := []struct {
		input    string
		expected MiniGameType
	}{
		{"dice_race", MiniGameTypeDiceRace},
		{"coin_flip", MiniGameTypeCoinFlip},
		{"count_seconds", MiniGameTypeCountSeconds},
		{"math_calc", MiniGameTypeMathCalc},
		{"rainbow_memory", MiniGameTypeRainbowMemory},
		{"vernier", MiniGameTypeVernier},
		{"dilemma_race", MiniGameTypeDilemmaRace},
		{"unknown", MiniGameTypeNone},
		{"", MiniGameTypeNone},
	}

	for _, tt := range tests {
		result := ParseMiniGameType(tt.input)
		if result != tt.expected {
			t.Errorf("ParseMiniGameType(%s) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}

func TestParseMiniGameMode(t *testing.T) {
	tests := []struct {
		input    string
		expected MiniGameMode
	}{
		{"frontend", MiniGameModeFrontend},
		{"online", MiniGameModeRPC},
		{"unknown", MiniGameModeFrontend},
		{"", MiniGameModeFrontend},
	}

	for _, tt := range tests {
		result := ParseMiniGameMode(tt.input)
		if result != tt.expected {
			t.Errorf("ParseMiniGameMode(%s) = %d, want %d", tt.input, result, tt.expected)
		}
	}
}

func TestMiniGameDefinitionIsOnline(t *testing.T) {
	frontendDef := &MiniGameDefinition{Mode: MiniGameModeFrontend}
	onlineDef := &MiniGameDefinition{Mode: MiniGameModeRPC}

	if frontendDef.IsOnline() {
		t.Error("Frontend mode definition should not be online")
	}
	if !onlineDef.IsOnline() {
		t.Error("RPC mode definition should be online")
	}
}