package hsm

import (
	"testing"
	"time"

	"github.com/b1tAction/Fated/internal/core"
	"github.com/b1tAction/Fated/internal/engine"
	"github.com/b1tAction/Fated/internal/gamemap"
	"github.com/b1tAction/Fated/pkg/event"
)

func TestNewStateContext(t *testing.T) {
	ctx := NewStateContext()

	if ctx.Metadata == nil {
		t.Error("Metadata should be initialized")
	}
	if ctx.Decisions == nil {
		t.Error("Decisions should be initialized")
	}
	if !ctx.Success {
		t.Error("Success should be true by default")
	}
}

func TestStateContextWithMethods(t *testing.T) {
	game := engine.NewGame("test", 0)
	ctx := NewStateContext().WithGame(game)
	if ctx.Game != game {
		t.Error("Game not set correctly")
	}
	if ctx.Bus == nil {
		t.Error("Bus adapter should be created from game")
	}

	// Test WithPlayer
	player := core.NewPlayer(core.PlayerConfig{UserID: "player-1"})
	ctx = ctx.WithPlayer(player)
	if ctx.Player != player {
		t.Error("Player not set correctly")
	}

	// Test WithPhase
	ctx = ctx.WithPhase(event.PhaseBeforeTurn, 10)
	if ctx.Phase != event.PhaseBeforeTurn {
		t.Error("Phase not set correctly")
	}
	if ctx.PhaseData != 10 {
		t.Error("PhaseData not set correctly")
	}

	// Test WithDiceSteps
	ctx = ctx.WithDiceSteps(6)
	if ctx.DiceSteps != 6 {
		t.Error("DiceSteps not set correctly")
	}

	// Test WithTimeout
	ctx = ctx.WithTimeout(30 * time.Second)
	if ctx.Timeout != 30*time.Second {
		t.Error("Timeout not set correctly")
	}

	// Test WithDecision
	decision := event.NewDecision("test", nil)
	ctx = ctx.WithDecision(decision)
	if ctx.Decision != decision {
		t.Error("Decision not set correctly")
	}
}

func TestStateContextMetadata(t *testing.T) {
	ctx := NewStateContext()

	// Test SetMetadata
	ctx.SetMetadata("key1", "value1")
	ctx.SetMetadata("key2", 123)

	// Test GetMetadata
	if ctx.GetMetadata("key1") != "value1" {
		t.Error("Metadata key1 not retrieved correctly")
	}
	if ctx.GetMetadata("key2") != 123 {
		t.Error("Metadata key2 not retrieved correctly")
	}
	if ctx.GetMetadata("nonexistent") != nil {
		t.Error("Nonexistent key should return nil")
	}
}

func TestStateContextClear(t *testing.T) {
	ctx := NewStateContext()
	ctx.SetMetadata("test", "value")
	ctx.DiceSteps = 10
	ctx.Success = false
	ctx.Error = nil // would be an error if set

	ctx.Clear()

	if ctx.Metadata["test"] != nil {
		t.Error("Metadata should be cleared")
	}
	if ctx.DiceSteps != 0 {
		t.Error("DiceSteps should be reset to 0")
	}
	if !ctx.Success {
		t.Error("Success should be reset to true")
	}
}

func TestStateContextWithBus(t *testing.T) {
	game := engine.NewGame("test", 0)

	// Test WithBus creates adapter
	ctx := NewStateContext().WithGame(game)
	if ctx.Bus == nil {
		t.Error("WithGame should create Bus adapter")
	}

	// Test WithBus can override
	mockBus := &mockEventBusAdapter{}
	ctx = NewStateContext().WithBus(mockBus)
	if ctx.Bus != mockBus {
		t.Error("WithBus should set adapter directly")
	}
}

func TestStateContextWithMapEngine(t *testing.T) {
	ctx := NewStateContext()

	// Test WithMapEngine
	mockEngine := &mockMapEngineAdapter{length: 100}
	ctx = ctx.WithMapEngine(mockEngine)
	if ctx.MapEngine != mockEngine {
		t.Error("WithMapEngine should set adapter")
	}
}

// ========== Mock Adapters for Testing ==========

type mockEventBusAdapter struct{}

func (m *mockEventBusAdapter) Publish(phase event.Phase, playerID string, ctx *event.Context) []*event.Decision {
	return nil
}
func (m *mockEventBusAdapter) Subscribe(phase event.Phase, ownerID, sourceID, sourceType string, decision *event.Decision) string {
	return ""
}
func (m *mockEventBusAdapter) Unsubscribe(subID string) bool { return false }
func (m *mockEventBusAdapter) UnsubscribeBySource(sourceID string) int { return 0 }
func (m *mockEventBusAdapter) UnsubscribeByOwner(ownerID string) int { return 0 }
func (m *mockEventBusAdapter) GetSubscriptionCount() int { return 0 }
func (m *mockEventBusAdapter) Clear() {}

type mockMapEngineAdapter struct {
	length int
}

func (m *mockMapEngineAdapter) GetLength() int { return m.length }
func (m *mockMapEngineAdapter) GetCell(pos int) (*gamemap.MapCell, error) { return nil, nil }
func (m *mockMapEngineAdapter) CalculatePath(startPos int, steps int) (*gamemap.PathResult, error) {
	return nil, nil
}
func (m *mockMapEngineAdapter) GetLastCheckpoint(pos int) int { return 0 }
func (m *mockMapEngineAdapter) SetCellType(pos int, cellType gamemap.CellType) error { return nil }
func (m *mockMapEngineAdapter) ActivateFog(pos int) error { return nil }
func (m *mockMapEngineAdapter) IsFogActivated(pos int) bool { return false }