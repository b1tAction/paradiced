package rng

import (
	"math/rand"
	"testing"

	"github.com/b1tAction/Fated/internal/core/buff"
	"github.com/b1tAction/Fated/internal/core/event"
	"github.com/b1tAction/Fated/internal/core/item"
	"github.com/b1tAction/Fated/pkg/constants"
)

// ========== PoolType Tests ==========

func TestPoolTypeString(t *testing.T) {
	tests := []struct {
		pt       PoolType
		expected string
	}{
		{PoolTypeGood, "Good"},
		{PoolTypeNeutral, "Neutral"},
		{PoolTypeBad, "Bad"},
		{PoolType(99), "Unknown"},
	}

	for _, test := range tests {
		result := test.pt.String()
		if result != test.expected {
			t.Errorf("PoolType(%d).String() = %s, expected %s", test.pt, result, test.expected)
		}
	}
}

func TestPoolTypeToCategoryString(t *testing.T) {
	tests := []struct {
		pt       PoolType
		expected string
	}{
		{PoolTypeGood, "Good"},
		{PoolTypeNeutral, "Neutral"},
		{PoolTypeBad, "Bad"},
	}

	for _, test := range tests {
		result := test.pt.ToCategoryString()
		if result != test.expected {
			t.Errorf("PoolType(%d).ToCategoryString() = %s, expected %s", test.pt, result, test.expected)
		}
	}
}

// ========== DrawEngine Tests ==========

func TestNewDrawEngine(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	engine := NewDrawEngine(rng)
	if engine == nil {
		t.Fatal("NewDrawEngine should not return nil")
	}
	if engine.rng == nil {
		t.Error("DrawEngine.rng should not be nil")
	}
}

func TestDrawEngineGetRNG(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	engine := NewDrawEngine(rng)
	if engine.GetRNG() != rng {
		t.Error("GetRNG should return the same rand.Rand instance")
	}
}

// ========== Weight Calculation Tests ==========

func TestCalculateGoodWeight(t *testing.T) {
	// LP=0, Evaluation=100: weight = 1.0 × (1 + 0) = 1.0
	weight := calculateGoodWeight(constants.EvaluationExcellent, 0)
	if weight < 0.95 || weight > 1.05 {
		t.Errorf("GoodWeight(Evaluation=100, LP=0) = %.2f, expected ~1.0", weight)
	}

	// LP=8, Evaluation=100: weight = 1.0 × (1 + 1.0×0.8) = 1.8
	weight = calculateGoodWeight(constants.EvaluationExcellent, 8)
	if weight < 1.75 || weight > 1.85 {
		t.Errorf("GoodWeight(Evaluation=100, LP=8) = %.2f, expected ~1.8", weight)
	}

	// LP=8, Evaluation=70: weight = 0.7 × (1 + 0.7×0.8) = 0.7 × 1.56 = 1.092
	weight = calculateGoodWeight(constants.EvaluationMildGood, 8)
	if weight < 1.0 || weight > 1.2 {
		t.Errorf("GoodWeight(Evaluation=70, LP=8) = %.2f, expected ~1.09", weight)
	}

	// LP=0, Evaluation=70: weight = 0.7 × (1 + 0) = 0.7
	weight = calculateGoodWeight(constants.EvaluationMildGood, 0)
	if weight < 0.65 || weight > 0.75 {
		t.Errorf("GoodWeight(Evaluation=70, LP=0) = %.2f, expected ~0.7", weight)
	}

	// Verify LP bonus scales with Evaluation
	// LP=8 should give more bonus to Evaluation=100 than Evaluation=70
	bonus100 := calculateGoodWeight(constants.EvaluationExcellent, 8) - calculateGoodWeight(constants.EvaluationExcellent, 0)
	bonus70 := calculateGoodWeight(constants.EvaluationMildGood, 8) - calculateGoodWeight(constants.EvaluationMildGood, 0)
	if bonus100 <= bonus70 {
		t.Errorf("LP bonus for Evaluation=100 (%.2f) should be higher than for Evaluation=70 (%.2f)", bonus100, bonus70)
	}
}

