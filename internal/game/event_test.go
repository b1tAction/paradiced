package game

import (
	"testing"
)

// ========== EventAttribute Tests ==========

func TestEventAttributeString(t *testing.T) {
	tests := []struct {
		ea       EventAttribute
		expected string
	}{
		{AttributeGood, "Good"},
		{AttributeNeutral, "Neutral"},
		{AttributeBad, "Bad"},
		{EventAttribute(999), "Unknown"},
	}

	for _, tt := range tests {
		result := tt.ea.String()
		if result != tt.expected {
			t.Errorf("EventAttribute(%d).String() = %s, expected %s", tt.ea, result, tt.expected)
		}
	}
}

func TestEventAttributeIsValid(t *testing.T) {
	validAttrs := []EventAttribute{AttributeGood, AttributeNeutral, AttributeBad}
	for _, ea := range validAttrs {
		if !ea.IsValid() {
			t.Errorf("EventAttribute(%d).IsValid() should be true", ea)
		}
	}

	invalidAttrs := []EventAttribute{EventAttribute(-1), EventAttribute(100)}
	for _, ea := range invalidAttrs {
		if ea.IsValid() {
			t.Errorf("EventAttribute(%d).IsValid() should be false", ea)
		}
	}
}

func TestEventAttributeIsPositive(t *testing.T) {
	if !AttributeGood.IsPositive() {
		t.Error("AttributeGood should be positive")
	}
	if AttributeNeutral.IsPositive() {
		t.Error("AttributeNeutral should not be positive")
	}
	if AttributeBad.IsPositive() {
		t.Error("AttributeBad should not be positive")
	}
}

func TestEventAttributeIsNegative(t *testing.T) {
	if !AttributeBad.IsNegative() {
		t.Error("AttributeBad should be negative")
	}
	if AttributeNeutral.IsNegative() {
		t.Error("AttributeNeutral should not be negative")
	}
	if AttributeGood.IsNegative() {
		t.Error("AttributeGood should not be negative")
	}
}

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

