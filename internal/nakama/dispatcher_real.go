// Package nakama provides Nakama match handler implementation for Paradiced.
package nakama

import (
	"context"

	"github.com/heroiclabs/nakama-common/runtime"
)

// RealDispatcherAdapter implements DispatcherAdapter using real Nakama MatchDispatcher.
// This adapter is used in production with actual Nakama server.
//
// Usage in MatchLoop/MatchJoin/MatchLeave:
//
//	func (m *MyMatch) MatchLoop(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, dispatcher runtime.MatchDispatcher, tick int64, state interface{}, messages []runtime.MatchData) interface{} {
//	    realDispatcher := NewRealDispatcherAdapter(dispatcher)
//	    handler.state.(*NakamaMatchHandler).WithDispatcher(realDispatcher)
//	    ...
//	}
type RealDispatcherAdapter struct {
	dispatcher runtime.MatchDispatcher
	ctx        context.Context

	// userID -> Presence mapping for sending individual messages
	userPresences map[string]runtime.Presence
}

// NewRealDispatcherAdapter creates a new real dispatcher with Nakama MatchDispatcher.
func NewRealDispatcherAdapter(ctx context.Context, dispatcher runtime.MatchDispatcher, presences []runtime.Presence) *RealDispatcherAdapter {
	// Build presence map
	userPresences := make(map[string]runtime.Presence)
	for _, p := range presences {
		userPresences[p.GetUserId()] = p
	}

	return &RealDispatcherAdapter{
		dispatcher:    dispatcher,
		ctx:           ctx,
		userPresences: userPresences,
	}
}

// BroadcastMessage broadcasts a message to all players in the match.
func (d *RealDispatcherAdapter) BroadcastMessage(opCode int64, data []byte) error {
	// Broadcast to all presences (nil = all), no sender, reliable delivery
	return d.dispatcher.BroadcastMessage(opCode, data, nil, nil, true)
}

// SendMessage sends a message to a specific player.
func (d *RealDispatcherAdapter) SendMessage(playerID string, opCode int64, data []byte) error {
	presence, ok := d.userPresences[playerID]
	if !ok {
		// Player not found in presences
		return nil // Silently ignore - player may have disconnected
	}

	// Send to specific presence
	return d.dispatcher.BroadcastMessage(opCode, data, []runtime.Presence{presence}, nil, true)
}

// UpdatePresence updates the presence map when players join.
func (d *RealDispatcherAdapter) UpdatePresence(presence runtime.Presence) {
	d.userPresences[presence.GetUserId()] = presence
}

// RemovePresence removes a presence when player leaves.
func (d *RealDispatcherAdapter) RemovePresence(userID string) {
	delete(d.userPresences, userID)
}

// RefreshPresences rebuilds the presence map from new presences.
// Call this after join/leave events to update the internal mapping.
func (d *RealDispatcherAdapter) RefreshPresences(presences []runtime.Presence) {
	d.userPresences = make(map[string]runtime.Presence)
	for _, p := range presences {
		d.userPresences[p.GetUserId()] = p
	}
}

// GetPresence returns the Presence for a given userID.
func (d *RealDispatcherAdapter) GetPresence(userID string) runtime.Presence {
	return d.userPresences[userID]
}

// GetAllPresences returns all current presences.
func (d *RealDispatcherAdapter) GetAllPresences() []runtime.Presence {
	result := make([]runtime.Presence, 0, len(d.userPresences))
	for _, p := range d.userPresences {
		result = append(result, p)
	}
	return result
}