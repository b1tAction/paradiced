// Package core provides core data structures for the Fated game.
// Importing this package automatically initializes all subpackages (buff, event, item).
package core

// Imports ensure all subpackages are initialized and types are available for re-export.
import (
	"github.com/b1tAction/Fated/internal/core/buff"
	"github.com/b1tAction/Fated/internal/core/event"
	"github.com/b1tAction/Fated/internal/core/item"
	"github.com/b1tAction/Fated/internal/core/types"
	"github.com/b1tAction/Fated/pkg/handler"
)

// Re-export types from subpackages for convenience.
// This allows users to import core and access all types directly.

// Evaluation type from types package
type Evaluation = types.Evaluation

// Evaluation constants from types package
const (
	EvaluationMin             = types.EvaluationMin
	EvaluationMax             = types.EvaluationMax
	EvaluationBadThreshold    = types.EvaluationBadThreshold
	EvaluationNeutralThreshold = types.EvaluationNeutralThreshold
	EvaluationVeryBad         = types.EvaluationVeryBad
	EvaluationBad             = types.EvaluationBad
	EvaluationMildBad         = types.EvaluationMildBad
	EvaluationNeutral         = types.EvaluationNeutral
	EvaluationMixed           = types.EvaluationMixed
	EvaluationMildGood        = types.EvaluationMildGood
	EvaluationGood            = types.EvaluationGood
	EvaluationVeryGood        = types.EvaluationVeryGood
	EvaluationExcellent       = types.EvaluationExcellent
)

// SpecialEffect type from types package
type SpecialEffect = types.SpecialEffect

// SpecialEffect constants from types package
const (
	SpecialNone          = types.SpecialNone
	SpecialImmune        = types.SpecialImmune
	SpecialReverse       = types.SpecialReverse
	SpecialImmunePoison  = types.SpecialImmunePoison
	SpecialBadEvent      = types.SpecialBadEvent
	SpecialZhuQuePassive = types.SpecialZhuQuePassive
	SpecialTeleport      = types.SpecialTeleport
	SpecialDiceSwap      = types.SpecialDiceSwap
	SpecialDiceUpgrade   = types.SpecialDiceUpgrade
	SpecialGiveLost      = types.SpecialGiveLost
	SpecialDrawItem      = types.SpecialDrawItem
	SpecialLoseItem      = types.SpecialLoseItem
	SpecialSwapPosition  = types.SpecialSwapPosition
	SpecialRandomBuff    = types.SpecialRandomBuff
)

// BuffType from buff package
type BuffType = buff.BuffType

// BuffType constants from buff package
const (
	BuffTypeNone     = buff.BuffTypeNone
	BuffTypeCurse    = buff.BuffTypeCurse
	BuffTypeLost     = buff.BuffTypeLost
	BuffTypeCorrupt  = buff.BuffTypeCorrupt
	BuffTypePoison   = buff.BuffTypePoison
	BuffTypeHidden   = buff.BuffTypeHidden
	BuffTypeDivine   = buff.BuffTypeDivine
	BuffTypeRain     = buff.BuffTypeRain
	BuffTypeExorcism = buff.BuffTypeExorcism
	BuffTypeFire     = buff.BuffTypeFire
)

// Buff from buff package
type Buff = buff.Buff

// NewBuff from buff package
var NewBuff = buff.NewBuff

// BuffDefinition from buff package
type BuffDefinition = buff.BuffDefinition

// EffectHandler from handler package (unified for Buff/Item/Event)
type EffectHandler = handler.EffectHandler

// Buff registry access functions from buff package
var (
	GetBuffDefinition    = buff.GetBuffDefinition
	GetBuffString        = buff.GetBuffString
	GetBuffEvaluation    = buff.GetBuffEvaluation
	GetBuffHandler       = buff.GetBuffHandler
	HasBuffHandler       = buff.HasBuffHandler
	GetAllBuffTypes      = buff.GetAllBuffTypes
	GetBuffTypesByCategory = buff.GetBuffTypesByCategory
	GetAllBuffDefinitions = buff.GetAllBuffDefinitions
)

