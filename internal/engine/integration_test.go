package engine

import (
	"testing"

	"github.com/b1tAction/Fated/internal/core"
	"github.com/b1tAction/Fated/pkg/event"
)

// ========== Integration Tests: Game + EventBus + StateMachine ==========

// TestIntegrationStateMachineFlow 测试状态机的完整回合流程
func TestIntegrationStateMachineFlow(t *testing.T) {
	// 1. 创建游戏和玩家
	game := NewGame("game-001", 0)
	player := core.NewPlayer(core.PlayerConfig{
		UserID:  "player-001",
		Faction: core.FactionQingLong,
		MaxHP:   6,
		MaxLP:   5,
	})
	game.AddPlayer(player)

	// 创建状态机
	sm := NewStateMachine(game)

	// 2. 添加 Buff（诅咒 - BeforeTurn Phase）
	curseBuff := core.NewBuff(core.BuffTypeCurse, 3)
	player.AddBuff(curseBuff)
	game.SubscribeBuff(player, curseBuff)

	// 验证订阅已创建
	if game.Bus.GetSubscriptionCount() != 1 {
		t.Errorf("Expected 1 subscription (curse buff), got %d", game.Bus.GetSubscriptionCount())
	}

	// 3. 执行回合开始前 Phase（诅咒效果：LP-1）
	t.Log("=== Phase: BeforeTurn ===")
	decisions := sm.ExecuteBeforeTurnPhase(player)

	// 诅咒自动执行，不需要确认
	if len(decisions) != 0 {
		t.Errorf("BeforeTurn should auto-execute Curse, decisions count = %d", len(decisions))
	}

	// 4. 模拟掷骰子移动
	t.Log("=== Phase: OnMove ===")
	decisions = sm.ExecuteOnMovePhase(player)

	// 5. 落地后
	t.Log("=== Phase: OnLand ===")
	decisions = sm.ExecuteOnLandPhase(player)

	// 6. 事件触发前
	t.Log("=== Phase: PreEvent ===")
	decisions = sm.ExecutePreEventPhase(player)

	// 7. 执行事件效果（简化：假设触发良性事件）
	t.Log("=== Event Execution ===")
	divineBuff := core.NewBuff(core.BuffTypeDivine, 3)
	player.AddBuff(divineBuff)
	game.SubscribeBuff(player, divineBuff)

	// 8. 回合结束后
	t.Log("=== Phase: AfterTurn ===")
	decisions = sm.ExecuteAfterTurnPhase(player)

	// 9. Tick Buff 持续时间
	expired := player.TickBuffs()
	t.Logf("Expired buffs: %d", len(expired))

	t.Logf("Player final state: HP=%d, LP=%d, Buffs=%d",
		player.HP, player.LP, len(player.ActiveBuffs))
}

// TestIntegrationItemNeedConfirm 测试道具需要用户确认
func TestIntegrationItemNeedConfirm(t *testing.T) {
	game := NewGame("game-001", 0)
	player := core.NewPlayer(core.PlayerConfig{UserID: "player-001"})
	game.AddPlayer(player)

	sm := NewStateMachine(game)

	// 添加骰子升级卡（BeforeTurn，需要确认）
	item := core.NewItem(core.ItemTypeDiceUpgrade, "item-001")
	player.AddItem(item)
	game.SubscribeItem(player, item)

	// 触发 BeforeTurn
	needsWait := sm.TriggerPhaseAndWait(event.PhaseBeforeTurn, player)

	// 需要等待用户确认
	if !needsWait {
		t.Error("Should need to wait for DiceUpgrade confirmation")
	}
	if !sm.IsWaiting() {
		t.Error("StateMachine should be in waiting state")
	}

	// 获取当前决策
	current := sm.GetCurrentDecision()
	if current == nil {
		t.Fatal("Should have a current decision")
	}
	if current.Prompt != core.GetItemDefinition(core.ItemTypeDiceUpgrade).Desc {
		t.Errorf("Decision prompt = %s, expected %s",
			current.Prompt, core.GetItemDefinition(core.ItemTypeDiceUpgrade).Desc)
	}

	// 用户选择使用
	sm.OnUserChoice(0)

	// 验证已退出等待状态
	if sm.IsWaiting() {
		t.Error("Should not be waiting after user choice")
	}
}

