package engine

import (
	"fmt"

	engineaction "github.com/b1tAction/paradiced/internal/engine/action"
	"github.com/b1tAction/paradiced/internal/core"
	"github.com/b1tAction/paradiced/internal/event"
	"github.com/b1tAction/paradiced/pkg/constants"
	"github.com/b1tAction/paradiced/pkg/resource"
	"github.com/b1tAction/paradiced/pkg/rng"
)

// EventHandlerConfig contains effect logic and priority for Event.
type EventHandlerConfig struct {
	Priority int            `json:"priority"`
	Handler  EffectHandler  `json:"-"`
}

// EventRegistry is the registry for Event definitions and handler configs.
type EventRegistry struct {
	defs    map[constants.EventType]*constants.EventDefinition
	configs map[constants.EventType]*EventHandlerConfig
	names   map[constants.EventType]string

	goodEvents    []constants.EventType
	badEvents     []constants.EventType
	neutralEvents []constants.EventType
}

// GlobalEventRegistry is the global Event registry.
var GlobalEventRegistry *EventRegistry

func init() {
	GlobalEventRegistry = NewEventRegistry()
	registerAllEvents()
}

// NewEventRegistry creates a new Event registry.
func NewEventRegistry() *EventRegistry {
	return &EventRegistry{
		defs:          make(map[constants.EventType]*constants.EventDefinition),
		configs:       make(map[constants.EventType]*EventHandlerConfig),
		names:         make(map[constants.EventType]string),
		goodEvents:    make([]constants.EventType, 0),
		badEvents:     make([]constants.EventType, 0),
		neutralEvents: make([]constants.EventType, 0),
	}
}

// RegisterEvent registers an Event definition with handler config.
func (r *EventRegistry) RegisterEvent(def *constants.EventDefinition, config *EventHandlerConfig) {
	if def == nil || def.Type == constants.EventTypeNone {
		return
	}

	r.defs[def.Type] = def
	r.names[def.Type] = def.Name

	if def.Eval.IsGood() {
		r.goodEvents = append(r.goodEvents, def.Type)
	} else if def.Eval.IsBad() {
		r.badEvents = append(r.badEvents, def.Type)
	} else {
		r.neutralEvents = append(r.neutralEvents, def.Type)
	}

	if config != nil {
		r.configs[def.Type] = config
	}
}

// GetEventDefinition returns the Event definition by type.
func GetEventDefinition(et constants.EventType) *constants.EventDefinition {
	return GlobalEventRegistry.GetEventDefinition(et)
}

func (r *EventRegistry) GetEventDefinition(et constants.EventType) *constants.EventDefinition {
	if def, ok := r.defs[et]; ok {
		return def
	}
	return nil
}

// GetEventHandlerConfig returns the Event's handler config.
func GetEventHandlerConfig(et constants.EventType) *EventHandlerConfig {
	return GlobalEventRegistry.GetEventHandlerConfig(et)
}

func (r *EventRegistry) GetEventHandlerConfig(et constants.EventType) *EventHandlerConfig {
	if config, ok := r.configs[et]; ok {
		return config
	}
	return nil
}

// HasEventHandler checks if an Event type has a registered handler.
func HasEventHandler(et constants.EventType) bool {
	return GlobalEventRegistry.HasEventHandler(et)
}

func (r *EventRegistry) HasEventHandler(et constants.EventType) bool {
	config, ok := r.configs[et]
	return ok && config != nil && config.Handler != nil
}

// GetEventName returns the Event Chinese display name.
func GetEventName(et constants.EventType) string {
	return GlobalEventRegistry.GetEventName(et)
}

func (r *EventRegistry) GetEventName(et constants.EventType) string {
	if name, ok := r.names[et]; ok {
		return name
	}
	return ""
}

// GetAllEventTypes returns all registered Event types.
func GetAllEventTypes() []constants.EventType {
	return GlobalEventRegistry.GetAllEventTypes()
}

