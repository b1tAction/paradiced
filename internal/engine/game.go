// Package engine provides game engine logic for the Fated game.
package engine

import (
	"math/rand"
	"sync"
	"time"

	"github.com/b1tAction/fated/internal/core"
	"github.com/b1tAction/fated/pkg/constants"
	"github.com/b1tAction/fated/pkg/event"
	"github.com/b1tAction/fated/pkg/gamelog"
	"github.com/b1tAction/fated/pkg/id"
	"github.com/b1tAction/fated/pkg/rng"
)

// Game is the game instance, containing EventBus and all players.
type Game struct {
	ID      id.GameID         `json:"id"`
	Bus     *event.EventBus   `json:"bus"`
	Players []*core.Player    `json:"players"`
	State   *GameState        `json:"state"`
	RNG     *rand.Rand        `json:"-"`   // Game unique random source
	Draw    *rng.DrawEngine   `json:"-"`   // Draw engine for random draws
	Log     *gamelog.GameLog  `json:"log"` // Global game log for playback
	mutex   sync.RWMutex
}

// GameState represents game state.
type GameState struct {
	Round        int    `json:"round"`         // Current round
	Turn         int    `json:"turn"`          // Current turn (player index)
	CurrentPhase string `json:"current_phase"` // Current phase
	Waiting      bool   `json:"waiting"`       // Waiting for user decision
}

// NewGame creates a new game instance.
// seed: random seed (0 for auto-generated from current time)
func NewGame(gameID id.GameID, seed int64) *Game {
	rngSource := seed
	if rngSource == 0 {
		rngSource = time.Now().UnixNano()
	}
	rngInst := rand.New(rand.NewSource(rngSource))

	return &Game{
		ID:      gameID,
		Bus:     event.NewEventBus(gameID.UUID()),
		Players: make([]*core.Player, 0),
		State:   &GameState{Round: 1, Turn: 0, CurrentPhase: "init", Waiting: false},
		RNG:     rngInst,
		Draw:    rng.NewDrawEngine(rngInst),
		Log:     gamelog.NewGameLog(),
	}
}

// AddPlayer adds a player to the game.
func (g *Game) AddPlayer(player *core.Player) {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	g.Players = append(g.Players, player)
	g.subscribePlayerEffects(player)
}

// RemovePlayer removes a player.
func (g *Game) RemovePlayer(playerID id.PlayerID) {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	for i, p := range g.Players {
		if p.ID.Equal(playerID.ID) {
			g.Bus.UnsubscribeByOwner(playerID.UUID())
			g.Players = append(g.Players[:i], g.Players[i+1:]...)
			return
		}
	}
}

// GetPlayer returns the specified player.
func (g *Game) GetPlayer(playerID id.PlayerID) *core.Player {
	g.mutex.RLock()
	defer g.mutex.RUnlock()

	for _, p := range g.Players {
		if p.ID.Equal(playerID.ID) {
			return p
		}
	}
	return nil
}

// GetCurrentPlayer returns the current turn player.
func (g *Game) GetCurrentPlayer() *core.Player {
	g.mutex.RLock()
	defer g.mutex.RUnlock()

	if g.State.Turn < len(g.Players) {
		return g.Players[g.State.Turn]
	}
	return nil
}

// GetPlayers returns all players in the game.
func (g *Game) GetPlayers() []*core.Player {
	g.mutex.RLock()
	defer g.mutex.RUnlock()
	return g.Players
}

// GetGameLog returns the global game log for playback.
// Implements protocol.Game interface.
func (g *Game) GetGameLog() *gamelog.GameLog {
	return g.Log
}

// NextTurn advances to next player's turn.
func (g *Game) NextTurn() {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	g.State.Turn++
	if g.State.Turn >= len(g.Players) {
		g.State.Turn = 0
		g.State.Round++
	}
}

// subscribePlayerEffects subscribes player's Buff and Item effects.
func (g *Game) subscribePlayerEffects(player *core.Player) {
	for _, buff := range player.ActiveBuffs {
		g.SubscribeBuff(player, buff)
	}

	for _, item := range player.Inventory {
		g.SubscribeItem(player, item)
	}
}

// createBuffDecision creates a Decision for Buff.
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

// createItemDecision creates a Decision for Item.
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

