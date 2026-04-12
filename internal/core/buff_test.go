package core

import (
	"testing"

	"github.com/b1tAction/Fated/pkg/event"
)

// ========== Evaluation Tests ==========

func TestEvaluationIsValid(t *testing.T) {
	// 有效评分
	validEvals := []Evaluation{0, 10, 25, 40, 50, 65, 70, 100}
	for _, e := range validEvals {
		if !e.IsValid() {
			t.Errorf("Evaluation(%d).IsValid() should be true", e)
		}
	}

	// 无效评分
	invalidEvals := []Evaluation{-1, 101, 200}
	for _, e := range invalidEvals {
		if e.IsValid() {
			t.Errorf("Evaluation(%d).IsValid() should be false", e)
		}
	}
}

func TestEvaluationGetCategory(t *testing.T) {
	tests := []struct {
		eval      Evaluation
		expected  string
	}{
		{0, "Bad"},
		{10, "Bad"},
		{25, "Bad"},
		{40, "Bad"},
		{41, "Neutral"},
		{50, "Neutral"},
		{65, "Neutral"},
		{66, "Good"},
		{70, "Good"},
		{80, "Good"},
		{100, "Good"},
	}

	for _, tt := range tests {
		result := tt.eval.GetCategory()
		if result != tt.expected {
			t.Errorf("Evaluation(%d).GetCategory() = %s, expected %s", tt.eval, result, tt.expected)
		}
	}
}

func TestEvaluationIsGood(t *testing.T) {
	// Good: > 65
	goodEvals := []Evaluation{66, 70, 80, 90, 100}
	for _, e := range goodEvals {
		if !e.IsGood() {
			t.Errorf("Evaluation(%d).IsGood() should be true", e)
		}
	}

	// Not Good: ≤ 65
	notGoodEvals := []Evaluation{0, 40, 50, 65}
	for _, e := range notGoodEvals {
		if e.IsGood() {
			t.Errorf("Evaluation(%d).IsGood() should be false", e)
		}
	}
}

func TestEvaluationIsNeutral(t *testing.T) {
	// Neutral: 41~65
	neutralEvals := []Evaluation{41, 50, 55, 65}
	for _, e := range neutralEvals {
		if !e.IsNeutral() {
			t.Errorf("Evaluation(%d).IsNeutral() should be true", e)
		}
	}

	// Not Neutral
	notNeutralEvals := []Evaluation{0, 40, 66, 100}
	for _, e := range notNeutralEvals {
		if e.IsNeutral() {
			t.Errorf("Evaluation(%d).IsNeutral() should be false", e)
		}
	}
}

func TestEvaluationIsBad(t *testing.T) {
	// Bad: ≤ 40
	badEvals := []Evaluation{0, 10, 25, 40}
	for _, e := range badEvals {
		if !e.IsBad() {
			t.Errorf("Evaluation(%d).IsBad() should be true", e)
		}
	}

	// Not Bad: > 40
	notBadEvals := []Evaluation{41, 50, 66, 100}
	for _, e := range notBadEvals {
		if e.IsBad() {
			t.Errorf("Evaluation(%d).IsBad() should be false", e)
		}
	}
}

func TestEvaluationCompare(t *testing.T) {
	tests := []struct {
		e1        Evaluation
		e2        Evaluation
		expected  int
	}{
		{80, 60, 1},  // e1 更好
		{60, 80, -1}, // e1 更差
		{50, 50, 0},  // 相同
		{100, 0, 1},  // 极良 vs 极恶
		{0, 100, -1}, // 极恶 vs 极良
	}

	for _, tt := range tests {
		result := tt.e1.Compare(tt.e2)
		if result != tt.expected {
			t.Errorf("Evaluation(%d).Compare(%d) = %d, expected %d", tt.e1, tt.e2, result, tt.expected)
		}
	}
}

func TestEvaluationConstants(t *testing.T) {
	// 验证预定义常量
	if EvaluationVeryBad != 10 {
		t.Errorf("EvaluationVeryBad = %d, expected 10", EvaluationVeryBad)
	}
	if EvaluationBad != 25 {
		t.Errorf("EvaluationBad = %d, expected 25", EvaluationBad)
	}
	if EvaluationMildBad != 35 {
		t.Errorf("EvaluationMildBad = %d, expected 35", EvaluationMildBad)
	}
	if EvaluationNeutral != 50 {
		t.Errorf("EvaluationNeutral = %d, expected 50", EvaluationNeutral)
	}
	if EvaluationMixed != 55 {
		t.Errorf("EvaluationMixed = %d, expected 55", EvaluationMixed)
	}
	if EvaluationMildGood != 70 {
		t.Errorf("EvaluationMildGood = %d, expected 70", EvaluationMildGood)
	}
	if EvaluationGood != 80 {
		t.Errorf("EvaluationGood = %d, expected 80", EvaluationGood)
	}
	if EvaluationVeryGood != 90 {
		t.Errorf("EvaluationVeryGood = %d, expected 90", EvaluationVeryGood)
	}
	if EvaluationExcellent != 100 {
		t.Errorf("EvaluationExcellent = %d, expected 100", EvaluationExcellent)
	}
}

