package event

import (
	"testing"

	"github.com/b1tAction/fated/pkg/constants"
)

// ========== EventType Tests ==========

func TestEventTypeIsValid(t *testing.T) {
	validEvents := []constants.EventType{
		constants.EventTypeHerb, constants.EventTypeMilkTea, constants.EventTypeRelic, constants.EventTypeDivineBless,
		constants.EventTypeExchange, constants.EventTypeHiddenBuff, constants.EventTypeTasteTest,
		constants.EventTypeMosquito, constants.EventTypeGhostHit, constants.EventTypeDogPoop,
		constants.EventTypeThief, constants.EventTypeCurseBuddha, constants.EventTypeLostWay, constants.EventTypeThunder,
	}
	for _, et := range validEvents {
		if !et.IsValid() {
			t.Errorf("EventType(%s).IsValid() should be true", et)
		}
	}

	invalidEvents := []constants.EventType{constants.EventTypeNone, constants.EventType(""), constants.EventType("invalid")}
	for _, et := range invalidEvents {
		if et.IsValid() {
			t.Errorf("EventType(%s).IsValid() should be false", et)
		}
	}
}

func TestEventTypeGetEvaluation(t *testing.T) {
	// Good events
	goodTests := []struct {
		et       constants.EventType
		expected constants.Evaluation
	}{
		{constants.EventTypeHerb, constants.EvaluationMildGood},
		{constants.EventTypeMilkTea, constants.EvaluationGood},
		{constants.EventTypeRelic, constants.EvaluationVeryGood},
		{constants.EventTypeDivineBless, constants.EvaluationExcellent},
	}
	for _, tt := range goodTests {
		result := GetEventEvaluation(tt.et)
		if result != tt.expected {
			t.Errorf("GetEventEvaluation(%s) = %d, expected %d", tt.et, result, tt.expected)
		}
		if !result.IsGood() {
			t.Errorf("EventType(%s) evaluation should be Good", tt.et)
		}
	}

	// Neutral events
	neutralTests := []struct {
		et       constants.EventType
		expected constants.Evaluation
	}{
		{constants.EventTypeExchange, constants.EvaluationNeutral},
		{constants.EventTypeHiddenBuff, constants.EvaluationGood},
		{constants.EventTypeTasteTest, constants.EvaluationMixed},
	}
	for _, tt := range neutralTests {
		result := GetEventEvaluation(tt.et)
		if result != tt.expected {
			t.Errorf("GetEventEvaluation(%s) = %d, expected %d", tt.et, result, tt.expected)
		}
	}

	// Bad events
	badTests := []struct {
		et       constants.EventType
		expected constants.Evaluation
	}{
		{constants.EventTypeMosquito, constants.EvaluationMildBad},
		{constants.EventTypeGhostHit, constants.EvaluationMildBad},
		{constants.EventTypeDogPoop, constants.EvaluationMildBad},
		{constants.EventTypeThief, constants.EvaluationBad},
		{constants.EventTypeCurseBuddha, constants.EvaluationBad},
		{constants.EventTypeLostWay, constants.EvaluationMildBad},
		{constants.EventTypeThunder, constants.EvaluationVeryBad},
	}
	for _, tt := range badTests {
		result := GetEventEvaluation(tt.et)
		if result != tt.expected {
			t.Errorf("GetEventEvaluation(%s) = %d, expected %d", tt.et, result, tt.expected)
		}
		if !result.IsBad() {
			t.Errorf("EventType(%s) evaluation should be Bad", tt.et)
		}
	}
}

func TestEventTypeGetEventDefinition(t *testing.T) {
	// Test good event definition
	def := GetEventDefinition(constants.EventTypeHerb)
	if def == nil {
		t.Fatal("EventTypeHerb should have definition")
	}
	if def.Name != "采集到草药" {
		t.Errorf("def.Name = %s, expected 采集到草药", def.Name)
	}
	if def.Eval != constants.EvaluationMildGood {
		t.Errorf("def.Eval = %d, expected MildGood(%d)", def.Eval, constants.EvaluationMildGood)
	}
	if def.HPChange != 1 {
		t.Errorf("def.HPChange = %d, expected 1", def.HPChange)
	}

	// Test bad event definition
	def = GetEventDefinition(constants.EventTypeThunder)
	if def == nil {
		t.Fatal("EventTypeThunder should have definition")
	}
	if def.Name != "雷劫" {
		t.Errorf("def.Name = %s, expected 雷劫", def.Name)
	}
	if def.Eval != constants.EvaluationVeryBad {
		t.Errorf("def.Eval = %d, expected VeryBad(%d)", def.Eval, constants.EvaluationVeryBad)
	}
	if def.HPChange != -999 {
		t.Errorf("def.HPChange = %d, expected -999 (HP归零)", def.HPChange)
	}

	// Test Buff-giving event
	def = GetEventDefinition(constants.EventTypeDivineBless)
	if def == nil {
		t.Fatal("EventTypeDivineBless should have definition")
	}
	if def.BuffType != constants.BuffTypeDivine {
		t.Errorf("def.BuffType = %s, expected 'divine'", def.BuffType)
	}

	// Test item-giving event
	def = GetEventDefinition(constants.EventTypeRelic)
	if def == nil {
		t.Fatal("EventTypeRelic should have definition")
	}
	if def.SpecialEffect != constants.SpecialDrawItem {
		t.Errorf("def.SpecialEffect = %s, expected 'draw_item'", def.SpecialEffect)
	}

	// Test unknown event
	def = GetEventDefinition(constants.EventType("invalid"))
	if def != nil {
		t.Error("unknown EventType should return nil definition")
	}
}

