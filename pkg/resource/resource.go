// Package resource provides game resource loading functionality.
// This package handles loading map data from JSON and definition data from YAML.
package resource

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strconv"

	pkgnet "github.com/b1tAction/paradiced/pkg/net"
	"gopkg.in/yaml.v3"

	"github.com/b1tAction/paradiced/internal/gamemap"
	"github.com/b1tAction/paradiced/pkg/constants"
	"github.com/b1tAction/paradiced/pkg/rng"
)

//go:embed default.json
var defaultJSON []byte

//go:embed paradiced.yml
var definitionsYAML []byte

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

// ========== Definition Loading ==========

// DefinitionSet holds all parsed event/buff/item/mini_game/faction/dice definitions from YAML.
type DefinitionSet struct {
	Events    map[constants.EventType]*constants.EventDefinition
	Buffs     map[constants.BuffType]*constants.BuffDefinition
	Items     map[constants.ItemType]*constants.ItemDefinition
	MiniGames map[constants.MiniGameType]*constants.MiniGameDefinition
	Factions  map[constants.Faction]*constants.FactionDefinition
	Dice      map[rng.DiceType]*rng.DiceDefinition
}

// GlobalDefinitionSet is the globally loaded definition set, populated at init time.
var GlobalDefinitionSet *DefinitionSet

func init() {
	set, err := LoadDefinitions()
	if err != nil {
		panic(fmt.Sprintf("failed to load definitions: %v", err))
	}
	GlobalDefinitionSet = set
	initMiniGamePools()
}

// LoadDefinitions parses paradiced.yml and returns a DefinitionSet.
func LoadDefinitions() (*DefinitionSet, error) {
	return LoadDefinitionsFromYAML(definitionsYAML)
}

// LoadDefinitionsFromYAML parses YAML data and returns a DefinitionSet.
func LoadDefinitionsFromYAML(data []byte) (*DefinitionSet, error) {
	var raw yamlDefinitions
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("yaml unmarshal: %w", err)
	}

	events := make(map[constants.EventType]*constants.EventDefinition, len(raw.Events))
	for key, def := range raw.Events {
		eval, err := parseEvaluation(def.Evaluation)
		if err != nil {
			return nil, fmt.Errorf("event %s: %w", key, err)
		}
		et := constants.ParseEventType(key)
		if et == constants.EventTypeNone {
			return nil, fmt.Errorf("event %s: unknown event type", key)
		}
		events[et] = &constants.EventDefinition{
			Type:        et,
			Eval:        eval,
			EnglishName: def.EnglishName,
			Name:        def.Name,
			Desc:        def.Desc,
		}
	}

	buffs := make(map[constants.BuffType]*constants.BuffDefinition, len(raw.Buffs))
	for key, def := range raw.Buffs {
		eval, err := parseEvaluation(def.Evaluation)
		if err != nil {
			return nil, fmt.Errorf("buff %s: %w", key, err)
		}
		bt := constants.ParseBuffType(key)
		if bt == constants.BuffTypeNone {
			return nil, fmt.Errorf("buff %s: unknown buff type", key)
		}
		buffs[bt] = &constants.BuffDefinition{
			Type:        bt,
			Eval:        eval,
			EnglishName: def.EnglishName,
			Name:        def.Name,
			Desc:        def.Desc,
			Duration:    def.Duration,
		}
	}

	items := make(map[constants.ItemType]*constants.ItemDefinition, len(raw.Items))
	for key, def := range raw.Items {
		eval, err := parseEvaluation(def.Evaluation)
		if err != nil {
			return nil, fmt.Errorf("item %s: %w", key, err)
		}
		it := constants.ParseItemType(key)
		if it == constants.ItemTypeNone {
			return nil, fmt.Errorf("item %s: unknown item type", key)
		}
		items[it] = &constants.ItemDefinition{
			Type:        it,
			Eval:        eval,
			EnglishName: def.EnglishName,
			Name:        def.Name,
			Desc:        def.Desc,
		}
	}

	miniGames := make(map[constants.MiniGameType]*constants.MiniGameDefinition, len(raw.MiniGames))
	for key, def := range raw.MiniGames {
		mt := constants.ParseMiniGameType(key)
		if mt == constants.MiniGameTypeNone {
			return nil, fmt.Errorf("mini_game %s: unknown mini-game type", key)
		}
		parsedMode := constants.ParseMiniGameMode(def.Mode)
		// Validate: YAML mode must match Go IsOnline() behavioral check
		if parsedMode == constants.MiniGameModeRPC && !mt.IsOnline() {
			return nil, fmt.Errorf("mini_game %s: YAML mode=online but Go IsOnline()=false (inconsistent)", key)
		}
		if parsedMode == constants.MiniGameModeFrontend && mt.IsOnline() {
			return nil, fmt.Errorf("mini_game %s: YAML mode=frontend but Go IsOnline()=true (inconsistent)", key)
		}
		miniGames[mt] = &constants.MiniGameDefinition{
			Type:        mt,
			Mode:        parsedMode,
			Available:   def.Available,
			EnglishName: def.EnglishName,
			Name:        def.Name,
			Desc:        def.Desc,
		}
	}

	factions := make(map[constants.Faction]*constants.FactionDefinition, len(raw.Factions))
	for key, def := range raw.Factions {
		ft := constants.ParseFaction(key)
		if ft == constants.FactionNone {
			return nil, fmt.Errorf("faction %s: unknown faction type", key)
		}
		factions[ft] = &constants.FactionDefinition{
			Type:        ft,
			EnglishName: def.EnglishName,
			Name:        def.Name,
			SkillName:   def.SkillName,
			SkillDesc:   def.SkillDesc,
		}
	}

	dice := make(map[rng.DiceType]*rng.DiceDefinition, len(raw.Dice))
	for key, def := range raw.Dice {
		dt := rng.DiceTypeFromString(key)
		if dt == rng.DiceTypeNone {
			return nil, fmt.Errorf("dice %s: unknown dice type", key)
		}
		dice[dt] = &rng.DiceDefinition{
			Type:        dt,
			EnglishName: def.EnglishName,
			Name:        def.Name,
			Desc:        def.Desc,
			Rank:        def.Rank,
		}
	}

	return &DefinitionSet{
		Events:    events,
		Buffs:     buffs,
		Items:     items,
		MiniGames: miniGames,
		Factions:  factions,
		Dice:      dice,
	}, nil
}

