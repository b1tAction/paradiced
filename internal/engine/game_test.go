package engine

import (
	"testing"

	"github.com/b1tAction/paradiced/internal/core"
	"github.com/b1tAction/paradiced/pkg/constants"
	"github.com/b1tAction/paradiced/pkg/event"
	"github.com/b1tAction/paradiced/pkg/id"
	"github.com/b1tAction/paradiced/pkg/rng"
)

// ========== Game Creation Tests ==========

func TestNewGame(t *testing.T) {
	gameID := id.NewGameID()
	game := NewGame(gameID, 0)
	if game == nil {
		t.Fatal("NewGame should not return nil")
	}
	if game.ID != gameID {
		t.Errorf("Game.ID = %s, expected %s", game.ID.UUID(), gameID.UUID())
	}
	if game.Bus == nil {
		t.Error("Game.Bus should not be nil")
	}
	if game.Players == nil {
		t.Error("Game.Players should not be nil")
	}
	if game.State == nil {
		t.Error("Game.State should not be nil")
	}
	if game.RNG == nil {
		t.Error("Game.RNG should not be nil")
	}
	if game.Draw == nil {
		t.Error("Game.Draw should not be nil")
	}
	if game.State.Round != 1 {
		t.Errorf("Initial Round = %d, expected 1", game.State.Round)
	}
	if game.State.Turn != 0 {
		t.Errorf("Initial Turn = %d, expected 0", game.State.Turn)
	}
	if game.State.CurrentPhase != "init" {
		t.Errorf("Initial Phase = %s, expected init", game.State.CurrentPhase)
	}
}

func TestNewGameWithSeed(t *testing.T) {
	game1 := NewGame(id.NewGameID(), 42)
	game2 := NewGame(id.NewGameID(), 42)

	// Same seed should produce same draw sequence
	et1 := game1.Draw.DrawEvent(rng.PoolTypeGood, 4)
	et2 := game2.Draw.DrawEvent(rng.PoolTypeGood, 4)

	if et1 != et2 {
		t.Errorf("Same seed should produce same draw: %s vs %s", string(et1), string(et2))
	}
}

func TestGameBusCreation(t *testing.T) {
	gameID := id.NewGameID()
	game := NewGame(gameID, 0)
	if game.Bus.GameID != gameID.UUID() {
		t.Errorf("Bus.GameID = %s, expected %s", game.Bus.GameID, gameID.UUID())
	}
	if game.Bus.GetSubscriptionCount() != 0 {
		t.Errorf("Initial subscription count should be 0")
	}
}

// ========== Player Management Tests ==========

func TestGameAddPlayer(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	playerID := id.NewPlayerID()
	player := core.NewPlayer(core.PlayerConfig{
		ID:      playerID,
		Faction: core.FactionQingLong,
	})

	game.AddPlayer(player)

	if len(game.Players) != 1 {
		t.Errorf("Players count = %d, expected 1", len(game.Players))
	}
	if game.Players[0].ID != playerID {
		t.Errorf("Player.ID = %s, expected %s", game.Players[0].ID.UUID(), playerID.UUID())
	}
}

func TestGameAddMultiplePlayers(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)

	for i := 1; i <= 4; i++ {
		player := core.NewPlayer(core.PlayerConfig{
			ID:      id.NewPlayerID(),
			Faction: core.Faction(i),
		})
		game.AddPlayer(player)
	}

	if len(game.Players) != 4 {
		t.Errorf("Players count = %d, expected 4", len(game.Players))
	}
}

func TestGameRemovePlayer(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	playerID1 := id.NewPlayerID()
	playerID2 := id.NewPlayerID()
	player1 := core.NewPlayer(core.PlayerConfig{ID: playerID1})
	player2 := core.NewPlayer(core.PlayerConfig{ID: playerID2})

	game.AddPlayer(player1)
	game.AddPlayer(player2)

	game.RemovePlayer(playerID1)

	if len(game.Players) != 1 {
		t.Errorf("Players count = %d, expected 1", len(game.Players))
	}
	if game.Players[0].ID != playerID2 {
		t.Errorf("Remaining player.ID = %s, expected %s", game.Players[0].ID.UUID(), playerID2.UUID())
	}
}

