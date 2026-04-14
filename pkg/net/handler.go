package net

import "context"

// MatchHandler defines the abstract interface for a match handler.
// Nakama implementation should implement this interface.
// This design allows testing without Nakama SDK dependency.
type MatchHandler interface {
	// ========== Lifecycle Methods ==========

	// MatchInit initializes the match with the given configuration.
	// Should create game instance, HSM, and set up initial state.
	MatchInit(ctx context.Context, config MatchConfig) error

	// MatchLoop runs the main match loop.
	// Should handle messages and advance game state.
	MatchLoop(ctx context.Context) error

	// MatchStop stops the match and performs cleanup.
	MatchStop(ctx context.Context) error

	// ========== Message Methods ==========

	// HandleMessage processes a message from a player.
	// The sender is the player's user ID.
	HandleMessage(sender string, msg Message) error

	// Broadcast sends a message to all players.
	Broadcast(opCode OpCode, data interface{}) error

	// SendToPlayer sends a message to a specific player.
	SendToPlayer(playerID string, opCode OpCode, data interface{}) error

	// ========== Presence Methods ==========

	// HandlePresenceJoin handles a player joining/rejoining the match.
	// sessionID identifies the connection session.
	HandlePresenceJoin(userID string, sessionID string) error

	// HandlePresenceLeave handles a player leaving the match.
	// If the leaving player is in their turn, timeout handling should apply.
	HandlePresenceLeave(userID string) error

	// ========== State Methods ==========

	// GetCurrentState returns the current complete game state.
	GetCurrentState() StateSync
}

// MatchConfig represents match initialization configuration.
type MatchConfig struct {
	// GameID is the unique game/match identifier.
	GameID string `json:"game_id"`

	// MaxPlayers is the maximum number of players (2-4).
	MaxPlayers int `json:"max_players"`

	// Seed is the RNG seed for reproducible games.
	// Use 0 for random seed from current time.
	Seed int64 `json:"seed"`

	// MapLength is the map total length (number of cells).
	MapLength int `json:"map_length"`
}

// DefaultMatchConfig provides default match configuration.
var DefaultMatchConfig = MatchConfig{
	MaxPlayers: 4,
	Seed:       0, // Auto-generate
	MapLength:   100,
}