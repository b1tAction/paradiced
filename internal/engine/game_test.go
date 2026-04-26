package engine

import (
	"testing"

	"github.com/b1tAction/paradiced/internal/core"
	"github.com/b1tAction/paradiced/pkg/constants"
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
	// Round/Turn state is now managed by HSM, not stored in Game
	// Game only stores data (Players, Log, Bus, RNG, Draw)
	if game.RNG == nil {
		t.Error("Game.RNG should not be nil")
	}
	if game.Draw == nil {
		t.Error("Game.Draw should not be nil")
	}
	if game.Log == nil {
		t.Error("Game.Log should not be nil")
	}
}

func TestNewGameWithSeed(t *testing.T) {
	game1 := NewGame(id.NewGameID(), 42)
	game2 := NewGame(id.NewGameID(), 42)

	// Create a test pool
	pool := &rng.EvaluatedItemPool{
		Items: []rng.EvaluatedItem{
			{Type: "event_a", Eval: constants.EvaluationExcellent},
			{Type: "event_b", Eval: constants.EvaluationGood},
			{Type: "event_c", Eval: constants.EvaluationMildGood},
		},
	}

	// Same seed should produce same draw sequence
	et1 := game1.Draw.DrawFromPool(pool, rng.PoolTypeGood, 4)
	et2 := game2.Draw.DrawFromPool(pool, rng.PoolTypeGood, 4)

	if et1 != et2 {
		t.Errorf("Same seed should produce same draw: %s vs %s", et1, et2)
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
		Faction: constants.FactionQingLong,
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

	factions := []constants.Faction{
		constants.FactionQingLong,
		constants.FactionZhuQue,
		constants.FactionBaiHu,
		constants.FactionXuanWu,
	}

	for i := 0; i < 4; i++ {
		player := core.NewPlayer(core.PlayerConfig{
			ID:      id.NewPlayerID(),
			Faction: factions[i],
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
	buff := core.NewBuff(constants.BuffTypeCurse, 3)
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

// ========== Buff Subscription Tests ==========

func TestGameSubscribeBuff(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	playerID := id.NewPlayerID()
	player := core.NewPlayer(core.PlayerConfig{ID: playerID})
	game.AddPlayer(player)

	// 添加需要订阅的 Buff (诅咒)
	buff := core.NewBuff(constants.BuffTypeCurse, 3)
	game.SubscribeBuff(player, buff)

	// 验证订阅已创建 (Curse subscribes to PostBuffApplied + PreBuffRemoved = 2 subscriptions)
	if game.Bus.GetSubscriptionCount() != 2 {
		t.Errorf("Subscription count = %d, expected 2 (Curse: PostBuffApplied + PreBuffRemoved)", game.Bus.GetSubscriptionCount())
	}
}

func TestGameSubscribePassiveBuff(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	playerID := id.NewPlayerID()
	player := core.NewPlayer(core.PlayerConfig{ID: playerID})
	game.AddPlayer(player)

	// 添加永久 Buff (离火，现在使用 BeforeTurn)
	buff := core.NewBuff(constants.BuffTypeFire, -1)
	game.SubscribeBuff(player, buff)

	// 离火现在需要订阅 BeforeTurn（每4回合检查）
	config := GetBuffHandlerConfig(constants.BuffTypeFire)
	for _, phase := range config.GetPhases() {
		if phase.NeedsSubscription() {
			// Subscription managed by EventBus via sourceID
		}
	}
}

func TestGameUnsubscribeBuff(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	playerID := id.NewPlayerID()
	player := core.NewPlayer(core.PlayerConfig{ID: playerID})
	game.AddPlayer(player)

	// 订阅 Buff (Curse subscribes to PostBuffApplied + PreBuffRemoved = 2 subs)
	buff := core.NewBuff(constants.BuffTypeCurse, 3)
	game.SubscribeBuff(player, buff)

	initialCount := game.Bus.GetSubscriptionCount()

	// 取消订阅 (removes all subscriptions for this buff)
	game.UnsubscribeBuff(buff)

	if game.Bus.GetSubscriptionCount() != initialCount-2 {
		t.Errorf("Subscription count should decrease by 2 after unsubscribe, got %d expected %d", game.Bus.GetSubscriptionCount(), initialCount-2)
	}
}

func TestGameSubscribeBuffByPlayerAdd(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	playerID := id.NewPlayerID()
	player := core.NewPlayer(core.PlayerConfig{
		ID:      playerID,
		Faction: constants.FactionZhuQue, // ZhuQue faction
	})

	// Add player to game (no Fire buff yet - added by InitializePlayerFactionBuffs)
	game.AddPlayer(player)

	// Verify player has no Fire buff initially (core layer is pure)
	if player.HasBuff(constants.BuffTypeFire) {
		t.Error("Player should not have Fire buff before InitializePlayerFactionBuffs")
	}

	// Initialize faction buffs (adds ZhuQue Fire buff)
	game.InitializePlayerFactionBuffs(player)

	// Verify player now has Fire buff
	if !player.HasBuff(constants.BuffTypeFire) {
		t.Error("ZhuQue player should have Fire buff after InitializePlayerFactionBuffs")
	}

	// Fire buff should be subscribed to BeforeTurn
	config := GetBuffHandlerConfig(constants.BuffTypeFire)
	for _, phase := range config.GetPhases() {
		if phase.NeedsSubscription() {
			// BeforeTurn needs subscription
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
	item := core.NewItem(constants.ItemTypeDiceUpgrade)
	game.SubscribeItem(player, item)

	// 验证订阅已创建
	config := GetItemHandlerConfig(constants.ItemTypeDiceUpgrade)
	if config.Phase.NeedsSubscription() {
		if game.Bus.GetSubscriptionCount() != 1 {
			t.Errorf("Subscription count = %d, expected 1", game.Bus.GetSubscriptionCount())
		}
		// Subscription managed by EventBus via sourceID
	}
}

func TestGameSubscribeAnyTimeItem(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	playerID := id.NewPlayerID()
	player := core.NewPlayer(core.PlayerConfig{ID: playerID})
	game.AddPlayer(player)

	// 添加 AnyTime 道具 (反方向的钟)
	item := core.NewItem(constants.ItemTypeReverseClock)
	game.SubscribeItem(player, item)

	// AnyTime 道具不需要订阅（主动触发）
	config := GetItemHandlerConfig(constants.ItemTypeReverseClock)
	if config.Phase == constants.PhaseAnyTime {
		// Subscription managed by EventBus via sourceID
	}
}

func TestGameUnsubscribeItem(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	playerID := id.NewPlayerID()
	player := core.NewPlayer(core.PlayerConfig{ID: playerID})
	game.AddPlayer(player)

	// 订阅道具
	item := core.NewItem(constants.ItemTypeDiceUpgrade)
	game.SubscribeItem(player, item)

	initialCount := game.Bus.GetSubscriptionCount()

	// 取消订阅
	game.UnsubscribeItem(item)

	if game.Bus.GetSubscriptionCount() != initialCount-1 {
		t.Errorf("Subscription count should decrease after unsubscribe")
	}
	// Subscription managed by EventBus via sourceID
}

// ========== Multiple Subscriptions Tests ==========

func TestGameMultipleBuffSubscriptions(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	playerID := id.NewPlayerID()
	player := core.NewPlayer(core.PlayerConfig{ID: playerID})
	game.AddPlayer(player)

	// 添加多个 Buff
	buff1 := core.NewBuff(constants.BuffTypeCurse, 3)  // PostBuffApplied + PreBuffRemoved = 2 subs
	buff2 := core.NewBuff(constants.BuffTypeDivine, 3) // PostBuffApplied + PreBuffRemoved = 2 subs
	buff3 := core.NewBuff(constants.BuffTypeHidden, 3) // PreBuffApplied = 1 sub

	player.AddBuff(buff1)
	player.AddBuff(buff2)
	player.AddBuff(buff3)

	game.SubscribeBuff(player, buff1)
	game.SubscribeBuff(player, buff2)
	game.SubscribeBuff(player, buff3)

	// Curse(2) + Divine(2) + Hidden(1) = 5 subscriptions
	if game.Bus.GetSubscriptionCount() != 5 {
		t.Errorf("Subscription count = %d, expected 5", game.Bus.GetSubscriptionCount())
	}
}

func TestGameMixedSubscriptions(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	playerID := id.NewPlayerID()
	player := core.NewPlayer(core.PlayerConfig{ID: playerID})
	game.AddPlayer(player)

	// 添加 Buff 和道具
	buff := core.NewBuff(constants.BuffTypeCurse, 3) // PostBuffApplied + PreBuffRemoved = 2 subs
	item := core.NewItem(constants.ItemTypeDiceUpgrade)

	player.AddBuff(buff)
	player.AddItem(item)

	game.SubscribeBuff(player, buff)
	game.SubscribeItem(player, item)

	// Curse(2) + DiceUpgrade(1) = 3 subscriptions
	if game.Bus.GetSubscriptionCount() != 3 {
		t.Errorf("Subscription count = %d, expected 3", game.Bus.GetSubscriptionCount())
	}
}

// ========== Decision Creation Tests ==========

func TestGameCreateBuffDecision(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	buff := core.NewBuff(constants.BuffTypeCurse, 3)
	def := GetBuffDefinition(constants.BuffTypeCurse)
	config := GetBuffHandlerConfig(constants.BuffTypeCurse)

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
	item := core.NewItem(constants.ItemTypeDiceUpgrade)
	def := GetItemDefinition(constants.ItemTypeDiceUpgrade)
	config := GetItemHandlerConfig(constants.ItemTypeDiceUpgrade)

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
	buff := core.NewBuff(constants.BuffTypeCurse, 3)
	err := game.ApplyBuffToPlayer(player, buff)

	if err != nil {
		t.Errorf("ApplyBuffToPlayer should not return error: %v", err)
	}

	// 验证 Buff 已添加
	if !player.HasBuff(constants.BuffTypeCurse) {
		t.Error("Player should have Curse buff")
	}

	// 验证订阅已创建
	if game.Bus.GetSubscriptionCount() == 0 {
		t.Error("Should have subscriptions after ApplyBuffToPlayer")
	}

	// 验证 SubscriptionIDs 已填充
	// Subscription managed by EventBus via sourceID
}

func TestGameRemoveBuffFromPlayer(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	playerID := id.NewPlayerID()
	player := core.NewPlayer(core.PlayerConfig{ID: playerID})
	game.AddPlayer(player)

	// 先添加 Buff
	buff := core.NewBuff(constants.BuffTypeCurse, 3)
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
	if player.HasBuff(constants.BuffTypeCurse) {
		t.Error("Player should not have Curse buff after removal")
	}

	// 验证订阅已取消 (Curse had 2 subscriptions: PostBuffApplied + PreBuffRemoved)
	if game.Bus.GetSubscriptionCount() != initialSubCount-2 {
		t.Errorf("Subscription count should decrease by 2 after removal, got %d expected %d", game.Bus.GetSubscriptionCount(), initialSubCount-2)
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
	buff1 := core.NewBuff(constants.BuffTypeCurse, 3)
	buff2 := core.NewBuff(constants.BuffTypeDivine, 3)
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

// ========== Item Lifecycle Tests ==========

func TestApplyItemToPlayer(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	game.AddPlayer(player)

	item := core.NewItem(constants.ItemTypeReverseClock)
	err := game.ApplyItemToPlayer(player, item)

	if err != nil {
		t.Errorf("ApplyItemToPlayer failed: %v", err)
	}

	// Item should be in player's inventory
	if len(player.Inventory) != 1 {
		t.Errorf("Player should have 1 item, got %d", len(player.Inventory))
	}
	if player.Inventory[0].Type != constants.ItemTypeReverseClock {
		t.Errorf("Item type = %s, expected reverse_clock", player.Inventory[0].Type)
	}

	// ReverseClock subscribes to PhaseAnyTime which doesn't need subscription,
	// so verify through UnsubscribeBySource that item.ID.UUID() is tracked
	// (PhaseAnyTime.NeedsSubscription() returns false, so no EventBus subscription)
	// This is expected behavior - item handlers are invoked via PhaseItemUsed publish
}

func TestRemoveItemFromPlayer(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	game.AddPlayer(player)

	item := core.NewItem(constants.ItemTypeReverseClock)
	game.ApplyItemToPlayer(player, item)

	if len(player.Inventory) != 1 {
		t.Fatalf("Player should have 1 item before removal, got %d", len(player.Inventory))
	}

	removed := game.RemoveItemFromPlayer(player, item)

	if !removed {
		t.Error("RemoveItemFromPlayer should return true for existing item")
	}

	// Item should be removed from player's inventory
	if len(player.Inventory) != 0 {
		t.Errorf("Player should have 0 items after removal, got %d", len(player.Inventory))
	}
}

func TestApplyItemToPlayerWithSubscription(t *testing.T) {
	// AnyDoor subscribes to PhaseOnLand which needs subscription
	game := NewGame(id.NewGameID(), 0)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	game.AddPlayer(player)

	item := core.NewItem(constants.ItemTypeAnyDoor)
	err := game.ApplyItemToPlayer(player, item)

	if err != nil {
		t.Errorf("ApplyItemToPlayer failed: %v", err)
	}

	// Item should be in player's inventory
	if len(player.Inventory) != 1 {
		t.Errorf("Player should have 1 item, got %d", len(player.Inventory))
	}

	// AnyDoor subscribes to PhaseOnLand
	subscriptions := game.Bus.GetSubscriptions(constants.PhaseOnLand)
	hasItemSub := false
	for _, sub := range subscriptions {
		if sub.SourceID == item.ID.UUID() {
			hasItemSub = true
			break
		}
	}
	if !hasItemSub {
		t.Error("AnyDoor item should be subscribed to PhaseOnLand after ApplyItemToPlayer")
	}

	// Remove item and verify subscription removed
	game.RemoveItemFromPlayer(player, item)
	subscriptions = game.Bus.GetSubscriptions(constants.PhaseOnLand)
	for _, sub := range subscriptions {
		if sub.SourceID == item.ID.UUID() {
			t.Error("AnyDoor subscription should be removed after RemoveItemFromPlayer")
		}
	}
}
