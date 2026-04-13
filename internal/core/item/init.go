package item

import (
	"github.com/b1tAction/Fated/internal/core/buff"
	"github.com/b1tAction/Fated/internal/core/types"
	"github.com/b1tAction/Fated/pkg/event"
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
		Type:          ItemTypeReverseClock,
		Eval:          types.EvaluationGood,
		EnglishName:   "ReverseClock",
		Name:          "反方向的钟",
		Desc:          "给予指定玩家迷途Buff",
		TargetSelf:    false,
		TargetOther:   true,
		BuffType:      buff.BuffTypeLost,
		SpecialEffect: types.SpecialGiveLost,
		Phase:         event.PhaseAnyTime,
		Priority:      50,
		NeedConfirm:   true,
	})

	GlobalItemRegistry.RegisterItem(&ItemDefinition{
		Type:          ItemTypeAnyDoor,
		Eval:          types.EvaluationNeutral,
		EnglishName:   "AnyDoor",
		Name:          "任意门",
		Desc:          "去到30格内指定玩家身边",
		TargetSelf:    false,
		TargetOther:   true,
		Range:         30,
		SpecialEffect: types.SpecialTeleport,
		Phase:         event.PhaseOnLand,
		Priority:      60,
		NeedConfirm:   true,
	})

	GlobalItemRegistry.RegisterItem(&ItemDefinition{
		Type:          ItemTypeDiceSwap,
		Eval:          types.EvaluationNeutral,
		EnglishName:   "DiceSwap",
		Name:          "骰子交换",
		Desc:          "与指定玩家交换骰子等级",
		TargetSelf:    false,
		TargetOther:   true,
		SpecialEffect: types.SpecialDiceSwap,
		Phase:         event.PhaseAnyTime,
		Priority:      40,
		NeedConfirm:   true,
	})

	GlobalItemRegistry.RegisterItem(&ItemDefinition{
		Type:          ItemTypeDiceUpgrade,
		Eval:          types.EvaluationGood,
		EnglishName:   "DiceUpgrade",
		Name:          "骰子升级卡",
		Desc:          "将当前骰子升级为更高等级",
		TargetSelf:    true,
		TargetOther:   false,
		SpecialEffect: types.SpecialDiceUpgrade,
		Phase:         event.PhaseBeforeTurn,
		Priority:      70,
		NeedConfirm:   true,
	})
}