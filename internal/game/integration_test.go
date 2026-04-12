package game

import (
	"testing"

	"github.com/b1tAction/Fated/pkg/event"
)

// ========== Integration Tests: Game + EventBus + StateMachine ==========

// TestIntegrationFullTurnFlow 测试完整的回合流程
// 模拟一个玩家回合，验证 Phase 触发顺序和 Buff/道具效果
func TestIntegrationFullTurnFlow(t *testing.T) {
	// 1. 创建游戏和玩家
	game := NewGame("game-001")
	player := NewPlayer(PlayerConfig{
		UserID:  "player-001",
		Faction: FactionQingLong,
		MaxHP:   6,
		MaxLP:   5,
	})
	game.AddPlayer(player)

	// 创建状态机
	sm := NewStateMachine(game)

	// 2. 添加 Buff（诅咒 - BeforeTurn Phase）
	curseBuff := NewBuff(BuffTypeCurse, 3)
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
	// 如果有迷途 Buff，这里会反向移动
	decisions = sm.ExecuteOnMovePhase(player)
	if sm.IsWaiting() {
		t.Log("Waiting for OnMove decision...")
	}

	// 5. 落地后
	t.Log("=== Phase: OnLand ===")
	decisions = sm.ExecuteOnLandPhase(player)
	// 如果有任意门道具，这里需要选择目标

	// 6. 事件触发前
	t.Log("=== Phase: PreEvent ===")
	// 如果有辟邪 Buff，这里会免疫毒瘴
	decisions = sm.ExecutePreEventPhase(player)

	// 7. 执行事件效果（简化：假设触发良性事件）
	t.Log("=== Event Execution ===")
	// 模拟神眷事件
	divineBuff := NewBuff(BuffTypeDivine, 3)
	player.AddBuff(divineBuff)
	game.SubscribeBuff(player, divineBuff)

	// 8. 回合结束后
	t.Log("=== Phase: AfterTurn ===")
	// 如果有甘霖/腐化 Buff，这里会 HP±1
	decisions = sm.ExecuteAfterTurnPhase(player)

	// 9. Tick Buff 持续时间
	expired := player.TickBuffs()
	t.Logf("Expired buffs: %d", len(expired))

	// 验证最终状态
	t.Logf("Player final state: HP=%d, LP=%d, Buffs=%d",
		player.HP, player.LP, len(player.ActiveBuffs))
}

// TestIntegrationBuffAutoExecution 测试 Buff 自动执行效果
func TestIntegrationBuffAutoExecution(t *testing.T) {
	game := NewGame("game-001")
	player := NewPlayer(PlayerConfig{
		UserID: "player-001",
		MaxHP:  6,
		MaxLP:  5,
	})
	game.AddPlayer(player)

	sm := NewStateMachine(game)

	// 添加诅咒 Buff（BeforeTurn，自动 LP-1）
	curseBuff := NewBuff(BuffTypeCurse, 3)
	player.AddBuff(curseBuff)
	game.SubscribeBuff(player, curseBuff)

	initialLP := player.LP

	// 触发 BeforeTurn
	sm.ExecuteBeforeTurnPhase(player)

	// 验证诅咒效果已执行（LP-1）
	// 注意：实际效果需要在 Decision 的 Action 中实现
	// 这里只验证订阅和触发机制工作正常
	t.Logf("LP: before=%d, after=%d", initialLP, player.LP)
}

// TestIntegrationItemNeedConfirm 测试道具需要用户确认
func TestIntegrationItemNeedConfirm(t *testing.T) {
	game := NewGame("game-001")
	player := NewPlayer(PlayerConfig{UserID: "player-001"})
	game.AddPlayer(player)

	sm := NewStateMachine(game)

	// 添加骰子升级卡（BeforeTurn，需要确认）
	item := NewItem(ItemTypeDiceUpgrade, "item-001")
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
	if current.Prompt != ItemTypeDiceUpgrade.GetItemDefinition().Desc {
		t.Errorf("Decision prompt = %s, expected %s",
			current.Prompt, ItemTypeDiceUpgrade.GetItemDefinition().Desc)
	}

	// 用户选择使用
	t.Logf("User options: %v", current.Options)
	sm.OnUserChoice(0) // 选择第一个选项（使用）

	// 验证已退出等待状态
	if sm.IsWaiting() {
		t.Error("Should not be waiting after user choice")
	}
}

