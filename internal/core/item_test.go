package core

import (
	"testing"

	"github.com/b1tAction/Fated/pkg/event"
)

// ========== ItemType Tests ==========

func TestItemTypeString(t *testing.T) {
	tests := []struct {
		it       ItemType
		expected string
	}{
		{ItemTypeNone, "Unknown"}, // None is not registered, returns Unknown
		{ItemTypeReverseClock, "ReverseClock"},
		{ItemTypeAnyDoor, "AnyDoor"},
		{ItemTypeDiceSwap, "DiceSwap"},
		{ItemTypeDiceUpgrade, "DiceUpgrade"},
		{ItemType(999), "Unknown"},
	}

	for _, tt := range tests {
		result := tt.it.String()
		if result != tt.expected {
			t.Errorf("ItemType(%d).String() = %s, expected %s", tt.it, result, tt.expected)
		}
	}
}

func TestItemTypeIsValid(t *testing.T) {
	validItems := []ItemType{ItemTypeReverseClock, ItemTypeAnyDoor, ItemTypeDiceSwap, ItemTypeDiceUpgrade}
	for _, it := range validItems {
		if !it.IsValid() {
			t.Errorf("ItemType(%d).IsValid() should be true", it)
		}
	}

	invalidItems := []ItemType{ItemTypeNone, ItemType(100)}
	for _, it := range invalidItems {
		if it.IsValid() {
			t.Errorf("ItemType(%d).IsValid() should be false", it)
		}
	}
}

func TestItemTypeGetEvaluation(t *testing.T) {
	tests := []struct {
		it       ItemType
		expected Evaluation
	}{
		{ItemTypeReverseClock, EvaluationGood},
		{ItemTypeAnyDoor, EvaluationNeutral},
		{ItemTypeDiceSwap, EvaluationNeutral},
		{ItemTypeDiceUpgrade, EvaluationGood},
	}

	for _, tt := range tests {
		result := GetItemEvaluation(tt.it)
		if result != tt.expected {
			t.Errorf("GetItemEvaluation(%s) = %d, expected %d", tt.it.String(), result, tt.expected)
		}
	}
}

// ========== Item Instance Tests ==========

func TestNewItem(t *testing.T) {
	item := NewItem(ItemTypeReverseClock, "item-001")
	if item.Type != ItemTypeReverseClock {
		t.Errorf("item.Type = %d, expected ReverseClock", item.Type)
	}
	if item.ID != "item-001" {
		t.Errorf("item.ID = %s, expected item-001", item.ID)
	}
	if !item.Usable {
		t.Error("item should be usable initially")
	}
}

func TestItemFields(t *testing.T) {
	item := &Item{
		Type:     ItemTypeAnyDoor,
		ID:       "item-002",
		Usable:   true,
		TargetID: "player-123",
	}

	if item.Type != ItemTypeAnyDoor {
		t.Errorf("item.Type = %d, expected AnyDoor", item.Type)
	}
	if item.TargetID != "player-123" {
		t.Errorf("item.TargetID = %s, expected player-123", item.TargetID)
	}

	// Set unusable
	item.Usable = false
	if item.Usable {
		t.Error("item should be unusable after setting")
	}
}

// ========== ItemDefinition Tests ==========

func TestItemTypeGetItemDefinition(t *testing.T) {
	// Test ReverseClock
	def := GetItemDefinition(ItemTypeReverseClock)
	if def == nil {
		t.Fatal("ItemTypeReverseClock should have definition")
	}
	if def.Name != "反方向的钟" {
		t.Errorf("def.Name = %s, expected 反方向的钟", def.Name)
	}
	if def.Eval != EvaluationGood {
		t.Errorf("def.Eval = %d, expected Good(%d)", def.Eval, EvaluationGood)
	}
	if def.TargetSelf {
		t.Error("ReverseClock should not target self")
	}
	if !def.TargetOther {
		t.Error("ReverseClock should target other")
	}
	if def.BuffType != BuffTypeLost {
		t.Errorf("def.BuffType = %d, expected Lost", def.BuffType)
	}

	// Test AnyDoor
	def = GetItemDefinition(ItemTypeAnyDoor)
	if def == nil {
		t.Fatal("ItemTypeAnyDoor should have definition")
	}
	if def.Name != "任意门" {
		t.Errorf("def.Name = %s, expected 任意门", def.Name)
	}
	if def.Eval != EvaluationNeutral {
		t.Errorf("def.Eval = %d, expected Neutral(%d)", def.Eval, EvaluationNeutral)
	}
	if def.Range != 30 {
		t.Errorf("def.Range = %d, expected 30", def.Range)
	}

	// Test DiceUpgrade
	def = GetItemDefinition(ItemTypeDiceUpgrade)
	if def == nil {
		t.Fatal("ItemTypeDiceUpgrade should have definition")
	}
	if def.Name != "骰子升级卡" {
		t.Errorf("def.Name = %s, expected 骰子升级卡", def.Name)
	}
	if def.Eval != EvaluationGood {
		t.Errorf("def.Eval = %d, expected Good(%d)", def.Eval, EvaluationGood)
	}
	if !def.TargetSelf {
		t.Error("DiceUpgrade should target self")
	}
	if def.TargetOther {
		t.Error("DiceUpgrade should not target other")
	}

	// Test unknown item
	def = GetItemDefinition(ItemType(999))
	if def != nil {
		t.Error("unknown ItemType should return nil definition")
	}
}

