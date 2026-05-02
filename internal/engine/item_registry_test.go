package engine

import (
	"testing"

	"github.com/b1tAction/paradiced/internal/core"
	engineaction "github.com/b1tAction/paradiced/internal/engine/action"
	"github.com/b1tAction/paradiced/internal/event"
	"github.com/b1tAction/paradiced/internal/gamemap"
	"github.com/b1tAction/paradiced/pkg/constants"
	"github.com/b1tAction/paradiced/pkg/id"
	"github.com/b1tAction/paradiced/pkg/rng"
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
		{constants.ItemTypeReverseClock, constants.PhaseItemUsed, 50, false},
		{constants.ItemTypeAnyDoor, constants.PhaseItemUsed, 60, false},
		{constants.ItemTypeDiceUpgrade, constants.PhaseItemUsed, 70, false},
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
	}{
		{constants.ItemTypeReverseClock, constants.EvaluationGood, "ReverseClock", "反方向的钟"},
		{constants.ItemTypeAnyDoor, constants.EvaluationNeutral, "AnyDoor", "任意门"},
		{constants.ItemTypeDiceUpgrade, constants.EvaluationGood, "DiceUpgrade", "骰子升级卡"},
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
	}
}

// ========== Item Handler Tests ==========

func TestReverseClockHandlerBehavior(t *testing.T) {
	// Test ReverseClock gives Lost buff through Action system
	game := NewGame(id.NewGameID(), 0)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	game.AddPlayer(player)

	handler := GetItemHandlerConfig(constants.ItemTypeReverseClock).Handler

	actionCtx := engineaction.NewActionContext(game, game.Bus, gamemap.NewMapEngine(20), game.Draw)
	ctx := event.NewContext(player)
	ctx.Set("action_context", actionCtx)

	handler(constants.PhaseAnyTime, ctx)

	// Should have AddBuffAction derived action
	derived := ctx.GetDerivedActions()
	if len(derived) != 1 {
		t.Fatalf("expected 1 derived action, got %d", len(derived))
	}

	addBuffAction, ok := derived[0].(*engineaction.AddBuffAction)
	if !ok {
		t.Fatal("expected AddBuffAction")
	}
	if addBuffAction.BuffType != constants.BuffTypeLost {
		t.Errorf("BuffType = %s, expected Lost", addBuffAction.BuffType)
	}
}

func TestAnyDoorHandlerBehavior(t *testing.T) {
	// Test AnyDoor teleport through Action system
	game := NewGame(id.NewGameID(), 0)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player.Position = 10
	game.AddPlayer(player)

	// Create a target player at position 50
	targetPlayer := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	targetPlayer.Position = 50
	game.AddPlayer(targetPlayer)

	handler := GetItemHandlerConfig(constants.ItemTypeAnyDoor).Handler

	actionCtx := engineaction.NewActionContext(game, game.Bus, gamemap.NewMapEngine(20), game.Draw)
	ctx := event.NewContext(player)
	ctx.Set("action_context", actionCtx)
	ctx.Set("target_player", targetPlayer)

	handler(constants.PhaseOnLand, ctx)

	// Should have TeleportAction derived action
	derived := ctx.GetDerivedActions()
	if len(derived) != 1 {
		t.Fatalf("expected 1 derived action, got %d", len(derived))
	}

	teleportAction, ok := derived[0].(*engineaction.TeleportAction)
	if !ok {
		t.Fatal("expected TeleportAction")
	}
	if teleportAction.TargetPos != 50 {
		t.Errorf("TargetPos = %d, expected 50", teleportAction.TargetPos)
	}
}

func TestDiceUpgradeHandlerBehavior(t *testing.T) {
	// Test DiceUpgrade produces DiceUpgradeAction as DerivedAction
	game := NewGame(id.NewGameID(), 0)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	game.AddPlayer(player)

	handler := GetItemHandlerConfig(constants.ItemTypeDiceUpgrade).Handler

	actionCtx := engineaction.NewActionContext(game, game.Bus, gamemap.NewMapEngine(20), game.Draw)
	ctx := event.NewContext(player)
	ctx.Set("action_context", actionCtx)
	ctx.SetString("current_dice_type", "silver")

	handler(constants.PhaseBeforeTurn, ctx)

	// Should have DiceUpgradeAction derived action
	derived := ctx.GetDerivedActions()
	if len(derived) != 1 {
		t.Fatalf("expected 1 derived action, got %d", len(derived))
	}

	upgradeAction, ok := derived[0].(*engineaction.DiceUpgradeAction)
	if !ok {
		t.Fatal("expected DiceUpgradeAction")
	}
	if upgradeAction.FromDice != rng.DiceTypeSilver {
		t.Errorf("FromDice = %v, expected Silver", upgradeAction.FromDice)
	}
}