// TestIntegrationHiddenBuffImmunity 测试隐匿 Buff 免疫效果
func TestIntegrationHiddenBuffImmunity(t *testing.T) {
	game := NewGame("game-001", 0)
	player := core.NewPlayer(core.PlayerConfig{
		UserID: "player-001",
		MaxHP:  6,
	})
	game.AddPlayer(player)

	sm := NewStateMachine(game)

	// 添加隐匿 Buff（PreDamage，Priority=100）
	hiddenBuff := core.NewBuff(core.BuffTypeHidden, 3)
	player.AddBuff(hiddenBuff)
	game.SubscribeBuff(player, hiddenBuff)

	// 模拟受伤
	damage := 5
	decisions := sm.ExecutePreDamagePhase(player, damage)

	// 隐匿自动执行免疫，不需要确认
	if len(decisions) != 0 {
		t.Errorf("Hidden should auto-immune, decisions count = %d", len(decisions))
	}

	// 验证 Context 包含伤害数据
	if sm.CurrentCtx.GetData() != damage {
		t.Errorf("Context damage = %v, expected %d", sm.CurrentCtx.GetData(), damage)
	}
}

// TestIntegrationBuffRemovalUnsubscribe 测试 Buff 移除时取消订阅
func TestIntegrationBuffRemovalUnsubscribe(t *testing.T) {
	game := NewGame("game-001", 0)
	player := core.NewPlayer(core.PlayerConfig{UserID: "player-001"})
	game.AddPlayer(player)

	// 添加诅咒 Buff
	curseBuff := core.NewBuff(core.BuffTypeCurse, 3)
	player.AddBuff(curseBuff)
	game.SubscribeBuff(player, curseBuff)

	initialCount := game.Bus.GetSubscriptionCount()
	if initialCount != 1 {
		t.Errorf("Expected 1 subscription, got %d", initialCount)
	}

	// 移除 Buff
	player.RemoveBuff(core.BuffTypeCurse)
	game.UnsubscribeBuff(curseBuff)

	// 验证订阅已取消
	if game.Bus.GetSubscriptionCount() != 0 {
		t.Errorf("Expected 0 subscriptions after removal, got %d", game.Bus.GetSubscriptionCount())
	}
}

// TestIntegrationPlayerRemovalCleansSubscriptions 测试玩家移除时清理订阅
func TestIntegrationPlayerRemovalCleansSubscriptions(t *testing.T) {
	game := NewGame("game-001", 0)
	player := core.NewPlayer(core.PlayerConfig{UserID: "player-001"})
	game.AddPlayer(player)

	// 添加多个 Buff
	buff1 := core.NewBuff(core.BuffTypeCurse, 3)
	buff2 := core.NewBuff(core.BuffTypeHidden, 3)
	player.AddBuff(buff1)
	player.AddBuff(buff2)
	game.SubscribeBuff(player, buff1)
	game.SubscribeBuff(player, buff2)

	// 添加道具
	item := core.NewItem(core.ItemTypeDiceUpgrade, "item-001")
	player.AddItem(item)
	game.SubscribeItem(player, item)

	// 验证订阅数
	if game.Bus.GetSubscriptionCount() != 3 {
		t.Errorf("Expected 3 subscriptions, got %d", game.Bus.GetSubscriptionCount())
	}

	// 移除玩家
	game.RemovePlayer("player-001")

	// 验证所有订阅已取消
	if game.Bus.GetSubscriptionCount() != 0 {
		t.Errorf("Expected 0 subscriptions after player removal, got %d", game.Bus.GetSubscriptionCount())
	}
}