func TestEventTypeGetEventAttribute(t *testing.T) {
	// 良性事件
	goodEvents := []EventType{EventTypeHerb, EventTypeMilkTea, EventTypeRelic, EventTypeDivineBless}
	for _, et := range goodEvents {
		if et.GetEventAttribute() != AttributeGood {
			t.Errorf("EventType(%d).GetEventAttribute() = %d, expected Good", et, et.GetEventAttribute())
		}
	}

	// 中性事件
	neutralEvents := []EventType{EventTypeExchange, EventTypeHiddenBuff, EventTypeTasteTest}
	for _, et := range neutralEvents {
		if et.GetEventAttribute() != AttributeNeutral {
			t.Errorf("EventType(%d).GetEventAttribute() = %d, expected Neutral", et, et.GetEventAttribute())
		}
	}

	// 恶性事件
	badEvents := []EventType{EventTypeMosquito, EventTypeGhostHit, EventTypeDogPoop, EventTypeThief, EventTypeCurseBuddha, EventTypeLostWay, EventTypeThunder}
	for _, et := range badEvents {
		if et.GetEventAttribute() != AttributeBad {
			t.Errorf("EventType(%d).GetEventAttribute() = %d, expected Bad", et, et.GetEventAttribute())
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
	if def.Attribute != AttributeGood {
		t.Errorf("def.Attribute = %d, expected Good", def.Attribute)
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
	if def.Attribute != AttributeBad {
		t.Errorf("def.Attribute = %d, expected Bad", def.Attribute)
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

// ========== BuffAttribute Tests ==========

func TestBuffTypeGetBuffAttribute(t *testing.T) {
	// 良性 Buff
	goodBuffs := []BuffType{BuffTypeDivine, BuffTypeHidden, BuffTypeRain, BuffTypeExorcism, BuffTypeFire}
	for _, bt := range goodBuffs {
		if bt.GetBuffAttribute() != AttributeGood {
			t.Errorf("BuffType(%d).GetBuffAttribute() = %d, expected Good", bt, bt.GetBuffAttribute())
		}
	}

	// 恶性 Buff
	badBuffs := []BuffType{BuffTypeCurse, BuffTypeLost, BuffTypeCorrupt, BuffTypePoison}
	for _, bt := range badBuffs {
		if bt.GetBuffAttribute() != AttributeBad {
			t.Errorf("BuffType(%d).GetBuffAttribute() = %d, expected Bad", bt, bt.GetBuffAttribute())
		}
	}
}

func TestBuffTypeGetBuffDefinition(t *testing.T) {
	// 测试诅咒 Buff
	def := BuffTypeCurse.GetBuffDefinition()
	if def == nil {
		t.Fatal("BuffTypeCurse should have definition")
	}
	if def.Name != "诅咒" {
		t.Errorf("def.Name = %s, expected 诅咒", def.Name)
	}
	if def.Attribute != AttributeBad {
		t.Errorf("def.Attribute = %d, expected Bad", def.Attribute)
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
	if def.Attribute != AttributeGood {
		t.Errorf("def.Attribute = %d, expected Good", def.Attribute)
	}
	if def.LPPerTurn != 1 {
		t.Errorf("def.LPPerTurn = %d, expected 1", def.LPPerTurn)
	}

	// 测试隐匿 Buff（特殊效果）
	def = BuffTypeHidden.GetBuffDefinition()
	if def == nil {
		t.Fatal("BuffTypeHidden should have definition")
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

// ========== EventRegistry Tests ==========

func TestNewEventRegistry(t *testing.T) {
	registry := NewEventRegistry()
	if registry == nil {
		t.Fatal("NewEventRegistry should not return nil")
	}
	if len(registry.AllEvents) != 14 {
		t.Errorf("AllEvents count = %d, expected 14 (including None)", len(registry.AllEvents))
	}
	if len(registry.GoodEvents) != 4 {
		t.Errorf("GoodEvents count = %d, expected 4", len(registry.GoodEvents))
	}
	if len(registry.NeutralEvents) != 3 {
		t.Errorf("NeutralEvents count = %d, expected 3", len(registry.NeutralEvents))
	}
	if len(registry.BadEvents) != 7 {
		t.Errorf("BadEvents count = %d, expected 7", len(registry.BadEvents))
	}
}

func TestEventRegistryGetEventsByAttribute(t *testing.T) {
	registry := NewEventRegistry()

	good := registry.GetEventsByAttribute(AttributeGood)
	if len(good) != 4 {
		t.Errorf("Good events count = %d, expected 4", len(good))
	}

	neutral := registry.GetEventsByAttribute(AttributeNeutral)
	if len(neutral) != 3 {
		t.Errorf("Neutral events count = %d, expected 3", len(neutral))
	}

	bad := registry.GetEventsByAttribute(AttributeBad)
	if len(bad) != 7 {
		t.Errorf("Bad events count = %d, expected 7", len(bad))
	}

	// 未知属性返回所有事件
	all := registry.GetEventsByAttribute(EventAttribute(999))
	if len(all) != 14 {
		t.Errorf("Unknown attribute should return all events (including None), got %d", len(all))
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
	if len(registry.GoodBuffs) != 5 {
		t.Errorf("GoodBuffs count = %d, expected 5", len(registry.GoodBuffs))
	}
	if len(registry.BadBuffs) != 4 {
		t.Errorf("BadBuffs count = %d, expected 4", len(registry.BadBuffs))
	}
}

func TestBuffRegistryGetBuffsByAttribute(t *testing.T) {
	registry := NewBuffRegistry()

	good := registry.GetBuffsByAttribute(AttributeGood)
	if len(good) != 5 {
		t.Errorf("Good buffs count = %d, expected 5", len(good))
	}

	bad := registry.GetBuffsByAttribute(AttributeBad)
	if len(bad) != 4 {
		t.Errorf("Bad buffs count = %d, expected 4", len(bad))
	}

	// 未知属性返回所有 Buff
	all := registry.GetBuffsByAttribute(EventAttribute(999))
	if len(all) != 9 {
		t.Errorf("Unknown attribute should return all buffs, got %d", len(all))
	}
}