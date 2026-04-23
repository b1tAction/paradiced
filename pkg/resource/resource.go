// Package resource provides game resource loading functionality.
// This package handles loading map data from JSON files.
package resource

import (
	_ "embed"
	"encoding/json"

	pkgnet "github.com/b1tAction/paradiced/pkg/net"

	"github.com/b1tAction/paradiced/internal/gamemap"
	"github.com/b1tAction/paradiced/pkg/constants"
)

//go:embed default.json
var defaultJSON []byte

// ResourceSet contains all loaded resources.
type ResourceSet struct {
	Map *pkgnet.MapConfig `json:"map"`
}

// LoadDefault loads the default resource set (default.json).
func LoadDefault() (*pkgnet.MapConfig, error) {
	return LoadMapFromJSON(defaultJSON)
}

// LoadMapFromJSON loads map configuration from JSON data.
func LoadMapFromJSON(data []byte) (*pkgnet.MapConfig, error) {
	var config pkgnet.MapConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

// BuildMapEngineFromConfig creates a MapEngine from loaded map configuration.
// This is a standalone function (not a method) because MapConfig is defined in pkg/net.
func BuildMapEngineFromConfig(m *pkgnet.MapConfig) *gamemap.MapEngine {
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