func TestGameRemovePlayerCancelsSubscriptions(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	playerID := id.NewPlayerID()
	player := core.NewPlayer(core.PlayerConfig{ID: playerID})

	// 添加一个需要订阅的 Buff
	buff := core.NewBuff(core.BuffTypeCurse, 3)
	player.AddBuff(buff)
	game.AddPlayer(player)

	// 验证订阅已创建
	if game.Bus.GetSubscriptionCount() == 0 {
		t.Error("Should have subscriptions after adding player with buff")
	}

	// 移除玩家
	game.RemovePlayer(playerID)

	// 验证订阅已取消
	if game.Bus.GetSubscriptionCount() != 0 {
		t.Errorf("Subscription count = %d, expected 0 after player removal", game.Bus.GetSubscriptionCount())
	}
}

func TestGameGetPlayer(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	playerID1 := id.NewPlayerID()
	playerID2 := id.NewPlayerID()
	player1 := core.NewPlayer(core.PlayerConfig{ID: playerID1})
	player2 := core.NewPlayer(core.PlayerConfig{ID: playerID2})

	game.AddPlayer(player1)
	game.AddPlayer(player2)

	found := game.GetPlayer(playerID1)
	if found == nil {
		t.Error("GetPlayer should find player1")
	}
	if found.ID != playerID1 {
		t.Errorf("Found player.ID = %s, expected %s", found.ID.UUID(), playerID1.UUID())
	}

	notFound := game.GetPlayer(id.NewPlayerID())
	if notFound != nil {
		t.Error("GetPlayer should return nil for non-existent player")
	}
}

func TestGameGetCurrentPlayer(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	playerID1 := id.NewPlayerID()
	playerID2 := id.NewPlayerID()
	player1 := core.NewPlayer(core.PlayerConfig{ID: playerID1})
	player2 := core.NewPlayer(core.PlayerConfig{ID: playerID2})

	game.AddPlayer(player1)
	game.AddPlayer(player2)

	current := game.GetCurrentPlayer()
	if current == nil {
		t.Error("GetCurrentPlayer should not return nil")
	}
	if current.ID != playerID1 {
		t.Errorf("Current player.ID = %s, expected %s (Turn=0)", current.ID.UUID(), playerID1.UUID())
	}

	// 切换回合
	game.NextTurn()
	current = game.GetCurrentPlayer()
	if current.ID != playerID2 {
		t.Errorf("After NextTurn, current player.ID = %s, expected %s", current.ID.UUID(), playerID2.UUID())
	}
}

func TestGameNextTurn(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	for i := 1; i <= 3; i++ {
		player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
		game.AddPlayer(player)
	}

	// Turn 0 -> 1
	game.NextTurn()
	if game.State.Turn != 1 {
		t.Errorf("Turn = %d, expected 1", game.State.Turn)
	}
	if game.State.Round != 1 {
		t.Errorf("Round should still be 1")
	}

	// Turn 1 -> 2
	game.NextTurn()
	if game.State.Turn != 2 {
		t.Errorf("Turn = %d, expected 2", game.State.Turn)
	}

	// Turn 2 -> 0 (wrap around, Round increases)
	game.NextTurn()
	if game.State.Turn != 0 {
		t.Errorf("Turn = %d, expected 0 (wrap around)", game.State.Turn)
	}
	if game.State.Round != 2 {
		t.Errorf("Round = %d, expected 2 (increased after wrap)", game.State.Round)
	}
}

// ========== Buff Subscription Tests ==========

func TestGameSubscribeBuff(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	playerID := id.NewPlayerID()
	player := core.NewPlayer(core.PlayerConfig{ID: playerID})
	game.AddPlayer(player)

	// 添加需要订阅的 Buff (诅咒)
	buff := core.NewBuff(core.BuffTypeCurse, 3)
	game.SubscribeBuff(player, buff)

	// 验证订阅已创建
	config := core.GetBuffHandlerConfig(core.BuffTypeCurse)
	for _, phase := range config.GetPhases() {
		if phase.NeedsSubscription() {
			if game.Bus.GetSubscriptionCount() != 1 {
				t.Errorf("Subscription count = %d, expected 1", game.Bus.GetSubscriptionCount())
			}
			if len(buff.SubscriptionIDs) != 1 {
				t.Error("Buff should have SubscriptionIDs")
			}
		}
	}
}

func TestGameSubscribePassiveBuff(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	playerID := id.NewPlayerID()
	player := core.NewPlayer(core.PlayerConfig{ID: playerID})
	game.AddPlayer(player)

	// 添加永久 Buff (离火，现在使用 BeforeTurn)
	buff := core.NewBuff(core.BuffTypeFire, -1)
	game.SubscribeBuff(player, buff)

	// 离火现在需要订阅 BeforeTurn（每4回合检查）
	config := core.GetBuffHandlerConfig(core.BuffTypeFire)
	for _, phase := range config.GetPhases() {
		if phase.NeedsSubscription() {
			if len(buff.SubscriptionIDs) == 0 {
				t.Error("Fire Buff should have SubscriptionIDs (BeforeTurn)")
			}
		}
	}
}

