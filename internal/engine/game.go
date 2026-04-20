// Package engine provides game engine logic for the Paradiced game.
package engine

import (
	"math/rand"
	"sync"
	"time"

	"github.com/b1tAction/paradiced/internal/core"
	"github.com/b1tAction/paradiced/internal/event"
	"github.com/b1tAction/paradiced/pkg/constants"
	"github.com/b1tAction/paradiced/pkg/gamelog"
	"github.com/b1tAction/paradiced/pkg/id"
	"github.com/b1tAction/paradiced/pkg/rng"
)

// Game is the game instance, containing EventBus and all players.
// Round/Turn state is managed by HSM (single source of truth).
// Game only stores data, not state.
type Game struct {
	ID      id.GameID        `json:"id"`
	Bus     *event.EventBus  `json:"bus"`
	Players []*core.Player   `json:"players"`
	RNG     *rand.Rand       `json:"-"`   // Game unique random source
	Draw    *rng.DrawEngine  `json:"-"`   // Draw engine for random draws
	Log     *gamelog.GameLog `json:"log"` // Global game log for playback
	mutex   sync.RWMutex
}

// NewGame creates a new game instance.
// seed: random seed (0 for auto-generated from current time)
// Note: Round/Turn state is managed by HSM, not stored in Game.
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

// GetPlayerInterface returns player by ID as interface{} (for protocol.Game).
func (g *Game) GetPlayerInterface(playerID id.PlayerID) interface{} {
	return g.GetPlayer(playerID)
}

