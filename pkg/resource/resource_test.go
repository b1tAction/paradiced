package resource

import (
	"testing"

	"github.com/b1tAction/paradiced/pkg/constants"
)

func TestLoadDefault(t *testing.T) {
	config, err := LoadDefault()
	if err != nil {
		t.Fatalf("LoadDefault failed: %v", err)
	}

	if config.Length != 20 {
		t.Errorf("config.Length = %d, want 20", config.Length)
	}

	if config.StartIndex != 0 {
		t.Errorf("config.StartIndex = %d, want 0", config.StartIndex)
	}

	if config.EndIndex != 19 {
		t.Errorf("config.EndIndex = %d, want 19", config.EndIndex)
	}

	if len(config.Cells) != 20 {
		t.Errorf("len(config.Cells) = %d, want 20", len(config.Cells))
	}
}

func TestMapConfigBuildMapEngine(t *testing.T) {
	config, err := LoadDefault()
	if err != nil {
		t.Fatalf("LoadDefault failed: %v", err)
	}

	engine := BuildMapEngineFromConfig(config)

	if engine.Length != 20 {
		t.Errorf("engine.Length = %d, want 20", engine.Length)
	}

	if engine.StartIndex != 0 {
		t.Errorf("engine.StartIndex = %d, want 0", engine.StartIndex)
	}

	if engine.EndIndex != 19 {
		t.Errorf("engine.EndIndex = %d, want 19", engine.EndIndex)
	}

	// Verify cell at index 0 (Checkpoint with Item draw)
	cell0 := engine.Cells[0]
	if cell0.CellType != constants.CellTypeCheckpoint {
		t.Errorf("cell[0].CellType = %v, want %v", cell0.CellType, constants.CellTypeCheckpoint)
	}
	if cell0.DrawType != constants.DrawTypeItem {
		t.Errorf("cell[0].DrawType = %v, want %v", cell0.DrawType, constants.DrawTypeItem)
	}

	// Verify cell at index 1 (Normal with mixed probability)
	cell1 := engine.Cells[1]
	if cell1.CellType != constants.CellTypeNormal {
		t.Errorf("cell[1].CellType = %v, want %v", cell1.CellType, constants.CellTypeNormal)
	}
	if cell1.DrawType != constants.DrawTypeEvent {
		t.Errorf("cell[1].DrawType = %v, want %v", cell1.DrawType, constants.DrawTypeEvent)
	}
	if cell1.ProbGood != 0.3 {
		t.Errorf("cell[1].ProbGood = %f, want 0.3", cell1.ProbGood)
	}
	if cell1.ProbNeutral != 0.5 {
		t.Errorf("cell[1].ProbNeutral = %f, want 0.5", cell1.ProbNeutral)
	}
	if cell1.ProbBad != 0.2 {
		t.Errorf("cell[1].ProbBad = %f, want 0.2", cell1.ProbBad)
	}

	// Verify cell at index 5 (Fog with 100% Bad)
	cell5 := engine.Cells[5]
	if cell5.CellType != constants.CellTypeFog {
		t.Errorf("cell[5].CellType = %v, want %v", cell5.CellType, constants.CellTypeFog)
	}
	if cell5.DrawType != constants.DrawTypeEvent {
		t.Errorf("cell[5].DrawType = %v, want %v", cell5.DrawType, constants.DrawTypeEvent)
	}
	if cell5.ProbBad != 1.0 {
		t.Errorf("cell[5].ProbBad = %f, want 1.0", cell5.ProbBad)
	}

	// Verify cell at index 7 (Event with bound event ID)
	cell7 := engine.Cells[7]
	if cell7.CellType != constants.CellTypeEvent {
		t.Errorf("cell[7].CellType = %v, want %v", cell7.CellType, constants.CellTypeEvent)
	}
	if cell7.EventID != "herb" {
		t.Errorf("cell[7].EventID = %q, want \"herb\"", cell7.EventID)
	}

	// Verify cell at index 19 (Boss, no draw)
	cell19 := engine.Cells[19]
	if cell19.CellType != constants.CellTypeBoss {
		t.Errorf("cell[19].CellType = %v, want %v", cell19.CellType, constants.CellTypeBoss)
	}
	if cell19.DrawType != constants.DrawTypeNone {
		t.Errorf("cell[19].DrawType = %v, want %v", cell19.DrawType, constants.DrawTypeNone)
	}
}

