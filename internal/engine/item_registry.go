package engine

import (
	"github.com/b1tAction/paradiced/internal/core"
	"github.com/b1tAction/paradiced/internal/event"
	"github.com/b1tAction/paradiced/pkg/constants"
)

// ItemHandlerConfig contains effect logic and execution configuration.
type ItemHandlerConfig struct {
	Phase       constants.Phase `json:"phase"`
	Priority    int             `json:"priority"`
	NeedConfirm bool            `json:"need_confirm"`
	Handler     EffectHandler   `json:"-"`
}

// ItemRegistry is the registry for Item definitions and handler configs.
type ItemRegistry struct {
	defs    map[constants.ItemType]*core.ItemDefinition
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
		defs:         make(map[constants.ItemType]*core.ItemDefinition),
		configs:      make(map[constants.ItemType]*ItemHandlerConfig),
		names:        make(map[constants.ItemType]string),
		goodItems:    make([]constants.ItemType, 0),
		neutralItems: make([]constants.ItemType, 0),
		badItems:     make([]constants.ItemType, 0),
	}
}

// RegisterItem registers an Item definition with handler config.
func (r *ItemRegistry) RegisterItem(def *core.ItemDefinition, config *ItemHandlerConfig) {
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
func GetItemDefinition(it constants.ItemType) *core.ItemDefinition {
	return GlobalItemRegistry.GetItemDefinition(it)
}

func (r *ItemRegistry) GetItemDefinition(it constants.ItemType) *core.ItemDefinition {
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

// ========== Item Handler Registration ==========

func registerAllItems() {
	// ReverseClock: Give Lost buff to target player
	GlobalItemRegistry.RegisterItem(&core.ItemDefinition{
		Type:        constants.ItemTypeReverseClock,
		Eval:        constants.EvaluationGood,
		EnglishName: "ReverseClock",
		Name:        "反方向的钟",
		Desc:        "给予指定玩家迷途Buff",
	}, &ItemHandlerConfig{
		Phase:       constants.PhaseAnyTime,
		Priority:    50,
		NeedConfirm: true,
		Handler:     createGiveBuffHandler(constants.BuffTypeLost, 1),
	})

	// AnyDoor: Teleport to target player within 30 range
	GlobalItemRegistry.RegisterItem(&core.ItemDefinition{
		Type:        constants.ItemTypeAnyDoor,
		Eval:        constants.EvaluationNeutral,
		EnglishName: "AnyDoor",
		Name:        "任意门",
		Desc:        "去到30格内指定玩家身边",
	}, &ItemHandlerConfig{
		Phase:       constants.PhaseOnLand,
		Priority:    60,
		NeedConfirm: true,
		Handler:     handleTeleport,
	})

	// DiceSwap: Swap dice level with target player
	GlobalItemRegistry.RegisterItem(&core.ItemDefinition{
		Type:        constants.ItemTypeDiceSwap,
		Eval:        constants.EvaluationNeutral,
		EnglishName: "DiceSwap",
		Name:        "骰子交换",
		Desc:        "与指定玩家交换骰子等级",
	}, &ItemHandlerConfig{
		Phase:       constants.PhaseAnyTime,
		Priority:    40,
		NeedConfirm: true,
		Handler:     handleDiceSwap,
	})

	// DiceUpgrade: Upgrade current dice level
	GlobalItemRegistry.RegisterItem(&core.ItemDefinition{
		Type:        constants.ItemTypeDiceUpgrade,
		Eval:        constants.EvaluationGood,
		EnglishName: "DiceUpgrade",
		Name:        "骰子升级卡",
		Desc:        "将当前骰子升级为更高等级",
	}, &ItemHandlerConfig{
		Phase:       constants.PhaseBeforeTurn,
		Priority:    70,
		NeedConfirm: true,
		Handler:     handleDiceUpgrade,
	})
}

// ========== Item Handler Helpers ==========

func createGiveBuffHandler(buffType constants.BuffType, duration int) EffectHandler {
	return func(phase constants.Phase, ctx *event.Context) {
		ctx.SetString("give_buff_type", string(buffType))
		ctx.SetInt("give_buff_duration", duration)
	}
}

func handleTeleport(phase constants.Phase, ctx *event.Context) {
	targetID, _ := ctx.GetString("target_id")
	if targetID == "" {
		return
	}
	ctx.SetString("teleport_target", targetID)
}

func handleDiceSwap(phase constants.Phase, ctx *event.Context) {
	targetID, _ := ctx.GetString("target_id")
	if targetID == "" {
		return
	}
	ctx.SetString("dice_swap_target", targetID)
}

func handleDiceUpgrade(phase constants.Phase, ctx *event.Context) {
	currentDice, _ := ctx.GetString("current_dice_type")
	ctx.SetString("dice_upgrade_from", currentDice)
}