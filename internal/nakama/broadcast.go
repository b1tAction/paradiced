// Package nakama provides Nakama match handler implementation for Paradiced.
package nakama

import (
	"encoding/json"

	"github.com/b1tAction/paradiced/pkg/net"
)

// NakamaBroadcastAdapter implements BroadcastAdapter for Nakama match.
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
func (a *NakamaBroadcastAdapter) BroadcastStateSync(state interface{}) error {
	if a.handler.dispatcher == nil {
		return nil // No dispatcher set
	}

	// Type assert to concrete type for JSON serialization
	stateSync, ok := state.(*net.StateSync)
	if !ok {
		return nil // Invalid type, skip
	}

	data, err := json.Marshal(stateSync)
	if err != nil {
		return err
	}

	return a.handler.dispatcher.BroadcastMessage(int64(net.OpStateSync), data)
}

// BroadcastTurnSync broadcasts turn action list to all players.
func (a *NakamaBroadcastAdapter) BroadcastTurnSync(turn interface{}) error {
	if a.handler.dispatcher == nil {
		return nil // No dispatcher set
	}

	turnSync, ok := turn.(*net.TurnSync)
	if !ok {
		return nil // Invalid type, skip
	}

	data, err := json.Marshal(turnSync)
	if err != nil {
		return err
	}

	return a.handler.dispatcher.BroadcastMessage(int64(net.OpTurnSync), data)
}

// SendDecision sends a decision request to a specific player.
func (a *NakamaBroadcastAdapter) SendDecision(playerID string, decision interface{}) error {
	if a.handler.dispatcher == nil {
		return nil // No dispatcher set
	}

	decisionReq, ok := decision.(*net.Decision)
	if !ok {
		return nil // Invalid type, skip
	}

	data, err := json.Marshal(decisionReq)
	if err != nil {
		return err
	}

	return a.handler.dispatcher.SendMessage(playerID, int64(net.OpDecisionRequest), data)
}

// SendAvailable sends available actions to a specific player.
func (a *NakamaBroadcastAdapter) SendAvailable(playerID string, available interface{}) error {
	if a.handler.dispatcher == nil {
		return nil // No dispatcher set
	}

	avail, ok := available.(*net.Available)
	if !ok {
		return nil // Invalid type, skip
	}

	data, err := json.Marshal(avail)
	if err != nil {
		return err
	}

	return a.handler.dispatcher.SendMessage(playerID, int64(net.OpAvailable), data)
}

// BroadcastMiniGameStart broadcasts mini-game start notification.
func (a *NakamaBroadcastAdapter) BroadcastMiniGameStart(start interface{}) error {
	if a.handler.dispatcher == nil {
		return nil // No dispatcher set
	}

	miniStart, ok := start.(*net.MiniGameStart)
	if !ok {
		return nil // Invalid type, skip
	}

	data, err := json.Marshal(miniStart)
	if err != nil {
		return err
	}

	return a.handler.dispatcher.BroadcastMessage(int64(net.OpMiniGameStart), data)
}

// BroadcastMiniGameResult broadcasts mini-game ranking results.
func (a *NakamaBroadcastAdapter) BroadcastMiniGameResult(result interface{}) error {
	if a.handler.dispatcher == nil {
		return nil // No dispatcher set
	}

	miniResult, ok := result.(*net.MiniGameResult)
	if !ok {
		return nil // Invalid type, skip
	}

	data, err := json.Marshal(miniResult)
	if err != nil {
		return err
	}

	return a.handler.dispatcher.BroadcastMessage(int64(net.OpMiniGameResult), data)
}

// BroadcastGameOver broadcasts game end notification.
func (a *NakamaBroadcastAdapter) BroadcastGameOver(over interface{}) error {
	if a.handler.dispatcher == nil {
		return nil // No dispatcher set
	}

	gameOver, ok := over.(*net.GameOver)
	if !ok {
		return nil // Invalid type, skip
	}

	data, err := json.Marshal(gameOver)
	if err != nil {
		return err
	}

	return a.handler.dispatcher.BroadcastMessage(int64(net.OpGameOver), data)
}

// SendFullSync sends complete state to a reconnecting player.
func (a *NakamaBroadcastAdapter) SendFullSync(playerID string, state, turn interface{}) error {
	if a.handler.dispatcher == nil {
		return nil // No dispatcher set
	}

	stateSync, ok := state.(*net.StateSync)
	if !ok {
		return nil // Invalid state type, skip
	}

	turnSync, ok := turn.(*net.TurnSync)
	if !ok {
		return nil // Invalid turn type, skip
	}

	// Send state sync and turn sync in one message
	fullSync := map[string]interface{}{
		"state_sync": stateSync,
		"turn_sync":  turnSync,
	}

	data, err := json.Marshal(fullSync)
	if err != nil {
		return err
	}

	return a.handler.dispatcher.SendMessage(playerID, int64(net.OpFullSync), data)
}