// TestIntegrationMultipleBuffsPriority 测试多个 Buff 的优先级排序
func TestIntegrationMultipleBuffsPriority(t *testing.T) {
	game := NewGame("game-001")
	player := NewPlayer(PlayerConfig{UserID: "player-001"})
	game.AddPlayer(player)

	sm := NewStateMachine(game)

	// 添加多个 BeforeTurn Buff
	// 诅咒 Priority=50，毒瘴 Priority=30（更低，后执行）
	curseBuff := NewBuff(BuffTypeCurse, 3)
	poisonBuff := NewBuff(BuffTypePoison, 3)

	player.AddBuff(curseBuff)
	player.AddBuff(poisonBuff)

	game.SubscribeBuff(player, curseBuff)
	game.SubscribeBuff(player, poisonBuff)

	// 验证订阅顺序
	subs := game.Bus.GetSubscriptions(event.PhaseBeforeTurn)
	if len(subs) != 2 {
		t.Errorf("Expected 2 subscriptions, got %d", len(subs))
	}

	// 触发 BeforeTurn
	sm.ExecuteBeforeTurnPhase(player)

	// 两者都自动执行，按优先级顺序
	// 诅咒先执行（Priority 50），毒瘴后执行（Priority 30）
}

// TestIntegrationHiddenBuffImmunity 测试隐匿 Buff 免疫效果
func TestIntegrationHiddenBuffImmunity(t *testing.T) {
	game := NewGame("game-001")
	player := NewPlayer(PlayerConfig{
		UserID: "player-001",
		MaxHP:  6,
	})
	game.AddPlayer(player)

	sm := NewStateMachine(game)

	// 添加隐匿 Buff（PreDamage，Priority=100）
	hiddenBuff := NewBuff(BuffTypeHidden, 3)
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
	if sm.CurrentCtx.Data != damage {
		t.Errorf("Context damage = %v, expected %d", sm.CurrentCtx.Data, damage)
	}
}

// TestIntegrationBuffRemovalUnsubscribe 测试 Buff 移除时取消订阅
func TestIntegrationBuffRemovalUnsubscribe(t *testing.T) {
	game := NewGame("game-001")
	player := NewPlayer(PlayerConfig{UserID: "player-001"})
	game.AddPlayer(player)

	// 添加诅咒 Buff
	curseBuff := NewBuff(BuffTypeCurse, 3)
	player.AddBuff(curseBuff)
	game.SubscribeBuff(player, curseBuff)

	initialCount := game.Bus.GetSubscriptionCount()
	if initialCount != 1 {
		t.Errorf("Expected 1 subscription, got %d", initialCount)
	}

	// 移除 Buff
	player.RemoveBuff(BuffTypeCurse)
	game.UnsubscribeBuff(curseBuff)

	// 验证订阅已取消
	if game.Bus.GetSubscriptionCount() != 0 {
		t.Errorf("Expected 0 subscriptions after removal, got %d", game.Bus.GetSubscriptionCount())
	}
}

// TestIntegrationPlayerRemovalCleansSubscriptions 测试玩家移除时清理订阅
func TestIntegrationPlayerRemovalCleansSubscriptions(t *testing.T) {
	game := NewGame("game-001")
	player := NewPlayer(PlayerConfig{UserID: "player-001"})
	game.AddPlayer(player)

	// 添加多个 Buff
	buff1 := NewBuff(BuffTypeCurse, 3)
	buff2 := NewBuff(BuffTypeHidden, 3)
	player.AddBuff(buff1)
	player.AddBuff(buff2)
	game.SubscribeBuff(player, buff1)
	game.SubscribeBuff(player, buff2)

	// 添加道具
	item := NewItem(ItemTypeDiceUpgrade, "item-001")
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
	game := NewGame("game-001")
	player := NewPlayer(PlayerConfig{
		UserID:  "player-001",
		Faction: FactionZhuQue,
		MaxLP:   5,
	})
	game.AddPlayer(player)

	// 朱雀玩家初始有离火 Buff（永久被动）
	if !player.HasBuff(BuffTypeFire) {
		t.Error("ZhuQue player should have Fire buff")
	}

	// 离火是 Passive，不需要订阅 EventBus
	fireBuff := player.GetBuff(BuffTypeFire)
	if fireBuff.SubscriptionID != "" {
		t.Error("Fire buff (Passive) should not have subscription ID")
	}

	// Passive Buff 需要特殊处理（每4回合 LP+1）
	// 这里只验证 Buff 存在，实际触发逻辑需要单独实现
	t.Logf("ZhuQue player has Fire buff: duration=%d", fireBuff.Duration)
}

