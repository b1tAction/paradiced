package engine

import (
	"testing"

	"github.com/b1tAction/Fated/internal/core"
	"github.com/b1tAction/Fated/pkg/event"
)

// ========== StateMachine Creation Tests ==========

func TestNewStateMachine(t *testing.T) {
	game := NewGame("game-001", 0)
	sm := NewStateMachine(game)

	if sm == nil {
		t.Fatal("NewStateMachine should not return nil")
	}
	if sm.Game != game {
		t.Error("StateMachine.Game should match input game")
	}
	if sm.WaitingFor == nil {
		t.Error("StateMachine.WaitingFor should not be nil")
	}
	if len(sm.WaitingFor) != 0 {
		t.Errorf("Initial WaitingFor count = %d, expected 0", len(sm.WaitingFor))
	}
	if sm.FlowState != "idle" {
		t.Errorf("Initial FlowState = %s, expected idle", sm.FlowState)
	}
}

// ========== TriggerPhase Tests ==========

func TestStateMachineTriggerPhase(t *testing.T) {
	game := NewGame("game-001", 0)
	player := core.NewPlayer(core.PlayerConfig{UserID: "player-001"})
	game.AddPlayer(player)

	sm := NewStateMachine(game)

	// 触发没有订阅的 Phase
	decisions := sm.TriggerPhase(event.PhaseBeforeTurn, player)

	if decisions == nil {
		t.Error("TriggerPhase should not return nil")
	}
	if len(decisions) != 0 {
		t.Errorf("Decisions count = %d, expected 0 (no subscriptions)", len(decisions))
	}
}

func TestStateMachineTriggerPhaseWithSubscription(t *testing.T) {
	game := NewGame("game-001", 0)
	player := core.NewPlayer(core.PlayerConfig{UserID: "player-001"})
	game.AddPlayer(player)

	// 添加一个需要确认的道具（会创建 Decision）
	item := core.NewItem(core.ItemTypeDiceUpgrade, "item-001")
	player.AddItem(item)
	game.SubscribeItem(player, item)

	sm := NewStateMachine(game)

	// 触发道具订阅的 Phase
	def := core.GetItemDefinition(core.ItemTypeDiceUpgrade)
	decisions := sm.TriggerPhase(def.Phase, player)

	// DiceUpgrade 需要确认，所以应该返回一个 Decision
	if len(decisions) != 1 {
		t.Errorf("Decisions count = %d, expected 1 (one subscribed item)", len(decisions))
	}
}

func TestStateMachineTriggerPhaseUpdatesContext(t *testing.T) {
	game := NewGame("game-001", 0)
	player := core.NewPlayer(core.PlayerConfig{UserID: "player-001"})
	game.AddPlayer(player)

	sm := NewStateMachine(game)

	// 触发 Phase
	sm.TriggerPhase(event.PhaseBeforeTurn, player)

	// 验证上下文已设置
	if sm.CurrentCtx == nil {
		t.Error("CurrentCtx should be set after TriggerPhase")
	}
	if sm.CurrentCtx.Player != player {
		t.Error("CurrentCtx.Player should match input player")
	}
	if sm.CurrentCtx.GameState == nil {
		t.Error("CurrentCtx.GameState should be set")
	}
	if sm.CurrentCtx.GameState.Round != game.State.Round {
		t.Errorf("GameState.Round = %d, expected %d", sm.CurrentCtx.GameState.Round, game.State.Round)
	}
}

// ========== TriggerPhaseAndWait Tests ==========

func TestStateMachineTriggerPhaseAndWaitNoSubscriptions(t *testing.T) {
	game := NewGame("game-001", 0)
	player := core.NewPlayer(core.PlayerConfig{UserID: "player-001"})
	game.AddPlayer(player)

	sm := NewStateMachine(game)

	// 没有订阅时，返回 false（不需要等待）
	needsWait := sm.TriggerPhaseAndWait(event.PhaseBeforeTurn, player)

	if needsWait {
		t.Error("TriggerPhaseAndWait should return false when no subscriptions")
	}
	if sm.IsWaiting() {
		t.Error("StateMachine should not be waiting")
	}
}

