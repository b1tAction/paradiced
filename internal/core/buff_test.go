package core

import (
	"testing"

	"github.com/b1tAction/Fated/pkg/event"
)

// ========== Evaluation Tests ==========

func TestEvaluationIsValid(t *testing.T) {
	// Valid evaluations
	validEvals := []Evaluation{0, 10, 25, 40, 50, 65, 70, 100}
	for _, e := range validEvals {
		if !e.IsValid() {
			t.Errorf("Evaluation(%d).IsValid() should be true", e)
		}
	}

	// Invalid evaluations
	invalidEvals := []Evaluation{-1, 101, 200}
	for _, e := range invalidEvals {
		if e.IsValid() {
			t.Errorf("Evaluation(%d).IsValid() should be false", e)
		}
	}
}

func TestEvaluationGetCategory(t *testing.T) {
	tests := []struct {
		eval     Evaluation
		expected string
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
		e1       Evaluation
		e2       Evaluation
		expected int
	}{
		{80, 60, 1},  // e1 is better
		{60, 80, -1}, // e1 is worse
		{50, 50, 0},  // equal
		{100, 0, 1},  // excellent vs very bad
		{0, 100, -1}, // very bad vs excellent
	}

	for _, tt := range tests {
		result := tt.e1.Compare(tt.e2)
		if result != tt.expected {
			t.Errorf("Evaluation(%d).Compare(%d) = %d, expected %d", tt.e1, tt.e2, result, tt.expected)
		}
	}
}

