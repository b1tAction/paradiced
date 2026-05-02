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
	switch et {
	case EventTypeHerb, EventTypeMilkTea, EventTypeRelic, EventTypeDivineBless,
		EventTypeExchange, EventTypeHiddenBuff, EventTypeTasteTest,
		EventTypeMosquito, EventTypeGhostHit, EventTypeDogPoop,
		EventTypeThief, EventTypeCurseBuddha, EventTypeLostWay, EventTypeThunder:
		return true
	default:
		return false
	}
}

// ParseEventType converts a string to EventType.
// Returns EventTypeNone if the string is not a valid event type.
func ParseEventType(s string) EventType {
	switch s {
	case "herb":
		return EventTypeHerb
	case "milk_tea":
		return EventTypeMilkTea
	case "relic":
		return EventTypeRelic
	case "divine_bless":
		return EventTypeDivineBless
	case "exchange":
		return EventTypeExchange
	case "hidden_buff":
		return EventTypeHiddenBuff
	case "taste_test":
		return EventTypeTasteTest
	case "mosquito":
		return EventTypeMosquito
	case "ghost_hit":
		return EventTypeGhostHit
	case "dog_poop":
		return EventTypeDogPoop
	case "thief":
		return EventTypeThief
	case "curse_buddha":
		return EventTypeCurseBuddha
	case "lost_way":
		return EventTypeLostWay
	case "thunder":
		return EventTypeThunder
	default:
		return EventTypeNone
	}
}

// ========== Event Definition (Static Metadata) ==========

// EventDefinition contains static metadata for Event display and classification.
// Effect logic is managed by engine layer's EventHandlerConfig.
type EventDefinition struct {
	Type        EventType  `json:"type"`
	Eval        Evaluation `json:"evaluation"`    // Evaluation score for random draw
	EnglishName string     `json:"english_name"`  // English identifier (snake_case)
	Name        string     `json:"name"`          // Chinese display name
	Desc        string     `json:"desc"`          // Description text
}

// IsGood checks if the event is beneficial (Evaluation > 65).
func (d *EventDefinition) IsGood() bool {
	return d.Eval.IsGood()
}

// IsBad checks if the event is harmful (Evaluation <= 40).
func (d *EventDefinition) IsBad() bool {
	return d.Eval.IsBad()
}

// IsNeutral checks if the event is neutral (Evaluation 41-65).
func (d *EventDefinition) IsNeutral() bool {
	return !d.IsGood() && !d.IsBad()
}