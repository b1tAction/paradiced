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
	// 良性事件
	goodTests := []struct {
		et       EventType
		expected Evaluation
	}{
		{EventTypeHerb, EvaluationMildGood},    // 70
		{EventTypeMilkTea, EvaluationGood},     // 80
		{EventTypeRelic, EvaluationVeryGood},  // 90
		{EventTypeDivineBless, EvaluationExcellent}, // 100
	}
	for _, tt := range goodTests {
		result := tt.et.GetEvaluation()
		if result != tt.expected {
			t.Errorf("EventType(%s).GetEvaluation() = %d, expected %d", tt.et.String(), result, tt.expected)
		}
		if !result.IsGood() {
			t.Errorf("EventType(%s) evaluation should be Good", tt.et.String())
		}
	}

	// 中性事件
	neutralTests := []struct {
		et       EventType
		expected Evaluation
	}{
		{EventTypeExchange, EvaluationNeutral}, // 50
		{EventTypeHiddenBuff, EvaluationGood},   // 80 (虽然是中性分类，但效果正面)
		{EventTypeTasteTest, EvaluationMixed},  // 55
	}
	for _, tt := range neutralTests {
		result := tt.et.GetEvaluation()
		if result != tt.expected {
			t.Errorf("EventType(%s).GetEvaluation() = %d, expected %d", tt.et.String(), result, tt.expected)
		}
	}

	// 恶性事件
	badTests := []struct {
		et       EventType
		expected Evaluation
	}{
		{EventTypeMosquito, EvaluationMildBad},    // 35
		{EventTypeGhostHit, EvaluationMildBad},    // 35
		{EventTypeDogPoop, EvaluationMildBad},     // 35
		{EventTypeThief, EvaluationBad},           // 25
		{EventTypeCurseBuddha, EvaluationBad},     // 25
		{EventTypeLostWay, EvaluationMildBad},     // 35
		{EventTypeThunder, EvaluationVeryBad},     // 10
	}
	for _, tt := range badTests {
		result := tt.et.GetEvaluation()
		if result != tt.expected {
			t.Errorf("EventType(%s).GetEvaluation() = %d, expected %d", tt.et.String(), result, tt.expected)
		}
		if !result.IsBad() {
			t.Errorf("EventType(%s) evaluation should be Bad", tt.et.String())
		}
	}
}

func TestEventTypeGetCategory(t *testing.T) {
	tests := []struct {
		et       EventType
		expected string
	}{
		{EventTypeHerb, "Good"},
		{EventTypeMilkTea, "Good"},
		{EventTypeExchange, "Neutral"},
		{EventTypeMosquito, "Bad"},
		{EventTypeThunder, "Bad"},
	}

	for _, tt := range tests {
		result := tt.et.GetCategory()
		if result != tt.expected {
			t.Errorf("EventType(%s).GetCategory() = %s, expected %s", tt.et.String(), result, tt.expected)
		}
	}
}

func TestEventTypeGetEventDefinition(t *testing.T) {
	// 测试良性事件定义
	def := EventTypeHerb.GetEventDefinition()
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

	// 测试恶性事件定义
	def = EventTypeThunder.GetEventDefinition()
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

	// 测试Buff类事件
	def = EventTypeDivineBless.GetEventDefinition()
	if def == nil {
		t.Fatal("EventTypeDivineBless should have definition")
	}
	if def.BuffType != BuffTypeDivine {
		t.Errorf("def.BuffType = %d, expected Divine", def.BuffType)
	}

	// 测试道具类事件
	def = EventTypeRelic.GetEventDefinition()
	if def == nil {
		t.Fatal("EventTypeRelic should have definition")
	}
	if def.ItemAction != "draw" {
		t.Errorf("def.ItemAction = %s, expected draw", def.ItemAction)
	}

	// 测试未知事件
	def = EventType(999).GetEventDefinition()
	if def != nil {
		t.Error("unknown EventType should return nil definition")
	}
}

func TestEventDefinitionEvaluationConsistency(t *testing.T) {
	// 验证所有事件定义的 Eval 与 GetEvaluation 一致
	registry := NewEventRegistry()
	for _, et := range registry.AllEvents {
		def := et.GetEventDefinition()
		if def == nil {
			t.Errorf("EventType(%s) has no definition", et.String())
			continue
		}
		eval := et.GetEvaluation()
		if def.Eval != eval {
			t.Errorf("EventType(%s) definition.Eval(%d) != GetEvaluation(%d)", et.String(), def.Eval, eval)
		}
	}
}

// ========== EventRegistry Tests ==========

