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
	// handleTeleport reads target_player, not ActionContext for teleport logic
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	ctx := event.NewContext(player)
	// No target_player set

	handler := GetItemHandlerConfig(constants.ItemTypeAnyDoor).Handler
	handler(constants.PhaseOnLand, ctx)

	// Should not produce derived actions without target_player
	if len(ctx.GetDerivedActions()) > 0 {
		t.Error("AnyDoor handler should not produce actions without target_player")
	}
}

func TestTeleportHandlerNoTargetPlayer(t *testing.T) {
	// handleTeleport requires target_player to produce TeleportAction
	game := NewGame(id.NewGameID(), 0)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player.Position = 10
	game.AddPlayer(player)

	actionCtx := engineaction.NewActionContext(game, game.Bus, gamemap.NewMapEngine(20), game.Draw)
	ctx := event.NewContext(player)
	ctx.Set("action_context", actionCtx)
	// No target_player set

	handler := GetItemHandlerConfig(constants.ItemTypeAnyDoor).Handler
	handler(constants.PhaseOnLand, ctx)

	// Should not produce derived actions without target_player
	if len(ctx.GetDerivedActions()) > 0 {
		t.Error("AnyDoor handler should not produce actions without target_player")
	}
}

func TestTeleportHandlerNilTargetPlayer(t *testing.T) {
	// handleTeleport gracefully handles nil target_player
	game := NewGame(id.NewGameID(), 0)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player.Position = 10
	game.AddPlayer(player)

	actionCtx := engineaction.NewActionContext(game, game.Bus, gamemap.NewMapEngine(20), game.Draw)
	ctx := event.NewContext(player)
	ctx.Set("action_context", actionCtx)
	ctx.Set("target_player", nil) // nil target_player

	handler := GetItemHandlerConfig(constants.ItemTypeAnyDoor).Handler
	handler(constants.PhaseOnLand, ctx)

	// Should not produce derived actions with nil target_player
	if len(ctx.GetDerivedActions()) > 0 {
		t.Error("AnyDoor handler should not produce actions with nil target_player")
	}
}

