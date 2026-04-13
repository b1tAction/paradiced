package hsm

import (
	"testing"
	"time"

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
	// Test WithGame
	game := &mockGameAdapter{}
	ctx := NewStateContext().WithGame(game)
	if ctx.Game != game {
		t.Error("Game not set correctly")
	}

	// Test WithPlayer
	player := &mockPlayerAdapter{id: "player-1"}
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

// ========== Mock Types for Testing ==========

type mockGameAdapter struct {
	id      string
	round   int
	turn    int
	players []*mockPlayerAdapter
}

func (m *mockGameAdapter) GetID() string            { return m.id }
func (m *mockGameAdapter) GetRound() int            { return m.round }
func (m *mockGameAdapter) GetTurn() int             { return m.turn }
func (m *mockGameAdapter) GetCurrentPhase() string  { return "" }
func (m *mockGameAdapter) SetRound(r int)           { m.round = r }
func (m *mockGameAdapter) SetTurn(t int)            { m.turn = t }
func (m *mockGameAdapter) SetWaiting(w bool)        {}
func (m *mockGameAdapter) GetPlayer(id string) PlayerAdapter {
	for _, p := range m.players {
		if p.id == id {
			return p
		}
	}
	return nil
}
func (m *mockGameAdapter) GetCurrentPlayer() PlayerAdapter {
	if len(m.players) > 0 {
		return m.players[0]
	}
	return nil
}
func (m *mockGameAdapter) GetAllPlayers() []PlayerAdapter { return nil }
func (m *mockGameAdapter) NextTurn()                       {}
func (m *mockGameAdapter) GetBus() EventBusAdapter         { return nil }
func (m *mockGameAdapter) PublishPhase(p event.Phase, pid string, ctx *StateContext) []*event.Decision {
	return nil
}
func (m *mockGameAdapter) SubscribeBuff(p PlayerAdapter, b BuffAdapter)     {}
func (m *mockGameAdapter) UnsubscribeBuff(b BuffAdapter)                     {}
func (m *mockGameAdapter) ApplyBuffToPlayer(p PlayerAdapter, b BuffAdapter) {}
func (m *mockGameAdapter) RemoveBuffFromPlayer(p PlayerAdapter, b BuffAdapter) {}
func (m *mockGameAdapter) SubscribeItem(p PlayerAdapter, i ItemAdapter)     {}
func (m *mockGameAdapter) UnsubscribeItem(i ItemAdapter)                     {}
func (m *mockGameAdapter) DrawEvent(lp int) EventAdapter                     { return nil }
func (m *mockGameAdapter) DrawItem(lp int) ItemAdapter                       { return nil }
func (m *mockGameAdapter) GetMapEngine() MapEngineAdapter                    { return nil }

type mockPlayerAdapter struct {
	id       string
	position int
	hp       int
	lp       int
	dead     bool
	skipTurn bool
}

func (m *mockPlayerAdapter) GetUserID() string     { return m.id }
func (m *mockPlayerAdapter) GetFaction() FactionAdapter { return nil }
func (m *mockPlayerAdapter) GetPosition() int      { return m.position }
func (m *mockPlayerAdapter) GetHP() int            { return m.hp }
func (m *mockPlayerAdapter) GetLP() int            { return m.lp }
func (m *mockPlayerAdapter) IsDead() bool          { return m.dead }
func (m *mockPlayerAdapter) CanAct() bool          { return !m.dead && !m.skipTurn }
func (m *mockPlayerAdapter) GetSkipTurn() bool     { return m.skipTurn }
func (m *mockPlayerAdapter) SetPosition(p int)     { m.position = p }
func (m *mockPlayerAdapter) SetHP(h int)           { m.hp = h }
func (m *mockPlayerAdapter) SetLP(l int)           { m.lp = l }
func (m *mockPlayerAdapter) SetSkipTurn(s bool)    { m.skipTurn = s }
func (m *mockPlayerAdapter) SetDead(d bool)        { m.dead = d }
func (m *mockPlayerAdapter) Move(p int, max int)   { m.position = p }
func (m *mockPlayerAdapter) ApplyDamage(a int)     { m.hp -= a }
func (m *mockPlayerAdapter) Heal(a int)            { m.hp += a }
func (m *mockPlayerAdapter) ModifyLP(a int)        { m.lp += a }
func (m *mockPlayerAdapter) AddBuff(b BuffAdapter) {}
func (m *mockPlayerAdapter) RemoveBuff(t BuffTypeAdapter) {}
func (m *mockPlayerAdapter) HasBuff(t BuffTypeAdapter) bool { return false }
func (m *mockPlayerAdapter) GetBuff(t BuffTypeAdapter) BuffAdapter { return nil }
func (m *mockPlayerAdapter) GetAllBuffs() []BuffAdapter { return nil }
func (m *mockPlayerAdapter) TickBuffs() []BuffAdapter { return nil }
func (m *mockPlayerAdapter) ClearNegativeBuffs() int { return 0 }
func (m *mockPlayerAdapter) AddItem(i ItemAdapter) {}
func (m *mockPlayerAdapter) RemoveItem(id string) ItemAdapter { return nil }
func (m *mockPlayerAdapter) GetItem(id string) ItemAdapter { return nil }
func (m *mockPlayerAdapter) HasItem(t ItemTypeAdapter) bool { return false }
func (m *mockPlayerAdapter) GetAllItems() []ItemAdapter { return nil }
func (m *mockPlayerAdapter) GetChargeCount() int { return 0 }
func (m *mockPlayerAdapter) SetChargeCount(c int) {}
func (m *mockPlayerAdapter) IncrementChargeCount() int { return 0 }
func (m *mockPlayerAdapter) GetFireCounter() int { return 0 }
func (m *mockPlayerAdapter) SetFireCounter(c int) {}
func (m *mockPlayerAdapter) IncrementFireCounter() int { return 0 }
func (m *mockPlayerAdapter) Respawn(p int) { m.position = p; m.dead = false }
func (m *mockPlayerAdapter) Clone() PlayerAdapter { return m }