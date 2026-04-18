// Package nakama provides Nakama match handler implementation for Paradiced.
package nakama

import (
	"time"

	"github.com/b1tAction/paradiced/internal/core"
	"github.com/b1tAction/paradiced/internal/engine/hsm"
	"github.com/b1tAction/paradiced/internal/net"
	"github.com/b1tAction/paradiced/pkg/constants"
	"github.com/b1tAction/paradiced/pkg/id"
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

	// Create builder for protocol messages
	builder := net.NewBuilder(h.hsm)

	// Set map engine in HSM
	h.hsm.SetMapEngine(h.mapEngine)

	// Create state context with all components
	ctx := hsm.NewStateContext().
		WithHSM(h.hsm).
		WithBroadcast(broadcastAdapter).
		WithBuilder(builder)

	// Start HSM - enters MatchInit state
	h.hsm.Start(hsm.StateMatchInit, ctx)

	return nil
}

// MatchLoop is called each tick to update game state.
// Called by Nakama at regular intervals (e.g., every 100ms).
func (h *NakamaMatchHandler) MatchLoop(delta time.Duration) error {
	// Check if HSM is initialized
	if h.hsm == nil {
		// Match not yet initialized (matchmaker-created match waiting for players)
		return nil
	}

	// Check if HSM is running
	if !h.hsm.IsRunning() {
		return nil // Match ended or paused
	}

	// Create state context with current player (using HSM reference)
	currentPlayer := h.getCurrentPlayer()
	broadcastAdapter := NewNakamaBroadcastAdapter(h)
	builder := net.NewBuilder(h.hsm)
	ctx := hsm.NewStateContext().
		WithHSM(h.hsm).
		WithPlayer(currentPlayer).
		WithBroadcast(broadcastAdapter).
		WithBuilder(builder)

	// Handle TurnEnd state - trigger next turn or round
	if h.hsm.GetGlobalStateID() == hsm.StateTurnLoop &&
		h.hsm.GetTurnStateID() == hsm.StateTurnEnd {
		// Get TurnLoop state
		globalState := h.hsm.GetGlobalState()
		turnLoopState, ok := globalState.(*hsm.TurnLoopState)
		if ok {
			// Mark turn as complete
			turnLoopState.OnTurnComplete(ctx)

			// Start next player's turn
			nextState := turnLoopState.StartPlayerTurn(ctx)
			if nextState != hsm.StateNone {
				h.hsm.TransitionTo(nextState, ctx)
			}
		}
	}

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

	// Handle waiting for decisions - send decision request to current player
	if h.hsm.IsWaiting() {
		decision := h.hsm.GetCurrentDecision()
		currentPlayer := h.getCurrentPlayer()

		// Only send if we have a decision and haven't sent it yet
		if decision != nil && currentPlayer != nil {
			decisionUUID := decision.ID.UUID()

			// Prevent duplicate sends
			if h.lastDecisionID != decisionUUID {
				h.lastDecisionID = decisionUUID

				// Build decision request
				decisionReq := builder.BuildDecisionFromEvent(decision)

				// Send to current player
				broadcastAdapter.SendDecision(currentPlayer.ID.UUID(), decisionReq)
			}
		}
	} else {
		// Clear last decision ID when not waiting
		h.lastDecisionID = ""
	}

	_ = delta // Delta time not currently used

	return nil
}

// MatchStop terminates the match.
// Called by Nakama when match ends.
func (h *NakamaMatchHandler) MatchStop() error {
	// Create final state context
	ctx := hsm.NewStateContext().WithHSM(h.hsm)

	// Stop HSM
	h.hsm.Stop(ctx)

	// Clear resources
	h.players = make(map[string]*core.Player)
	h.playerList = make([]string, 0)
	h.disconnected = make(map[string]bool)

	return nil
}

// getCurrentPlayer returns the current player for the turn.
func (h *NakamaMatchHandler) getCurrentPlayer() *core.Player {
	if h.hsm == nil {
		return nil
	}
	game := h.hsm.GetGame()
	if game == nil || len(game.Players) == 0 {
		return nil
	}

	turnIndex := h.hsm.GetTurn()
	if turnIndex >= 0 && turnIndex < len(game.Players) {
		return game.Players[turnIndex]
	}

	return nil
}

// addPlayer adds a new player to the match.
// Called during MatchInit or when players join.
func (h *NakamaMatchHandler) addPlayer(userID string, faction constants.Faction) *core.Player {
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
// Note: Faction-specific buffs (like ZhuQue Fire) are added later by
// game.InitializePlayerFactionBuffs() during MatchInitState.Enter().
func (h *NakamaMatchHandler) assignFactions() {
	// This method is deprecated - factions are set during addPlayer via PlayerConfig.
	// The function exists for backwards compatibility but does nothing.
	// Buff initialization is handled by engine.InitializePlayerFactionBuffs().
}