// TestIntegrationZhuQueFactionPassive 测试朱雀阵营永久被动
func TestIntegrationZhuQueFactionPassive(t *testing.T) {
	game := NewGame("game-001", 0)
	player := core.NewPlayer(core.PlayerConfig{
		UserID:  "player-001",
		Faction: core.FactionZhuQue,
		MaxLP:   5,
	})
	game.AddPlayer(player)

	// 朱雀玩家初始有离火 Buff（永久，每4回合LP+1）
	if !player.HasBuff(core.BuffTypeFire) {
		t.Error("ZhuQue player should have Fire buff")
	}

	// 离火现在使用 BeforeTurn，需要订阅 EventBus
	fireBuff := player.GetBuff(core.BuffTypeFire)
	def := core.GetBuffDefinition(core.BuffTypeFire)
	for _, phase := range def.GetPhases() {
		if phase.NeedsSubscription() && len(fireBuff.SubscriptionIDs) == 0 {
			t.Error("Fire buff (BeforeTurn) should have subscription IDs")
		}
	}

	t.Logf("ZhuQue player has Fire buff: duration=%d, phases=%v", fireBuff.Duration, def.GetPhases())
}

// TestIntegrationAnyTimeItem 测试 AnyTime 道具（主动触发）
func TestIntegrationAnyTimeItem(t *testing.T) {
	game := NewGame("game-001", 0)
	player := core.NewPlayer(core.PlayerConfig{UserID: "player-001"})
	game.AddPlayer(player)

	// 添加反方向的钟（AnyTime，不需要订阅）
	item := core.NewItem(core.ItemTypeReverseClock, "item-001")
	player.AddItem(item)
	game.SubscribeItem(player, item)

	// AnyTime 道具不订阅 EventBus
	def := core.GetItemDefinition(core.ItemTypeReverseClock)
	if def.Phase.NeedsSubscription() {
		t.Error("ReverseClock (AnyTime) should not need subscription")
	}
	if item.SubscriptionID != "" {
		t.Error("AnyTime item should not have subscription ID")
	}

	// AnyTime 道具需要主动触发（不在 Phase 流程中）
	if !item.Usable {
		t.Error("Item should be usable")
	}
}

// TestIntegrationFullGameScenario 测试完整游戏场景
func TestIntegrationFullGameScenario(t *testing.T) {
	game := NewGame("game-001", 0)

	// 创建 4 个玩家，不同阵营
	factions := []core.Faction{core.FactionQingLong, core.FactionZhuQue, core.FactionBaiHu, core.FactionXuanWu}
	for i := 0; i < 4; i++ {
		player := core.NewPlayer(core.PlayerConfig{
			UserID:  "player-" + string(rune('0'+i+1)),
			Faction: factions[i],
			MaxHP:   6,
			MaxLP:   5,
		})
		game.AddPlayer(player)
	}

	sm := NewStateMachine(game)

	// 验证玩家数
	if len(game.Players) != 4 {
		t.Errorf("Expected 4 players, got %d", len(game.Players))
	}

	// 朱雀玩家应该有离火 Buff
	zhuquePlayer := game.GetPlayer("player-2")
	if zhuquePlayer == nil {
		t.Fatal("ZhuQue player should exist")
	}
	if !zhuquePlayer.HasBuff(core.BuffTypeFire) {
		t.Error("ZhuQue player should have Fire buff")
	}

	// 模拟多回合
	for round := 1; round <= 3; round++ {
		t.Logf("=== Round %d ===", round)

		for turn := 0; turn < 4; turn++ {
			currentPlayer := game.GetCurrentPlayer()
			t.Logf("  Turn %d: Player %s", turn, currentPlayer.UserID)

			// 执行回合流程（简化）
			sm.ExecuteBeforeTurnPhase(currentPlayer)
			sm.ExecuteOnMovePhase(currentPlayer)
			sm.ExecuteOnLandPhase(currentPlayer)
			sm.ExecutePreEventPhase(currentPlayer)
			sm.ExecuteAfterTurnPhase(currentPlayer)

			// Tick Buff
			currentPlayer.TickBuffs()

			// 下一回合
			game.NextTurn()
		}
	}

	t.Logf("Final game state: Round=%d, Turn=%d", game.State.Round, game.State.Turn)
}