func TestStateMachineTriggerPhaseAndWaitWithSubscription(t *testing.T) {
	game := NewGame("game-001", 0)
	player := core.NewPlayer(core.PlayerConfig{UserID: "player-001"})
	game.AddPlayer(player)

	// 添加需要确认的道具
	item := core.NewItem(core.ItemTypeDiceUpgrade, "item-001")
	player.AddItem(item)
	game.SubscribeItem(player, item)

	sm := NewStateMachine(game)

	// 有订阅时，返回 true（需要等待）
	def := core.GetItemDefinition(core.ItemTypeDiceUpgrade)
	needsWait := sm.TriggerPhaseAndWait(def.Phase, player)

	if !needsWait {
		t.Error("TriggerPhaseAndWait should return true when there are decisions")
	}
	if !sm.IsWaiting() {
		t.Error("StateMachine should be waiting")
	}
	if sm.GetWaitingCount() != 1 {
		t.Errorf("Waiting count = %d, expected 1", sm.GetWaitingCount())
	}
}

// ========== Waiting State Tests ==========

func TestStateMachineEnterWaitingState(t *testing.T) {
	game := NewGame("game-001", 0)
	sm := NewStateMachine(game)

	// 创建 Decision 列表
	d1 := event.NewDecision("决策1", []event.Option{{ID: "a", Label: "A"}})
	d2 := event.NewDecision("决策2", []event.Option{{ID: "b", Label: "B"}})
	decisions := []*event.Decision{d1, d2}

	sm.EnterWaitingState(decisions)

	if sm.FlowState != "waiting" {
		t.Errorf("FlowState = %s, expected waiting", sm.FlowState)
	}
	if !game.State.Waiting {
		t.Error("Game.State.Waiting should be true")
	}
	if len(sm.WaitingFor) != 2 {
		t.Errorf("WaitingFor count = %d, expected 2", len(sm.WaitingFor))
	}
}

func TestStateMachineExitWaitingState(t *testing.T) {
	game := NewGame("game-001", 0)
	sm := NewStateMachine(game)

	// 先进入等待状态
	d := event.NewDecision("测试决策", []event.Option{{ID: "ok", Label: "OK"}})
	sm.EnterWaitingState([]*event.Decision{d})

	// 退出等待状态
	sm.ExitWaitingState()

	if sm.FlowState != "running" {
		t.Errorf("FlowState = %s, expected running", sm.FlowState)
	}
	if game.State.Waiting {
		t.Error("Game.State.Waiting should be false")
	}
	if len(sm.WaitingFor) != 0 {
		t.Errorf("WaitingFor should be cleared, count = %d", len(sm.WaitingFor))
	}
}

// ========== User Choice Tests ==========

func TestStateMachineOnUserChoice(t *testing.T) {
	game := NewGame("game-001", 0)
	sm := NewStateMachine(game)

	executed := false
	ctx := event.NewContext(nil)

	d := event.NewDecision("测试决策", []event.Option{
		{ID: "ok", Label: "OK", Action: func(c *event.Context) { executed = true }},
	})

	sm.EnterWaitingState([]*event.Decision{d})
	sm.CurrentCtx = ctx

	// 用户选择第一个选项
	sm.OnUserChoice(0)

	if !executed {
		t.Error("Option action should be executed")
	}
	if sm.GetWaitingCount() != 0 {
		t.Errorf("Waiting count should be 0 after choice, got %d", sm.GetWaitingCount())
	}
	if sm.IsWaiting() {
		t.Error("StateMachine should not be waiting after all choices processed")
	}
}

func TestStateMachineOnUserChoiceMultipleDecisions(t *testing.T) {
	game := NewGame("game-001", 0)
	sm := NewStateMachine(game)

	choice1 := false
	choice2 := false

	d1 := event.NewDecision("决策1", []event.Option{
		{ID: "a", Label: "A", Action: func(c *event.Context) { choice1 = true }},
	})
	d2 := event.NewDecision("决策2", []event.Option{
		{ID: "b", Label: "B", Action: func(c *event.Context) { choice2 = true }},
	})

	sm.EnterWaitingState([]*event.Decision{d1, d2})
	sm.CurrentCtx = event.NewContext(nil)

	// 第一个选择
	sm.OnUserChoice(0)

	if !choice1 {
		t.Error("First decision should be executed")
	}
	if choice2 {
		t.Error("Second decision should not be executed yet")
	}
	if sm.GetWaitingCount() != 1 {
		t.Errorf("Waiting count = %d, expected 1", sm.GetWaitingCount())
	}
	if !sm.IsWaiting() {
		t.Error("StateMachine should still be waiting")
	}

	// 第二个选择
	sm.OnUserChoice(0)

	if !choice2 {
		t.Error("Second decision should be executed")
	}
	if sm.IsWaiting() {
		t.Error("StateMachine should not be waiting after all choices")
	}
}