func (r *EventRegistry) GetAllEventTypes() []constants.EventType {
	result := make([]constants.EventType, 0, len(r.defs))
	for et := range r.defs {
		result = append(result, et)
	}
	return result
}

// GetEventTypesByCategory returns Event types by category.
func GetEventTypesByCategory(category string) []constants.EventType {
	return GlobalEventRegistry.GetEventTypesByCategory(category)
}

func (r *EventRegistry) GetEventTypesByCategory(category string) []constants.EventType {
	switch category {
	case "Good":
		return r.goodEvents
	case "Bad":
		return r.badEvents
	case "Neutral":
		return r.neutralEvents
	}
	return r.GetAllEventTypes()
}

// BuildEventPool builds an []*rng.EvaluatedItem pool from all registered EventDefinitions.
// This is the single source of truth for event pool data — no need to manually
// duplicate Type+Eval mappings elsewhere.
func BuildEventPool() []*rng.EvaluatedItem {
	return GlobalEventRegistry.buildEventPool()
}

func (r *EventRegistry) buildEventPool() []*rng.EvaluatedItem {
	pool := make([]*rng.EvaluatedItem, 0, len(r.defs))
	for _, def := range r.defs {
		pool = append(pool, &rng.EvaluatedItem{
			Type: string(def.Type),
			Eval: def.Eval,
		})
	}
	return pool
}

// ========== Event Handler Registration ==========

func registerAllEvents() {
	defs := resource.GlobalDefinitionSet

	// Good Events (Evaluation > 65)

	// Herb: HP+1
	GlobalEventRegistry.RegisterEvent(defs.Events[constants.EventTypeHerb], &EventHandlerConfig{
		Priority: 60,
		Handler:  createEventModifyHPHandler(1, constants.SourceEventHerb),
	})

	// LuckyBubble: LP+1
	GlobalEventRegistry.RegisterEvent(defs.Events[constants.EventTypeLuckyBubble], &EventHandlerConfig{
		Priority: 70,
		Handler:  createEventModifyLPHandler(1, constants.SourceEventLuckyBubble),
	})

	// Relic: Draw item
	GlobalEventRegistry.RegisterEvent(defs.Events[constants.EventTypeRelic], &EventHandlerConfig{
		Priority: 80,
		Handler:  handleDrawItem,
	})

	// DivineBless: Give Divine buff
	GlobalEventRegistry.RegisterEvent(defs.Events[constants.EventTypeDivineBless], &EventHandlerConfig{
		Priority: 80,
		Handler:  createEventGiveBuffHandler(constants.BuffTypeDivine, constants.SourceEventDivineBless),
	})

	// Neutral Events (Evaluation 41-65)

	// Exchange: Swap position with another player
	GlobalEventRegistry.RegisterEvent(defs.Events[constants.EventTypeExchange], &EventHandlerConfig{
		Priority: 50,
		Handler:  handleSwapPosition,
	})

	// HiddenBuff: Give Hidden buff
	GlobalEventRegistry.RegisterEvent(defs.Events[constants.EventTypeHiddenBuff], &EventHandlerConfig{
		Priority: 60,
		Handler:  createEventGiveBuffHandler(constants.BuffTypeHidden, constants.SourceEventHiddenBuff),
	})

	// TasteTest: Random buff effect
	GlobalEventRegistry.RegisterEvent(defs.Events[constants.EventTypeTasteTest], &EventHandlerConfig{
		Priority: 50,
		Handler:  handleRandomBuff,
	})

	// Bad Events (Evaluation <= 40)

	// Mosquito: HP-1
	GlobalEventRegistry.RegisterEvent(defs.Events[constants.EventTypeMosquito], &EventHandlerConfig{
		Priority: 40,
		Handler:  createEventModifyHPHandler(-1, constants.SourceEventMosquito),
	})

	// GhostHit: HP-1
	GlobalEventRegistry.RegisterEvent(defs.Events[constants.EventTypeGhostHit], &EventHandlerConfig{
		Priority: 40,
		Handler:  createEventModifyHPHandler(-1, constants.SourceEventGhostHit),
	})

	// DogPoop: LP-1
	GlobalEventRegistry.RegisterEvent(defs.Events[constants.EventTypeDogPoop], &EventHandlerConfig{
		Priority: 40,
		Handler:  createEventModifyLPHandler(-1, constants.SourceEventDogPoop),
	})

	// WindGust: Lose random item
	GlobalEventRegistry.RegisterEvent(defs.Events[constants.EventTypeWindGust], &EventHandlerConfig{
		Priority: 30,
		Handler:  handleLoseItem,
	})

	// SkullGaze: Give Curse buff
	GlobalEventRegistry.RegisterEvent(defs.Events[constants.EventTypeSkullGaze], &EventHandlerConfig{
		Priority: 30,
		Handler:  createEventGiveBuffHandler(constants.BuffTypeCurse, constants.SourceEventSkullGaze),
	})

	// LostWay: Give Lost buff
	GlobalEventRegistry.RegisterEvent(defs.Events[constants.EventTypeLostWay], &EventHandlerConfig{
		Priority: 40,
		Handler:  createEventGiveBuffHandler(constants.BuffTypeLost, constants.SourceEventLostWay),
	})

	// Thunder: HP to 0 (death)
	GlobalEventRegistry.RegisterEvent(defs.Events[constants.EventTypeThunder], &EventHandlerConfig{
		Priority: 20,
		Handler:  handleThunderDeath,
	})
}

