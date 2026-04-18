package engine

import (
	"testing"

	"github.com/b1tAction/paradiced/internal/core"
	"github.com/b1tAction/paradiced/internal/event"
	"github.com/b1tAction/paradiced/pkg/constants"
	"github.com/b1tAction/paradiced/pkg/id"
)

// ========== ItemRegistry Tests ==========

func TestAllItemsHaveDefinition(t *testing.T) {
	// All Items should have Definition registered
	allItems := GetAllItemTypes()
	for _, it := range allItems {
		def := GetItemDefinition(it)
		if def == nil {
			t.Errorf("ItemType(%s) should have Definition", it)
		}
	}
}

func TestAllItemsHaveHandlerConfig(t *testing.T) {
	// All Items should have HandlerConfig registered
	allItems := GetAllItemTypes()
	for _, it := range allItems {
		config := GetItemHandlerConfig(it)
		if config == nil {
			t.Errorf("ItemType(%s) should have HandlerConfig", it)
		}
	}
}

func TestItemHandlerConfigFields(t *testing.T) {
	tests := []struct {
		itemType    constants.ItemType
		phase       constants.Phase
		priority    int
		needConfirm bool
	}{
		{constants.ItemTypeReverseClock, constants.PhaseAnyTime, 50, true},
		{constants.ItemTypeAnyDoor, constants.PhaseOnLand, 60, true},
		{constants.ItemTypeDiceSwap, constants.PhaseAnyTime, 40, true},
		{constants.ItemTypeDiceUpgrade, constants.PhaseBeforeTurn, 70, true},
	}

	for _, tt := range tests {
		config := GetItemHandlerConfig(tt.itemType)
		if config == nil {
			t.Errorf("ItemType(%s) has no HandlerConfig", tt.itemType)
			continue
		}

		if config.Phase != tt.phase {
			t.Errorf("%s.Phase = %s, expected %s", tt.itemType, config.Phase, tt.phase)
		}

		if config.Priority != tt.priority {
			t.Errorf("%s.Priority = %d, expected %d", tt.itemType, config.Priority, tt.priority)
		}

		if config.NeedConfirm != tt.needConfirm {
			t.Errorf("%s.NeedConfirm = %v, expected %v", tt.itemType, config.NeedConfirm, tt.needConfirm)
		}
	}
}

func TestItemDefinitionsFields(t *testing.T) {
	tests := []struct {
		itemType    constants.ItemType
		eval        constants.Evaluation
		englishName string
		name        string
		targetSelf  bool
		targetOther bool
		rangeVal    int
	}{
		{constants.ItemTypeReverseClock, constants.EvaluationGood, "ReverseClock", "反方向的钟", false, true, 0},
		{constants.ItemTypeAnyDoor, constants.EvaluationNeutral, "AnyDoor", "任意门", false, true, 30},
		{constants.ItemTypeDiceSwap, constants.EvaluationNeutral, "DiceSwap", "骰子交换", false, true, 0},
		{constants.ItemTypeDiceUpgrade, constants.EvaluationGood, "DiceUpgrade", "骰子升级卡", true, false, 0},
	}

	for _, tt := range tests {
		def := GetItemDefinition(tt.itemType)
		if def == nil {
			t.Errorf("ItemType(%s) has no Definition", tt.itemType)
			continue
		}

		if def.Eval != tt.eval {
			t.Errorf("%s.Eval = %d, expected %d", tt.itemType, def.Eval, tt.eval)
		}

		if def.EnglishName != tt.englishName {
			t.Errorf("%s.EnglishName = %s, expected %s", tt.itemType, def.EnglishName, tt.englishName)
		}

		if def.Name != tt.name {
			t.Errorf("%s.Name = %s, expected %s", tt.itemType, def.Name, tt.name)
		}

		if def.TargetSelf != tt.targetSelf {
			t.Errorf("%s.TargetSelf = %v, expected %v", tt.itemType, def.TargetSelf, tt.targetSelf)
		}

		if def.TargetOther != tt.targetOther {
			t.Errorf("%s.TargetOther = %v, expected %v", tt.itemType, def.TargetOther, tt.targetOther)
		}

		if def.Range != tt.rangeVal {
			t.Errorf("%s.Range = %d, expected %d", tt.itemType, def.Range, tt.rangeVal)
		}
	}
}

// ========== Item Handler Tests ==========