// ========== BuffType Tests ==========

func TestBuffTypeString(t *testing.T) {
	tests := []struct {
		bt       BuffType
		expected string
	}{
		{BuffTypeNone, "None"},
		{BuffTypeCurse, "Curse"},
		{BuffTypeDivine, "Divine"},
		{BuffTypeHidden, "Hidden"},
		{BuffType(999), "Unknown"},
	}

	for _, tt := range tests {
		result := tt.bt.String()
		if result != tt.expected {
			t.Errorf("BuffType(%d).String() = %s, expected %s", tt.bt, result, tt.expected)
		}
	}
}

func TestBuffTypeIsPositive(t *testing.T) {
	positiveBuffs := []BuffType{BuffTypeDivine, BuffTypeHidden, BuffTypeRain, BuffTypeExorcism, BuffTypeFire}
	for _, bt := range positiveBuffs {
		if !bt.IsPositive() {
			t.Errorf("BuffType(%d).IsPositive() should be true", bt)
		}
	}

	negativeBuffs := []BuffType{BuffTypeCurse, BuffTypeLost, BuffTypeCorrupt, BuffTypePoison}
	for _, bt := range negativeBuffs {
		if bt.IsPositive() {
			t.Errorf("BuffType(%d).IsPositive() should be false", bt)
		}
	}
}

func TestBuffTypeGetEvaluation(t *testing.T) {
	tests := []struct {
		bt       BuffType
		expected Evaluation
	}{
		{BuffTypeCurse, EvaluationBad},      // 诅咒：较恶 (25)
		{BuffTypeLost, EvaluationMildBad},   // 迷途：轻恶 (35)
		{BuffTypeCorrupt, EvaluationBad},    // 腐化：较恶 (25)
		{BuffTypePoison, EvaluationVeryBad}, // 毒瘴：极恶 (10)
		{BuffTypeDivine, EvaluationVeryGood}, // 神眷：极良 (90)
		{BuffTypeHidden, EvaluationExcellent}, // 隐匿：最佳 (100)
		{BuffTypeRain, EvaluationGood},      // 甘霖：较良 (80)
		{BuffTypeExorcism, EvaluationMildGood}, // 辟邪：轻良 (70)
		{BuffTypeFire, EvaluationGood},      // 离火：较良 (80)
	}

	for _, tt := range tests {
		result := tt.bt.GetEvaluation()
		if result != tt.expected {
			t.Errorf("BuffType(%s).GetEvaluation() = %d, expected %d", tt.bt.String(), result, tt.expected)
		}
	}
}

// ========== Buff Instance Tests ==========

func TestNewBuff(t *testing.T) {
	buff := NewBuff(BuffTypeCurse, 3)
	if buff.Type != BuffTypeCurse {
		t.Errorf("buff.Type = %d, expected Curse", buff.Type)
	}
	if buff.Duration != 3 {
		t.Errorf("buff.Duration = %d, expected 3", buff.Duration)
	}
	if buff.Charge != 0 {
		t.Errorf("buff.Charge = %d, expected 0", buff.Charge)
	}
}

func TestBuffIsActive(t *testing.T) {
	// 有持续时间
	buff1 := NewBuff(BuffTypeCurse, 3)
	if !buff1.IsActive() {
		t.Error("buff with duration > 0 should be active")
	}

	// 无持续时间
	buff2 := NewBuff(BuffTypeCurse, 0)
	if buff2.IsActive() {
		t.Error("buff with duration = 0 should not be active")
	}

	// 永久 Buff (-1)
	buff3 := NewBuff(BuffTypeFire, -1)
	if !buff3.IsActive() {
		t.Error("permanent buff (duration=-1) should be active")
	}

	// 有充能
	buff4 := NewBuff(BuffTypeFire, 0)
	buff4.Charge = 1
	if !buff4.IsActive() {
		t.Error("buff with charge > 0 should be active")
	}
}