// TestIntegrationContextDataPassing 测试 Context 数据传递
func TestIntegrationContextDataPassing(t *testing.T) {
	game := NewGame("game-001", 0)
	player := core.NewPlayer(core.PlayerConfig{UserID: "player-001"})
	game.AddPlayer(player)

	sm := NewStateMachine(game)

	// 创建接收 Context 数据的 Decision
	receivedDamage := 0
	d := event.NewDecision("测试伤害", []event.Option{
		{ID: "ok", Label: "OK", Action: func(ctx *event.Context) {
			if damage, ok := ctx.GetData().(int); ok {
				receivedDamage = damage
			}
		}},
	})

	sm.EnterWaitingState([]*event.Decision{d})
	sm.CurrentCtx = event.NewContext(player).WithData(10)

	// 执行
	sm.OnUserChoice(0)

	// 验证数据已传递
	if receivedDamage != 10 {
		t.Errorf("Received damage = %d, expected 10", receivedDamage)
	}
}
// ========== 多 Phase 和生命周期集成测试 ==========

// TestIntegrationMultiPhaseBuffSubscription 测试多 Phase Buff 的订阅和触发
func TestIntegrationMultiPhaseBuffSubscription(t *testing.T) {
	game := NewGame("game-001", 0)
	player := core.NewPlayer(core.PlayerConfig{UserID: "player-001"})
	game.AddPlayer(player)

	// 添加一个 Buff（诅咒，单个 Phase: BeforeTurn）
	buff := core.NewBuff(core.BuffTypeCurse, 3)
	game.ApplyBuffToPlayer(player, buff)

	// 验证订阅数量
	def := core.GetBuffDefinition(core.BuffTypeCurse)
	expectedSubs := 0
	for _, phase := range def.GetPhases() {
		if phase.NeedsSubscription() {
			expectedSubs++
		}
	}
	if game.Bus.GetSubscriptionCount() != expectedSubs {
		t.Errorf("Subscription count = %d, expected %d", game.Bus.GetSubscriptionCount(), expectedSubs)
	}

	// 创建状态机并触发 BeforeTurn
	sm := NewStateMachine(game)
	sm.ExecuteBeforeTurnPhase(player)

	// 验证诅咒效果已执行（LP-1）
	if player.LP != 7 { // 初始 LP=8, 诅咒扣1
		t.Errorf("Player LP = %d, expected 7 (诅咒扣1)", player.LP)
	}
}

// TestIntegrationBuffLifecycleWithAppliedEvent 测试 Buff Applied 事件的完整流程
func TestIntegrationBuffLifecycleWithAppliedEvent(t *testing.T) {
	game := NewGame("game-001", 0)
	player := core.NewPlayer(core.PlayerConfig{UserID: "player-001"})
	game.AddPlayer(player)

	// 设置监听器监听 PhaseOnBuffApplied
	receivedBuffType := core.BuffTypeNone
	d := event.NewAutoDecision("监听Applied", []event.Option{
		{ID: "listen", Label: "监听", Action: func(ctx *event.Context) {
			buff, ok := ctx.GetData().(*core.Buff)
			if ok {
				receivedBuffType = buff.Type
			}
		}},
	})
	game.Bus.Subscribe(event.PhaseOnBuffApplied, player.UserID, "applied-listener", "test", d)

	// 使用 ApplyBuffToPlayer 添加 Buff
	buff := core.NewBuff(core.BuffTypeDivine, 3)
	game.ApplyBuffToPlayer(player, buff)

	// 验证 Applied 事件已广播
	if receivedBuffType != core.BuffTypeDivine {
		t.Errorf("Received buff type = %v, expected Divine", receivedBuffType)
	}

	// 验证 Buff 已添加
	if !player.HasBuff(core.BuffTypeDivine) {
		t.Error("Player should have Divine buff")
	}

	// 验证订阅已创建
	if game.Bus.GetSubscriptionCount() == 0 {
		t.Error("Should have subscriptions")
	}
}

