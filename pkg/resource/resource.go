// Package resource provides game resource loading functionality.
// This package handles loading map data from JSON files.
package resource

import (
	_ "embed"
	"encoding/json"

	"github.com/b1tAction/paradiced/internal/gamemap"
	"github.com/b1tAction/paradiced/pkg/constants"
)

//go:embed default.json
var defaultJSON []byte

// MapConfig represents map configuration in JSON.
type MapConfig struct {
	Length     int            `json:"length"`
	StartIndex int            `json:"start_index"`
	EndIndex   int            `json:"end_index"`
	Cells      []MapCellConfig `json:"cells"`
}

// MapCellConfig represents single cell configuration in JSON.
type MapCellConfig struct {
	Index       int     `json:"index"`
	CellType    string  `json:"cell_type"`
	IsBroken    bool    `json:"is_broken"`
	EventID     string  `json:"event_id"`
	FogActive   bool    `json:"fog_active"`
	DrawType    string  `json:"draw_type"`
	ProbGood    float64 `json:"prob_good"`
	ProbNeutral float64 `json:"prob_neutral"`
	ProbBad     float64 `json:"prob_bad"`
}

// ResourceSet contains all loaded resources.
type ResourceSet struct {
	Map *MapConfig `json:"map"`
}

// LoadDefault loads the default resource set (default.json).
func LoadDefault() (*MapConfig, error) {
	return LoadMapFromJSON(defaultJSON)
}

// LoadMapFromJSON loads map configuration from JSON data.
func LoadMapFromJSON(data []byte) (*MapConfig, error) {
	var config MapConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

// BuildMapEngine creates a MapEngine from loaded configuration.
func (m *MapConfig) BuildMapEngine() *gamemap.MapEngine {
	engine := gamemap.NewMapEngine(m.Length)
	engine.StartIndex = m.StartIndex
	engine.EndIndex = m.EndIndex

	for _, cellConfig := range m.Cells {
		cell := engine.Cells[cellConfig.Index]
		cell.CellType = constants.ParseCellType(cellConfig.CellType)
		cell.IsBroken = cellConfig.IsBroken
		cell.EventID = cellConfig.EventID
		cell.FogActive = cellConfig.FogActive
		cell.DrawType = constants.ParseDrawType(cellConfig.DrawType)
		cell.ProbGood = cellConfig.ProbGood
		cell.ProbNeutral = cellConfig.ProbNeutral
		cell.ProbBad = cellConfig.ProbBad
	}

	return engine
}
