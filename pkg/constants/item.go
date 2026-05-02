// Package constants provides unified enum type definitions.
package constants

// ItemType defines Item type identifiers.
type ItemType string

// ItemType constants - snake_case values for JSON serialization.
const (
	ItemTypeNone ItemType = "none"

	// Consumable Items
	ItemTypeReverseClock ItemType = "reverse_clock" // 反方向的钟: give Lost buff
	ItemTypeAnyDoor       ItemType = "any_door"     // 任意门: teleport
	ItemTypeDiceUpgrade   ItemType = "dice_upgrade" // 骰子升级: upgrade dice
)

// IsValid checks if ItemType is valid.
func (it ItemType) IsValid() bool {
	switch it {
	case ItemTypeReverseClock, ItemTypeAnyDoor, ItemTypeDiceUpgrade:
		return true
	default:
		return false
	}
}

// ParseItemType converts a string to ItemType.
// Returns ItemTypeNone if the string is not a valid item type.
func ParseItemType(s string) ItemType {
	switch s {
	case "reverse_clock":
		return ItemTypeReverseClock
	case "any_door":
		return ItemTypeAnyDoor
	case "dice_upgrade":
		return ItemTypeDiceUpgrade
	default:
		return ItemTypeNone
	}
}

// ========== Item Definition (Static Metadata) ==========

// ItemDefinition contains static metadata for Item display and classification.
// Effect logic is managed by engine layer's ItemHandlerConfig.
type ItemDefinition struct {
	Type        ItemType   `json:"type"`
	Eval        Evaluation `json:"evaluation"`    // Evaluation score for random draw
	EnglishName string     `json:"english_name"`  // English identifier (snake_case)
	Name        string     `json:"name"`          // Chinese display name
	Desc        string     `json:"desc"`          // Description text
}