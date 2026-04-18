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

// BroadcastStateSync broadcasts state sync to all players.
func (a *NakamaBroadcastAdapter) BroadcastStateSync(state *net.StateSync) error {
	if a.handler.dispatcher == nil {
		return nil // No dispatcher set
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

	data, err := json.Marshal(decision)
	if err != nil {
		return err
	}

	return a.handler.dispatcher.SendMessage(playerID, int64(net.OpDecisionRequest), data)
}

// SendAvailable sends available actions to a specific player.
func (a *NakamaBroadcastAdapter) SendAvailable(playerID string, available *net.Available) error {
	if a.handler.dispatcher == nil {
		return nil // No dispatcher set
	}

	data, err := json.Marshal(available)
	if err != nil {
		return err
	}

	return a.handler.dispatcher.SendMessage(playerID, int64(net.OpAvailable), data)
}

// BroadcastMiniGameStart broadcasts mini-game start notification.
func (a *NakamaBroadcastAdapter) BroadcastMiniGameStart(start *net.MiniGameStart) error {
	if a.handler.dispatcher == nil {
		return nil // No dispatcher set
	}

	data, err := json.Marshal(start)
	if err != nil {
		return err
	}

	return a.handler.dispatcher.BroadcastMessage(int64(net.OpMiniGameStart), data)
}

// BroadcastMiniGameResult broadcasts mini-game ranking results.
func (a *NakamaBroadcastAdapter) BroadcastMiniGameResult(result *net.MiniGameResult) error {
	if a.handler.dispatcher == nil {
		return nil // No dispatcher set
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

	// Send state sync and turn sync in one message
	fullSync := map[string]interface{}{
		"state_sync": state,
		"turn_sync":  turn,
	}

	data, err := json.Marshal(fullSync)
	if err != nil {
		return err
	}

	return a.handler.dispatcher.SendMessage(playerID, int64(net.OpFullSync), data)
}