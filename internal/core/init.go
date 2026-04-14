// Package core provides core data structures for the Fated game.
// Importing this package automatically initializes all subpackages (buff, event, item).
package core

// Imports ensure all subpackages are initialized and types are available for re-export.
import (
	"github.com/b1tAction/fated/internal/core/buff"
	"github.com/b1tAction/fated/internal/core/event"
	"github.com/b1tAction/fated/internal/core/item"
	"github.com/b1tAction/fated/pkg/constants"
	"github.com/b1tAction/fated/pkg/handler"
)

// Re-export types from subpackages for convenience.
// This allows users to import core and access all types directly.

// BuffType from constants package
type BuffType = constants.BuffType

// BuffType constants from constants package
const (
	BuffTypeNone     = constants.BuffTypeNone
	BuffTypeCurse    = constants.BuffTypeCurse
	BuffTypeLost     = constants.BuffTypeLost
	BuffTypeCorrupt  = constants.BuffTypeCorrupt
	BuffTypePoison   = constants.BuffTypePoison
	BuffTypeHidden   = constants.BuffTypeHidden
	BuffTypeDivine   = constants.BuffTypeDivine
	BuffTypeRain     = constants.BuffTypeRain
	BuffTypeExorcism = constants.BuffTypeExorcism
	BuffTypeFire     = constants.BuffTypeFire
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
	GetBuffDefinition      = buff.GetBuffDefinition
	GetBuffString          = buff.GetBuffString
	GetBuffEvaluation      = buff.GetBuffEvaluation
	GetBuffHandler         = buff.GetBuffHandler
	HasBuffHandler         = buff.HasBuffHandler
	GetAllBuffTypes        = buff.GetAllBuffTypes
	GetBuffTypesByCategory = buff.GetBuffTypesByCategory
	GetAllBuffDefinitions  = buff.GetAllBuffDefinitions
)

// EventType from constants package
type EventType = constants.EventType

// EventType constants from constants package
const (
	EventTypeNone        = constants.EventTypeNone
	EventTypeHerb        = constants.EventTypeHerb
	EventTypeMilkTea     = constants.EventTypeMilkTea
	EventTypeRelic       = constants.EventTypeRelic
	EventTypeDivineBless = constants.EventTypeDivineBless
	EventTypeExchange    = constants.EventTypeExchange
	EventTypeHiddenBuff  = constants.EventTypeHiddenBuff
	EventTypeTasteTest   = constants.EventTypeTasteTest
	EventTypeMosquito    = constants.EventTypeMosquito
	EventTypeGhostHit    = constants.EventTypeGhostHit
	EventTypeDogPoop     = constants.EventTypeDogPoop
	EventTypeThief       = constants.EventTypeThief
	EventTypeCurseBuddha = constants.EventTypeCurseBuddha
	EventTypeLostWay     = constants.EventTypeLostWay
	EventTypeThunder     = constants.EventTypeThunder
)

// EventDefinition from event package
type EventDefinition = event.EventDefinition

// Event registry access functions from event package
var (
	GetEventDefinition      = event.GetEventDefinition
	GetEventString          = event.GetEventString
	GetEventEvaluation      = event.GetEventEvaluation
	GetAllEventTypes        = event.GetAllEventTypes
	GetEventTypesByCategory = event.GetEventTypesByCategory
	GetAllEventDefinitions  = event.GetAllEventDefinitions
)

// ItemType from constants package
type ItemType = constants.ItemType

// ItemType constants from constants package
const (
	ItemTypeNone         = constants.ItemTypeNone
	ItemTypeReverseClock = constants.ItemTypeReverseClock
	ItemTypeAnyDoor      = constants.ItemTypeAnyDoor
	ItemTypeDiceSwap     = constants.ItemTypeDiceSwap
	ItemTypeDiceUpgrade  = constants.ItemTypeDiceUpgrade
)

// Item from item package
type Item = item.Item

// NewItem from item package
var NewItem = item.NewItem

// ItemDefinition from item package
type ItemDefinition = item.ItemDefinition

// Item registry access functions from item package
var (
	GetItemDefinition      = item.GetItemDefinition
	GetItemString          = item.GetItemString
	GetItemEvaluation      = item.GetItemEvaluation
	GetAllItemTypes        = item.GetAllItemTypes
	GetItemTypesByCategory = item.GetItemTypesByCategory
	GetAllItemDefinitions  = item.GetAllItemDefinitions
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
func (r *CombinedRegistry) GetBuffTypesByEvaluationRange(minEval, maxEval constants.Evaluation) []constants.BuffType {
	return GlobalBuffRegistry.GetBuffTypesByEvaluationRange(minEval, maxEval)
}

// GetEventTypesByEvaluationRange delegates to EventRegistry.
func (r *CombinedRegistry) GetEventTypesByEvaluationRange(minEval, maxEval constants.Evaluation) []constants.EventType {
	return GlobalEventRegistry.GetEventTypesByEvaluationRange(minEval, maxEval)
}

// GetItemTypesByEvaluationRange delegates to ItemRegistry.
func (r *CombinedRegistry) GetItemTypesByEvaluationRange(minEval, maxEval constants.Evaluation) []constants.ItemType {
	return GlobalItemRegistry.GetItemTypesByEvaluationRange(minEval, maxEval)
}