func TestItemDefinitionEvaluation(t *testing.T) {
	// Verify all item definitions have valid Evaluation
	for _, it := range GetAllItemTypes() {
		def := GetItemDefinition(it)
		if def == nil {
			t.Errorf("ItemType(%s) has no definition", it.String())
			continue
		}
		if !def.Eval.IsValid() {
			t.Errorf("ItemType(%s) has invalid Evaluation %d", it.String(), def.Eval)
		}

		// Verify GetItemEvaluation matches definition Eval
		eval := GetItemEvaluation(it)
		if def.Eval != eval {
			t.Errorf("ItemType(%s) definition.Eval(%d) != GetItemEvaluation(%d)", it.String(), def.Eval, eval)
		}
	}
}

// ========== Registry Tests ==========

func TestGlobalRegistryItemTypes(t *testing.T) {
	allItems := GetAllItemTypes()
	if len(allItems) != 4 {
		t.Errorf("AllItemTypes count = %d, expected 4", len(allItems))
	}
}

func TestGetItemTypesByEvaluationRange(t *testing.T) {
	// Get bad items (0~40) - currently no bad items
	badItems := GlobalRegistry.GetItemTypesByEvaluationRange(EvaluationMin, EvaluationBadThreshold)
	if len(badItems) != 0 {
		t.Errorf("bad items count = %d, expected 0 (no bad items)", len(badItems))
	}

	// Get neutral items (41~65)
	neutralItems := GlobalRegistry.GetItemTypesByEvaluationRange(41, 65)
	if len(neutralItems) != 2 {
		t.Errorf("neutral items count = %d, expected 2 (AnyDoor+DiceSwap)", len(neutralItems))
	}

	// Get good items (66~100) - ReverseClock and DiceUpgrade
	goodItems := GlobalRegistry.GetItemTypesByEvaluationRange(66, EvaluationMax)
	if len(goodItems) != 2 {
		t.Errorf("good items count = %d, expected 2 (ReverseClock+DiceUpgrade)", len(goodItems))
	}
}

func TestGetItemTypesByCategory(t *testing.T) {
	good := GetItemTypesByCategory("Good")
	if len(good) != 2 {
		t.Errorf("good items count = %d, expected 2 (ReverseClock+DiceUpgrade)", len(good))
	}

	neutral := GetItemTypesByCategory("Neutral")
	if len(neutral) != 2 {
		t.Errorf("neutral items count = %d, expected 2", len(neutral))
	}

	bad := GetItemTypesByCategory("Bad")
	if len(bad) != 0 {
		t.Errorf("bad items count = %d, expected 0 (no bad items)", len(bad))
	}

	all := GetItemTypesByCategory("Unknown")
	if len(all) != 4 {
		t.Errorf("unknown category should return all items, got %d", len(all))
	}
}

func TestGetAllItemDefinitions(t *testing.T) {
	defs := GetAllItemDefinitions()

	if len(defs) != 4 {
		t.Errorf("definitions count = %d, expected 4", len(defs))
	}

	// Verify each definition
	defMap := make(map[ItemType]bool)
	for _, def := range defs {
		defMap[def.Type] = true
		if !def.Eval.IsValid() {
			t.Errorf("ItemDefinition for %s has invalid Evaluation %d", def.Name, def.Eval)
		}
	}

	// Verify all items have definitions
	for _, it := range GetAllItemTypes() {
		if !defMap[it] {
			t.Errorf("ItemType(%s) missing in definitions", it.String())
		}
	}
}

