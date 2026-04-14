// Package constants provides unified enum type definitions.
package constants

// EntryType defines game log entry type.
type EntryType string

// EntryType constants - snake_case values for JSON serialization.
const (
	EntryTypeAction   EntryType = "action"    // Action execution
	EntryTypeState    EntryType = "state"     // HSM state transition
	EntryTypeMiniGame EntryType = "mini_game" // Mini game result
	EntryTypeBoss     EntryType = "boss"      // Boss battle
	EntryTypeDecision EntryType = "decision"  // User decision
)

// IsValid checks if EntryType is valid.
func (et EntryType) IsValid() bool {
	return et == EntryTypeAction || et == EntryTypeState ||
		et == EntryTypeMiniGame || et == EntryTypeBoss ||
		et == EntryTypeDecision
}