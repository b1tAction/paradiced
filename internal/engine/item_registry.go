package engine

import (
	"fmt"

	"github.com/b1tAction/paradiced/internal/core"
	engineaction "github.com/b1tAction/paradiced/internal/engine/action"
	"github.com/b1tAction/paradiced/internal/event"
	"github.com/b1tAction/paradiced/pkg/constants"
	"github.com/b1tAction/paradiced/pkg/resource"
	"github.com/b1tAction/paradiced/pkg/rng"
	"github.com/b1tAction/paradiced/pkg/util"
)

// ItemHandlerConfig contains effect logic and execution configuration.
type ItemHandlerConfig struct {
	Phases      []constants.Phase `json:"phases"`
	Priority    int               `json:"priority"`
	NeedConfirm bool              `json:"need_confirm"`
	Handler     EffectHandler     `json:"-"`
}

// GetPhases returns the Item's trigger phase list.
func (c *ItemHandlerConfig) GetPhases() []constants.Phase {
	return c.Phases
}

// HasPhase checks if the Item triggers at the specified Phase.
func (c *ItemHandlerConfig) HasPhase(phase constants.Phase) bool {
	for _, p := range c.Phases {
		if p == phase {
			return true
		}
	}
	return false
}

// ItemRegistry is the registry for Item definitions and handler configs.
type ItemRegistry struct {
	defs    map[constants.ItemType]*constants.ItemDefinition
	configs map[constants.ItemType]*ItemHandlerConfig
	names   map[constants.ItemType]string

	goodItems    []constants.ItemType
	neutralItems []constants.ItemType
	badItems     []constants.ItemType
}

// GlobalItemRegistry is the global Item registry.
var GlobalItemRegistry *ItemRegistry

func init() {
	GlobalItemRegistry = NewItemRegistry()
	registerAllItems()
}

// NewItemRegistry creates a new Item registry.
func NewItemRegistry() *ItemRegistry {
	return &ItemRegistry{
		defs:         make(map[constants.ItemType]*constants.ItemDefinition),
		configs:      make(map[constants.ItemType]*ItemHandlerConfig),
		names:        make(map[constants.ItemType]string),
		goodItems:    make([]constants.ItemType, 0),
		neutralItems: make([]constants.ItemType, 0),
		badItems:     make([]constants.ItemType, 0),
	}
}

// RegisterItem registers an Item definition with handler config.
func (r *ItemRegistry) RegisterItem(def *constants.ItemDefinition, config *ItemHandlerConfig) {
	if def == nil || def.Type == constants.ItemTypeNone {
		return
	}

	r.defs[def.Type] = def
	r.names[def.Type] = def.Name

	if def.Eval.IsGood() {
		r.goodItems = append(r.goodItems, def.Type)
	} else if def.Eval.IsBad() {
		r.badItems = append(r.badItems, def.Type)
	} else {
		r.neutralItems = append(r.neutralItems, def.Type)
	}

	if config != nil {
		r.configs[def.Type] = config
	}
}

// GetItemDefinition returns the Item definition by type.
func GetItemDefinition(it constants.ItemType) *constants.ItemDefinition {
	return GlobalItemRegistry.GetItemDefinition(it)
}

func (r *ItemRegistry) GetItemDefinition(it constants.ItemType) *constants.ItemDefinition {
	if def, ok := r.defs[it]; ok {
		return def
	}
	return nil
}

// GetItemHandlerConfig returns the Item's handler config.
func GetItemHandlerConfig(it constants.ItemType) *ItemHandlerConfig {
	return GlobalItemRegistry.GetItemHandlerConfig(it)
}

func (r *ItemRegistry) GetItemHandlerConfig(it constants.ItemType) *ItemHandlerConfig {
	if config, ok := r.configs[it]; ok {
		return config
	}
	return nil
}

// GetItemName returns the Item Chinese display name.
func GetItemName(it constants.ItemType) string {
	return GlobalItemRegistry.GetItemName(it)
}

func (r *ItemRegistry) GetItemName(it constants.ItemType) string {
	if name, ok := r.names[it]; ok {
		return name
	}
	return ""
}

