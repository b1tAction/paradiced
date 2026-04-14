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
	ItemTypeDiceSwap      ItemType = "dice_swap"    // 骰子交换: swap dice
	ItemTypeDiceUpgrade   ItemType = "dice_upgrade" // 骰子升级: upgrade dice
)

// IsValid checks if ItemType is valid.
func (it ItemType) IsValid() bool {
	switch it {
	case ItemTypeReverseClock, ItemTypeAnyDoor, ItemTypeDiceSwap, ItemTypeDiceUpgrade:
		return true
	default:
		return false
	}
}