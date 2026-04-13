package core

import (
	"testing"
)

// ========== EventType Tests ==========

func TestEventTypeString(t *testing.T) {
	tests := []struct {
		et       EventType
		expected string
	}{
		{EventTypeHerb, "Herb"},
		{EventTypeMilkTea, "MilkTea"},
		{EventTypeMosquito, "Mosquito"},
		{EventTypeThunder, "Thunder"},
		{EventType(999), "Unknown"},
	}

	for _, tt := range tests {
		result := tt.et.String()
		if result != tt.expected {
			t.Errorf("EventType(%d).String() = %s, expected %s", tt.et, result, tt.expected)
		}
	}
}

func TestEventTypeIsValid(t *testing.T) {
	validEvents := []EventType{
		EventTypeHerb, EventTypeMilkTea, EventTypeRelic, EventTypeDivineBless,
		EventTypeExchange, EventTypeHiddenBuff, EventTypeTasteTest,
		EventTypeMosquito, EventTypeGhostHit, EventTypeDogPoop,
		EventTypeThief, EventTypeCurseBuddha, EventTypeLostWay, EventTypeThunder,
	}
	for _, et := range validEvents {
		if !et.IsValid() {
			t.Errorf("EventType(%d).IsValid() should be true", et)
		}
	}

	invalidEvents := []EventType{EventTypeNone, EventType(100)}
	for _, et := range invalidEvents {
		if et.IsValid() {
			t.Errorf("EventType(%d).IsValid() should be false", et)
		}
	}
}

func TestEventTypeGetEvaluation(t *testing.T) {
	// Good events
	goodTests := []struct {
		et       EventType
		expected Evaluation
	}{
		{EventTypeHerb, EvaluationMildGood},
		{EventTypeMilkTea, EvaluationGood},
		{EventTypeRelic, EvaluationVeryGood},
		{EventTypeDivineBless, EvaluationExcellent},
	}
	for _, tt := range goodTests {
		result := GetEventEvaluation(tt.et)
		if result != tt.expected {
			t.Errorf("GetEventEvaluation(%s) = %d, expected %d", tt.et.String(), result, tt.expected)
		}
		if !result.IsGood() {
			t.Errorf("EventType(%s) evaluation should be Good", tt.et.String())
		}
	}

	// Neutral events
	neutralTests := []struct {
		et       EventType
		expected Evaluation
	}{
		{EventTypeExchange, EvaluationNeutral},
		{EventTypeHiddenBuff, EvaluationGood},
		{EventTypeTasteTest, EvaluationMixed},
	}
	for _, tt := range neutralTests {
		result := GetEventEvaluation(tt.et)
		if result != tt.expected {
			t.Errorf("GetEventEvaluation(%s) = %d, expected %d", tt.et.String(), result, tt.expected)
		}
	}

	// Bad events
	badTests := []struct {
		et       EventType
		expected Evaluation
	}{
		{EventTypeMosquito, EvaluationMildBad},
		{EventTypeGhostHit, EvaluationMildBad},
		{EventTypeDogPoop, EvaluationMildBad},
		{EventTypeThief, EvaluationBad},
		{EventTypeCurseBuddha, EvaluationBad},
		{EventTypeLostWay, EvaluationMildBad},
		{EventTypeThunder, EvaluationVeryBad},
	}
	for _, tt := range badTests {
		result := GetEventEvaluation(tt.et)
		if result != tt.expected {
			t.Errorf("GetEventEvaluation(%s) = %d, expected %d", tt.et.String(), result, tt.expected)
		}
		if !result.IsBad() {
			t.Errorf("EventType(%s) evaluation should be Bad", tt.et.String())
		}
	}
}