// ========== Evaluation Range Tests ==========

func TestItemEvaluationRanges(t *testing.T) {
	// Verify item evaluations are in reasonable ranges
	for _, it := range GetAllItemTypes() {
		eval := GetItemEvaluation(it)

		// ReverseClock is now good (beneficial to holder)
		if it == ItemTypeReverseClock {
			if !eval.IsGood() {
				t.Errorf("ItemTypeReverseClock should be good, got eval %d", eval)
			}
		}

		// AnyDoor and DiceSwap should be neutral
		if it == ItemTypeAnyDoor || it == ItemTypeDiceSwap {
			if !eval.IsNeutral() {
				t.Errorf("%s should be neutral, got eval %d", it.String(), eval)
			}
		}

		// DiceUpgrade should be good
		if it == ItemTypeDiceUpgrade {
			if !eval.IsGood() {
				t.Errorf("ItemTypeDiceUpgrade should be good, got eval %d", eval)
			}
		}
	}
}

// ========== ItemDefinition Phase Tests ==========

func TestItemDefinitionPhase(t *testing.T) {
	// Test ReverseClock Phase (AnyTime)
	def := GetItemDefinition(ItemTypeReverseClock)
	if def == nil {
		t.Fatal("ItemTypeReverseClock should have definition")
	}
	if def.Phase != event.PhaseAnyTime {
		t.Errorf("ReverseClock Phase = %s, expected AnyTime", def.Phase.String())
	}

	// Test AnyDoor Phase (OnLand)
	def = GetItemDefinition(ItemTypeAnyDoor)
	if def == nil {
		t.Fatal("ItemTypeAnyDoor should have definition")
	}
	if def.Phase != event.PhaseOnLand {
		t.Errorf("AnyDoor Phase = %s, expected OnLand", def.Phase.String())
	}

	// Test DiceSwap Phase (AnyTime)
	def = GetItemDefinition(ItemTypeDiceSwap)
	if def == nil {
		t.Fatal("ItemTypeDiceSwap should have definition")
	}
	if def.Phase != event.PhaseAnyTime {
		t.Errorf("DiceSwap Phase = %s, expected AnyTime", def.Phase.String())
	}

	// Test DiceUpgrade Phase (BeforeTurn)
	def = GetItemDefinition(ItemTypeDiceUpgrade)
	if def == nil {
		t.Fatal("ItemTypeDiceUpgrade should have definition")
	}
	if def.Phase != event.PhaseBeforeTurn {
		t.Errorf("DiceUpgrade Phase = %s, expected BeforeTurn", def.Phase.String())
	}
}

func TestItemDefinitionPriority(t *testing.T) {
	// Test DiceUpgrade priority (high)
	def := GetItemDefinition(ItemTypeDiceUpgrade)
	if def == nil {
		t.Fatal("ItemTypeDiceUpgrade should have definition")
	}
	if def.Priority != 70 {
		t.Errorf("DiceUpgrade Priority = %d, expected 70", def.Priority)
	}

	// Test AnyDoor priority
	def = GetItemDefinition(ItemTypeAnyDoor)
	if def == nil {
		t.Fatal("ItemTypeAnyDoor should have definition")
	}
	if def.Priority != 60 {
		t.Errorf("AnyDoor Priority = %d, expected 60", def.Priority)
	}

	// Test ReverseClock priority (standard)
	def = GetItemDefinition(ItemTypeReverseClock)
	if def == nil {
		t.Fatal("ItemTypeReverseClock should have definition")
	}
	if def.Priority != 50 {
		t.Errorf("ReverseClock Priority = %d, expected 50", def.Priority)
	}

	// Test DiceSwap priority (low)
	def = GetItemDefinition(ItemTypeDiceSwap)
	if def == nil {
		t.Fatal("ItemTypeDiceSwap should have definition")
	}
	if def.Priority != 40 {
		t.Errorf("DiceSwap Priority = %d, expected 40", def.Priority)
	}
}

func TestItemDefinitionNeedConfirm(t *testing.T) {
	// All items default to needing user confirmation
	for _, it := range GetAllItemTypes() {
		def := GetItemDefinition(it)
		if def == nil {
			continue
		}
		// Item usage needs user confirmation for target
		if !def.NeedConfirm {
			t.Errorf("Item %s should need confirm by default", it.String())
		}
	}
}