// GetAllItemTypes returns all registered Item types.
func GetAllItemTypes() []constants.ItemType {
	return GlobalItemRegistry.GetAllItemTypes()
}

func (r *ItemRegistry) GetAllItemTypes() []constants.ItemType {
	result := make([]constants.ItemType, 0, len(r.defs))
	for it := range r.defs {
		result = append(result, it)
	}
	return result
}

// GetItemTypesByCategory returns Item types by category.
func GetItemTypesByCategory(category string) []constants.ItemType {
	return GlobalItemRegistry.GetItemTypesByCategory(category)
}

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

// BuildItemPool builds an []*rng.EvaluatedItem pool from all registered ItemDefinitions.
// This is the single source of truth for item pool data — no need to manually
// duplicate Type+Eval mappings elsewhere.
func BuildItemPool() []*rng.EvaluatedItem {
	return GlobalItemRegistry.buildItemPool()
}

func (r *ItemRegistry) buildItemPool() []*rng.EvaluatedItem {
	pool := make([]*rng.EvaluatedItem, 0, len(r.defs))
	for _, def := range r.defs {
		pool = append(pool, &rng.EvaluatedItem{
			Type: string(def.Type),
			Eval: def.Eval,
		})
	}
	return pool
}

// ========== Item Handler Registration ==========

func registerAllItems() {
	defs := resource.GlobalDefinitionSet

	// ReverseClock: Give Lost buff to target player
	GlobalItemRegistry.RegisterItem(defs.Items[constants.ItemTypeReverseClock], &ItemHandlerConfig{
		Phases:      []constants.Phase{constants.PhaseItemUsed},
		Priority:    50,
		NeedConfirm: false,
		Handler:     createGiveBuffHandler(constants.BuffTypeLost, constants.SourceItemReverseClockBuff),
	})

	// AnyDoor: Teleport to target player's position
	GlobalItemRegistry.RegisterItem(defs.Items[constants.ItemTypeAnyDoor], &ItemHandlerConfig{
		Phases:      []constants.Phase{constants.PhaseItemUsed},
		Priority:    60,
		NeedConfirm: false,
		Handler:     handleTeleport,
	})

	// DiceUpgrade: Upgrade current dice level
	GlobalItemRegistry.RegisterItem(defs.Items[constants.ItemTypeDiceUpgrade], &ItemHandlerConfig{
		Phases:      []constants.Phase{constants.PhaseItemUsed},
		Priority:    70,
		NeedConfirm: false,
		Handler:     handleDiceUpgrade,
	})

	// MagicFlute: Give Sinking buff to self and target player
	GlobalItemRegistry.RegisterItem(defs.Items[constants.ItemTypeMagicFlute], &ItemHandlerConfig{
		Phases:      []constants.Phase{constants.PhaseItemUsed},
		Priority:    50,
		NeedConfirm: false,
		Handler:     handleMagicFlute,
	})

	// CupidArrow: Give Eternal buff to self and target player
	GlobalItemRegistry.RegisterItem(defs.Items[constants.ItemTypeCupidArrow], &ItemHandlerConfig{
		Phases:      []constants.Phase{constants.PhaseItemUsed},
		Priority:    50,
		NeedConfirm: false,
		Handler:     handleCupidArrow,
	})

	// CrimsonBlade: Sacrifice half HP, deal same amount as damage to target
	GlobalItemRegistry.RegisterItem(defs.Items[constants.ItemTypeCrimsonBlade], &ItemHandlerConfig{
		Phases:      []constants.Phase{constants.PhaseItemUsed},
		Priority:    50,
		NeedConfirm: false,
		Handler:     handleCrimsonBlade,
	})

	// WisdomRing: Give Divine buff to self
	GlobalItemRegistry.RegisterItem(defs.Items[constants.ItemTypeWisdomRing], &ItemHandlerConfig{
		Phases:      []constants.Phase{constants.PhaseItemUsed},
		Priority:    50,
		NeedConfirm: false,
		Handler:     createGiveBuffHandler(constants.BuffTypeDivine, constants.SourceItemWisdomRingBuff),
	})

	// MeditationRing: Give Rain buff to self
	GlobalItemRegistry.RegisterItem(defs.Items[constants.ItemTypeMeditationRing], &ItemHandlerConfig{
		Phases:      []constants.Phase{constants.PhaseItemUsed},
		Priority:    50,
		NeedConfirm: false,
		Handler:     createGiveBuffHandler(constants.BuffTypeRain, constants.SourceItemMeditationRingBuff),
	})

	// DisciplineRing: Give Golden Body buff to self
	GlobalItemRegistry.RegisterItem(defs.Items[constants.ItemTypeDisciplineRing], &ItemHandlerConfig{
		Phases:      []constants.Phase{constants.PhaseItemUsed},
		Priority:    50,
		NeedConfirm: false,
		Handler:     createGiveBuffHandler(constants.BuffTypeGoldenBody, constants.SourceItemDisciplineRingBuff),
	})

	// FoolishRing: HP+1, LP-1
	GlobalItemRegistry.RegisterItem(defs.Items[constants.ItemTypeFoolishRing], &ItemHandlerConfig{
		Phases:      []constants.Phase{constants.PhaseItemUsed},
		Priority:    50,
		NeedConfirm: false,
		Handler:     handleFoolishRing,
	})

	// GreedyRing: LP+1, HP-1
	GlobalItemRegistry.RegisterItem(defs.Items[constants.ItemTypeGreedyRing], &ItemHandlerConfig{
		Phases:      []constants.Phase{constants.PhaseItemUsed},
		Priority:    50,
		NeedConfirm: false,
		Handler:     handleGreedyRing,
	})

	// WrathRing: HP-1, gain Wrath buff
	GlobalItemRegistry.RegisterItem(defs.Items[constants.ItemTypeWrathRing], &ItemHandlerConfig{
		Phases:      []constants.Phase{constants.PhaseItemUsed},
		Priority:    50,
		NeedConfirm: false,
		Handler:     handleWrathRing,
	})

	// NamedBlade: Passive item - auto-grants Savior buff via PhasePostItemAdded handler
	GlobalItemRegistry.RegisterItem(defs.Items[constants.ItemTypeNamedBlade], &ItemHandlerConfig{
		Phases:      []constants.Phase{constants.PhasePostItemAdded},
		Priority:    50,
		NeedConfirm: false,
		Handler:     handleNamedBladePassive,
	})

	// SageProtection: Passive item - auto-grants SageProtection buff via PhasePostItemAdded handler
	GlobalItemRegistry.RegisterItem(defs.Items[constants.ItemTypeSageProtection], &ItemHandlerConfig{
		Phases:      []constants.Phase{constants.PhasePostItemAdded},
		Priority:    50,
		NeedConfirm: false,
		Handler:     handleSageProtectionPassive,
	})
}

