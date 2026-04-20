package engine

import (
	"testing"

	"github.com/b1tAction/paradiced/internal/core"
	engineaction "github.com/b1tAction/paradiced/internal/engine/action"
	"github.com/b1tAction/paradiced/internal/event"
	"github.com/b1tAction/paradiced/internal/gamemap"
	"github.com/b1tAction/paradiced/pkg/constants"
	"github.com/b1tAction/paradiced/pkg/id"
)

// ========== EventRegistry Tests ==========

func TestAllEventsHaveDefinition(t *testing.T) {
	// All Events should have Definition registered
	allEvents := GetAllEventTypes()
	for _, et := range allEvents {
		def := GetEventDefinition(et)
		if def == nil {
			t.Errorf("EventType(%s) should have Definition", et)
		}
	}
}

func TestAllEventsHaveHandlerConfig(t *testing.T) {
	// All Events should have HandlerConfig registered
	allEvents := GetAllEventTypes()
	for _, et := range allEvents {
		config := GetEventHandlerConfig(et)
		if config == nil {
			t.Errorf("EventType(%s) should have HandlerConfig", et)
		}
	}
}

func TestEventDefinitionsFields(t *testing.T) {
	tests := []struct {
		eventType   constants.EventType
		eval        constants.Evaluation
		englishName string
		name        string
	}{
		// Good Events
		{constants.EventTypeHerb, constants.EvaluationMildGood, "Herb", "采集到草药"},
		{constants.EventTypeMilkTea, constants.EvaluationGood, "MilkTea", "捡到奶茶"},
		{constants.EventTypeRelic, constants.EvaluationVeryGood, "Relic", "捡到勇士的圣遗物"},
		{constants.EventTypeDivineBless, constants.EvaluationExcellent, "DivineBless", "受到天使眷顾"},
		// Neutral Events
		{constants.EventTypeExchange, constants.EvaluationNeutral, "Exchange", "交换"},
		{constants.EventTypeHiddenBuff, constants.EvaluationGood, "HiddenBuff", "麻了"},
		{constants.EventTypeTasteTest, constants.EvaluationMixed, "TasteTest", "这是什么？尝一口"},
		// Bad Events
		{constants.EventTypeMosquito, constants.EvaluationMildBad, "Mosquito", "被蚊虫叮咬"},
		{constants.EventTypeGhostHit, constants.EvaluationMildBad, "GhostHit", "偶遇孤魂野鬼"},
		{constants.EventTypeDogPoop, constants.EvaluationMildBad, "DogPoop", "踩到了狗屎"},
		{constants.EventTypeThief, constants.EvaluationBad, "Thief", "啊？！贼"},
		{constants.EventTypeCurseBuddha, constants.EvaluationBad, "CurseBuddha", "虔诚拜三拜"},
		{constants.EventTypeLostWay, constants.EvaluationMildBad, "LostWay", "迷途"},
		{constants.EventTypeThunder, constants.EvaluationVeryBad, "Thunder", "雷劫"},
	}

	for _, tt := range tests {
		def := GetEventDefinition(tt.eventType)
		if def == nil {
			t.Errorf("EventType(%s) has no Definition", tt.eventType)
			continue
		}

		if def.Eval != tt.eval {
			t.Errorf("%s.Eval = %d, expected %d", tt.eventType, def.Eval, tt.eval)
		}

		if def.EnglishName != tt.englishName {
			t.Errorf("%s.EnglishName = %s, expected %s", tt.eventType, def.EnglishName, tt.englishName)
		}

		if def.Name != tt.name {
			t.Errorf("%s.Name = %s, expected %s", tt.eventType, def.Name, tt.name)
		}
	}
}

// ========== Event Handler Tests ==========

func TestHerbEventHandler(t *testing.T) {
	// Test Herb HP+1 through Action system
	game := NewGame(id.NewGameID(), 0)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player.HP = 5
	game.AddPlayer(player)
	game.Log.StartTurn(1, 0, player.ID.UUID())

	handler := GetEventHandlerConfig(constants.EventTypeHerb).Handler

	actionCtx := engineaction.NewActionContext(game, game.Bus, gamemap.NewMapEngine(20), game.Draw)
	ctx := event.NewContext(player)
	ctx.Set("action_context", actionCtx)

	handler(constants.PhaseOnLand, ctx)

	// Should have HealAction derived action
	derived := ctx.GetDerivedActions()
	if len(derived) != 1 {
		t.Fatalf("expected 1 derived action, got %d", len(derived))
	}

	healAction, ok := derived[0].(*engineaction.HealAction)
	if !ok {
		t.Fatal("expected HealAction")
	}
	if healAction.Amount != 1 {
		t.Errorf("HealAction.Amount = %d, expected 1", healAction.Amount)
	}
}

