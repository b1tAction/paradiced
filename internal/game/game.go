package game

import (
	"sync"

	"github.com/b1tAction/Fated/pkg/event"
)

// Game 游戏实例，包含EventBus和所有玩家
type Game struct {
	ID        string          `json:"id"`
	Bus       *event.EventBus `json:"bus"`
	Players   []*Player       `json:"players"`
	State     *GameState      `json:"state"`
	mutex     sync.RWMutex
}

// GameState 游戏状态
type GameState struct {
	Round        int    `json:"round"`        // 当前轮次
	Turn         int    `json:"turn"`         // 当前回合（玩家索引）
	CurrentPhase string `json:"current_phase"` // 当前阶段
	Waiting      bool   `json:"waiting"`      // 是否等待用户决策
}

// NewGame 创建新的游戏实例
func NewGame(gameID string) *Game {
	return &Game{
		ID:      gameID,
		Bus:     event.NewEventBus(gameID),
		Players: make([]*Player, 0),
		State:   &GameState{Round: 1, Turn: 0, CurrentPhase: "init", Waiting: false},
	}
}

// AddPlayer 添加玩家到游戏
func (g *Game) AddPlayer(player *Player) {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	g.Players = append(g.Players, player)

	// 玩家添加时，自动订阅其Buff和道具到EventBus
	g.subscribePlayerEffects(player)
}

// RemovePlayer 移除玩家
func (g *Game) RemovePlayer(userID string) {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	for i, p := range g.Players {
		if p.UserID == userID {
			// 取消所有订阅
			g.Bus.UnsubscribeByOwner(userID)
			g.Players = append(g.Players[:i], g.Players[i+1:]...)
			return
		}
	}
}

// GetPlayer 获取指定玩家
func (g *Game) GetPlayer(userID string) *Player {
	g.mutex.RLock()
	defer g.mutex.RUnlock()

	for _, p := range g.Players {
		if p.UserID == userID {
			return p
		}
	}
	return nil
}

// GetCurrentPlayer 获取当前回合的玩家
func (g *Game) GetCurrentPlayer() *Player {
	g.mutex.RLock()
	defer g.mutex.RUnlock()

	if g.State.Turn < len(g.Players) {
		return g.Players[g.State.Turn]
	}
	return nil
}

// NextTurn 下一个玩家回合
func (g *Game) NextTurn() {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	g.State.Turn++
	if g.State.Turn >= len(g.Players) {
		g.State.Turn = 0
		g.State.Round++
	}
}

// subscribePlayerEffects 订阅玩家的Buff和道具效果
func (g *Game) subscribePlayerEffects(player *Player) {
	// 订阅Buff
	for _, buff := range player.ActiveBuffs {
		def := buff.Type.GetBuffDefinition()
		if def.Phase.NeedsSubscription() {
			decision := g.createBuffDecision(buff, def)
			subID := g.Bus.Subscribe(def.Phase, player.UserID, buff.ID, "buff", decision)
			buff.SubscriptionID = subID
		}
	}

	// 订阅道具（PhaseAnyTime的道具不订阅）
	for _, item := range player.Inventory {
		def := item.Type.GetItemDefinition()
		if def.Phase.NeedsSubscription() {
			decision := g.createItemDecision(item, def)
			subID := g.Bus.Subscribe(def.Phase, player.UserID, item.ID, "item", decision)
			item.SubscriptionID = subID
		}
	}
}

// createBuffDecision 为Buff创建Decision
func (g *Game) createBuffDecision(buff *Buff, def *BuffDefinition) *event.Decision {
	if def.NeedConfirm {
		return event.NewDecision(def.Desc, []event.Option{
			{ID: "apply", Label: "应用效果"},
		})
	}
	return event.NewAutoDecision(def.Desc, []event.Option{
		{ID: "apply", Label: "自动执行"},
	})
}

// createItemDecision 为道具创建Decision
func (g *Game) createItemDecision(item *Item, def *ItemDefinition) *event.Decision {
	if def.NeedConfirm {
		return event.NewDecision(def.Desc, []event.Option{
			{ID: "use", Label: "使用"},
			{ID: "skip", Label: "跳过"},
		})
	}
	return event.NewAutoDecision(def.Desc, []event.Option{
		{ID: "use", Label: "自动执行"},
	})
}

// SubscribeBuff 当玩家获得新Buff时订阅
func (g *Game) SubscribeBuff(player *Player, buff *Buff) {
	def := buff.Type.GetBuffDefinition()
	if def.Phase.NeedsSubscription() {
		decision := g.createBuffDecision(buff, def)
		subID := g.Bus.Subscribe(def.Phase, player.UserID, buff.ID, "buff", decision)
		buff.SubscriptionID = subID
	}
}

// UnsubscribeBuff 当Buff移除时取消订阅
func (g *Game) UnsubscribeBuff(buff *Buff) {
	if buff.SubscriptionID != "" {
		g.Bus.Unsubscribe(buff.SubscriptionID)
		buff.SubscriptionID = ""
	}
}

// SubscribeItem 当玩家获得新道具时订阅
func (g *Game) SubscribeItem(player *Player, item *Item) {
	def := item.Type.GetItemDefinition()
	if def.Phase.NeedsSubscription() {
		decision := g.createItemDecision(item, def)
		subID := g.Bus.Subscribe(def.Phase, player.UserID, item.ID, "item", decision)
		item.SubscriptionID = subID
	}
}

// UnsubscribeItem 当道具移除时取消订阅
func (g *Game) UnsubscribeItem(item *Item) {
	if item.SubscriptionID != "" {
		g.Bus.Unsubscribe(item.SubscriptionID)
		item.SubscriptionID = ""
	}
}