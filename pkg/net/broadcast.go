// Package net provides network message protocol definitions for client-server communication.
package net

// BroadcastAdapter defines the interface for broadcasting messages to clients.
// HSM and ActionContext use this interface to send sync messages.
// Nakama implementation will implement this interface.
type BroadcastAdapter interface {
	// BroadcastStateSync broadcasts state sync to all players.
	// Called when HSM transitions to a new state.
	BroadcastStateSync(state *StateSync) error

	// BroadcastTurnSync broadcasts turn action list to all players.
	// Called after executing turn effects.
	BroadcastTurnSync(turn *TurnSync) error

	// SendDecision sends a decision request to a specific player.
	// Called when HSM enters WaitDecision state.
	SendDecision(playerID string, decision *Decision) error

	// SendAvailable sends available actions to a specific player.
	// Called when entering MainAction state.
	SendAvailable(playerID string, available *Available) error

	// BroadcastMiniGameStart broadcasts mini-game start notification.
	// Called when entering RoundMiniGame state.
	BroadcastMiniGameStart(start *MiniGameStart) error

	// BroadcastMiniGameResult broadcasts mini-game ranking results.
	// Called after mini-game completes.
	BroadcastMiniGameResult(result *MiniGameResult) error

	// BroadcastGameOver broadcasts game end notification.
	// Called when entering GameOver state.
	BroadcastGameOver(over *GameOver) error

	// SendFullSync sends complete state to a reconnecting player.
	// Called when a player rejoins the match.
	SendFullSync(playerID string, state *StateSync, turn *TurnSync) error

	// BroadcastStartGameAck broadcasts game start acknowledgment with map config.
	// Called when host starts the game (after OpStartGame received).
	BroadcastStartGameAck(ack *StartGameAck) error
}

// MockBroadcastAdapter is a test implementation that captures messages.
// Use in tests to verify broadcast behavior.
type MockBroadcastAdapter struct {
	// StateSyncs captures all BroadcastStateSync calls.
	StateSyncs []*StateSync

	// TurnSyncs captures all BroadcastTurnSync calls.
	TurnSyncs []*TurnSync

	// Decisions captures SendDecision calls (keyed by playerID).
	Decisions map[string]*Decision

	// Availables captures SendAvailable calls (keyed by playerID).
	Availables map[string]*Available

	// MiniGameStarts captures BroadcastMiniGameStart calls.
	MiniGameStarts []*MiniGameStart

	// MiniGameResults captures BroadcastMiniGameResult calls.
	MiniGameResults []*MiniGameResult

	// GameOvers captures BroadcastGameOver calls.
	GameOvers []*GameOver

	// FullSyncs captures SendFullSync calls (keyed by playerID).
	FullSyncs map[string]struct {
		State *StateSync
		Turn  *TurnSync
	}

	// StartGameAcks captures BroadcastStartGameAck calls.
	StartGameAcks []*StartGameAck
}

// NewMockBroadcastAdapter creates a new mock adapter.
func NewMockBroadcastAdapter() *MockBroadcastAdapter {
	return &MockBroadcastAdapter{
		StateSyncs:      make([]*StateSync, 0),
		TurnSyncs:       make([]*TurnSync, 0),
		Decisions:       make(map[string]*Decision),
		Availables:      make(map[string]*Available),
		MiniGameStarts:  make([]*MiniGameStart, 0),
		MiniGameResults: make([]*MiniGameResult, 0),
		GameOvers:       make([]*GameOver, 0),
		FullSyncs:       make(map[string]struct {
			State *StateSync
			Turn  *TurnSync
		}),
		StartGameAcks:  make([]*StartGameAck, 0),
	}
}

// BroadcastStateSync captures state sync.
func (m *MockBroadcastAdapter) BroadcastStateSync(state *StateSync) error {
	m.StateSyncs = append(m.StateSyncs, state)
	return nil
}

// BroadcastTurnSync captures turn sync.
func (m *MockBroadcastAdapter) BroadcastTurnSync(turn *TurnSync) error {
	m.TurnSyncs = append(m.TurnSyncs, turn)
	return nil
}

// SendDecision captures decision request.
func (m *MockBroadcastAdapter) SendDecision(playerID string, decision *Decision) error {
	m.Decisions[playerID] = decision
	return nil
}

// SendAvailable captures available actions.
func (m *MockBroadcastAdapter) SendAvailable(playerID string, available *Available) error {
	m.Availables[playerID] = available
	return nil
}

// BroadcastMiniGameStart captures mini-game start.
func (m *MockBroadcastAdapter) BroadcastMiniGameStart(start *MiniGameStart) error {
	m.MiniGameStarts = append(m.MiniGameStarts, start)
	return nil
}

// BroadcastMiniGameResult captures mini-game result.
func (m *MockBroadcastAdapter) BroadcastMiniGameResult(result *MiniGameResult) error {
	m.MiniGameResults = append(m.MiniGameResults, result)
	return nil
}

// BroadcastGameOver captures game over.
func (m *MockBroadcastAdapter) BroadcastGameOver(over *GameOver) error {
	m.GameOvers = append(m.GameOvers, over)
	return nil
}

// SendFullSync captures full sync for reconnect.
func (m *MockBroadcastAdapter) SendFullSync(playerID string, state *StateSync, turn *TurnSync) error {
	m.FullSyncs[playerID] = struct {
		State *StateSync
		Turn  *TurnSync
	}{State: state, Turn: turn}
	return nil
}

// BroadcastStartGameAck captures start game ack.
func (m *MockBroadcastAdapter) BroadcastStartGameAck(ack *StartGameAck) error {
	m.StartGameAcks = append(m.StartGameAcks, ack)
	return nil
}

// Clear resets all captured messages.
func (m *MockBroadcastAdapter) Clear() {
	m.StateSyncs = make([]*StateSync, 0)
	m.TurnSyncs = make([]*TurnSync, 0)
	m.Decisions = make(map[string]*Decision)
	m.Availables = make(map[string]*Available)
	m.MiniGameStarts = make([]*MiniGameStart, 0)
	m.MiniGameResults = make([]*MiniGameResult, 0)
	m.GameOvers = make([]*GameOver, 0)
	m.FullSyncs = make(map[string]struct {
		State *StateSync
		Turn  *TurnSync
	})
	m.StartGameAcks = make([]*StartGameAck, 0)
}