func TestMilkTeaEventHandler(t *testing.T) {
	// Test MilkTea LP+1 through Action system
	game := NewGame(id.NewGameID(), 0)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player.LP = 5
	game.AddPlayer(player)
	game.Log.StartTurn(1, 0, player.ID.UUID())

	handler := GetEventHandlerConfig(constants.EventTypeMilkTea).Handler

	actionCtx := engineaction.NewActionContext(game, game.Bus, gamemap.NewMapEngine(20), game.Draw)
	ctx := event.NewContext(player)
	ctx.Set("action_context", actionCtx)

	handler(constants.PhaseOnLand, ctx)

	// Execute derived actions
	for _, derived := range ctx.GetDerivedActions() {
		if execAction, ok := derived.(engineaction.Action); ok {
			actionCtx.PushDerivedAction(execAction)
		}
	}
	actionCtx.ProcessQueue()

	if player.LP != 6 {
		t.Errorf("LP = %d, expected 6 (LP+1)", player.LP)
	}
}

func TestRelicEventHandler(t *testing.T) {
	// Test Relic draw item
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})

	handler := GetEventHandlerConfig(constants.EventTypeRelic).Handler

	ctx := event.NewContext(player)
	handler(constants.PhaseOnLand, ctx)

	// Should signal draw_item
	drawItem, err := ctx.GetBool("draw_item")
	if err != nil {
		t.Error("draw_item should be set")
	}
	if !drawItem {
		t.Error("draw_item should be true")
	}
}

func TestDivineBlessEventHandler(t *testing.T) {
	// Test DivineBless gives Divine buff through Action system
	game := NewGame(id.NewGameID(), 0)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	game.AddPlayer(player)

	handler := GetEventHandlerConfig(constants.EventTypeDivineBless).Handler

	actionCtx := engineaction.NewActionContext(game, game.Bus, gamemap.NewMapEngine(20), game.Draw)
	ctx := event.NewContext(player)
	ctx.Set("action_context", actionCtx)

	handler(constants.PhaseOnLand, ctx)

	// Should have AddBuffAction derived action
	derived := ctx.GetDerivedActions()
	if len(derived) != 1 {
		t.Fatalf("expected 1 derived action, got %d", len(derived))
	}

	addBuffAction, ok := derived[0].(*engineaction.AddBuffAction)
	if !ok {
		t.Fatal("expected AddBuffAction")
	}
	if addBuffAction.BuffType != constants.BuffTypeDivine {
		t.Errorf("BuffType = %s, expected Divine", addBuffAction.BuffType)
	}
	if addBuffAction.Duration != 3 {
		t.Errorf("Duration = %d, expected 3", addBuffAction.Duration)
	}
}

func TestExchangeEventHandler(t *testing.T) {
	// Test Exchange swap position
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})

	handler := GetEventHandlerConfig(constants.EventTypeExchange).Handler

	ctx := event.NewContext(player)
	handler(constants.PhaseOnLand, ctx)

	// Should signal swap_position
	swap, err := ctx.GetBool("swap_position")
	if err != nil {
		t.Error("swap_position should be set")
	}
	if !swap {
		t.Error("swap_position should be true")
	}
}

func TestHiddenBuffEventHandler(t *testing.T) {
	// Test HiddenBuff gives Hidden buff through Action system
	game := NewGame(id.NewGameID(), 0)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	game.AddPlayer(player)

	handler := GetEventHandlerConfig(constants.EventTypeHiddenBuff).Handler

	actionCtx := engineaction.NewActionContext(game, game.Bus, gamemap.NewMapEngine(20), game.Draw)
	ctx := event.NewContext(player)
	ctx.Set("action_context", actionCtx)

	handler(constants.PhaseOnLand, ctx)

	// Should have AddBuffAction derived action
	derived := ctx.GetDerivedActions()
	if len(derived) != 1 {
		t.Fatalf("expected 1 derived action, got %d", len(derived))
	}

	addBuffAction, ok := derived[0].(*engineaction.AddBuffAction)
	if !ok {
		t.Fatal("expected AddBuffAction")
	}
	if addBuffAction.BuffType != constants.BuffTypeHidden {
		t.Errorf("BuffType = %s, expected Hidden", addBuffAction.BuffType)
	}
}