func TestLoadMapFromJSON(t *testing.T) {
	jsonData := []byte(`{
		"length": 5,
		"start_index": 0,
		"end_index": 4,
		"cells": [
			{"index": 0, "cell_type": "checkpoint", "draw_type": "item", "prob_good": 1.0},
			{"index": 1, "cell_type": "normal", "draw_type": "event", "prob_good": 0.5, "prob_bad": 0.5},
			{"index": 2, "cell_type": "fragile", "draw_type": "none"},
			{"index": 3, "cell_type": "fog", "draw_type": "event", "prob_bad": 1.0},
			{"index": 4, "cell_type": "boss", "draw_type": "none"}
		]
	}`)

	config, err := LoadMapFromJSON(jsonData)
	if err != nil {
		t.Fatalf("LoadMapFromJSON failed: %v", err)
	}

	if config.Length != 5 {
		t.Errorf("config.Length = %d, want 5", config.Length)
	}

	if len(config.Cells) != 5 {
		t.Errorf("len(config.Cells) = %d, want 5", len(config.Cells))
	}

	// Verify first cell
	cell0 := config.Cells[0]
	if cell0.CellType != "checkpoint" {
		t.Errorf("cell[0].CellType = %v, want checkpoint", cell0.CellType)
	}
	if cell0.DrawType != "item" {
		t.Errorf("cell[0].DrawType = %v, want item", cell0.DrawType)
	}
	if cell0.ProbGood != 1.0 {
		t.Errorf("cell[0].ProbGood = %f, want 1.0", cell0.ProbGood)
	}
}

func TestLoadMapFromJSONInvalid(t *testing.T) {
	_, err := LoadMapFromJSON([]byte("invalid json"))
	if err == nil {
		t.Error("LoadMapFromJSON with invalid JSON should return error")
	}
}

// ========== Definition Loading Tests ==========

func TestLoadDefinitionsFromYAML(t *testing.T) {
	// Load definitions from the embedded paradiced.yml
	defs, err := LoadDefinitions()
	if err != nil {
		t.Fatalf("LoadDefinitions failed: %v", err)
	}

	// Verify all expected events are loaded with typed keys
	expectedEvents := []constants.EventType{
		constants.EventTypeHerb, constants.EventTypeLuckyBubble, constants.EventTypeRelic,
		constants.EventTypeDivineBless, constants.EventTypeExchange, constants.EventTypeHiddenBuff,
		constants.EventTypeTasteTest, constants.EventTypeMosquito, constants.EventTypeGhostHit,
		constants.EventTypeDogPoop, constants.EventTypeWindGust, constants.EventTypeSkullGaze,
		constants.EventTypeLostWay, constants.EventTypeThunder,
	}
	if len(defs.Events) != len(expectedEvents) {
		t.Errorf("len(defs.Events) = %d, want %d", len(defs.Events), len(expectedEvents))
	}
	for _, et := range expectedEvents {
		if _, ok := defs.Events[et]; !ok {
			t.Errorf("missing event definition for %s", et)
		}
	}

	// Verify all expected buffs are loaded with typed keys
	expectedBuffs := []constants.BuffType{
		constants.BuffTypeCurse, constants.BuffTypeLost, constants.BuffTypeCorrupt,
		constants.BuffTypePoison, constants.BuffTypeHidden, constants.BuffTypeThorns,
		constants.BuffTypeDivine, constants.BuffTypeRain, constants.BuffTypeExorcism,
		constants.BuffTypeFire, constants.BuffTypeDeathMark,
		constants.BuffTypeDominance, constants.BuffTypeRobLuck, constants.BuffTypeSuppress,
		constants.BuffTypeSinking, constants.BuffTypeEternal, constants.BuffTypeFearless,
		constants.BuffTypeGoldenBody, constants.BuffTypeWrath,
		constants.BuffTypeSavior, constants.BuffTypeSageProtection,
	}
	if len(defs.Buffs) != len(expectedBuffs) {
		t.Errorf("len(defs.Buffs) = %d, want %d", len(defs.Buffs), len(expectedBuffs))
	}
	for _, bt := range expectedBuffs {
		if _, ok := defs.Buffs[bt]; !ok {
			t.Errorf("missing buff definition for %s", bt)
		}
	}

	// Verify all expected items are loaded with typed keys
	expectedItems := []constants.ItemType{
		constants.ItemTypeReverseClock, constants.ItemTypeAnyDoor, constants.ItemTypeDiceUpgrade,
		constants.ItemTypeMagicFlute, constants.ItemTypeCupidArrow, constants.ItemTypeCrimsonBlade,
		constants.ItemTypeWisdomRing, constants.ItemTypeMeditationRing, constants.ItemTypeDisciplineRing,
		constants.ItemTypeFoolishRing, constants.ItemTypeGreedyRing, constants.ItemTypeWrathRing,
		constants.ItemTypeNamedBlade, constants.ItemTypeSageProtection,
	}
	if len(defs.Items) != len(expectedItems) {
		t.Errorf("len(defs.Items) = %d, want %d", len(defs.Items), len(expectedItems))
	}
	for _, it := range expectedItems {
		if _, ok := defs.Items[it]; !ok {
			t.Errorf("missing item definition for %s", it)
		}
	}
}

