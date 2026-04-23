// Package net provides synchronization data builder
// for converting internal game structures to protocol messages.
package net

import (
	pkgnet "github.com/b1tAction/paradiced/pkg/net"
	"github.com/b1tAction/paradiced/internal/core"
	"github.com/b1tAction/paradiced/internal/engine"
	"github.com/b1tAction/paradiced/internal/engine/hsm"
	"github.com/b1tAction/paradiced/internal/event"
	"github.com/b1tAction/paradiced/pkg/constants"
	"github.com/b1tAction/paradiced/pkg/gamelog"
	"github.com/b1tAction/paradiced/pkg/rng"
)

// Builder converts internal game structures to protocol sync data.
// Used by MatchHandler implementations to build messages for clients.
// HSM is the single source of truth - Game is accessed via hsm.GetGame().
type Builder struct {
	hsm         *hsm.HSM
	turnDiceType rng.DiceType // Current player's dice type (from StateContext)
}

// NewBuilder creates a new sync data builder.
// Game is accessed via hsm.GetGame() - no need to pass separately.
func NewBuilder(hsmInstance *hsm.HSM) *Builder {
	return &Builder{
		hsm: hsmInstance,
	}
}

// SetDiceType sets the current player's dice type for BuildAvailable.
// Accepts string format ("gold", "silver", "copper", "wood") for pkg/net.Builder interface.
func (b *Builder) SetDiceType(diceType string) {
	b.turnDiceType = rng.DiceTypeFromString(diceType)
}

// SetDiceTypeFromRng sets the current player's dice type using rng.DiceType directly.
// Used internally when dice type is already known as rng.DiceType.
func (b *Builder) SetDiceTypeFromRng(diceType rng.DiceType) {
	b.turnDiceType = diceType
}

// BuildStateSync builds a complete state sync message.
func (b *Builder) BuildStateSync() *pkgnet.StateSync {
	globalState := b.hsm.GetGlobalStateID()
	turnState := b.hsm.GetTurnStateID()
	turnPlayer := b.hsm.GetTurnPlayer()

	var currentPlayerID string
	if turnPlayer != nil {
		currentPlayerID = turnPlayer.ID.UUID()
	}

	return &pkgnet.StateSync{
		GlobalState:     globalState.String(),
		TurnState:       turnState.String(),
		CurrentPlayerID: currentPlayerID,
		Round:           b.hsm.GetRound(),
		Turn:            b.hsm.GetTurn(),
		Paused:          b.hsm.IsPaused(),
		Players:         b.BuildPlayers(),
		Map:             *b.BuildMapInfo(),
	}
}

// BuildTurnSync builds a turn sync message with all log entries.
func (b *Builder) BuildTurnSync() *pkgnet.TurnSync {
	entries := b.GetCurrentTurnEntries()

	player := b.hsm.GetTurnPlayer()
	currentPlayerID := ""
	if player != nil {
		currentPlayerID = player.ID.UUID()
	}

	return &pkgnet.TurnSync{
		Round:           b.hsm.GetRound(),
		Turn:            b.hsm.GetTurn(),
		CurrentPlayerID: currentPlayerID,
		Entries:         entries,
	}
}

// BuildPlayers builds all player state snapshots.
func (b *Builder) BuildPlayers() []pkgnet.Player {
	game := b.hsm.GetGame()
	if game == nil {
		return nil
	}
	players := game.GetPlayers()
	result := make([]pkgnet.Player, len(players))
	for i, p := range players {
		result[i] = b.BuildPlayer(p)
	}
	return result
}