func TestEvaluationConstants(t *testing.T) {
	// Verify predefined constants
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
		{BuffTypeNone, "Unknown"}, // None is not registered, returns Unknown
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
		{BuffTypeCurse, EvaluationBad},
		{BuffTypeLost, EvaluationMildBad},
		{BuffTypeCorrupt, EvaluationBad},
		{BuffTypePoison, EvaluationVeryBad},
		{BuffTypeDivine, EvaluationVeryGood},
		{BuffTypeHidden, EvaluationNeutral},
		{BuffTypeRain, EvaluationGood},
		{BuffTypeExorcism, EvaluationMildGood},
		{BuffTypeFire, EvaluationGood},
	}

	for _, tt := range tests {
		result := GetBuffEvaluation(tt.bt)
		if result != tt.expected {
			t.Errorf("GetBuffEvaluation(%s) = %d, expected %d", tt.bt.String(), result, tt.expected)
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
	// With duration
	buff1 := NewBuff(BuffTypeCurse, 3)
	if !buff1.IsActive() {
		t.Error("buff with duration > 0 should be active")
	}

	// No duration
	buff2 := NewBuff(BuffTypeCurse, 0)
	if buff2.IsActive() {
		t.Error("buff with duration = 0 should not be active")
	}

	// Permanent buff (-1)
	buff3 := NewBuff(BuffTypeFire, -1)
	if !buff3.IsActive() {
		t.Error("permanent buff (duration=-1) should be active")
	}

	// With charge
	buff4 := NewBuff(BuffTypeFire, 0)
	buff4.Charge = 1
	if !buff4.IsActive() {
		t.Error("buff with charge > 0 should be active")
	}
}

func TestBuffTickDuration(t *testing.T) {
	buff := NewBuff(BuffTypeCurse, 3)

	// First tick
	if !buff.TickDuration() {
		t.Error("buff should still be active after first tick")
	}
	if buff.Duration != 2 {
		t.Errorf("buff.Duration = %d, expected 2", buff.Duration)
	}

	// Continue ticking until inactive
	buff.TickDuration()
	buff.TickDuration()
	if buff.IsActive() {
		t.Error("buff should not be active after all ticks")
	}

	// Permanent buff doesn't decrease
	permanentBuff := NewBuff(BuffTypeFire, -1)
	permanentBuff.TickDuration()
	if permanentBuff.Duration != -1 {
		t.Errorf("permanent buff duration should remain -1, got %d", permanentBuff.Duration)
	}
}

// ========== BuffDefinition Tests ==========

func TestBuffTypeGetBuffDefinition(t *testing.T) {
	// Test Curse Buff
	def := GetBuffDefinition(BuffTypeCurse)
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

	// Test Divine Buff
	def = GetBuffDefinition(BuffTypeDivine)
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

	// Test Hidden Buff (neutral evaluation)
	def = GetBuffDefinition(BuffTypeHidden)
	if def == nil {
		t.Fatal("BuffTypeHidden should have definition")
	}
	if def.Eval != EvaluationNeutral {
		t.Errorf("def.Eval = %d, expected Neutral(%d)", def.Eval, EvaluationNeutral)
	}
	if def.SpecialEffect != SpecialImmune {
		t.Errorf("def.SpecialEffect = %d, expected SpecialImmune", def.SpecialEffect)
	}

	// Test Fire Buff (permanent)
	def = GetBuffDefinition(BuffTypeFire)
	if def == nil {
		t.Fatal("BuffTypeFire should have definition")
	}
	if def.Duration != -1 {
		t.Errorf("def.Duration = %d, expected -1 (permanent)", def.Duration)
	}

	// Test unknown Buff
	def = GetBuffDefinition(BuffType(999))
	if def != nil {
		t.Error("unknown BuffType should return nil definition")
	}
}

// ========== Registry Tests ==========

func TestGlobalRegistryBuffTypes(t *testing.T) {
	allBuffs := GetAllBuffTypes()
	if len(allBuffs) != 9 {
		t.Errorf("AllBuffTypes count = %d, expected 9", len(allBuffs))
	}
}

func TestGetBuffTypesByEvaluationRange(t *testing.T) {
	// Get bad buffs (0~40)
	badBuffs := GlobalRegistry.GetBuffTypesByEvaluationRange(EvaluationMin, EvaluationBadThreshold)
	if len(badBuffs) != 4 {
		t.Errorf("bad buffs count = %d, expected 4", len(badBuffs))
	}

	// Get good buffs (66~100)
	goodBuffs := GlobalRegistry.GetBuffTypesByEvaluationRange(66, EvaluationMax)
	if len(goodBuffs) != 4 {
		t.Errorf("good buffs count = %d, expected 4", len(goodBuffs))
	}

	// Get excellent buffs (90~100)
	excellentBuffs := GlobalRegistry.GetBuffTypesByEvaluationRange(90, 100)
	if len(excellentBuffs) != 1 {
		t.Errorf("excellent buffs count = %d, expected 1 (Divine)", len(excellentBuffs))
	}
}

func TestGetBuffTypesByCategory(t *testing.T) {
	good := GetBuffTypesByCategory("Good")
	if len(good) != 4 {
		t.Errorf("good buffs count = %d, expected 4 (Divine, Rain, Exorcism, Fire)", len(good))
	}

	bad := GetBuffTypesByCategory("Bad")
	if len(bad) != 4 {
		t.Errorf("bad buffs count = %d, expected 4 (Curse, Lost, Corrupt, Poison)", len(bad))
	}

	neutral := GetBuffTypesByCategory("Neutral")
	if len(neutral) != 1 {
		t.Errorf("neutral buffs count = %d, expected 1 (Hidden)", len(neutral))
	}

	all := GetBuffTypesByCategory("Unknown")
	if len(all) != 9 {
		t.Errorf("unknown category should return all buffs, got %d", len(all))
	}
}

func TestGetAllBuffDefinitions(t *testing.T) {
	defs := GetAllBuffDefinitions()

	if len(defs) != 9 {
		t.Errorf("definitions count = %d, expected 9", len(defs))
	}

	// Verify each definition has valid Evaluation
	for _, def := range defs {
		if !def.Eval.IsValid() {
			t.Errorf("BuffDefinition for %s has invalid Evaluation %d", def.Name, def.Eval)
		}
	}
}

// ========== BuffDefinition Phase Tests ==========

func TestBuffDefinitionPhase(t *testing.T) {
	// Test Curse Buff Phases
	def := GetBuffDefinition(BuffTypeCurse)
	if !def.HasPhase(event.PhaseBeforeTurn) {
		t.Errorf("Curse should have BeforeTurn phase")
	}
	if len(def.Phases) != 1 {
		t.Errorf("Curse Phases count = %d, expected 1", len(def.Phases))
	}

	// Test Divine Buff Phases
	def = GetBuffDefinition(BuffTypeDivine)
	if !def.HasPhase(event.PhaseBeforeTurn) {
		t.Errorf("Divine should have BeforeTurn phase")
	}

	// Test Hidden Buff Phases (pre-damage immunity)
	def = GetBuffDefinition(BuffTypeHidden)
	if !def.HasPhase(event.PhasePreDamage) {
		t.Errorf("Hidden should have PreDamage phase")
	}

	// Test Lost Buff Phases (reverse during move)
	def = GetBuffDefinition(BuffTypeLost)
	if !def.HasPhase(event.PhaseOnMove) {
		t.Errorf("Lost should have OnMove phase")
	}

	// Test Corrupt Buff Phases (after turn)
	def = GetBuffDefinition(BuffTypeCorrupt)
	if !def.HasPhase(event.PhaseAfterTurn) {
		t.Errorf("Corrupt should have AfterTurn phase")
	}

	// Test Rain Buff Phases (after turn)
	def = GetBuffDefinition(BuffTypeRain)
	if !def.HasPhase(event.PhaseAfterTurn) {
		t.Errorf("Rain should have AfterTurn phase")
	}

	// Test Exorcism Buff Phases (pre-event)
	def = GetBuffDefinition(BuffTypeExorcism)
	if !def.HasPhase(event.PhasePreEvent) {
		t.Errorf("Exorcism should have PreEvent phase")
	}

	// Test Poison Buff Phases (before turn)
	def = GetBuffDefinition(BuffTypePoison)
	if !def.HasPhase(event.PhaseBeforeTurn) {
		t.Errorf("Poison should have BeforeTurn phase")
	}

	// Test Fire Buff Phases (before turn check)
	def = GetBuffDefinition(BuffTypeFire)
	if !def.HasPhase(event.PhaseBeforeTurn) {
		t.Errorf("Fire should have BeforeTurn phase")
	}
}

func TestBuffDefinitionPriority(t *testing.T) {
	// Test Hidden Buff priority (high priority, damage immunity)
	def := GetBuffDefinition(BuffTypeHidden)
	if def.Priority != 100 {
		t.Errorf("Hidden Priority = %d, expected 100 (highest)", def.Priority)
	}

	// Test Lost Buff priority (high priority)
	def = GetBuffDefinition(BuffTypeLost)
	if def.Priority != 100 {
		t.Errorf("Lost Priority = %d, expected 100", def.Priority)
	}

	// Test Exorcism Buff priority
	def = GetBuffDefinition(BuffTypeExorcism)
	if def.Priority != 80 {
		t.Errorf("Exorcism Priority = %d, expected 80", def.Priority)
	}

	// Test Divine/Curse Buff priority (standard)
	def = GetBuffDefinition(BuffTypeDivine)
	if def.Priority != 50 {
		t.Errorf("Divine Priority = %d, expected 50", def.Priority)
	}

	def = GetBuffDefinition(BuffTypeCurse)
	if def.Priority != 50 {
		t.Errorf("Curse Priority = %d, expected 50", def.Priority)
	}

	// Test Poison Buff priority (low priority)
	def = GetBuffDefinition(BuffTypePoison)
	if def.Priority != 30 {
		t.Errorf("Poison Priority = %d, expected 30 (lowest)", def.Priority)
	}
}

func TestBuffDefinitionNeedConfirm(t *testing.T) {
	// All Buffs default to not needing user confirmation
	for _, bt := range GetAllBuffTypes() {
		def := GetBuffDefinition(bt)
		if def == nil {
			continue
		}
		// Buff effects auto-execute by default, no confirmation needed
		if def.NeedConfirm {
			t.Errorf("Buff %s should not need confirm by default", bt.String())
		}
	}
}

func TestBuffDefinitionSpecialEffects(t *testing.T) {
	// Test special effect markers
	tests := []struct {
		bt       BuffType
		expected SpecialEffect
	}{
		{BuffTypeHidden, SpecialImmune},
		{BuffTypeLost, SpecialReverse},
		{BuffTypeExorcism, SpecialImmunePoison},
		{BuffTypePoison, SpecialBadEvent},
		{BuffTypeFire, SpecialZhuQuePassive},
	}

	for _, tt := range tests {
		def := GetBuffDefinition(tt.bt)
		if def == nil {
			t.Errorf("BuffType(%s) has no definition", tt.bt.String())
			continue
		}
		if def.SpecialEffect != tt.expected {
			t.Errorf("%s SpecialEffect = %d, expected %d", tt.bt.String(), def.SpecialEffect, tt.expected)
		}
	}
}

// ========== Multi Phase Support Tests ==========

func TestBuffDefinitionGetPhases(t *testing.T) {
	// Test GetPhases method returns correct Phase list
	def := GetBuffDefinition(BuffTypeCurse)
	phases := def.GetPhases()
	if len(phases) != 1 {
		t.Errorf("Curse GetPhases count = %d, expected 1", len(phases))
	}
	if phases[0] != event.PhaseBeforeTurn {
		t.Errorf("Curse GetPhases[0] = %s, expected BeforeTurn", phases[0].String())
	}
}

func TestBuffDefinitionHasPhase(t *testing.T) {
	// Test HasPhase method
	tests := []struct {
		bt       BuffType
		phase    event.Phase
		expected bool
	}{
		{BuffTypeCurse, event.PhaseBeforeTurn, true},
		{BuffTypeCurse, event.PhaseAfterTurn, false},
		{BuffTypeHidden, event.PhasePreDamage, true},
		{BuffTypeHidden, event.PhaseBeforeTurn, false},
		{BuffTypeLost, event.PhaseOnMove, true},
		{BuffTypeLost, event.PhaseOnLand, false},
	}

	for _, tt := range tests {
		def := GetBuffDefinition(tt.bt)
		if def == nil {
			t.Errorf("BuffType(%s) has no definition", tt.bt.String())
			continue
		}
		result := def.HasPhase(tt.phase)
		if result != tt.expected {
			t.Errorf("%s.HasPhase(%s) = %v, expected %v",
				tt.bt.String(), tt.phase.String(), result, tt.expected)
		}
	}
}

func TestBuffInstanceSubscriptionIDs(t *testing.T) {
	// Test Buff instance SubscriptionIDs slice
	buff := NewBuff(BuffTypeCurse, 3)

	// Initially should be empty slice
	if buff.SubscriptionIDs == nil {
		buff.SubscriptionIDs = make([]string, 0)
	}
	if len(buff.SubscriptionIDs) != 0 {
		t.Errorf("New Buff SubscriptionIDs count = %d, expected 0", len(buff.SubscriptionIDs))
	}

	// Can add multiple subscription IDs
	buff.SubscriptionIDs = append(buff.SubscriptionIDs, "sub-001", "sub-002")
	if len(buff.SubscriptionIDs) != 2 {
		t.Errorf("Buff SubscriptionIDs count = %d, expected 2", len(buff.SubscriptionIDs))
	}
}

func TestBuffDefinitionPhasesSlice(t *testing.T) {
	// Test Phases is slice type
	for _, bt := range GetAllBuffTypes() {
		def := GetBuffDefinition(bt)
		if def == nil {
			continue
		}
		// Phases should be slice, at least one element
		if len(def.Phases) == 0 {
			t.Errorf("Buff %s should have at least one Phase", bt.String())
		}
	}
}

func TestHasBuffHandler(t *testing.T) {
	// Fire buff has custom handler
	if !HasBuffHandler(BuffTypeFire) {
		t.Error("BuffTypeFire should have custom handler")
	}

	// Curse buff has no custom handler (uses default)
	if HasBuffHandler(BuffTypeCurse) {
		t.Error("BuffTypeCurse should not have custom handler")
	}
}