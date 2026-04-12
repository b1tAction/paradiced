package engine

import (
	"testing"

	"github.com/b1tAction/Fated/internal/core"
	"github.com/b1tAction/Fated/pkg/event"
)

// ========== Integration Tests: Game + EventBus + StateMachine ==========

// TestIntegrationFullTurnFlow 测试完整的回合流程
func TestIntegrationFullTurnFlow(t *testing.T) {
	// 1. 创建游戏和玩家
	game := NewGame("game-001")
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
	game := NewGame("game-001")
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
	if current.Prompt != core.ItemTypeDiceUpgrade.GetItemDefinition().Desc {
		t.Errorf("Decision prompt = %s, expected %s",
			current.Prompt, core.ItemTypeDiceUpgrade.GetItemDefinition().Desc)
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
	game := NewGame("game-001")
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
	if sm.CurrentCtx.Data != damage {
		t.Errorf("Context damage = %v, expected %d", sm.CurrentCtx.Data, damage)
	}
}

// TestIntegrationBuffRemovalUnsubscribe 测试 Buff 移除时取消订阅
func TestIntegrationBuffRemovalUnsubscribe(t *testing.T) {
	game := NewGame("game-001")
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
	game := NewGame("game-001")
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
	game := NewGame("game-001")
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
	def := core.BuffTypeFire.GetBuffDefinition()
	if def.Phase.NeedsSubscription() && fireBuff.SubscriptionID == "" {
		t.Error("Fire buff (BeforeTurn) should have subscription ID")
	}

	t.Logf("ZhuQue player has Fire buff: duration=%d, phase=%s", fireBuff.Duration, def.Phase.String())
}

// TestIntegrationAnyTimeItem 测试 AnyTime 道具（主动触发）
func TestIntegrationAnyTimeItem(t *testing.T) {
	game := NewGame("game-001")
	player := core.NewPlayer(core.PlayerConfig{UserID: "player-001"})
	game.AddPlayer(player)

	// 添加反方向的钟（AnyTime，不需要订阅）
	item := core.NewItem(core.ItemTypeReverseClock, "item-001")
	player.AddItem(item)
	game.SubscribeItem(player, item)

	// AnyTime 道具不订阅 EventBus
	def := core.ItemTypeReverseClock.GetItemDefinition()
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
	game := NewGame("game-001")

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
	game := NewGame("game-001")
	player := core.NewPlayer(core.PlayerConfig{UserID: "player-001"})
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