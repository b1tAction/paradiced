package buff

import (
	"testing"

	"github.com/b1tAction/paradiced/pkg/constants"
)

// ========== BuffType Tests ==========

func TestBuffTypeIsValid(t *testing.T) {
	validBuffs := []constants.BuffType{
		constants.BuffTypeCurse, constants.BuffTypeLost, constants.BuffTypeCorrupt,
		constants.BuffTypePoison, constants.BuffTypeHidden, constants.BuffTypeDivine,
		constants.BuffTypeRain, constants.BuffTypeExorcism, constants.BuffTypeFire,
	}
	for _, bt := range validBuffs {
		if !bt.IsValid() {
			t.Errorf("BuffType(%s).IsValid() should be true", bt)
		}
	}

	invalidBuffs := []constants.BuffType{constants.BuffTypeNone, constants.BuffType(""), constants.BuffType("invalid")}
	for _, bt := range invalidBuffs {
		if bt.IsValid() {
			t.Errorf("BuffType(%s).IsValid() should be false", bt)
		}
	}
}

func TestBuffTypeIsPositive(t *testing.T) {
	positiveBuffs := []constants.BuffType{
		constants.BuffTypeDivine, constants.BuffTypeHidden, constants.BuffTypeRain,
		constants.BuffTypeExorcism, constants.BuffTypeFire,
	}
	for _, bt := range positiveBuffs {
		if !bt.IsPositive() {
			t.Errorf("BuffType(%s).IsPositive() should be true", bt)
		}
	}

	negativeBuffs := []constants.BuffType{
		constants.BuffTypeCurse, constants.BuffTypeLost, constants.BuffTypeCorrupt, constants.BuffTypePoison,
	}
	for _, bt := range negativeBuffs {
		if bt.IsPositive() {
			t.Errorf("BuffType(%s).IsPositive() should be false", bt)
		}
	}
}

func TestBuffTypeIsNegative(t *testing.T) {
	negativeBuffs := []constants.BuffType{
		constants.BuffTypeCurse, constants.BuffTypeLost, constants.BuffTypeCorrupt, constants.BuffTypePoison,
	}
	for _, bt := range negativeBuffs {
		if !bt.IsNegative() {
			t.Errorf("BuffType(%s).IsNegative() should be true", bt)
		}
	}

	positiveBuffs := []constants.BuffType{
		constants.BuffTypeDivine, constants.BuffTypeHidden, constants.BuffTypeRain,
	}
	for _, bt := range positiveBuffs {
		if bt.IsNegative() {
			t.Errorf("BuffType(%s).IsNegative() should be false", bt)
		}
	}
}

func TestBuffTypeGetEvaluation(t *testing.T) {
	tests := []struct {
		bt       constants.BuffType
		expected constants.Evaluation
	}{
		{constants.BuffTypeCurse, constants.EvaluationBad},
		{constants.BuffTypeLost, constants.EvaluationMildBad},
		{constants.BuffTypeCorrupt, constants.EvaluationBad},
		{constants.BuffTypePoison, constants.EvaluationVeryBad},
		{constants.BuffTypeDivine, constants.EvaluationVeryGood},
		{constants.BuffTypeHidden, constants.EvaluationNeutral},
		{constants.BuffTypeRain, constants.EvaluationGood},
		{constants.BuffTypeExorcism, constants.EvaluationMildGood},
		{constants.BuffTypeFire, constants.EvaluationGood},
	}

	for _, tt := range tests {
		result := GetBuffEvaluation(tt.bt)
		if result != tt.expected {
			t.Errorf("GetBuffEvaluation(%s) = %d, expected %d", tt.bt, result, tt.expected)
		}
	}
}

// ========== Buff Instance Tests ==========

func TestNewBuff(t *testing.T) {
	b := NewBuff(constants.BuffTypeCurse, 3)
	if b.Type != constants.BuffTypeCurse {
		t.Errorf("buff.Type = %s, expected Curse", b.Type)
	}
	if b.Duration != 3 {
		t.Errorf("buff.Duration = %d, expected 3", b.Duration)
	}
	if b.Charge != 0 {
		t.Errorf("buff.Charge = %d, expected 0", b.Charge)
	}
}