func TestNewEventRegistry(t *testing.T) {
	registry := NewEventRegistry()
	if registry == nil {
		t.Fatal("NewEventRegistry should not return nil")
	}
	if len(registry.AllEvents) != 14 {
		t.Errorf("AllEvents count = %d, expected 14", len(registry.AllEvents))
	}

	// 验证按 Evaluation 分类后的数量
	goodCount := len(registry.GoodEvents)
	neutralCount := len(registry.NeutralEvents)
	badCount := len(registry.BadEvents)

	if goodCount+neutralCount+badCount != 14 {
		t.Errorf("category sum = %d, expected 14", goodCount+neutralCount+badCount)
	}

	// 验证良性事件数量（5个：Herb, MilkTea, Relic, DivineBless, HiddenBuff）
	if goodCount != 5 {
		t.Errorf("GoodEvents count = %d, expected 5", goodCount)
	}

	// 验证恶性事件数量（7个）
	if badCount != 7 {
		t.Errorf("BadEvents count = %d, expected 7", badCount)
	}
}

func TestEventRegistryGetEventsByEvaluationRange(t *testing.T) {
	registry := NewEventRegistry()

	// 获取极良事件（90~100）
	excellentEvents := registry.GetEventsByEvaluationRange(90, 100)
	if len(excellentEvents) != 2 {
		t.Errorf("excellent events count = %d, expected 2 (Relic+DivineBless)", len(excellentEvents))
	}

	// 获取良性事件（66~100，包含 HiddenBuff）
	goodEvents := registry.GetEventsByEvaluationRange(66, 100)
	if len(goodEvents) != 5 {
		t.Errorf("good events count = %d, expected 5 (Herb+MilkTea+Relic+DivineBless+HiddenBuff)", len(goodEvents))
	}

	// 获取轻恶事件（30~40）
	mildBadEvents := registry.GetEventsByEvaluationRange(30, 40)
	if len(mildBadEvents) != 4 {
		t.Errorf("mild bad events count = %d, expected 4 (Mosquito+GhostHit+DogPoop+LostWay)", len(mildBadEvents))
	}

	// 获取极恶事件（0~15）
	veryBadEvents := registry.GetEventsByEvaluationRange(0, 15)
	if len(veryBadEvents) != 1 {
		t.Errorf("very bad events count = %d, expected 1 (Thunder)", len(veryBadEvents))
	}
}

func TestEventRegistryGetEventsByCategory(t *testing.T) {
	registry := NewEventRegistry()

	good := registry.GetEventsByCategory("Good")
	if len(good) != 5 {
		t.Errorf("Good events count = %d, expected 5", len(good))
	}

	neutral := registry.GetEventsByCategory("Neutral")
	// 只有 Exchange(50) 和 TasteTest(55) 是 Neutral
	if len(neutral) != 2 {
		t.Errorf("Neutral events count = %d, expected 2", len(neutral))
	}

	bad := registry.GetEventsByCategory("Bad")
	if len(bad) != 7 {
		t.Errorf("Bad events count = %d, expected 7", len(bad))
	}

	all := registry.GetEventsByCategory("Unknown")
	if len(all) != 14 {
		t.Errorf("unknown category should return all events, got %d", len(all))
	}
}

func TestEventRegistryGetAllEventDefinitions(t *testing.T) {
	registry := NewEventRegistry()
	defs := registry.GetAllEventDefinitions()

	if len(defs) != 14 {
		t.Errorf("definitions count = %d, expected 14", len(defs))
	}

	// 验证每个定义都有有效的 Evaluation
	for _, def := range defs {
		if !def.Eval.IsValid() {
			t.Errorf("EventDefinition for %s has invalid Evaluation %d", def.Name, def.Eval)
		}
	}
}

func TestEventRegistryEvaluationDistribution(t *testing.T) {
	registry := NewEventRegistry()

	// 统计各评分范围的事件数量
	var bad, neutral, good int
	for _, et := range registry.AllEvents {
		eval := et.GetEvaluation()
		if eval <= EvaluationBadThreshold {
			bad++
		} else if eval <= EvaluationNeutralThreshold {
			neutral++
		} else {
			good++
		}
	}

	// 验证分布
	// 良性：Herb(70), MilkTea(80), Relic(90), DivineBless(100) + HiddenBuff(80) = 5
	// 中性：Exchange(50), TasteTest(55) = 2
	// 恶性：7个
	t.Logf("Distribution: Bad=%d, Neutral=%d, Good=%d", bad, neutral, good)

	// 由于 HiddenBuff 被分类为 Good，所以 Registry 分类可能不同
	// 这里验证 Registry 的自动分类是正确的
	if len(registry.BadEvents) != bad {
		t.Errorf("BadEvents count mismatch: registry=%d, calculated=%d", len(registry.BadEvents), bad)
	}
	if len(registry.GoodEvents) != good {
		t.Errorf("GoodEvents count mismatch: registry=%d, calculated=%d", len(registry.GoodEvents), good)
	}
}