// ========== Event Handler Helpers ==========

// createEventModifyHPHandler creates a handler that modifies HP through Action system.
func createEventModifyHPHandler(amount int, source constants.ActionSource) EffectHandler {
	return func(phase constants.Phase, ctx *event.Context) error {
		if ctx == nil {
			return fmt.Errorf("handler: event context is nil")
		}
		if ctx.Player == nil {
			return fmt.Errorf("handler: player is nil in event context")
		}
		if amount == 0 {
			return nil
		}

		// Check ActionContext exists
		actionCtx, err := getActionCtxFromEventCtx(ctx)
		if err != nil {
			return err
		}
		_ = actionCtx // ActionContext used for derived action processing

		if amount > 0 {
			ctx.AddDerivedAction(engineaction.NewHealAction(ctx.Player, amount, string(source)))
		} else {
			ctx.AddDerivedAction(engineaction.NewDamageAction(ctx.Player, -amount, string(source)))
		}
		return nil
	}
}

// createEventModifyLPHandler creates a handler that modifies LP through Action system.
func createEventModifyLPHandler(amount int, source constants.ActionSource) EffectHandler {
	return func(phase constants.Phase, ctx *event.Context) error {
		if ctx == nil {
			return fmt.Errorf("handler: event context is nil")
		}
		if ctx.Player == nil {
			return fmt.Errorf("handler: player is nil in event context")
		}

		// Check ActionContext exists
		actionCtx, err := getActionCtxFromEventCtx(ctx)
		if err != nil {
			return err
		}
		_ = actionCtx // ActionContext used for derived action processing

		ctx.AddDerivedAction(engineaction.NewModifyLPAction(ctx.Player, amount, string(source)))
		return nil
	}
}

// createEventGiveBuffHandler creates a handler that gives a Buff through Action system.
func createEventGiveBuffHandler(buffType constants.BuffType, source constants.ActionSource) EffectHandler {
	return func(phase constants.Phase, ctx *event.Context) error {
		if ctx == nil {
			return fmt.Errorf("handler: event context is nil")
		}
		if ctx.Player == nil {
			return fmt.Errorf("handler: player is nil in event context")
		}

		// Check ActionContext exists
		actionCtx, err := getActionCtxFromEventCtx(ctx)
		if err != nil {
			return err
		}
		_ = actionCtx // ActionContext used for derived action processing

		ctx.AddDerivedAction(engineaction.NewAddBuffAction(ctx.Player, buffType, string(source)))
		return nil
	}
}

