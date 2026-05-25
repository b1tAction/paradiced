package rng

import (
	"fmt"
	"math/rand"
)

// DiceType represents the type of dice with different weight distributions.
// Higher rank dice types have higher probability of rolling larger numbers.
type DiceType int

const (
	// DiceTypeNone represents no dice (invalid).
	DiceTypeNone DiceType = iota
	// DiceTypeGold is awarded to the 1st place in mini-game.
	// Weight distribution: 1-2(10%), 3-4(20%), 5-6(70%) - easiest to get high numbers.
	DiceTypeGold
	// DiceTypeSilver is awarded to the 2nd place in mini-game.
	// Weight distribution: 1-2(20%), 3-4(30%), 5-6(50%).
	DiceTypeSilver
	// DiceTypeCopper is awarded to the 3rd place in mini-game.
	// Weight distribution: 1-2(30%), 3-4(30%), 5-6(40%).
	DiceTypeCopper
	// DiceTypeWood is awarded to the 4th place (or default).
	// Uniform distribution: 1-6 each 16.67%.
	DiceTypeWood
	// DiceTypeNormal is a standard dice (always available to all players).
	// Uniform distribution: 1-6 each 16.67%.
	DiceTypeNormal
)

// String returns the dice type name.
func (dt DiceType) String() string {
	switch dt {
	case DiceTypeGold:
		return "gold"
	case DiceTypeSilver:
		return "silver"
	case DiceTypeCopper:
		return "copper"
	case DiceTypeWood:
		return "wood"
	case DiceTypeNormal:
		return "normal"
	default:
		return "none"
	}
}

// DiceTypeFromString converts string to DiceType.
func DiceTypeFromString(s string) DiceType {
	switch s {
	case "gold":
		return DiceTypeGold
	case "silver":
		return DiceTypeSilver
	case "copper":
		return DiceTypeCopper
	case "wood":
		return DiceTypeWood
	case "normal":
		return DiceTypeNormal
	default:
		return DiceTypeNone
	}
}

// IsValid checks if the dice type is valid.
func (dt DiceType) IsValid() bool {
	return dt >= DiceTypeGold && dt <= DiceTypeNormal
}

// Dice represents a dice instance with type and roll capabilities.
type Dice struct {
	Type DiceType
	rng  *rand.Rand
}

// NewDice creates a new dice with the given type and random source.
func NewDice(diceType DiceType, rng *rand.Rand) *Dice {
	return &Dice{
		Type: diceType,
		rng:  rng,
	}
}

// Roll performs a weighted random roll based on dice type.
// Returns a value between 1-6 (inclusive).
func (d *Dice) Roll() int {
	if d.rng == nil {
		return 1 // Fallback
	}

	weights := d.getWeights()
	totalWeight := 0.0
	for _, w := range weights {
		totalWeight += w
	}

	// Weighted random selection
	r := d.rng.Float64() * totalWeight
	cumulative := 0.0
	for i, w := range weights {
		cumulative += w
		if r < cumulative {
			return i + 1 // Dice values are 1-6
		}
	}

	return 6 // Fallback to max
}

// getWeights returns the weight distribution for each dice face (1-6).
func (d *Dice) getWeights() []float64 {
	switch d.Type {
	case DiceTypeGold:
		// 1-2(10%), 3-4(20%), 5-6(70%)
		return []float64{0.05, 0.05, 0.10, 0.10, 0.35, 0.35}
	case DiceTypeSilver:
		// 1-2(20%), 3-4(30%), 5-6(50%)
		return []float64{0.10, 0.10, 0.15, 0.15, 0.25, 0.25}
	case DiceTypeCopper:
		// 1-2(30%), 3-4(30%), 5-6(40%)
		return []float64{0.15, 0.15, 0.15, 0.15, 0.20, 0.20}
	case DiceTypeWood, DiceTypeNormal:
		// Uniform distribution: 1-6 each ~16.67%
		return []float64{1, 1, 1, 1, 1, 1}
	default:
		// Fallback to uniform
		return []float64{1, 1, 1, 1, 1, 1}
	}
}

