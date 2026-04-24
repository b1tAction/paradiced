// Package nakama provides Nakama match handler implementation for Paradiced.
package nakama

import (
	"encoding/json"

	"github.com/b1tAction/paradiced/pkg/net"
)

// NakamaBroadcastAdapter implements pkg/net.BroadcastAdapter for Nakama match.
// Uses DispatcherAdapter to send messages to clients.
type NakamaBroadcastAdapter struct {
	handler *NakamaMatchHandler
}

// NewNakamaBroadcastAdapter creates a new broadcast adapter.
func NewNakamaBroadcastAdapter(handler *NakamaMatchHandler) *NakamaBroadcastAdapter {
	return &NakamaBroadcastAdapter{
		handler: handler,
	}
}

// resolveRecipientUserID maps an input identifier to Nakama userID.
// The input may already be a userID, or an internal player ID (core.Player.ID).
func (a *NakamaBroadcastAdapter) resolveRecipientUserID(playerID string) string {
	if a == nil || a.handler == nil {
		return playerID
	}

	// Already a known Nakama userID.
	if _, ok := a.handler.players[playerID]; ok {
		return playerID
	}

	// Try mapping internal player ID -> userID.
	for userID, p := range a.handler.players {
		if p != nil && p.ID.UUID() == playerID {
			return userID
		}
	}

	// Fallback to input for tests/mocks.
	return playerID
}

// BroadcastStateSync broadcasts state sync to all players.
func (a *NakamaBroadcastAdapter) BroadcastStateSync(state *net.StateSync) error {
	if a.handler.dispatcher == nil {
		return nil // No dispatcher set
	}

	// Inject DisplayName into each Player for client-side UI rendering.
	// PlayerID already equals Nakama userID, so clients can self-identify by matching PlayerID.
	if state != nil && a.handler != nil {
		for i := range state.Players {
			// Boss player has fixed display name
			if state.Players[i].IsBoss {
				state.Players[i].DisplayName = "Boss"
				continue
			}
			// Find Nakama userID for this player by matching PlayerID
			for userID, player := range a.handler.players {
				if player != nil && player.ID.UUID() == state.Players[i].PlayerID {
					// Extract display_name from Player.Metadata, fallback to userID (which equals PlayerID)
					state.Players[i].DisplayName = player.Metadata.GetStringOrDefault("display_name", userID)
					break
				}
			}
		}
	}

	data, err := json.Marshal(state)
	if err != nil {
		return err
	}

	return a.handler.dispatcher.BroadcastMessage(int64(net.OpStateSync), data)
}

// BroadcastTurnSync broadcasts turn action list to all players.
func (a *NakamaBroadcastAdapter) BroadcastTurnSync(turn *net.TurnSync) error {
	if a.handler.dispatcher == nil {
		return nil // No dispatcher set
	}

	data, err := json.Marshal(turn)
	if err != nil {
		return err
	}

	return a.handler.dispatcher.BroadcastMessage(int64(net.OpTurnSync), data)
}

// SendDecision sends a decision request to a specific player.
func (a *NakamaBroadcastAdapter) SendDecision(playerID string, decision *net.Decision) error {
	if a.handler.dispatcher == nil {
		return nil // No dispatcher set
	}

	userID := a.resolveRecipientUserID(playerID)

	data, err := json.Marshal(decision)
	if err != nil {
		return err
	}

	return a.handler.dispatcher.SendMessage(userID, int64(net.OpDecisionRequest), data)
}

// SendAvailable sends available actions to a specific player.
func (a *NakamaBroadcastAdapter) SendAvailable(playerID string, available *net.Available) error {
	if a.handler.dispatcher == nil {
		return nil // No dispatcher set
	}

	userID := a.resolveRecipientUserID(playerID)

	data, err := json.Marshal(available)
	if err != nil {
		return err
	}

	return a.handler.dispatcher.SendMessage(userID, int64(net.OpAvailable), data)
}

// BroadcastMiniGameStart broadcasts mini-game start notification.
func (a *NakamaBroadcastAdapter) BroadcastMiniGameStart(start *net.MiniGameStart) error {
	if a.handler.dispatcher == nil {
		a.handler.logDebug("BroadcastMiniGameStart: no dispatcher set")
		return nil // No dispatcher set
	}

	a.handler.logInfo("BroadcastMiniGameStart: broadcasting mini-game start",
		"game_type", start.GameType,
		"players_count", len(start.Players))

	// PlayerID already equals Nakama userID, no conversion needed.
	// The Players array already contains PlayerIDs which clients can match against.

	data, err := json.Marshal(start)
	if err != nil {
		a.handler.logError("BroadcastMiniGameStart: failed to marshal", "error", err)
		return err
	}

	a.handler.logDebug("BroadcastMiniGameStart: sending broadcast", "op_code", net.OpMiniGameStart)
	return a.handler.dispatcher.BroadcastMessage(int64(net.OpMiniGameStart), data)
}