func handleDrawItem(phase constants.Phase, ctx *event.Context) error {
	if ctx == nil || ctx.Player == nil {
		return nil
	}

	// Check ActionContext exists
	actionCtx, err := getActionCtxFromEventCtx(ctx)
	if err != nil {
		return err
	}
	_ = actionCtx // ActionContext used for derived action processing

	// Produce DrawItemAction as DerivedAction
	ctx.AddDerivedAction(engineaction.NewDrawItemAction(ctx.Player, string(constants.SourceEventRelic)))
	return nil
}

func handleSwapPosition(phase constants.Phase, ctx *event.Context) error {
	if ctx == nil || ctx.Player == nil {
		return nil
	}

	// Check ActionContext exists
	actionCtx, err := getActionCtxFromEventCtx(ctx)
	if err != nil {
		return err
	}

	// Find another player to swap with via Game interface
	if actionCtx.Game == nil {
		return nil
	}

	playerInterfaces := actionCtx.Game.GetPlayersInterface()
	if len(playerInterfaces) < 2 {
		return nil // Need at least 2 players to swap
	}

	// Find first valid non-self, non-dead player
	var targetPlayer *core.Player
	for _, pi := range playerInterfaces {
		p, ok := pi.(*core.Player)
		if !ok {
			continue
		}
		if p.ID.UUID() != ctx.Player.ID.UUID() && !p.IsDead {
			targetPlayer = p
			break
		}
	}
	if targetPlayer == nil {
		return nil // No valid swap target
	}

	// Save positions before teleporting
	playerPos := ctx.Player.Position
	targetPos := targetPlayer.Position

	// Produce two TeleportActions for the swap
	ctx.AddDerivedAction(engineaction.NewTeleportAction(ctx.Player, targetPos, string(constants.SourceEventExchange)))
	ctx.AddDerivedAction(engineaction.NewTeleportAction(targetPlayer, playerPos, string(constants.SourceEventExchange)))
	return nil
}

func handleRandomBuff(phase constants.Phase, ctx *event.Context) error {
	if ctx == nil || ctx.Player == nil {
		return nil
	}

	// Check ActionContext exists
	actionCtx, err := getActionCtxFromEventCtx(ctx)
	if err != nil {
		return err
	}
	_ = actionCtx // ActionContext used for derived action processing

	// Produce DrawBuffAction as DerivedAction
	ctx.AddDerivedAction(engineaction.NewDrawBuffAction(ctx.Player, string(constants.SourceEventTasteTest)))
	return nil
}

func handleLoseItem(phase constants.Phase, ctx *event.Context) error {
	if ctx == nil || ctx.Player == nil {
		return nil
	}

	// Check ActionContext exists
	actionCtx, err := getActionCtxFromEventCtx(ctx)
	if err != nil {
		return err
	}
	_ = actionCtx // ActionContext used for derived action processing

	// Select a random item from player's inventory
	if len(ctx.Player.Inventory) == 0 {
		return nil // No items to lose
	}

	// Take the first item from inventory as the lost item
	// (In production, this would use RNG for random selection)
	lostItem := ctx.Player.Inventory[0]
	ctx.AddDerivedAction(engineaction.NewRemoveItemAction(ctx.Player, lostItem.Type, string(constants.SourceEventWindGust)))
	return nil
}

func handleThunderDeath(phase constants.Phase, ctx *event.Context) error {
	if ctx == nil || ctx.Player == nil {
		return nil
	}

	source := string(constants.SourceEventThunder)

	// Deal massive damage to set HP to 0
	// Use player's current HP as damage amount
	currentHP := ctx.Player.HP
	if currentHP > 0 {
		action := engineaction.NewDamageAction(ctx.Player, currentHP, source)
		ctx.AddDerivedAction(action)
	}

	ctx.SetBool("instant_death", true)
	return nil
}