// TestIntegrationBuffLifecycleWithRemovedEvent 测试 Buff Removed 事件的完整流程
func TestIntegrationBuffLifecycleWithRemovedEvent(t *testing.T) {
	game := NewGame("game-001", 0)
	player := core.NewPlayer(core.PlayerConfig{UserID: "player-001"})
	game.AddPlayer(player)

	// 设置监听器监听 PhaseOnBuffRemoved
	receivedBuffType := core.BuffTypeNone
	d := event.NewAutoDecision("监听Removed", []event.Option{
		{ID: "listen", Label: "监听", Action: func(ctx *event.Context) {
			buff, ok := ctx.GetData().(*core.Buff)
			if ok {
				receivedBuffType = buff.Type
			}
		}},
	})
	game.Bus.Subscribe(event.PhaseOnBuffRemoved, player.UserID, "removed-listener", "test", d)

	// 先添加 Buff
	buff := core.NewBuff(core.BuffTypeCurse, 3)
	game.ApplyBuffToPlayer(player, buff)

	// 使用 RemoveBuffFromPlayer 移除 Buff
	game.RemoveBuffFromPlayer(player, buff)

	// 验证 Removed 事件已广播
	if receivedBuffType != core.BuffTypeCurse {
		t.Errorf("Received buff type = %v, expected Curse", receivedBuffType)
	}

	// 验证 Buff 已移除
	if player.HasBuff(core.BuffTypeCurse) {
		t.Error("Player should not have Curse buff")
	}

	// 验证订阅已取消
	if game.Bus.GetSubscriptionCount() != 1 { // 只剩监听器的订阅
		t.Errorf("Subscription count = %d, expected 1 (only listener)", game.Bus.GetSubscriptionCount())
	}
}

// TestIntegrationZhuQueFireHandler 测试朱雀离火的定制处理器
func TestIntegrationZhuQueFireHandler(t *testing.T) {
	game := NewGame("game-001", 0)
	player := core.NewPlayer(core.PlayerConfig{
		UserID:  "player-001",
		Faction: core.FactionZhuQue,
		MaxLP:   5,
	})
	game.AddPlayer(player)

	sm := NewStateMachine(game)

	// 朱雀玩家初始有离火 Buff
	if !player.HasBuff(core.BuffTypeFire) {
		t.Fatal("ZhuQue player should have Fire buff")
	}

	initialLP := player.LP

	// 执行 4 次回合开始前 Phase
	for i := 0; i < 4; i++ {
		sm.ExecuteBeforeTurnPhase(player)
	}

	// 每 4 回合 LP+1
	if player.LP != initialLP+1 {
		t.Errorf("LP = %d, expected %d (每4回合+1)", player.LP, initialLP+1)
	}

	// 计数器应该重置为 0
	if player.GetFireCounter() != 0 {
		t.Errorf("FireCounter = %d, expected 0", player.GetFireCounter())
	}
}

// TestIntegrationEventHandlerStrategy 测试 EventHandler 策略执行
func TestIntegrationEventHandlerStrategy(t *testing.T) {
	game := NewGame("game-001", 0)
	player := core.NewPlayer(core.PlayerConfig{
		UserID: "player-001",
		MaxLP:  5,
		MaxHP:  6,
	})
	game.AddPlayer(player)

	sm := NewStateMachine(game)

	// 测试诅咒 Buff（使用默认处理器：LP-1）
	curseBuff := core.NewBuff(core.BuffTypeCurse, 3)
	game.ApplyBuffToPlayer(player, curseBuff)

	sm.ExecuteBeforeTurnPhase(player)
	if player.LP != 4 { // 初始 LP=5, 诅咒扣1
		t.Errorf("Curse: LP = %d, expected 4", player.LP)
	}

	// 测试神眷 Buff（使用默认处理器：LP+1）
	divineBuff := core.NewBuff(core.BuffTypeDivine, 3)
	game.ApplyBuffToPlayer(player, divineBuff)

	sm.ExecuteBeforeTurnPhase(player)
	// 此时诅咒和神眷都会执行：诅咒 LP-1，神眷 LP+1
	// LP = 4 - 1 + 1 = 4
	if player.LP != 4 { // 诅咒扣1 + 神眷加1 = 相互抵消
		t.Errorf("After Divine+Curse: LP = %d, expected 4", player.LP)
	}

	// 测试甘霖 Buff（使用默认处理器：HP+1）
	rainBuff := core.NewBuff(core.BuffTypeRain, 4)
	game.ApplyBuffToPlayer(player, rainBuff)

	sm.ExecuteAfterTurnPhase(player)
	if player.HP != 7 {
		t.Errorf("Rain: HP = %d, expected 7", player.HP)
	}
}