func TestTasteTestEventHandler(t *testing.T) {
	// Test TasteTest random buff
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})

	handler := GetEventHandlerConfig(constants.EventTypeTasteTest).Handler

	ctx := event.NewContext(player)
	handler(constants.PhaseOnLand, ctx)

	// Should signal random_buff
	randomBuff, err := ctx.GetBool("random_buff")
	if err != nil {
		t.Error("random_buff should be set")
	}
	if !randomBuff {
		t.Error("random_buff should be true")
	}
}

func TestMosquitoEventHandler(t *testing.T) {
	// Test Mosquito HP-1 through Action system
	game := NewGame(id.NewGameID(), 0)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player.HP = 5
	game.AddPlayer(player)

	handler := GetEventHandlerConfig(constants.EventTypeMosquito).Handler

	actionCtx := engineaction.NewActionContext(game, game.Bus, gamemap.NewMapEngine(20), game.Draw)
	ctx := event.NewContext(player)
	ctx.Set("action_context", actionCtx)

	handler(constants.PhaseOnLand, ctx)

	// Should have DamageAction derived action
	derived := ctx.GetDerivedActions()
	if len(derived) != 1 {
		t.Fatalf("expected 1 derived action, got %d", len(derived))
	}

	damageAction, ok := derived[0].(*engineaction.DamageAction)
	if !ok {
		t.Fatal("expected DamageAction")
	}
	if damageAction.Amount != 1 {
		t.Errorf("DamageAction.Amount = %d, expected 1", damageAction.Amount)
	}
}

func TestGhostHitEventHandler(t *testing.T) {
	// Test GhostHit HP-1 through Action system
	game := NewGame(id.NewGameID(), 0)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player.HP = 5
	game.AddPlayer(player)

	handler := GetEventHandlerConfig(constants.EventTypeGhostHit).Handler

	actionCtx := engineaction.NewActionContext(game, game.Bus, gamemap.NewMapEngine(20), game.Draw)
	ctx := event.NewContext(player)
	ctx.Set("action_context", actionCtx)

	handler(constants.PhaseOnLand, ctx)

	// Should have DamageAction derived action
	derived := ctx.GetDerivedActions()
	if len(derived) != 1 {
		t.Fatalf("expected 1 derived action, got %d", len(derived))
	}

	damageAction, ok := derived[0].(*engineaction.DamageAction)
	if !ok {
		t.Fatal("expected DamageAction")
	}
	if damageAction.Amount != 1 {
		t.Errorf("DamageAction.Amount = %d, expected 1", damageAction.Amount)
	}
}

func TestDogPoopEventHandler(t *testing.T) {
	// Test DogPoop LP-1 through Action system
	game := NewGame(id.NewGameID(), 0)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player.LP = 5
	game.AddPlayer(player)
	game.Log.StartTurn(1, 0, player.ID.UUID())

	handler := GetEventHandlerConfig(constants.EventTypeDogPoop).Handler

	actionCtx := engineaction.NewActionContext(game, game.Bus, gamemap.NewMapEngine(20), game.Draw)
	ctx := event.NewContext(player)
	ctx.Set("action_context", actionCtx)

	handler(constants.PhaseOnLand, ctx)

	// Execute derived actions
	for _, derived := range ctx.GetDerivedActions() {
		if execAction, ok := derived.(engineaction.Action); ok {
			actionCtx.PushDerivedAction(execAction)
		}
	}
	actionCtx.ProcessQueue()

	if player.LP != 4 {
		t.Errorf("LP = %d, expected 4 (LP-1)", player.LP)
	}
}

func TestThiefEventHandler(t *testing.T) {
	// Test Thief lose item
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})

	handler := GetEventHandlerConfig(constants.EventTypeThief).Handler

	ctx := event.NewContext(player)
	handler(constants.PhaseOnLand, ctx)

	// Should signal lose_item
	loseItem, err := ctx.GetBool("lose_item")
	if err != nil {
		t.Error("lose_item should be set")
	}
	if !loseItem {
		t.Error("lose_item should be true")
	}
}

