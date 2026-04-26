package engine

import (
	"fmt"

	engineaction "github.com/b1tAction/paradiced/internal/engine/action"
	"github.com/b1tAction/paradiced/internal/core"
	"github.com/b1tAction/paradiced/internal/event"
	"github.com/b1tAction/paradiced/pkg/constants"
	"github.com/b1tAction/paradiced/pkg/rng"
)

// EventHandlerConfig contains effect logic and priority for Event.
type EventHandlerConfig struct {
	Priority int            `json:"priority"`
	Handler  EffectHandler  `json:"-"`
}

// EventRegistry is the registry for Event definitions and handler configs.
type EventRegistry struct {
	defs    map[constants.EventType]*core.EventDefinition
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
		defs:          make(map[constants.EventType]*core.EventDefinition),
		configs:       make(map[constants.EventType]*EventHandlerConfig),
		names:         make(map[constants.EventType]string),
		goodEvents:    make([]constants.EventType, 0),
		badEvents:     make([]constants.EventType, 0),
		neutralEvents: make([]constants.EventType, 0),
	}
}

// RegisterEvent registers an Event definition with handler config.
func (r *EventRegistry) RegisterEvent(def *core.EventDefinition, config *EventHandlerConfig) {
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
func GetEventDefinition(et constants.EventType) *core.EventDefinition {
	return GlobalEventRegistry.GetEventDefinition(et)
}

func (r *EventRegistry) GetEventDefinition(et constants.EventType) *core.EventDefinition {
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
	// Good Events (Evaluation > 65)

	// Herb: HP+1
	GlobalEventRegistry.RegisterEvent(&core.EventDefinition{
		Type:        constants.EventTypeHerb,
		Eval:        constants.EvaluationMildGood,
		EnglishName: "Herb",
		Name:        "采集到草药",
		Desc:        "在路边发现了草药，恢复了体力",
	}, &EventHandlerConfig{
		Priority: 60,
		Handler:  createEventModifyHPHandler(1),
	})

	// MilkTea: LP+1
	GlobalEventRegistry.RegisterEvent(&core.EventDefinition{
		Type:        constants.EventTypeMilkTea,
		Eval:        constants.EvaluationGood,
		EnglishName: "MilkTea",
		Name:        "捡到奶茶",
		Desc:        "捡到了一杯奶茶，一口就吃到了猪猪欸",
	}, &EventHandlerConfig{
		Priority: 70,
		Handler:  createEventModifyLPHandler(1),
	})

	// Relic: Draw item
	GlobalEventRegistry.RegisterEvent(&core.EventDefinition{
		Type:        constants.EventTypeRelic,
		Eval:        constants.EvaluationVeryGood,
		EnglishName: "Relic",
		Name:        "捡到勇士的圣遗物",
		Desc:        "发现了古老圣遗物，获得一次道具抽奖机会",
	}, &EventHandlerConfig{
		Priority: 80,
		Handler:  handleDrawItem,
	})

	// DivineBless: Give Divine buff
	GlobalEventRegistry.RegisterEvent(&core.EventDefinition{
		Type:        constants.EventTypeDivineBless,
		Eval:        constants.EvaluationExcellent,
		EnglishName: "DivineBless",
		Name:        "受到天使眷顾",
		Desc:        "天使的祝福降临，获得神眷Buff",
	}, &EventHandlerConfig{
		Priority: 80,
		Handler:  createEventGiveBuffHandler(constants.BuffTypeDivine),
	})

	// Neutral Events (Evaluation 41-65)

	// Exchange: Swap position with another player
	GlobalEventRegistry.RegisterEvent(&core.EventDefinition{
		Type:        constants.EventTypeExchange,
		Eval:        constants.EvaluationNeutral,
		EnglishName: "Exchange",
		Name:        "交换",
		Desc:        "命运之手将你与另一位玩家交换位置",
	}, &EventHandlerConfig{
		Priority: 50,
		Handler:  handleSwapPosition,
	})

	// HiddenBuff: Give Hidden buff
	GlobalEventRegistry.RegisterEvent(&core.EventDefinition{
		Type:        constants.EventTypeHiddenBuff,
		Eval:        constants.EvaluationGood,
		EnglishName: "HiddenBuff",
		Name:        "麻了",
		Desc:        "身体麻木，获得隐匿Buff",
	}, &EventHandlerConfig{
		Priority: 60,
		Handler:  createEventGiveBuffHandler(constants.BuffTypeHidden),
	})

	// TasteTest: Random buff effect
	GlobalEventRegistry.RegisterEvent(&core.EventDefinition{
		Type:        constants.EventTypeTasteTest,
		Eval:        constants.EvaluationMixed,
		EnglishName: "TasteTest",
		Name:        "这是什么？尝一口",
		Desc:        "发现神秘物质，尝试后获得随机效果",
	}, &EventHandlerConfig{
		Priority: 50,
		Handler:  handleRandomBuff,
	})

	// Bad Events (Evaluation <= 40)

	// Mosquito: HP-1
	GlobalEventRegistry.RegisterEvent(&core.EventDefinition{
		Type:        constants.EventTypeMosquito,
		Eval:        constants.EvaluationMildBad,
		EnglishName: "Mosquito",
		Name:        "被蚊虫叮咬",
		Desc:        "丛林中的蚊虫叮咬了你",
	}, &EventHandlerConfig{
		Priority: 40,
		Handler:  createEventModifyHPHandler(-1),
	})

	// GhostHit: HP-1
	GlobalEventRegistry.RegisterEvent(&core.EventDefinition{
		Type:        constants.EventTypeGhostHit,
		Eval:        constants.EvaluationMildBad,
		EnglishName: "GhostHit",
		Name:        "偶遇孤魂野鬼",
		Desc:        "被野鬼打了一闷棍",
	}, &EventHandlerConfig{
		Priority: 40,
		Handler:  createEventModifyHPHandler(-1),
	})

	// DogPoop: LP-1
	GlobalEventRegistry.RegisterEvent(&core.EventDefinition{
		Type:        constants.EventTypeDogPoop,
		Eval:        constants.EvaluationMildBad,
		EnglishName: "DogPoop",
		Name:        "踩到了狗屎",
		Desc:        "运气糟糕的一天",
	}, &EventHandlerConfig{
		Priority: 40,
		Handler:  createEventModifyLPHandler(-1),
	})

	// Thief: Lose random item
	GlobalEventRegistry.RegisterEvent(&core.EventDefinition{
		Type:        constants.EventTypeThief,
		Eval:        constants.EvaluationBad,
		EnglishName: "Thief",
		Name:        "啊？！贼",
		Desc:        "遭遇盗贼，随机丢失一个道具",
	}, &EventHandlerConfig{
		Priority: 30,
		Handler:  handleLoseItem,
	})

	// CurseBuddha: Give Curse buff
	GlobalEventRegistry.RegisterEvent(&core.EventDefinition{
		Type:        constants.EventTypeCurseBuddha,
		Eval:        constants.EvaluationBad,
		EnglishName: "CurseBuddha",
		Name:        "虔诚拜三拜",
		Desc:        "拜路边的野佛，获得诅咒Buff",
	}, &EventHandlerConfig{
		Priority: 30,
		Handler:  createEventGiveBuffHandler(constants.BuffTypeCurse),
	})

	// LostWay: Give Lost buff
	GlobalEventRegistry.RegisterEvent(&core.EventDefinition{
		Type:        constants.EventTypeLostWay,
		Eval:        constants.EvaluationMildBad,
		EnglishName: "LostWay",
		Name:        "迷途",
		Desc:        "迷失方向，获得迷途Buff",
	}, &EventHandlerConfig{
		Priority: 40,
		Handler:  createEventGiveBuffHandler(constants.BuffTypeLost),
	})

	// Thunder: HP to 0 (death)
	GlobalEventRegistry.RegisterEvent(&core.EventDefinition{
		Type:        constants.EventTypeThunder,
		Eval:        constants.EvaluationVeryBad,
		EnglishName: "Thunder",
		Name:        "雷劫",
		Desc:        "天雷降临，HP归零",
	}, &EventHandlerConfig{
		Priority: 20,
		Handler:  handleThunderDeath,
	})
}

// ========== Event Handler Helpers ==========

// createEventModifyHPHandler creates a handler that modifies HP through Action system.
func createEventModifyHPHandler(amount int) EffectHandler {
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

		source := "Event_Effect"

		if amount > 0 {
			ctx.AddDerivedAction(engineaction.NewHealAction(ctx.Player, amount, source))
		} else {
			ctx.AddDerivedAction(engineaction.NewDamageAction(ctx.Player, -amount, source))
		}
		return nil
	}
}