// ========== Current Decision Tests ==========

func TestStateMachineGetCurrentDecision(t *testing.T) {
	game := NewGame("game-001", 0)
	sm := NewStateMachine(game)

	// 没有决策时返回 nil
	if sm.GetCurrentDecision() != nil {
		t.Error("GetCurrentDecision should return nil when no decisions")
	}

	// 有决策时返回第一个
	d1 := event.NewDecision("决策1", []event.Option{{ID: "a", Label: "A"}})
	d2 := event.NewDecision("决策2", []event.Option{{ID: "b", Label: "B"}})
	sm.EnterWaitingState([]*event.Decision{d1, d2})

	current := sm.GetCurrentDecision()
	if current == nil {
		t.Error("GetCurrentDecision should return first decision")
	}
	if current.Prompt != "决策1" {
		t.Errorf("Current decision Prompt = %s, expected 决策1", current.Prompt)
	}
}

func TestStateMachineIsWaiting(t *testing.T) {
	game := NewGame("game-001", 0)
	sm := NewStateMachine(game)

	// 初始不等待
	if sm.IsWaiting() {
		t.Error("StateMachine should not be waiting initially")
	}

	// 进入等待状态
	d := event.NewDecision("测试", []event.Option{{ID: "ok", Label: "OK"}})
	sm.EnterWaitingState([]*event.Decision{d})

	if !sm.IsWaiting() {
		t.Error("StateMachine should be waiting after EnterWaitingState")
	}

	// 退出等待状态
	sm.ExitWaitingState()

	if sm.IsWaiting() {
		t.Error("StateMachine should not be waiting after ExitWaitingState")
	}
}

func TestStateMachineGetWaitingCount(t *testing.T) {
	game := NewGame("game-001", 0)
	sm := NewStateMachine(game)

	if sm.GetWaitingCount() != 0 {
		t.Errorf("Initial waiting count = %d, expected 0", sm.GetWaitingCount())
	}

	d1 := event.NewDecision("决策1", []event.Option{{ID: "a", Label: "A"}})
	d2 := event.NewDecision("决策2", []event.Option{{ID: "b", Label: "B"}})
	d3 := event.NewDecision("决策3", []event.Option{{ID: "c", Label: "C"}})
	sm.EnterWaitingState([]*event.Decision{d1, d2, d3})

	if sm.GetWaitingCount() != 3 {
		t.Errorf("Waiting count = %d, expected 3", sm.GetWaitingCount())
	}
}

// ========== Flow Control Tests ==========

func TestStateMachineContinueFlow(t *testing.T) {
	game := NewGame("game-001", 0)
	sm := NewStateMachine(game)

	// 进入等待状态
	d := event.NewDecision("测试", []event.Option{{ID: "ok", Label: "OK"}})
	sm.EnterWaitingState([]*event.Decision{d})

	// 强制继续流程
	sm.ContinueFlow()

	if sm.IsWaiting() {
		t.Error("StateMachine should not be waiting after ContinueFlow")
	}
	if sm.FlowState != "running" {
		t.Errorf("FlowState = %s, expected running", sm.FlowState)
	}
}

func TestStateMachineCancelWaiting(t *testing.T) {
	game := NewGame("game-001", 0)
	sm := NewStateMachine(game)

	executed := false
	d := event.NewDecision("测试", []event.Option{
		{ID: "a", Label: "A"},
		{ID: "b", Label: "B", Action: func(c *event.Context) { executed = true }},
	})
	d.Default = 1 // 默认选项是第二个

	sm.EnterWaitingState([]*event.Decision{d})
	sm.CurrentCtx = event.NewContext(nil)

	// 取消等待，执行默认选项
	sm.CancelWaiting()

	if !executed {
		t.Error("Default option should be executed on cancel")
	}
	if sm.IsWaiting() {
		t.Error("StateMachine should not be waiting after cancel")
	}
}

// ========== Phase Executor Tests ==========

func TestStateMachineExecuteBeforeTurnPhase(t *testing.T) {
	game := NewGame("game-001", 0)
	player := core.NewPlayer(core.PlayerConfig{UserID: "player-001"})
	game.AddPlayer(player)

	sm := NewStateMachine(game)

	decisions := sm.ExecuteBeforeTurnPhase(player)

	// 没有订阅，返回空列表
	if len(decisions) != 0 {
		t.Errorf("Decisions count = %d, expected 0", len(decisions))
	}
}