// ========== Item Handler Helpers ==========

func createGiveBuffHandler(buffType constants.BuffType, source constants.ActionSource) EffectHandler {
	return func(phase constants.Phase, ctx *event.Context) error {
		if ctx == nil {
			return fmt.Errorf("handler: event context is nil")
		}
		if ctx.Player == nil {
			return fmt.Errorf("handler: player is nil in event context")
		}

		// Source item instance matching
		sourceItemID, _ := ctx.GetString("source_item_id")
		triggerItemID, _ := ctx.GetString("item_id")
		if triggerItemID != "" && sourceItemID != "" && triggerItemID != sourceItemID {
			return nil // Not this item instance
		}

		// Resolve target player: use target_player from context if set (targeted items like ReverseClock),
		// otherwise fall back to the current player (self-targeted items)
		var targetPlayer *core.Player
		if val, ok := ctx.Get("target_player"); ok {
			if p, ok2 := val.(*core.Player); ok2 && p != nil {
				targetPlayer = p
			}
		}
		if targetPlayer == nil {
			targetPlayer = ctx.Player
		}

		// Check ActionContext exists
		actionCtx, err := getActionCtxFromEventCtx(ctx)
		if err != nil {
			return err
		}
		_ = actionCtx // ActionContext used for derived action processing

		ctx.AddDerivedAction(engineaction.NewAddBuffAction(targetPlayer, buffType, string(source)))
		return nil
	}
}

