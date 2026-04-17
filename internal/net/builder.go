// Package net provides synchronization data builder
// for converting internal game structures to protocol messages.
package net

import (
	pkgnet "github.com/b1tAction/paradiced/pkg/net"
	"github.com/b1tAction/paradiced/internal/core"
	"github.com/b1tAction/paradiced/internal/core/buff"
	"github.com/b1tAction/paradiced/internal/core/item"
	"github.com/b1tAction/paradiced/internal/engine"
	"github.com/b1tAction/paradiced/internal/engine/hsm"
	"github.com/b1tAction/paradiced/pkg/constants"
	"github.com/b1tAction/paradiced/pkg/gamelog"
	"github.com/b1tAction/paradiced/pkg/rng"
)

// Builder converts internal game structures to protocol sync data.
// Used by MatchHandler implementations to build messages for clients.
type Builder struct {
	hsm         *hsm.HSM
	game        *engine.Game
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

// BuildTurnSync builds a turn sync message with all log entries.
// Client loops through entries and plays animations sequentially.
// No conversion needed - directly uses GameLog entries.
func (b *Builder) BuildTurnSync() *pkgnet.TurnSync {
	entries := b.GetCurrentTurnEntries()

	player := b.hsm.GetTurnPlayer()
	playerID := ""
	if player != nil {
		playerID = player.ID.UUID()
	}

	return &pkgnet.TurnSync{
		Round:   b.game.State.Round,
		Turn:    b.game.State.Turn,
		Player:  playerID,
		Entries: entries,
	}
}

// BuildPlayers builds all player state snapshots.
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
		Faction:     p.Faction.SnakeCase(),
		Position:    p.Position,
		HP:          p.HP,
		LP:          p.LP,
		Buffs:       b.BuildBuffs(p.ActiveBuffs),
		Items:       b.BuildItems(p.Inventory),
		Charge:      p.GetChargeCount(),
		FireCounter: p.GetFireCounter(),
		IsDead:      p.IsDead,
		SkipTurn:    p.SkipTurn,
	}
}

// BuildBuffs builds buff sync data from active buffs.
// Includes display name from definition.
func (b *Builder) BuildBuffs(activeBuffs []*buff.Buff) []pkgnet.Buff {
	result := make([]pkgnet.Buff, len(activeBuffs))
	for i, bf := range activeBuffs {
		def := buff.GetBuffDefinition(bf.Type)
		name := ""
		if def != nil {
			name = def.Name
		}
		result[i] = pkgnet.Buff{
			Type:     string(bf.Type),
			Name:     name,
			Duration: bf.Duration,
		}
	}
	return result
}

// BuildItems builds item sync data from inventory.
// Includes display name from definition.
func (b *Builder) BuildItems(inventory []*item.Item) []pkgnet.Item {
	result := make([]pkgnet.Item, len(inventory))
	for i, it := range inventory {
		def := item.GetItemDefinition(it.Type)
		name := ""
		if def != nil {
			name = def.Name
		}
		result[i] = pkgnet.Item{
			ID:   it.ID.UUID(),
			Type: string(it.Type),
			Name: name,
		}
	}
	return result
}

// BuildAvailable builds available actions for the current player.
// Includes PhaseAnyTime items and faction skill availability.
func (b *Builder) BuildAvailable(player *core.Player) *pkgnet.Available {
	// Build items with names
	usableItems := make([]pkgnet.Item, 0)
	for _, it := range player.Inventory {
		if it.Usable {
			def := item.GetItemDefinition(it.Type)
			name := ""
			if def != nil {
				name = def.Name
			}
			usableItems = append(usableItems, pkgnet.Item{
				ID:   it.ID.UUID(),
				Type: string(it.Type),
				Name: name,
			})
		}
	}

	// Check faction skill availability
	canUseSkill := false
	switch player.Faction {
	case constants.FactionQingLong, constants.FactionXuanWu:
		// 青龙行迹 / 玄武镇厄: requires charge count >= 1
		canUseSkill = player.GetChargeCount() >= 1
	}

	return &pkgnet.Available{
		Items:       usableItems,
		CanUseSkill: canUseSkill,
		DiceType:    b.turnDiceType.String(),
	}
}

// BuildDecision builds decision request from event.Decision.
func (b *Builder) BuildDecision(decisionID string, prompt string, context string, options []pkgnet.Option, timeout int, defaultIdx int) *pkgnet.Decision {
	return &pkgnet.Decision{
		ID:      decisionID,
		Prompt:  prompt,
		Context: context,
		Options: options,
		Timeout: timeout,
		Default: defaultIdx,
	}
}

// BuildFullSync builds complete sync data for reconnecting players.
func (b *Builder) BuildFullSync() (*pkgnet.StateSync, *pkgnet.TurnSync) {
	return b.BuildStateSync(), b.BuildTurnSync()
}

// GetCurrentTurnEntries returns current turn's log entries.
func (b *Builder) GetCurrentTurnEntries() []gamelog.LogEntry {
	if b.game.Log == nil {
		return nil
	}
	return b.game.Log.GetCurrentTurnEntries()
}