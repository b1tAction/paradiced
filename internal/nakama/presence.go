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
	h.logDebug("HandlePresenceJoin: player joining", "user_id", userID)

	// Check if player already exists (rejoin case or matchmaker pre-added)
	if h.players[userID] != nil {
		// Player was pre-added from matchmaker or is reconnecting
		if h.disconnected[userID] {
			// Player was disconnected (pre-added or previously disconnected)
			// Mark as connected and send full sync
			h.logDebug("HandlePresenceJoin: player reconnecting", "user_id", userID)
			h.disconnected[userID] = false
			if h.hsm != nil && h.hsm.IsRunning() {
				return h.handlePlayerRejoin(userID)
			}
			// Game not started yet, just mark as connected
			h.logDebug("HandlePresenceJoin: player marked connected (game not started)", "user_id", userID)
			return nil
		}
		// Player still connected (duplicate join) - ignore
		h.logWarn("HandlePresenceJoin: duplicate join ignored", "user_id", userID)
		return nil
	}

	// Check if match is full (for new players)
	if len(h.playerList) >= h.maxPlayers {
		h.logError("HandlePresenceJoin: match is full", "user_id", userID)
		return fmt.Errorf("match is full")
	}

	// Get faction from metadata (if provided)
	faction := parseFactionFromMetadata(metadata)
	h.logDebug("HandlePresenceJoin: faction determined", "user_id", userID, "faction", faction)

	// Add player to match (faction set via PlayerConfig)
	h.addPlayer(userID, faction)

	// Start the game only when match reaches max players and hasn't been initialized yet.
	// This avoids starting an empty/incomplete game before all expected players join.
	if len(h.playerList) >= h.maxPlayers && h.hsm == nil {
		h.logInfo("HandlePresenceJoin: match full, initializing game", "user_id", userID, "total_players", len(h.playerList))
		// Initialize game (factions already set, buff initialization happens in MatchInitState.Enter())
		if err := h.MatchInit(); err != nil {
			h.logError("HandlePresenceJoin: failed to initialize game", "error", err)
			return err
		}
	} else if h.hsm != nil && h.hsm.IsRunning() {
		// Game is already running, send state sync to new player
		h.logDebug("HandlePresenceJoin: sending state sync to late joiner", "user_id", userID)
		broadcastAdapter := NewNakamaBroadcastAdapter(h)
		builder := net.NewBuilder(h.hsm)
		stateSync := builder.BuildStateSync()
		turnSync := builder.BuildTurnSync()
		// Send full sync to joining player
		if err := broadcastAdapter.SendFullSync(userID, stateSync, turnSync); err != nil {
			h.logError("HandlePresenceJoin: failed to send state sync", "error", err)
			return err
		}
	}

	h.logInfo("HandlePresenceJoin: player joined successfully", "user_id", userID, "total_players", len(h.playerList))
	return nil
}