// BroadcastMiniGameResult broadcasts mini-game ranking results.
func (a *NakamaBroadcastAdapter) BroadcastMiniGameResult(result *net.MiniGameResult) error {
	if a.handler.dispatcher == nil {
		return nil // No dispatcher set
	}

	// Inject DisplayName for UI rendering.
	// PlayerID already equals Nakama userID, no need for conversion.
	if result != nil && a.handler != nil {
		for i := range result.Rankings {
			for userID, player := range a.handler.players {
				if player != nil && player.ID.UUID() == result.Rankings[i].PlayerID {
					result.Rankings[i].DisplayName = player.Metadata.GetStringOrDefault("display_name", userID)
					break
				}
			}
		}
	}

	data, err := json.Marshal(result)
	if err != nil {
		return err
	}

	return a.handler.dispatcher.BroadcastMessage(int64(net.OpMiniGameResult), data)
}

// BroadcastGameOver broadcasts game end notification.
func (a *NakamaBroadcastAdapter) BroadcastGameOver(over *net.GameOver) error {
	if a.handler.dispatcher == nil {
		return nil // No dispatcher set
	}

	// Inject DisplayName for UI rendering.
	// PlayerID already equals Nakama userID, no need for conversion.
	if over != nil && a.handler != nil {
		// Inject DisplayName for WinnerID
		for userID, player := range a.handler.players {
			if player != nil && player.ID.UUID() == over.WinnerID {
				over.WinnerID = player.Metadata.GetStringOrDefault("display_name", userID)
				break
			}
		}

		// Inject DisplayName for Stats
		for i := range over.Stats {
			for userID, player := range a.handler.players {
				if player != nil && player.ID.UUID() == over.Stats[i].PlayerID {
					over.Stats[i].PlayerID = player.Metadata.GetStringOrDefault("display_name", userID)
					break
				}
			}
		}
	}

	data, err := json.Marshal(over)
	if err != nil {
		return err
	}

	return a.handler.dispatcher.BroadcastMessage(int64(net.OpGameOver), data)
}

// SendFullSync sends complete state to a reconnecting player.
func (a *NakamaBroadcastAdapter) SendFullSync(playerID string, state *net.StateSync, turn *net.TurnSync) error {
	if a.handler.dispatcher == nil {
		return nil // No dispatcher set
	}

	userID := a.resolveRecipientUserID(playerID)

	// Send state sync and turn sync in one message
	fullSync := map[string]interface{}{
		"state_sync": state,
		"turn_sync":  turn,
	}

	data, err := json.Marshal(fullSync)
	if err != nil {
		return err
	}

	return a.handler.dispatcher.SendMessage(userID, int64(net.OpFullSync), data)
}

// SendActionRejected sends action rejection notification to a specific player.
func (a *NakamaBroadcastAdapter) SendActionRejected(playerID string, rejected *net.ActionRejected) error {
	if a.handler.dispatcher == nil {
		return nil // No dispatcher set
	}

	userID := a.resolveRecipientUserID(playerID)

	data, err := json.Marshal(rejected)
	if err != nil {
		return err
	}

	return a.handler.dispatcher.SendMessage(userID, int64(net.OpActionRejected), data)
}

// SendWaitingSync sends waiting room status to the host.
func (a *NakamaBroadcastAdapter) SendWaitingSync(userID string, waiting *net.WaitingSync) error {
	if a.handler.dispatcher == nil {
		return nil // No dispatcher set
	}

	data, err := json.Marshal(waiting)
	if err != nil {
		return err
	}

	return a.handler.dispatcher.SendMessage(userID, int64(net.OpWaitingSync), data)
}

// BroadcastStartGameAck broadcasts game start acknowledgment with map config.
func (a *NakamaBroadcastAdapter) BroadcastStartGameAck(ack *net.StartGameAck) error {
	if a.handler.dispatcher == nil {
		return nil // No dispatcher set
	}

	data, err := json.Marshal(ack)
	if err != nil {
		return err
	}

	return a.handler.dispatcher.BroadcastMessage(int64(net.OpStartGameAck), data)
}
