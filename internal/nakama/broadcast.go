// Package nakama provides Nakama match handler implementation for Paradiced.
package nakama

import (
	"encoding/json"

	"github.com/b1tAction/paradiced/pkg/net"
)

// NakamaBroadcastAdapter implements BroadcastAdapter for Nakama match.
// Uses Nakama's message broadcast API to send messages to clients.
type NakamaBroadcastAdapter struct {
	handler *NakamaMatchHandler
}

// NewNakamaBroadcastAdapter creates a new broadcast adapter.
func NewNakamaBroadcastAdapter(handler *NakamaMatchHandler) *NakamaBroadcastAdapter {
	return &NakamaBroadcastAdapter{
		handler: handler,
	}
}

// BroadcastStateSync broadcasts state sync to all players.
// In actual Nakama implementation, this would use Nakama's broadcast API.
func (a *NakamaBroadcastAdapter) BroadcastStateSync(state *net.StateSync) error {
	// Serialize state sync
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}

	// Create message
	msg := net.MustNewMessage(net.OpStateSync, state)

	// Broadcast to all players (placeholder - actual Nakama implementation needed)
	// In production: a.handler.nakama.BroadcastMessage(a.handler.matchID, msg)
	_ = data // Placeholder for actual broadcast
	_ = msg  // Placeholder for actual broadcast

	return nil
}

// BroadcastTurnSync broadcasts turn action list to all players.
func (a *NakamaBroadcastAdapter) BroadcastTurnSync(turn *net.TurnSync) error {
	msg := net.MustNewMessage(net.OpTurnSync, turn)

	// Broadcast to all players (placeholder - actual Nakama implementation needed)
	_ = msg // Placeholder

	return nil
}

// SendDecision sends a decision request to a specific player.
func (a *NakamaBroadcastAdapter) SendDecision(playerID string, decision *net.Decision) error {
	msg := net.MustNewMessage(net.OpDecisionRequest, decision)

	// Send to specific player (placeholder - actual Nakama implementation needed)
	_ = playerID // Placeholder
	_ = msg      // Placeholder

	return nil
}

// SendAvailable sends available actions to a specific player.
func (a *NakamaBroadcastAdapter) SendAvailable(playerID string, available *net.Available) error {
	msg := net.MustNewMessage(net.OpAvailable, available)

	// Send to specific player (placeholder - actual Nakama implementation needed)
	_ = playerID // Placeholder
	_ = msg      // Placeholder

	return nil
}

// BroadcastMiniGameStart broadcasts mini-game start notification.
func (a *NakamaBroadcastAdapter) BroadcastMiniGameStart(start *net.MiniGameStart) error {
	msg := net.MustNewMessage(net.OpMiniGameStart, start)

	// Broadcast to all players (placeholder - actual Nakama implementation needed)
	_ = msg // Placeholder

	return nil
}

// BroadcastMiniGameResult broadcasts mini-game ranking results.
func (a *NakamaBroadcastAdapter) BroadcastMiniGameResult(result *net.MiniGameResult) error {
	msg := net.MustNewMessage(net.OpMiniGameResult, result)

	// Broadcast to all players (placeholder - actual Nakama implementation needed)
	_ = msg // Placeholder

	return nil
}

// BroadcastGameOver broadcasts game end notification.
func (a *NakamaBroadcastAdapter) BroadcastGameOver(over *net.GameOver) error {
	msg := net.MustNewMessage(net.OpGameOver, over)

	// Broadcast to all players (placeholder - actual Nakama implementation needed)
	_ = msg // Placeholder

	return nil
}

// SendFullSync sends complete state to a reconnecting player.
func (a *NakamaBroadcastAdapter) SendFullSync(playerID string, state *net.StateSync, turn *net.TurnSync) error {
	// Send state sync and turn sync in one message
	fullSync := map[string]interface{}{
		"state_sync": state,
		"turn_sync":  turn,
	}
	msg := net.MustNewMessage(net.OpFullSync, fullSync)

	// Send to specific player (placeholder - actual Nakama implementation needed)
	_ = playerID // Placeholder
	_ = msg      // Placeholder

	return nil
}