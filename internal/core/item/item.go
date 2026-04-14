// Package item provides Item related data structures and registry.
// This package is independently usable via Direct Import.
package item

import (
	"github.com/b1tAction/fated/pkg/constants"
	"github.com/b1tAction/fated/pkg/id"
)

// ========== Item Instance ==========

type Item struct {
	Type           constants.ItemType `json:"type"`
	ID             id.ItemID          `json:"id"` // Item instance ID (UUID v7)
	Usable         bool               `json:"usable"`
	TargetID       string             `json:"target_id"`       // Target player ID (UUID string for protocol)
	SubscriptionID string             `json:"subscription_id"` // EventBus subscription ID (managed by engine package)
}

// NewItem creates a new Item instance with auto-generated UUID v7 ID.
func NewItem(itemType constants.ItemType) *Item {
	return &Item{
		Type:   itemType,
		ID:     id.NewItemID(),
		Usable: true,
	}
}

// NewItemWithID creates a new Item instance with a specific ID.
// Used for testing and special cases where ID needs to be controlled.
func NewItemWithID(itemType constants.ItemType, itemID id.ItemID) *Item {
	return &Item{
		Type:   itemType,
		ID:     itemID,
		Usable: true,
	}
}

// NewItemWithStringID creates a new Item with ID from string (for backward compatibility).
// Deprecated: Use NewItemWithID with id.ItemID instead.
func NewItemWithStringID(itemType constants.ItemType, idStr string) *Item {
	return &Item{
		Type:   itemType,
		ID:     id.MustParseItemID(idStr),
		Usable: true,
	}
}

// ========== Item Definition ==========

type ItemDefinition struct {
	Type          constants.ItemType      `json:"type"`
	Eval          constants.Evaluation    `json:"evaluation"`
	EnglishName   string                  `json:"english_name"` // English identifier (for String())
	Name          string                  `json:"name"`         // Chinese display name
	Desc          string                  `json:"desc"`
	TargetSelf    bool                    `json:"target_self"`
	TargetOther   bool                    `json:"target_other"`
	BuffType      constants.BuffType      `json:"buff_type"`
	Range         int                     `json:"range"`
	SpecialEffect constants.SpecialEffect `json:"special_effect"` // Special effect type
	Phase         constants.Phase         `json:"phase"`          // Usable phase
	Priority      int                     `json:"priority"`       // Execution priority
	NeedConfirm   bool                    `json:"need_confirm"`   // Whether user confirmation needed
}

// ========== Item Registry ==========

// ItemRegistry is the registry for Item definitions.
type ItemRegistry struct {
	defs    map[constants.ItemType]*ItemDefinition
	strings map[constants.ItemType]string // English identifier
	names   map[constants.ItemType]string // Chinese name
	evals   map[constants.ItemType]constants.Evaluation

	// Category lists (auto-generated)
	goodItems    []constants.ItemType
	neutralItems []constants.ItemType
	badItems     []constants.ItemType
}

// NewItemRegistry creates a new Item registry.
func NewItemRegistry() *ItemRegistry {
	return &ItemRegistry{
		defs:         make(map[constants.ItemType]*ItemDefinition),
		strings:      make(map[constants.ItemType]string),
		names:        make(map[constants.ItemType]string),
		evals:        make(map[constants.ItemType]constants.Evaluation),
		goodItems:    make([]constants.ItemType, 0),
		neutralItems: make([]constants.ItemType, 0),
		badItems:     make([]constants.ItemType, 0),
	}
}

// RegisterItem registers an Item definition.
func (r *ItemRegistry) RegisterItem(def *ItemDefinition) {
	if def == nil || def.Type == constants.ItemTypeNone {
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
func (r *ItemRegistry) GetItemDefinition(it constants.ItemType) *ItemDefinition {
	if def, ok := r.defs[it]; ok {
		return def
	}
	return nil
}

// GetItemString returns the Item English identifier.
func (r *ItemRegistry) GetItemString(it constants.ItemType) string {
	if name, ok := r.strings[it]; ok {
		return name
	}
	return "Unknown"
}

// GetItemName returns the Item Chinese display name.
func (r *ItemRegistry) GetItemName(it constants.ItemType) string {
	if name, ok := r.names[it]; ok {
		return name
	}
	return "未知"
}

// GetItemEvaluation returns the Item evaluation score.
func (r *ItemRegistry) GetItemEvaluation(it constants.ItemType) constants.Evaluation {
	if eval, ok := r.evals[it]; ok {
		return eval
	}
	return constants.EvaluationNeutral
}

// GetAllItemTypes returns all registered Item types.
func (r *ItemRegistry) GetAllItemTypes() []constants.ItemType {
	result := make([]constants.ItemType, 0, len(r.defs))
	for it := range r.defs {
		result = append(result, it)
	}
	return result
}

// GetItemTypesByCategory returns Item types by category.
func (r *ItemRegistry) GetItemTypesByCategory(category string) []constants.ItemType {
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
func (r *ItemRegistry) GetItemTypesByEvaluationRange(minEval, maxEval constants.Evaluation) []constants.ItemType {
	var result []constants.ItemType
	for it, eval := range r.evals {
		if eval >= minEval && eval <= maxEval {
			result = append(result, it)
		}
	}
	return result
}

// ========== Global Registry Access Functions ==========

// GetItemDefinition returns the Item definition from GlobalItemRegistry.
func GetItemDefinition(it constants.ItemType) *ItemDefinition {
	return GlobalItemRegistry.GetItemDefinition(it)
}

// GetItemString returns the Item name string from GlobalItemRegistry.
func GetItemString(it constants.ItemType) string {
	return GlobalItemRegistry.GetItemString(it)
}

// GetItemEvaluation returns the Item evaluation score from GlobalItemRegistry.
func GetItemEvaluation(it constants.ItemType) constants.Evaluation {
	return GlobalItemRegistry.GetItemEvaluation(it)
}

// GetAllItemTypes returns all registered Item types.
func GetAllItemTypes() []constants.ItemType {
	return GlobalItemRegistry.GetAllItemTypes()
}

// GetItemTypesByCategory returns Item types by category.
func GetItemTypesByCategory(category string) []constants.ItemType {
	return GlobalItemRegistry.GetItemTypesByCategory(category)
}

// GetAllItemDefinitions returns all Item definitions.
func GetAllItemDefinitions() []*ItemDefinition {
	return GlobalItemRegistry.GetAllItemDefinitions()
}

// GetItemName returns the Item Chinese display name from GlobalItemRegistry.
func GetItemName(it constants.ItemType) string {
	return GlobalItemRegistry.GetItemName(it)
}
