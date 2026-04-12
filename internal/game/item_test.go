package game

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
		{ItemTypeNone, "None"},
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
		{ItemTypeReverseClock, EvaluationBad},     // 反方向的钟：较恶 (25)
		{ItemTypeAnyDoor, EvaluationNeutral},      // 任意门：中性 (50)
		{ItemTypeDiceSwap, EvaluationNeutral},     // 骰子交换：中性 (50)
		{ItemTypeDiceUpgrade, EvaluationGood},     // 骰子升级：较良 (80)
	}

	for _, tt := range tests {
		result := tt.it.GetEvaluation()
		if result != tt.expected {
			t.Errorf("ItemType(%s).GetEvaluation() = %d, expected %d", tt.it.String(), result, tt.expected)
		}
	}
}

func TestItemTypeGetCategory(t *testing.T) {
	tests := []struct {
		it       ItemType
		expected string
	}{
		{ItemTypeReverseClock, "Bad"},
		{ItemTypeAnyDoor, "Neutral"},
		{ItemTypeDiceSwap, "Neutral"},
		{ItemTypeDiceUpgrade, "Good"},
	}

	for _, tt := range tests {
		result := tt.it.GetCategory()
		if result != tt.expected {
			t.Errorf("ItemType(%s).GetCategory() = %s, expected %s", tt.it.String(), result, tt.expected)
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

	// 设置不可用
	item.Usable = false
	if item.Usable {
		t.Error("item should be unusable after setting")
	}
}

// ========== ItemDefinition Tests ==========

func TestItemTypeGetItemDefinition(t *testing.T) {
	// 测试反方向的钟
	def := ItemTypeReverseClock.GetItemDefinition()
	if def == nil {
		t.Fatal("ItemTypeReverseClock should have definition")
	}
	if def.Name != "反方向的钟" {
		t.Errorf("def.Name = %s, expected 反方向的钟", def.Name)
	}
	if def.Eval != EvaluationBad {
		t.Errorf("def.Eval = %d, expected Bad(%d)", def.Eval, EvaluationBad)
	}
	if def.TargetSelf {
		t.Error("反方向的钟 should not target self")
	}
	if !def.TargetOther {
		t.Error("反方向的钟 should target other")
	}
	if def.BuffType != BuffTypeLost {
		t.Errorf("def.BuffType = %d, expected Lost", def.BuffType)
	}

	// 测试任意门
	def = ItemTypeAnyDoor.GetItemDefinition()
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

	// 测试骰子升级卡
	def = ItemTypeDiceUpgrade.GetItemDefinition()
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
		t.Error("骰子升级卡 should target self")
	}
	if def.TargetOther {
		t.Error("骰子升级卡 should not target other")
	}

	// 测试未知道具
	def = ItemType(999).GetItemDefinition()
	if def != nil {
		t.Error("unknown ItemType should return nil definition")
	}
}

func TestItemDefinitionEvaluation(t *testing.T) {
	// 验证所有道具定义都有有效的 Evaluation
	registry := NewItemRegistry()
	for _, it := range registry.AllItems {
		def := it.GetItemDefinition()
		if def == nil {
			t.Errorf("ItemType(%s) has no definition", it.String())
			continue
		}
		if !def.Eval.IsValid() {
			t.Errorf("ItemType(%s) has invalid Evaluation %d", it.String(), def.Eval)
		}

		// 验证 GetEvaluation 和定义中的 Eval 一致
		eval := it.GetEvaluation()
		if def.Eval != eval {
			t.Errorf("ItemType(%s) definition.Eval(%d) != GetEvaluation(%d)", it.String(), def.Eval, eval)
		}
	}
}

// ========== ItemRegistry Tests ==========

func TestNewItemRegistry(t *testing.T) {
	registry := NewItemRegistry()
	if registry == nil {
		t.Fatal("NewItemRegistry should not return nil")
	}
	if len(registry.AllItems) != 4 {
		t.Errorf("AllItems count = %d, expected 4", len(registry.AllItems))
	}
}

func TestItemRegistryGetItemsByEvaluationRange(t *testing.T) {
	registry := NewItemRegistry()

	// 获取恶性道具（0~40）
	badItems := registry.GetItemsByEvaluationRange(EvaluationMin, EvaluationBadThreshold)
	if len(badItems) != 1 {
		t.Errorf("bad items count = %d, expected 1 (ReverseClock)", len(badItems))
	}
	for _, it := range badItems {
		if it != ItemTypeReverseClock {
			t.Errorf("bad item should be ReverseClock, got %s", it.String())
		}
	}

	// 获取中性道具（41~65）
	neutralItems := registry.GetItemsByEvaluationRange(41, 65)
	if len(neutralItems) != 2 {
		t.Errorf("neutral items count = %d, expected 2 (AnyDoor+DiceSwap)", len(neutralItems))
	}

	// 获取良性道具（66~100）
	goodItems := registry.GetItemsByEvaluationRange(66, EvaluationMax)
	if len(goodItems) != 1 {
		t.Errorf("good items count = %d, expected 1 (DiceUpgrade)", len(goodItems))
	}
	for _, it := range goodItems {
		if it != ItemTypeDiceUpgrade {
			t.Errorf("good item should be DiceUpgrade, got %s", it.String())
		}
	}
}

func TestItemRegistryGetItemsByCategory(t *testing.T) {
	registry := NewItemRegistry()

	good := registry.GetItemsByCategory("Good")
	if len(good) != 1 {
		t.Errorf("good items count = %d, expected 1", len(good))
	}

	neutral := registry.GetItemsByCategory("Neutral")
	if len(neutral) != 2 {
		t.Errorf("neutral items count = %d, expected 2", len(neutral))
	}

	bad := registry.GetItemsByCategory("Bad")
	if len(bad) != 1 {
		t.Errorf("bad items count = %d, expected 1", len(bad))
	}

	all := registry.GetItemsByCategory("Unknown")
	if len(all) != 4 {
		t.Errorf("unknown category should return all items, got %d", len(all))
	}
}

func TestItemRegistryGetAllItemDefinitions(t *testing.T) {
	registry := NewItemRegistry()
	defs := registry.GetAllItemDefinitions()

	if len(defs) != 4 {
		t.Errorf("definitions count = %d, expected 4", len(defs))
	}

	// 验证每个定义
	defMap := make(map[ItemType]bool)
	for _, def := range defs {
		defMap[def.Type] = true
		if !def.Eval.IsValid() {
			t.Errorf("ItemDefinition for %s has invalid Evaluation %d", def.Name, def.Eval)
		}
	}

	// 验证所有道具都有定义
	for _, it := range registry.AllItems {
		if !defMap[it] {
			t.Errorf("ItemType(%s) missing in definitions", it.String())
		}
	}
}

// ========== Evaluation Range Tests ==========

func TestItemEvaluationRanges(t *testing.T) {
	// 验证道具评分在合理范围内
	registry := NewItemRegistry()

	for _, it := range registry.AllItems {
		eval := it.GetEvaluation()

		// 反方向的钟应该是恶性
		if it == ItemTypeReverseClock {
			if !eval.IsBad() {
				t.Errorf("ItemTypeReverseClock should be bad, got eval %d", eval)
			}
		}

		// 任意门和骰子交换应该是中性
		if it == ItemTypeAnyDoor || it == ItemTypeDiceSwap {
			if !eval.IsNeutral() {
				t.Errorf("%s should be neutral, got eval %d", it.String(), eval)
			}
		}

		// 骰子升级应该是良性
		if it == ItemTypeDiceUpgrade {
			if !eval.IsGood() {
				t.Errorf("ItemTypeDiceUpgrade should be good, got eval %d", eval)
			}
		}
	}
}

// ========== ItemDefinition Phase Tests ==========

func TestItemDefinitionPhase(t *testing.T) {
	// 测试反方向的钟的 Phase（任何时候可用）
	def := ItemTypeReverseClock.GetItemDefinition()
	if def == nil {
		t.Fatal("ItemTypeReverseClock should have definition")
	}
	if def.Phase != event.PhaseAnyTime {
		t.Errorf("ReverseClock Phase = %s, expected AnyTime", def.Phase.String())
	}

	// 测试任意门的 Phase（落地后可用）
	def = ItemTypeAnyDoor.GetItemDefinition()
	if def == nil {
		t.Fatal("ItemTypeAnyDoor should have definition")
	}
	if def.Phase != event.PhaseOnLand {
		t.Errorf("AnyDoor Phase = %s, expected OnLand", def.Phase.String())
	}

	// 测试骰子交换的 Phase（任何时候可用）
	def = ItemTypeDiceSwap.GetItemDefinition()
	if def == nil {
		t.Fatal("ItemTypeDiceSwap should have definition")
	}
	if def.Phase != event.PhaseAnyTime {
		t.Errorf("DiceSwap Phase = %s, expected AnyTime", def.Phase.String())
	}

	// 测试骰子升级卡的 Phase（回合开始前可用）
	def = ItemTypeDiceUpgrade.GetItemDefinition()
	if def == nil {
		t.Fatal("ItemTypeDiceUpgrade should have definition")
	}
	if def.Phase != event.PhaseBeforeTurn {
		t.Errorf("DiceUpgrade Phase = %s, expected BeforeTurn", def.Phase.String())
	}
}

func TestItemDefinitionPriority(t *testing.T) {
	// 测试骰子升级卡的优先级（较高）
	def := ItemTypeDiceUpgrade.GetItemDefinition()
	if def == nil {
		t.Fatal("ItemTypeDiceUpgrade should have definition")
	}
	if def.Priority != 70 {
		t.Errorf("DiceUpgrade Priority = %d, expected 70", def.Priority)
	}

	// 测试任意门的优先级
	def = ItemTypeAnyDoor.GetItemDefinition()
	if def == nil {
		t.Fatal("ItemTypeAnyDoor should have definition")
	}
	if def.Priority != 60 {
		t.Errorf("AnyDoor Priority = %d, expected 60", def.Priority)
	}

	// 测试反方向的钟的优先级（标准）
	def = ItemTypeReverseClock.GetItemDefinition()
	if def == nil {
		t.Fatal("ItemTypeReverseClock should have definition")
	}
	if def.Priority != 50 {
		t.Errorf("ReverseClock Priority = %d, expected 50", def.Priority)
	}

	// 测试骰子交换的优先级（较低）
	def = ItemTypeDiceSwap.GetItemDefinition()
	if def == nil {
		t.Fatal("ItemTypeDiceSwap should have definition")
	}
	if def.Priority != 40 {
		t.Errorf("DiceSwap Priority = %d, expected 40", def.Priority)
	}
}

func TestItemDefinitionNeedConfirm(t *testing.T) {
	// 所有道具默认需要用户确认
	registry := NewItemRegistry()
	for _, it := range registry.AllItems {
		def := it.GetItemDefinition()
		if def == nil {
			continue
		}
		// 道具使用需要用户确认目标
		if !def.NeedConfirm {
			t.Errorf("Item %s should need confirm by default", it.String())
		}
	}
}

func TestItemDefinitionPhaseNeedsSubscription(t *testing.T) {
	// 测试 Phase 是否需要订阅 EventBus
	tests := []struct {
		it          ItemType
		needsSub    bool
		reason      string
	}{
		{ItemTypeReverseClock, false, "AnyTime 不需要订阅（主动触发）"},
		{ItemTypeAnyDoor, true, "OnLand 需要订阅"},
		{ItemTypeDiceSwap, false, "AnyTime 不需要订阅（主动触发）"},
		{ItemTypeDiceUpgrade, true, "BeforeTurn 需要订阅"},
	}

	for _, tt := range tests {
		def := tt.it.GetItemDefinition()
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
	// 测试道具的目标类型
	tests := []struct {
		it           ItemType
		targetSelf   bool
		targetOther  bool
	}{
		{ItemTypeReverseClock, false, true},  // 对他人使用
		{ItemTypeAnyDoor, false, true},        // 对他人使用
		{ItemTypeDiceSwap, false, true},       // 对他人使用
		{ItemTypeDiceUpgrade, true, false},    // 对自己使用
	}

	for _, tt := range tests {
		def := tt.it.GetItemDefinition()
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
	// 测试任意门的有效范围
	def := ItemTypeAnyDoor.GetItemDefinition()
	if def == nil {
		t.Fatal("ItemTypeAnyDoor should have definition")
	}
	if def.Range != 30 {
		t.Errorf("AnyDoor Range = %d, expected 30", def.Range)
	}

	// 其他道具默认 Range 为 0
	def = ItemTypeReverseClock.GetItemDefinition()
	if def == nil {
		t.Fatal("ItemTypeReverseClock should have definition")
	}
	if def.Range != 0 {
		t.Errorf("ReverseClock Range = %d, expected 0", def.Range)
	}
}

func TestItemDefinitionBuffType(t *testing.T) {
	// 测试反方向的钟赋予的 Buff 类型
	def := ItemTypeReverseClock.GetItemDefinition()
	if def == nil {
		t.Fatal("ItemTypeReverseClock should have definition")
	}
	if def.BuffType != BuffTypeLost {
		t.Errorf("ReverseClock BuffType = %d, expected Lost", def.BuffType)
	}

	// 其他道具默认不赋予 Buff
	def = ItemTypeAnyDoor.GetItemDefinition()
	if def == nil {
		t.Fatal("ItemTypeAnyDoor should have definition")
	}
	if def.BuffType != BuffTypeNone {
		t.Errorf("AnyDoor BuffType = %d, expected None", def.BuffType)
	}
}