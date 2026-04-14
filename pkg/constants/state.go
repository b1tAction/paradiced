// Package constants provides unified enum type definitions.
package constants

// StateID defines HSM state identifier.
type StateID string

// StateID constants - snake_case values for JSON serialization.
// Layer 1: Global states (match-level)
// Layer 2: Turn states (player-level)
// Layer 3: Interrupt states (decision-level)
const (
	StateNone StateID = "none"

	// ========== Layer 1: Global States ==========

	StateMatchInit      StateID = "match_init"
	StateRoundMiniGame  StateID = "round_mini_game"
	StateRoundPrep      StateID = "round_prep"
	StateTurnLoop       StateID = "turn_loop"
	StateBossBattle     StateID = "boss_battle"
	StateGameOver       StateID = "game_over"

	// ========== Layer 2: Turn States ==========

	StateTurnUpkeep  StateID = "turn_upkeep"
	StateMainAction  StateID = "main_action"
	StateTurnMoving  StateID = "turn_moving"
	StateTurnLanded  StateID = "turn_landed"
	StateTurnEvent   StateID = "turn_event"
	StateTurnEnd     StateID = "turn_end"

	// ========== Layer 3: Interrupt States ==========

	StateWaitDecision StateID = "wait_decision"

	// ========== Invalid State ==========

	StateInvalid StateID = "invalid"
)

// IsValid checks if StateID is valid.
func (sid StateID) IsValid() bool {
	return sid != StateNone && sid != StateInvalid && sid != ""
}

// IsGlobalState checks if this is a Layer 1 global state.
func (sid StateID) IsGlobalState() bool {
	return sid == StateMatchInit || sid == StateRoundMiniGame ||
		sid == StateRoundPrep || sid == StateTurnLoop ||
		sid == StateBossBattle || sid == StateGameOver
}

// IsTurnState checks if this is a Layer 2 turn state.
func (sid StateID) IsTurnState() bool {
	return sid == StateTurnUpkeep || sid == StateMainAction ||
		sid == StateTurnMoving || sid == StateTurnLanded ||
		sid == StateTurnEvent || sid == StateTurnEnd
}

// IsInterruptState checks if this is a Layer 3 interrupt state.
func (sid StateID) IsInterruptState() bool {
	return sid == StateWaitDecision
}

// Layer returns the state layer (1, 2, or 3).
func (sid StateID) Layer() int {
	if sid.IsGlobalState() {
		return 1
	}
	if sid.IsTurnState() {
		return 2
	}
	if sid.IsInterruptState() {
		return 3
	}
	return 0
}