package item

import (
	"testing"

	"github.com/b1tAction/Fated/internal/core/buff"
	"github.com/b1tAction/Fated/internal/core/types"
	"github.com/b1tAction/Fated/pkg/event"
)

// ========== ItemType Tests ==========

func TestItemTypeString(t *testing.T) {
	tests := []struct {
		it       ItemType
		expected string
	}{
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
	validItems := []ItemType{
		ItemTypeReverseClock, ItemTypeAnyDoor, ItemTypeDiceSwap, ItemTypeDiceUpgrade,
	}
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
		expected types.Evaluation
	}{
		{ItemTypeReverseClock, types.EvaluationGood},
		{ItemTypeAnyDoor, types.EvaluationNeutral},
		{ItemTypeDiceSwap, types.EvaluationNeutral},
		{ItemTypeDiceUpgrade, types.EvaluationGood},
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
	it := NewItem(ItemTypeReverseClock, "test-id-001")
	if it.Type != ItemTypeReverseClock {
		t.Errorf("item.Type = %d, expected ReverseClock", it.Type)
	}
	if it.ID != "test-id-001" {
		t.Errorf("item.ID = %s, expected test-id-001", it.ID)
	}
	if !it.Usable {
		t.Error("new item should be usable")
	}
}

func TestGenerateItemID(t *testing.T) {
	id1 := GenerateItemID()
	id2 := GenerateItemID()

	// IDs should be different (timestamp-based)
	if id1 == id2 {
		t.Errorf("GenerateItemID() should produce unique IDs, got %s twice", id1)
	}

	// IDs should have correct prefix
	if len(id1) < 5 || id1[:5] != "item-" {
		t.Errorf("GenerateItemID() should produce IDs starting with 'item-', got %s", id1)
	}
}

// ========== ItemDefinition Tests ==========

func TestGetItemDefinition(t *testing.T) {
	// Test ReverseClock
	def := GetItemDefinition(ItemTypeReverseClock)
	if def == nil {
		t.Fatal("ItemTypeReverseClock should have definition")
	}
	if def.Name != "反方向的钟" {
		t.Errorf("def.Name = %s, expected 反方向的钟", def.Name)
	}
	if def.Eval != types.EvaluationGood {
		t.Errorf("def.Eval = %d, expected Good(%d)", def.Eval, types.EvaluationGood)
	}
	if def.TargetSelf {
		t.Error("ReverseClock should not target self")
	}
	if !def.TargetOther {
		t.Error("ReverseClock should target other")
	}
	if def.BuffType != buff.BuffTypeLost {
		t.Errorf("def.BuffType = %d, expected Lost", def.BuffType)
	}
	if def.SpecialEffect != types.SpecialGiveLost {
		t.Errorf("def.SpecialEffect = %d, expected SpecialGiveLost", def.SpecialEffect)
	}
	if def.NeedConfirm != true {
		t.Error("ReverseClock should need confirm")
	}

	// Test AnyDoor
	def = GetItemDefinition(ItemTypeAnyDoor)
	if def == nil {
		t.Fatal("ItemTypeAnyDoor should have definition")
	}
	if def.Name != "任意门" {
		t.Errorf("def.Name = %s, expected 任意门", def.Name)
	}
	if def.Range != 30 {
		t.Errorf("def.Range = %d, expected 30", def.Range)
	}

	// Test DiceUpgrade
	def = GetItemDefinition(ItemTypeDiceUpgrade)
	if def == nil {
		t.Fatal("ItemTypeDiceUpgrade should have definition")
	}
	if !def.TargetSelf {
		t.Error("DiceUpgrade should target self")
	}
	if def.TargetOther {
		t.Error("DiceUpgrade should not target other")
	}

	// Test unknown Item
	def = GetItemDefinition(ItemType(999))
	if def != nil {
		t.Error("unknown ItemType should return nil definition")
	}
}

// ========== Registry Tests ==========

func TestGlobalItemRegistryItemTypes(t *testing.T) {
	allItems := GetAllItemTypes()
	if len(allItems) != 4 {
		t.Errorf("AllItemTypes count = %d, expected 4", len(allItems))
	}
}

func TestGetItemTypesByEvaluationRange(t *testing.T) {
	// Get good items (66~100)
	goodItems := GlobalItemRegistry.GetItemTypesByEvaluationRange(66, 100)
	if len(goodItems) != 2 {
		t.Errorf("good items count = %d, expected 2 (ReverseClock, DiceUpgrade)", len(goodItems))
	}

	// Get neutral items (41~65)
	neutralItems := GlobalItemRegistry.GetItemTypesByEvaluationRange(41, 65)
	if len(neutralItems) != 2 {
		t.Errorf("neutral items count = %d, expected 2 (AnyDoor, DiceSwap)", len(neutralItems))
	}
}

func TestGetItemTypesByCategory(t *testing.T) {
	good := GetItemTypesByCategory("Good")
	if len(good) != 2 {
		t.Errorf("Good items count = %d, expected 2", len(good))
	}

	neutral := GetItemTypesByCategory("Neutral")
	if len(neutral) != 2 {
		t.Errorf("Neutral items count = %d, expected 2", len(neutral))
	}

	bad := GetItemTypesByCategory("Bad")
	if len(bad) != 0 {
		t.Errorf("Bad items count = %d, expected 0", len(bad))
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

	// Verify each definition has valid Evaluation
	for _, def := range defs {
		if !def.Eval.IsValid() {
			t.Errorf("ItemDefinition for %s has invalid Evaluation %d", def.Name, def.Eval)
		}
	}
}

func TestItemDefinitionPhase(t *testing.T) {
	tests := []struct {
		it       ItemType
		expected event.Phase
	}{
		{ItemTypeReverseClock, event.PhaseAnyTime},
		{ItemTypeAnyDoor, event.PhaseOnLand},
		{ItemTypeDiceSwap, event.PhaseAnyTime},
		{ItemTypeDiceUpgrade, event.PhaseBeforeTurn},
	}

	for _, tt := range tests {
		def := GetItemDefinition(tt.it)
		if def == nil {
			t.Errorf("ItemType(%s) has no definition", tt.it.String())
			continue
		}
		if def.Phase != tt.expected {
			t.Errorf("%s Phase = %s, expected %s", tt.it.String(), def.Phase.String(), tt.expected.String())
		}
	}
}

func TestItemDefinitionPriority(t *testing.T) {
	tests := []struct {
		it       ItemType
		expected int
	}{
		{ItemTypeReverseClock, 50},
		{ItemTypeAnyDoor, 60},
		{ItemTypeDiceSwap, 40},
		{ItemTypeDiceUpgrade, 70},
	}

	for _, tt := range tests {
		def := GetItemDefinition(tt.it)
		if def == nil {
			t.Errorf("ItemType(%s) has no definition", tt.it.String())
			continue
		}
		if def.Priority != tt.expected {
			t.Errorf("%s Priority = %d, expected %d", tt.it.String(), def.Priority, tt.expected)
		}
	}
}

func TestItemDefinitionNeedConfirm(t *testing.T) {
	// All items need confirmation by default
	for _, it := range GetAllItemTypes() {
		def := GetItemDefinition(it)
		if def == nil {
			continue
		}
		if !def.NeedConfirm {
			t.Errorf("Item %s should need confirm", it.String())
		}
	}
}

func TestItemDefinitionSpecialEffects(t *testing.T) {
	tests := []struct {
		it       ItemType
		expected types.SpecialEffect
	}{
		{ItemTypeReverseClock, types.SpecialGiveLost},
		{ItemTypeAnyDoor, types.SpecialTeleport},
		{ItemTypeDiceSwap, types.SpecialDiceSwap},
		{ItemTypeDiceUpgrade, types.SpecialDiceUpgrade},
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

func TestItemDefinitionTargetTypes(t *testing.T) {
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

func TestItemDefinitionBuffType(t *testing.T) {
	// Test BuffType granted by ReverseClock
	def := GetItemDefinition(ItemTypeReverseClock)
	if def == nil {
		t.Fatal("ItemTypeReverseClock should have definition")
	}
	if def.BuffType != buff.BuffTypeLost {
		t.Errorf("ReverseClock BuffType = %d, expected Lost", def.BuffType)
	}

	// Other items don't grant Buff by default
	def = GetItemDefinition(ItemTypeAnyDoor)
	if def == nil {
		t.Fatal("ItemTypeAnyDoor should have definition")
	}
	if def.BuffType != buff.BuffTypeNone {
		t.Errorf("AnyDoor BuffType = %d, expected None", def.BuffType)
	}
}