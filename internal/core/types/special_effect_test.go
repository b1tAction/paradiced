package types

import "testing"

func TestSpecialEffectString(t *testing.T) {
	tests := []struct {
		se       SpecialEffect
		expected string
	}{
		{SpecialNone, "None"},
		{SpecialImmune, "Immune"},
		{SpecialReverse, "Reverse"},
		{SpecialImmunePoison, "ImmunePoison"},
		{SpecialBadEvent, "BadEvent"},
		{SpecialZhuQuePassive, "ZhuQuePassive"},
		{SpecialTeleport, "Teleport"},
		{SpecialDiceSwap, "DiceSwap"},
		{SpecialDiceUpgrade, "DiceUpgrade"},
		{SpecialGiveLost, "GiveLost"},
		{SpecialDrawItem, "DrawItem"},
		{SpecialLoseItem, "LoseItem"},
		{SpecialSwapPosition, "SwapPosition"},
		{SpecialRandomBuff, "RandomBuff"},
		{SpecialEffect(999), "Unknown"},
	}

	for _, tt := range tests {
		result := tt.se.String()
		if result != tt.expected {
			t.Errorf("SpecialEffect(%d).String() = %s, expected %s", tt.se, result, tt.expected)
		}
	}
}

func TestSpecialEffectIsValid(t *testing.T) {
	// Valid special effects (0~13)
	for i := SpecialNone; i <= SpecialRandomBuff; i++ {
		if !i.IsValid() {
			t.Errorf("SpecialEffect(%d).IsValid() should be true", i)
		}
	}

	// Invalid special effects
	invalidEffects := []SpecialEffect{SpecialEffect(-1), SpecialEffect(100)}
	for _, se := range invalidEffects {
		if se.IsValid() {
			t.Errorf("SpecialEffect(%d).IsValid() should be false", se)
		}
	}
}

func TestSpecialEffectIsBuffEffect(t *testing.T) {
	// Buff effects: 1~5
	buffEffects := []SpecialEffect{SpecialImmune, SpecialReverse, SpecialImmunePoison, SpecialBadEvent, SpecialZhuQuePassive}
	for _, se := range buffEffects {
		if !se.IsBuffEffect() {
			t.Errorf("SpecialEffect(%d).IsBuffEffect() should be true", se)
		}
	}

	// Not Buff effects
	notBuffEffects := []SpecialEffect{SpecialNone, SpecialTeleport, SpecialDrawItem}
	for _, se := range notBuffEffects {
		if se.IsBuffEffect() {
			t.Errorf("SpecialEffect(%d).IsBuffEffect() should be false", se)
		}
	}
}

func TestSpecialEffectIsItemEffect(t *testing.T) {
	// Item effects: 6~9
	itemEffects := []SpecialEffect{SpecialTeleport, SpecialDiceSwap, SpecialDiceUpgrade, SpecialGiveLost}
	for _, se := range itemEffects {
		if !se.IsItemEffect() {
			t.Errorf("SpecialEffect(%d).IsItemEffect() should be true", se)
		}
	}

	// Not Item effects
	notItemEffects := []SpecialEffect{SpecialNone, SpecialImmune, SpecialDrawItem}
	for _, se := range notItemEffects {
		if se.IsItemEffect() {
			t.Errorf("SpecialEffect(%d).IsItemEffect() should be false", se)
		}
	}
}

func TestSpecialEffectIsEventEffect(t *testing.T) {
	// Event effects: 10~13
	eventEffects := []SpecialEffect{SpecialDrawItem, SpecialLoseItem, SpecialSwapPosition, SpecialRandomBuff}
	for _, se := range eventEffects {
		if !se.IsEventEffect() {
			t.Errorf("SpecialEffect(%d).IsEventEffect() should be true", se)
		}
	}

	// Not Event effects
	notEventEffects := []SpecialEffect{SpecialNone, SpecialImmune, SpecialTeleport}
	for _, se := range notEventEffects {
		if se.IsEventEffect() {
			t.Errorf("SpecialEffect(%d).IsEventEffect() should be false", se)
		}
	}
}