func TestLoadDefinitionsEventFields(t *testing.T) {
	defs, err := LoadDefinitions()
	if err != nil {
		t.Fatalf("LoadDefinitions failed: %v", err)
	}

	// Spot-check specific event fields
	herb := defs.Events[constants.EventTypeHerb]
	if herb == nil {
		t.Fatal("missing herb event")
	}
	if herb.Eval != constants.EvaluationMildGood {
		t.Errorf("herb.Eval = %d, want %d (mild_good=70)", herb.Eval, constants.EvaluationMildGood)
	}
	if herb.EnglishName != "Herb" {
		t.Errorf("herb.EnglishName = %s, want Herb", herb.EnglishName)
	}
	if herb.Name != "采集到草药" {
		t.Errorf("herb.Name = %s, want 采集到草药", herb.Name)
	}
	if herb.Type != constants.EventTypeHerb {
		t.Errorf("herb.Type = %s, want %s", herb.Type, constants.EventTypeHerb)
	}

	thunder := defs.Events[constants.EventTypeThunder]
	if thunder == nil {
		t.Fatal("missing thunder event")
	}
	if thunder.Eval != constants.EvaluationVeryBad {
		t.Errorf("thunder.Eval = %d, want %d (very_bad=10)", thunder.Eval, constants.EvaluationVeryBad)
	}
	if thunder.EnglishName != "Thunder" {
		t.Errorf("thunder.EnglishName = %s, want Thunder", thunder.EnglishName)
	}
}

func TestLoadDefinitionsBuffFields(t *testing.T) {
	defs, err := LoadDefinitions()
	if err != nil {
		t.Fatalf("LoadDefinitions failed: %v", err)
	}

	// Spot-check specific buff fields
	divine := defs.Buffs[constants.BuffTypeDivine]
	if divine == nil {
		t.Fatal("missing divine buff")
	}
	if divine.Eval != constants.EvaluationVeryGood {
		t.Errorf("divine.Eval = %d, want %d (very_good=90)", divine.Eval, constants.EvaluationVeryGood)
	}
	if divine.Name != "神眷" {
		t.Errorf("divine.Name = %s, want 神眷", divine.Name)
	}
	if divine.Duration != 3 {
		t.Errorf("divine.Duration = %d, want 3", divine.Duration)
	}
	if divine.Type != constants.BuffTypeDivine {
		t.Errorf("divine.Type = %s, want %s", divine.Type, constants.BuffTypeDivine)
	}

	fire := defs.Buffs[constants.BuffTypeFire]
	if fire == nil {
		t.Fatal("missing fire buff")
	}
	if fire.Duration != -1 {
		t.Errorf("fire.Duration = %d, want -1 (permanent)", fire.Duration)
	}

	thorns := defs.Buffs[constants.BuffTypeThorns]
	if thorns == nil {
		t.Fatal("missing thorns buff")
	}
	if thorns.Eval != constants.EvaluationNeutral {
		t.Errorf("thorns.Eval = %d, want %d (neutral=50)", thorns.Eval, constants.EvaluationNeutral)
	}
}

