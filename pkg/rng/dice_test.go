package rng

import (
	"math/rand"
	"testing"
	"time"
)

func TestDiceTypeString(t *testing.T) {
	tests := []struct {
		dt       DiceType
		expected string
	}{
		{DiceTypeGold, "gold"},
		{DiceTypeSilver, "silver"},
		{DiceTypeCopper, "copper"},
		{DiceTypeWood, "wood"},
		{DiceTypeNormal, "normal"},
		{DiceTypeNone, "none"},
	}

	for _, tt := range tests {
		if tt.dt.String() != tt.expected {
			t.Errorf("DiceType(%d).String() = %s, want %s", tt.dt, tt.dt.String(), tt.expected)
		}
	}
}

func TestDiceTypeIsValid(t *testing.T) {
	if DiceTypeNone.IsValid() {
		t.Error("DiceTypeNone should not be valid")
	}
	if !DiceTypeGold.IsValid() {
		t.Error("DiceTypeGold should be valid")
	}
	if !DiceTypeNormal.IsValid() {
		t.Error("DiceTypeNormal should be valid")
	}
}

func TestDiceRoll(t *testing.T) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	tests := []DiceType{
		DiceTypeGold,
		DiceTypeSilver,
		DiceTypeCopper,
		DiceTypeWood,
		DiceTypeNormal,
	}

	for _, dt := range tests {
		dice := NewDice(dt, rng)
		for i := 0; i < 100; i++ {
			result := dice.Roll()
			if result < 1 || result > 6 {
				t.Errorf("Dice(%s).Roll() = %d, want 1-6", dt.String(), result)
			}
		}
	}
}

func TestDiceRollDistribution(t *testing.T) {
	// Use fixed seed for reproducible test
	rng := rand.New(rand.NewSource(42))

	// Test gold dice distribution (should favor high numbers)
	goldDice := NewDice(DiceTypeGold, rng)
	highCount := 0 // count of 5-6
	for i := 0; i < 1000; i++ {
		result := goldDice.Roll()
		if result >= 5 {
			highCount++
		}
	}

	// Gold dice should have ~70% of rolls in 5-6 range
	if highCount < 600 || highCount > 800 {
		t.Errorf("Gold dice high count = %d, expected around 700 (±100)", highCount)
	}

	// Test normal dice distribution (should be uniform)
	normalDice := NewDice(DiceTypeNormal, rng)
	counts := make(map[int]int)
	for i := 0; i < 6000; i++ {
		result := normalDice.Roll()
		counts[result]++
	}

	// Each number should appear ~1000 times (±100)
	for i := 1; i <= 6; i++ {
		if counts[i] < 800 || counts[i] > 1200 {
			t.Errorf("Normal dice count for %d = %d, expected around 1000 (±200)", i, counts[i])
		}
	}
}

func TestDiceRollMultiple(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	dice := NewDice(DiceTypeNormal, rng)

	total := dice.RollMultiple(3)
	if total < 3 || total > 18 {
		t.Errorf("RollMultiple(3) = %d, want 3-18", total)
	}

	// Test zero count
	if dice.RollMultiple(0) != 0 {
		t.Error("RollMultiple(0) should return 0")
	}

	// Test negative count
	if dice.RollMultiple(-1) != 0 {
		t.Error("RollMultiple(-1) should return 0")
	}
}

func TestDiceManager(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	dm := NewDiceManager(rng)

	// Assign dice to players
	dm.AssignDice("p1", 1) // Gold
	dm.AssignDice("p2", 2) // Silver
	dm.AssignDice("p3", 3) // Copper
	dm.AssignDice("p4", 4) // Wood

	// Check assignments
	if dm.GetPlayerDiceType("p1") != DiceTypeGold {
		t.Error("p1 should have gold dice")
	}
	if dm.GetPlayerDiceType("p2") != DiceTypeSilver {
		t.Error("p2 should have silver dice")
	}
	if dm.GetPlayerDiceType("p3") != DiceTypeCopper {
		t.Error("p3 should have copper dice")
	}
	if dm.GetPlayerDiceType("p4") != DiceTypeWood {
		t.Error("p4 should have wood dice")
	}

	// Unknown player gets wood
	if dm.GetPlayerDiceType("unknown") != DiceTypeWood {
		t.Error("unknown player should have wood dice")
	}
}

func TestDiceManagerRoll(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	dm := NewDiceManager(rng)
	dm.AssignDice("p1", 1) // Gold

	// Roll normal dice
	normal := dm.RollNormalDice()
	if normal < 1 || normal > 6 {
		t.Errorf("RollNormalDice() = %d, want 1-6", normal)
	}

	// Roll special dice
	special := dm.RollSpecialDice("p1")
	if special < 1 || special > 6 {
		t.Errorf("RollSpecialDice(p1) = %d, want 1-6", special)
	}

	// Roll both dice
	n, s, total := dm.RollBothDice("p1")
	if n < 1 || n > 6 || s < 1 || s > 6 {
		t.Errorf("RollBothDice() = (%d, %d), want both 1-6", n, s)
	}
	if total != n+s {
		t.Errorf("total = %d, want %d+%d = %d", total, n, s, n+s)
	}

	// Roll total
	total2 := dm.RollTotalDice("p1")
	if total2 < 2 || total2 > 12 {
		t.Errorf("RollTotalDice() = %d, want 2-12", total2)
	}
}

func TestDiceManagerClear(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	dm := NewDiceManager(rng)

	dm.AssignDice("p1", 1)
	if dm.GetPlayerDiceType("p1") != DiceTypeGold {
		t.Error("p1 should have gold dice before clear")
	}

	dm.Clear()
	if dm.GetPlayerDiceType("p1") != DiceTypeWood {
		t.Error("p1 should have wood dice after clear")
	}
}

func TestRankToDiceType(t *testing.T) {
	tests := []struct {
		rank     int
		expected DiceType
	}{
		{1, DiceTypeGold},
		{2, DiceTypeSilver},
		{3, DiceTypeCopper},
		{4, DiceTypeWood},
		{5, DiceTypeWood},
		{0, DiceTypeWood},
	}

	for _, tt := range tests {
		result := RankToDiceType(tt.rank)
		if result != tt.expected {
			t.Errorf("RankToDiceType(%d) = %s, want %s", tt.rank, result.String(), tt.expected.String())
		}
	}
}

func TestDiceTypeToRank(t *testing.T) {
	tests := []struct {
		dt       DiceType
		expected int
	}{
		{DiceTypeGold, 1},
		{DiceTypeSilver, 2},
		{DiceTypeCopper, 3},
		{DiceTypeWood, 4},
		{DiceTypeNormal, 4},
	}

	for _, tt := range tests {
		result := DiceTypeToRank(tt.dt)
		if result != tt.expected {
			t.Errorf("DiceTypeToRank(%s) = %d, want %d", tt.dt.String(), result, tt.expected)
		}
	}
}

func TestDiceTypeFromString(t *testing.T) {
	tests := []struct {
		input    string
		expected DiceType
	}{
		{"gold", DiceTypeGold},
		{"silver", DiceTypeSilver},
		{"copper", DiceTypeCopper},
		{"wood", DiceTypeWood},
		{"normal", DiceTypeNormal},
		{"invalid", DiceTypeNone},
		{"", DiceTypeNone},
		{"GOLD", DiceTypeNone}, // case sensitive
	}

	for _, tt := range tests {
		result := DiceTypeFromString(tt.input)
		if result != tt.expected {
			t.Errorf("DiceTypeFromString(%s) = %s, want %s", tt.input, result.String(), tt.expected.String())
		}
	}
}