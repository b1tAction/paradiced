// Package nakama provides Nakama match handler implementation for Paradiced.
package nakama

import (
	"encoding/json"
	"sync"
)

// MockDispatcherAdapter is a test implementation of DispatcherAdapter.
// It captures all messages for verification in tests.
type MockDispatcherAdapter struct {
	mu sync.RWMutex

	// Broadcasts captures all BroadcastMessage calls.
	Broadcasts []BroadcastRecord

	// Messages captures all SendMessage calls (keyed by playerID).
	Messages map[string][]MessageRecord
}

// NewMockDispatcherAdapter creates a new mock dispatcher.
func NewMockDispatcherAdapter() *MockDispatcherAdapter {
	return &MockDispatcherAdapter{
		Broadcasts: make([]BroadcastRecord, 0),
		Messages:   make(map[string][]MessageRecord),
	}
}

// BroadcastMessage captures broadcast for testing.
func (m *MockDispatcherAdapter) BroadcastMessage(opCode int64, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.Broadcasts = append(m.Broadcasts, BroadcastRecord{
		OpCode: opCode,
		Data:   data,
	})

	return nil
}

// SendMessage captures message for testing.
func (m *MockDispatcherAdapter) SendMessage(playerID string, opCode int64, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.Messages[playerID] == nil {
		m.Messages[playerID] = make([]MessageRecord, 0)
	}

	m.Messages[playerID] = append(m.Messages[playerID], MessageRecord{
		OpCode: opCode,
		Data:   data,
	})

	return nil
}

// GetBroadcasts returns all captured broadcasts.
func (m *MockDispatcherAdapter) GetBroadcasts() []BroadcastRecord {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]BroadcastRecord, len(m.Broadcasts))
	copy(result, m.Broadcasts)
	return result
}

// GetMessages returns all captured messages for a player.
func (m *MockDispatcherAdapter) GetMessages(playerID string) []MessageRecord {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.Messages[playerID] == nil {
		return nil
	}

	result := make([]MessageRecord, len(m.Messages[playerID]))
	copy(result, m.Messages[playerID])
	return result
}

// Clear resets all captured messages.
func (m *MockDispatcherAdapter) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.Broadcasts = make([]BroadcastRecord, 0)
	m.Messages = make(map[string][]MessageRecord)
}

// ParseBroadcastData parses the data of a broadcast record.
func (m *MockDispatcherAdapter) ParseBroadcastData(index int, target interface{}) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if index < 0 || index >= len(m.Broadcasts) {
		return nil
	}

	return json.Unmarshal(m.Broadcasts[index].Data, target)
}

// ParseMessageData parses the data of a message record for a player.
func (m *MockDispatcherAdapter) ParseMessageData(playerID string, index int, target interface{}) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.Messages[playerID] == nil {
		return nil
	}

	if index < 0 || index >= len(m.Messages[playerID]) {
		return nil
	}

	return json.Unmarshal(m.Messages[playerID][index].Data, target)
}

// CountBroadcasts returns the number of captured broadcasts.
func (m *MockDispatcherAdapter) CountBroadcasts() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.Broadcasts)
}

// CountMessages returns the number of captured messages for a player.
func (m *MockDispatcherAdapter) CountMessages(playerID string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.Messages[playerID] == nil {
		return 0
	}

	return len(m.Messages[playerID])
}