func TestLoadDefinitionsItemFields(t *testing.T) {
	defs, err := LoadDefinitions()
	if err != nil {
		t.Fatalf("LoadDefinitions failed: %v", err)
	}

	// Spot-check specific item fields
	anyDoor := defs.Items[constants.ItemTypeAnyDoor]
	if anyDoor == nil {
		t.Fatal("missing any_door item")
	}
	if anyDoor.Eval != constants.EvaluationNeutral {
		t.Errorf("anyDoor.Eval = %d, want %d (neutral=50)", anyDoor.Eval, constants.EvaluationNeutral)
	}
	if anyDoor.EnglishName != "AnyDoor" {
		t.Errorf("anyDoor.EnglishName = %s, want AnyDoor", anyDoor.EnglishName)
	}
	if anyDoor.Name != "任意门" {
		t.Errorf("anyDoor.Name = %s, want 任意门", anyDoor.Name)
	}
	if anyDoor.Desc != "前往指定玩家身边" {
		t.Errorf("anyDoor.Desc = %s, want 前往指定玩家身边", anyDoor.Desc)
	}
	if anyDoor.Type != constants.ItemTypeAnyDoor {
		t.Errorf("anyDoor.Type = %s, want %s", anyDoor.Type, constants.ItemTypeAnyDoor)
	}
}

func TestLoadDefinitionsFromYAMLInvalidYAML(t *testing.T) {
	_, err := LoadDefinitionsFromYAML([]byte("invalid: yaml: [["))
	if err == nil {
		t.Error("LoadDefinitionsFromYAML with invalid YAML should return error")
	}
}

func TestLoadDefinitionsFromYAMLUnknownEventType(t *testing.T) {
	yaml := []byte(`
events:
  unknown_event:
    evaluation: good
    english_name: Unknown
    name: UnknownEvent
    desc: Test
buffs: {}
items: {}
`)
	_, err := LoadDefinitionsFromYAML(yaml)
	if err == nil {
		t.Error("LoadDefinitionsFromYAML with unknown event type should return error")
	}
}

func TestLoadDefinitionsFromYAMLUnknownBuffType(t *testing.T) {
	yaml := []byte(`
events: {}
buffs:
  unknown_buff:
    evaluation: good
    english_name: Unknown
    name: UnknownBuff
    desc: Test
    duration: 3
items: {}
`)
	_, err := LoadDefinitionsFromYAML(yaml)
	if err == nil {
		t.Error("LoadDefinitionsFromYAML with unknown buff type should return error")
	}
}

func TestLoadDefinitionsFromYAMLUnknownItemType(t *testing.T) {
	yaml := []byte(`
events: {}
buffs: {}
items:
  unknown_item:
    evaluation: good
    english_name: Unknown
    name: UnknownItem
    desc: Test
`)
	_, err := LoadDefinitionsFromYAML(yaml)
	if err == nil {
		t.Error("LoadDefinitionsFromYAML with unknown item type should return error")
	}
}

func TestLoadDefinitionsFromYAMLInvalidEvaluation(t *testing.T) {
	yaml := []byte(`
events:
  herb:
    evaluation: not_a_real_evaluation
    english_name: Herb
    name: Test
    desc: Test
buffs: {}
items: {}
`)
	_, err := LoadDefinitionsFromYAML(yaml)
	if err == nil {
		t.Error("LoadDefinitionsFromYAML with invalid evaluation should return error")
	}
}

func TestLoadDefinitionsFromYAMLOutOfRangeEvaluation(t *testing.T) {
	yaml := []byte(`
events:
  herb:
    evaluation: "150"
    english_name: Herb
    name: Test
    desc: Test
buffs: {}
items: {}
`)
	_, err := LoadDefinitionsFromYAML(yaml)
	if err == nil {
		t.Error("LoadDefinitionsFromYAML with out-of-range evaluation should return error")
	}
}