func TestReverseClockHandlerBehavior(t *testing.T) {
	// Test ReverseClock gives Lost buff
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})

	config := GetItemHandlerConfig(constants.ItemTypeReverseClock)
	if config == nil {
		t.Fatal("ReverseClock should have HandlerConfig")
	}
	handler := config.Handler
	if handler == nil {
		t.Fatal("ReverseClock should have Handler")
	}

	ctx := event.NewContext(player)
	handler(constants.PhaseAnyTime, ctx)

	// Should signal give_buff_type=Lost
	buffType, err := ctx.GetString("give_buff_type")
	if err != nil {
		t.Error("give_buff_type should be set")
	}
	if buffType != string(constants.BuffTypeLost) {
		t.Errorf("give_buff_type = %s, expected %s", buffType, constants.BuffTypeLost)
	}

	duration, err := ctx.GetInt("give_buff_duration")
	if err != nil {
		t.Error("give_buff_duration should be set")
	}
	if duration != 1 {
		t.Errorf("give_buff_duration = %d, expected 1", duration)
	}
}

func TestAnyDoorHandlerBehavior(t *testing.T) {
	// Test AnyDoor teleport to target
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})

	config := GetItemHandlerConfig(constants.ItemTypeAnyDoor)
	if config == nil {
		t.Fatal("AnyDoor should have HandlerConfig")
	}
	handler := config.Handler
	if handler == nil {
		t.Fatal("AnyDoor should have Handler")
	}

	ctx := event.NewContext(player)
	ctx.SetString("target_id", "target-player-123")
	handler(constants.PhaseOnLand, ctx)

	// Should signal teleport_target
	target, err := ctx.GetString("teleport_target")
	if err != nil {
		t.Error("teleport_target should be set")
	}
	if target != "target-player-123" {
		t.Errorf("teleport_target = %s, expected target-player-123", target)
	}
}

func TestDiceSwapHandlerBehavior(t *testing.T) {
	// Test DiceSwap signals dice swap target
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})

	config := GetItemHandlerConfig(constants.ItemTypeDiceSwap)
	if config == nil {
		t.Fatal("DiceSwap should have HandlerConfig")
	}
	handler := config.Handler
	if handler == nil {
		t.Fatal("DiceSwap should have Handler")
	}

	ctx := event.NewContext(player)
	ctx.SetString("target_id", "swap-target-456")
	handler(constants.PhaseAnyTime, ctx)

	// Should signal dice_swap_target
	target, err := ctx.GetString("dice_swap_target")
	if err != nil {
		t.Error("dice_swap_target should be set")
	}
	if target != "swap-target-456" {
		t.Errorf("dice_swap_target = %s, expected swap-target-456", target)
	}
}

func TestDiceUpgradeHandlerBehavior(t *testing.T) {
	// Test DiceUpgrade signals upgrade from current dice
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})

	config := GetItemHandlerConfig(constants.ItemTypeDiceUpgrade)
	if config == nil {
		t.Fatal("DiceUpgrade should have HandlerConfig")
	}
	handler := config.Handler
	if handler == nil {
		t.Fatal("DiceUpgrade should have Handler")
	}

	ctx := event.NewContext(player)
	ctx.SetString("current_dice_type", "silver")
	handler(constants.PhaseBeforeTurn, ctx)

	// Should signal dice_upgrade_from
	from, err := ctx.GetString("dice_upgrade_from")
	if err != nil {
		t.Error("dice_upgrade_from should be set")
	}
	if from != "silver" {
		t.Errorf("dice_upgrade_from = %s, expected silver", from)
	}
}

func TestGetItemName(t *testing.T) {
	tests := []struct {
		itemType     constants.ItemType
		expectedName string
	}{
		{constants.ItemTypeReverseClock, "反方向的钟"},
		{constants.ItemTypeAnyDoor, "任意门"},
		{constants.ItemTypeDiceSwap, "骰子交换"},
		{constants.ItemTypeDiceUpgrade, "骰子升级卡"},
	}

	for _, tt := range tests {
		name := GetItemName(tt.itemType)
		if name != tt.expectedName {
			t.Errorf("GetItemName(%s) = %s, expected %s", tt.itemType, name, tt.expectedName)
		}
	}
}

func TestGetItemTypesByCategory(t *testing.T) {
	goodItems := GetItemTypesByCategory("Good")
	if len(goodItems) == 0 {
		t.Error("Good items should not be empty")
	}

	neutralItems := GetItemTypesByCategory("Neutral")
	if len(neutralItems) == 0 {
		t.Error("Neutral items should not be empty")
	}

	// Bad items category should return empty (Items have no bad category)
	_ = GetItemTypesByCategory("Bad")

	// Unknown category returns all
	allItems := GetItemTypesByCategory("Unknown")
	if len(allItems) != len(GetAllItemTypes()) {
		t.Error("Unknown category should return all items")
	}
}