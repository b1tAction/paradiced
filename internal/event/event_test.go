package event

import (
	"testing"
	"time"

	"github.com/b1tAction/paradiced/internal/core"
	"github.com/b1tAction/paradiced/pkg/constants"
	"github.com/b1tAction/paradiced/pkg/id"
)

// ========== Phase Tests ==========

func TestPhaseIsValid(t *testing.T) {
	validPhases := []constants.Phase{
		constants.PhaseBeforeTurn, constants.PhasePreMove, constants.PhaseOnLand, constants.PhasePreEvent,
		constants.PhasePreDamage, constants.PhaseAfterTurn, constants.PhaseAnyTime,
		constants.PhasePreBuffApplied, constants.PhasePostBuffApplied,
		constants.PhasePreBuffRemoved, constants.PhasePostBuffRemoved,
	}
	for _, p := range validPhases {
		if !p.IsValid() {
			t.Errorf("Phase(%s).IsValid() should be true", p)
		}
	}

	invalidPhases := []constants.Phase{constants.Phase(""), constants.Phase("invalid")}
	for _, p := range invalidPhases {
		if p.IsValid() {
			t.Errorf("Phase(%s).IsValid() should be false", p)
		}
	}
}

func TestPhaseNeedsSubscription(t *testing.T) {
	// AnyTime does not need subscription (triggered manually)
	if constants.PhaseAnyTime.NeedsSubscription() {
		t.Error("PhaseAnyTime should not need subscription")
	}

	// Other phases need subscription
	needsSub := []constants.Phase{
		constants.PhaseBeforeTurn, constants.PhasePreMove, constants.PhaseOnLand, constants.PhasePreEvent,
		constants.PhasePreDamage, constants.PhaseAfterTurn,
		constants.PhasePreBuffApplied, constants.PhasePostBuffApplied,
		constants.PhasePreBuffRemoved, constants.PhasePostBuffRemoved,
	}
	for _, p := range needsSub {
		if !p.NeedsSubscription() {
			t.Errorf("Phase(%s) should need subscription", p)
		}
	}
}

func TestPhaseIsHSMPublished(t *testing.T) {
	hsmPhases := []constants.Phase{
		constants.PhaseBeforeTurn, constants.PhaseOnLand, constants.PhaseAfterTurn,
	}
	for _, p := range hsmPhases {
		if !p.IsHSMPublished() {
			t.Errorf("Phase(%s) should be HSM published", p)
		}
	}
}

func TestPhaseIsActionPublished(t *testing.T) {
	actionPhases := []constants.Phase{
		constants.PhasePreDamage, constants.PhasePreEvent, constants.PhasePreMove,
		constants.PhasePreRespawn,
		constants.PhasePreBuffApplied, constants.PhasePostBuffApplied,
		constants.PhasePreBuffRemoved, constants.PhasePostBuffRemoved,
	}
	for _, p := range actionPhases {
		if !p.IsActionPublished() {
			t.Errorf("Phase(%s) should be Action published", p)
		}
	}
}

// ========== Decision Tests ==========

func TestNewDecision(t *testing.T) {
	options := []Option{
		{ID: "yes", Label: "是"},
		{ID: "no", Label: "否"},
	}
	d := NewDecision("是否使用护盾？", options)

	if d.Prompt != "是否使用护盾？" {
		t.Errorf("Prompt = %s, expected '是否使用护盾？'", d.Prompt)
	}
	if len(d.Options) != 2 {
		t.Errorf("Options count = %d, expected 2", len(d.Options))
	}
	if d.ID.IsZero() {
		t.Error("ID should not be empty")
	}
}

func TestDecisionWithPriority(t *testing.T) {
	d := NewDecision("test", []Option{})
	d.WithPriority(100)

	if d.Priority != 100 {
		t.Errorf("Priority = %d, expected 100", d.Priority)
	}
}

