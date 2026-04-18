// Package nakama provides Nakama match handler implementation for Paradiced.
package nakama

import (
	"encoding/json"
	"fmt"

	"github.com/b1tAction/paradiced/internal/engine/hsm"
	"github.com/b1tAction/paradiced/internal/net"
	"github.com/b1tAction/paradiced/pkg/constants"
	"github.com/b1tAction/paradiced/pkg/util"
	pkgnet "github.com/b1tAction/paradiced/pkg/net"
)

func parseFactionFromMetadata(metadata *util.Metadata) constants.Faction {
	if metadata == nil {
		return constants.FactionQingLong
	}

	faction := metadata.GetStringOrDefault("faction", "")
	parsed := constants.ParseFaction(faction)
	if parsed.IsValid() {
		return parsed
	}

	return constants.FactionQingLong
}

// HandlePresenceJoin handles a player joining the match.
// Called by Nakama when a new player joins the match.
func (h *NakamaMatchHandler) HandlePresenceJoin(userID string, metadata *util.Metadata) error {
	// Check if player already exists (rejoin case or matchmaker pre-added)
	if h.players[userID] != nil {
		// Player was pre-added from matchmaker or is reconnecting
		if h.disconnected[userID] {
			// Player was disconnected (pre-added or previously disconnected)
			// Mark as connected and send full sync
			h.disconnected[userID] = false
			if h.hsm != nil && h.hsm.IsRunning() {
				return h.handlePlayerRejoin(userID)
			}
			// Game not started yet, just mark as connected
			return nil
		}
		// Player still connected (duplicate join) - ignore
		return nil
	}

	// Check if match is full (for new players)
	if len(h.playerList) >= h.maxPlayers {
		return fmt.Errorf("match is full")
	}

	// Get faction from metadata (if provided)
	faction := parseFactionFromMetadata(metadata)

	// Add player to match (faction set via PlayerConfig)
	h.addPlayer(userID, faction)

	// Start the game only when match reaches max players and hasn't been initialized yet.
	// This avoids starting an empty/incomplete game before all expected players join.
	if len(h.playerList) >= h.maxPlayers && h.hsm == nil {
		// Initialize game (factions already set, buff initialization happens in MatchInitState.Enter())
		if err := h.MatchInit(); err != nil {
			return err
		}
	} else if h.hsm != nil && h.hsm.IsRunning() {
		// Game is already running, send state sync to new player
		broadcastAdapter := NewNakamaBroadcastAdapter(h)
		builder := net.NewBuilder(h.hsm)
		stateSync := builder.BuildStateSync()
		turnSync := builder.BuildTurnSync()
		// Send full sync to joining player
		if err := broadcastAdapter.SendFullSync(userID, stateSync, turnSync); err != nil {
			return err
		}
	}

	return nil
}

// handlePlayerRejoin handles a player rejoining an ongoing match.
func (h *NakamaMatchHandler) handlePlayerRejoin(userID string) error {
	player := h.players[userID]
	if player == nil {
		return nil
	}

	// Create broadcast adapter and builder
	broadcastAdapter := NewNakamaBroadcastAdapter(h)
	builder := net.NewBuilder(h.hsm)

	// Build full sync data
	stateSync := builder.BuildStateSync()
	turnSync := builder.BuildTurnSync()

	// Send full sync to rejoining player
	if err := broadcastAdapter.SendFullSync(userID, stateSync, turnSync); err != nil {
		return err
	}

	// If match is currently in mini-game related states, re-send mini-game start
	// to the rejoining player. This avoids stalls when a player joins after the
	// original broadcast and therefore misses the trigger to submit result.
	if h.dispatcher != nil {
		globalState := h.hsm.GetGlobalStateID()
		if globalState == hsm.StateMatchInit || globalState == hsm.StateRoundMiniGame {
			players := make([]string, len(h.playerList))
			copy(players, h.playerList)

			miniGameStart := &pkgnet.MiniGameStart{
				GameType: "dice_race",
				Players:  players,
			}

			data, err := json.Marshal(miniGameStart)
			if err == nil {
				_ = h.dispatcher.SendMessage(userID, int64(pkgnet.OpMiniGameStart), data)
			}
		}
	}

	// Re-send actionable prompt for the current turn when needed.
	// This is important for matchmaker-created matches where the game may have
	// already started before socket presences fully joined, causing the initial
	// Available/Decision message to be missed.
	if h.hsm == nil || !h.hsm.IsRunning() {
		return nil
	}

	currentPlayer := h.getCurrentPlayer()
	if currentPlayer == nil || currentPlayer.ID.UUID() != player.ID.UUID() {
		return nil
	}

	if h.hsm.IsWaiting() {
		decision := h.hsm.GetCurrentDecision()
		if decision == nil {
			return nil
		}
		decisionReq := builder.BuildDecisionFromEvent(decision)
		return broadcastAdapter.SendDecision(userID, decisionReq)
	}

	if h.hsm.GetCurrentStateID() != hsm.StateMainAction {
		return nil
	}

	available := builder.BuildAvailableForPlayer(player)
	if available == nil {
		return nil
	}
	return broadcastAdapter.SendAvailable(userID, available)
}

// HandlePresenceLeave handles a player leaving the match.
// Called by Nakama when a player leaves/disconnects.
func (h *NakamaMatchHandler) HandlePresenceLeave(userID string) error {
	// Get player
	player := h.players[userID]
	if player == nil {
		return nil // Player not in match
	}

	// Mark player as disconnected (but keep in game for rejoin)
	h.disconnected[userID] = true

	// If game is running, handle player disconnect
	if h.hsm != nil && h.hsm.IsRunning() {
		// If current player disconnects during decision, handle gracefully
		currentPlayer := h.getCurrentPlayer()
		if currentPlayer != nil && currentPlayer.ID.UUID() == player.ID.UUID() {
			// Current turn player disconnected - may need timeout handling
			// Decision timeout is handled by HSM, so we just mark disconnected
		}

		// Broadcast player disconnect to others
		broadcast := NewNakamaBroadcastAdapter(h)
		builder := net.NewBuilder(h.hsm)
		stateSync := builder.BuildStateSync()
		stateSync.Players = builder.BuildPlayers() // Update players list with disconnect status
		broadcast.BroadcastStateSync(stateSync)
	}

	// If all players are disconnected, stop the match
	allDisconnected := true
	for _, id := range h.playerList {
		if !h.disconnected[id] {
			allDisconnected = false
			break
		}
	}
	if allDisconnected {
		h.MatchStop()
	}

	return nil
}

// IsPlayerConnected returns whether a player is currently connected.
func (h *NakamaMatchHandler) IsPlayerConnected(userID string) bool {
	return h.players[userID] != nil && !h.disconnected[userID]
}

// GetConnectedPlayers returns list of currently connected player IDs.
func (h *NakamaMatchHandler) GetConnectedPlayers() []string {
	result := make([]string, 0)
	for _, id := range h.playerList {
		if !h.disconnected[id] {
			result = append(result, id)
		}
	}
	return result
}
