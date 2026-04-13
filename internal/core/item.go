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

func (it ItemType) String() string {
	names := map[ItemType]string{
		ItemTypeNone:         "None",
		ItemTypeReverseClock: "ReverseClock",
		ItemTypeAnyDoor:      "AnyDoor",
		ItemTypeDiceSwap:     "DiceSwap",
		ItemTypeDiceUpgrade:  "DiceUpgrade",
	}
	if name, ok := names[it]; ok {
		return name
	}
	return "Unknown"
}

func (it ItemType) IsValid() bool {
	return it > ItemTypeNone && it <= ItemTypeDiceUpgrade
}

// GetEvaluation returns the item's evaluation score.
func (it ItemType) GetEvaluation() Evaluation {
	evalMap := map[ItemType]Evaluation{
		ItemTypeReverseClock: EvaluationGood,     // ReverseClock: good (negative for others)
		ItemTypeAnyDoor:      EvaluationNeutral,  // AnyDoor: neutral
		ItemTypeDiceSwap:     EvaluationNeutral,  // DiceSwap: neutral
		ItemTypeDiceUpgrade:  EvaluationGood,     // DiceUpgrade: good
	}
	if eval, ok := evalMap[it]; ok {
		return eval
	}
	return EvaluationNeutral
}

// GetCategory returns the item's category (based on Evaluation).
func (it ItemType) GetCategory() string {
	return it.GetEvaluation().GetCategory()
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
	Type        ItemType    `json:"type"`
	Eval        Evaluation  `json:"evaluation"`
	Name        string      `json:"name"`
	Desc        string      `json:"desc"`
	TargetSelf  bool        `json:"target_self"`
	TargetOther bool        `json:"target_other"`
	BuffType    BuffType    `json:"buff_type"`
	Range       int         `json:"range"`
	Phase       event.Phase `json:"phase"`        // Usable phase
	Priority    int         `json:"priority"`     // Execution priority
	NeedConfirm bool        `json:"need_confirm"` // Whether user confirmation is needed (default true)
}

func (it ItemType) GetItemDefinition() *ItemDefinition {
	eval := it.GetEvaluation()
	definitions := map[ItemType]*ItemDefinition{
		ItemTypeReverseClock: {
			Type:        ItemTypeReverseClock,
			Eval:        eval,
			Name:        "反方向的钟",
			Desc:        "给予指定玩家迷途Buff",
			TargetSelf:  false,
			TargetOther: true,
			BuffType:    BuffTypeLost,
			Phase:       event.PhaseAnyTime,
			Priority:    50,
			NeedConfirm: true,
		},
		ItemTypeAnyDoor: {
			Type:        ItemTypeAnyDoor,
			Eval:        eval,
			Name:        "任意门",
			Desc:        "去到30格内指定玩家身边",
			TargetSelf:  false,
			TargetOther: true,
			Range:       30,
			Phase:       event.PhaseOnLand,
			Priority:    60,
			NeedConfirm: true,
		},
		ItemTypeDiceSwap: {
			Type:        ItemTypeDiceSwap,
			Eval:        eval,
			Name:        "骰子交换",
			Desc:        "与指定玩家交换骰子等级",
			TargetSelf:  false,
			TargetOther: true,
			Phase:       event.PhaseAnyTime,
			Priority:    40,
			NeedConfirm: true,
		},
		ItemTypeDiceUpgrade: {
			Type:        ItemTypeDiceUpgrade,
			Eval:        eval,
			Name:        "骰子升级卡",
			Desc:        "将当前骰子升级为更高等级",
			TargetSelf:  true,
			TargetOther: false,
			Phase:       event.PhaseBeforeTurn,
			Priority:    70,
			NeedConfirm: true,
		},
	}
	if def, ok := definitions[it]; ok {
		return def
	}
	return nil
}

// ========== Item Registry ==========

type ItemRegistry struct {
	AllItems     []ItemType `json:"all_items"`
	GoodItems    []ItemType `json:"good_items"`
	NeutralItems []ItemType `json:"neutral_items"`
	BadItems     []ItemType `json:"bad_items"`
}

func NewItemRegistry() *ItemRegistry {
	all := []ItemType{
		ItemTypeReverseClock, ItemTypeAnyDoor, ItemTypeDiceSwap, ItemTypeDiceUpgrade,
	}

	var good, neutral, bad []ItemType
	for _, it := range all {
		eval := it.GetEvaluation()
		if eval.IsGood() {
			good = append(good, it)
		} else if eval.IsNeutral() {
			neutral = append(neutral, it)
		} else {
			bad = append(bad, it)
		}
	}

	return &ItemRegistry{
		AllItems:     all,
		GoodItems:    good,
		NeutralItems: neutral,
		BadItems:     bad,
	}
}

// GetItemsByEvaluationRange returns Items within the specified Evaluation range.
func (ir *ItemRegistry) GetItemsByEvaluationRange(minEval, maxEval Evaluation) []ItemType {
	var result []ItemType
	for _, it := range ir.AllItems {
		eval := it.GetEvaluation()
		if eval >= minEval && eval <= maxEval {
			result = append(result, it)
		}
	}
	return result
}

// GetItemsByCategory returns Items by category.
func (ir *ItemRegistry) GetItemsByCategory(category string) []ItemType {
	switch category {
	case "Good":
		return ir.GoodItems
	case "Neutral":
		return ir.NeutralItems
	case "Bad":
		return ir.BadItems
	}
	return ir.AllItems
}

// GetAllItemDefinitions returns all Item definitions.
func (ir *ItemRegistry) GetAllItemDefinitions() []*ItemDefinition {
	defs := make([]*ItemDefinition, 0, len(ir.AllItems))
	for _, it := range ir.AllItems {
		def := it.GetItemDefinition()
		if def != nil {
			defs = append(defs, def)
		}
	}
	return defs
}

// GenerateItemID generates an Item ID (for engine package use).
func GenerateItemID() string {
	return fmt.Sprintf("item-%d", time.Now().UnixNano())
}