func TestDecisionWithTimeout(t *testing.T) {
	d := NewDecision("test", []Option{
		{ID: "a", Label: "A"},
		{ID: "b", Label: "B"},
	})
	d.WithTimeout(30*time.Second, 1)

	if d.Timeout != 30*time.Second {
		t.Errorf("Timeout = %v, expected 30s", d.Timeout)
	}
	if d.Default != 1 {
		t.Errorf("Default = %d, expected 1", d.Default)
	}
}

func TestDecisionShouldAsk(t *testing.T) {
	// No condition, always ask
	d1 := NewDecision("test", []Option{})
	if !d1.ShouldAsk() {
		t.Error("Decision without condition should always ask")
	}

	// With condition, ask when true
	d2 := NewDecision("test", []Option{})
	d2.WithCondition(func() bool { return true })
	if !d2.ShouldAsk() {
		t.Error("Decision with true condition should ask")
	}

	// Do not ask when condition is false
	d3 := NewDecision("test", []Option{})
	d3.WithCondition(func() bool { return false })
	if d3.ShouldAsk() {
		t.Error("Decision with false condition should not ask")
	}
}

func TestDecisionExecute(t *testing.T) {
	executed := false
	choice := -1

	d := NewDecision("test", []Option{
		{ID: "yes", Label: "是", Action: func(ctx *Context) { executed = true }},
		{ID: "no", Label: "否"},
	})
	d.WithOnChoice(func(c int, ctx *Context) { choice = c })

	ctx := NewContext(nil)
	d.Execute(0, ctx)

	if !executed {
		t.Error("Option action should be executed")
	}
	if choice != 0 {
		t.Errorf("OnChoice callback choice = %d, expected 0", choice)
	}
}

func TestDecisionExecuteDefault(t *testing.T) {
	executed := false

	d := NewDecision("test", []Option{
		{ID: "a", Label: "A"},
		{ID: "b", Label: "B", Action: func(ctx *Context) { executed = true }},
	})
	d.WithTimeout(10*time.Second, 1) // Default option 1

	ctx := NewContext(nil)
	d.Execute(-1, ctx) // Invalid choice, use default

	if !executed {
		t.Error("Default option should be executed for invalid choice")
	}
}

func TestDecisionClone(t *testing.T) {
	d := NewDecision("test", []Option{
		{ID: "yes", Label: "是"},
	})
	d.WithPriority(50)
	d.WithSource("buff-001", "buff")

	cloned := d.Clone()

	if cloned.ID == d.ID {
		t.Error("Cloned Decision should have new ID")
	}
	if cloned.Priority != d.Priority {
		t.Errorf("Cloned Priority = %d, expected %d", cloned.Priority, d.Priority)
	}
	if cloned.SourceID != "" {
		t.Error("Cloned SourceID should be empty")
	}
}

func TestDecisionBuilder(t *testing.T) {
	d := NewDecisionBuilder("是否使用道具？").
		AddOption("use", "使用", func(ctx *Context) {}).
		AddOption("skip", "跳过", func(ctx *Context) {}).
		SetPriority(100).
		SetTimeout(30*time.Second, 1).
		SetSource("item-001", "item").
		Build()

	if d.Prompt != "是否使用道具？" {
		t.Errorf("Prompt = %s", d.Prompt)
	}
	if len(d.Options) != 2 {
		t.Errorf("Options count = %d", len(d.Options))
	}
	if d.Priority != 100 {
		t.Errorf("Priority = %d", d.Priority)
	}
	if d.SourceID != "item-001" {
		t.Errorf("SourceID = %s", d.SourceID)
	}
}

// ========== EventBus Tests ==========

func TestNewEventBus(t *testing.T) {
	bus := NewEventBus("game-001")
	if bus == nil {
		t.Fatal("NewEventBus should not return nil")
	}
	if bus.GameID != "game-001" {
		t.Errorf("GameID = %s, expected game-001", bus.GameID)
	}
	if bus.GetSubscriptionCount() != 0 {
		t.Errorf("Initial subscription count should be 0")
	}
}