func TestEventDefinitionEvaluationConsistency(t *testing.T) {
	// Verify all event definitions have consistent Evaluation
	for _, et := range GetAllEventTypes() {
		def := GetEventDefinition(et)
		if def == nil {
			t.Errorf("EventType(%s) has no definition", et)
			continue
		}
		eval := GetEventEvaluation(et)
		if def.Eval != eval {
			t.Errorf("EventType(%s) definition.Eval(%d) != GetEventEvaluation(%d)", et, def.Eval, eval)
		}
	}
}

// ========== Registry Tests ==========

func TestGlobalEventRegistryEventTypes(t *testing.T) {
	allEvents := GetAllEventTypes()
	if len(allEvents) != 14 {
		t.Errorf("AllEventTypes count = %d, expected 14", len(allEvents))
	}
}

func TestGetEventTypesByEvaluationRange(t *testing.T) {
	// Get excellent events (90~100)
	excellentEvents := GlobalEventRegistry.GetEventTypesByEvaluationRange(90, 100)
	if len(excellentEvents) != 2 {
		t.Errorf("excellent events count = %d, expected 2 (Relic+DivineBless)", len(excellentEvents))
	}

	// Get good events (66~100, includes HiddenBuff)
	goodEvents := GlobalEventRegistry.GetEventTypesByEvaluationRange(66, 100)
	if len(goodEvents) != 5 {
		t.Errorf("good events count = %d, expected 5 (Herb+MilkTea+Relic+DivineBless+HiddenBuff)", len(goodEvents))
	}

	// Get mild bad events (30~40)
	mildBadEvents := GlobalEventRegistry.GetEventTypesByEvaluationRange(30, 40)
	if len(mildBadEvents) != 4 {
		t.Errorf("mild bad events count = %d, expected 4 (Mosquito+GhostHit+DogPoop+LostWay)", len(mildBadEvents))
	}

	// Get very bad events (0~15)
	veryBadEvents := GlobalEventRegistry.GetEventTypesByEvaluationRange(0, 15)
	if len(veryBadEvents) != 1 {
		t.Errorf("very bad events count = %d, expected 1 (Thunder)", len(veryBadEvents))
	}
}

func TestGetEventTypesByCategory(t *testing.T) {
	good := GetEventTypesByCategory("Good")
	if len(good) != 5 {
		t.Errorf("Good events count = %d, expected 5", len(good))
	}

	neutral := GetEventTypesByCategory("Neutral")
	// Only Exchange(50) and TasteTest(55) are Neutral
	if len(neutral) != 2 {
		t.Errorf("Neutral events count = %d, expected 2", len(neutral))
	}

	bad := GetEventTypesByCategory("Bad")
	if len(bad) != 7 {
		t.Errorf("Bad events count = %d, expected 7", len(bad))
	}

	all := GetEventTypesByCategory("Unknown")
	if len(all) != 14 {
		t.Errorf("unknown category should return all events, got %d", len(all))
	}
}

func TestGetAllEventDefinitions(t *testing.T) {
	defs := GetAllEventDefinitions()

	if len(defs) != 14 {
		t.Errorf("definitions count = %d, expected 14", len(defs))
	}

	// Verify each definition has valid Evaluation
	for _, def := range defs {
		if !def.Eval.IsValid() {
			t.Errorf("EventDefinition for %s has invalid Evaluation %d", def.Name, def.Eval)
		}
	}
}

func TestEventRegistryEvaluationDistribution(t *testing.T) {
	// Count events by evaluation range
	var bad, neutral, good int
	for _, et := range GetAllEventTypes() {
		eval := GetEventEvaluation(et)
		if eval <= constants.EvaluationBadThreshold {
			bad++
		} else if eval <= constants.EvaluationNeutralThreshold {
			neutral++
		} else {
			good++
		}
	}

	// Verify distribution
	// Good: Herb(70), MilkTea(80), Relic(90), DivineBless(100), HiddenBuff(80) = 5
	// Neutral: Exchange(50), TasteTest(55) = 2
	// Bad: 7
	t.Logf("Distribution: Bad=%d, Neutral=%d, Good=%d", bad, neutral, good)

	// Verify Registry auto-classification is correct
	badEvents := GetEventTypesByCategory("Bad")
	if len(badEvents) != bad {
		t.Errorf("BadEvents count mismatch: registry=%d, calculated=%d", len(badEvents), bad)
	}
	goodEvents := GetEventTypesByCategory("Good")
	if len(goodEvents) != good {
		t.Errorf("GoodEvents count mismatch: registry=%d, calculated=%d", len(goodEvents), good)
	}
}

func TestGetEventString(t *testing.T) {
	tests := []struct {
		et       constants.EventType
		expected string
	}{
		{constants.EventTypeHerb, "Herb"},
		{constants.EventTypeMilkTea, "MilkTea"},
		{constants.EventTypeThunder, "Thunder"},
		{constants.EventType("invalid"), "Unknown"},
	}

	for _, tt := range tests {
		result := GetEventString(tt.et)
		if result != tt.expected {
			t.Errorf("GetEventString(%s) = %s, expected %s", tt.et, result, tt.expected)
		}
	}
}

func TestGetEventName(t *testing.T) {
	tests := []struct {
		et       constants.EventType
		expected string
	}{
		{constants.EventTypeHerb, "采集到草药"},
		{constants.EventTypeMilkTea, "捡到奶茶"},
		{constants.EventTypeThunder, "雷劫"},
		{constants.EventType("invalid"), "未知"},
	}

	for _, tt := range tests {
		result := GetEventName(tt.et)
		if result != tt.expected {
			t.Errorf("GetEventName(%s) = %s, expected %s", tt.et, result, tt.expected)
		}
	}
}
