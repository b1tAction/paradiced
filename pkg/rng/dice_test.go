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

// ========== Boss Attack Calculation Tests ==========

func TestCalcBossCritSkillProb(t *testing.T) {
	tests := []struct {
		name     string
		avgLP    float64
		bossHP   int
		bossMax  int
		expected float64
	}{
		{"base rate", 8, 50, 50, 0.25},
		{"lowLP wounded boss", 4, 25, 50, 0.60},
		{"zeroLP zeroHP", 0, 0, 50, 0.95},
	}
	for _, tt := range tests {
		result := CalcBossCritSkillProb(tt.avgLP, tt.bossHP, tt.bossMax)
		// Use approximate comparison due to floating point
		if result < tt.expected-0.01 || result > tt.expected+0.01 {
			t.Errorf("%s: CalcBossCritSkillProb(%v, %d, %d) = %.4f, want %.2f",
				tt.name, tt.avgLP, tt.bossHP, tt.bossMax, result, tt.expected)
		}
	}
}

func TestCalcBossAttackType(t *testing.T) {
	r := rand.New(rand.NewSource(42))

	// Test normal attack when probability is very low (avgLP=8, bossHP=50)
	result := CalcBossAttackType(r, 8.0, 50, 50, nil)
	if result.AttackType != "normal" && result.AttackType != "crit" && result.AttackType != "skill" {
		t.Errorf("CalcBossAttackType returned invalid attack type: %s", result.AttackType)
	}

	// Test with skill pool
	pool := []*EvaluatedItem{
		{Type: "thunder", Eval: 50},
		{Type: "curse", Eval: 50},
	}
	result2 := CalcBossAttackType(r, 8.0, 50, 50, pool)
	// Result should be one of the valid types
	if result2.AttackType != "normal" && result2.AttackType != "crit" && result2.AttackType != "skill" {
		t.Errorf("CalcBossAttackType with pool returned invalid type: %s", result2.AttackType)
	}
	// If skill, check damage is 0 and skill type is from pool
	if result2.AttackType == "skill" {
		if result2.Damage != 0 {
			t.Errorf("Skill damage = %d, expected 0", result2.Damage)
		}
		if result2.SkillType != "thunder" && result2.SkillType != "curse" {
			t.Errorf("Skill type = %s, expected thunder or curse", result2.SkillType)
		}
	}
}

func TestSelectBossTarget(t *testing.T) {
	r := rand.New(rand.NewSource(42))

	// Empty players returns empty string
	result := SelectBossTarget(r, nil)
	if result != "" {
		t.Errorf("SelectBossTarget with nil players = %s, want empty", result)
	}

	// Single player always selected
	single := []BossTargetCandidate{{PlayerID: "p1", LP: 5}}
	result = SelectBossTarget(r, single)
	if result != "p1" {
		t.Errorf("SelectBossTarget single player = %s, want p1", result)
	}

	// Multiple players - low LP player should be more likely
	players := []BossTargetCandidate{
		{PlayerID: "p1", LP: 8},
		{PlayerID: "p2", LP: 2},
		{PlayerID: "p3", LP: 5},
	}
	// Run many iterations to check distribution
	p2Count := 0
	for i := 0; i < 1000; i++ {
		r2 := rand.New(rand.NewSource(int64(i)))
		target := SelectBossTarget(r2, players)
		if target == "p2" {
			p2Count++
		}
	}
	// p2 (LP=2) should be selected more often than random (~1/3 = 333)
	if p2Count < 400 {
		t.Errorf("p2 selection count = %d, expected >400 (low LP should be targeted more)", p2Count)
	}
}

func TestCalcPlayerCritRate(t *testing.T) {
	tests := []struct {
		diceType DiceType
		expected float64
	}{
		{DiceTypeGold, 0.30},
		{DiceTypeSilver, 0.20},
		{DiceTypeCopper, 0.10},
		{DiceTypeWood, 0.05},
		{DiceTypeNormal, 0.05},
		{DiceTypeNone, 0.05}, // defaults to wood rate
	}
	for _, tt := range tests {
		result := CalcPlayerCritRate(tt.diceType)
		if result != tt.expected {
			t.Errorf("CalcPlayerCritRate(%s) = %.2f, want %.2f", tt.diceType.String(), result, tt.expected)
		}
	}
}

func TestCalcPlayerCrit(t *testing.T) {
	r := rand.New(rand.NewSource(42))

	// Gold dice should crit more often
	goldCritCount := 0
	for i := 0; i < 1000; i++ {
		if CalcPlayerCrit(r, DiceTypeGold) {
			goldCritCount++
		}
	}
	if goldCritCount < 200 || goldCritCount > 400 {
		t.Errorf("Gold dice crit count = %d, expected ~300 (±100)", goldCritCount)
	}

	// Wood dice should rarely crit
	r2 := rand.New(rand.NewSource(42))
	woodCritCount := 0
	for i := 0; i < 1000; i++ {
		if CalcPlayerCrit(r2, DiceTypeWood) {
			woodCritCount++
		}
	}
	if woodCritCount > 100 {
		t.Errorf("Wood dice crit count = %d, expected <100", woodCritCount)
	}
}