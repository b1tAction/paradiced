package item

import (
	"github.com/b1tAction/paradiced/pkg/constants"
)

// GlobalItemRegistry is the global Item definition registry.
// Initialized at package load time with all Item definitions.
var GlobalItemRegistry *ItemRegistry

// init initializes the global Item registry and registers all Item definitions.
func init() {
	GlobalItemRegistry = NewItemRegistry()
	registerAllItems()
}

// registerAllItems registers all Item definitions.
func registerAllItems() {
	GlobalItemRegistry.RegisterItem(&ItemDefinition{
		Type:          constants.ItemTypeReverseClock,
		Eval:          constants.EvaluationGood,
		EnglishName:   "ReverseClock",
		Name:          "反方向的钟",
		Desc:          "给予指定玩家迷途Buff",
		TargetSelf:    false,
		TargetOther:   true,
		BuffType:      constants.BuffTypeLost,
		SpecialEffect: constants.SpecialGiveLost,
		Phase:         constants.PhaseAnyTime,
		Priority:      50,
		NeedConfirm:   true,
	})

	GlobalItemRegistry.RegisterItem(&ItemDefinition{
		Type:          constants.ItemTypeAnyDoor,
		Eval:          constants.EvaluationNeutral,
		EnglishName:   "AnyDoor",
		Name:          "任意门",
		Desc:          "去到30格内指定玩家身边",
		TargetSelf:    false,
		TargetOther:   true,
		Range:         30,
		BuffType:      constants.BuffTypeNone,
		SpecialEffect: constants.SpecialTeleport,
		Phase:         constants.PhaseOnLand,
		Priority:      60,
		NeedConfirm:   true,
	})

	GlobalItemRegistry.RegisterItem(&ItemDefinition{
		Type:          constants.ItemTypeDiceSwap,
		Eval:          constants.EvaluationNeutral,
		EnglishName:   "DiceSwap",
		Name:          "骰子交换",
		Desc:          "与指定玩家交换骰子等级",
		TargetSelf:    false,
		TargetOther:   true,
		BuffType:      constants.BuffTypeNone,
		SpecialEffect: constants.SpecialDiceSwap,
		Phase:         constants.PhaseAnyTime,
		Priority:      40,
		NeedConfirm:   true,
	})

	GlobalItemRegistry.RegisterItem(&ItemDefinition{
		Type:          constants.ItemTypeDiceUpgrade,
		Eval:          constants.EvaluationGood,
		EnglishName:   "DiceUpgrade",
		Name:          "骰子升级卡",
		Desc:          "将当前骰子升级为更高等级",
		TargetSelf:    true,
		TargetOther:   false,
		BuffType:      constants.BuffTypeNone,
		SpecialEffect: constants.SpecialDiceUpgrade,
		Phase:         constants.PhaseBeforeTurn,
		Priority:      70,
		NeedConfirm:   true,
	})
}
