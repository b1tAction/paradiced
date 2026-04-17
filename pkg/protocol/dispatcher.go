package protocol

// Dispatcher defines the message sending interface for SDK isolation.
// Used by Nakama broadcast adapter to send messages to clients.
type Dispatcher interface {
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