// Package item provides Item related data structures and registry.
// This package is independently usable via Direct Import.
package item

import (
	"fmt"
	"time"

	"github.com/b1tAction/Fated/internal/core/buff"
	"github.com/b1tAction/Fated/internal/core/types"
	"github.com/b1tAction/Fated/pkg/event"
)

// ========== Item Type Definitions ==========

type ItemType int

const (
	ItemTypeNone ItemType = iota
	ItemTypeReverseClock // ReverseClock反方向的钟
	ItemTypeAnyDoor      // AnyDoor任意门
	ItemTypeDiceSwap     // DiceSwap骰子交换
	ItemTypeDiceUpgrade  // DiceUpgrade骰子升级卡
)

// IsValid checks if the Item type is valid.
func (it ItemType) IsValid() bool {
	return it > ItemTypeNone && it <= ItemTypeDiceUpgrade
}

// String returns the Item type name from GlobalItemRegistry.
func (it ItemType) String() string {
	return GlobalItemRegistry.GetItemString(it)
}

// ========== Item Instance ==========

type Item struct {
	Type           ItemType `json:"type"`
	ID             string   `json:"id"`
	Usable         bool     `json:"usable"`
	TargetID       string   `json:"target_id"`
	SubscriptionID string   `json:"subscription_id"` // EventBus subscription ID (managed by engine package)
}

func NewItem(itemType ItemType, id string) *Item {
	return &Item{
		Type:   itemType,
		ID:     id,
		Usable: true,
	}
}

// ========== Item Definition ==========

type ItemDefinition struct {
	Type          ItemType             `json:"type"`
	Eval          types.Evaluation     `json:"evaluation"`
	EnglishName   string               `json:"english_name"` // English identifier (for String())
	Name          string               `json:"name"`         // Chinese display name
	Desc          string               `json:"desc"`
	TargetSelf    bool                 `json:"target_self"`
	TargetOther   bool                 `json:"target_other"`
	BuffType      buff.BuffType        `json:"buff_type"`
	Range         int                  `json:"range"`
	SpecialEffect types.SpecialEffect  `json:"special_effect"` // Special effect type
	Phase         event.Phase          `json:"phase"`          // Usable phase
	Priority      int                  `json:"priority"`       // Execution priority
	NeedConfirm   bool                 `json:"need_confirm"`   // Whether user confirmation needed
}

// ========== Item Registry ==========

// ItemRegistry is the registry for Item definitions.
type ItemRegistry struct {
	defs    map[ItemType]*ItemDefinition
	strings map[ItemType]string // English identifier
	names   map[ItemType]string // Chinese name
	evals   map[ItemType]types.Evaluation

	// Category lists (auto-generated)
	goodItems    []ItemType
	neutralItems []ItemType
	badItems     []ItemType
}

// NewItemRegistry creates a new Item registry.
func NewItemRegistry() *ItemRegistry {
	return &ItemRegistry{
		defs:         make(map[ItemType]*ItemDefinition),
		strings:      make(map[ItemType]string),
		names:        make(map[ItemType]string),
		evals:        make(map[ItemType]types.Evaluation),
		goodItems:    make([]ItemType, 0),
		neutralItems: make([]ItemType, 0),
		badItems:     make([]ItemType, 0),
	}
}

// RegisterItem registers an Item definition.
func (r *ItemRegistry) RegisterItem(def *ItemDefinition) {
	if def == nil || def.Type == ItemTypeNone {
		return
	}

	r.defs[def.Type] = def
	r.strings[def.Type] = def.EnglishName
	r.names[def.Type] = def.Name
	r.evals[def.Type] = def.Eval

	// Auto-classify by Evaluation
	if def.Eval.IsGood() {
		r.goodItems = append(r.goodItems, def.Type)
	} else if def.Eval.IsBad() {
		r.badItems = append(r.badItems, def.Type)
	} else {
		r.neutralItems = append(r.neutralItems, def.Type)
	}
}

// GetItemDefinition returns the Item definition by type.
func (r *ItemRegistry) GetItemDefinition(it ItemType) *ItemDefinition {
	if def, ok := r.defs[it]; ok {
		return def
	}
	return nil
}

// GetItemString returns the Item English identifier.
func (r *ItemRegistry) GetItemString(it ItemType) string {
	if name, ok := r.strings[it]; ok {
		return name
	}
	return "Unknown"
}

// GetItemName returns the Item Chinese display name.
func (r *ItemRegistry) GetItemName(it ItemType) string {
	if name, ok := r.names[it]; ok {
		return name
	}
	return "未知"
}

// GetItemEvaluation returns the Item evaluation score.
func (r *ItemRegistry) GetItemEvaluation(it ItemType) types.Evaluation {
	if eval, ok := r.evals[it]; ok {
		return eval
	}
	return types.EvaluationNeutral
}

// GetAllItemTypes returns all registered Item types.
func (r *ItemRegistry) GetAllItemTypes() []ItemType {
	result := make([]ItemType, 0, len(r.defs))
	for it := range r.defs {
		result = append(result, it)
	}
	return result
}

// GetItemTypesByCategory returns Item types by category.
func (r *ItemRegistry) GetItemTypesByCategory(category string) []ItemType {
	switch category {
	case "Good":
		return r.goodItems
	case "Bad":
		return r.badItems
	case "Neutral":
		return r.neutralItems
	}
	return r.GetAllItemTypes()
}

// GetAllItemDefinitions returns all Item definitions.
func (r *ItemRegistry) GetAllItemDefinitions() []*ItemDefinition {
	defs := make([]*ItemDefinition, 0, len(r.defs))
	for _, def := range r.defs {
		defs = append(defs, def)
	}
	return defs
}

// GetItemTypesByEvaluationRange returns Items within the specified Evaluation range.
func (r *ItemRegistry) GetItemTypesByEvaluationRange(minEval, maxEval types.Evaluation) []ItemType {
	var result []ItemType
	for it, eval := range r.evals {
		if eval >= minEval && eval <= maxEval {
			result = append(result, it)
		}
	}
	return result
}

// ========== Global Registry Access Functions ==========

// GetItemDefinition returns the Item definition from GlobalItemRegistry.
func GetItemDefinition(it ItemType) *ItemDefinition {
	return GlobalItemRegistry.GetItemDefinition(it)
}

// GetItemString returns the Item name string from GlobalItemRegistry.
func GetItemString(it ItemType) string {
	return GlobalItemRegistry.GetItemString(it)
}

// GetItemEvaluation returns the Item evaluation score from GlobalItemRegistry.
func GetItemEvaluation(it ItemType) types.Evaluation {
	return GlobalItemRegistry.GetItemEvaluation(it)
}

// GetAllItemTypes returns all registered Item types.
func GetAllItemTypes() []ItemType {
	return GlobalItemRegistry.GetAllItemTypes()
}

// GetItemTypesByCategory returns Item types by category.
func GetItemTypesByCategory(category string) []ItemType {
	return GlobalItemRegistry.GetItemTypesByCategory(category)
}

// GetAllItemDefinitions returns all Item definitions.
func GetAllItemDefinitions() []*ItemDefinition {
	return GlobalItemRegistry.GetAllItemDefinitions()
}

// GenerateItemID generates an Item ID (for engine package use).
func GenerateItemID() string {
	return fmt.Sprintf("item-%d", time.Now().UnixNano())
}