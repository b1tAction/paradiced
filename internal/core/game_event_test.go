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