func TestCalculateBadWeight(t *testing.T) {
	// LP=0, Evaluation=10: weight = 0.25 × (1 + 0) = 0.25
	weight := calculateBadWeight(constants.EvaluationVeryBad, 0)
	if weight < 0.2 || weight > 0.3 {
		t.Errorf("BadWeight(Evaluation=10, LP=0) = %.2f, expected ~0.25", weight)
	}

	// LP=8, Evaluation=10: weight = 0.25 × (1 + 0.25×0.8) = 0.25 × 1.2 = 0.3
	weight = calculateBadWeight(constants.EvaluationVeryBad, 8)
	if weight < 0.25 || weight > 0.4 {
		t.Errorf("BadWeight(Evaluation=10, LP=8) = %.2f, expected ~0.3", weight)
	}

	// LP=0, Evaluation=35: weight = 0.875 × (1 + 0) = 0.875
	weight = calculateBadWeight(constants.EvaluationMildBad, 0)
	if weight < 0.8 || weight > 0.95 {
		t.Errorf("BadWeight(Evaluation=35, LP=0) = %.2f, expected ~0.875", weight)
	}

	// LP=8, Evaluation=35: weight = 0.875 × (1 + 0.875×0.8) = 0.875 × 1.7 = 1.4875
	weight = calculateBadWeight(constants.EvaluationMildBad, 8)
	if weight < 1.3 || weight > 1.7 {
		t.Errorf("BadWeight(Evaluation=35, LP=8) = %.2f, expected ~1.49", weight)
	}

	// Verify LP bonus scales with Evaluation (higher Eval = more bonus)
	bonus35 := calculateBadWeight(constants.EvaluationMildBad, 8) - calculateBadWeight(constants.EvaluationMildBad, 0)
	bonus10 := calculateBadWeight(constants.EvaluationVeryBad, 8) - calculateBadWeight(constants.EvaluationVeryBad, 0)
	if bonus35 <= bonus10 {
		t.Errorf("LP bonus for Evaluation=35 (%.2f) should be higher than for Evaluation=10 (%.2f)", bonus35, bonus10)
	}
}

func TestCalculateNeutralWeight(t *testing.T) {
	// Evaluation=50: weight = 0.5
	weight := calculateNeutralWeight(constants.EvaluationNeutral)
	if weight < 0.45 || weight > 0.55 {
		t.Errorf("NeutralWeight(Evaluation=50) = %.2f, expected ~0.5", weight)
	}

	// Evaluation=55: weight = 0.55
	weight = calculateNeutralWeight(constants.EvaluationMixed)
	if weight < 0.5 || weight > 0.6 {
		t.Errorf("NeutralWeight(Evaluation=55) = %.2f, expected ~0.55", weight)
	}
}

// ========== DrawEvent Tests ==========

func TestDrawEventGood(t *testing.T) {
	engine := NewDrawEngine(rand.New(rand.NewSource(42)))

	// Draw from Good pool
	et := engine.DrawEvent(PoolTypeGood, 0)
	if !et.IsValid() {
		t.Errorf("DrawEvent(Good, LP=0) returned invalid event type: %s", string(et))
	}

	// Verify it's actually a Good event
	eval := event.GetEventEvaluation(et)
	if !eval.IsGood() {
		t.Errorf("DrawEvent(Good) returned non-Good event: %s (Evaluation=%d)", string(et), eval)
	}
}

func TestDrawEventBad(t *testing.T) {
	engine := NewDrawEngine(rand.New(rand.NewSource(42)))

	// Draw from Bad pool
	et := engine.DrawEvent(PoolTypeBad, 0)
	if !et.IsValid() {
		t.Errorf("DrawEvent(Bad, LP=0) returned invalid event type: %s", string(et))
	}

	// Verify it's actually a Bad event
	eval := event.GetEventEvaluation(et)
	if !eval.IsBad() {
		t.Errorf("DrawEvent(Bad) returned non-Bad event: %s (Evaluation=%d)", string(et), eval)
	}
}

func TestDrawEventNeutral(t *testing.T) {
	engine := NewDrawEngine(rand.New(rand.NewSource(42)))

	// Draw from Neutral pool
	et := engine.DrawEvent(PoolTypeNeutral, 0)
	if !et.IsValid() {
		t.Errorf("DrawEvent(Neutral, LP=0) returned invalid event type: %s", string(et))
	}

	// Verify it's actually a Neutral event
	eval := event.GetEventEvaluation(et)
	if !eval.IsNeutral() {
		t.Errorf("DrawEvent(Neutral) returned non-Neutral event: %s (Evaluation=%d)", string(et), eval)
	}
}

