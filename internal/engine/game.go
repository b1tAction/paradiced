// Package engine provides game engine logic for the Fated game.
package engine

import (
	"sync"

	"github.com/b1tAction/Fated/internal/core"
	"github.com/b1tAction/Fated/pkg/event"
)

// Game 游戏实例，包含EventBus和所有玩家
type Game struct {
	ID        string          `json:"id"`
	Bus       *event.EventBus `json:"bus"`
	Players   []*core.Player  `json:"players"`
	State     *GameState      `json:"state"`
	mutex     sync.RWMutex
}

// GameState 游戏状态
type GameState struct {
	Round        int    `json:"round"`         // 当前轮次
	Turn         int    `json:"turn"`          // 当前回合（玩家索引）
	CurrentPhase string `json:"current_phase"` // 当前阶段
	Waiting      bool   `json:"waiting"`       // 是否等待用户决策
}

// NewGame 创建新的游戏实例
func NewGame(gameID string) *Game {
	return &Game{
		ID:      gameID,
		Bus:     event.NewEventBus(gameID),
		Players: make([]*core.Player, 0),
		State:   &GameState{Round: 1, Turn: 0, CurrentPhase: "init", Waiting: false},
	}
}

// AddPlayer 添加玩家到游戏
func (g *Game) AddPlayer(player *core.Player) {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	g.Players = append(g.Players, player)
	g.subscribePlayerEffects(player)
}

// RemovePlayer 移除玩家
func (g *Game) RemovePlayer(userID string) {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	for i, p := range g.Players {
		if p.UserID == userID {
			g.Bus.UnsubscribeByOwner(userID)
			g.Players = append(g.Players[:i], g.Players[i+1:]...)
			return
		}
	}
}