// handlePlayerRejoin handles a player rejoining an ongoing match.
func (h *NakamaMatchHandler) handlePlayerRejoin(userID string) error {
	h.logInfo("handlePlayerRejoin: player rejoining", "user_id", userID)

	player := h.players[userID]
	if player == nil {
		h.logWarn("handlePlayerRejoin: player not found", "user_id", userID)
		return nil
	}

	// Create broadcast adapter and builder
	broadcastAdapter := NewNakamaBroadcastAdapter(h)
	builder := net.NewBuilder(h.hsm)

	// Build full sync data
	stateSync := builder.BuildStateSync()
	turnSync := builder.BuildTurnSync()

	// Send full sync to rejoining player
	h.logDebug("handlePlayerRejoin: sending full sync", "user_id", userID)
	if err := broadcastAdapter.SendFullSync(userID, stateSync, turnSync); err != nil {
		h.logError("handlePlayerRejoin: failed to send full sync", "user_id", userID, "error", err)
		return err
	}

	// If match is currently in mini-game related states, re-send mini-game start
	// to the rejoining player. This avoids stalls when a player joins after the
	// original broadcast and therefore misses the trigger to submit result.
	if h.dispatcher != nil {
		globalState := h.hsm.GetGlobalStateID()
		h.logDebug("handlePlayerRejoin: checking mini-game state", "global_state", globalState.String())
		if globalState == hsm.StateMatchInit || globalState == hsm.StateRoundMiniGame {
			players := make([]string, len(h.playerList))
			copy(players, h.playerList)

			miniGameStart := &pkgnet.MiniGameStart{
				GameType: "dice_race",
				Players:  players,
			}

			data, err := json.Marshal(miniGameStart)
			if err == nil {
				h.logDebug("handlePlayerRejoin: sending mini-game start to rejoining player")
				_ = h.dispatcher.SendMessage(userID, int64(pkgnet.OpMiniGameStart), data)
			}
		}
	}

	// Re-send actionable prompt for the current turn when needed.
	// This is important for matchmaker-created matches where the game may have
	// already started before socket presences fully joined, causing the initial
	// Available/Decision message to be missed.
	if h.hsm == nil || !h.hsm.IsRunning() {
		h.logDebug("handlePlayerRejoin: HSM not running, skipping prompt resend")
		return nil
	}

	currentPlayer := h.getCurrentPlayer()
	if currentPlayer == nil || currentPlayer.ID.UUID() != player.ID.UUID() {
		h.logDebug("handlePlayerRejoin: not current player, skipping prompt resend")
		return nil
	}

	if h.hsm.IsWaiting() {
		decision := h.hsm.GetCurrentDecision()
		if decision == nil {
			h.logDebug("handlePlayerRejoin: HSM waiting but no decision to send")
			return nil
		}
		h.logDebug("handlePlayerRejoin: resending decision request", "player_id", userID)
		decisionReq := builder.BuildDecisionFromEvent(decision)
		return broadcastAdapter.SendDecision(userID, decisionReq)
	}

	if h.hsm.GetCurrentStateID() != hsm.StateMainAction {
		h.logDebug("handlePlayerRejoin: not in MainAction state, skipping prompt resend")
		return nil
	}

	available := builder.BuildAvailableForPlayer(player)
	if available == nil {
		h.logDebug("handlePlayerRejoin: no available actions for player")
		return nil
	}
	h.logDebug("handlePlayerRejoin: resending available actions", "player_id", userID)
	return broadcastAdapter.SendAvailable(userID, available)
}

// HandlePresenceLeave handles a player leaving the match.
// Called by Nakama when a player leaves/disconnects.
func (h *NakamaMatchHandler) HandlePresenceLeave(userID string) error {
	h.logDebug("HandlePresenceLeave: player leaving", "user_id", userID)

	// Get player
	player := h.players[userID]
	if player == nil {
		h.logDebug("HandlePresenceLeave: player not in match", "user_id", userID)
		return nil // Player not in match
	}

	// Mark player as disconnected (but keep in game for rejoin)
	h.disconnected[userID] = true
	h.logDebug("HandlePresenceLeave: player marked disconnected", "user_id", userID)

	// If game is running, handle player disconnect
	if h.hsm != nil && h.hsm.IsRunning() {
		// If current player disconnects during decision, handle gracefully
		currentPlayer := h.getCurrentPlayer()
		if currentPlayer != nil && currentPlayer.ID.UUID() == player.ID.UUID() {
			h.logWarn("HandlePresenceLeave: current turn player disconnected", "user_id", userID)
			// Current turn player disconnected - may need timeout handling
			// Decision timeout is handled by HSM, so we just mark disconnected
		}

		// Broadcast player disconnect to others
		h.logDebug("HandlePresenceLeave: broadcasting disconnect to others")
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
		h.logInfo("HandlePresenceLeave: all players disconnected, stopping match")
		h.MatchStop()
	}

	h.logInfo("HandlePresenceLeave: player left", "user_id", userID, "connected_count", h.GetConnectedPlayers())
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
