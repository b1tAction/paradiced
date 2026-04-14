package hsm

import (
	"testing"
	"time"

	"github.com/b1tAction/fated/internal/core"
	"github.com/b1tAction/fated/internal/engine"
	"github.com/b1tAction/fated/internal/gamemap"
	"github.com/b1tAction/fated/pkg/constants"
	"github.com/b1tAction/fated/pkg/event"
	"github.com/b1tAction/fated/pkg/rng"
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
	ctx = ctx.WithPhase(constants.PhaseBeforeTurn, 10)
	if ctx.Phase != constants.PhaseBeforeTurn {
		t.Error("Phase not set correctly")
	}
	if ctx.PhaseData != 10 {
		t.Error("PhaseData not set correctly")
	}

	// Test WithDiceSteps (stored in Metadata)
	ctx = ctx.WithDiceSteps(6)
	if ctx.GetDiceSteps() != 6 {
		t.Error("DiceSteps not set correctly")
	}

	// Test WithTargetPos (stored in Metadata)
	ctx = ctx.WithTargetPos(20)
	if ctx.GetTargetPos() != 20 {
		t.Error("TargetPos not set correctly")
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

func TestStateContextMetadataMethods(t *testing.T) {
	ctx := NewStateContext()

	// Test Set/Get
	ctx.Set("key1", "value1")
	ctx.SetInt("key2", 123)
	ctx.SetBool("key3", true)

	// Test Get with default
	if ctx.GetStringOrDefault("key1", "") != "value1" {
		t.Error("key1 not retrieved correctly")
	}
	if ctx.GetIntOrDefault("key2", 0) != 123 {
		t.Error("key2 not retrieved correctly")
	}
	if !ctx.GetBoolOrDefault("key3", false) {
		t.Error("key3 not retrieved correctly")
	}
}

func TestStateContextStateMarkers(t *testing.T) {
	ctx := NewStateContext()

	// Test SkipTurn
	if ctx.IsSkipTurn() {
		t.Error("IsSkipTurn should be false by default")
	}
	ctx.SetSkipTurn(true)
	if !ctx.IsSkipTurn() {
		t.Error("IsSkipTurn should be true after SetSkipTurn(true)")
	}

	// Test FellDown
	if ctx.IsFellDown() {
		t.Error("IsFellDown should be false by default")
	}
	ctx.SetFellDown(true)
	if !ctx.IsFellDown() {
		t.Error("IsFellDown should be true after SetFellDown(true)")
	}

	// Test ReachedEnd
	if ctx.HasReachedEnd() {
		t.Error("HasReachedEnd should be false by default")
	}
	ctx.SetReachedEnd(true)
	if !ctx.HasReachedEnd() {
		t.Error("HasReachedEnd should be true after SetReachedEnd(true)")
	}
}

func TestStateContextMiniGameMethods(t *testing.T) {
	ctx := NewStateContext()

	// Test mini-game rank
	ctx.SetMiniGameRank("p1", 1)
	ctx.SetMiniGameRank("p2", 2)
	ctx.SetMiniGameRank("p3", 3)

	if ctx.GetMiniGameRank("p1") != 1 {
		t.Error("p1 rank should be 1")
	}
	if ctx.GetMiniGameRank("p2") != 2 {
		t.Error("p2 rank should be 2")
	}
	if ctx.GetMiniGameRank("p3") != 3 {
		t.Error("p3 rank should be 3")
	}
	if ctx.GetMiniGameRank("unknown") != 0 {
		t.Error("unknown player rank should default to 0")
	}

	// Test dice type
	ctx.SetDiceType("p1", rng.DiceTypeGold)
	ctx.SetDiceType("p2", rng.DiceTypeSilver)

	if ctx.GetDiceType("p1") != rng.DiceTypeGold {
		t.Error("p1 dice should be gold")
	}
	if ctx.GetDiceType("p2") != rng.DiceTypeSilver {
		t.Error("p2 dice should be silver")
	}
	if ctx.GetDiceType("unknown") != rng.DiceTypeWood {
		t.Error("unknown player dice should default to wood")
	}
}

func TestStateContextClear(t *testing.T) {
	ctx := NewStateContext()
	ctx.SetInt(KeyDiceSteps, 10)
	ctx.SetBool(KeySkipTurn, true)
	ctx.SetBool(KeyFellDown, true)
	ctx.Success = false

	ctx.Clear()

	if ctx.GetDiceSteps() != 0 {
		t.Error("DiceSteps should be reset to 0")
	}
	if ctx.IsSkipTurn() {
		t.Error("SkipTurn should be reset to false")
	}
	if ctx.IsFellDown() {
		t.Error("FellDown should be reset to false")
	}
	if !ctx.Success {
		t.Error("Success should be reset to true")
	}
}

func TestStateContextWithBus(t *testing.T) {
	game := engine.NewGame("test", 0)

	// Test WithGame creates adapter
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

func (m *mockEventBusAdapter) Publish(phase constants.Phase, playerID string, ctx *event.Context) []*event.Decision {
	return nil
}
func (m *mockEventBusAdapter) Subscribe(phase constants.Phase, ownerID, sourceID, sourceType string, decision *event.Decision) string {
	return ""
}
func (m *mockEventBusAdapter) Unsubscribe(subID string) bool           { return false }
func (m *mockEventBusAdapter) UnsubscribeBySource(sourceID string) int { return 0 }
func (m *mockEventBusAdapter) UnsubscribeByOwner(ownerID string) int   { return 0 }
func (m *mockEventBusAdapter) GetSubscriptionCount() int               { return 0 }
func (m *mockEventBusAdapter) Clear()                                  {}

type mockMapEngineAdapter struct {
	length int
}

func (m *mockMapEngineAdapter) GetLength() int                            { return m.length }
func (m *mockMapEngineAdapter) GetCell(pos int) (*gamemap.MapCell, error) { return nil, nil }
func (m *mockMapEngineAdapter) CalculatePath(startPos int, steps int) (*gamemap.PathResult, error) {
	return nil, nil
}
func (m *mockMapEngineAdapter) GetLastCheckpoint(pos int) int                        { return 0 }
func (m *mockMapEngineAdapter) SetCellType(pos int, cellType gamemap.CellType) error { return nil }
func (m *mockMapEngineAdapter) ActivateFog(pos int) error                            { return nil }
func (m *mockMapEngineAdapter) IsFogActivated(pos int) bool                          { return false }