func TestEventBusSubscribe(t *testing.T) {
	bus := NewEventBus("game-001")
	d := NewDecision("test", []Option{{ID: "yes", Label: "是"}})

	playerID := id.NewPlayerID()
	subID := bus.Subscribe(constants.PhasePreDamage, playerID, "buff-001", "buff", d)
	if subID.IsZero() {
		t.Error("Subscribe should return subscription ID")
	}

	if bus.GetSubscriptionCount() != 1 {
		t.Errorf("Subscription count = %d, expected 1", bus.GetSubscriptionCount())
	}

	subs := bus.GetSubscriptions(constants.PhasePreDamage)
	if len(subs) != 1 {
		t.Errorf("PreDamage subscriptions = %d, expected 1", len(subs))
	}
	if subs[0].OwnerID != playerID.UUID() {
		t.Errorf("OwnerID = %s, expected %s", subs[0].OwnerID, playerID.UUID())
	}
}

func TestEventBusUnsubscribe(t *testing.T) {
	bus := NewEventBus("game-001")
	d := NewDecision("test", []Option{})

	playerID := id.NewPlayerID()
	subID := bus.Subscribe(constants.PhasePreDamage, playerID, "buff-001", "buff", d)

	// Unsubscribe
	ok := bus.Unsubscribe(subID)
	if !ok {
		t.Error("Unsubscribe should return true")
	}

	if bus.GetSubscriptionCount() != 0 {
		t.Errorf("Subscription count should be 0 after unsubscribe")
	}

	// Unsubscribe again should return false
	ok = bus.Unsubscribe(subID)
	if ok {
		t.Error("Unsubscribe non-existent should return false")
	}
}

func TestEventBusUnsubscribeBySource(t *testing.T) {
	bus := NewEventBus("game-001")
	d := NewDecision("test", []Option{})

	playerID1 := id.NewPlayerID()
	playerID2 := id.NewPlayerID()
	bus.Subscribe(constants.PhasePreDamage, playerID1, "buff-001", "buff", d)
	bus.Subscribe(constants.PhasePreEvent, playerID1, "buff-001", "buff", d)
	bus.Subscribe(constants.PhasePreDamage, playerID2, "buff-002", "buff", d)

	// Remove all subscriptions for buff-001
	count := bus.UnsubscribeBySource("buff-001")
	if count != 2 {
		t.Errorf("Removed count = %d, expected 2", count)
	}

	if bus.GetSubscriptionCount() != 1 {
		t.Errorf("Remaining count should be 1")
	}
}

func TestEventBusUnsubscribeByOwner(t *testing.T) {
	bus := NewEventBus("game-001")
	d := NewDecision("test", []Option{})

	playerID1 := id.NewPlayerID()
	playerID2 := id.NewPlayerID()
	bus.Subscribe(constants.PhasePreDamage, playerID1, "buff-001", "buff", d)
	bus.Subscribe(constants.PhasePreEvent, playerID1, "buff-002", "buff", d)
	bus.Subscribe(constants.PhasePreDamage, playerID2, "buff-003", "buff", d)

	// Remove all subscriptions for playerID1
	count := bus.UnsubscribeByOwner(playerID1.UUID())
	if count != 2 {
		t.Errorf("Removed count = %d, expected 2", count)
	}

	if bus.GetSubscriptionCount() != 1 {
		t.Errorf("Remaining count should be 1")
	}
}