func TestLoadDefinitionsFromYAMLNumericEvaluation(t *testing.T) {
	yaml := []byte(`
events:
  herb:
    evaluation: "65"
    english_name: Herb
    name: Test
    desc: Test
buffs: {}
items: {}
`)
	defs, err := LoadDefinitionsFromYAML(yaml)
	if err != nil {
		t.Fatalf("LoadDefinitionsFromYAML with numeric evaluation: %v", err)
	}
	herb := defs.Events[constants.EventTypeHerb]
	if herb.Eval != 65 {
		t.Errorf("herb.Eval = %d, want 65", herb.Eval)
	}
}

// ========== parseEvaluation Tests ==========

func TestParseEvaluationNamedConstants(t *testing.T) {
	tests := []struct {
		input    string
		expected constants.Evaluation
	}{
		{"very_bad", constants.EvaluationVeryBad},
		{"bad", constants.EvaluationBad},
		{"mild_bad", constants.EvaluationMildBad},
		{"neutral", constants.EvaluationNeutral},
		{"mixed", constants.EvaluationMixed},
		{"mild_good", constants.EvaluationMildGood},
		{"good", constants.EvaluationGood},
		{"very_good", constants.EvaluationVeryGood},
		{"excellent", constants.EvaluationExcellent},
	}

	for _, tt := range tests {
		eval, err := parseEvaluation(tt.input)
		if err != nil {
			t.Errorf("parseEvaluation(%s) unexpected error: %v", tt.input, err)
			continue
		}
		if eval != tt.expected {
			t.Errorf("parseEvaluation(%s) = %d, want %d", tt.input, eval, tt.expected)
		}
	}
}

func TestParseEvaluationNumeric(t *testing.T) {
	tests := []struct {
		input    string
		expected constants.Evaluation
	}{
		{"0", 0},
		{"50", 50},
		{"100", 100},
		{"65", 65},
	}

	for _, tt := range tests {
		eval, err := parseEvaluation(tt.input)
		if err != nil {
			t.Errorf("parseEvaluation(%s) unexpected error: %v", tt.input, err)
			continue
		}
		if eval != tt.expected {
			t.Errorf("parseEvaluation(%s) = %d, want %d", tt.input, eval, tt.expected)
		}
	}
}

func TestParseEvaluationInvalid(t *testing.T) {
	tests := []string{
		"not_real",
		"150",   // out of range
		"-10",   // out of range
		"abcde", // not a number
	}

	for _, input := range tests {
		_, err := parseEvaluation(input)
		if err == nil {
			t.Errorf("parseEvaluation(%s) should return error", input)
		}
	}
}

func TestGlobalDefinitionSetInitialized(t *testing.T) {
	// GlobalDefinitionSet should be populated at init time
	if GlobalDefinitionSet == nil {
		t.Fatal("GlobalDefinitionSet should not be nil (initialized at init)")
	}
	if len(GlobalDefinitionSet.Events) == 0 {
		t.Error("GlobalDefinitionSet.Events should not be empty")
	}
	if len(GlobalDefinitionSet.Buffs) == 0 {
		t.Error("GlobalDefinitionSet.Buffs should not be empty")
	}
	if len(GlobalDefinitionSet.Items) == 0 {
		t.Error("GlobalDefinitionSet.Items should not be empty")
	}
	if len(GlobalDefinitionSet.MiniGames) == 0 {
		t.Error("GlobalDefinitionSet.MiniGames should not be empty")
	}
}

