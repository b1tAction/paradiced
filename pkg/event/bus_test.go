package event

import (
	"testing"
	"time"
)

// TestPlayer 用于测试的简单玩家结构
type TestPlayer struct {
	UserID string
}

// ========== Phase Tests ==========

func TestPhaseString(t *testing.T) {
	tests := []struct {
		phase    Phase
		expected string
	}{
		{PhaseBeforeTurn, "BeforeTurn"},
		{PhaseOnMove, "OnMove"},
		{PhaseOnLand, "OnLand"},
		{PhasePreEvent, "PreEvent"},
		{PhasePreDamage, "PreDamage"},
		{PhaseAfterTurn, "AfterTurn"},
		{PhaseAnyTime, "AnyTime"},
		{PhasePassive, "Passive"},
		{Phase(999), "Unknown"},
	}

	for _, tt := range tests {
		result := tt.phase.String()
		if result != tt.expected {
			t.Errorf("Phase(%d).String() = %s, expected %s", tt.phase, result, tt.expected)
		}
	}
}

func TestPhaseIsValid(t *testing.T) {
	validPhases := []Phase{
		PhaseBeforeTurn, PhaseOnMove, PhaseOnLand, PhasePreEvent,
		PhasePreDamage, PhaseAfterTurn, PhaseAnyTime, PhasePassive,
	}
	for _, p := range validPhases {
		if !p.IsValid() {
			t.Errorf("Phase(%d).IsValid() should be true", p)
		}
	}

	invalidPhases := []Phase{Phase(-1), Phase(100)}
	for _, p := range invalidPhases {
		if p.IsValid() {
			t.Errorf("Phase(%d).IsValid() should be false", p)
		}
	}
}

