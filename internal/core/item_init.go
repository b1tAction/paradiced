package core

import (
	"github.com/b1tAction/Fated/pkg/event"
)

// registerAllItems registers all Item definitions.
// Called from init() in buff_init.go.
func registerAllItems() {
	GlobalRegistry.RegisterItem(&ItemDefinition{
		Type:          ItemTypeReverseClock,
		Eval:          EvaluationGood,
		EnglishName:   "ReverseClock",
		Name:          "反方向的钟",
		Desc:          "给予指定玩家迷途Buff",
		TargetSelf:    false,
		TargetOther:   true,
		BuffType:      BuffTypeLost,
		SpecialEffect: SpecialGiveLost,
		Phase:         event.PhaseAnyTime,
		Priority:      50,
		NeedConfirm:   true,
	})

	GlobalRegistry.RegisterItem(&ItemDefinition{
		Type:          ItemTypeAnyDoor,
		Eval:          EvaluationNeutral,
		EnglishName:   "AnyDoor",
		Name:          "任意门",
		Desc:          "去到30格内指定玩家身边",
		TargetSelf:    false,
		TargetOther:   true,
		Range:         30,
		SpecialEffect: SpecialTeleport,
		Phase:         event.PhaseOnLand,
		Priority:      60,
		NeedConfirm:   true,
	})

	GlobalRegistry.RegisterItem(&ItemDefinition{
		Type:          ItemTypeDiceSwap,
		Eval:          EvaluationNeutral,
		EnglishName:   "DiceSwap",
		Name:          "骰子交换",
		Desc:          "与指定玩家交换骰子等级",
		TargetSelf:    false,
		TargetOther:   true,
		SpecialEffect: SpecialDiceSwap,
		Phase:         event.PhaseAnyTime,
		Priority:      40,
		NeedConfirm:   true,
	})

	GlobalRegistry.RegisterItem(&ItemDefinition{
		Type:          ItemTypeDiceUpgrade,
		Eval:          EvaluationGood,
		EnglishName:   "DiceUpgrade",
		Name:          "骰子升级卡",
		Desc:          "将当前骰子升级为更高等级",
		TargetSelf:    true,
		TargetOther:   false,
		SpecialEffect: SpecialDiceUpgrade,
		Phase:         event.PhaseBeforeTurn,
		Priority:      70,
		NeedConfirm:   true,
	})
}