func handleTeleport(phase constants.Phase, ctx *event.Context) error {
	if ctx == nil {
		return fmt.Errorf("handler: event context is nil")
	}
	if ctx.Player == nil {
		return fmt.Errorf("handler: player is nil in event context")
	}


	// Source item instance matching
	sourceItemID, _ := ctx.GetString("source_item_id")
	triggerItemID, _ := ctx.GetString("item_id")
	if triggerItemID != "" && sourceItemID != "" && triggerItemID != sourceItemID {
		return nil // Not this item instance
	}

	// Resolve target player: use target_player from context if set (AnyDoor targets a player)
	var targetPlayer *core.Player
	if val, ok := ctx.Get("target_player"); ok {
		if p, ok2 := val.(*core.Player); ok2 && p != nil {
			targetPlayer = p
		}
	}
	if targetPlayer == nil {
		// No target player specified, cannot teleport
		return nil
	}

	// Teleport to the target player's position
	ctx.AddDerivedAction(engineaction.NewTeleportAction(ctx.Player, targetPlayer.Position, string(constants.SourceItemAnyDoor)))
	return nil
}

func handleDiceUpgrade(phase constants.Phase, ctx *event.Context) error {
	if ctx == nil || ctx.Player == nil {
		return nil
	}


	// Source item instance matching
	sourceItemID, _ := ctx.GetString("source_item_id")
	triggerItemID, _ := ctx.GetString("item_id")
	if triggerItemID != "" && sourceItemID != "" && triggerItemID != sourceItemID {
		return nil // Not this item instance
	}

	actionCtx, err := getActionCtxFromEventCtx(ctx)
	if err != nil {
		return err
	}
	_ = actionCtx // ActionContext used for derived action processing

	currentDice, err := ctx.GetString("current_dice_type")
	if err != nil || currentDice == "" {
		return nil // No current dice type, cannot upgrade
	}

	fromDice := rng.DiceTypeFromString(currentDice)
	ctx.AddDerivedAction(engineaction.NewDiceUpgradeAction(ctx.Player, string(constants.SourceItemDiceUpgrade), fromDice))
	return nil
}

// ========== New Item Handlers ==========

func handleMagicFlute(phase constants.Phase, ctx *event.Context) error {
	if ctx == nil {
		return fmt.Errorf("handler: event context is nil")
	}
	if ctx.Player == nil {
		return fmt.Errorf("handler: player is nil in event context")
	}


	// Source item instance matching
	sourceItemID, _ := ctx.GetString("source_item_id")
	triggerItemID, _ := ctx.GetString("item_id")
	if triggerItemID != "" && sourceItemID != "" && triggerItemID != sourceItemID {
		return nil // Not this item instance
	}

	// Resolve target player (required for MagicFlute)
	var targetPlayer *core.Player
	if val, ok := ctx.Get("target_player"); ok {
		if p, ok2 := val.(*core.Player); ok2 && p != nil {
			targetPlayer = p
		}
	}
	if targetPlayer == nil {
		return fmt.Errorf("handler: 魔笛 requires a target player")
	}

	actionCtx, err := getActionCtxFromEventCtx(ctx)
	if err != nil {
		return err
	}
	_ = actionCtx

	// Give Sinking buff to self, linked to target player
	metaSelf := util.NewMetadata()
	metaSelf.Set("sinking_linked_player", targetPlayer.ID.UUID())
	ctx.AddDerivedAction(engineaction.NewAddBuffActionWithMetadata(ctx.Player, constants.BuffTypeSinking, string(constants.SourceItemMagicFluteBuff), metaSelf))

	// Give Sinking buff to target, linked to self
	metaTarget := util.NewMetadata()
	metaTarget.Set("sinking_linked_player", ctx.Player.ID.UUID())
	ctx.AddDerivedAction(engineaction.NewAddBuffActionWithMetadata(targetPlayer, constants.BuffTypeSinking, string(constants.SourceItemMagicFluteBuff), metaTarget))

	return nil
}