func TestDrawEventLPInfluence(t *testing.T) {
	// Test that LP influences Good draws - higher LP should favor higher Evaluation
	iterations := 5000

	// LP=0 Good draws
	stats := make(map[constants.Evaluation]int)
	for i := 0; i < iterations; i++ {
		e := NewDrawEngine(rand.New(rand.NewSource(int64(i))))
		et := e.DrawEvent(PoolTypeGood, 0)
		eval := event.GetEventEvaluation(et)
		stats[eval]++
	}

	// LP=8 Good draws - should have higher average Evaluation
	statsHighLP := make(map[constants.Evaluation]int)
	for i := 0; i < iterations; i++ {
		e := NewDrawEngine(rand.New(rand.NewSource(int64(i + 100000)))) // Different seed offset
		et := e.DrawEvent(PoolTypeGood, 8)
		eval := event.GetEventEvaluation(et)
		statsHighLP[eval]++
	}

	// Verify LP=8 has higher average Evaluation
	avgEval0 := calculateAverageEvaluation(stats)
	avgEval8 := calculateAverageEvaluation(statsHighLP)

	if avgEval8 <= avgEval0 {
		t.Errorf("LP=8 average Evaluation (%.2f) should be higher than LP=0 (%.2f)", avgEval8, avgEval0)
	}
}

func TestDrawEventBadLPInfluence(t *testing.T) {
	// Test that LP influences Bad draws - higher LP should favor less bad outcomes
	iterations := 5000

	// LP=0 Bad draws
	stats := make(map[constants.Evaluation]int)
	for i := 0; i < iterations; i++ {
		e := NewDrawEngine(rand.New(rand.NewSource(int64(i))))
		et := e.DrawEvent(PoolTypeBad, 0)
		eval := event.GetEventEvaluation(et)
		stats[eval]++
	}

	// LP=8 Bad draws - should have higher average Evaluation (less bad)
	statsHighLP := make(map[constants.Evaluation]int)
	for i := 0; i < iterations; i++ {
		e := NewDrawEngine(rand.New(rand.NewSource(int64(i + 100000)))) // Different seed offset
		et := e.DrawEvent(PoolTypeBad, 8)
		eval := event.GetEventEvaluation(et)
		statsHighLP[eval]++
	}

	// Verify LP=8 has higher average Evaluation (less bad)
	avgEval0 := calculateAverageEvaluation(stats)
	avgEval8 := calculateAverageEvaluation(statsHighLP)

	if avgEval8 <= avgEval0 {
		t.Errorf("LP=8 average Evaluation (%.2f) should be higher (less bad) than LP=0 (%.2f)", avgEval8, avgEval0)
	}
}

// ========== DrawBuff Tests ==========

func TestDrawBuffGood(t *testing.T) {
	engine := NewDrawEngine(rand.New(rand.NewSource(42)))

	bt := engine.DrawBuff(PoolTypeGood, 0)
	if !bt.IsValid() {
		t.Errorf("DrawBuff(Good, LP=0) returned invalid buff type: %s", string(bt))
	}

	eval := buff.GetBuffEvaluation(bt)
	if !eval.IsGood() {
		t.Errorf("DrawBuff(Good) returned non-Good buff: %s (Evaluation=%d)", string(bt), eval)
	}
}

func TestDrawBuffBad(t *testing.T) {
	engine := NewDrawEngine(rand.New(rand.NewSource(42)))

	bt := engine.DrawBuff(PoolTypeBad, 0)
	if !bt.IsValid() {
		t.Errorf("DrawBuff(Bad, LP=0) returned invalid buff type: %s", string(bt))
	}

	eval := buff.GetBuffEvaluation(bt)
	if !eval.IsBad() {
		t.Errorf("DrawBuff(Bad) returned non-Bad buff: %s (Evaluation=%d)", string(bt), eval)
	}
}

func TestDrawBuffNeutral(t *testing.T) {
	engine := NewDrawEngine(rand.New(rand.NewSource(42)))

	bt := engine.DrawBuff(PoolTypeNeutral, 0)
	if !bt.IsValid() {
		t.Errorf("DrawBuff(Neutral, LP=0) returned invalid buff type: %s", string(bt))
	}

	eval := buff.GetBuffEvaluation(bt)
	if !eval.IsNeutral() {
		t.Errorf("DrawBuff(Neutral) returned non-Neutral buff: %s (Evaluation=%d)", string(bt), eval)
	}
}