func TestTeleportHandlerSourceAnyDoor(t *testing.T) {
	// TeleportAction from AnyDoor should have source = item_any_door
	game := NewGame(id.NewGameID(), 0)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player.Position = 10
	game.AddPlayer(player)

	targetPlayer := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	targetPlayer.Position = 50
	game.AddPlayer(targetPlayer)

	actionCtx := engineaction.NewActionContext(game, game.Bus, gamemap.NewMapEngine(20), game.Draw)
	ctx := event.NewContext(player)
	ctx.Set("action_context", actionCtx)
	ctx.Set("target_player", targetPlayer)

	handler := GetItemHandlerConfig(constants.ItemTypeAnyDoor).Handler
	handler(constants.PhaseOnLand, ctx)

	derived := ctx.GetDerivedActions()
	if len(derived) != 1 {
		t.Fatalf("expected 1 derived action, got %d", len(derived))
	}

	teleportAction, ok := derived[0].(*engineaction.TeleportAction)
	if !ok {
		t.Fatal("expected TeleportAction")
	}
	if teleportAction.SourceID != string(constants.SourceItemAnyDoor) {
		t.Errorf("SourceID = %s, expected %s", teleportAction.SourceID, constants.SourceItemAnyDoor)
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

// ========== New Item Handler Tests ==========

func TestMagicFluteHandlerBehavior(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	target := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	game.AddPlayer(player)
	game.AddPlayer(target)

	handler := GetItemHandlerConfig(constants.ItemTypeMagicFlute).Handler
	actionCtx := engineaction.NewActionContext(game, game.Bus, gamemap.NewMapEngine(20), game.Draw)
	ctx := event.NewContext(player)
	ctx.Set("action_context", actionCtx)
	ctx.Set("target_player", target)

	handler(constants.PhaseItemUsed, ctx)

	derived := ctx.GetDerivedActions()
	if len(derived) != 2 {
		t.Fatalf("expected 2 derived actions (Sinking for self and target), got %d", len(derived))
	}

	// First: AddBuffActionWithMetadata for self
	selfBuff, ok := derived[0].(*engineaction.AddBuffAction)
	if !ok {
		t.Fatalf("expected AddBuffAction, got %T", derived[0])
	}
	if selfBuff.BuffType != constants.BuffTypeSinking {
		t.Errorf("self buff type = %s, expected sinking", selfBuff.BuffType)
	}
	if selfBuff.TargetPlayer() != player {
		t.Error("self buff should target self")
	}

	// Second: AddBuffActionWithMetadata for target
	targetBuff, ok := derived[1].(*engineaction.AddBuffAction)
	if !ok {
		t.Fatalf("expected AddBuffAction, got %T", derived[1])
	}
	if targetBuff.BuffType != constants.BuffTypeSinking {
		t.Errorf("target buff type = %s, expected sinking", targetBuff.BuffType)
	}
	if targetBuff.TargetPlayer() != target {
		t.Error("target buff should target target player")
	}
}

func TestCupidArrowHandlerBehavior(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	target := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	game.AddPlayer(player)
	game.AddPlayer(target)

	handler := GetItemHandlerConfig(constants.ItemTypeCupidArrow).Handler
	actionCtx := engineaction.NewActionContext(game, game.Bus, gamemap.NewMapEngine(20), game.Draw)
	ctx := event.NewContext(player)
	ctx.Set("action_context", actionCtx)
	ctx.Set("target_player", target)

	handler(constants.PhaseItemUsed, ctx)

	derived := ctx.GetDerivedActions()
	if len(derived) != 2 {
		t.Fatalf("expected 2 derived actions, got %d", len(derived))
	}

	selfBuff, ok := derived[0].(*engineaction.AddBuffAction)
	if !ok {
		t.Fatalf("expected AddBuffAction, got %T", derived[0])
	}
	if selfBuff.BuffType != constants.BuffTypeEternal {
		t.Errorf("self buff type = %s, expected eternal", selfBuff.BuffType)
	}

	targetBuff, ok := derived[1].(*engineaction.AddBuffAction)
	if !ok {
		t.Fatalf("expected AddBuffAction, got %T", derived[1])
	}
	if targetBuff.BuffType != constants.BuffTypeEternal {
		t.Errorf("target buff type = %s, expected eternal", targetBuff.BuffType)
	}
	if targetBuff.TargetPlayer() != target {
		t.Error("target buff should target target player")
	}
}

func TestCrimsonBladeHandlerBehavior(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID(), MaxHP: 10})
	player.HP = 6
	target := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID(), MaxHP: 10})
	game.AddPlayer(player)
	game.AddPlayer(target)

	handler := GetItemHandlerConfig(constants.ItemTypeCrimsonBlade).Handler
	actionCtx := engineaction.NewActionContext(game, game.Bus, gamemap.NewMapEngine(20), game.Draw)
	ctx := event.NewContext(player)
	ctx.Set("action_context", actionCtx)
	ctx.Set("target_player", target)

	handler(constants.PhaseItemUsed, ctx)

	derived := ctx.GetDerivedActions()
	if len(derived) != 2 {
		t.Fatalf("expected 2 derived actions, got %d", len(derived))
	}

	// First: PiercingDamageAction to self (sacrifice half HP)
	selfDamage, ok := derived[0].(*engineaction.DamageAction)
	if !ok {
		t.Fatalf("expected DamageAction, got %T", derived[0])
	}
	if !selfDamage.IsPiercing {
		t.Error("self damage should be piercing")
	}
	if selfDamage.Amount != 3 {
		t.Errorf("self damage = %d, expected 3 (HP/2)", selfDamage.Amount)
	}
	if selfDamage.Source() != string(constants.SourceItemCrimsonBlade) {
		t.Errorf("self damage source = %s, expected %s", selfDamage.Source(), string(constants.SourceItemCrimsonBlade))
	}

	// Second: DamageActionWithSource to target
	targetDamage, ok := derived[1].(*engineaction.DamageAction)
	if !ok {
		t.Fatalf("expected DamageAction, got %T", derived[1])
	}
	if targetDamage.Amount != 3 {
		t.Errorf("target damage = %d, expected 3", targetDamage.Amount)
	}
	if targetDamage.TargetPlayer() != target {
		t.Error("target damage should target target player")
	}
}

func TestCrimsonBladeZeroDamage(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID(), MaxHP: 10})
	player.HP = 1
	target := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID(), MaxHP: 10})
	game.AddPlayer(player)
	game.AddPlayer(target)

	handler := GetItemHandlerConfig(constants.ItemTypeCrimsonBlade).Handler
	actionCtx := engineaction.NewActionContext(game, game.Bus, gamemap.NewMapEngine(20), game.Draw)
	ctx := event.NewContext(player)
	ctx.Set("action_context", actionCtx)
	ctx.Set("target_player", target)

	handler(constants.PhaseItemUsed, ctx)

	// HP=1, damageAmount = 1/2 = 0 → no derived actions
	if len(ctx.GetDerivedActions()) > 0 {
		t.Error("CrimsonBlade should produce no actions when HP=1 (damageAmount=0)")
	}
}