func TestItemDefinitionPhaseNeedsSubscription(t *testing.T) {
	// Test Phase needs EventBus subscription
	tests := []struct {
		it       ItemType
		needsSub bool
		reason   string
	}{
		{ItemTypeReverseClock, false, "AnyTime doesn't need subscription (manual trigger)"},
		{ItemTypeAnyDoor, true, "OnLand needs subscription"},
		{ItemTypeDiceSwap, false, "AnyTime doesn't need subscription (manual trigger)"},
		{ItemTypeDiceUpgrade, true, "BeforeTurn needs subscription"},
	}

	for _, tt := range tests {
		def := GetItemDefinition(tt.it)
		if def == nil {
			t.Errorf("ItemType(%s) has no definition", tt.it.String())
			continue
		}
		if def.Phase.NeedsSubscription() != tt.needsSub {
			t.Errorf("%s: Phase.NeedsSubscription() = %v, expected %v (%s)",
				tt.it.String(), def.Phase.NeedsSubscription(), tt.needsSub, tt.reason)
		}
	}
}

func TestItemDefinitionTargetTypes(t *testing.T) {
	// Test item target types
	tests := []struct {
		it          ItemType
		targetSelf  bool
		targetOther bool
	}{
		{ItemTypeReverseClock, false, true},
		{ItemTypeAnyDoor, false, true},
		{ItemTypeDiceSwap, false, true},
		{ItemTypeDiceUpgrade, true, false},
	}

	for _, tt := range tests {
		def := GetItemDefinition(tt.it)
		if def == nil {
			t.Errorf("ItemType(%s) has no definition", tt.it.String())
			continue
		}
		if def.TargetSelf != tt.targetSelf {
			t.Errorf("%s: TargetSelf = %v, expected %v", tt.it.String(), def.TargetSelf, tt.targetSelf)
		}
		if def.TargetOther != tt.targetOther {
			t.Errorf("%s: TargetOther = %v, expected %v", tt.it.String(), def.TargetOther, tt.targetOther)
		}
	}
}

func TestItemDefinitionRange(t *testing.T) {
	// Test AnyDoor effective range
	def := GetItemDefinition(ItemTypeAnyDoor)
	if def == nil {
		t.Fatal("ItemTypeAnyDoor should have definition")
	}
	if def.Range != 30 {
		t.Errorf("AnyDoor Range = %d, expected 30", def.Range)
	}

	// Other items default Range is 0
	def = GetItemDefinition(ItemTypeReverseClock)
	if def == nil {
		t.Fatal("ItemTypeReverseClock should have definition")
	}
	if def.Range != 0 {
		t.Errorf("ReverseClock Range = %d, expected 0", def.Range)
	}
}

func TestItemDefinitionBuffType(t *testing.T) {
	// Test BuffType granted by ReverseClock
	def := GetItemDefinition(ItemTypeReverseClock)
	if def == nil {
		t.Fatal("ItemTypeReverseClock should have definition")
	}
	if def.BuffType != BuffTypeLost {
		t.Errorf("ReverseClock BuffType = %d, expected Lost", def.BuffType)
	}

	// Other items don't grant Buff by default
	def = GetItemDefinition(ItemTypeAnyDoor)
	if def == nil {
		t.Fatal("ItemTypeAnyDoor should have definition")
	}
	if def.BuffType != BuffTypeNone {
		t.Errorf("AnyDoor BuffType = %d, expected None", def.BuffType)
	}
}

func TestItemDefinitionSpecialEffects(t *testing.T) {
	// Test special effects for items
	tests := []struct {
		it       ItemType
		expected SpecialEffect
	}{
		{ItemTypeReverseClock, SpecialGiveLost},
		{ItemTypeAnyDoor, SpecialTeleport},
		{ItemTypeDiceSwap, SpecialDiceSwap},
		{ItemTypeDiceUpgrade, SpecialDiceUpgrade},
	}

	for _, tt := range tests {
		def := GetItemDefinition(tt.it)
		if def == nil {
			t.Errorf("ItemType(%s) has no definition", tt.it.String())
			continue
		}
		if def.SpecialEffect != tt.expected {
			t.Errorf("%s SpecialEffect = %d, expected %d", tt.it.String(), def.SpecialEffect, tt.expected)
		}
	}
}