// GetPlayer 获取指定玩家
func (g *Game) GetPlayer(userID string) *core.Player {
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
func (g *Game) GetCurrentPlayer() *core.Player {
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
func (g *Game) subscribePlayerEffects(player *core.Player) {
	for _, buff := range player.ActiveBuffs {
		g.SubscribeBuff(player, buff)
	}

	for _, item := range player.Inventory {
		g.SubscribeItem(player, item)
	}
}

// createBuffDecision 为Buff创建Decision
func (g *Game) createBuffDecision(buff *core.Buff, def *core.BuffDefinition) *event.Decision {
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
func (g *Game) createItemDecision(item *core.Item, def *core.ItemDefinition) *event.Decision {
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
// 支持多Phase订阅：遍历 def.Phases 为每个Phase创建订阅
// 注意：如果 Buff 已经有订阅（SubscriptionIDs 不为空），不会重复订阅
func (g *Game) SubscribeBuff(player *core.Player, buff *core.Buff) {
	// 防止重复订阅
	if len(buff.SubscriptionIDs) > 0 {
		return
	}

	def := buff.Type.GetBuffDefinition()
	if def == nil {
		return
	}

	// 初始化 SubscriptionIDs 列表
	buff.SubscriptionIDs = make([]string, 0)

	// 获取定制处理器
	handler := GetHandler(buff.Type)
	hasCustomHandler := handler != nil

	// 为每个 Phase 创建订阅
	for _, phase := range def.GetPhases() {
		if !phase.NeedsSubscription() {
			continue
		}

		// 创建闭包 Action
		action := g.createBuffAction(buff, def, phase, player, hasCustomHandler, handler)

		// 创建 Decision
		decision := g.createBuffDecisionWithAction(buff, def, action)

		// 向 EventBus 注册
		subID := g.Bus.Subscribe(phase, player.UserID, buff.ID, "buff", decision)
		buff.SubscriptionIDs = append(buff.SubscriptionIDs, subID)
	}
}

// createBuffAction 创建 Buff 触发时的 Action 闭包
func (g *Game) createBuffAction(buff *core.Buff, def *core.BuffDefinition, phase event.Phase, player *core.Player, hasCustomHandler bool, handler EventHandler) func(ctx *event.Context) {
	return func(ctx *event.Context) {
		// 从 Context 获取 Player（确保类型正确）
		p, ok := ctx.Player.(*core.Player)
		if !ok {
			return
		}

		if hasCustomHandler {
			// 调用定制处理器
			handler(phase, ctx)
		} else {
			// 执行默认数值效果
			executeDefaultBuffAction(def, p)
		}
	}
}

// createBuffDecisionWithAction 创建带有 Action 的 Buff Decision
func (g *Game) createBuffDecisionWithAction(buff *core.Buff, def *core.BuffDefinition, action func(ctx *event.Context)) *event.Decision {
	if def.NeedConfirm {
		return event.NewDecision(def.Desc, []event.Option{
			{ID: "apply", Label: "应用效果", Action: action},
		})
	}
	return event.NewAutoDecision(def.Desc, []event.Option{
		{ID: "apply", Label: "自动执行", Action: action},
	})
}

// UnsubscribeBuff 当Buff移除时取消订阅
// 支持多订阅取消：遍历 buff.SubscriptionIDs 取消所有订阅
func (g *Game) UnsubscribeBuff(buff *core.Buff) {
	for _, subID := range buff.SubscriptionIDs {
		if subID != "" {
			g.Bus.Unsubscribe(subID)
		}
	}
	buff.SubscriptionIDs = make([]string, 0)
}

// SubscribeItem 当玩家获得新道具时订阅
func (g *Game) SubscribeItem(player *core.Player, item *core.Item) {
	def := item.Type.GetItemDefinition()
	if def.Phase.NeedsSubscription() {
		decision := g.createItemDecision(item, def)
		subID := g.Bus.Subscribe(def.Phase, player.UserID, item.ID, "item", decision)
		item.SubscriptionID = subID
	}
}

// UnsubscribeItem 当道具移除时取消订阅
func (g *Game) UnsubscribeItem(item *core.Item) {
	if item.SubscriptionID != "" {
		g.Bus.Unsubscribe(item.SubscriptionID)
		item.SubscriptionID = ""
	}
}

// ========== Buff 生命周期管理 ==========

// ApplyBuffToPlayer 为玩家添加 Buff 并处理完整的生命周期
// 流程顺序：
//  1. 底层数据添加 (player.AddBuff)
//  2. 挂载到 EventBus (SubscribeBuff) - 此时 Buff 已在监听列表
//  3. 广播 Applied 事件 - Buff 可以听到自己的 Applied 事件
func (g *Game) ApplyBuffToPlayer(player *core.Player, buff *core.Buff) error {
	// 1. 底层数据添加
	if err := player.AddBuff(buff); err != nil {
		return err
	}

	// 2. 挂载到 EventBus
	g.SubscribeBuff(player, buff)

	// 3. 广播 Applied 事件
	g.BroadcastBuffApplied(player, buff)

	return nil
}

// RemoveBuffFromPlayer 从玩家移除 Buff 并处理完整的生命周期
// 流程顺序：
//  1. 广播 Removed 事件（在移除前） - Buff 可以听到自己的 Removed 事件
//  2. 取消订阅 (UnsubscribeBuff)
//  3. 底层数据移除 (player.RemoveBuff)
func (g *Game) RemoveBuffFromPlayer(player *core.Player, buff *core.Buff) bool {
	// 1. 先广播 Removed 事件
	g.BroadcastBuffRemoved(player, buff)

	// 2. 取消订阅
	g.UnsubscribeBuff(buff)

	// 3. 底层数据移除
	return player.RemoveBuff(buff.Type)
}

// BroadcastBuffApplied 广播 Buff Applied 事件
// 触发 PhaseOnBuffApplied，所有订阅该 Phase 的 Buff/道具都会收到通知
func (g *Game) BroadcastBuffApplied(player *core.Player, buff *core.Buff) {
	ctx := event.NewContext(player).WithData(buff)
	g.Bus.Publish(event.PhaseOnBuffApplied, player.UserID, ctx)
}

// BroadcastBuffRemoved 广播 Buff Removed 事件
// 触发 PhaseOnBuffRemoved，所有订阅该 Phase 的 Buff/道具都会收到通知
func (g *Game) BroadcastBuffRemoved(player *core.Player, buff *core.Buff) {
	ctx := event.NewContext(player).WithData(buff)
	g.Bus.Publish(event.PhaseOnBuffRemoved, player.UserID, ctx)
}

// ========== 辅助方法 ==========

// GetActiveBuffCount 获取玩家当前活跃的 Buff 数量
func (g *Game) GetActiveBuffCount(playerID string) int {
	player := g.GetPlayer(playerID)
	if player == nil {
		return 0
	}
	return len(player.ActiveBuffs)
}

// GetBuffSubscriptionCount 获取 Buff 的订阅数量
func (g *Game) GetBuffSubscriptionCount(buff *core.Buff) int {
	return len(buff.SubscriptionIDs)
}