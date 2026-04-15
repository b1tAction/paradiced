// Package nakama provides Nakama match handler implementation for Paradiced.
package nakama

// DispatcherAdapter isolates Nakama SDK from our codebase.
// This interface allows testing without a real Nakama server.
// In production, use RealDispatcherAdapter with actual Nakama SDK.
type DispatcherAdapter interface {
	// BroadcastMessage sends a message to all players in the match.
	BroadcastMessage(opCode int64, data []byte) error

	// SendMessage sends a message to a specific player.
	SendMessage(playerID string, opCode int64, data []byte) error
}

// BroadcastRecord records a broadcast message for testing.
type BroadcastRecord struct {
	OpCode int64
	Data   []byte
}

// MessageRecord records a sent message for testing.
type MessageRecord struct {
	OpCode int64
	Data   []byte
}