func TestBuffIsActive(t *testing.T) {
	// With duration
	b1 := NewBuff(constants.BuffTypeCurse, 3)
	if !b1.IsActive() {
		t.Error("buff with duration > 0 should be active")
	}

	// No duration
	b2 := NewBuff(constants.BuffTypeCurse, 0)
	if b2.IsActive() {
		t.Error("buff with duration = 0 should not be active")
	}

	// Permanent buff (-1)
	b3 := NewBuff(constants.BuffTypeFire, -1)
	if !b3.IsActive() {
		t.Error("permanent buff (duration=-1) should be active")
	}

	// With charge
	b4 := NewBuff(constants.BuffTypeFire, 0)
	b4.Charge = 1
	if !b4.IsActive() {
		t.Error("buff with charge > 0 should be active")
	}
}

func TestBuffTickDuration(t *testing.T) {
	b := NewBuff(constants.BuffTypeCurse, 3)

	// First tick
	if !b.TickDuration() {
		t.Error("buff should still be active after first tick")
	}
	if b.Duration != 2 {
		t.Errorf("buff.Duration = %d, expected 2", b.Duration)
	}

	// Continue ticking until inactive
	b.TickDuration()
	b.TickDuration()
	if b.IsActive() {
		t.Error("buff should not be active after all ticks")
	}

	// Permanent buff doesn't decrease
	permanentBuff := NewBuff(constants.BuffTypeFire, -1)
	permanentBuff.TickDuration()
	if permanentBuff.Duration != -1 {
		t.Errorf("permanent buff duration should remain -1, got %d", permanentBuff.Duration)
	}
}

// ========== BuffDefinition Tests ==========

func TestBuffTypeGetBuffDefinition(t *testing.T) {
	// Test Curse Buff
	def := GetBuffDefinition(constants.BuffTypeCurse)
	if def == nil {
		t.Fatal("BuffTypeCurse should have definition")
	}
	if def.Name != "诅咒" {
		t.Errorf("def.Name = %s, expected 诅咒", def.Name)
	}
	if def.Eval != constants.EvaluationBad {
		t.Errorf("def.Eval = %d, expected Bad(%d)", def.Eval, constants.EvaluationBad)
	}
	if def.Duration != 3 {
		t.Errorf("def.Duration = %d, expected 3", def.Duration)
	}
	if def.LPPerTurn != -1 {
		t.Errorf("def.LPPerTurn = %d, expected -1", def.LPPerTurn)
	}

	// Test Divine Buff
	def = GetBuffDefinition(constants.BuffTypeDivine)
	if def == nil {
		t.Fatal("BuffTypeDivine should have definition")
	}
	if def.Name != "神眷" {
		t.Errorf("def.Name = %s, expected 神眷", def.Name)
	}
	if def.Eval != constants.EvaluationVeryGood {
		t.Errorf("def.Eval = %d, expected VeryGood(%d)", def.Eval, constants.EvaluationVeryGood)
	}
	if def.LPPerTurn != 1 {
		t.Errorf("def.LPPerTurn = %d, expected 1", def.LPPerTurn)
	}

	// Test Hidden Buff (neutral evaluation)
	def = GetBuffDefinition(constants.BuffTypeHidden)
	if def == nil {
		t.Fatal("BuffTypeHidden should have definition")
	}
	if def.Eval != constants.EvaluationNeutral {
		t.Errorf("def.Eval = %d, expected Neutral(%d)", def.Eval, constants.EvaluationNeutral)
	}
	if def.SpecialEffect != constants.SpecialImmune {
		t.Errorf("def.SpecialEffect = %s, expected SpecialImmune", def.SpecialEffect)
	}

	// Test Fire Buff (permanent)
	def = GetBuffDefinition(constants.BuffTypeFire)
	if def == nil {
		t.Fatal("BuffTypeFire should have definition")
	}
	if def.Duration != -1 {
		t.Errorf("def.Duration = %d, expected -1 (permanent)", def.Duration)
	}

	// Test unknown Buff
	def = GetBuffDefinition(constants.BuffType("invalid"))
	if def != nil {
		t.Error("unknown BuffType should return nil definition")
	}
}

// ========== Registry Tests ==========

func TestGlobalBuffRegistryBuffTypes(t *testing.T) {
	allBuffs := GetAllBuffTypes()
	if len(allBuffs) != 9 {
		t.Errorf("AllBuffTypes count = %d, expected 9", len(allBuffs))
	}
}