func TestGameUnsubscribeBuff(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	playerID := id.NewPlayerID()
	player := core.NewPlayer(core.PlayerConfig{ID: playerID})
	game.AddPlayer(player)

	// 订阅 Buff
	buff := core.NewBuff(core.BuffTypeCurse, 3)
	game.SubscribeBuff(player, buff)

	initialCount := game.Bus.GetSubscriptionCount()

	// 取消订阅
	game.UnsubscribeBuff(buff)

	if game.Bus.GetSubscriptionCount() != initialCount-1 {
		t.Errorf("Subscription count should decrease after unsubscribe")
	}
	if len(buff.SubscriptionIDs) != 0 {
		t.Error("Buff SubscriptionIDs should be empty after unsubscribe")
	}
}

func TestGameSubscribeBuffByPlayerAdd(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	playerID := id.NewPlayerID()
	player := core.NewPlayer(core.PlayerConfig{
		ID:      playerID,
		Faction: core.FactionZhuQue, // 朱雀初始携带离火 Buff
	})

	// 朱雀玩家初始有离火 Buff（永久被动）
	game.AddPlayer(player)

	// 验证玩家有离火 Buff
	if !player.HasBuff(core.BuffTypeFire) {
		t.Error("ZhuQue player should have Fire buff")
	}

	// 离火现在使用 BeforeTurn，需要订阅
	config := core.GetBuffHandlerConfig(core.BuffTypeFire)
	for _, phase := range config.GetPhases() {
		if phase.NeedsSubscription() {
			// BeforeTurn 需要订阅
			if game.Bus.GetSubscriptionCount() != 1 {
				t.Errorf("Fire Buff should subscribe to BeforeTurn, count = %d", game.Bus.GetSubscriptionCount())
			}
		}
	}
}

// ========== Item Subscription Tests ==========

func TestGameSubscribeItem(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	playerID := id.NewPlayerID()
	player := core.NewPlayer(core.PlayerConfig{ID: playerID})
	game.AddPlayer(player)

	// 添加需要订阅的道具 (骰子升级卡)
	item := core.NewItem(core.ItemTypeDiceUpgrade)
	game.SubscribeItem(player, item)

	// 验证订阅已创建
	config := core.GetItemHandlerConfig(core.ItemTypeDiceUpgrade)
	if config.Phase.NeedsSubscription() {
		if game.Bus.GetSubscriptionCount() != 1 {
			t.Errorf("Subscription count = %d, expected 1", game.Bus.GetSubscriptionCount())
		}
		if item.SubscriptionID == "" {
			t.Error("Item should have SubscriptionID")
		}
	}
}

func TestGameSubscribeAnyTimeItem(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	playerID := id.NewPlayerID()
	player := core.NewPlayer(core.PlayerConfig{ID: playerID})
	game.AddPlayer(player)

	// 添加 AnyTime 道具 (反方向的钟)
	item := core.NewItem(core.ItemTypeReverseClock)
	game.SubscribeItem(player, item)

	// AnyTime 道具不需要订阅（主动触发）
	config := core.GetItemHandlerConfig(core.ItemTypeReverseClock)
	if config.Phase == constants.PhaseAnyTime {
		if item.SubscriptionID != "" {
			t.Error("AnyTime Item should not have SubscriptionID")
		}
	}
}

func TestGameUnsubscribeItem(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	playerID := id.NewPlayerID()
	player := core.NewPlayer(core.PlayerConfig{ID: playerID})
	game.AddPlayer(player)

	// 订阅道具
	item := core.NewItem(core.ItemTypeDiceUpgrade)
	game.SubscribeItem(player, item)

	initialCount := game.Bus.GetSubscriptionCount()

	// 取消订阅
	game.UnsubscribeItem(item)

	if game.Bus.GetSubscriptionCount() != initialCount-1 {
		t.Errorf("Subscription count should decrease after unsubscribe")
	}
	if item.SubscriptionID != "" {
		t.Error("Item SubscriptionID should be cleared after unsubscribe")
	}
}

// ========== Multiple Subscriptions Tests ==========