func handleCupidArrow(phase constants.Phase, ctx *event.Context) error {
	if ctx == nil {
		return fmt.Errorf("handler: event context is nil")
	}
	if ctx.Player == nil {
		return fmt.Errorf("handler: player is nil in event context")
	}


	// Source item instance matching
	sourceItemID, _ := ctx.GetString("source_item_id")
	triggerItemID, _ := ctx.GetString("item_id")
	if triggerItemID != "" && sourceItemID != "" && triggerItemID != sourceItemID {
		return nil // Not this item instance
	}

	// Resolve target player (required for CupidArrow)
	var targetPlayer *core.Player
	if val, ok := ctx.Get("target_player"); ok {
		if p, ok2 := val.(*core.Player); ok2 && p != nil {
			targetPlayer = p
		}
	}
	if targetPlayer == nil {
		return fmt.Errorf("handler: 丘比特之箭 requires a target player")
	}

	actionCtx, err := getActionCtxFromEventCtx(ctx)
	if err != nil {
		return err
	}
	_ = actionCtx

	// Give Eternal buff to self, linked to target player
	metaSelf := util.NewMetadata()
	metaSelf.Set("eternal_linked_player", targetPlayer.ID.UUID())
	ctx.AddDerivedAction(engineaction.NewAddBuffActionWithMetadata(ctx.Player, constants.BuffTypeEternal, string(constants.SourceItemCupidArrowBuff), metaSelf))

	// Give Eternal buff to target, linked to self
	metaTarget := util.NewMetadata()
	metaTarget.Set("eternal_linked_player", ctx.Player.ID.UUID())
	ctx.AddDerivedAction(engineaction.NewAddBuffActionWithMetadata(targetPlayer, constants.BuffTypeEternal, string(constants.SourceItemCupidArrowBuff), metaTarget))

	return nil
}

func handleCrimsonBlade(phase constants.Phase, ctx *event.Context) error {
	if ctx == nil {
		return fmt.Errorf("handler: event context is nil")
	}
	if ctx.Player == nil {
		return fmt.Errorf("handler: player is nil in event context")
	}


	// Source item instance matching
	sourceItemID, _ := ctx.GetString("source_item_id")
	triggerItemID, _ := ctx.GetString("item_id")
	if triggerItemID != "" && sourceItemID != "" && triggerItemID != sourceItemID {
		return nil // Not this item instance
	}

	// Resolve target player (required for CrimsonBlade)
	var targetPlayer *core.Player
	if val, ok := ctx.Get("target_player"); ok {
		if p, ok2 := val.(*core.Player); ok2 && p != nil {
			targetPlayer = p
		}
	}
	if targetPlayer == nil {
		return fmt.Errorf("handler: 猩红之刃 requires a target player")
	}

	actionCtx, err := getActionCtxFromEventCtx(ctx)
	if err != nil {
		return err
	}
	_ = actionCtx

	// Sacrifice half HP as piercing damage to self, deal same amount to target
	damageAmount := ctx.Player.HP / 2
	if damageAmount <= 0 {
		return nil // No damage if HP too low
	}

	ctx.AddDerivedAction(engineaction.NewPiercingDamageAction(ctx.Player, damageAmount, string(constants.SourceItemCrimsonBlade)))
	ctx.AddDerivedAction(engineaction.NewDamageActionWithSource(targetPlayer, damageAmount, ctx.Player, string(constants.SourceItemCrimsonBlade)))

	return nil
}

func handleFoolishRing(phase constants.Phase, ctx *event.Context) error {
	if ctx == nil {
		return fmt.Errorf("handler: event context is nil")
	}
	if ctx.Player == nil {
		return fmt.Errorf("handler: player is nil in event context")
	}


	// Source item instance matching
	sourceItemID, _ := ctx.GetString("source_item_id")
	triggerItemID, _ := ctx.GetString("item_id")
	if triggerItemID != "" && sourceItemID != "" && triggerItemID != sourceItemID {
		return nil // Not this item instance
	}

	actionCtx, err := getActionCtxFromEventCtx(ctx)
	if err != nil {
		return err
	}
	_ = actionCtx

	// HP+1, LP-1
	ctx.AddDerivedAction(engineaction.NewHealAction(ctx.Player, 1, string(constants.SourceItemFoolishRing)))
	ctx.AddDerivedAction(engineaction.NewModifyLPAction(ctx.Player, -1, string(constants.SourceItemFoolishRing)))

	return nil
}