func TestGetBuffTypesByEvaluationRange(t *testing.T) {
	// Get bad buffs (0~40)
	badBuffs := GlobalBuffRegistry.GetBuffTypesByEvaluationRange(constants.EvaluationMin, constants.EvaluationBadThreshold)
	if len(badBuffs) != 4 {
		t.Errorf("bad buffs count = %d, expected 4", len(badBuffs))
	}

	// Get good buffs (66~100)
	goodBuffs := GlobalBuffRegistry.GetBuffTypesByEvaluationRange(66, constants.EvaluationMax)
	if len(goodBuffs) != 4 {
		t.Errorf("good buffs count = %d, expected 4", len(goodBuffs))
	}

	// Get excellent buffs (90~100)
	excellentBuffs := GlobalBuffRegistry.GetBuffTypesByEvaluationRange(90, 100)
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
	def := GetBuffDefinition(constants.BuffTypeCurse)
	if !def.HasPhase(constants.PhaseBeforeTurn) {
		t.Errorf("Curse should have BeforeTurn phase")
	}
	if len(def.Phases) != 1 {
		t.Errorf("Curse Phases count = %d, expected 1", len(def.Phases))
	}

	// Test Divine Buff Phases
	def = GetBuffDefinition(constants.BuffTypeDivine)
	if !def.HasPhase(constants.PhaseBeforeTurn) {
		t.Errorf("Divine should have BeforeTurn phase")
	}

	// Test Hidden Buff Phases (pre-damage immunity)
	def = GetBuffDefinition(constants.BuffTypeHidden)
	if !def.HasPhase(constants.PhasePreDamage) {
		t.Errorf("Hidden should have PreDamage phase")
	}

	// Test Lost Buff Phases (reverse during move)
	def = GetBuffDefinition(constants.BuffTypeLost)
	if !def.HasPhase(constants.PhasePreMove) {
		t.Errorf("Lost should have PreMove phase")
	}

	// Test Corrupt Buff Phases (after turn)
	def = GetBuffDefinition(constants.BuffTypeCorrupt)
	if !def.HasPhase(constants.PhaseAfterTurn) {
		t.Errorf("Corrupt should have AfterTurn phase")
	}

	// Test Rain Buff Phases (after turn)
	def = GetBuffDefinition(constants.BuffTypeRain)
	if !def.HasPhase(constants.PhaseAfterTurn) {
		t.Errorf("Rain should have AfterTurn phase")
	}

	// Test Exorcism Buff Phases (pre-event)
	def = GetBuffDefinition(constants.BuffTypeExorcism)
	if !def.HasPhase(constants.PhasePreEvent) {
		t.Errorf("Exorcism should have PreEvent phase")
	}

	// Test Poison Buff Phases (before turn)
	def = GetBuffDefinition(constants.BuffTypePoison)
	if !def.HasPhase(constants.PhaseBeforeTurn) {
		t.Errorf("Poison should have BeforeTurn phase")
	}

	// Test Fire Buff Phases (before turn check)
	def = GetBuffDefinition(constants.BuffTypeFire)
	if !def.HasPhase(constants.PhaseBeforeTurn) {
		t.Errorf("Fire should have BeforeTurn phase")
	}
}

func TestBuffDefinitionPriority(t *testing.T) {
	// Test Hidden Buff priority (high priority, damage immunity)
	def := GetBuffDefinition(constants.BuffTypeHidden)
	if def.Priority != 100 {
		t.Errorf("Hidden Priority = %d, expected 100 (highest)", def.Priority)
	}

	// Test Lost Buff priority (high priority)
	def = GetBuffDefinition(constants.BuffTypeLost)
	if def.Priority != 100 {
		t.Errorf("Lost Priority = %d, expected 100", def.Priority)
	}

	// Test Exorcism Buff priority
	def = GetBuffDefinition(constants.BuffTypeExorcism)
	if def.Priority != 80 {
		t.Errorf("Exorcism Priority = %d, expected 80", def.Priority)
	}

	// Test Divine/Curse Buff priority (standard)
	def = GetBuffDefinition(constants.BuffTypeDivine)
	if def.Priority != 50 {
		t.Errorf("Divine Priority = %d, expected 50", def.Priority)
	}

	def = GetBuffDefinition(constants.BuffTypeCurse)
	if def.Priority != 50 {
		t.Errorf("Curse Priority = %d, expected 50", def.Priority)
	}

	// Test Poison Buff priority (low priority)
	def = GetBuffDefinition(constants.BuffTypePoison)
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
			t.Errorf("Buff %s should not need confirm by default", bt)
		}
	}
}

func TestBuffDefinitionSpecialEffects(t *testing.T) {
	// Test special effect markers
	tests := []struct {
		bt       constants.BuffType
		expected constants.SpecialEffect
	}{
		{constants.BuffTypeHidden, constants.SpecialImmune},
		{constants.BuffTypeLost, constants.SpecialReverse},
		{constants.BuffTypeExorcism, constants.SpecialImmunePoison},
		{constants.BuffTypePoison, constants.SpecialBadEvent},
		{constants.BuffTypeFire, constants.SpecialZhuQuePassive},
	}

	for _, tt := range tests {
		def := GetBuffDefinition(tt.bt)
		if def == nil {
			t.Errorf("BuffType(%s) has no definition", tt.bt)
			continue
		}
		if def.SpecialEffect != tt.expected {
			t.Errorf("%s SpecialEffect = %s, expected %s", tt.bt, def.SpecialEffect, tt.expected)
		}
	}
}