func TestBuffTickDuration(t *testing.T) {
	buff := NewBuff(BuffTypeCurse, 3)

	// 第一次 tick
	if !buff.TickDuration() {
		t.Error("buff should still be active after first tick")
	}
	if buff.Duration != 2 {
		t.Errorf("buff.Duration = %d, expected 2", buff.Duration)
	}

	// 继续 tick 直到失效
	buff.TickDuration()
	buff.TickDuration()
	if buff.IsActive() {
		t.Error("buff should not be active after all ticks")
	}

	// 永久 Buff 不减少
	permanentBuff := NewBuff(BuffTypeFire, -1)
	permanentBuff.TickDuration()
	if permanentBuff.Duration != -1 {
		t.Errorf("permanent buff duration should remain -1, got %d", permanentBuff.Duration)
	}
}

// ========== BuffDefinition Tests ==========

func TestBuffTypeGetBuffDefinition(t *testing.T) {
	// 测试诅咒 Buff
	def := BuffTypeCurse.GetBuffDefinition()
	if def == nil {
		t.Fatal("BuffTypeCurse should have definition")
	}
	if def.Name != "诅咒" {
		t.Errorf("def.Name = %s, expected 诅咒", def.Name)
	}
	if def.Eval != EvaluationBad {
		t.Errorf("def.Eval = %d, expected Bad(%d)", def.Eval, EvaluationBad)
	}
	if def.Duration != 3 {
		t.Errorf("def.Duration = %d, expected 3", def.Duration)
	}
	if def.LPPerTurn != -1 {
		t.Errorf("def.LPPerTurn = %d, expected -1", def.LPPerTurn)
	}

	// 测试神眷 Buff
	def = BuffTypeDivine.GetBuffDefinition()
	if def == nil {
		t.Fatal("BuffTypeDivine should have definition")
	}
	if def.Name != "神眷" {
		t.Errorf("def.Name = %s, expected 神眷", def.Name)
	}
	if def.Eval != EvaluationVeryGood {
		t.Errorf("def.Eval = %d, expected VeryGood(%d)", def.Eval, EvaluationVeryGood)
	}
	if def.LPPerTurn != 1 {
		t.Errorf("def.LPPerTurn = %d, expected 1", def.LPPerTurn)
	}

	// 测试隐匿 Buff（最佳评分）
	def = BuffTypeHidden.GetBuffDefinition()
	if def == nil {
		t.Fatal("BuffTypeHidden should have definition")
	}
	if def.Eval != EvaluationExcellent {
		t.Errorf("def.Eval = %d, expected Excellent(%d)", def.Eval, EvaluationExcellent)
	}
	if def.Special != "immune" {
		t.Errorf("def.Special = %s, expected immune", def.Special)
	}

	// 测试离火 Buff（永久）
	def = BuffTypeFire.GetBuffDefinition()
	if def == nil {
		t.Fatal("BuffTypeFire should have definition")
	}
	if def.Duration != -1 {
		t.Errorf("def.Duration = %d, expected -1 (permanent)", def.Duration)
	}

	// 测试未知 Buff
	def = BuffType(999).GetBuffDefinition()
	if def != nil {
		t.Error("unknown BuffType should return nil definition")
	}
}

// ========== BuffRegistry Tests ==========

func TestNewBuffRegistry(t *testing.T) {
	registry := NewBuffRegistry()
	if registry == nil {
		t.Fatal("NewBuffRegistry should not return nil")
	}
	if len(registry.AllBuffs) != 9 {
		t.Errorf("AllBuffs count = %d, expected 9", len(registry.AllBuffs))
	}
}

func TestBuffRegistryGetBuffsByEvaluationRange(t *testing.T) {
	registry := NewBuffRegistry()

	// 获取恶性 Buff（0~40）
	badBuffs := registry.GetBuffsByEvaluationRange(EvaluationMin, EvaluationBadThreshold)
	if len(badBuffs) != 4 {
		t.Errorf("bad buffs count = %d, expected 4", len(badBuffs))
	}

	// 获取良性 Buff（66~100）
	goodBuffs := registry.GetBuffsByEvaluationRange(66, EvaluationMax)
	if len(goodBuffs) != 5 {
		t.Errorf("good buffs count = %d, expected 5", len(goodBuffs))
	}

	// 获取极良 Buff（90~100）
	excellentBuffs := registry.GetBuffsByEvaluationRange(90, 100)
	if len(excellentBuffs) != 2 {
		t.Errorf("excellent buffs count = %d, expected 2 (Divine+Hidden)", len(excellentBuffs))
	}
}