// SubscribeBuff subscribes when player gets a new Buff.
// Supports multi-phase subscription: iterates def.Phases to create subscription for each Phase.
// Note: If Buff already has subscriptions (SubscriptionIDs not empty), won't subscribe again.
func (g *Game) SubscribeBuff(player *core.Player, buff *core.Buff) {
	// Prevent duplicate subscriptions
	if len(buff.SubscriptionIDs) > 0 {
		return
	}

	def := core.GetBuffDefinition(buff.Type)
	if def == nil {
		return
	}

	// Initialize SubscriptionIDs list
	buff.SubscriptionIDs = make([]string, 0)

	// Create subscription for each Phase
	for _, phase := range def.GetPhases() {
		if !phase.NeedsSubscription() {
			continue
		}

		// Create closure Action (using handlers.go createBuffAction)
		buffAction := createBuffAction(buff, def, phase, player)

		// Wrap to match event.Option.Action signature (discards return value)
		action := func(ctx *event.Context) {
			buffAction(ctx) // Actions are pushed to queue via ActionContext
		}

		// Create Decision
		decision := g.createBuffDecisionWithAction(buff, def, action)

		// Register with EventBus
		subID := g.Bus.Subscribe(phase, player.ID, buff.ID.UUID(), "buff", decision)
		buff.SubscriptionIDs = append(buff.SubscriptionIDs, subID.UUID())
	}
}

// createBuffDecisionWithAction creates a Buff Decision with Action.
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

// UnsubscribeBuff unsubscribes when Buff is removed.
// Supports multi-subscription unsubscribe: iterates buff.SubscriptionIDs to unsubscribe all.
func (g *Game) UnsubscribeBuff(buff *core.Buff) {
	for _, subIDStr := range buff.SubscriptionIDs {
		if subIDStr != "" {
			subID := id.MustParseSubscriptionID(subIDStr)
			g.Bus.Unsubscribe(subID)
		}
	}
	buff.SubscriptionIDs = make([]string, 0)
}

// SubscribeItem subscribes when player gets a new Item.
func (g *Game) SubscribeItem(player *core.Player, item *core.Item) {
	def := core.GetItemDefinition(item.Type)
	if def == nil {
		return
	}
	if def.Phase.NeedsSubscription() {
		decision := g.createItemDecision(item, def)
		subID := g.Bus.Subscribe(def.Phase, player.ID, item.ID.UUID(), "item", decision)
		item.SubscriptionID = subID.UUID()
	}
}

// UnsubscribeItem unsubscribes when Item is removed.
func (g *Game) UnsubscribeItem(item *core.Item) {
	if item.SubscriptionID != "" {
		subID := id.MustParseSubscriptionID(item.SubscriptionID)
		g.Bus.Unsubscribe(subID)
		item.SubscriptionID = ""
	}
}

// ========== Buff Lifecycle Management ==========

// ApplyBuffToPlayer adds Buff to player and handles complete lifecycle.
// Process order:
//  1. Underlying data add (player.AddBuff)
//  2. Subscribe to EventBus (SubscribeBuff) - Buff is now in listener list
//  3. Broadcast Applied event - Buff can hear its own Applied event
func (g *Game) ApplyBuffToPlayer(player *core.Player, buff *core.Buff) error {
	// 1. Underlying data add
	if err := player.AddBuff(buff); err != nil {
		return err
	}

	// 2. Subscribe to EventBus
	g.SubscribeBuff(player, buff)

	// 3. Broadcast Applied event
	g.BroadcastBuffApplied(player, buff)

	return nil
}

// RemoveBuffFromPlayer removes Buff from player and handles complete lifecycle.
// Process order:
//  1. Broadcast Removed event (before removal) - Buff can hear its own Removed event
//  2. Unsubscribe (UnsubscribeBuff)
//  3. Underlying data remove (player.RemoveBuff)
func (g *Game) RemoveBuffFromPlayer(player *core.Player, buff *core.Buff) bool {
	// 1. Broadcast Removed event first
	g.BroadcastBuffRemoved(player, buff)

	// 2. Unsubscribe
	g.UnsubscribeBuff(buff)

	// 3. Underlying data remove
	return player.RemoveBuff(buff.Type)
}

// BroadcastBuffApplied broadcasts Buff Applied event.
// Triggers PhaseOnBuffApplied, all Buffs/Items subscribed to this Phase receive notification.
func (g *Game) BroadcastBuffApplied(player *core.Player, buff *core.Buff) {
	ctx := event.NewContext(player).WithData(buff)
	g.Bus.Publish(constants.PhaseOnBuffApplied, player.ID.UUID(), ctx)
}

// BroadcastBuffRemoved broadcasts Buff Removed event.
// Triggers PhaseOnBuffRemoved, all Buffs/Items subscribed to this Phase receive notification.
func (g *Game) BroadcastBuffRemoved(player *core.Player, buff *core.Buff) {
	ctx := event.NewContext(player).WithData(buff)
	g.Bus.Publish(constants.PhaseOnBuffRemoved, player.ID.UUID(), ctx)
}

// ========== Helper Methods ==========

// GetActiveBuffCount returns the player's current active Buff count.
func (g *Game) GetActiveBuffCount(playerID id.PlayerID) int {
	player := g.GetPlayer(playerID)
	if player == nil {
		return 0
	}
	return len(player.ActiveBuffs)
}

// GetBuffSubscriptionCount returns Buff's subscription count.
func (g *Game) GetBuffSubscriptionCount(buff *core.Buff) int {
	return len(buff.SubscriptionIDs)
}
