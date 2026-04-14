package item

import (
	"github.com/b1tAction/paradiced/pkg/constants"
	"github.com/b1tAction/paradiced/pkg/event"
)

// GlobalItemRegistry is the global Item definition registry.
// Initialized at package load time with all Item definitions.
var GlobalItemRegistry *ItemRegistry

// init initializes the global Item registry and registers all Item definitions.
func init() {
	GlobalItemRegistry = NewItemRegistry()
	registerAllItems()
}

// registerAllItems registers all Item definitions with their handler configs.
func registerAllItems() {
	// ReverseClock: Give Lost buff to target player
	GlobalItemRegistry.RegisterItem(&ItemDefinition{
		Type:        constants.ItemTypeReverseClock,
		Eval:        constants.EvaluationGood,
		EnglishName: "ReverseClock",
		Name:        "反方向的钟",
		Desc:        "给予指定玩家迷途Buff",
		TargetSelf:  false,
		TargetOther: true,
		Range:       0, // Any distance
	}, &ItemHandlerConfig{
		Phase:       constants.PhaseAnyTime,
		Priority:    50,
		NeedConfirm: true,
		Handler:     createGiveBuffHandler(constants.BuffTypeLost, 1),
	})

	// AnyDoor: Teleport to target player within 30 range
	GlobalItemRegistry.RegisterItem(&ItemDefinition{
		Type:        constants.ItemTypeAnyDoor,
		Eval:        constants.EvaluationNeutral,
		EnglishName: "AnyDoor",
		Name:        "任意门",
		Desc:        "去到30格内指定玩家身边",
		TargetSelf:  false,
		TargetOther: true,
		Range:       30,
	}, &ItemHandlerConfig{
		Phase:       constants.PhaseOnLand,
		Priority:    60,
		NeedConfirm: true,
		Handler:     handleTeleport,
	})

	// DiceSwap: Swap dice level with target player
	GlobalItemRegistry.RegisterItem(&ItemDefinition{
		Type:        constants.ItemTypeDiceSwap,
		Eval:        constants.EvaluationNeutral,
		EnglishName: "DiceSwap",
		Name:        "骰子交换",
		Desc:        "与指定玩家交换骰子等级",
		TargetSelf:  false,
		TargetOther: true,
		Range:       0, // Any distance
	}, &ItemHandlerConfig{
		Phase:       constants.PhaseAnyTime,
		Priority:    40,
		NeedConfirm: true,
		Handler:     handleDiceSwap,
	})

	// DiceUpgrade: Upgrade current dice level
	GlobalItemRegistry.RegisterItem(&ItemDefinition{
		Type:        constants.ItemTypeDiceUpgrade,
		Eval:        constants.EvaluationGood,
		EnglishName: "DiceUpgrade",
		Name:        "骰子升级卡",
		Desc:        "将当前骰子升级为更高等级",
		TargetSelf:  true,
		TargetOther: false,
		Range:       0,
	}, &ItemHandlerConfig{
		Phase:       constants.PhaseBeforeTurn,
		Priority:    70,
		NeedConfirm: true,
		Handler:     handleDiceUpgrade,
	})
}

// ========== Handler Helper Functions ==========

// createGiveBuffHandler creates a handler that gives a buff to target.
func createGiveBuffHandler(buffType constants.BuffType, duration int) EffectHandler {
	return func(phase constants.Phase, ctx *event.Context) {
		// Signal buff addition intent for target
		ctx.SetString("give_buff_type", string(buffType))
		ctx.SetInt("give_buff_duration", duration)
	}
}

// ========== Custom Item Handlers ==========

// handleTeleport teleports player to target location.
func handleTeleport(phase constants.Phase, ctx *event.Context) {
	// Get target info from context
	targetID, _ := ctx.GetString("target_id")
	if targetID == "" {
		return
	}
	// Signal teleport intent (actual movement in engine)
	ctx.SetString("teleport_target", targetID)
}

// handleDiceSwap swaps dice levels between players.
func handleDiceSwap(phase constants.Phase, ctx *event.Context) {
	targetID, _ := ctx.GetString("target_id")
	if targetID == "" {
		return
	}
	// Signal dice swap intent
	ctx.SetString("dice_swap_target", targetID)
}

// handleDiceUpgrade upgrades the player's dice level.
func handleDiceUpgrade(phase constants.Phase, ctx *event.Context) {
	// Get current dice type and upgrade
	currentDice, _ := ctx.GetString("current_dice_type")
	ctx.SetString("dice_upgrade_from", currentDice)
	// Signal upgrade intent (dice management in engine)
}