func TestPhaseNeedsSubscription(t *testing.T) {
	// Passive和AnyTime不需要订阅
	if PhasePassive.NeedsSubscription() {
		t.Error("PhasePassive should not need subscription")
	}
	if PhaseAnyTime.NeedsSubscription() {
		t.Error("PhaseAnyTime should not need subscription")
	}

	// 其他Phase需要订阅
	needsSub := []Phase{
		PhaseBeforeTurn, PhaseOnMove, PhaseOnLand, PhasePreEvent,
		PhasePreDamage, PhaseAfterTurn,
	}
	for _, p := range needsSub {
		if !p.NeedsSubscription() {
			t.Errorf("Phase(%s) should need subscription", p.String())
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
	if d.ID == "" {
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
	// 无条件，总是询问
	d1 := NewDecision("test", []Option{})
	if !d1.ShouldAsk() {
		t.Error("Decision without condition should always ask")
	}

	// 有条件，条件为true时询问
	d2 := NewDecision("test", []Option{})
	d2.WithCondition(func() bool { return true })
	if !d2.ShouldAsk() {
		t.Error("Decision with true condition should ask")
	}

	// 条件为false时不询问
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
	d.WithTimeout(10*time.Second, 1) // 默认选项1

	ctx := NewContext(nil)
	d.Execute(-1, ctx) // 无效选项，使用默认

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

	subID := bus.Subscribe(PhasePreDamage, "player-001", "buff-001", "buff", d)
	if subID == "" {
		t.Error("Subscribe should return subscription ID")
	}

	if bus.GetSubscriptionCount() != 1 {
		t.Errorf("Subscription count = %d, expected 1", bus.GetSubscriptionCount())
	}

	subs := bus.GetSubscriptions(PhasePreDamage)
	if len(subs) != 1 {
		t.Errorf("PreDamage subscriptions = %d, expected 1", len(subs))
	}
	if subs[0].OwnerID != "player-001" {
		t.Errorf("OwnerID = %s, expected player-001", subs[0].OwnerID)
	}
}

func TestEventBusUnsubscribe(t *testing.T) {
	bus := NewEventBus("game-001")
	d := NewDecision("test", []Option{})

	subID := bus.Subscribe(PhasePreDamage, "player-001", "buff-001", "buff", d)

	// 取消订阅
	ok := bus.Unsubscribe(subID)
	if !ok {
		t.Error("Unsubscribe should return true")
	}

	if bus.GetSubscriptionCount() != 0 {
		t.Errorf("Subscription count should be 0 after unsubscribe")
	}

	// 再次取消应该返回false
	ok = bus.Unsubscribe(subID)
	if ok {
		t.Error("Unsubscribe non-existent should return false")
	}
}

func TestEventBusUnsubscribeBySource(t *testing.T) {
	bus := NewEventBus("game-001")
	d := NewDecision("test", []Option{})

	bus.Subscribe(PhasePreDamage, "player-001", "buff-001", "buff", d)
	bus.Subscribe(PhasePreEvent, "player-001", "buff-001", "buff", d)
	bus.Subscribe(PhasePreDamage, "player-002", "buff-002", "buff", d)

	// 移除buff-001的所有订阅
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

	bus.Subscribe(PhasePreDamage, "player-001", "buff-001", "buff", d)
	bus.Subscribe(PhasePreEvent, "player-001", "buff-002", "buff", d)
	bus.Subscribe(PhasePreDamage, "player-002", "buff-003", "buff", d)

	// 移除player-001的所有订阅
	count := bus.UnsubscribeByOwner("player-001")
	if count != 2 {
		t.Errorf("Removed count = %d, expected 2", count)
	}

	if bus.GetSubscriptionCount() != 1 {
		t.Errorf("Remaining count should be 1")
	}
}

func TestEventBusPublish(t *testing.T) {
	bus := NewEventBus("game-001")

	// 订阅不需要确认的Decision（自动执行）
	d1 := NewAutoDecision("auto", []Option{{ID: "ok", Label: "OK"}})
	d1.WithPriority(100) // 高优先级

	// 订阅需要确认的Decision
	d2 := NewDecision("是否使用护盾？", []Option{
		{ID: "use", Label: "使用"},
		{ID: "skip", Label: "跳过"},
	})
	d2.WithPriority(50) // 低优先级

	bus.Subscribe(PhasePreDamage, "player-001", "buff-001", "buff", d1)
	bus.Subscribe(PhasePreDamage, "player-001", "item-001", "item", d2)

	// 发布
	ctx := NewContext(nil)
	decisions := bus.Publish(PhasePreDamage, "player-001", ctx)

	// 只返回需要确认的Decision
	if len(decisions) != 1 {
		t.Errorf("Decisions count = %d, expected 1 (only NeedConfirm)", len(decisions))
	}
	if decisions[0].Prompt != "是否使用护盾？" {
		t.Errorf("Decision Prompt = %s, expected '是否使用护盾？'", decisions[0].Prompt)
	}
}

func TestEventBusPublishPriorityOrder(t *testing.T) {
	bus := NewEventBus("game-001")

	// 创建两个需要确认的Decision，不同优先级
	d1 := NewDecision("高优先级", []Option{{ID: "a", Label: "A"}})
	d1.WithPriority(100)

	d2 := NewDecision("低优先级", []Option{{ID: "b", Label: "B"}})
	d2.WithPriority(50)

	bus.Subscribe(PhasePreEvent, "player-001", "buff-001", "buff", d1)
	bus.Subscribe(PhasePreEvent, "player-001", "buff-002", "buff", d2)

	ctx := NewContext(nil)
	decisions := bus.Publish(PhasePreEvent, "player-001", ctx)

	if len(decisions) != 2 {
		t.Errorf("Decisions count = %d, expected 2", len(decisions))
	}

	// 检查排序：高优先级在前
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

	bus.Subscribe(PhasePreDamage, "player-001", "buff-001", "buff", d)
	bus.Subscribe(PhasePreDamage, "player-002", "buff-002", "buff", d)

	ctx := NewContext(nil)
	decisions := bus.Publish(PhasePreDamage, "player-001", ctx)

	// 只返回player-001的Decision
	if len(decisions) != 1 {
		t.Errorf("Decisions count = %d, expected 1 (filtered by owner)", len(decisions))
	}
}

func TestEventBusClear(t *testing.T) {
	bus := NewEventBus("game-001")
	d := NewDecision("test", []Option{})

	bus.Subscribe(PhasePreDamage, "player-001", "buff-001", "buff", d)
	bus.Subscribe(PhasePreEvent, "player-002", "buff-002", "buff", d)

	bus.Clear()

	if bus.GetSubscriptionCount() != 0 {
		t.Errorf("Subscription count should be 0 after clear")
	}
}

// ========== Context Tests ==========

func TestNewContext(t *testing.T) {
	player := &TestPlayer{UserID: "test"}
	ctx := NewContext(player)

	if ctx.Player == nil {
		t.Error("Player should not be nil")
	}
}

func TestContextWithEvent(t *testing.T) {
	ctx := NewContext(&TestPlayer{})
	ctx.WithEvent("some_event")

	if ctx.GameEvent == nil {
		t.Error("GameEvent should not be nil")
	}
}

func TestContextWithData(t *testing.T) {
	ctx := NewContext(nil)
	ctx.WithData(100) // 例如伤害值

	if ctx.Data != 100 {
		t.Errorf("Data = %v, expected 100", ctx.Data)
	}
}