// createEventModifyLPHandler creates a handler that modifies LP through Action system.
func createEventModifyLPHandler(amount int) EffectHandler {
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

		source := "Event_Effect"
		ctx.AddDerivedAction(engineaction.NewModifyLPAction(ctx.Player, amount, source))
		return nil
	}
}

// createEventGiveBuffHandler creates a handler that gives a Buff through Action system.
func createEventGiveBuffHandler(buffType constants.BuffType) EffectHandler {
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

		ctx.AddDerivedAction(engineaction.NewAddBuffAction(ctx.Player, buffType, "Event_Effect"))
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
	ctx.AddDerivedAction(engineaction.NewDrawItemAction(ctx.Player, "Event_Relic"))
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
	ctx.AddDerivedAction(engineaction.NewTeleportAction(ctx.Player, targetPos, "Event_Exchange"))
	ctx.AddDerivedAction(engineaction.NewTeleportAction(targetPlayer, playerPos, "Event_Exchange"))
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
	ctx.AddDerivedAction(engineaction.NewDrawBuffAction(ctx.Player, "Event_TasteTest"))
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
	ctx.AddDerivedAction(engineaction.NewRemoveItemAction(ctx.Player, lostItem.Type, "Event_Thief"))
	return nil
}

func handleThunderDeath(phase constants.Phase, ctx *event.Context) error {
	if ctx == nil || ctx.Player == nil {
		return nil
	}

	source := "Event_Thunder"

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