func TestEventTypeGetEventDefinition(t *testing.T) {
	// Test good event definition
	def := GetEventDefinition(EventTypeHerb)
	if def == nil {
		t.Fatal("EventTypeHerb should have definition")
	}
	if def.Name != "采集到草药" {
		t.Errorf("def.Name = %s, expected 采集到草药", def.Name)
	}
	if def.Eval != EvaluationMildGood {
		t.Errorf("def.Eval = %d, expected MildGood(%d)", def.Eval, EvaluationMildGood)
	}
	if def.HPChange != 1 {
		t.Errorf("def.HPChange = %d, expected 1", def.HPChange)
	}

	// Test bad event definition
	def = GetEventDefinition(EventTypeThunder)
	if def == nil {
		t.Fatal("EventTypeThunder should have definition")
	}
	if def.Name != "雷劫" {
		t.Errorf("def.Name = %s, expected 雷劫", def.Name)
	}
	if def.Eval != EvaluationVeryBad {
		t.Errorf("def.Eval = %d, expected VeryBad(%d)", def.Eval, EvaluationVeryBad)
	}
	if def.HPChange != -999 {
		t.Errorf("def.HPChange = %d, expected -999 (HP归零)", def.HPChange)
	}

	// Test Buff-giving event
	def = GetEventDefinition(EventTypeDivineBless)
	if def == nil {
		t.Fatal("EventTypeDivineBless should have definition")
	}
	if def.BuffType != BuffTypeDivine {
		t.Errorf("def.BuffType = %d, expected Divine", def.BuffType)
	}

	// Test item-giving event
	def = GetEventDefinition(EventTypeRelic)
	if def == nil {
		t.Fatal("EventTypeRelic should have definition")
	}
	if def.SpecialEffect != SpecialDrawItem {
		t.Errorf("def.SpecialEffect = %d, expected SpecialDrawItem", def.SpecialEffect)
	}

	// Test unknown event
	def = GetEventDefinition(EventType(999))
	if def != nil {
		t.Error("unknown EventType should return nil definition")
	}
}

func TestEventDefinitionEvaluationConsistency(t *testing.T) {
	// Verify all event definitions have consistent Evaluation
	for _, et := range GetAllEventTypes() {
		def := GetEventDefinition(et)
		if def == nil {
			t.Errorf("EventType(%s) has no definition", et.String())
			continue
		}
		eval := GetEventEvaluation(et)
		if def.Eval != eval {
			t.Errorf("EventType(%s) definition.Eval(%d) != GetEventEvaluation(%d)", et.String(), def.Eval, eval)
		}
	}
}

// ========== Registry Tests ==========

func TestGlobalRegistryEventTypes(t *testing.T) {
	allEvents := GetAllEventTypes()
	if len(allEvents) != 14 {
		t.Errorf("AllEventTypes count = %d, expected 14", len(allEvents))
	}
}

func TestGetEventTypesByEvaluationRange(t *testing.T) {
	// Get excellent events (90~100)
	excellentEvents := GlobalRegistry.GetEventTypesByEvaluationRange(90, 100)
	if len(excellentEvents) != 2 {
		t.Errorf("excellent events count = %d, expected 2 (Relic+DivineBless)", len(excellentEvents))
	}

	// Get good events (66~100, includes HiddenBuff)
	goodEvents := GlobalRegistry.GetEventTypesByEvaluationRange(66, 100)
	if len(goodEvents) != 5 {
		t.Errorf("good events count = %d, expected 5 (Herb+MilkTea+Relic+DivineBless+HiddenBuff)", len(goodEvents))
	}

	// Get mild bad events (30~40)
	mildBadEvents := GlobalRegistry.GetEventTypesByEvaluationRange(30, 40)
	if len(mildBadEvents) != 4 {
		t.Errorf("mild bad events count = %d, expected 4 (Mosquito+GhostHit+DogPoop+LostWay)", len(mildBadEvents))
	}

	// Get very bad events (0~15)
	veryBadEvents := GlobalRegistry.GetEventTypesByEvaluationRange(0, 15)
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
		if eval <= EvaluationBadThreshold {
			bad++
		} else if eval <= EvaluationNeutralThreshold {
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