// BuildPlayer builds a single player state snapshot.
func (b *Builder) BuildPlayer(p *core.Player) pkgnet.Player {
	return pkgnet.Player{
		PlayerID:    p.ID.UUID(),
		Faction:     string(p.Faction),
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
// Hidden buffs are filtered out (not sent to client).
func (b *Builder) BuildBuffs(activeBuffs []*core.Buff) []pkgnet.Buff {
	result := make([]pkgnet.Buff, 0, len(activeBuffs))
	for _, bf := range activeBuffs {
		// Skip hidden buffs (internal mechanism, not visible to player)
		if engine.IsHidden(bf.Type) {
			continue
		}
		result = append(result, pkgnet.Buff{
			Type:     string(bf.Type),
			Name:     engine.GetBuffName(bf.Type),
			Duration: bf.Duration,
		})
	}
	return result
}

// BuildItems builds item sync data from inventory.
func (b *Builder) BuildItems(inventory []*core.Item) []pkgnet.Item {
	result := make([]pkgnet.Item, len(inventory))
	for i, it := range inventory {
		result[i] = pkgnet.Item{
			ID:   it.ID.UUID(),
			Type: string(it.Type),
			Name: engine.GetItemName(it.Type),
		}
	}
	return result
}

// BuildAvailable builds available actions for the current player.
// Implements pkg/net.Builder interface - gets player from HSM.
func (b *Builder) BuildAvailable() *pkgnet.Available {
	player := b.hsm.GetTurnPlayer()
	if player == nil {
		return nil
	}
	return b.BuildAvailableForPlayer(player)
}

// BuildAvailableForPlayer builds available actions for a specific player.
// Used internally when player is already known.
func (b *Builder) BuildAvailableForPlayer(player *core.Player) *pkgnet.Available {
	usableItems := make([]pkgnet.Item, 0)
	for _, it := range player.Inventory {
		if it.Usable {
			usableItems = append(usableItems, pkgnet.Item{
				ID:   it.ID.UUID(),
				Type: string(it.Type),
				Name: engine.GetItemName(it.Type),
			})
		}
	}

	canUseSkill := false
	switch player.Faction {
	case constants.FactionQingLong, constants.FactionXuanWu:
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

// BuildDecisionFromEvent builds pkgnet.Decision from event.Decision.
// Converts internal decision structure to protocol format for client.
func (b *Builder) BuildDecisionFromEvent(decision *event.Decision) *pkgnet.Decision {
	if decision == nil {
		return nil
	}

	// Convert options
	options := make([]pkgnet.Option, len(decision.Options))
	for i, opt := range decision.Options {
		options[i] = pkgnet.Option{
			ID:    opt.ID,
			Label: opt.Label,
		}
	}

	// Build context string from source info
	context := decision.SourceType
	if decision.SourceID != "" {
		context = decision.SourceType + "_" + decision.SourceID
	}

	// Convert timeout to seconds
	timeoutSec := int(decision.Timeout.Seconds())

	return &pkgnet.Decision{
		ID:      decision.ID.UUID(),
		Prompt:  decision.Prompt,
		Context: context,
		Options: options,
		Timeout: timeoutSec,
		Default: decision.Default,
	}
}

// BuildFullSync builds complete sync data for reconnecting players.
func (b *Builder) BuildFullSync() (*pkgnet.StateSync, *pkgnet.TurnSync) {
	return b.BuildStateSync(), b.BuildTurnSync()
}

// BuildMapInfo builds map information from MapEngine data.
// Implements pkg/net.Builder interface.
func (b *Builder) BuildMapInfo() *pkgnet.MapInfo {
	mapEngine := b.hsm.GetMapEngine()
	if mapEngine == nil {
		return &pkgnet.MapInfo{}
	}

	cells := make([]pkgnet.CellInfo, len(mapEngine.Cells))
	for i, cell := range mapEngine.Cells {
		cellInfo := pkgnet.CellInfo{
			Index:    cell.Index,
			CellType: string(cell.CellType),
		}
		if cell.EventID != "" {
			cellInfo.EventID = cell.EventID
		}
		if cell.IsBroken {
			cellInfo.IsBroken = true
		}
		cells[i] = cellInfo
	}

	return &pkgnet.MapInfo{
		Length: mapEngine.Length,
		Cells: cells,
	}
}

// GetCurrentTurnEntries returns current turn's log entries.
func (b *Builder) GetCurrentTurnEntries() []gamelog.LogEntry {
	game := b.hsm.GetGame()
	if game == nil || game.Log == nil {
		return nil
	}
	return game.Log.GetCurrentTurnEntries()
}