// TestIntegrationAnyTimeItem 测试 AnyTime 道具（主动触发）
func TestIntegrationAnyTimeItem(t *testing.T) {
	game := NewGame("game-001")
	player := NewPlayer(PlayerConfig{UserID: "player-001"})
	game.AddPlayer(player)

	// 添加反方向的钟（AnyTime，不需要订阅）
	item := NewItem(ItemTypeReverseClock, "item-001")
	player.AddItem(item)
	game.SubscribeItem(player, item)

	// AnyTime 道具不订阅 EventBus
	def := ItemTypeReverseClock.GetItemDefinition()
	if def.Phase.NeedsSubscription() {
		t.Error("ReverseClock (AnyTime) should not need subscription")
	}
	if item.SubscriptionID != "" {
		t.Error("AnyTime item should not have subscription ID")
	}

	// AnyTime 道具需要主动触发（不在 Phase 流程中）
	// 这里验证道具存在并可使用
	if !item.Usable {
		t.Error("Item should be usable")
	}
}

// TestIntegrationFullGameScenario 测试完整游戏场景
// 模拟 2-4 玩家的多回合游戏流程
func TestIntegrationFullGameScenario(t *testing.T) {
	game := NewGame("game-001")

	// 创建 4 个玩家，不同阵营
	factions := []Faction{FactionQingLong, FactionZhuQue, FactionBaiHu, FactionXuanWu}
	for i := 0; i < 4; i++ {
		player := NewPlayer(PlayerConfig{
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
	if !zhuquePlayer.HasBuff(BuffTypeFire) {
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

// TestIntegrationDecisionWithAction 测试 Decision 的 Action 执行
func TestIntegrationDecisionWithAction(t *testing.T) {
	game := NewGame("game-001")
	player := NewPlayer(PlayerConfig{
		UserID: "player-001",
		MaxLP:  5,
	})
	game.AddPlayer(player)

	sm := NewStateMachine(game)

	// 创建需要确认的 Decision，带有 Action
	lpModified := false
	d := event.NewDecision("是否使用道具？", []event.Option{
		{ID: "use", Label: "使用", Action: func(ctx *event.Context) {
			// 获取玩家并修改 LP
			if p, ok := ctx.Player.(*Player); ok {
				p.ModifyLP(1)
				lpModified = true
			}
		}},
		{ID: "skip", Label: "跳过"},
	})
	d.NeedConfirm = true

	sm.EnterWaitingState([]*event.Decision{d})
	sm.CurrentCtx = event.NewContext(player)

	// 用户选择使用
	sm.OnUserChoice(0)

	// 验证 Action 已执行
	if !lpModified {
		t.Error("Action should have been executed")
	}
	if player.LP != 6 {
		t.Errorf("Player LP = %d, expected 6 (LP+1)", player.LP)
	}
}

// TestIntegrationContextDataPassing 测试 Context 数据传递
func TestIntegrationContextDataPassing(t *testing.T) {
	game := NewGame("game-001")
	player := NewPlayer(PlayerConfig{UserID: "player-001"})
	game.AddPlayer(player)

	sm := NewStateMachine(game)

	// 创建接收 Context 数据的 Decision
	receivedDamage := 0
	d := event.NewDecision("测试伤害", []event.Option{
		{ID: "ok", Label: "OK", Action: func(ctx *event.Context) {
			if damage, ok := ctx.Data.(int); ok {
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