// ========== Multi Phase Support Tests ==========

func TestBuffDefinitionGetPhases(t *testing.T) {
	// Test GetPhases method returns correct Phase list
	def := GetBuffDefinition(constants.BuffTypeCurse)
	phases := def.GetPhases()
	if len(phases) != 1 {
		t.Errorf("Curse GetPhases count = %d, expected 1", len(phases))
	}
	if phases[0] != constants.PhaseBeforeTurn {
		t.Errorf("Curse GetPhases[0] = %s, expected BeforeTurn", phases[0])
	}
}

func TestBuffDefinitionHasPhase(t *testing.T) {
	// Test HasPhase method
	tests := []struct {
		bt       constants.BuffType
		phase    constants.Phase
		expected bool
	}{
		{constants.BuffTypeCurse, constants.PhaseBeforeTurn, true},
		{constants.BuffTypeCurse, constants.PhaseAfterTurn, false},
		{constants.BuffTypeHidden, constants.PhasePreDamage, true},
		{constants.BuffTypeHidden, constants.PhaseBeforeTurn, false},
		{constants.BuffTypeLost, constants.PhasePreMove, true},
		{constants.BuffTypeLost, constants.PhaseOnLand, false},
	}

	for _, tt := range tests {
		def := GetBuffDefinition(tt.bt)
		if def == nil {
			t.Errorf("BuffType(%s) has no definition", tt.bt)
			continue
		}
		result := def.HasPhase(tt.phase)
		if result != tt.expected {
			t.Errorf("%s.HasPhase(%s) = %v, expected %v", tt.bt, tt.phase, result, tt.expected)
		}
	}
}

func TestBuffInstanceSubscriptionIDs(t *testing.T) {
	// Test Buff instance SubscriptionIDs slice
	b := NewBuff(constants.BuffTypeCurse, 3)

	// Initially should be empty slice
	if b.SubscriptionIDs == nil {
		b.SubscriptionIDs = make([]string, 0)
	}
	if len(b.SubscriptionIDs) != 0 {
		t.Errorf("New Buff SubscriptionIDs count = %d, expected 0", len(b.SubscriptionIDs))
	}

	// Can add multiple subscription IDs
	b.SubscriptionIDs = append(b.SubscriptionIDs, "sub-001", "sub-002")
	if len(b.SubscriptionIDs) != 2 {
		t.Errorf("Buff SubscriptionIDs count = %d, expected 2", len(b.SubscriptionIDs))
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
			t.Errorf("Buff %s should have at least one Phase", bt)
		}
	}
}

func TestHasBuffHandler(t *testing.T) {
	// Fire buff has custom handler
	if !HasBuffHandler(constants.BuffTypeFire) {
		t.Error("BuffTypeFire should have custom handler")
	}

	// Curse buff has no custom handler (uses default)
	if HasBuffHandler(constants.BuffTypeCurse) {
		t.Error("BuffTypeCurse should not have custom handler")
	}
}

func TestGetBuffName(t *testing.T) {
	tests := []struct {
		bt       constants.BuffType
		expected string
	}{
		{constants.BuffTypeCurse, "诅咒"},
		{constants.BuffTypeDivine, "神眷"},
		{constants.BuffTypeHidden, "隐匿"},
		{constants.BuffTypeLost, "迷途"},
		{constants.BuffTypeRain, "甘霖"},
		{constants.BuffTypeExorcism, "辟邪"},
		{constants.BuffTypeFire, "离火"},
		{constants.BuffTypeCorrupt, "腐化"},
		{constants.BuffTypePoison, "毒瘴"},
		{constants.BuffType("invalid"), "未知"},
	}

	for _, tt := range tests {
		result := GetBuffName(tt.bt)
		if result != tt.expected {
			t.Errorf("GetBuffName(%s) = %s, expected %s", tt.bt, result, tt.expected)
		}
	}
}

func TestGetBuffString(t *testing.T) {
	// GetBuffString now returns the string directly (BuffType is already string)
	result := GetBuffString(constants.BuffTypeCurse)
	if result != "curse" {
		t.Errorf("GetBuffString(Curse) = %s, expected 'curse'", result)
	}
}
