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

	engine := config.BuildMapEngine()

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