// GetPlayersInterface returns all players as []interface{} (for protocol.Game).
func (g *Game) GetPlayersInterface() []interface{} {
	players := g.GetPlayers()
	result := make([]interface{}, len(players))
	for i, p := range players {
		result[i] = p
	}
	return result
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

// subscribePlayerEffects subscribes player's Buff and Item effects.
func (g *Game) subscribePlayerEffects(player *core.Player) {
	for _, buff := range player.ActiveBuffs {
		g.SubscribeBuff(player, buff)
	}

	for _, item := range player.Inventory {
		g.SubscribeItem(player, item)
	}
}

// createBuffDecision creates a Decision for Buff using handler config.
func (g *Game) createBuffDecision(buff *core.Buff, def *core.BuffDefinition, config *BuffHandlerConfig) *event.Decision {
	desc := def.Desc
	if config.NeedConfirm {
		return event.NewDecision(desc, []event.Option{
			{ID: "apply", Label: "应用效果"},
		}).WithPriority(config.Priority)
	}
	return event.NewAutoDecision(desc, []event.Option{
		{ID: "apply", Label: "自动执行"},
	}).WithPriority(config.Priority)
}

// createItemDecision creates a Decision for Item using handler config.
func (g *Game) createItemDecision(item *core.Item, def *core.ItemDefinition, config *ItemHandlerConfig) *event.Decision {
	desc := def.Desc
	if config.NeedConfirm {
		return event.NewDecision(desc, []event.Option{
			{ID: "use", Label: "使用"},
			{ID: "skip", Label: "跳过"},
		}).WithPriority(config.Priority)
	}
	return event.NewAutoDecision(desc, []event.Option{
		{ID: "use", Label: "自动执行"},
	}).WithPriority(config.Priority)
}

// SubscribeBuff subscribes when player gets a new Buff.
// Uses BuffHandlerConfig for Phases/Priority/NeedConfirm.
func (g *Game) SubscribeBuff(player *core.Player, buff *core.Buff) {
	def := GetBuffDefinition(buff.Type)
	config := GetBuffHandlerConfig(buff.Type)
	if def == nil || config == nil {
		return
	}

	// Create subscription for each Phase
	for _, phase := range config.GetPhases() {
		if !phase.NeedsSubscription() {
			continue
		}

		// Create handler closure - executes config.Handler
		action := func(ctx *event.Context) error {
			if config.Handler != nil {
				return config.Handler(phase, ctx)
			}
			return nil
		}

		// Create Decision with Priority from config
		decision := g.createBuffDecisionWithAction(buff, def, config, action)

		// Register with EventBus using buff.ID.UUID() as sourceID
		g.Bus.Subscribe(phase, player.ID, buff.ID.UUID(), "buff", decision)
	}
}

// createBuffDecisionWithAction creates a Buff Decision with Action using handler config.
func (g *Game) createBuffDecisionWithAction(buff *core.Buff, def *core.BuffDefinition, config *BuffHandlerConfig, action func(ctx *event.Context) error) *event.Decision {
	desc := def.Desc
	if config.NeedConfirm {
		return event.NewDecision(desc, []event.Option{
			{ID: "apply", Label: "应用效果", Action: action},
		}).WithPriority(config.Priority)
	}
	return event.NewAutoDecision(desc, []event.Option{
		{ID: "apply", Label: "自动执行", Action: action},
	}).WithPriority(config.Priority)
}

// UnsubscribeBuff unsubscribes when Buff is removed.
// Uses UnsubscribeBySource with buff.ID.UUID() to remove all subscriptions.
func (g *Game) UnsubscribeBuff(buff *core.Buff) {
	g.Bus.UnsubscribeBySource(buff.ID.UUID())
}

// SubscribeItem subscribes when player gets a new Item.
// Uses ItemHandlerConfig for Phase/Priority/NeedConfirm.
func (g *Game) SubscribeItem(player *core.Player, item *core.Item) {
	def := GetItemDefinition(item.Type)
	config := GetItemHandlerConfig(item.Type)
	if def == nil || config == nil {
		return
	}
	if config.Phase.NeedsSubscription() {
		// Create handler closure - executes config.Handler
		action := func(ctx *event.Context) error {
			if config.Handler != nil {
				return config.Handler(constants.PhaseItemUsed, ctx)
			}
			return nil
		}
		decision := g.createItemDecisionWithAction(item, def, config, action)
		// Register with EventBus using item.ID.UUID() as sourceID
		g.Bus.Subscribe(config.Phase, player.ID, item.ID.UUID(), "item", decision)
	}
}

// createItemDecisionWithAction creates an Item Decision with Action using handler config.
func (g *Game) createItemDecisionWithAction(item *core.Item, def *core.ItemDefinition, config *ItemHandlerConfig, action func(ctx *event.Context) error) *event.Decision {
	desc := def.Desc
	if config.NeedConfirm {
		return event.NewDecision(desc, []event.Option{
			{ID: "use", Label: "使用", Action: action},
			{ID: "skip", Label: "跳过"},
		}).WithPriority(config.Priority)
	}
	return event.NewAutoDecision(desc, []event.Option{
		{ID: "use", Label: "自动执行", Action: action},
	}).WithPriority(config.Priority)
}

// UnsubscribeItem unsubscribes when Item is removed.
// Uses UnsubscribeBySource with item.ID.UUID() to remove subscription.
func (g *Game) UnsubscribeItem(item *core.Item) {
	g.Bus.UnsubscribeBySource(item.ID.UUID())
}

// ========== Faction Initialization ==========

// InitializePlayerFactionBuffs adds faction-specific buffs to players.
// Called during match initialization after EventBus is ready.
// Uses ApplyBuffToPlayer for complete lifecycle (AddBuff + Subscribe).
func (g *Game) InitializePlayerFactionBuffs(player *core.Player) {
	if player == nil {
		return
	}

	faction := player.GetFaction()
	switch faction {
	case constants.FactionZhuQue:
		// ZhuQue朱雀: Fire离火 buff (permanent, LP+1 every 4 turns)
		fireBuff := core.NewBuff(constants.BuffTypeFire, -1)
		g.ApplyBuffToPlayer(player, fireBuff)
	}
}

// ========== Buff Lifecycle Management ==========

// ApplyBuffToPlayer adds Buff to player and handles complete lifecycle.
// Process order:
//  1. Underlying data add (player.AddBuff)
//  2. Subscribe to EventBus (SubscribeBuff)
//
// Note: Buff lifecycle Phases (PhasePostBuffApplied, PhasePreBuffRemoved) are
// published by Action system (AddBuffAction/RemoveBuffAction) during execution.
func (g *Game) ApplyBuffToPlayer(player *core.Player, buff *core.Buff) error {
	// 1. Underlying data add
	if err := player.AddBuff(buff); err != nil {
		return err
	}

	// 2. Subscribe to EventBus
	g.SubscribeBuff(player, buff)

	return nil
}

// RemoveBuffFromPlayer removes Buff from player and handles complete lifecycle.
// Process order:
//  1. Unsubscribe (UnsubscribeBuff)
//  2. Underlying data remove (player.RemoveBuff)
//
// Note: Buff lifecycle Phases (PhasePostBuffApplied, PhasePreBuffRemoved) are
// published by Action system (AddBuffAction/RemoveBuffAction) during execution.
func (g *Game) RemoveBuffFromPlayer(player *core.Player, buff *core.Buff) bool {
	// 1. Unsubscribe
	g.UnsubscribeBuff(buff)

	// 2. Underlying data remove
	return player.RemoveBuff(buff.Type)
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