// ========== DrawItem Tests ==========

func TestDrawItem(t *testing.T) {
	engine := NewDrawEngine(rand.New(rand.NewSource(42)))

	it := engine.DrawItem(0)
	if !it.IsValid() {
		t.Errorf("DrawItem(LP=0) returned invalid item type: %s", string(it))
	}

	it = engine.DrawItem(8)
	if !it.IsValid() {
		t.Errorf("DrawItem(LP=8) returned invalid item type: %s", string(it))
	}
}

func TestDrawItemLPInfluence(t *testing.T) {
	// Test that LP influences Item draws - higher LP should favor higher Evaluation items
	iterations := 5000

	// LP=0 Item draws
	stats := make(map[constants.Evaluation]int)
	for i := 0; i < iterations; i++ {
		e := NewDrawEngine(rand.New(rand.NewSource(int64(i))))
		it := e.DrawItem(0)
		eval := item.GetItemEvaluation(it)
		stats[eval]++
	}

	// LP=8 Item draws - should have higher average Evaluation
	statsHighLP := make(map[constants.Evaluation]int)
	for i := 0; i < iterations; i++ {
		e := NewDrawEngine(rand.New(rand.NewSource(int64(i + 100000)))) // Different seed offset
		it := e.DrawItem(8)
		eval := item.GetItemEvaluation(it)
		statsHighLP[eval]++
	}

	// Verify LP=8 has higher average Evaluation
	avgEval0 := calculateAverageEvaluation(stats)
	avgEval8 := calculateAverageEvaluation(statsHighLP)

	if avgEval8 <= avgEval0 {
		t.Errorf("LP=8 average Evaluation (%.2f) should be higher than LP=0 (%.2f)", avgEval8, avgEval0)
	}
}

// ========== Utility Method Tests ==========

func TestDrawEngineIntn(t *testing.T) {
	engine := NewDrawEngine(rand.New(rand.NewSource(42)))

	for i := 0; i < 100; i++ {
		n := engine.Intn(100)
		if n < 0 || n >= 100 {
			t.Errorf("Intn(100) = %d, should be in [0, 100)", n)
		}
	}
}

func TestDrawEngineFloat64(t *testing.T) {
	engine := NewDrawEngine(rand.New(rand.NewSource(42)))

	for i := 0; i < 100; i++ {
		f := engine.Float64()
		if f < 0.0 || f >= 1.0 {
			t.Errorf("Float64() = %.3f, should be in [0.0, 1.0)", f)
		}
	}
}

// ========== Edge Case Tests ==========

func TestDrawEngineLPClamp(t *testing.T) {
	engine := NewDrawEngine(rand.New(rand.NewSource(42)))

	// Negative LP should be clamped to 0
	et := engine.DrawEvent(PoolTypeGood, -5)
	if !et.IsValid() {
		t.Error("DrawEvent with negative LP should still return valid event")
	}

	// LP > 8 should be clamped to 8
	et = engine.DrawEvent(PoolTypeGood, 100)
	if !et.IsValid() {
		t.Error("DrawEvent with LP > 8 should still return valid event")
	}
}

func TestDrawEngineConsistency(t *testing.T) {
	// Same seed should produce same sequence
	seed := int64(12345)
	e1 := NewDrawEngine(rand.New(rand.NewSource(seed)))
	e2 := NewDrawEngine(rand.New(rand.NewSource(seed)))

	for i := 0; i < 10; i++ {
		et1 := e1.DrawEvent(PoolTypeGood, 4)
		et2 := e2.DrawEvent(PoolTypeGood, 4)
		if et1 != et2 {
			t.Errorf("Same seed should produce same draw sequence, iteration %d: %s vs %s", i, string(et1), string(et2))
		}
	}
}

// ========== Helper Functions ==========

func calculateAverageEvaluation(stats map[constants.Evaluation]int) float64 {
	total := 0
	count := 0
	for eval, cnt := range stats {
		total += int(eval) * cnt
		count += cnt
	}
	if count == 0 {
		return 0
	}
	return float64(total) / float64(count)
}