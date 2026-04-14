// Package net provides synchronization data builder
// for converting internal game structures to protocol messages.
package net

import (
	pkgnet "github.com/b1tAction/fated/pkg/net"
	"github.com/b1tAction/fated/internal/core"
	"github.com/b1tAction/fated/internal/core/buff"
	"github.com/b1tAction/fated/internal/core/item"
	"github.com/b1tAction/fated/internal/engine"
	"github.com/b1tAction/fated/internal/engine/hsm"
	"github.com/b1tAction/fated/pkg/gamelog"
	"github.com/b1tAction/fated/pkg/protocol"
	"github.com/b1tAction/fated/pkg/rng"
)

// Builder converts internal game structures to protocol sync data.
// Used by MatchHandler implementations to build messages for clients.
type Builder struct {
	hsm       *hsm.HSM
	game      *engine.Game
	turnDiceType rng.DiceType // Current player's dice type (from StateContext)
}

// NewBuilder creates a new sync data builder.
func NewBuilder(hsmInstance *hsm.HSM, gameInstance *engine.Game) *Builder {
	return &Builder{
		hsm:  hsmInstance,
		game: gameInstance,
	}
}

// SetDiceType sets the current player's dice type for BuildAvailable.
func (b *Builder) SetDiceType(diceType rng.DiceType) {
	b.turnDiceType = diceType
}

// BuildStateSync builds a complete state sync message.
func (b *Builder) BuildStateSync() *pkgnet.StateSync {
	globalState := b.hsm.GetGlobalStateID()
	turnState := b.hsm.GetTurnStateID()
	turnPlayer := b.hsm.GetTurnPlayer()

	var turnPlayerID string
	if turnPlayer != nil {
		turnPlayerID = turnPlayer.ID.UUID()
	}

	return &pkgnet.StateSync{
		GlobalState: globalState.String(),
		TurnState:   turnState.String(),
		TurnPlayer:  turnPlayerID,
		Round:       b.game.State.Round,
		Turn:        b.game.State.Turn,
		Paused:      b.hsm.IsPaused(),
		Players:     b.BuildPlayers(),
	}
}

// BuildPlayers builds all player state snapshots.
// Extracts known keys from core.Player.Metadata into typed fields.
func (b *Builder) BuildPlayers() []pkgnet.Player {
	players := b.game.GetPlayers()
	result := make([]pkgnet.Player, len(players))
	for i, p := range players {
		result[i] = b.BuildPlayer(p)
	}
	return result
}

// BuildPlayer builds a single player state snapshot.
func (b *Builder) BuildPlayer(p *core.Player) pkgnet.Player {
	return pkgnet.Player{
		UserID:      p.ID.UUID(),
		Faction:     p.Faction.String(),
		Position:    p.Position,
		HP:          p.HP,
		LP:          p.LP,
		Buffs:       b.BuildBuffs(p.ActiveBuffs),
		Items:       b.BuildItems(p.Inventory),
		// Extract known Metadata keys into typed fields
		Charge:      p.GetChargeCount(),
		FireCounter: p.GetFireCounter(),
		IsDead:      p.IsDead,
		SkipTurn:    p.SkipTurn,
	}
}

// BuildBuffs builds buff sync data from active buffs.
func (b *Builder) BuildBuffs(activeBuffs []*buff.Buff) []pkgnet.Buff {
	result := make([]pkgnet.Buff, len(activeBuffs))
	for i, bf := range activeBuffs {
		result[i] = pkgnet.Buff{
			Type:     string(bf.Type), // BuffType is already a string
			Duration: bf.Duration,
		}
	}
	return result
}

// BuildItems builds item sync data from inventory.
func (b *Builder) BuildItems(inventory []*item.Item) []pkgnet.Item {
	result := make([]pkgnet.Item, len(inventory))
	for i, it := range inventory {
		result[i] = pkgnet.Item{
			ID:   it.ID.UUID(),
			Type: string(it.Type), // ItemType is already a string
		}
	}
	return result
}

// BuildActionSync builds action sync from a log entry.
func (b *Builder) BuildActionSync(entry gamelog.LogEntry) *pkgnet.ActionSync {
	metadata := make(map[string]interface{})
	if entry.Metadata != nil {
		metadata = entry.Metadata.ToMap()
	}

	return &pkgnet.ActionSync{
		ActionType: entry.ActionType,
		Target:     entry.Target,
		Delta:      entry.Delta,
		Source:     entry.Source,
		Metadata:   metadata,
	}
}

// BuildAvailable builds available actions for the current player.
// Includes PhaseAnyTime items and faction skill availability.
func (b *Builder) BuildAvailable(player *core.Player) *pkgnet.Available {
	// Filter PhaseAnyTime usable items
	usableItems := make([]pkgnet.Item, 0)
	for _, it := range player.Inventory {
		if it.Usable {
			usableItems = append(usableItems, pkgnet.Item{
				ID:   it.ID.UUID(),
				Type: string(it.Type), // ItemType is already a string
			})
		}
	}

	// Check faction skill availability
	canUseSkill := false
	faction := player.Faction
	switch faction {
	case protocol.FactionQingLong, protocol.FactionXuanWu:
		// 青龙行迹 / 玄武镇厄: requires charge count >= 1
		canUseSkill = player.GetChargeCount() >= 1
	}

	return &pkgnet.Available{
		Items:       usableItems,
		CanUseSkill: canUseSkill,
		DiceType:    b.turnDiceType.String(), // Convert DiceType to string for protocol
	}
}

// BuildDecision builds decision request from event.Decision.
func (b *Builder) BuildDecision(d *pkgnet.Decision) *pkgnet.Decision {
	if d == nil {
		return nil
	}

	options := make([]pkgnet.Option, len(d.Options))
	for i, opt := range d.Options {
		options[i] = pkgnet.Option{
			ID:    opt.ID,
			Label: opt.Label,
		}
	}

	return &pkgnet.Decision{
		ID:      d.ID,
		Prompt:  d.Prompt,
		Options: options,
		Timeout: d.Timeout,
		Default: d.Default,
	}
}

// BuildFullSync builds complete sync data for reconnecting players.
// Includes current state and recent game log entries.
func (b *Builder) BuildFullSync() *pkgnet.StateSync {
	return b.BuildStateSync()
}

// GetCurrentTurnEntries returns current turn's log entries for ActionSync.
func (b *Builder) GetCurrentTurnEntries() []gamelog.LogEntry {
	if b.game.Log == nil {
		return nil
	}
	return b.game.Log.GetCurrentTurnEntries()
}

// BuildActionSyncBatch builds multiple ActionSync from current turn entries.
func (b *Builder) BuildActionSyncBatch() []*pkgnet.ActionSync {
	entries := b.GetCurrentTurnEntries()
	result := make([]*pkgnet.ActionSync, len(entries))
	for i, entry := range entries {
		result[i] = b.BuildActionSync(entry)
	}
	return result
}