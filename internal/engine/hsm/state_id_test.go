package hsm

import (
	"testing"
)

func TestStateIDString(t *testing.T) {
	tests := []struct {
		id       StateID
		expected string
	}{
		{StateMatchInit, "MatchInit"},
		{StateRoundMiniGame, "RoundMiniGame"},
		{StateRoundPrep, "RoundPrep"},
		{StateTurnLoop, "TurnLoop"},
		{StateBossBattle, "BossBattle"},
		{StateGameOver, "GameOver"},
		{StateTurnUpkeep, "TurnUpkeep"},
		{StateMainAction, "MainAction"},
		{StateTurnMoving, "TurnMoving"},
		{StateTurnLanded, "TurnLanded"},
		{StateTurnEvent, "TurnEvent"},
		{StateTurnEnd, "TurnEnd"},
		{StateWaitDecision, "WaitDecision"},
		{StateNone, "None"},
		{StateInvalid, "Invalid"},
		{StateID(999), "Unknown"},
	}

	for _, tt := range tests {
		if got := tt.id.String(); got != tt.expected {
			t.Errorf("StateID(%d).String() = %s, want %s", tt.id, got, tt.expected)
		}
	}
}

func TestStateIDIsValid(t *testing.T) {
	validIDs := []StateID{
		StateMatchInit, StateRoundMiniGame, StateRoundPrep, StateTurnLoop,
		StateBossBattle, StateGameOver,
		StateTurnUpkeep, StateMainAction, StateTurnMoving, StateTurnLanded,
		StateTurnEvent, StateTurnEnd,
		StateWaitDecision,
		StateNone,
	}

	for _, id := range validIDs {
		if !id.IsValid() {
			t.Errorf("StateID(%d).IsValid() should be true", id)
		}
	}

	invalidIDs := []StateID{StateInvalid, StateID(50), StateID(99)}
	for _, id := range invalidIDs {
		if id.IsValid() {
			t.Errorf("StateID(%d).IsValid() should be false", id)
		}
	}
}

func TestStateIDIsGlobalState(t *testing.T) {
	globalStates := []StateID{
		StateMatchInit, StateRoundMiniGame, StateRoundPrep,
		StateTurnLoop, StateBossBattle, StateGameOver,
	}

	for _, id := range globalStates {
		if !id.IsGlobalState() {
			t.Errorf("StateID(%d).IsGlobalState() should be true", id)
		}
	}

	nonGlobalStates := []StateID{
		StateTurnUpkeep, StateMainAction, StateTurnMoving,
		StateTurnLanded, StateTurnEvent, StateTurnEnd,
		StateWaitDecision, StateNone,
	}

	for _, id := range nonGlobalStates {
		if id.IsGlobalState() {
			t.Errorf("StateID(%d).IsGlobalState() should be false", id)
		}
	}
}

func TestStateIDIsTurnState(t *testing.T) {
	turnStates := []StateID{
		StateTurnUpkeep, StateMainAction, StateTurnMoving,
		StateTurnLanded, StateTurnEvent, StateTurnEnd,
	}

	for _, id := range turnStates {
		if !id.IsTurnState() {
			t.Errorf("StateID(%d).IsTurnState() should be true", id)
		}
	}

	nonTurnStates := []StateID{
		StateMatchInit, StateTurnLoop, StateGameOver,
		StateWaitDecision, StateNone,
	}

	for _, id := range nonTurnStates {
		if id.IsTurnState() {
			t.Errorf("StateID(%d).IsTurnState() should be false", id)
		}
	}
}

func TestStateIDIsInterruptState(t *testing.T) {
	if !StateWaitDecision.IsInterruptState() {
		t.Error("StateWaitDecision.IsInterruptState() should be true")
	}

	nonInterruptStates := []StateID{
		StateMatchInit, StateTurnLoop, StateTurnUpkeep,
		StateNone, StateInvalid,
	}

	for _, id := range nonInterruptStates {
		if id.IsInterruptState() {
			t.Errorf("StateID(%d).IsInterruptState() should be false", id)
		}
	}
}

func TestStateIDLayer(t *testing.T) {
	tests := []struct {
		id       StateID
		expected int
	}{
		{StateMatchInit, 1},
		{StateTurnLoop, 1},
		{StateGameOver, 1},
		{StateTurnUpkeep, 2},
		{StateMainAction, 2},
		{StateTurnEnd, 2},
		{StateWaitDecision, 3},
		{StateNone, 0},
		{StateInvalid, 0},
	}

	for _, tt := range tests {
		if got := tt.id.Layer(); got != tt.expected {
			t.Errorf("StateID(%d).Layer() = %d, want %d", tt.id, got, tt.expected)
		}
	}
}