func TestCurseBuddhaEventHandler(t *testing.T) {
	// Test CurseBuddha gives Curse buff through Action system
	game := NewGame(id.NewGameID(), 0)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	game.AddPlayer(player)

	handler := GetEventHandlerConfig(constants.EventTypeCurseBuddha).Handler

	actionCtx := engineaction.NewActionContext(game, game.Bus, gamemap.NewMapEngine(20), game.Draw)
	ctx := event.NewContext(player)
	ctx.Set("action_context", actionCtx)

	handler(constants.PhaseOnLand, ctx)

	// Should have AddBuffAction derived action
	derived := ctx.GetDerivedActions()
	if len(derived) != 1 {
		t.Fatalf("expected 1 derived action, got %d", len(derived))
	}

	addBuffAction, ok := derived[0].(*engineaction.AddBuffAction)
	if !ok {
		t.Fatal("expected AddBuffAction")
	}
	if addBuffAction.BuffType != constants.BuffTypeCurse {
		t.Errorf("BuffType = %s, expected Curse", addBuffAction.BuffType)
	}
	if addBuffAction.Duration != 3 {
		t.Errorf("Duration = %d, expected 3", addBuffAction.Duration)
	}
}

func TestLostWayEventHandler(t *testing.T) {
	// Test LostWay gives Lost buff through Action system
	game := NewGame(id.NewGameID(), 0)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	game.AddPlayer(player)

	handler := GetEventHandlerConfig(constants.EventTypeLostWay).Handler

	actionCtx := engineaction.NewActionContext(game, game.Bus, gamemap.NewMapEngine(20), game.Draw)
	ctx := event.NewContext(player)
	ctx.Set("action_context", actionCtx)

	handler(constants.PhaseOnLand, ctx)

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
	if addBuffAction.Duration != 1 {
		t.Errorf("Duration = %d, expected 1", addBuffAction.Duration)
	}
}

func TestThunderEventHandler(t *testing.T) {
	// Test Thunder instant death through Action system
	game := NewGame(id.NewGameID(), 0)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player.HP = 5
	game.AddPlayer(player)

	handler := GetEventHandlerConfig(constants.EventTypeThunder).Handler

	actionCtx := engineaction.NewActionContext(game, game.Bus, gamemap.NewMapEngine(20), game.Draw)
	ctx := event.NewContext(player)
	ctx.Set("action_context", actionCtx)

	handler(constants.PhaseOnLand, ctx)

	// Should have DamageAction derived action that sets HP to 0
	derived := ctx.GetDerivedActions()
	if len(derived) != 1 {
		t.Fatalf("expected 1 derived action, got %d", len(derived))
	}

	damageAction, ok := derived[0].(*engineaction.DamageAction)
	if !ok {
		t.Fatal("expected DamageAction")
	}
	if damageAction.Amount != 5 {
		t.Errorf("DamageAction.Amount = %d, expected 5 (current HP)", damageAction.Amount)
	}

	// Should signal instant_death
	instantDeath, err := ctx.GetBool("instant_death")
	if err != nil {
		t.Error("instant_death should be set")
	}
	if !instantDeath {
		t.Error("instant_death should be true")
	}
}

func TestGetEventName(t *testing.T) {
	tests := []struct {
		eventType    constants.EventType
		expectedName string
	}{
		{constants.EventTypeHerb, "采集到草药"},
		{constants.EventTypeMilkTea, "捡到奶茶"},
		{constants.EventTypeRelic, "捡到勇士的圣遗物"},
		{constants.EventTypeDivineBless, "受到天使眷顾"},
		{constants.EventTypeExchange, "交换"},
		{constants.EventTypeMosquito, "被蚊虫叮咬"},
		{constants.EventTypeThunder, "雷劫"},
	}

	for _, tt := range tests {
		name := GetEventName(tt.eventType)
		if name != tt.expectedName {
			t.Errorf("GetEventName(%s) = %s, expected %s", tt.eventType, name, tt.expectedName)
		}
	}
}

func TestGetEventTypesByCategory(t *testing.T) {
	goodEvents := GetEventTypesByCategory("Good")
	if len(goodEvents) == 0 {
		t.Error("Good events should not be empty")
	}

	badEvents := GetEventTypesByCategory("Bad")
	if len(badEvents) == 0 {
		t.Error("Bad events should not be empty")
	}

	neutralEvents := GetEventTypesByCategory("Neutral")
	if len(neutralEvents) == 0 {
		t.Error("Neutral events should not be empty")
	}

	// Unknown category returns all
	allEvents := GetEventTypesByCategory("Unknown")
	if len(allEvents) != len(GetAllEventTypes()) {
		t.Error("Unknown category should return all events")
	}
}
// ========== Edge Case Tests: nil player/context ==========

