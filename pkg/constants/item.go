// Package constants provides unified enum type definitions.
package constants

// ItemType defines Item type identifiers.
type ItemType string

// ItemType constants - snake_case values for JSON serialization.
const (
	ItemTypeNone ItemType = "none"

	// Consumable Items
	ItemTypeReverseClock  ItemType = "reverse_clock"  // 反方向的钟: give Lost buff
	ItemTypeAnyDoor       ItemType = "any_door"       // 任意门: teleport
	ItemTypeDiceUpgrade   ItemType = "dice_upgrade"   // 骰子升级: upgrade dice
	ItemTypeMagicFlute    ItemType = "magic_flute"    // 魔笛: give Sinking buff to self and target
	ItemTypeCupidArrow    ItemType = "cupid_arrow"    // 丘比特之箭: give Eternal buff to self and target
	ItemTypeCrimsonBlade  ItemType = "crimson_blade"  // 猩红之刃: sacrifice half HP, deal damage to target
	ItemTypeWisdomRing    ItemType = "wisdom_ring"    // 智慧玄戒: give Divine buff
	ItemTypeMeditationRing ItemType = "meditation_ring" // 禅定玄戒: give Rain buff
	ItemTypeDisciplineRing ItemType = "discipline_ring" // 持戒玄戒: give Golden Body buff
	ItemTypeFoolishRing   ItemType = "foolish_ring"   // 痴愚煞戒: HP+1, LP-1
	ItemTypeGreedyRing    ItemType = "greedy_ring"    // 贪婪煞戒: LP+1, HP-1
	ItemTypeWrathRing     ItemType = "wrath_ring"     // 嗔恨煞戒: HP-1, gain Wrath buff
	ItemTypeNamedBlade    ItemType = "named_blade"    // 名刀司命: block one fatal damage
	ItemTypeSageProtection ItemType = "sage_protection" // 贤者的庇护: respawn in-place
)

// IsValid checks if ItemType is valid.
func (it ItemType) IsValid() bool {
	switch it {
	case ItemTypeReverseClock, ItemTypeAnyDoor, ItemTypeDiceUpgrade,
		ItemTypeMagicFlute, ItemTypeCupidArrow, ItemTypeCrimsonBlade,
		ItemTypeWisdomRing, ItemTypeMeditationRing, ItemTypeDisciplineRing,
		ItemTypeFoolishRing, ItemTypeGreedyRing, ItemTypeWrathRing,
		ItemTypeNamedBlade, ItemTypeSageProtection:
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
	case "magic_flute":
		return ItemTypeMagicFlute
	case "cupid_arrow":
		return ItemTypeCupidArrow
	case "crimson_blade":
		return ItemTypeCrimsonBlade
	case "wisdom_ring":
		return ItemTypeWisdomRing
	case "meditation_ring":
		return ItemTypeMeditationRing
	case "discipline_ring":
		return ItemTypeDisciplineRing
	case "foolish_ring":
		return ItemTypeFoolishRing
	case "greedy_ring":
		return ItemTypeGreedyRing
	case "wrath_ring":
		return ItemTypeWrathRing
	case "named_blade":
		return ItemTypeNamedBlade
	case "sage_protection":
		return ItemTypeSageProtection
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
	Triggerable bool       `json:"triggerable"`   // Whether player can actively use this item
	Targetable  bool       `json:"targetable"`    // Whether item requires target player selection
}