func handleGreedyRing(phase constants.Phase, ctx *event.Context) error {
	if ctx == nil {
		return fmt.Errorf("handler: event context is nil")
	}
	if ctx.Player == nil {
		return fmt.Errorf("handler: player is nil in event context")
	}


	// Source item instance matching
	sourceItemID, _ := ctx.GetString("source_item_id")
	triggerItemID, _ := ctx.GetString("item_id")
	if triggerItemID != "" && sourceItemID != "" && triggerItemID != sourceItemID {
		return nil // Not this item instance
	}

	actionCtx, err := getActionCtxFromEventCtx(ctx)
	if err != nil {
		return err
	}
	_ = actionCtx

	// LP+1, HP-1
	ctx.AddDerivedAction(engineaction.NewModifyLPAction(ctx.Player, 1, string(constants.SourceItemGreedyRing)))
	ctx.AddDerivedAction(engineaction.NewDamageAction(ctx.Player, 1, string(constants.SourceItemGreedyRing)))

	return nil
}

func handleWrathRing(phase constants.Phase, ctx *event.Context) error {
	if ctx == nil {
		return fmt.Errorf("handler: event context is nil")
	}
	if ctx.Player == nil {
		return fmt.Errorf("handler: player is nil in event context")
	}


	// Source item instance matching
	sourceItemID, _ := ctx.GetString("source_item_id")
	triggerItemID, _ := ctx.GetString("item_id")
	if triggerItemID != "" && sourceItemID != "" && triggerItemID != sourceItemID {
		return nil // Not this item instance
	}

	actionCtx, err := getActionCtxFromEventCtx(ctx)
	if err != nil {
		return err
	}
	_ = actionCtx

	// HP-1, gain Wrath buff
	ctx.AddDerivedAction(engineaction.NewDamageAction(ctx.Player, 1, string(constants.SourceItemWrathRing)))
	ctx.AddDerivedAction(engineaction.NewAddBuffAction(ctx.Player, constants.BuffTypeWrath, string(constants.SourceItemWrathRingBuff)))

	return nil
}

// ========== Passive Item Handlers ==========

func handleNamedBladePassive(phase constants.Phase, ctx *event.Context) error {
	if phase != constants.PhasePostItemAdded {
		return nil
	}
	if ctx == nil || ctx.Player == nil {
		return nil
	}

	// Source item instance matching
	sourceItemID, _ := ctx.GetString("source_item_id")
	triggerItemID, _ := ctx.GetString("item_id")
	if triggerItemID != "" && sourceItemID != "" && triggerItemID != sourceItemID {
		return nil // Not this item instance
	}

	meta := util.NewMetadata()
	meta.SetString("source_item_id", triggerItemID)
	meta.SetString("source_item_type", string(constants.ItemTypeNamedBlade))
	ctx.AddDerivedAction(engineaction.NewAddBuffActionWithMetadata(
		ctx.Player, constants.BuffTypeSavior,
		string(constants.SourceItemNamedBladePassive), meta,
	))
	return nil
}

func handleSageProtectionPassive(phase constants.Phase, ctx *event.Context) error {
	if phase != constants.PhasePostItemAdded {
		return nil
	}
	if ctx == nil || ctx.Player == nil {
		return nil
	}

	// Source item instance matching
	sourceItemID, _ := ctx.GetString("source_item_id")
	triggerItemID, _ := ctx.GetString("item_id")
	if triggerItemID != "" && sourceItemID != "" && triggerItemID != sourceItemID {
		return nil // Not this item instance
	}

	meta := util.NewMetadata()
	meta.SetString("source_item_id", triggerItemID)
	meta.SetString("source_item_type", string(constants.ItemTypeSageProtection))
	ctx.AddDerivedAction(engineaction.NewAddBuffActionWithMetadata(
		ctx.Player, constants.BuffTypeSageProtection,
		string(constants.SourceItemSageProtectionPassive), meta,
	))
	return nil
}