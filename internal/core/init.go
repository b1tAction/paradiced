// Package core provides core data structures for the Paradiced game.
// Importing this package automatically initializes all subpackages (buff, event, item).
package core

// Imports ensure all subpackages are initialized and types are available for re-export.
import (
	"github.com/b1tAction/paradiced/internal/core/buff"
	"github.com/b1tAction/paradiced/internal/core/event"
	"github.com/b1tAction/paradiced/internal/core/item"
	"github.com/b1tAction/paradiced/pkg/handler"
)

// Re-export types from subpackages for convenience.
// This allows users to import core and access all types directly.

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
	GetBuffHandlerConfig   = buff.GetBuffHandlerConfig
	GetBuffPhases          = buff.GetBuffPhases
	HasBuffHandler         = buff.HasBuffHandler
	GetAllBuffTypes        = buff.GetAllBuffTypes
	GetBuffTypesByCategory = buff.GetBuffTypesByCategory
	GetAllBuffDefinitions  = buff.GetAllBuffDefinitions
)

// BuffHandlerConfig from buff package
type BuffHandlerConfig = buff.BuffHandlerConfig

// Event registry access functions from event package
var (
	GetEventDefinition      = event.GetEventDefinition
	GetEventString          = event.GetEventString
	GetEventEvaluation      = event.GetEventEvaluation
	GetEventHandlerConfig   = event.GetEventHandlerConfig
	GetAllEventTypes        = event.GetAllEventTypes
	GetEventTypesByCategory = event.GetEventTypesByCategory
	GetAllEventDefinitions  = event.GetAllEventDefinitions
)

// EventDefinition from event package
type EventDefinition = event.EventDefinition

// EventHandlerConfig from event package
type EventHandlerConfig = event.EventHandlerConfig

// Item from item package
type Item = item.Item

// NewItem from item package
var NewItem = item.NewItem

// ItemDefinition from item package
type ItemDefinition = item.ItemDefinition

// ItemHandlerConfig from item package
type ItemHandlerConfig = item.ItemHandlerConfig

// Item registry access functions from item package
var (
	GetItemDefinition      = item.GetItemDefinition
	GetItemString          = item.GetItemString
	GetItemEvaluation      = item.GetItemEvaluation
	GetItemHandlerConfig   = item.GetItemHandlerConfig
	GetItemPhase           = item.GetItemPhase
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