func TestEventBusPublish(t *testing.T) {
	bus := NewEventBus("game-001")

	// Subscribe auto-executing Decision
	d1 := NewAutoDecision("auto", []Option{{ID: "ok", Label: "OK"}})
	d1.WithPriority(100) // High priority

	// Subscribe Decision that needs confirmation
	d2 := NewDecision("是否使用护盾？", []Option{
		{ID: "use", Label: "使用"},
		{ID: "skip", Label: "跳过"},
	})
	d2.WithPriority(50) // Low priority

	playerID := id.NewPlayerID()
	bus.Subscribe(constants.PhasePreDamage, playerID, "buff-001", "buff", d1)
	bus.Subscribe(constants.PhasePreDamage, playerID, "item-001", "item", d2)

	// Publish
	ctx := NewContext(nil)
	decisions := bus.Publish(constants.PhasePreDamage, playerID.UUID(), ctx)

	// Only return Decisions that need confirmation
	if len(decisions) != 1 {
		t.Errorf("Decisions count = %d, expected 1 (only NeedConfirm)", len(decisions))
	}
	if decisions[0].Prompt != "是否使用护盾？" {
		t.Errorf("Decision Prompt = %s, expected '是否使用护盾？'", decisions[0].Prompt)
	}
}

func TestEventBusPublishPriorityOrder(t *testing.T) {
	bus := NewEventBus("game-001")

	// Create two Decisions with different priorities
	d1 := NewDecision("高优先级", []Option{{ID: "a", Label: "A"}})
	d1.WithPriority(100)

	d2 := NewDecision("低优先级", []Option{{ID: "b", Label: "B"}})
	d2.WithPriority(50)

	playerID := id.NewPlayerID()
	bus.Subscribe(constants.PhasePreEvent, playerID, "buff-001", "buff", d1)
	bus.Subscribe(constants.PhasePreEvent, playerID, "buff-002", "buff", d2)

	ctx := NewContext(nil)
	decisions := bus.Publish(constants.PhasePreEvent, playerID.UUID(), ctx)

	if len(decisions) != 2 {
		t.Errorf("Decisions count = %d, expected 2", len(decisions))
	}

	// Check order: high priority first
	if decisions[0].Priority != 100 {
		t.Errorf("First decision priority = %d, expected 100", decisions[0].Priority)
	}
	if decisions[1].Priority != 50 {
		t.Errorf("Second decision priority = %d, expected 50", decisions[1].Priority)
	}
}

func TestEventBusPublishFilterOwner(t *testing.T) {
	bus := NewEventBus("game-001")
	d := NewDecision("test", []Option{{ID: "ok", Label: "OK"}})

	playerID1 := id.NewPlayerID()
	playerID2 := id.NewPlayerID()
	bus.Subscribe(constants.PhasePreDamage, playerID1, "buff-001", "buff", d)
	bus.Subscribe(constants.PhasePreDamage, playerID2, "buff-002", "buff", d)

	ctx := NewContext(nil)
	decisions := bus.Publish(constants.PhasePreDamage, playerID1.UUID(), ctx)

	// Only return playerID1's Decision
	if len(decisions) != 1 {
		t.Errorf("Decisions count = %d, expected 1 (filtered by owner)", len(decisions))
	}
}

func TestEventBusClear(t *testing.T) {
	bus := NewEventBus("game-001")
	d := NewDecision("test", []Option{})

	playerID1 := id.NewPlayerID()
	playerID2 := id.NewPlayerID()
	bus.Subscribe(constants.PhasePreDamage, playerID1, "buff-001", "buff", d)
	bus.Subscribe(constants.PhasePreEvent, playerID2, "buff-002", "buff", d)

	bus.Clear()

	if bus.GetSubscriptionCount() != 0 {
		t.Errorf("Subscription count should be 0 after clear")
	}
}

// ========== Context Tests ==========

func TestNewContext(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	ctx := NewContext(player)

	if ctx.Player == nil {
		t.Error("Player should not be nil")
	}
	if ctx.Player != player {
		t.Error("Player should match input")
	}
}

func TestNewContextNilPlayer(t *testing.T) {
	ctx := NewContext(nil)

	if ctx.Player != nil {
		t.Error("Player should be nil")
	}
}

func TestContextWithEvent(t *testing.T) {
	ctx := NewContext(nil)
	ctx.WithEvent("some_event")

	if ctx.GameEvent == nil {
		t.Error("GameEvent should not be nil")
	}
}