func TestEventHandlerWithNilContext(t *testing.T) {
	// All event handlers should gracefully handle nil context
	eventTypes := GetAllEventTypes()
	for _, et := range eventTypes {
		handler := GetEventHandlerConfig(et).Handler
		if handler == nil {
			continue
		}
		// Should not panic with nil context
		handler(constants.PhaseOnLand, nil)
	}
}

func TestEventHandlerWithNilPlayer(t *testing.T) {
	// All event handlers should gracefully handle nil player in context
	eventTypes := GetAllEventTypes()
	for _, et := range eventTypes {
		handler := GetEventHandlerConfig(et).Handler
		if handler == nil {
			continue
		}
		ctx := event.NewContext(nil) // nil player
		handler(constants.PhaseOnLand, ctx)
		// Should not produce derived actions
		if len(ctx.GetDerivedActions()) > 0 {
			t.Errorf("EventType %s should not produce actions with nil player", et)
		}
	}
}

func TestEventModifyHPHandlerNilActionContext(t *testing.T) {
	// createEventModifyHPHandler requires ActionContext
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	ctx := event.NewContext(player)
	// No action_context set

	handler := GetEventHandlerConfig(constants.EventTypeHerb).Handler
	handler(constants.PhaseOnLand, ctx)

	// Should not produce derived actions without ActionContext
	if len(ctx.GetDerivedActions()) > 0 {
		t.Error("Herb handler should not produce actions without ActionContext")
	}
}

func TestEventModifyLPHandlerNilActionContext(t *testing.T) {
	// createEventModifyLPHandler requires ActionContext
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	ctx := event.NewContext(player)
	// No action_context set

	handler := GetEventHandlerConfig(constants.EventTypeMilkTea).Handler
	handler(constants.PhaseOnLand, ctx)

	// Should not produce derived actions without ActionContext
	if len(ctx.GetDerivedActions()) > 0 {
		t.Error("MilkTea handler should not produce actions without ActionContext")
	}
}

func TestEventGiveBuffHandlerNilActionContext(t *testing.T) {
	// createEventGiveBuffHandler requires ActionContext
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	ctx := event.NewContext(player)
	// No action_context set

	handler := GetEventHandlerConfig(constants.EventTypeDivineBless).Handler
	handler(constants.PhaseOnLand, ctx)

	// Should not produce derived actions without ActionContext
	if len(ctx.GetDerivedActions()) > 0 {
		t.Error("DivineBless handler should not produce actions without ActionContext")
	}
}

func TestThunderHandlerNilPlayer(t *testing.T) {
	// Thunder handler sets HP to 0, requires player
	ctx := event.NewContext(nil)
	handler := GetEventHandlerConfig(constants.EventTypeThunder).Handler
	handler(constants.PhaseOnLand, ctx)

	// Should not panic
}

func TestExchangeHandlerNilPlayer(t *testing.T) {
	// Exchange handler signals position swap, requires player
	ctx := event.NewContext(nil)
	handler := GetEventHandlerConfig(constants.EventTypeExchange).Handler
	handler(constants.PhaseOnLand, ctx)

	// Should not panic
}

func TestThiefHandlerNilPlayer(t *testing.T) {
	// Thief handler signals item loss, requires player
	ctx := event.NewContext(nil)
	handler := GetEventHandlerConfig(constants.EventTypeThief).Handler
	handler(constants.PhaseOnLand, ctx)

	// Should not panic
}

func TestTasteTestHandlerNilPlayer(t *testing.T) {
	// TasteTest handler signals random buff, requires player
	ctx := event.NewContext(nil)
	handler := GetEventHandlerConfig(constants.EventTypeTasteTest).Handler
	handler(constants.PhaseOnLand, ctx)

	// Should not panic
}

func TestRelicHandlerNilPlayer(t *testing.T) {
	// Relic handler signals draw item, requires player
	ctx := event.NewContext(nil)
	handler := GetEventHandlerConfig(constants.EventTypeRelic).Handler
	handler(constants.PhaseOnLand, ctx)

	// Should not panic, and should not set draw_item
	drawItem, err := ctx.GetBool("draw_item")
	if err == nil && drawItem {
		t.Error("Relic handler should not set draw_item with nil player")
	}
}
