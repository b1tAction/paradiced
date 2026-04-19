// Package nakama provides Nakama match handler implementation for Paradiced.
package nakama

import (
	"time"

	"github.com/b1tAction/paradiced/internal/core"
	"github.com/b1tAction/paradiced/internal/engine/hsm"
	"github.com/b1tAction/paradiced/internal/net"
	"github.com/b1tAction/paradiced/pkg/constants"
	pkgnet "github.com/b1tAction/paradiced/pkg/net"
	"github.com/b1tAction/paradiced/pkg/id"
)

// MatchInit initializes the match when created.
// Called by Nakama when a new match is created.
func (h *NakamaMatchHandler) MatchInit() error {
	if h.logger != nil {
		h.logger.Info("MatchInit: initializing match", "match_id", h.matchID)
	}

	// Initialize game instance
	if h.logger != nil {
		h.logger.Debug("MatchInit: initializing game")
	}
	if err := h.initializeGame(); err != nil {
		if h.logger != nil {
			h.logger.Error("MatchInit: failed to initialize game", "error", err)
		}
		return err
	}
	if h.logger != nil {
		h.logger.Debug("MatchInit: game initialized", "game_id", h.hsm.GetGame().ID)
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

	// Check state execution error
	if ctx.Error != nil {
		if h.logger != nil {
			h.logger.Error("MatchInit: state execution failed",
				"state", hsm.StateMatchInit.String(),
				"error", ctx.Error.Error())
		}
		errCode := ErrorCodeForError(ctx.Error)
		return h.sendActionRejectedWithCode("", pkgnet.OpStateSync, errCode, ctx.Error.Error())
	}

	if h.logger != nil {
		h.logger.Debug("MatchInit: starting HSM")
		h.logger.Info("MatchInit: HSM started", "initial_state", hsm.StateMatchInit.String())
	}

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

	globalStateID := h.hsm.GetGlobalStateID()
	turnStateID := h.hsm.GetTurnStateID()

	if h.logger != nil {
		h.logger.Debug("MatchLoop: tick",
			"global_state", globalStateID.String(),
			"turn_state", turnStateID.String(),
			"current_player", func() string {
				if currentPlayer != nil {
					return currentPlayer.ID.UUID()
				}
				return "none"
			}())
	}

	// Handle TurnEnd state - trigger next turn or round
	if globalStateID == hsm.StateTurnLoop &&
		turnStateID == hsm.StateTurnEnd {
		if h.logger != nil {
			h.logger.Debug("MatchLoop: handling TurnEnd state")
		}
		// Get TurnLoop state
		globalState := h.hsm.GetGlobalState()
		turnLoopState, ok := globalState.(*hsm.TurnLoopState)
		if ok {
			// Mark turn as complete
			turnLoopState.OnTurnComplete(ctx)

			// Start next player's turn
			nextState := turnLoopState.StartPlayerTurn(ctx)
			if nextState != hsm.StateNone {
				if h.logger != nil {
					h.logger.Debug("MatchLoop: starting next player turn", "next_state", nextState.String())
				}
				h.hsm.TransitionTo(nextState, ctx)
			}
		}
	}

	// Update HSM
	if h.logger != nil {
		h.logger.Debug("MatchLoop: updating HSM")
	}
	nextState, err := h.hsm.Update(ctx)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("MatchLoop: HSM update failed", "error", err)
		}
		return err
	}

	// Check state execution error
	if ctx.Error != nil {
		if h.logger != nil {
			h.logger.Error("MatchLoop: state execution failed",
				"global_state", globalStateID.String(),
				"turn_state", turnStateID.String(),
				"error", ctx.Error.Error())
		}
		errCode := ErrorCodeForError(ctx.Error)
		return h.broadcastErrorState(errCode, ctx.Error.Error())
	}

	// Handle state transitions
	if nextState != hsm.StateNone {
		if h.logger != nil {
			h.logger.Debug("MatchLoop: transitioning to state", "next_state", nextState.String())
		}
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

				if h.logger != nil {
					h.logger.Debug("MatchLoop: sending decision request",
						"decision_id", decisionUUID,
						"player_id", currentPlayer.ID.UUID())
				}

				// Build decision request
				decisionReq := builder.BuildDecisionFromEvent(decision)

				// Send to current player
				broadcastAdapter.SendDecision(currentPlayer.ID.UUID(), decisionReq)
			}
		}
	} else {
		// Clear last decision ID when not waiting
		if h.lastDecisionID != "" && h.logger != nil {
			h.logger.Debug("MatchLoop: clearing decision ID (not waiting)")
		}
		h.lastDecisionID = ""
	}

	_ = delta // Delta time not currently used

	return nil
}

// MatchStop terminates the match.
// Called by Nakama when match ends.
func (h *NakamaMatchHandler) MatchStop() error {
	if h.logger != nil {
		h.logger.Info("MatchStop: terminating match", "match_id", h.matchID)
	}

	// Create final state context
	ctx := hsm.NewStateContext().WithHSM(h.hsm)

	// Stop HSM
	if h.logger != nil {
		h.logger.Debug("MatchStop: stopping HSM")
	}
	h.hsm.Stop(ctx)

	// Clear resources
	if h.logger != nil {
		h.logger.Debug("MatchStop: clearing player data",
			"players_count", len(h.players),
			"player_list_len", len(h.playerList))
	}
	h.players = make(map[string]*core.Player)
	h.playerList = make([]string, 0)
	h.disconnected = make(map[string]bool)

	if h.logger != nil {
		h.logger.Info("MatchStop: match terminated")
	}
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

	if h.logger != nil {
		h.logger.Debug("Player added to match",
			"user_id", userID,
			"player_id", playerID.UUID(),
			"faction", faction,
			"total_players", len(h.playerList))
	}

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

// broadcastErrorState broadcasts an error state to all clients.
// Used when a state execution error occurs during MatchLoop.
func (h *NakamaMatchHandler) broadcastErrorState(errCode constants.ErrorCode, message string) error {
	// Log error for debugging
	if h.logger != nil {
		h.logger.Error("MatchLoop: state execution error",
			"error_code", errCode,
			"message", message)
	}

	// Return error to stop MatchLoop - error will be handled by Nakama runtime
	return nil
}