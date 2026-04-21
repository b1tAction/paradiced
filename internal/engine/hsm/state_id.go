package hsm

// StateID defines unique identifiers for all states in the HSM.
// States are organized into three layers: Global, Turn, and Interrupt.
type StateID int

// ========== Layer 1: Global Game States ==========

const (
	// Global layer states manage the overall game lifecycle
	StateMatchInit StateID = iota + 100 // 100: Initialize match (map, factions, initial buffs)
	StateWaitingForHost                  // 101: Wait for host to start game (manual start mode)
	StateRoundMiniGame                   // 102: Mini-game phase, wait for rankings
	StateRoundPrep                       // 103: Round preparation, assign dice types
	StateTurnLoop                        // 104: Turn loop, iterate through player turns
	StateBossBattle                      // 105: Boss battle when player reaches end
	StateGameOver                        // 106: Game ended, broadcast winner
)

// ========== Layer 2: Player Turn States ==========

const (
	// Turn layer states manage individual player turn flow
	StateTurnUpkeep StateID = iota + 200 // 200: Turn preparation, check SkipTurn/IsDead, trigger PhaseBeforeTurn
	StateMainAction                       // 201: Main action phase, wait for item/skill/dice
	StateTurnMoving                       // 202: Movement phase, calculate path, handle Fragile/Fog
	StateTurnLanded                       // 203: Landing phase, trigger PhaseOnLand
	StateTurnEvent                        // 204: Event phase, trigger PhasePreEvent/PhasePreDamage
	StateTurnEnd                          // 205: Turn end phase, trigger PhaseAfterTurn, TickBuffs
	StateTurnCheckpoint                   // 206: CheckPoint processing (DrawItem etc.)
)

// ========== Layer 3: Interrupt States ==========

const (
	// Interrupt layer states handle user decisions with stack mechanism
	StateWaitDecision StateID = iota + 300 // 300: Waiting for user decision, timeout support
)

// ========== Helper Constants ==========

const (
	StateNone    StateID = 0   // No active state
	StateInvalid StateID = -1  // Invalid state marker
)

// String returns the string representation of StateID.
func (s StateID) String() string {
	names := map[StateID]string{
		StateMatchInit:      "MatchInit",
		StateWaitingForHost: "WaitingForHost",
		StateRoundMiniGame:  "RoundMiniGame",
		StateRoundPrep:      "RoundPrep",
		StateTurnLoop:       "TurnLoop",
		StateBossBattle:     "BossBattle",
		StateGameOver:       "GameOver",
		StateTurnUpkeep:     "TurnUpkeep",
		StateMainAction:     "MainAction",
		StateTurnMoving:     "TurnMoving",
		StateTurnLanded:     "TurnLanded",
		StateTurnEvent:      "TurnEvent",
		StateTurnEnd:        "TurnEnd",
		StateTurnCheckpoint: "TurnCheckpoint",
		StateWaitDecision:   "WaitDecision",
		StateNone:           "None",
		StateInvalid:        "Invalid",
	}
	if name, ok := names[s]; ok {
		return name
	}
	return "Unknown"
}

// IsValid checks if StateID is valid.
func (s StateID) IsValid() bool {
	return s >= StateMatchInit && s <= StateWaitDecision || s == StateNone
}

// IsGlobalState checks if state belongs to Layer 1 (Global).
func (s StateID) IsGlobalState() bool {
	return s >= StateMatchInit && s <= StateGameOver
}

// IsTurnState checks if state belongs to Layer 2 (Turn).
func (s StateID) IsTurnState() bool {
	return s >= StateTurnUpkeep && s <= StateTurnCheckpoint
}

// IsInterruptState checks if state belongs to Layer 3 (Interrupt).
func (s StateID) IsInterruptState() bool {
	return s >= StateWaitDecision && s <= StateWaitDecision
}

// Layer returns the layer number of the state (1, 2, or 3).
func (s StateID) Layer() int {
	if s.IsGlobalState() {
		return 1
	}
	if s.IsTurnState() {
		return 2
	}
	if s.IsInterruptState() {
		return 3
	}
	return 0
}