// TestIntegrationBuffHearsOwnLifecycleEvent 测试 Buff 能听到自己的 Applied/Removed 事件
func TestIntegrationBuffHearsOwnLifecycleEvent(t *testing.T) {
	game := NewGame("game-001", 0)
	player := core.NewPlayer(core.PlayerConfig{UserID: "player-001"})
	game.AddPlayer(player)

	// 创建一个订阅了 Applied/Removed 的 Buff（模拟复杂 Buff）
	selfAppliedTriggered := false
	selfRemovedTriggered := false

	// 订阅 Applied 事件
	appliedHandler := event.NewAutoDecision("AppliedHandler", []event.Option{
		{ID: "handle", Label: "处理", Action: func(ctx *event.Context) {
			buff, ok := ctx.GetData().(*core.Buff)
			if ok && buff.Type == core.BuffTypeDivine {
				selfAppliedTriggered = true
			}
		}},
	})
	game.Bus.Subscribe(event.PhaseOnBuffApplied, player.UserID, "divine-applied", "buff", appliedHandler)

	// 订阅 Removed 事件
	removedHandler := event.NewAutoDecision("RemovedHandler", []event.Option{
		{ID: "handle", Label: "处理", Action: func(ctx *event.Context) {
			buff, ok := ctx.GetData().(*core.Buff)
			if ok && buff.Type == core.BuffTypeDivine {
				selfRemovedTriggered = true
			}
		}},
	})
	game.Bus.Subscribe(event.PhaseOnBuffRemoved, player.UserID, "divine-removed", "buff", removedHandler)

	// 添加 Buff
	buff := core.NewBuff(core.BuffTypeDivine, 3)
	game.ApplyBuffToPlayer(player, buff)

	// 验证 Applied 事件已触发
	if !selfAppliedTriggered {
		t.Error("Buff should hear its own Applied event")
	}

	// 移除 Buff
	game.RemoveBuffFromPlayer(player, buff)

	// 验证 Removed 事件已触发
	if !selfRemovedTriggered {
		t.Error("Buff should hear its own Removed event")
	}
}

// TestIntegrationMultipleBuffsMultiplePhases 测试多个 Buff 多个 Phase 的完整流程
func TestIntegrationMultipleBuffsMultiplePhases(t *testing.T) {
	game := NewGame("game-001", 0)
	player := core.NewPlayer(core.PlayerConfig{
		UserID: "player-001",
		MaxHP:  6,
		MaxLP:  5,
	})
	game.AddPlayer(player)

	sm := NewStateMachine(game)

	// 添加多个不同 Phase 的 Buff
	curseBuff := core.NewBuff(core.BuffTypeCurse, 3)     // BeforeTurn: LP-1
	divineBuff := core.NewBuff(core.BuffTypeDivine, 3)  // BeforeTurn: LP+1
	corruptBuff := core.NewBuff(core.BuffTypeCorrupt, 4) // AfterTurn: HP-1
	rainBuff := core.NewBuff(core.BuffTypeRain, 4)       // AfterTurn: HP+1

	game.ApplyBuffToPlayer(player, curseBuff)
	game.ApplyBuffToPlayer(player, divineBuff)
	game.ApplyBuffToPlayer(player, corruptBuff)
	game.ApplyBuffToPlayer(player, rainBuff)

	// 验证所有订阅已创建
	// Curse: 1, Divine: 1, Corrupt: 1, Rain: 1 = 4
	if game.Bus.GetSubscriptionCount() != 4 {
		t.Errorf("Subscription count = %d, expected 4", game.Bus.GetSubscriptionCount())
	}

	// 执行 BeforeTurn Phase（诅咒-1 + 神眷+1 = 0）
	sm.ExecuteBeforeTurnPhase(player)
	if player.LP != 5 {
		t.Errorf("After BeforeTurn: LP = %d, expected 5", player.LP)
	}

	// 执行 AfterTurn Phase（腐化-1 + 甘霖+1 = 0）
	sm.ExecuteAfterTurnPhase(player)
	if player.HP != 6 {
		t.Errorf("After AfterTurn: HP = %d, expected 6", player.HP)
	}
}
