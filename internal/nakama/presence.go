// Package nakama provides Nakama match handler implementation for Paradiced.
package nakama

import (
	"fmt"

	"github.com/b1tAction/paradiced/internal/net"
	"github.com/b1tAction/paradiced/pkg/constants"
)

// HandlePresenceJoin handles a player joining the match.
// Called by Nakama when a new player joins the match.
func (h *NakamaMatchHandler) HandlePresenceJoin(userID string, metadata map[string]string) error {
	// Check if player already exists (rejoin case)
	if h.players[userID] != nil {
		// Player was previously in match
		if h.disconnected[userID] {
			// Player reconnecting - send full sync
			h.disconnected[userID] = false
			if h.hsm != nil && h.hsm.IsRunning() {
				return h.handlePlayerRejoin(userID)
			}
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
	faction := constants.FactionQingLong // Default
	if factionStr, ok := metadata["faction"]; ok {
		switch factionStr {
		case "qing_long":
			faction = constants.FactionQingLong
		case "zhu_que":
			faction = constants.FactionZhuQue
		case "bai_hu":
			faction = constants.FactionBaiHu
		case "xuan_wu":
			faction = constants.FactionXuanWu
		}
	}

	// Add player to match (faction set via PlayerConfig)
	h.addPlayer(userID, faction)

	// If match is now full, start the game
	if len(h.playerList) == h.maxPlayers {
		// Initialize game (factions already set, buff initialization happens in MatchInitState.Enter())
		if err := h.MatchInit(); err != nil {
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
	return broadcastAdapter.SendFullSync(userID, stateSync, turnSync)
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