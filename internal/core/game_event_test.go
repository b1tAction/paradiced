package core

import (
	"testing"

	"github.com/b1tAction/paradiced/pkg/constants"
)

func TestNewGameEvent(t *testing.T) {
	event := NewGameEvent(constants.EventTypeHerb)
	if event.Type != constants.EventTypeHerb {
		t.Errorf("GameEvent.Type = %s, expected herb", event.Type)
	}
	if event.ID.UUID() == "" {
		t.Error("GameEvent.ID should have a valid UUID")
	}
}

func TestEventDefinitionIsGood(t *testing.T) {
	tests := []struct {
		eval     constants.Evaluation
		expected bool
	}{
		{constants.EvaluationGood, true},
		{constants.EvaluationVeryGood, true},
		{constants.EvaluationMildGood, true},
		{constants.EvaluationNeutral, false},
		{constants.EvaluationBad, false},
	}
	for _, tt := range tests {
		def := &EventDefinition{Type: constants.EventTypeHerb, Eval: tt.eval}
		if def.IsGood() != tt.expected {
			t.Errorf("EventDefinition.IsGood() with Eval=%d = %v, expected %v", tt.eval, def.IsGood(), tt.expected)
		}
	}
}

func TestEventDefinitionIsBad(t *testing.T) {
	tests := []struct {
		eval     constants.Evaluation
		expected bool
	}{
		{constants.EvaluationBad, true},
		{constants.EvaluationVeryBad, true},
		{constants.EvaluationMildBad, true},
		{constants.EvaluationNeutral, false},
		{constants.EvaluationGood, false},
	}
	for _, tt := range tests {
		def := &EventDefinition{Type: constants.EventTypeMosquito, Eval: tt.eval}
		if def.IsBad() != tt.expected {
			t.Errorf("EventDefinition.IsBad() with Eval=%d = %v, expected %v", tt.eval, def.IsBad(), tt.expected)
		}
	}
}

func TestEventDefinitionIsNeutral(t *testing.T) {
	tests := []struct {
		eval     constants.Evaluation
		expected bool
	}{
		{constants.EvaluationNeutral, true},
		{constants.EvaluationMixed, true},
		{constants.EvaluationGood, false},
		{constants.EvaluationBad, false},
	}
	for _, tt := range tests {
		def := &EventDefinition{Type: constants.EventTypeExchange, Eval: tt.eval}
		if def.IsNeutral() != tt.expected {
			t.Errorf("EventDefinition.IsNeutral() with Eval=%d = %v, expected %v", tt.eval, def.IsNeutral(), tt.expected)
		}
	}
}