func TestStateMachineExecuteOnMovePhase(t *testing.T) {
	game := NewGame("game-001", 0)
	player := core.NewPlayer(core.PlayerConfig{UserID: "player-001"})
	game.AddPlayer(player)

	sm := NewStateMachine(game)

	decisions := sm.ExecuteOnMovePhase(player)

	if len(decisions) != 0 {
		t.Errorf("Decisions count = %d, expected 0", len(decisions))
	}
}

func TestStateMachineExecuteOnLandPhase(t *testing.T) {
	game := NewGame("game-001", 0)
	player := core.NewPlayer(core.PlayerConfig{UserID: "player-001"})
	game.AddPlayer(player)

	sm := NewStateMachine(game)

	decisions := sm.ExecuteOnLandPhase(player)

	if len(decisions) != 0 {
		t.Errorf("Decisions count = %d, expected 0", len(decisions))
	}
}

func TestStateMachineExecutePreEventPhase(t *testing.T) {
	game := NewGame("game-001", 0)
	player := core.NewPlayer(core.PlayerConfig{UserID: "player-001"})
	game.AddPlayer(player)

	sm := NewStateMachine(game)

	decisions := sm.ExecutePreEventPhase(player)

	if len(decisions) != 0 {
		t.Errorf("Decisions count = %d, expected 0", len(decisions))
	}
}

func TestStateMachineExecutePreDamagePhase(t *testing.T) {
	game := NewGame("game-001", 0)
	player := core.NewPlayer(core.PlayerConfig{UserID: "player-001"})
	game.AddPlayer(player)

	sm := NewStateMachine(game)

	// 测试带伤害数据的调用
	decisions := sm.ExecutePreDamagePhase(player, 5)

	if len(decisions) != 0 {
		t.Errorf("Decisions count = %d, expected 0", len(decisions))
	}

	// 验证上下文已设置
	if sm.CurrentCtx == nil {
		t.Error("CurrentCtx should be set")
	}
	// 验证伤害数据已保留
	if sm.CurrentCtx.GetData() != 5 {
		t.Errorf("Context Data = %v, expected 5", sm.CurrentCtx.GetData())
	}
}

func TestStateMachineExecuteAfterTurnPhase(t *testing.T) {
	game := NewGame("game-001", 0)
	player := core.NewPlayer(core.PlayerConfig{UserID: "player-001"})
	game.AddPlayer(player)

	sm := NewStateMachine(game)

	decisions := sm.ExecuteAfterTurnPhase(player)

	if len(decisions) != 0 {
		t.Errorf("Decisions count = %d, expected 0", len(decisions))
	}
}

// ========== Phase Executor with Buff Tests ==========

func TestStateMachineExecutePreDamageWithHiddenBuff(t *testing.T) {
	game := NewGame("game-001", 0)
	player := core.NewPlayer(core.PlayerConfig{UserID: "player-001"})
	game.AddPlayer(player)

	// 添加隐匿 Buff（PreDamage Phase，高优先级）
	buff := core.NewBuff(core.BuffTypeHidden, 3)
	player.AddBuff(buff)
	game.SubscribeBuff(player, buff)

	sm := NewStateMachine(game)

	// 触发 PreDamage
	decisions := sm.ExecutePreDamagePhase(player, 10)

	// 隐匿不需要用户确认，所以返回空列表
	// 但 Buff 效果已自动执行
	if len(decisions) != 0 {
		t.Errorf("Hidden buff should auto-execute, decisions count = %d", len(decisions))
	}
}

func TestStateMachineExecuteBeforeTurnWithCurseBuff(t *testing.T) {
	game := NewGame("game-001", 0)
	player := core.NewPlayer(core.PlayerConfig{UserID: "player-001"})
	game.AddPlayer(player)

	// 添加诅咒 Buff（BeforeTurn Phase）
	buff := core.NewBuff(core.BuffTypeCurse, 3)
	player.AddBuff(buff)
	game.SubscribeBuff(player, buff)

	sm := NewStateMachine(game)

	// 触发 BeforeTurn
	decisions := sm.ExecuteBeforeTurnPhase(player)

	// 诅咒自动执行，不需要确认
	if len(decisions) != 0 {
		t.Errorf("Curse buff should auto-execute, decisions count = %d", len(decisions))
	}
}