func TestGameMultipleBuffSubscriptions(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	playerID := id.NewPlayerID()
	player := core.NewPlayer(core.PlayerConfig{ID: playerID})
	game.AddPlayer(player)

	// 添加多个 Buff
	buff1 := core.NewBuff(core.BuffTypeCurse, 3)  // BeforeTurn
	buff2 := core.NewBuff(core.BuffTypeDivine, 3) // BeforeTurn
	buff3 := core.NewBuff(core.BuffTypeHidden, 3) // PreDamage

	player.AddBuff(buff1)
	player.AddBuff(buff2)
	player.AddBuff(buff3)

	game.SubscribeBuff(player, buff1)
	game.SubscribeBuff(player, buff2)
	game.SubscribeBuff(player, buff3)

	// 应该有 3 个订阅
	if game.Bus.GetSubscriptionCount() != 3 {
		t.Errorf("Subscription count = %d, expected 3", game.Bus.GetSubscriptionCount())
	}
}

func TestGameMixedSubscriptions(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	playerID := id.NewPlayerID()
	player := core.NewPlayer(core.PlayerConfig{ID: playerID})
	game.AddPlayer(player)

	// 添加 Buff 和道具
	buff := core.NewBuff(core.BuffTypeCurse, 3)
	item := core.NewItem(core.ItemTypeDiceUpgrade)

	player.AddBuff(buff)
	player.AddItem(item)

	game.SubscribeBuff(player, buff)
	game.SubscribeItem(player, item)

	// 验证订阅数（两个都需要订阅）
	if game.Bus.GetSubscriptionCount() != 2 {
		t.Errorf("Subscription count = %d, expected 2", game.Bus.GetSubscriptionCount())
	}
}

// ========== Decision Creation Tests ==========

func TestGameCreateBuffDecision(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	buff := core.NewBuff(core.BuffTypeCurse, 3)
	def := core.GetBuffDefinition(core.BuffTypeCurse)
	config := core.GetBuffHandlerConfig(core.BuffTypeCurse)

	decision := game.createBuffDecision(buff, def, config)

	if decision == nil {
		t.Fatal("createBuffDecision should not return nil")
	}
	if decision.Prompt != def.Desc {
		t.Errorf("Decision Prompt = %s, expected %s", decision.Prompt, def.Desc)
	}
	// Buff 默认不需要确认
	if decision.NeedConfirm {
		t.Error("Buff Decision should not need confirm by default")
	}
}

func TestGameCreateItemDecision(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	item := core.NewItem(core.ItemTypeDiceUpgrade)
	def := core.GetItemDefinition(core.ItemTypeDiceUpgrade)
	config := core.GetItemHandlerConfig(core.ItemTypeDiceUpgrade)

	decision := game.createItemDecision(item, def, config)

	if decision == nil {
		t.Fatal("createItemDecision should not return nil")
	}
	if decision.Prompt != def.Desc {
		t.Errorf("Decision Prompt = %s, expected %s", decision.Prompt, def.Desc)
	}
	// 道具默认需要确认
	if !decision.NeedConfirm {
		t.Error("Item Decision should need confirm by default")
	}
	if len(decision.Options) != 2 {
		t.Errorf("Item Decision should have 2 options (use/skip), got %d", len(decision.Options))
	}
}

// ========== Buff Lifecycle Tests ==========

func TestGameApplyBuffToPlayer(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	playerID := id.NewPlayerID()
	player := core.NewPlayer(core.PlayerConfig{ID: playerID})
	game.AddPlayer(player)

	// 使用 ApplyBuffToPlayer 添加 Buff
	buff := core.NewBuff(core.BuffTypeCurse, 3)
	err := game.ApplyBuffToPlayer(player, buff)

	if err != nil {
		t.Errorf("ApplyBuffToPlayer should not return error: %v", err)
	}

	// 验证 Buff 已添加
	if !player.HasBuff(core.BuffTypeCurse) {
		t.Error("Player should have Curse buff")
	}

	// 验证订阅已创建
	if game.Bus.GetSubscriptionCount() == 0 {
		t.Error("Should have subscriptions after ApplyBuffToPlayer")
	}

	// 验证 SubscriptionIDs 已填充
	if len(buff.SubscriptionIDs) == 0 {
		t.Error("Buff should have SubscriptionIDs")
	}
}

func TestGameRemoveBuffFromPlayer(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	playerID := id.NewPlayerID()
	player := core.NewPlayer(core.PlayerConfig{ID: playerID})
	game.AddPlayer(player)

	// 先添加 Buff
	buff := core.NewBuff(core.BuffTypeCurse, 3)
	game.ApplyBuffToPlayer(player, buff)

	// 验证初始状态
	initialSubCount := game.Bus.GetSubscriptionCount()
	if initialSubCount == 0 {
		t.Fatal("Should have subscriptions before removal")
	}

	// 移除 Buff
	result := game.RemoveBuffFromPlayer(player, buff)

	if !result {
		t.Error("RemoveBuffFromPlayer should return true")
	}

	// 验证 Buff 已移除
	if player.HasBuff(core.BuffTypeCurse) {
		t.Error("Player should not have Curse buff after removal")
	}

	// 验证订阅已取消
	if game.Bus.GetSubscriptionCount() != initialSubCount-1 {
		t.Errorf("Subscription count should decrease after removal")
	}
}