func TestBuffRegistryGetBuffsByCategory(t *testing.T) {
	registry := NewBuffRegistry()

	good := registry.GetBuffsByCategory("Good")
	if len(good) != 5 {
		t.Errorf("good buffs count = %d, expected 5", len(good))
	}

	bad := registry.GetBuffsByCategory("Bad")
	if len(bad) != 4 {
		t.Errorf("bad buffs count = %d, expected 4", len(bad))
	}

	all := registry.GetBuffsByCategory("Unknown")
	if len(all) != 9 {
		t.Errorf("unknown category should return all buffs, got %d", len(all))
	}
}

func TestBuffRegistryGetAllBuffDefinitions(t *testing.T) {
	registry := NewBuffRegistry()
	defs := registry.GetAllBuffDefinitions()

	if len(defs) != 9 {
		t.Errorf("definitions count = %d, expected 9", len(defs))
	}

	// 验证每个定义都有 Evaluation
	for _, def := range defs {
		if !def.Eval.IsValid() {
			t.Errorf("BuffDefinition for %s has invalid Evaluation %d", def.Name, def.Eval)
		}
	}
}

// ========== BuffDefinition Phase Tests ==========

func TestBuffDefinitionPhase(t *testing.T) {
	// 测试诅咒 Buff 的 Phase
	def := BuffTypeCurse.GetBuffDefinition()
	if def.Phase != event.PhaseBeforeTurn {
		t.Errorf("Curse Phase = %s, expected BeforeTurn", def.Phase.String())
	}

	// 测试神眷 Buff 的 Phase
	def = BuffTypeDivine.GetBuffDefinition()
	if def.Phase != event.PhaseBeforeTurn {
		t.Errorf("Divine Phase = %s, expected BeforeTurn", def.Phase.String())
	}

	// 测试隐匿 Buff 的 Phase（受伤前免疫）
	def = BuffTypeHidden.GetBuffDefinition()
	if def.Phase != event.PhasePreDamage {
		t.Errorf("Hidden Phase = %s, expected PreDamage", def.Phase.String())
	}

	// 测试迷途 Buff 的 Phase（移动时反向）
	def = BuffTypeLost.GetBuffDefinition()
	if def.Phase != event.PhaseOnMove {
		t.Errorf("Lost Phase = %s, expected OnMove", def.Phase.String())
	}

	// 测试腐化 Buff 的 Phase（回合结束后）
	def = BuffTypeCorrupt.GetBuffDefinition()
	if def.Phase != event.PhaseAfterTurn {
		t.Errorf("Corrupt Phase = %s, expected AfterTurn", def.Phase.String())
	}

	// 测试甘霖 Buff 的 Phase（回合结束后）
	def = BuffTypeRain.GetBuffDefinition()
	if def.Phase != event.PhaseAfterTurn {
		t.Errorf("Rain Phase = %s, expected AfterTurn", def.Phase.String())
	}

	// 测试辟邪 Buff 的 Phase（事件触发前）
	def = BuffTypeExorcism.GetBuffDefinition()
	if def.Phase != event.PhasePreEvent {
		t.Errorf("Exorcism Phase = %s, expected PreEvent", def.Phase.String())
	}

	// 测试毒瘴 Buff 的 Phase（回合开始前）
	def = BuffTypePoison.GetBuffDefinition()
	if def.Phase != event.PhaseBeforeTurn {
		t.Errorf("Poison Phase = %s, expected BeforeTurn", def.Phase.String())
	}

	// 测试离火 Buff 的 Phase（回合开始前检查）
	def = BuffTypeFire.GetBuffDefinition()
	if def.Phase != event.PhaseBeforeTurn {
		t.Errorf("Fire Phase = %s, expected BeforeTurn", def.Phase.String())
	}
}