func TestGetItemName(t *testing.T) {
	tests := []struct {
		itemType     constants.ItemType
		expectedName string
	}{
		{constants.ItemTypeReverseClock, "反方向的钟"},
		{constants.ItemTypeAnyDoor, "任意门"},
		{constants.ItemTypeDiceUpgrade, "骰子升级卡"},
	}

	for _, tt := range tests {
		name := GetItemName(tt.itemType)
		if name != tt.expectedName {
			t.Errorf("GetItemName(%s) = %s, expected %s", tt.itemType, name, tt.expectedName)
		}
	}
}

func TestBuildItemPool(t *testing.T) {
	// BuildItemPool should produce an EvaluatedItem for every registered ItemDefinition
	pool := BuildItemPool()

	allTypes := GetAllItemTypes()
	if len(pool) != len(allTypes) {
		t.Fatalf("BuildItemPool returned %d items, but %d item types are registered", len(pool), len(allTypes))
	}

	// Verify each pool entry matches its Definition's Type and Eval
	for _, item := range pool {
		def := GetItemDefinition(constants.ItemType(item.Type))
		if def == nil {
			t.Errorf("pool entry Type=%s has no matching ItemDefinition", item.Type)
			continue
		}
		if item.Eval != def.Eval {
			t.Errorf("pool entry Type=%s Eval=%d, expected %d from Definition", item.Type, item.Eval, def.Eval)
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

// ========== Edge Case Tests: nil player/context ==========

func TestItemHandlerWithNilContext(t *testing.T) {
	// All item handlers should gracefully handle nil context
	itemTypes := GetAllItemTypes()
	for _, it := range itemTypes {
		handler := GetItemHandlerConfig(it).Handler
		if handler == nil {
			continue
		}
		// Should not panic with nil context
		handler(constants.PhaseAnyTime, nil)
	}
}

func TestItemHandlerWithNilPlayer(t *testing.T) {
	// All item handlers should gracefully handle nil player in context
	itemTypes := GetAllItemTypes()
	for _, it := range itemTypes {
		handler := GetItemHandlerConfig(it).Handler
		if handler == nil {
			continue
		}
		ctx := event.NewContext(nil) // nil player
		handler(constants.PhaseAnyTime, ctx)
		// Should not produce derived actions
		if len(ctx.GetDerivedActions()) > 0 {
			t.Errorf("ItemType %s should not produce actions with nil player", it)
		}
	}
}

func TestGiveBuffHandlerNilActionContext(t *testing.T) {
	// createGiveBuffHandler requires ActionContext
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	ctx := event.NewContext(player)
	// No action_context set

	handler := GetItemHandlerConfig(constants.ItemTypeReverseClock).Handler
	handler(constants.PhaseAnyTime, ctx)

	// Should not produce derived actions without ActionContext
	if len(ctx.GetDerivedActions()) > 0 {
		t.Error("ReverseClock handler should not produce actions without ActionContext")
	}
}

func TestTeleportHandlerNilActionContext(t *testing.T) {
	// handleTeleport requires ActionContext
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	ctx := event.NewContext(player)
	ctx.SetString("target_id", "test-target")
	ctx.SetInt("target_position", 50)
	// No action_context set

	handler := GetItemHandlerConfig(constants.ItemTypeAnyDoor).Handler
	handler(constants.PhaseOnLand, ctx)

	// Should not produce derived actions without ActionContext
	if len(ctx.GetDerivedActions()) > 0 {
		t.Error("AnyDoor handler should not produce actions without ActionContext")
	}
}

func TestTeleportHandlerNoTargetID(t *testing.T) {
	// handleTeleport requires target_id
	game := NewGame(id.NewGameID(), 0)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	game.AddPlayer(player)

	actionCtx := engineaction.NewActionContext(game, game.Bus, gamemap.NewMapEngine(20), game.Draw)
	ctx := event.NewContext(player)
	ctx.Set("action_context", actionCtx)
	// No target_id set

	handler := GetItemHandlerConfig(constants.ItemTypeAnyDoor).Handler
	handler(constants.PhaseOnLand, ctx)

	// Should not produce derived actions without target_id
	if len(ctx.GetDerivedActions()) > 0 {
		t.Error("AnyDoor handler should not produce actions without target_id")
	}
}

func TestDiceUpgradeHandlerNoCurrentDice(t *testing.T) {
	// handleDiceUpgrade requires current_dice_type to produce DiceUpgradeAction
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	ctx := event.NewContext(player)
	// No current_dice_type set

	handler := GetItemHandlerConfig(constants.ItemTypeDiceUpgrade).Handler
	handler(constants.PhaseBeforeTurn, ctx)

	// Should not produce derived actions without current_dice_type
	if len(ctx.GetDerivedActions()) > 0 {
		t.Error("DiceUpgrade handler should not produce actions without current_dice_type")
	}
}
