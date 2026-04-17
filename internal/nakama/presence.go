// Package nakama provides Nakama match handler implementation for Paradiced.
package nakama

import (
	"fmt"

	"github.com/b1tAction/paradiced/pkg/constants"
	"github.com/b1tAction/paradiced/pkg/net"
)

// HandlePresenceJoin handles a player joining the match.
// Called by Nakama when a new player joins the match.
func (h *NakamaMatchHandler) HandlePresenceJoin(userID string, metadata map[string]string) error {
	// Check if match is full
	if len(h.players) >= h.maxPlayers {
		return fmt.Errorf("match is full")
	}

	// Check if player already exists
	if h.players[userID] != nil {
		return nil // Player already in match
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

	// Add player to match
	h.addPlayer(userID, faction)

	// If match is now full, start the game
	if len(h.players) == h.maxPlayers {
		// Assign factions based on join order
		h.assignFactions()

		// Initialize game
		if err := h.MatchInit(); err != nil {
			return err
		}
	}

	return nil
}

// HandlePresenceLeave handles a player leaving the match.
// Called by Nakama when a player leaves/disconnects.
func (h *NakamaMatchHandler) HandlePresenceLeave(userID string) error {
	// Get player
	player := h.players[userID]
	if player == nil {
		return nil // Player not in match
	}

	// Remove player from tracking
	delete(h.players, userID)

	// Remove from player list
	for i, id := range h.playerList {
		if id == userID {
			h.playerList = append(h.playerList[:i], h.playerList[i+1:]...)
			break
		}
	}

	// If game is running, handle player disconnect
	if h.hsm != nil && h.hsm.IsRunning() {
		// Mark player as disconnected (but don't remove from game)
		// Player will be skipped in turn order

		// Broadcast player disconnect to others
		broadcast := NewNakamaBroadcastAdapter(h)
		stateSync := &net.StateSync{
			GlobalState: h.game.State.CurrentPhase,
			Round:       h.game.State.Round,
			Turn:        h.game.State.Turn,
			// Update players list
		}
		broadcast.BroadcastStateSync(stateSync)
	}

	// If all players left, stop the match
	if len(h.players) == 0 {
		h.MatchStop()
	}

	return nil
}