func TestBuffDefinitionPriority(t *testing.T) {
	// 测试隐匿 Buff 的优先级（高优先级，免疫伤害）
	def := BuffTypeHidden.GetBuffDefinition()
	if def.Priority != 100 {
		t.Errorf("Hidden Priority = %d, expected 100 (highest)", def.Priority)
	}

	// 测试迷途 Buff 的优先级（高优先级）
	def = BuffTypeLost.GetBuffDefinition()
	if def.Priority != 100 {
		t.Errorf("Lost Priority = %d, expected 100", def.Priority)
	}

	// 测试辟邪 Buff 的优先级
	def = BuffTypeExorcism.GetBuffDefinition()
	if def.Priority != 80 {
		t.Errorf("Exorcism Priority = %d, expected 80", def.Priority)
	}

	// 测试神眷/诅咒 Buff 的优先级（标准）
	def = BuffTypeDivine.GetBuffDefinition()
	if def.Priority != 50 {
		t.Errorf("Divine Priority = %d, expected 50", def.Priority)
	}

	def = BuffTypeCurse.GetBuffDefinition()
	if def.Priority != 50 {
		t.Errorf("Curse Priority = %d, expected 50", def.Priority)
	}

	// 测试毒瘴 Buff 的优先级（低优先级）
	def = BuffTypePoison.GetBuffDefinition()
	if def.Priority != 30 {
		t.Errorf("Poison Priority = %d, expected 30 (lowest)", def.Priority)
	}
}

func TestBuffDefinitionNeedConfirm(t *testing.T) {
	// 所有 Buff 默认不需要用户确认
	registry := NewBuffRegistry()
	for _, bt := range registry.AllBuffs {
		def := bt.GetBuffDefinition()
		if def == nil {
			continue
		}
		// Buff 效果默认自动执行，不需要确认
		if def.NeedConfirm {
			t.Errorf("Buff %s should not need confirm by default", bt.String())
		}
	}
}

func TestBuffDefinitionPhaseNeedsSubscription(t *testing.T) {
	// 测试 Phase 是否需要订阅 EventBus
	// 所有 Buff（除 AnyTime道具）都需要订阅
	tests := []struct {
		bt           BuffType
		needsSub     bool
		reason       string
	}{
		{BuffTypeCurse, true, "BeforeTurn 需要订阅"},
		{BuffTypeDivine, true, "BeforeTurn 需要订阅"},
		{BuffTypeHidden, true, "PreDamage 需要订阅"},
		{BuffTypeLost, true, "OnMove 需要订阅"},
		{BuffTypeCorrupt, true, "AfterTurn 需要订阅"},
		{BuffTypeRain, true, "AfterTurn 需要订阅"},
		{BuffTypeExorcism, true, "PreEvent 需要订阅"},
		{BuffTypePoison, true, "BeforeTurn 需要订阅"},
		{BuffTypeFire, true, "BeforeTurn 需要订阅（每4回合检查）"},
	}

	for _, tt := range tests {
		def := tt.bt.GetBuffDefinition()
		if def == nil {
			t.Errorf("BuffType(%s) has no definition", tt.bt.String())
			continue
		}
		if def.Phase.NeedsSubscription() != tt.needsSub {
			t.Errorf("%s: Phase.NeedsSubscription() = %v, expected %v (%s)",
				tt.bt.String(), def.Phase.NeedsSubscription(), tt.needsSub, tt.reason)
		}
	}
}

func TestBuffDefinitionDurationAndPhaseConsistency(t *testing.T) {
	// 验证 Duration 和 Phase 的合理性
	registry := NewBuffRegistry()
	for _, bt := range registry.AllBuffs {
		def := bt.GetBuffDefinition()
		if def == nil {
			continue
		}

		// 永久 Buff（Duration=-1）现在也使用 BeforeTurn
		// 离火在 BeforeTurn 时检查是否是第4回合
		if def.Duration == -1 {
			if bt == BuffTypeFire {
				if def.Phase != event.PhaseBeforeTurn {
					t.Errorf("Permanent buff %s should use BeforeTurn phase", bt.String())
				}
			}
		}

		// 有持续时间的 Buff 应该有有效的 Phase
		if def.Duration > 0 {
			if !def.Phase.IsValid() {
				t.Errorf("Buff %s with duration %d has invalid Phase", bt.String(), def.Duration)
			}
		}
	}
}

func TestBuffDefinitionSpecialEffects(t *testing.T) {
	// 测试特殊效果标记
	tests := []struct {
		bt          BuffType
		expected    string
	}{
		{BuffTypeHidden, "immune"},
		{BuffTypeLost, "reverse"},
		{BuffTypeExorcism, "immune_poison"},
		{BuffTypePoison, "bad_event_per_turn"},
		{BuffTypeFire, "zhuque_passive"},
	}

	for _, tt := range tests {
		def := tt.bt.GetBuffDefinition()
		if def == nil {
			t.Errorf("BuffType(%s) has no definition", tt.bt.String())
			continue
		}
		if def.Special != tt.expected {
			t.Errorf("%s Special = %s, expected %s", tt.bt.String(), def.Special, tt.expected)
		}
	}
}