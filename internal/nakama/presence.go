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

func parseDisplayNameFromMetadata(metadata *util.Metadata, userID string) string {
	if metadata == nil {
		return userID // fallback to userID
	}

	displayName := metadata.GetStringOrDefault("display_name", "")
	if displayName == "" {
		return userID // fallback to userID
	}

	return displayName
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

	// Get faction and display name from metadata (if provided)
	faction := parseFactionFromMetadata(metadata)
	displayName := parseDisplayNameFromMetadata(metadata, userID)
	h.logDebug("HandlePresenceJoin: faction and display_name determined", "user_id", userID, "faction", faction, "display_name", displayName)

	// Set hostUserID for first player joining
	if len(h.playerList) == 0 {
		h.hostUserID = userID
		h.logInfo("HandlePresenceJoin: host set", "user_id", userID)
	}

	// Add player to match (faction and displayName set via PlayerConfig and Metadata)
	player := h.addPlayer(userID, faction, displayName)

	// Initialize game when first player joins (enter WaitingForHost state)
	// This allows manual start instead of auto-start at max players
	if len(h.playerList) == 1 && h.hsm == nil {
		h.logInfo("HandlePresenceJoin: first player joined, initializing game", "user_id", userID)
		if err := h.MatchInit(); err != nil {
			h.logError("HandlePresenceJoin: failed to initialize game", "error", err)
			return err
		}
	} else if h.hsm != nil {
		// Game already initialized - add late-joining player to game
		game := h.hsm.GetGame()
		if game != nil {
			h.logInfo("HandlePresenceJoin: adding late-joining player to game", "user_id", userID)
			game.AddPlayer(player)
		}
	}

	// Broadcast WaitingSync to host when in WaitingForHost state
	if h.hsm != nil && h.hsm.GetGlobalStateID() == hsm.StateWaitingForHost {
		h.broadcastWaitingSyncToAll()
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

// broadcastWaitingSyncToHost broadcasts waiting room status to the host.
func (h *NakamaMatchHandler) broadcastWaitingSyncToHost() {
	if h.hostUserID == "" {
		h.logWarn("broadcastWaitingSyncToHost: no host set")
		return
	}

	if h.dispatcher == nil {
		h.logDebug("broadcastWaitingSyncToHost: no dispatcher set")
		return
	}

	// Build waiting players list
	waitingPlayers := make([]pkgnet.WaitingPlayer, len(h.playerList))
	for i, userID := range h.playerList {
		player := h.players[userID]
		if player == nil {
			continue
		}
		waitingPlayers[i] = pkgnet.WaitingPlayer{
			UserID:      userID,
			DisplayName: player.Metadata.GetStringOrDefault("display_name", userID),
			Faction:     string(player.Faction),
			IsHost:      userID == h.hostUserID,
		}
	}

	// Build waiting sync message
	playerCount := len(h.playerList)
	canStart := playerCount >= 2 // Minimum 2 players to start

	message := fmt.Sprintf("Waiting for host to start (%d/%d players)", playerCount, h.maxPlayers)
	if canStart {
		message = fmt.Sprintf("Ready to start (%d players). Type 'start' to begin.", playerCount)
	}

	// Note: MatchID is not set here because it's the internal game ID, not Nakama's match ID.
	// Client already knows the correct match ID from the RPC response or JoinMatch parameter.
	waitingSync := &pkgnet.WaitingSync{
		MatchID:     "", // Client knows its match ID from RPC/JoinMatch
		HostUserID:  h.hostUserID,
		Players:     waitingPlayers,
		PlayerCount: playerCount,
		MinPlayers:  2,
		MaxPlayers:  h.maxPlayers,
		CanStart:    canStart,
		Message:     message,
	}

	// Send to host
	adapter := NewNakamaBroadcastAdapter(h)
	if err := adapter.SendWaitingSync(h.hostUserID, waitingSync); err != nil {
		h.logError("broadcastWaitingSyncToHost: failed to send", "error", err)
	}

	h.logDebug("broadcastWaitingSyncToHost: sent to host", "host_user_id", h.hostUserID, "player_count", playerCount)
}

// broadcastWaitingSyncToAll broadcasts waiting room status to all connected players.
func (h *NakamaMatchHandler) broadcastWaitingSyncToAll() {
	if h.dispatcher == nil {
		h.logDebug("broadcastWaitingSyncToAll: no dispatcher set")
		return
	}

	// Build waiting players list
	waitingPlayers := make([]pkgnet.WaitingPlayer, len(h.playerList))
	for i, userID := range h.playerList {
		player := h.players[userID]
		if player == nil {
			continue
		}
		waitingPlayers[i] = pkgnet.WaitingPlayer{
			UserID:      userID,
			DisplayName: player.Metadata.GetStringOrDefault("display_name", userID),
			Faction:     string(player.Faction),
			IsHost:      userID == h.hostUserID,
		}
	}

	// Build waiting sync message
	playerCount := len(h.playerList)
	canStart := playerCount >= 2 // Minimum 2 players to start

	message := fmt.Sprintf("Waiting for host to start (%d/%d players)", playerCount, h.maxPlayers)
	if canStart {
		message = fmt.Sprintf("Ready to start (%d players). Host can type 'start' to begin.", playerCount)
	}

	waitingSync := &pkgnet.WaitingSync{
		MatchID:     "",
		HostUserID:  h.hostUserID,
		Players:     waitingPlayers,
		PlayerCount: playerCount,
		MinPlayers:  2,
		MaxPlayers:  h.maxPlayers,
		CanStart:    canStart,
		Message:     message,
	}

	// Broadcast to all connected players
	adapter := NewNakamaBroadcastAdapter(h)
	for _, userID := range h.playerList {
		if !h.disconnected[userID] {
			if err := adapter.SendWaitingSync(userID, waitingSync); err != nil {
				h.logError("broadcastWaitingSyncToAll: failed to send", "user_id", userID, "error", err)
			}
		}
	}

	h.logDebug("broadcastWaitingSyncToAll: sent to all players", "player_count", playerCount)
}