func TestWisdomRingHandlerBehavior(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	game.AddPlayer(player)

	handler := GetItemHandlerConfig(constants.ItemTypeWisdomRing).Handler
	actionCtx := engineaction.NewActionContext(game, game.Bus, gamemap.NewMapEngine(20), game.Draw)
	ctx := event.NewContext(player)
	ctx.Set("action_context", actionCtx)

	handler(constants.PhaseItemUsed, ctx)

	derived := ctx.GetDerivedActions()
	if len(derived) != 1 {
		t.Fatalf("expected 1 derived action, got %d", len(derived))
	}
	addBuff, ok := derived[0].(*engineaction.AddBuffAction)
	if !ok {
		t.Fatalf("expected AddBuffAction, got %T", derived[0])
	}
	if addBuff.BuffType != constants.BuffTypeDivine {
		t.Errorf("buff type = %s, expected divine", addBuff.BuffType)
	}
	if addBuff.Source() != string(constants.SourceItemWisdomRingBuff) {
		t.Errorf("source = %s, expected %s", addBuff.Source(), string(constants.SourceItemWisdomRingBuff))
	}
}

func TestMeditationRingHandlerBehavior(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	game.AddPlayer(player)

	handler := GetItemHandlerConfig(constants.ItemTypeMeditationRing).Handler
	actionCtx := engineaction.NewActionContext(game, game.Bus, gamemap.NewMapEngine(20), game.Draw)
	ctx := event.NewContext(player)
	ctx.Set("action_context", actionCtx)

	handler(constants.PhaseItemUsed, ctx)

	derived := ctx.GetDerivedActions()
	if len(derived) != 1 {
		t.Fatalf("expected 1 derived action, got %d", len(derived))
	}
	addBuff, ok := derived[0].(*engineaction.AddBuffAction)
	if !ok {
		t.Fatalf("expected AddBuffAction, got %T", derived[0])
	}
	if addBuff.BuffType != constants.BuffTypeRain {
		t.Errorf("buff type = %s, expected rain", addBuff.BuffType)
	}
}

func TestDisciplineRingHandlerBehavior(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	game.AddPlayer(player)

	handler := GetItemHandlerConfig(constants.ItemTypeDisciplineRing).Handler
	actionCtx := engineaction.NewActionContext(game, game.Bus, gamemap.NewMapEngine(20), game.Draw)
	ctx := event.NewContext(player)
	ctx.Set("action_context", actionCtx)

	handler(constants.PhaseItemUsed, ctx)

	derived := ctx.GetDerivedActions()
	if len(derived) != 1 {
		t.Fatalf("expected 1 derived action, got %d", len(derived))
	}
	addBuff, ok := derived[0].(*engineaction.AddBuffAction)
	if !ok {
		t.Fatalf("expected AddBuffAction, got %T", derived[0])
	}
	if addBuff.BuffType != constants.BuffTypeGoldenBody {
		t.Errorf("buff type = %s, expected golden_body", addBuff.BuffType)
	}
}

func TestFoolishRingHandlerBehavior(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	game.AddPlayer(player)

	handler := GetItemHandlerConfig(constants.ItemTypeFoolishRing).Handler
	actionCtx := engineaction.NewActionContext(game, game.Bus, gamemap.NewMapEngine(20), game.Draw)
	ctx := event.NewContext(player)
	ctx.Set("action_context", actionCtx)

	handler(constants.PhaseItemUsed, ctx)

	derived := ctx.GetDerivedActions()
	if len(derived) != 2 {
		t.Fatalf("expected 2 derived actions, got %d", len(derived))
	}

	healAction, ok := derived[0].(*engineaction.HealAction)
	if !ok {
		t.Fatalf("expected HealAction, got %T", derived[0])
	}
	if healAction.Amount != 1 {
		t.Errorf("heal amount = %d, expected 1", healAction.Amount)
	}
	if healAction.Source() != string(constants.SourceItemFoolishRing) {
		t.Errorf("heal source = %s, expected %s", healAction.Source(), string(constants.SourceItemFoolishRing))
	}

	lpAction, ok := derived[1].(*engineaction.ModifyLPAction)
	if !ok {
		t.Fatalf("expected ModifyLPAction, got %T", derived[1])
	}
	if lpAction.Amount != -1 {
		t.Errorf("LP change = %d, expected -1", lpAction.Amount)
	}
}

func TestGreedyRingHandlerBehavior(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	game.AddPlayer(player)

	handler := GetItemHandlerConfig(constants.ItemTypeGreedyRing).Handler
	actionCtx := engineaction.NewActionContext(game, game.Bus, gamemap.NewMapEngine(20), game.Draw)
	ctx := event.NewContext(player)
	ctx.Set("action_context", actionCtx)

	handler(constants.PhaseItemUsed, ctx)

	derived := ctx.GetDerivedActions()
	if len(derived) != 2 {
		t.Fatalf("expected 2 derived actions, got %d", len(derived))
	}

	lpAction, ok := derived[0].(*engineaction.ModifyLPAction)
	if !ok {
		t.Fatalf("expected ModifyLPAction, got %T", derived[0])
	}
	if lpAction.Amount != 1 {
		t.Errorf("LP change = %d, expected 1", lpAction.Amount)
	}

	damageAction, ok := derived[1].(*engineaction.DamageAction)
	if !ok {
		t.Fatalf("expected DamageAction, got %T", derived[1])
	}
	if damageAction.Amount != 1 {
		t.Errorf("damage = %d, expected 1", damageAction.Amount)
	}
}