func TestContextWithState(t *testing.T) {
	state := &GameState{
		Round:        1,
		Turn:         2,
		CurrentPhase: constants.PhaseBeforeTurn,
	}

	ctx := NewContext(nil).WithState(state)

	if ctx.GameState == nil {
		t.Error("GameState should not be nil")
	}
	if ctx.GameState.Round != 1 {
		t.Errorf("Round = %d, expected 1", ctx.GameState.Round)
	}
}

func TestContextWithData(t *testing.T) {
	ctx := NewContext(nil)
	ctx.WithData(100)

	// Use GetData() backward-compatible method
	if ctx.GetData() != 100 {
		t.Errorf("GetData() = %v, expected 100", ctx.GetData())
	}

	// Use GetIntOrDefault for type-safe retrieval
	if ctx.GetIntOrDefault("data", 0) != 100 {
		t.Errorf("GetIntOrDefault(\"data\") = %d, expected 100", ctx.GetIntOrDefault("data", 0))
	}
}

func TestContextMetadata(t *testing.T) {
	ctx := NewContext(nil)

	// Test direct use of Metadata methods
	ctx.SetInt("damage", 50)
	ctx.SetString("element", "fire")
	ctx.SetBool("blocked", true)

	// Use GetIntOrDefault/GetStringOrDefault/GetBoolOrDefault
	if ctx.GetIntOrDefault("damage", 0) != 50 {
		t.Errorf("GetIntOrDefault(\"damage\") = %d, expected 50", ctx.GetIntOrDefault("damage", 0))
	}
	if ctx.GetStringOrDefault("element", "") != "fire" {
		t.Errorf("GetStringOrDefault(\"element\") = %s, expected fire", ctx.GetStringOrDefault("element", ""))
	}
	if !ctx.GetBoolOrDefault("blocked", false) {
		t.Error("GetBoolOrDefault(\"blocked\") should be true")
	}

	// Test chained calls
	ctx.SetInt("count", 1).SetString("name", "test")
	if ctx.GetIntOrDefault("count", 0) != 1 {
		t.Errorf("GetIntOrDefault(\"count\") = %d, expected 1", ctx.GetIntOrDefault("count", 0))
	}
}

func TestContextClone(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	ctx := NewContext(player)
	ctx.SetInt("damage", 100)
	ctx.SetString("source", "fire")

	cloned := ctx.Clone()

	// Modify clone doesn't affect original
	cloned.SetInt("damage", 50)

	if ctx.GetIntOrDefault("damage", 0) != 100 {
		t.Errorf("original damage = %d, expected 100", ctx.GetIntOrDefault("damage", 0))
	}
	if cloned.GetIntOrDefault("damage", 0) != 50 {
		t.Errorf("cloned damage = %d, expected 50", cloned.GetIntOrDefault("damage", 0))
	}
}

func TestContextDerivedActions(t *testing.T) {
	ctx := NewContext(nil)

	// Add derived actions
	ctx.AddDerivedAction("action1")
	ctx.AddDerivedAction("action2")

	// Get derived actions
	actions := ctx.GetDerivedActions()
	if len(actions) != 2 {
		t.Errorf("DerivedActions count = %d, expected 2", len(actions))
	}

	// Clear derived actions
	ctx.ClearDerivedActions()
	if len(ctx.GetDerivedActions()) != 0 {
		t.Error("DerivedActions should be empty after clear")
	}
}

func TestContextWithChoice(t *testing.T) {
	ctx := NewContext(nil).WithChoice(2)

	if ctx.Choice != 2 {
		t.Errorf("Choice = %d, expected 2", ctx.Choice)
	}
}

func TestContextClear(t *testing.T) {
	ctx := NewContext(nil)
	ctx.SetInt("damage", 50)
	ctx.AddDerivedAction("action1")

	ctx.Clear()

	// Check metadata is cleared by trying to get the value
	if ctx.GetIntOrDefault("damage", 0) != 0 {
		t.Error("Metadata should be empty after Clear")
	}
	if len(ctx.DerivedActions) != 0 {
		t.Error("DerivedActions should be empty after Clear")
	}
}
