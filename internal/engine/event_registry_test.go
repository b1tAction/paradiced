package engine

import (
	"testing"

	"github.com/b1tAction/paradiced/internal/core"
	"github.com/b1tAction/paradiced/internal/event"
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
	// Test Herb HP+1
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})

	config := GetEventHandlerConfig(constants.EventTypeHerb)
	if config == nil {
		t.Fatal("Herb should have HandlerConfig")
	}
	handler := config.Handler
	if handler == nil {
		t.Fatal("Herb should have Handler")
	}

	ctx := event.NewContext(player)
	handler(constants.PhaseOnLand, ctx)

	// Should signal hp_change=1
	hpChange, err := ctx.GetInt("hp_change")
	if err != nil {
		t.Error("hp_change should be set")
	}
	if hpChange != 1 {
		t.Errorf("hp_change = %d, expected 1", hpChange)
	}
}

func TestMilkTeaEventHandler(t *testing.T) {
	// Test MilkTea LP+1
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player.LP = 5

	config := GetEventHandlerConfig(constants.EventTypeMilkTea)
	if config == nil {
		t.Fatal("MilkTea should have HandlerConfig")
	}
	handler := config.Handler
	if handler == nil {
		t.Fatal("MilkTea should have Handler")
	}

	ctx := event.NewContext(player)
	handler(constants.PhaseOnLand, ctx)

	// LP should be 6 after handler execution
	if player.LP != 6 {
		t.Errorf("LP = %d, expected 6 (LP+1)", player.LP)
	}
}

func TestRelicEventHandler(t *testing.T) {
	// Test Relic draw item
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})

	config := GetEventHandlerConfig(constants.EventTypeRelic)
	if config == nil {
		t.Fatal("Relic should have HandlerConfig")
	}
	handler := config.Handler
	if handler == nil {
		t.Fatal("Relic should have Handler")
	}

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
	// Test DivineBless gives Divine buff
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})

	config := GetEventHandlerConfig(constants.EventTypeDivineBless)
	if config == nil {
		t.Fatal("DivineBless should have HandlerConfig")
	}
	handler := config.Handler
	if handler == nil {
		t.Fatal("DivineBless should have Handler")
	}

	ctx := event.NewContext(player)
	handler(constants.PhaseOnLand, ctx)

	// Should signal give_buff_type=Divine
	buffType, err := ctx.GetString("give_buff_type")
	if err != nil {
		t.Error("give_buff_type should be set")
	}
	if buffType != string(constants.BuffTypeDivine) {
		t.Errorf("give_buff_type = %s, expected %s", buffType, constants.BuffTypeDivine)
	}

	duration, err := ctx.GetInt("give_buff_duration")
	if err != nil {
		t.Error("give_buff_duration should be set")
	}
	if duration != 3 {
		t.Errorf("give_buff_duration = %d, expected 3", duration)
	}
}

func TestExchangeEventHandler(t *testing.T) {
	// Test Exchange swap position
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})

	config := GetEventHandlerConfig(constants.EventTypeExchange)
	if config == nil {
		t.Fatal("Exchange should have HandlerConfig")
	}
	handler := config.Handler
	if handler == nil {
		t.Fatal("Exchange should have Handler")
	}

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
	// Test HiddenBuff gives Hidden buff
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})

	config := GetEventHandlerConfig(constants.EventTypeHiddenBuff)
	handler := config.Handler

	ctx := event.NewContext(player)
	handler(constants.PhaseOnLand, ctx)

	// Should signal give_buff_type=Hidden
	buffType, err := ctx.GetString("give_buff_type")
	if err != nil {
		t.Error("give_buff_type should be set")
	}
	if buffType != string(constants.BuffTypeHidden) {
		t.Errorf("give_buff_type = %s, expected %s", buffType, constants.BuffTypeHidden)
	}
}

func TestTasteTestEventHandler(t *testing.T) {
	// Test TasteTest random buff
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})

	config := GetEventHandlerConfig(constants.EventTypeTasteTest)
	handler := config.Handler

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
	// Test Mosquito HP-1
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})

	config := GetEventHandlerConfig(constants.EventTypeMosquito)
	handler := config.Handler

	ctx := event.NewContext(player)
	handler(constants.PhaseOnLand, ctx)

	// Should signal hp_change=-1
	hpChange, err := ctx.GetInt("hp_change")
	if err != nil {
		t.Error("hp_change should be set")
	}
	if hpChange != -1 {
		t.Errorf("hp_change = %d, expected -1", hpChange)
	}
}

func TestGhostHitEventHandler(t *testing.T) {
	// Test GhostHit HP-1
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})

	config := GetEventHandlerConfig(constants.EventTypeGhostHit)
	handler := config.Handler

	ctx := event.NewContext(player)
	handler(constants.PhaseOnLand, ctx)

	// Should signal hp_change=-1
	hpChange, err := ctx.GetInt("hp_change")
	if err != nil {
		t.Error("hp_change should be set")
	}
	if hpChange != -1 {
		t.Errorf("hp_change = %d, expected -1", hpChange)
	}
}

func TestDogPoopEventHandler(t *testing.T) {
	// Test DogPoop LP-1
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player.LP = 5

	config := GetEventHandlerConfig(constants.EventTypeDogPoop)
	handler := config.Handler

	ctx := event.NewContext(player)
	handler(constants.PhaseOnLand, ctx)

	// LP should be 4 after handler execution
	if player.LP != 4 {
		t.Errorf("LP = %d, expected 4 (LP-1)", player.LP)
	}
}

func TestThiefEventHandler(t *testing.T) {
	// Test Thief lose item
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})

	config := GetEventHandlerConfig(constants.EventTypeThief)
	handler := config.Handler

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
	// Test CurseBuddha gives Curse buff
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})

	config := GetEventHandlerConfig(constants.EventTypeCurseBuddha)
	handler := config.Handler

	ctx := event.NewContext(player)
	handler(constants.PhaseOnLand, ctx)

	// Should signal give_buff_type=Curse
	buffType, err := ctx.GetString("give_buff_type")
	if err != nil {
		t.Error("give_buff_type should be set")
	}
	if buffType != string(constants.BuffTypeCurse) {
		t.Errorf("give_buff_type = %s, expected %s", buffType, constants.BuffTypeCurse)
	}

	duration, err := ctx.GetInt("give_buff_duration")
	if err != nil {
		t.Error("give_buff_duration should be set")
	}
	if duration != 3 {
		t.Errorf("give_buff_duration = %d, expected 3", duration)
	}
}

func TestLostWayEventHandler(t *testing.T) {
	// Test LostWay gives Lost buff
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})

	config := GetEventHandlerConfig(constants.EventTypeLostWay)
	handler := config.Handler

	ctx := event.NewContext(player)
	handler(constants.PhaseOnLand, ctx)

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

func TestThunderEventHandler(t *testing.T) {
	// Test Thunder instant death
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})

	config := GetEventHandlerConfig(constants.EventTypeThunder)
	handler := config.Handler

	ctx := event.NewContext(player)
	handler(constants.PhaseOnLand, ctx)

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