// Package constants provides unified enum type definitions.
package constants

// EventType defines Event type identifiers.
type EventType string

// EventType constants - snake_case values for JSON serialization.
const (
	EventTypeNone EventType = "none"

	// Good Events
	EventTypeHerb        EventType = "herb"         // 草药: HP+1
	EventTypeMilkTea     EventType = "milk_tea"     // 奶茶: LP+1
	EventTypeRelic       EventType = "relic"        // 圣遗物: draw item
	EventTypeDivineBless EventType = "divine_bless" // 天使眷顾: Divine buff

	// Neutral Events
	EventTypeExchange   EventType = "exchange"    // 交换: swap position
	EventTypeHiddenBuff EventType = "hidden_buff" // 隐匿: Hidden buff
	EventTypeTasteTest  EventType = "taste_test"  // 尝一口: random buff

	// Bad Events
	EventTypeMosquito    EventType = "mosquito"     // 蚊虫: HP-1
	EventTypeGhostHit    EventType = "ghost_hit"    // 野鬼: HP-1
	EventTypeDogPoop     EventType = "dog_poop"     // 狗屎: LP-1
	EventTypeThief       EventType = "thief"        // 盗贼: lose item
	EventTypeCurseBuddha EventType = "curse_buddha" // 野佛: Curse buff
	EventTypeLostWay     EventType = "lost_way"     // 迷途: Lost buff
	EventTypeThunder     EventType = "thunder"      // 雷劫: HP=0
)

// IsValid checks if EventType is valid.
func (et EventType) IsValid() bool {
	return et != EventTypeNone && et != ""
}