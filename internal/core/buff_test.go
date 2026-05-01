package core

import (
	"testing"

	"github.com/b1tAction/paradiced/pkg/constants"
	"github.com/b1tAction/paradiced/pkg/id"
)

func TestTickDurationNotEligibleNoDecrement(t *testing.T) {
	// Buff with tickEligible=false: TickDuration should NOT decrement
	buff := NewBuff(constants.BuffTypeCurse, 3)

	active := buff.TickDuration()
	if !active {
		t.Error("buff should still be active")
	}
	if buff.Duration != 3 {
		t.Errorf("Duration = %d, expected 3 (not decremented when not eligible)", buff.Duration)
	}
}

func TestTickDurationEligibleDecrements(t *testing.T) {
	// Buff with tickEligible=true: TickDuration should decrement
	buff := NewBuff(constants.BuffTypeCurse, 3)
	buff.MarkTickEligible()

	active := buff.TickDuration()
	if !active {
		t.Error("buff should still be active after tick")
	}
	if buff.Duration != 2 {
		t.Errorf("Duration = %d, expected 2 (decremented when eligible)", buff.Duration)
	}
}

func TestTickDurationPermanentNotAffected(t *testing.T) {
	// Permanent buffs (Duration=-1) are unaffected
	buff := NewBuff(constants.BuffTypeFire, -1)

	active := buff.TickDuration()
	if !active {
		t.Error("permanent buff should always be active")
	}
	if buff.Duration != -1 {
		t.Errorf("Duration = %d, expected -1 (permanent)", buff.Duration)
	}
}

func TestTickDurationExpiryWhenEligible(t *testing.T) {
	// Duration=1 with tickEligible=true: TickDuration decrements to 0 → expires
	buff := NewBuff(constants.BuffTypeLost, 1)
	buff.MarkTickEligible()

	active := buff.TickDuration()
	if active {
		t.Error("buff with Duration=1 should expire after tick when eligible")
	}
	if buff.Duration != 0 {
		t.Errorf("Duration = %d, expected 0", buff.Duration)
	}
}

func TestNewBuffDefaultNotTickEligible(t *testing.T) {
	buff := NewBuff(constants.BuffTypeDivine, 3)
	if buff.TickEligible() {
		t.Error("new buff should have tickEligible=false by default")
	}
}

func TestNewBuffWithIDDefaultNotTickEligible(t *testing.T) {
	buff := NewBuffWithID(constants.BuffTypeDivine, id.NewBuffID(), 3)
	if buff.TickEligible() {
		t.Error("new buff with ID should have tickEligible=false by default")
	}
}

func TestMarkTickEligible(t *testing.T) {
	buff := NewBuff(constants.BuffTypeCurse, 3)
	if buff.TickEligible() {
		t.Error("new buff should not be tickEligible by default")
	}
	buff.MarkTickEligible()
	if !buff.TickEligible() {
		t.Error("buff should be tickEligible after MarkTickEligible()")
	}
}

func TestBuffDefinitionIsPositive(t *testing.T) {
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
		def := &BuffDefinition{Type: constants.BuffTypeDivine, Eval: tt.eval}
		if def.IsPositive() != tt.expected {
			t.Errorf("BuffDefinition.IsPositive() with Eval=%d = %v, expected %v", tt.eval, def.IsPositive(), tt.expected)
		}
	}
}

func TestBuffDefinitionIsNegative(t *testing.T) {
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
		def := &BuffDefinition{Type: constants.BuffTypeCurse, Eval: tt.eval}
		if def.IsNegative() != tt.expected {
			t.Errorf("BuffDefinition.IsNegative() with Eval=%d = %v, expected %v", tt.eval, def.IsNegative(), tt.expected)
		}
	}
}