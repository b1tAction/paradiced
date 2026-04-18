// Package nakama provides Nakama match handler implementation for Paradiced.
package nakama

import (
	"context"
)

// NakamaPresence represents a player presence in Nakama match.
// This is a local stub that matches runtime.MatchPresence interface.
// When deployed to Nakama server, use the real runtime.MatchPresence type.
type NakamaPresence interface {
	// GetUserId returns the user ID of this presence.
	GetUserId() string

	// GetSessionId returns the session ID of this presence.
	GetSessionId() string

	// GetNodeId returns the node ID where this session is connected.
	GetNodeId() string
}

// NakamaMatchWrapper isolates Nakama Match interface functions needed for dispatching.
// This interface captures the essential methods from Nakama's runtime.Match interface.
// When deployed to Nakama server, implement this using real Match functions.
type NakamaMatchWrapper interface {
	// BroadcastData broadcasts message to all or specific presences.
	// If presences is nil, broadcasts to all match members.
	BroadcastData(opCode int64, data []byte, presences []NakamaPresence, reliability int) error

	// SendData sends message to specific presences.
	SendData(opCode int64, data []byte, presences []NakamaPresence, reliability int) error

	// GetPresences returns current match presences.
	GetPresences() []NakamaPresence
}

// Reliability constants for Nakama message delivery.
const (
	Reliable   = 0 // Reliable delivery (TCP-like)
	Unreliable = 1 // Unreliable delivery (UDP-like)
)

// RealDispatcherAdapter implements DispatcherAdapter using real Nakama Match functions.
// This adapter is used in production with actual Nakama server.
//
// Usage:
//
//	func (m *MyMatch) MatchInit(ctx context.Context, logger runtime.Logger, match runtime.Match) error {
//	    dispatcher := NewRealDispatcherAdapter(ctx, match)
//	    handler := NewNakamaMatchHandler(matchId, seed, maxPlayers, mapLength)
//	    handler.WithDispatcher(dispatcher)
//	    ...
//	}
type RealDispatcherAdapter struct {
	ctx   context.Context
	match NakamaMatchWrapper

	// userID -> Presence mapping for sending individual messages
	userPresences map[string]NakamaPresence
}

// NewRealDispatcherAdapter creates a new real dispatcher with Nakama match context.
func NewRealDispatcherAdapter(ctx context.Context, match NakamaMatchWrapper) *RealDispatcherAdapter {
	// Build presence map
	userPresences := make(map[string]NakamaPresence)
	for _, p := range match.GetPresences() {
		userPresences[p.GetUserId()] = p
	}

	return &RealDispatcherAdapter{
		ctx:           ctx,
		match:         match,
		userPresences: userPresences,
	}
}

// BroadcastMessage broadcasts a message to all players in the match.
func (d *RealDispatcherAdapter) BroadcastMessage(opCode int64, data []byte) error {
	// Broadcast to all presences (nil = all)
	return d.match.BroadcastData(opCode, data, nil, Reliable)
}

// SendMessage sends a message to a specific player.
func (d *RealDispatcherAdapter) SendMessage(playerID string, opCode int64, data []byte) error {
	presence, ok := d.userPresences[playerID]
	if !ok {
		// Player not found in presences
		return nil // Silently ignore - player may have disconnected
	}

	// Send to specific presence
	return d.match.SendData(opCode, data, []NakamaPresence{presence}, Reliable)
}

// UpdatePresence updates the presence map when players join.
func (d *RealDispatcherAdapter) UpdatePresence(userID string, presence NakamaPresence) {
	d.userPresences[userID] = presence
}

// RemovePresence removes a presence when player leaves.
func (d *RealDispatcherAdapter) RemovePresence(userID string) {
	delete(d.userPresences, userID)
}

// RefreshPresences rebuilds the presence map from current match presences.
// Call this after join/leave events to update the internal mapping.
func (d *RealDispatcherAdapter) RefreshPresences() {
	d.userPresences = make(map[string]NakamaPresence)
	for _, p := range d.match.GetPresences() {
		d.userPresences[p.GetUserId()] = p
	}
}