func TestLoadDefinitionsMiniGameFields(t *testing.T) {
	defs, err := LoadDefinitions()
	if err != nil {
		t.Fatalf("LoadDefinitions failed: %v", err)
	}

	// Verify all expected mini-game types are loaded
	expectedMiniGames := []constants.MiniGameType{
		constants.MiniGameTypeDiceRace, constants.MiniGameTypeCoinFlip,
		constants.MiniGameTypeCountSeconds, constants.MiniGameTypeMathCalc,
		constants.MiniGameTypeRainbowMemory, constants.MiniGameTypeVernier,
		constants.MiniGameTypeDilemmaRace, constants.MiniGameTypeTrustDilemma,
		constants.MiniGameTypeCakeCutting, constants.MiniGameTypeTypingSpeed,
	}
	if len(defs.MiniGames) != len(expectedMiniGames) {
		t.Errorf("len(defs.MiniGames) = %d, want %d", len(defs.MiniGames), len(expectedMiniGames))
	}
	for _, mt := range expectedMiniGames {
		if _, ok := defs.MiniGames[mt]; !ok {
			t.Errorf("missing mini-game definition for %s", mt)
		}
	}

	// Spot-check: dice_race is a frontend type (behavioral invariant)
	diceRace := defs.MiniGames[constants.MiniGameTypeDiceRace]
	if diceRace == nil {
		t.Fatal("missing dice_race mini-game")
	}
	if diceRace.Mode != constants.MiniGameModeFrontend {
		t.Errorf("dice_race.Mode = %d, want %d (frontend)", diceRace.Mode, constants.MiniGameModeFrontend)
	}
	if diceRace.EnglishName != "DiceRace" {
		t.Errorf("dice_race.EnglishName = %s, want DiceRace", diceRace.EnglishName)
	}
	if diceRace.Name != "骰子竞速" {
		t.Errorf("dice_race.Name = %s, want 骰子竞速", diceRace.Name)
	}

	// Spot-check: coin_flip is loaded (available is config-dependent, not asserted)
	coinFlip := defs.MiniGames[constants.MiniGameTypeCoinFlip]
	if coinFlip == nil {
		t.Fatal("missing coin_flip mini-game")
	}

	// Spot-check: dilemma_race is an online type (behavioral invariant)
	dilemmaRace := defs.MiniGames[constants.MiniGameTypeDilemmaRace]
	if dilemmaRace == nil {
		t.Fatal("missing dilemma_race mini-game")
	}
	if dilemmaRace.Mode != constants.MiniGameModeRPC {
		t.Errorf("dilemma_race.Mode = %d, want %d (RPC)", dilemmaRace.Mode, constants.MiniGameModeRPC)
	}
}

func TestInitMiniGamePools(t *testing.T) {
	// After init, AllMiniGameTypes should not be empty
	if len(constants.AllMiniGameTypes) == 0 {
		t.Error("AllMiniGameTypes should not be empty (populated by initMiniGamePools)")
	}

	// Verify that unavailable types are not in the pool
	for _, mt := range constants.AllMiniGameTypes {
		def := GlobalDefinitionSet.MiniGames[mt]
		if def == nil {
			t.Errorf("AllMiniGameTypes contains type without definition: %s", mt)
		}
		if !def.Available {
			t.Errorf("AllMiniGameTypes contains unavailable type: %s", mt)
		}
	}

	// AllOnlineMiniGameTypes should only contain online types
	if len(constants.AllOnlineMiniGameTypes) == 0 {
		t.Error("AllOnlineMiniGameTypes should not be empty")
	}
	for _, mt := range constants.AllOnlineMiniGameTypes {
		if !mt.IsOnline() {
			t.Errorf("AllOnlineMiniGameTypes contains non-online type: %s", mt)
		}
	}
}

func TestLoadDefinitionsFromYAMLUnknownMiniGameType(t *testing.T) {
	yaml := []byte(`
events: {}
buffs: {}
items: {}
mini_games:
  unknown_game:
    mode: frontend
    available: true
    english_name: Unknown
    name: Unknown
    desc: Test
`)
	_, err := LoadDefinitionsFromYAML(yaml)
	if err == nil {
		t.Error("LoadDefinitionsFromYAML with unknown mini-game type should return error")
	}
}

func TestLoadDefinitionsFromYAMLInconsistentModeOnline(t *testing.T) {
	yaml := []byte(`
events: {}
buffs: {}
items: {}
mini_games:
  dice_race:
    mode: online
    available: true
    english_name: DiceRace
    name: Test
    desc: Test
`)
	_, err := LoadDefinitionsFromYAML(yaml)
	if err == nil {
		t.Error("LoadDefinitionsFromYAML with mode=online for dice_race (not online in Go) should return error")
	}
}

func TestLoadDefinitionsFromYAMLInconsistentModeFrontend(t *testing.T) {
	yaml := []byte(`
events: {}
buffs: {}
items: {}
mini_games:
  dilemma_race:
    mode: frontend
    available: true
    english_name: DilemmaRace
    name: Test
    desc: Test
`)
	_, err := LoadDefinitionsFromYAML(yaml)
	if err == nil {
		t.Error("LoadDefinitionsFromYAML with mode=frontend for dilemma_race (online in Go) should return error")
	}
}