// RollMultiple rolls multiple dice and returns the sum.
// For each roll, uses the same dice type but independent random draws.
func (d *Dice) RollMultiple(count int) int {
	if count <= 0 {
		return 0
	}

	total := 0
	for i := 0; i < count; i++ {
		total += d.Roll()
	}
	return total
}

// ========== Dice Manager ==========

// DiceManager manages dice for players in a game.
// Each player has one normal dice (always) and one special dice (from mini-game ranking).
type DiceManager struct {
	playerDice map[string]DiceType // playerID -> special dice type (won from mini-game)
	rng        *rand.Rand
}

// NewDiceManager creates a new dice manager with the given random source.
func NewDiceManager(rng *rand.Rand) *DiceManager {
	return &DiceManager{
		playerDice: make(map[string]DiceType),
		rng:        rng,
	}
}

// AssignDice assigns a special dice type to a player based on mini-game ranking.
func (dm *DiceManager) AssignDice(playerID string, rank int) {
	diceType := RankToDiceType(rank)
	dm.playerDice[playerID] = diceType
}

// GetPlayerDiceType returns the special dice type for a player.
// Returns DiceTypeWood if player has no assigned special dice.
func (dm *DiceManager) GetPlayerDiceType(playerID string) DiceType {
	if dt, ok := dm.playerDice[playerID]; ok {
		return dt
	}
	return DiceTypeWood
}

// RollNormalDice rolls a normal dice (uniform 1-6).
func (dm *DiceManager) RollNormalDice() int {
	dice := NewDice(DiceTypeNormal, dm.rng)
	return dice.Roll()
}

// RollSpecialDice rolls a player's special dice.
func (dm *DiceManager) RollSpecialDice(playerID string) int {
	diceType := dm.GetPlayerDiceType(playerID)
	dice := NewDice(diceType, dm.rng)
	return dice.Roll()
}

// RollBothDice rolls both normal and special dice for a player.
// Returns (normalResult, specialResult, total).
func (dm *DiceManager) RollBothDice(playerID string) (int, int, int) {
	normal := dm.RollNormalDice()
	special := dm.RollSpecialDice(playerID)
	return normal, special, normal + special
}

// RollTotalDice rolls both dice and returns only the total.
func (dm *DiceManager) RollTotalDice(playerID string) int {
	normal, special, _ := dm.RollBothDice(playerID)
	return normal + special
}

// Clear removes all dice assignments.
func (dm *DiceManager) Clear() {
	dm.playerDice = make(map[string]DiceType)
}

// ========== Helper Functions ==========

// RankToDiceType converts mini-game ranking to dice type.
// Rank 1 -> Gold, Rank 2 -> Silver, Rank 3 -> Copper, Rank 4+ -> Wood.
func RankToDiceType(rank int) DiceType {
	switch rank {
	case 1:
		return DiceTypeGold
	case 2:
		return DiceTypeSilver
	case 3:
		return DiceTypeCopper
	default:
		return DiceTypeWood
	}
}

// DiceTypeToRank converts dice type back to minimum ranking required.
func DiceTypeToRank(dt DiceType) int {
	switch dt {
	case DiceTypeGold:
		return 1
	case DiceTypeSilver:
		return 2
	case DiceTypeCopper:
		return 3
	default:
		return 4
	}
}

// ========== Dice Definition (Static Metadata) ==========

// DiceDefinition contains static metadata for Dice display.
type DiceDefinition struct {
	Type        DiceType `json:"type"`
	EnglishName string   `json:"english_name"` // e.g. "Gold"
	Name        string   `json:"name"`         // Chinese display name e.g. "金骰子"
	Desc        string   `json:"desc"`         // Description e.g. "1-2(10%), 3-4(20%), 5-6(70%)"
	Rank        int      `json:"rank"`         // Mini-game rank that earns this dice
}

// String returns a formatted description of the dice type.
func (d *Dice) String() string {
	return fmt.Sprintf("Dice{type=%s}", d.Type.String())
}