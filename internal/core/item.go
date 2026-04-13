package core

import (
	"fmt"
	"time"

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

// String returns the Item type name from GlobalRegistry.
func (it ItemType) String() string {
	return GlobalRegistry.GetItemString(it)
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
	Type          ItemType      `json:"type"`
	Eval          Evaluation    `json:"evaluation"`
	EnglishName   string        `json:"english_name"` // English identifier (for String())
	Name          string        `json:"name"`         // Chinese display name
	Desc          string        `json:"desc"`
	TargetSelf    bool          `json:"target_self"`
	TargetOther   bool          `json:"target_other"`
	BuffType      BuffType      `json:"buff_type"`
	Range         int           `json:"range"`
	SpecialEffect SpecialEffect `json:"special_effect"` // Special effect type
	Phase         event.Phase   `json:"phase"`          // Usable phase
	Priority      int           `json:"priority"`       // Execution priority
	NeedConfirm   bool          `json:"need_confirm"`   // Whether user confirmation is needed (default true)
}

// ========== Global Registry Access Functions ==========

// GetItemDefinition returns the Item definition from GlobalRegistry.
func GetItemDefinition(it ItemType) *ItemDefinition {
	return GlobalRegistry.GetItemDefinition(it)
}

// GetItemString returns the Item name string from GlobalRegistry.
func GetItemString(it ItemType) string {
	return GlobalRegistry.GetItemString(it)
}

// GetItemEvaluation returns the Item evaluation score from GlobalRegistry.
func GetItemEvaluation(it ItemType) Evaluation {
	return GlobalRegistry.GetItemEvaluation(it)
}

// GetAllItemTypes returns all registered Item types.
func GetAllItemTypes() []ItemType {
	return GlobalRegistry.GetAllItemTypes()
}

// GetItemTypesByCategory returns Item types by category.
func GetItemTypesByCategory(category string) []ItemType {
	return GlobalRegistry.GetItemTypesByCategory(category)
}

// GetAllItemDefinitions returns all Item definitions.
func GetAllItemDefinitions() []*ItemDefinition {
	return GlobalRegistry.GetAllItemDefinitions()
}

// GenerateItemID generates an Item ID (for engine package use).
func GenerateItemID() string {
	return fmt.Sprintf("item-%d", time.Now().UnixNano())
}