// EventType from event package
type EventType = event.EventType

// EventType constants from event package
const (
	EventTypeNone        = event.EventTypeNone
	EventTypeHerb        = event.EventTypeHerb
	EventTypeMilkTea     = event.EventTypeMilkTea
	EventTypeRelic       = event.EventTypeRelic
	EventTypeDivineBless = event.EventTypeDivineBless
	EventTypeExchange    = event.EventTypeExchange
	EventTypeHiddenBuff  = event.EventTypeHiddenBuff
	EventTypeTasteTest   = event.EventTypeTasteTest
	EventTypeMosquito    = event.EventTypeMosquito
	EventTypeGhostHit    = event.EventTypeGhostHit
	EventTypeDogPoop     = event.EventTypeDogPoop
	EventTypeThief       = event.EventTypeThief
	EventTypeCurseBuddha = event.EventTypeCurseBuddha
	EventTypeLostWay     = event.EventTypeLostWay
	EventTypeThunder     = event.EventTypeThunder
)

// EventDefinition from event package
type EventDefinition = event.EventDefinition

// Event registry access functions from event package
var (
	GetEventDefinition    = event.GetEventDefinition
	GetEventString        = event.GetEventString
	GetEventEvaluation    = event.GetEventEvaluation
	GetAllEventTypes      = event.GetAllEventTypes
	GetEventTypesByCategory = event.GetEventTypesByCategory
	GetAllEventDefinitions = event.GetAllEventDefinitions
)

// ItemType from item package
type ItemType = item.ItemType

// ItemType constants from item package
const (
	ItemTypeNone        = item.ItemTypeNone
	ItemTypeReverseClock = item.ItemTypeReverseClock
	ItemTypeAnyDoor     = item.ItemTypeAnyDoor
	ItemTypeDiceSwap    = item.ItemTypeDiceSwap
	ItemTypeDiceUpgrade = item.ItemTypeDiceUpgrade
)

// Item from item package
type Item = item.Item

// NewItem from item package
var NewItem = item.NewItem

// ItemDefinition from item package
type ItemDefinition = item.ItemDefinition

// Item registry access functions from item package
var (
	GetItemDefinition    = item.GetItemDefinition
	GetItemString        = item.GetItemString
	GetItemEvaluation    = item.GetItemEvaluation
	GetAllItemTypes      = item.GetAllItemTypes
	GetItemTypesByCategory = item.GetItemTypesByCategory
	GetAllItemDefinitions = item.GetAllItemDefinitions
	GenerateItemID       = item.GenerateItemID
)

// GlobalBuffRegistry from buff package (for advanced use)
var GlobalBuffRegistry = buff.GlobalBuffRegistry

// GlobalEventRegistry from event package (for advanced use)
var GlobalEventRegistry = event.GlobalEventRegistry

// GlobalItemRegistry from item package (for advanced use)
var GlobalItemRegistry = item.GlobalItemRegistry

// CombinedRegistry provides a unified interface for all registries.
// Used for backward compatibility with tests and legacy code.
type CombinedRegistry struct{}

// GlobalRegistry is the combined registry for backward compatibility.
var GlobalRegistry = &CombinedRegistry{}

// GetBuffTypesByEvaluationRange delegates to BuffRegistry.
func (r *CombinedRegistry) GetBuffTypesByEvaluationRange(minEval, maxEval Evaluation) []BuffType {
	return GlobalBuffRegistry.GetBuffTypesByEvaluationRange(minEval, maxEval)
}

// GetEventTypesByEvaluationRange delegates to EventRegistry.
func (r *CombinedRegistry) GetEventTypesByEvaluationRange(minEval, maxEval Evaluation) []EventType {
	return GlobalEventRegistry.GetEventTypesByEvaluationRange(minEval, maxEval)
}

// GetItemTypesByEvaluationRange delegates to ItemRegistry.
func (r *CombinedRegistry) GetItemTypesByEvaluationRange(minEval, maxEval Evaluation) []ItemType {
	return GlobalItemRegistry.GetItemTypesByEvaluationRange(minEval, maxEval)
}