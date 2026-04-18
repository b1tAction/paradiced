package rng

import (
	"math/rand"
	"testing"

	"github.com/b1tAction/paradiced/pkg/constants"
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
	// LP=0, Evaluation=100: weight = 1.0 * (1 + 0) = 1.0
	weight := calculateGoodWeight(constants.EvaluationExcellent, 0)
	if weight < 0.95 || weight > 1.05 {
		t.Errorf("GoodWeight(Evaluation=100, LP=0) = %.2f, expected ~1.0", weight)
	}

	// LP=8, Evaluation=100: weight = 1.0 * (1 + 1.0*0.8) = 1.8
	weight = calculateGoodWeight(constants.EvaluationExcellent, 8)
	if weight < 1.75 || weight > 1.85 {
		t.Errorf("GoodWeight(Evaluation=100, LP=8) = %.2f, expected ~1.8", weight)
	}

	// LP=8, Evaluation=70: weight = 0.7 * (1 + 0.7*0.8) = 0.7 * 1.56 = 1.092
	weight = calculateGoodWeight(constants.EvaluationMildGood, 8)
	if weight < 1.0 || weight > 1.2 {
		t.Errorf("GoodWeight(Evaluation=70, LP=8) = %.2f, expected ~1.09", weight)
	}

	// LP=0, Evaluation=70: weight = 0.7 * (1 + 0) = 0.7
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
	// LP=0, Evaluation=10: weight = 0.25 * (1 + 0) = 0.25
	weight := calculateBadWeight(constants.EvaluationVeryBad, 0)
	if weight < 0.2 || weight > 0.3 {
		t.Errorf("BadWeight(Evaluation=10, LP=0) = %.2f, expected ~0.25", weight)
	}

	// LP=8, Evaluation=10: weight = 0.25 * (1 + 0.25*0.8) = 0.25 * 1.2 = 0.3
	weight = calculateBadWeight(constants.EvaluationVeryBad, 8)
	if weight < 0.25 || weight > 0.4 {
		t.Errorf("BadWeight(Evaluation=10, LP=8) = %.2f, expected ~0.3", weight)
	}

	// LP=0, Evaluation=35: weight = 0.875 * (1 + 0) = 0.875
	weight = calculateBadWeight(constants.EvaluationMildBad, 0)
	if weight < 0.8 || weight > 0.95 {
		t.Errorf("BadWeight(Evaluation=35, LP=0) = %.2f, expected ~0.875", weight)
	}

	// LP=8, Evaluation=35: weight = 0.875 * (1 + 0.875*0.8) = 0.875 * 1.7 = 1.4875
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

// ========== DrawFromPool Tests ==========

func TestDrawFromPoolGood(t *testing.T) {
	engine := NewDrawEngine(rand.New(rand.NewSource(42)))

	// Create a pool with Good items
	pool := &EvaluatedItemPool{
		Items: []EvaluatedItem{
			{Type: "event_divine_bless", Eval: constants.EvaluationExcellent},
			{Type: "event_herb", Eval: constants.EvaluationMildGood},
			{Type: "event_milk_tea", Eval: constants.EvaluationGood},
		},
	}

	// Draw from Good pool
	result := engine.DrawFromPool(pool, PoolTypeGood, 0)
	if result == "" {
		t.Error("DrawFromPool(Good, LP=0) should return non-empty result")
	}

	// Verify result is one of the pool items
	valid := false
	for _, item := range pool.Items {
		if result == item.Type {
			valid = true
			break
		}
	}
	if !valid {
		t.Errorf("DrawFromPool returned invalid result: %s", result)
	}
}

func TestDrawFromPoolBad(t *testing.T) {
	engine := NewDrawEngine(rand.New(rand.NewSource(42)))

	// Create a pool with Bad items
	pool := &EvaluatedItemPool{
		Items: []EvaluatedItem{
			{Type: "event_thunder", Eval: constants.EvaluationVeryBad},
			{Type: "event_mosquito", Eval: constants.EvaluationMildBad},
			{Type: "event_ghost_hit", Eval: constants.EvaluationMildBad},
		},
	}

	// Draw from Bad pool
	result := engine.DrawFromPool(pool, PoolTypeBad, 0)
	if result == "" {
		t.Error("DrawFromPool(Bad, LP=0) should return non-empty result")
	}

	// Verify result is one of the pool items
	valid := false
	for _, item := range pool.Items {
		if result == item.Type {
			valid = true
			break
		}
	}
	if !valid {
		t.Errorf("DrawFromPool returned invalid result: %s", result)
	}
}

func TestDrawFromPoolNeutral(t *testing.T) {
	engine := NewDrawEngine(rand.New(rand.NewSource(42)))

	// Create a pool with Neutral items
	pool := &EvaluatedItemPool{
		Items: []EvaluatedItem{
			{Type: "event_exchange", Eval: constants.EvaluationNeutral},
			{Type: "event_taste_test", Eval: constants.EvaluationMixed},
		},
	}

	// Draw from Neutral pool
	result := engine.DrawFromPool(pool, PoolTypeNeutral, 0)
	if result == "" {
		t.Error("DrawFromPool(Neutral, LP=0) should return non-empty result")
	}

	// Verify result is one of the pool items
	valid := false
	for _, item := range pool.Items {
		if result == item.Type {
			valid = true
			break
		}
	}
	if !valid {
		t.Errorf("DrawFromPool returned invalid result: %s", result)
	}
}

func TestDrawFromPoolLPInfluence(t *testing.T) {
	// Test that LP influences Good draws - higher LP should favor higher Evaluation
	iterations := 5000

	pool := &EvaluatedItemPool{
		Items: []EvaluatedItem{
			{Type: "event_divine_bless", Eval: constants.EvaluationExcellent},
			{Type: "event_herb", Eval: constants.EvaluationMildGood},
			{Type: "event_milk_tea", Eval: constants.EvaluationGood},
		},
	}

	// LP=0 Good draws
	stats := make(map[constants.Evaluation]int)
	for i := 0; i < iterations; i++ {
		e := NewDrawEngine(rand.New(rand.NewSource(int64(i))))
		result := e.DrawFromPool(pool, PoolTypeGood, 0)
		// Find the evaluation of the drawn item
		for _, item := range pool.Items {
			if result == item.Type {
				stats[item.Eval]++
				break
			}
		}
	}

	// LP=8 Good draws - should have higher average Evaluation
	statsHighLP := make(map[constants.Evaluation]int)
	for i := 0; i < iterations; i++ {
		e := NewDrawEngine(rand.New(rand.NewSource(int64(i + 100000))))
		result := e.DrawFromPool(pool, PoolTypeGood, 8)
		for _, item := range pool.Items {
			if result == item.Type {
				statsHighLP[item.Eval]++
				break
			}
		}
	}

	// Verify LP=8 has higher average Evaluation
	avgEval0 := calculateAverageEvaluation(stats)
	avgEval8 := calculateAverageEvaluation(statsHighLP)

	if avgEval8 <= avgEval0 {
		t.Errorf("LP=8 average Evaluation (%.2f) should be higher than LP=0 (%.2f)", avgEval8, avgEval0)
	}
}

func TestDrawFromPoolEmpty(t *testing.T) {
	engine := NewDrawEngine(rand.New(rand.NewSource(42)))

	// Empty pool
	pool := &EvaluatedItemPool{}

	result := engine.DrawFromPool(pool, PoolTypeGood, 0)
	if result != "" {
		t.Errorf("DrawFromPool with empty pool should return empty string, got %s", result)
	}

	// Nil pool
	result = engine.DrawFromPool(nil, PoolTypeGood, 0)
	if result != "" {
		t.Errorf("DrawFromPool with nil pool should return empty string, got %s", result)
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

func TestDrawFromPoolLPClamp(t *testing.T) {
	engine := NewDrawEngine(rand.New(rand.NewSource(42)))

	pool := &EvaluatedItemPool{
		Items: []EvaluatedItem{
			{Type: "event_herb", Eval: constants.EvaluationMildGood},
		},
	}

	// Negative LP should be clamped to 0
	result := engine.DrawFromPool(pool, PoolTypeGood, -5)
	if result != "event_herb" {
		t.Errorf("DrawFromPool with negative LP should still return valid result, got %s", result)
	}

	// LP > 8 should be clamped to 8
	result = engine.DrawFromPool(pool, PoolTypeGood, 100)
	if result != "event_herb" {
		t.Errorf("DrawFromPool with LP > 8 should still return valid result, got %s", result)
	}
}

func TestDrawFromPoolConsistency(t *testing.T) {
	// Same seed should produce same sequence
	seed := int64(12345)

	pool := &EvaluatedItemPool{
		Items: []EvaluatedItem{
			{Type: "event_a", Eval: constants.EvaluationExcellent},
			{Type: "event_b", Eval: constants.EvaluationGood},
			{Type: "event_c", Eval: constants.EvaluationMildGood},
		},
	}

	e1 := NewDrawEngine(rand.New(rand.NewSource(seed)))
	e2 := NewDrawEngine(rand.New(rand.NewSource(seed)))

	for i := 0; i < 10; i++ {
		r1 := e1.DrawFromPool(pool, PoolTypeGood, 4)
		r2 := e2.DrawFromPool(pool, PoolTypeGood, 4)
		if r1 != r2 {
			t.Errorf("Same seed should produce same draw sequence, iteration %d: %s vs %s", i, r1, r2)
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