// parseEvaluation converts an evaluation string to a constants.Evaluation value.
// Supports named constants (e.g., "mild_good" → 70) and raw numeric strings (e.g., "70" → 70).
func parseEvaluation(s string) (constants.Evaluation, error) {
	// Try named constant first
	switch s {
	case "very_bad":
		return constants.EvaluationVeryBad, nil
	case "bad":
		return constants.EvaluationBad, nil
	case "mild_bad":
		return constants.EvaluationMildBad, nil
	case "neutral":
		return constants.EvaluationNeutral, nil
	case "mixed":
		return constants.EvaluationMixed, nil
	case "mild_good":
		return constants.EvaluationMildGood, nil
	case "good":
		return constants.EvaluationGood, nil
	case "very_good":
		return constants.EvaluationVeryGood, nil
	case "excellent":
		return constants.EvaluationExcellent, nil
	}

	// Try numeric value
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("invalid evaluation: %s (expected named constant or 0-100 number)", s)
	}
	eval := constants.Evaluation(n)
	if !eval.IsValid() {
		return 0, fmt.Errorf("evaluation %d out of range (0-100)", n)
	}
	return eval, nil
}

// ========== YAML Intermediate Structs ==========

type yamlDefinitions struct {
	Events    map[string]yamlEventDef    `yaml:"events"`
	Buffs     map[string]yamlBuffDef     `yaml:"buffs"`
	Items     map[string]yamlItemDef     `yaml:"items"`
	MiniGames map[string]yamlMiniGameDef `yaml:"mini_games"`
	Factions  map[string]yamlFactionDef  `yaml:"factions"`
	Dice      map[string]yamlDiceDef     `yaml:"dice"`
}

type yamlEventDef struct {
	Evaluation  string `yaml:"evaluation"`
	EnglishName string `yaml:"english_name"`
	Name        string `yaml:"name"`
	Desc        string `yaml:"desc"`
}

type yamlBuffDef struct {
	Evaluation  string `yaml:"evaluation"`
	EnglishName string `yaml:"english_name"`
	Name        string `yaml:"name"`
	Desc        string `yaml:"desc"`
	Duration    int    `yaml:"duration"`
}

type yamlItemDef struct {
	Evaluation  string `yaml:"evaluation"`
	EnglishName string `yaml:"english_name"`
	Name        string `yaml:"name"`
	Desc        string `yaml:"desc"`
}

type yamlMiniGameDef struct {
	Mode        string `yaml:"mode"`
	Available   bool   `yaml:"available"`
	EnglishName string `yaml:"english_name"`
	Name        string `yaml:"name"`
	Desc        string `yaml:"desc"`
}

type yamlFactionDef struct {
	EnglishName string `yaml:"english_name"`
	Name        string `yaml:"name"`
	SkillName   string `yaml:"skill_name"`
	SkillDesc   string `yaml:"skill_desc"`
}

type yamlDiceDef struct {
	EnglishName string `yaml:"english_name"`
	Name        string `yaml:"name"`
	Desc        string `yaml:"desc"`
	Rank        int    `yaml:"rank"`
}

// initMiniGamePools populates constants.AllMiniGameTypes and constants.AllOnlineMiniGameTypes
// from loaded GlobalDefinitionSet. Only types with Available=true are included in pools.
func initMiniGamePools() {
	pool := make([]constants.MiniGameType, 0, len(GlobalDefinitionSet.MiniGames))
	onlinePool := make([]constants.MiniGameType, 0)
	for _, def := range GlobalDefinitionSet.MiniGames {
		if def.Available {
			pool = append(pool, def.Type)
			if def.Mode == constants.MiniGameModeRPC {
				onlinePool = append(onlinePool, def.Type)
			}
		}
	}
	constants.AllMiniGameTypes = pool
	constants.AllOnlineMiniGameTypes = onlinePool
}