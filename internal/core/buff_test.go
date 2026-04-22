package core

import (
	"testing"

	"github.com/b1tAction/paradiced/pkg/constants"
	"github.com/b1tAction/paradiced/pkg/id"
)

func TestTickDurationFirstCallMarksEligible(t *testing.T) {
	// First TickDuration call on new buff: should NOT decrement, but mark tickEligible
	buff := NewBuff(constants.BuffTypeCurse, 3)

	active := buff.TickDuration()
	if !active {
		t.Error("buff should still be active")
	}
	if buff.Duration != 3 {
		t.Errorf("Duration = %d, expected 3 (not decremented on first call)", buff.Duration)
	}
	if !buff.tickEligible {
		t.Error("buff should be tickEligible=true after first TickDuration call")
	}
}

func TestTickDurationSecondCallDecrements(t *testing.T) {
	// Second TickDuration call: should decrement now
	buff := NewBuff(constants.BuffTypeCurse, 3)

	// First call: marks eligible, no decrement
	buff.TickDuration()

	// Second call: decrements
	active := buff.TickDuration()
	if !active {
		t.Error("buff should still be active after second tick")
	}
	if buff.Duration != 2 {
		t.Errorf("Duration = %d, expected 2 (decremented on second call)", buff.Duration)
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

func TestTickDurationExpiryAfterTwoCalls(t *testing.T) {
	// Duration=1: first call marks eligible, second call decrements to 0 → expires
	buff := NewBuff(constants.BuffTypeLost, 1)

	// First call: marks eligible, Duration stays 1
	active := buff.TickDuration()
	if !active {
		t.Error("buff should still be active after first call")
	}

	// Second call: Duration 1→0, expires
	active = buff.TickDuration()
	if active {
		t.Error("buff with Duration=1 should expire after second tick")
	}
	if buff.Duration != 0 {
		t.Errorf("Duration = %d, expected 0", buff.Duration)
	}
}

func TestNewBuffDefaultNotTickEligible(t *testing.T) {
	buff := NewBuff(constants.BuffTypeDivine, 3)
	if buff.tickEligible {
		t.Error("new buff should have tickEligible=false by default")
	}
}

func TestNewBuffWithIDDefaultNotTickEligible(t *testing.T) {
	buff := NewBuffWithID(constants.BuffTypeDivine, id.NewBuffID(), 3)
	if buff.tickEligible {
		t.Error("new buff with ID should have tickEligible=false by default")
	}
}