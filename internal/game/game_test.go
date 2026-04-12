package game

import (
	"testing"

	"github.com/b1tAction/Fated/pkg/event"
)

// ========== Game Creation Tests ==========

func TestNewGame(t *testing.T) {
	game := NewGame("game-001")
	if game == nil {
		t.Fatal("NewGame should not return nil")
	}
	if game.ID != "game-001" {
		t.Errorf("Game.ID = %s, expected game-001", game.ID)
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

func TestGameBusCreation(t *testing.T) {
	game := NewGame("game-002")
	if game.Bus.GameID != "game-002" {
		t.Errorf("Bus.GameID = %s, expected game-002", game.Bus.GameID)
	}
	if game.Bus.GetSubscriptionCount() != 0 {
		t.Errorf("Initial subscription count should be 0")
	}
}

// ========== Player Management Tests ==========

func TestGameAddPlayer(t *testing.T) {
	game := NewGame("game-001")
	player := NewPlayer(PlayerConfig{
		UserID:  "player-001",
		Faction: FactionQingLong,
	})

	game.AddPlayer(player)

	if len(game.Players) != 1 {
		t.Errorf("Players count = %d, expected 1", len(game.Players))
	}
	if game.Players[0].UserID != "player-001" {
		t.Errorf("Player.UserID = %s, expected player-001", game.Players[0].UserID)
	}
}

func TestGameAddMultiplePlayers(t *testing.T) {
	game := NewGame("game-001")

	for i := 1; i <= 4; i++ {
		player := NewPlayer(PlayerConfig{
			UserID:  "player-" + string(rune('0'+i)),
			Faction: Faction(i),
		})
		game.AddPlayer(player)
	}

	if len(game.Players) != 4 {
		t.Errorf("Players count = %d, expected 4", len(game.Players))
	}
}

func TestGameRemovePlayer(t *testing.T) {
	game := NewGame("game-001")
	player1 := NewPlayer(PlayerConfig{UserID: "player-001"})
	player2 := NewPlayer(PlayerConfig{UserID: "player-002"})

	game.AddPlayer(player1)
	game.AddPlayer(player2)

	game.RemovePlayer("player-001")

	if len(game.Players) != 1 {
		t.Errorf("Players count = %d, expected 1", len(game.Players))
	}
	if game.Players[0].UserID != "player-002" {
		t.Errorf("Remaining player.UserID = %s, expected player-002", game.Players[0].UserID)
	}
}

func TestGameRemovePlayerCancelsSubscriptions(t *testing.T) {
	game := NewGame("game-001")
	player := NewPlayer(PlayerConfig{UserID: "player-001"})

	// 添加一个需要订阅的 Buff
	buff := NewBuff(BuffTypeCurse, 3)
	player.AddBuff(buff)
	game.AddPlayer(player)

	// 验证订阅已创建
	if game.Bus.GetSubscriptionCount() == 0 {
		t.Error("Should have subscriptions after adding player with buff")
	}

	// 移除玩家
	game.RemovePlayer("player-001")

	// 验证订阅已取消
	if game.Bus.GetSubscriptionCount() != 0 {
		t.Errorf("Subscription count = %d, expected 0 after player removal", game.Bus.GetSubscriptionCount())
	}
}

func TestGameGetPlayer(t *testing.T) {
	game := NewGame("game-001")
	player1 := NewPlayer(PlayerConfig{UserID: "player-001"})
	player2 := NewPlayer(PlayerConfig{UserID: "player-002"})

	game.AddPlayer(player1)
	game.AddPlayer(player2)

	found := game.GetPlayer("player-001")
	if found == nil {
		t.Error("GetPlayer should find player-001")
	}
	if found.UserID != "player-001" {
		t.Errorf("Found player.UserID = %s, expected player-001", found.UserID)
	}

	notFound := game.GetPlayer("player-999")
	if notFound != nil {
		t.Error("GetPlayer should return nil for non-existent player")
	}
}

func TestGameGetCurrentPlayer(t *testing.T) {
	game := NewGame("game-001")
	player1 := NewPlayer(PlayerConfig{UserID: "player-001"})
	player2 := NewPlayer(PlayerConfig{UserID: "player-002"})

	game.AddPlayer(player1)
	game.AddPlayer(player2)

	current := game.GetCurrentPlayer()
	if current == nil {
		t.Error("GetCurrentPlayer should not return nil")
	}
	if current.UserID != "player-001" {
		t.Errorf("Current player.UserID = %s, expected player-001 (Turn=0)", current.UserID)
	}

	// 切换回合
	game.NextTurn()
	current = game.GetCurrentPlayer()
	if current.UserID != "player-002" {
		t.Errorf("After NextTurn, current player.UserID = %s, expected player-002", current.UserID)
	}
}

func TestGameNextTurn(t *testing.T) {
	game := NewGame("game-001")
	for i := 1; i <= 3; i++ {
		player := NewPlayer(PlayerConfig{UserID: "player-00" + string(rune('0'+i))})
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
	game := NewGame("game-001")
	player := NewPlayer(PlayerConfig{UserID: "player-001"})
	game.AddPlayer(player)

	// 添加需要订阅的 Buff (诅咒)
	buff := NewBuff(BuffTypeCurse, 3)
	game.SubscribeBuff(player, buff)

	// 验证订阅已创建
	def := BuffTypeCurse.GetBuffDefinition()
	if def.Phase.NeedsSubscription() {
		if game.Bus.GetSubscriptionCount() != 1 {
			t.Errorf("Subscription count = %d, expected 1", game.Bus.GetSubscriptionCount())
		}
		if buff.SubscriptionID == "" {
			t.Error("Buff should have SubscriptionID")
		}
	}
}

func TestGameSubscribePassiveBuff(t *testing.T) {
	game := NewGame("game-001")
	player := NewPlayer(PlayerConfig{UserID: "player-001"})
	game.AddPlayer(player)

	// 添加永久 Buff (离火，现在使用 BeforeTurn)
	buff := NewBuff(BuffTypeFire, -1)
	game.SubscribeBuff(player, buff)

	// 离火现在需要订阅 BeforeTurn（每4回合检查）
	def := BuffTypeFire.GetBuffDefinition()
	if def.Phase.NeedsSubscription() {
		if buff.SubscriptionID == "" {
			t.Error("Fire Buff should have SubscriptionID (BeforeTurn)")
		}
	}
}

func TestGameUnsubscribeBuff(t *testing.T) {
	game := NewGame("game-001")
	player := NewPlayer(PlayerConfig{UserID: "player-001"})
	game.AddPlayer(player)

	// 订阅 Buff
	buff := NewBuff(BuffTypeCurse, 3)
	game.SubscribeBuff(player, buff)

	initialCount := game.Bus.GetSubscriptionCount()

	// 取消订阅
	game.UnsubscribeBuff(buff)

	if game.Bus.GetSubscriptionCount() != initialCount-1 {
		t.Errorf("Subscription count should decrease after unsubscribe")
	}
	if buff.SubscriptionID != "" {
		t.Error("Buff SubscriptionID should be cleared after unsubscribe")
	}
}

func TestGameSubscribeBuffByPlayerAdd(t *testing.T) {
	game := NewGame("game-001")
	player := NewPlayer(PlayerConfig{
		UserID:  "player-001",
		Faction: FactionZhuQue, // 朱雀初始携带离火 Buff
	})

	// 朱雀玩家初始有离火 Buff（永久被动，不需要订阅）
	game.AddPlayer(player)

	// 验证玩家有离火 Buff
	if !player.HasBuff(BuffTypeFire) {
		t.Error("ZhuQue player should have Fire buff")
	}

	// 离火现在使用 BeforeTurn，需要订阅
	def := BuffTypeFire.GetBuffDefinition()
	if def.Phase.NeedsSubscription() {
		// BeforeTurn 需要订阅
		if game.Bus.GetSubscriptionCount() != 1 {
			t.Errorf("Fire Buff should subscribe to BeforeTurn, count = %d", game.Bus.GetSubscriptionCount())
		}
	}
}

// ========== Item Subscription Tests ==========

func TestGameSubscribeItem(t *testing.T) {
	game := NewGame("game-001")
	player := NewPlayer(PlayerConfig{UserID: "player-001"})
	game.AddPlayer(player)

	// 添加需要订阅的道具 (骰子升级卡)
	item := NewItem(ItemTypeDiceUpgrade, "item-001")
	game.SubscribeItem(player, item)

	// 验证订阅已创建
	def := ItemTypeDiceUpgrade.GetItemDefinition()
	if def.Phase.NeedsSubscription() {
		if game.Bus.GetSubscriptionCount() != 1 {
			t.Errorf("Subscription count = %d, expected 1", game.Bus.GetSubscriptionCount())
		}
		if item.SubscriptionID == "" {
			t.Error("Item should have SubscriptionID")
		}
	}
}

func TestGameSubscribeAnyTimeItem(t *testing.T) {
	game := NewGame("game-001")
	player := NewPlayer(PlayerConfig{UserID: "player-001"})
	game.AddPlayer(player)

	// 添加 AnyTime 道具 (反方向的钟)
	item := NewItem(ItemTypeReverseClock, "item-001")
	game.SubscribeItem(player, item)

	// AnyTime 道具不需要订阅（主动触发）
	def := ItemTypeReverseClock.GetItemDefinition()
	if def.Phase == event.PhaseAnyTime {
		if item.SubscriptionID != "" {
			t.Error("AnyTime Item should not have SubscriptionID")
		}
	}
}

func TestGameUnsubscribeItem(t *testing.T) {
	game := NewGame("game-001")
	player := NewPlayer(PlayerConfig{UserID: "player-001"})
	game.AddPlayer(player)

	// 订阅道具
	item := NewItem(ItemTypeDiceUpgrade, "item-001")
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
	game := NewGame("game-001")
	player := NewPlayer(PlayerConfig{UserID: "player-001"})
	game.AddPlayer(player)

	// 添加多个 Buff
	buff1 := NewBuff(BuffTypeCurse, 3)     // BeforeTurn
	buff2 := NewBuff(BuffTypeDivine, 3)    // BeforeTurn
	buff3 := NewBuff(BuffTypeHidden, 3)    // PreDamage

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
	game := NewGame("game-001")
	player := NewPlayer(PlayerConfig{UserID: "player-001"})
	game.AddPlayer(player)

	// 添加 Buff 和道具
	buff := NewBuff(BuffTypeCurse, 3)
	item := NewItem(ItemTypeDiceUpgrade, "item-001")

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
	game := NewGame("game-001")
	buff := NewBuff(BuffTypeCurse, 3)
	def := BuffTypeCurse.GetBuffDefinition()

	decision := game.createBuffDecision(buff, def)

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
	game := NewGame("game-001")
	item := NewItem(ItemTypeDiceUpgrade, "item-001")
	def := ItemTypeDiceUpgrade.GetItemDefinition()

	decision := game.createItemDecision(item, def)

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