func TestGameBroadcastBuffApplied(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	playerID := id.NewPlayerID()
	player := core.NewPlayer(core.PlayerConfig{ID: playerID})
	game.AddPlayer(player)

	// 订阅 PhaseOnBuffApplied（用于测试接收）
	received := false
	buffTypeCheck := core.BuffTypeNone
	d := event.NewAutoDecision("测试", []event.Option{
		{ID: "ok", Label: "OK", Action: func(ctx *event.Context) {
			received = true
			// 验证 Context.Data 包含 Buff
			buff, ok := ctx.GetData().(*core.Buff)
			if !ok {
				t.Error("Context.Data should be Buff")
				return
			}
			buffTypeCheck = buff.Type
		}},
	})
	game.Bus.Subscribe(constants.PhaseOnBuffApplied, playerID, "test-listener", "test", d)

	// 广播 Applied 事件
	buff := core.NewBuff(core.BuffTypeCurse, 3)
	game.BroadcastBuffApplied(player, buff)

	// 验证事件已触发
	if !received {
		t.Error("PhaseOnBuffApplied should be triggered")
	}
	if buffTypeCheck != core.BuffTypeCurse {
		t.Errorf("Buff.Type = %v, expected Curse", buffTypeCheck)
	}
}

func TestGameBroadcastBuffRemoved(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	playerID := id.NewPlayerID()
	player := core.NewPlayer(core.PlayerConfig{ID: playerID})
	game.AddPlayer(player)

	// 订阅 PhaseOnBuffRemoved（用于测试接收）
	received := false
	buffTypeCheck := core.BuffTypeNone
	d := event.NewAutoDecision("测试", []event.Option{
		{ID: "ok", Label: "OK", Action: func(ctx *event.Context) {
			received = true
			// 验证 Context.Data 包含 Buff
			buff, ok := ctx.GetData().(*core.Buff)
			if !ok {
				t.Error("Context.Data should be Buff")
				return
			}
			buffTypeCheck = buff.Type
		}},
	})
	game.Bus.Subscribe(constants.PhaseOnBuffRemoved, playerID, "test-listener", "test", d)

	// 广播 Removed 事件
	buff := core.NewBuff(core.BuffTypeDivine, 3)
	game.BroadcastBuffRemoved(player, buff)

	// 验证事件已触发
	if !received {
		t.Error("PhaseOnBuffRemoved should be triggered")
	}
	if buffTypeCheck != core.BuffTypeDivine {
		t.Errorf("Buff.Type = %v, expected Divine", buffTypeCheck)
	}
}

func TestGameGetActiveBuffCount(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	playerID := id.NewPlayerID()
	player := core.NewPlayer(core.PlayerConfig{ID: playerID})
	game.AddPlayer(player)

	// 初始 Buff 数量
	if game.GetActiveBuffCount(playerID) != 0 {
		t.Error("Initial buff count should be 0")
	}

	// 添加 Buff
	buff1 := core.NewBuff(core.BuffTypeCurse, 3)
	buff2 := core.NewBuff(core.BuffTypeDivine, 3)
	game.ApplyBuffToPlayer(player, buff1)
	game.ApplyBuffToPlayer(player, buff2)

	// 验证 Buff 数量
	if game.GetActiveBuffCount(playerID) != 2 {
		t.Errorf("Buff count = %d, expected 2", game.GetActiveBuffCount(playerID))
	}

	// 不存在的玩家
	if game.GetActiveBuffCount(id.NewPlayerID()) != 0 {
		t.Error("Unknown player buff count should be 0")
	}
}

func TestGameGetBuffSubscriptionCount(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	playerID := id.NewPlayerID()
	player := core.NewPlayer(core.PlayerConfig{ID: playerID})
	game.AddPlayer(player)

	// 添加 Buff
	buff := core.NewBuff(core.BuffTypeCurse, 3)
	game.SubscribeBuff(player, buff)

	// 验证订阅数量
	count := game.GetBuffSubscriptionCount(buff)
	if count != 1 {
		t.Errorf("Subscription count = %d, expected 1", count)
	}
}