func TestWrathRingHandlerBehavior(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	game.AddPlayer(player)

	handler := GetItemHandlerConfig(constants.ItemTypeWrathRing).Handler
	actionCtx := engineaction.NewActionContext(game, game.Bus, gamemap.NewMapEngine(20), game.Draw)
	ctx := event.NewContext(player)
	ctx.Set("action_context", actionCtx)

	handler(constants.PhaseItemUsed, ctx)

	derived := ctx.GetDerivedActions()
	if len(derived) != 2 {
		t.Fatalf("expected 2 derived actions, got %d", len(derived))
	}

	damageAction, ok := derived[0].(*engineaction.DamageAction)
	if !ok {
		t.Fatalf("expected DamageAction, got %T", derived[0])
	}
	if damageAction.Amount != 1 {
		t.Errorf("damage = %d, expected 1", damageAction.Amount)
	}
	if damageAction.Source() != string(constants.SourceItemWrathRing) {
		t.Errorf("damage source = %s, expected %s", damageAction.Source(), string(constants.SourceItemWrathRing))
	}

	addBuff, ok := derived[1].(*engineaction.AddBuffAction)
	if !ok {
		t.Fatalf("expected AddBuffAction, got %T", derived[1])
	}
	if addBuff.BuffType != constants.BuffTypeWrath {
		t.Errorf("buff type = %s, expected wrath", addBuff.BuffType)
	}
}

func TestNamedBladeHandlerBehavior(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	game.AddPlayer(player)

	handler := GetItemHandlerConfig(constants.ItemTypeNamedBlade).Handler
	actionCtx := engineaction.NewActionContext(game, game.Bus, gamemap.NewMapEngine(20), game.Draw)
	ctx := event.NewContext(player)
	ctx.Set("action_context", actionCtx)

	handler(constants.PhaseItemUsed, ctx)

	derived := ctx.GetDerivedActions()
	if len(derived) != 1 {
		t.Fatalf("expected 1 derived action, got %d", len(derived))
	}
	addBuff, ok := derived[0].(*engineaction.AddBuffAction)
	if !ok {
		t.Fatalf("expected AddBuffAction, got %T", derived[0])
	}
	if addBuff.BuffType != constants.BuffTypeSavior {
		t.Errorf("buff type = %s, expected savior", addBuff.BuffType)
	}
}

func TestSageProtectionItemHandlerBehavior(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	game.AddPlayer(player)

	handler := GetItemHandlerConfig(constants.ItemTypeSageProtection).Handler
	actionCtx := engineaction.NewActionContext(game, game.Bus, gamemap.NewMapEngine(20), game.Draw)
	ctx := event.NewContext(player)
	ctx.Set("action_context", actionCtx)

	handler(constants.PhaseItemUsed, ctx)

	derived := ctx.GetDerivedActions()
	if len(derived) != 1 {
		t.Fatalf("expected 1 derived action, got %d", len(derived))
	}
	addBuff, ok := derived[0].(*engineaction.AddBuffAction)
	if !ok {
		t.Fatalf("expected AddBuffAction, got %T", derived[0])
	}
	if addBuff.BuffType != constants.BuffTypeSageProtection {
		t.Errorf("buff type = %s, expected sage_protection", addBuff.BuffType)
	}
}

func TestMagicFluteNilTargetPlayer(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	ctx := event.NewContext(player)
	// No target_player set

	handler := GetItemHandlerConfig(constants.ItemTypeMagicFlute).Handler
	err := handler(constants.PhaseItemUsed, ctx)
	if err == nil {
		t.Error("MagicFlute should return error without target_player")
	}
}

func TestCupidArrowNilTargetPlayer(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	ctx := event.NewContext(player)

	handler := GetItemHandlerConfig(constants.ItemTypeCupidArrow).Handler
	err := handler(constants.PhaseItemUsed, ctx)
	if err == nil {
		t.Error("CupidArrow should return error without target_player")
	}
}

func TestCrimsonBladeNilTargetPlayer(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	ctx := event.NewContext(player)

	handler := GetItemHandlerConfig(constants.ItemTypeCrimsonBlade).Handler
	err := handler(constants.PhaseItemUsed, ctx)
	if err == nil {
		t.Error("CrimsonBlade should return error without target_player")
	}
}
