// Package nakama provides Nakama match handler implementation for Paradiced.
package nakama

import (
	"time"

	"github.com/b1tAction/paradiced/internal/core"
	"github.com/b1tAction/paradiced/internal/engine/hsm"
	"github.com/b1tAction/paradiced/pkg/constants"
	"github.com/b1tAction/paradiced/pkg/id"
	"github.com/b1tAction/paradiced/pkg/protocol"
)

// MatchInit initializes the match when created.
// Called by Nakama when a new match is created.
func (h *NakamaMatchHandler) MatchInit() error {
	// Initialize game instance
	if err := h.initializeGame(); err != nil {
		return err
	}

	// Create broadcast adapter
	broadcastAdapter := NewNakamaBroadcastAdapter(h)

	// Create map engine adapter for HSM
	mapAdapter := hsm.NewMapEngineWrapper(h.mapEngine)

	// Create state context for HSM
	ctx := hsm.NewStateContext().
		WithGame(h.game).
		WithBus(hsm.NewEventBusWrapper(h.game.Bus)).
		WithMapEngine(mapAdapter).
		WithBroadcast(broadcastAdapter)

	// Start HSM - enters MatchInit state
	h.hsm.Start(hsm.StateMatchInit, ctx)

	return nil
}

// MatchLoop is called each tick to update game state.
// Called by Nakama at regular intervals (e.g., every 100ms).
func (h *NakamaMatchHandler) MatchLoop(delta time.Duration) error {
	// Check if HSM is running
	if !h.hsm.IsRunning() {
		return nil // Match ended or paused
	}

	// Create state context with current player
	currentPlayer := h.getCurrentPlayer()
	ctx := hsm.NewStateContext().
		WithGame(h.game).
		WithPlayer(currentPlayer).
		WithBus(hsm.NewEventBusWrapper(h.game.Bus)).
		WithMapEngine(hsm.NewMapEngineWrapper(h.mapEngine)).
		WithBroadcast(NewNakamaBroadcastAdapter(h))

	// Update HSM
	nextState, err := h.hsm.Update(ctx)
	if err != nil {
		return err
	}

	// Handle state transitions
	if nextState != hsm.StateNone {
		// Transition to new state
		h.hsm.TransitionTo(nextState, ctx)
	}

	// Handle waiting for decisions
	if h.hsm.IsWaiting() {
		// Send decision request to current player
		// Decision timeout is handled by HSM
	}

	_ = delta // Delta time not currently used

	return nil
}

// MatchStop terminates the match.
// Called by Nakama when match ends.
func (h *NakamaMatchHandler) MatchStop() error {
	// Create final state context
	ctx := hsm.NewStateContext().WithGame(h.game)

	// Stop HSM
	h.hsm.Stop(ctx)

	// Clear resources
	h.players = make(map[string]*core.Player)
	h.playerList = make([]string, 0)

	return nil
}

// getCurrentPlayer returns the current player for the turn.
func (h *NakamaMatchHandler) getCurrentPlayer() *core.Player {
	if h.game == nil || len(h.game.Players) == 0 {
		return nil
	}

	turnIndex := h.game.State.Turn
	if turnIndex >= 0 && turnIndex < len(h.game.Players) {
		return h.game.Players[turnIndex]
	}

	return nil
}

// addPlayer adds a new player to the match.
// Called during MatchInit or when players join.
func (h *NakamaMatchHandler) addPlayer(userID string, faction protocol.Faction) *core.Player {
	playerID := id.NewPlayerID()
	player := core.NewPlayer(core.PlayerConfig{
		ID:      playerID,
		MaxHP:   6,
		MaxLP:   8,
		Faction: faction,
	})

	// Store player
	h.players[userID] = player
	h.playerList = append(h.playerList, userID)

	return player
}

// assignFactions assigns factions to players based on join order.
func (h *NakamaMatchHandler) assignFactions() {
	factions := []protocol.Faction{
		protocol.FactionQingLong,
		protocol.FactionZhuQue,
		protocol.FactionBaiHu,
		protocol.FactionXuanWu,
	}

	for i, userID := range h.playerList {
		player := h.players[userID]
		if player != nil && i < len(factions) {
			// Faction is set via PlayerConfig, but we can update the field
			// Player.Faction is a private field, set via constructor
			// For re-assignment, we need to use reflection or a setter method
			// For now, we'll create new players with correct factions in addPlayer

			// ZhuQue players get Fire buff (离火 passive)
			if factions[i] == protocol.FactionZhuQue {
				player.AddBuff(core.NewBuff(constants.BuffTypeFire, -1)) // Permanent buff
			}
		}
	}
}