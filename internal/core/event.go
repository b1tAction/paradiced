package core

// ========== Event Type Definitions ==========

type EventType int

const (
	EventTypeNone EventType = iota

	// Good events (Evaluation > 65)
	EventTypeHerb         // Herb: HP+1
	EventTypeMilkTea      // MilkTea: LP+1
	EventTypeRelic        // Relic: item draw
	EventTypeDivineBless  // DivineBless: Divine buff

	// Neutral events (Evaluation 41~65)
	EventTypeExchange     // Exchange: swap position with random player
	EventTypeHiddenBuff   // HiddenBuff: Hidden buff
	EventTypeTasteTest    // TasteTest: random buff (Corrupt/Rain)

	// Bad events (Evaluation ≤ 40)
	EventTypeMosquito     // Mosquito: HP-1
	EventTypeGhostHit     // GhostHit: HP-1
	EventTypeDogPoop      // DogPoop: LP-1
	EventTypeThief        // Thief: random item loss
	EventTypeCurseBuddha  // CurseBuddha: Curse buff
	EventTypeLostWay      // LostWay: Lost buff
	EventTypeThunder      // Thunder: HP to 0
)

// IsValid checks if the Event type is valid.
func (et EventType) IsValid() bool {
	return et > EventTypeNone && et <= EventTypeThunder
}

// String returns the Event type name from GlobalRegistry.
func (et EventType) String() string {
	return GlobalRegistry.GetEventString(et)
}

// ========== Event Definition ==========

type EventDefinition struct {
	Type          EventType     `json:"type"`
	Eval          Evaluation    `json:"evaluation"`
	EnglishName   string        `json:"english_name"` // English identifier (for String())
	Name          string        `json:"name"`         // Chinese display name
	Desc          string        `json:"desc"`
	HPChange      int           `json:"hp_change"`
	LPChange      int           `json:"lp_change"`
	BuffType      BuffType      `json:"buff_type"`
	SpecialEffect SpecialEffect `json:"special_effect"` // Special effect type
}

// ========== Global Registry Access Functions ==========

// GetEventDefinition returns the Event definition from GlobalRegistry.
func GetEventDefinition(et EventType) *EventDefinition {
	return GlobalRegistry.GetEventDefinition(et)
}

// GetEventString returns the Event name string from GlobalRegistry.
func GetEventString(et EventType) string {
	return GlobalRegistry.GetEventString(et)
}

// GetEventEvaluation returns the Event evaluation score from GlobalRegistry.
func GetEventEvaluation(et EventType) Evaluation {
	return GlobalRegistry.GetEventEvaluation(et)
}

// GetAllEventTypes returns all registered Event types.
func GetAllEventTypes() []EventType {
	return GlobalRegistry.GetAllEventTypes()
}

// GetEventTypesByCategory returns Event types by category.
func GetEventTypesByCategory(category string) []EventType {
	return GlobalRegistry.GetEventTypesByCategory(category)
}

// GetAllEventDefinitions returns all Event definitions.
func GetAllEventDefinitions() []*EventDefinition {
	return GlobalRegistry.GetAllEventDefinitions()
}