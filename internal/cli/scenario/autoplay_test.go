// Package scenario provides automated play scenarios for testing.
package scenario

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/b1tAction/paradiced/internal/cli/model"
	"github.com/b1tAction/paradiced/internal/cli/nakama"
)

// ========== Mock SocketClient ==========

// MockSocketClient implements SocketClientAdapter for testing.
type MockSocketClient struct {
	msgChan    chan *nakama.SocketMessage
	sendMu     sync.Mutex
	sentMsgs   []struct {
		opCode int64
		data   any
	}
	closed bool
}

// NewMockSocketClient creates a new mock socket client.
func NewMockSocketClient() *MockSocketClient {
	return &MockSocketClient{
		msgChan:  make(chan *nakama.SocketMessage, 100),
		sentMsgs: make([]struct {
			opCode int64
			data   any
		}, 0),
	}
}

// MessageChan returns the message channel.
func (m *MockSocketClient) MessageChan() <-chan *nakama.SocketMessage {
	return m.msgChan
}

// SendMessage records the message for testing.
func (m *MockSocketClient) SendMessage(ctx context.Context, opCode int64, data any) error {
	m.sendMu.Lock()
	defer m.sendMu.Unlock()
	m.sentMsgs = append(m.sentMsgs, struct {
		opCode int64
		data   any
	}{opCode, data})
	return nil
}

// Close closes the mock client.
func (m *MockSocketClient) Close() error {
	m.closed = true
	close(m.msgChan)
	return nil
}

// SimulateMessage simulates receiving a message from server.
func (m *MockSocketClient) SimulateMessage(opCode int64, data []byte) {
	m.msgChan <- &nakama.SocketMessage{
		OpCode: opCode,
		Data:   data,
	}
}

// SimulateJSONMessage simulates receiving a JSON message.
func (m *MockSocketClient) SimulateJSONMessage(opCode int64, v any) {
	data, _ := json.Marshal(v)
	m.SimulateMessage(opCode, data)
}

// GetSentMessages returns all sent messages.
func (m *MockSocketClient) GetSentMessages() []struct {
	opCode int64
	data   any
} {
	m.sendMu.Lock()
	defer m.sendMu.Unlock()
	return m.sentMsgs
}

// ClearSentMessages clears sent messages history.
func (m *MockSocketClient) ClearSentMessages() {
	m.sendMu.Lock()
	defer m.sendMu.Unlock()
	m.sentMsgs = make([]struct {
		opCode int64
		data   any
	}, 0)
}

// ========== AutoPlayPlayer Creation Tests ==========

func TestNewAutoPlayPlayer(t *testing.T) {
	mockSocket := NewMockSocketClient()
	logger := nakama.NewLogger(false)

	player := NewAutoPlayPlayerStandalone(mockSocket, "test-user-id", logger)

	if player == nil {
		t.Fatal("NewAutoPlayPlayerStandalone should return non-nil")
	}
	if player.userID != "test-user-id" {
		t.Errorf("userID = %s, expected test-user-id", player.userID)
	}
	if player.socket == nil {
		t.Error("socket should be set")
	}
	if player.gameOverChan == nil {
		t.Error("gameOverChan should be initialized")
	}
}

// ========== State Accessor Tests ==========

func TestAutoPlayPlayerGameOverChan(t *testing.T) {
	mockSocket := NewMockSocketClient()
	logger := nakama.NewLogger(false)
	player := NewAutoPlayPlayerStandalone(mockSocket, "user-001", logger)

	ch := player.GameOverChan()
	if ch == nil {
		t.Error("GameOverChan should return non-nil channel")
	}
}

func TestAutoPlayPlayerMessagesReceived(t *testing.T) {
	mockSocket := NewMockSocketClient()
	logger := nakama.NewLogger(false)
	player := NewAutoPlayPlayerStandalone(mockSocket, "user-001", logger)

	// Initial count should be 0
	if player.MessagesReceived() != 0 {
		t.Errorf("Initial MessagesReceived = %d, expected 0", player.MessagesReceived())
	}

	// Simulate receiving messages
	player.mu.Lock()
	player.messagesReceived = 5
	player.mu.Unlock()

	if player.MessagesReceived() != 5 {
		t.Errorf("MessagesReceived = %d, expected 5", player.MessagesReceived())
	}
}

func TestAutoPlayPlayerTurnsCompleted(t *testing.T) {
	mockSocket := NewMockSocketClient()
	logger := nakama.NewLogger(false)
	player := NewAutoPlayPlayerStandalone(mockSocket, "user-001", logger)

	// Initial count should be 0
	if player.TurnsCompleted() != 0 {
		t.Errorf("Initial TurnsCompleted = %d, expected 0", player.TurnsCompleted())
	}

	// Simulate turns completed
	player.mu.Lock()
	player.turnsCompleted = 3
	player.mu.Unlock()

	if player.TurnsCompleted() != 3 {
		t.Errorf("TurnsCompleted = %d, expected 3", player.TurnsCompleted())
	}
}

func TestAutoPlayPlayerGlobalState(t *testing.T) {
	mockSocket := NewMockSocketClient()
	logger := nakama.NewLogger(false)
	player := NewAutoPlayPlayerStandalone(mockSocket, "user-001", logger)

	// Initial state should be empty
	if player.GlobalState() != "" {
		t.Errorf("Initial GlobalState = %s, expected empty", player.GlobalState())
	}

	// Set global state
	player.mu.Lock()
	player.globalState = "turn_loop"
	player.mu.Unlock()

	if player.GlobalState() != "turn_loop" {
		t.Errorf("GlobalState = %s, expected turn_loop", player.GlobalState())
	}
}

func TestAutoPlayPlayerRejections(t *testing.T) {
	mockSocket := NewMockSocketClient()
	logger := nakama.NewLogger(false)
	player := NewAutoPlayPlayerStandalone(mockSocket, "user-001", logger)

	// Initial count should be 0
	if player.Rejections() != 0 {
		t.Errorf("Initial Rejections = %d, expected 0", player.Rejections())
	}

	// Simulate rejections
	player.mu.Lock()
	player.rejections = 2
	player.mu.Unlock()

	if player.Rejections() != 2 {
		t.Errorf("Rejections = %d, expected 2", player.Rejections())
	}
}

func TestAutoPlayPlayerLastRejection(t *testing.T) {
	mockSocket := NewMockSocketClient()
	logger := nakama.NewLogger(false)
	player := NewAutoPlayPlayerStandalone(mockSocket, "user-001", logger)

	// Initial should be nil
	if player.LastRejection() != nil {
		t.Error("Initial LastRejection should be nil")
	}

	// Set last rejection
	rejection := &model.ActionRejected{
		OpCode:  100,
		Reason:  "not_current_player",
		Message: "当前不是你的回合",
	}
	player.mu.Lock()
	player.lastRejection = rejection
	player.mu.Unlock()

	last := player.LastRejection()
	if last == nil {
		t.Fatal("LastRejection should return non-nil")
	}
	if last.OpCode != 100 {
		t.Errorf("OpCode = %d, expected 100", last.OpCode)
	}
	if last.Reason != "not_current_player" {
		t.Errorf("Reason = %s, expected not_current_player", last.Reason)
	}
}

func TestAutoPlayPlayerLastError(t *testing.T) {
	mockSocket := NewMockSocketClient()
	logger := nakama.NewLogger(false)
	player := NewAutoPlayPlayerStandalone(mockSocket, "user-001", logger)

	// Initial should be nil
	if player.LastError() != nil {
		t.Error("Initial LastError should be nil")
	}

	// Set last error
	testErr := context.DeadlineExceeded
	player.mu.Lock()
	player.lastErr = testErr
	player.mu.Unlock()

	if player.LastError() != testErr {
		t.Errorf("LastError = %v, expected %v", player.LastError(), testErr)
	}
}

// ========== Message Handler Tests ==========

func TestHandleStateSync(t *testing.T) {
	mockSocket := NewMockSocketClient()
	logger := nakama.NewLogger(false)
	// userID now equals PlayerID for direct matching
	player := NewAutoPlayPlayerStandalone(mockSocket, "player-001", logger)

	// Simulate StateSync message
	stateSync := model.StateSync{
		GlobalState:     "turn_loop",
		TurnState:       "main_action",
		CurrentPlayerID: "player-001",
		Round:           1,
		Turn:            2,
		Players: []model.Player{
			{
				PlayerID: "player-001",
				Faction:  "qing_long",
			},
			{
				PlayerID: "player-002",
				Faction:  "zhu_que",
			},
		},
	}

	ctx := context.Background()
	data, _ := json.Marshal(stateSync)
	player.handleStateSync(ctx, data)

	// Verify globalState updated
	if player.GlobalState() != "turn_loop" {
		t.Errorf("GlobalState = %s, expected turn_loop", player.GlobalState())
	}

	// Verify playerID extracted
	player.mu.RLock()
	pID := player.playerID
	player.mu.RUnlock()
	if pID != "player-001" {
		t.Errorf("playerID = %s, expected player-001", pID)
	}
}

func TestHandleStateSyncInvalidJSON(t *testing.T) {
	mockSocket := NewMockSocketClient()
	logger := nakama.NewLogger(true) // verbose to see error log
	player := NewAutoPlayPlayerStandalone(mockSocket, "user-001", logger)

	ctx := context.Background()
	// Invalid JSON should not panic
	player.handleStateSync(ctx, []byte("invalid json"))

	// State should remain unchanged
	if player.GlobalState() != "" {
		t.Errorf("GlobalState should remain empty after invalid JSON")
	}
}

func TestHandleAvailableRollDice(t *testing.T) {
	mockSocket := NewMockSocketClient()
	logger := nakama.NewLogger(false)
	player := NewAutoPlayPlayerStandalone(mockSocket, "user-001", logger)

	// Set currentPlayerID to match
	player.mu.Lock()
	player.currentPlayerID = "player-001"
	player.playerID = "player-001"
	player.mu.Unlock()

	// Simulate Available with no items, no skill
	available := model.Available{
		Items:       []model.Item{},
		CanUseSkill: false,
		DiceType:    "normal",
	}

	ctx := context.Background()
	data, _ := json.Marshal(available)
	player.handleAvailable(ctx, data)

	// Verify RollDice was sent
	sent := mockSocket.GetSentMessages()
	if len(sent) != 1 {
		t.Fatalf("Sent messages count = %d, expected 1", len(sent))
	}
	if sent[0].opCode != nakama.OpRollDice {
		t.Errorf("OpCode = %d, expected OpRollDice (%d)", sent[0].opCode, nakama.OpRollDice)
	}
}

func TestHandleAvailableUseItem(t *testing.T) {
	mockSocket := NewMockSocketClient()
	logger := nakama.NewLogger(false)
	player := NewAutoPlayPlayerStandalone(mockSocket, "user-001", logger)

	// Set currentPlayerID
	player.mu.Lock()
	player.currentPlayerID = "player-001"
	player.playerID = "player-001"
	player.mu.Unlock()

	// Simulate Available with items
	available := model.Available{
		Items: []model.Item{
			{ID: "item-001", Type: "reverse_clock", Name: "逆流沙漏"},
		},
		CanUseSkill: false,
		DiceType:    "normal",
	}

	ctx := context.Background()
	data, _ := json.Marshal(available)
	player.handleAvailable(ctx, data)

	// Verify UseItem was sent
	sent := mockSocket.GetSentMessages()
	if len(sent) != 1 {
		t.Fatalf("Sent messages count = %d, expected 1", len(sent))
	}
	if sent[0].opCode != nakama.OpUseItem {
		t.Errorf("OpCode = %d, expected OpUseItem (%d)", sent[0].opCode, nakama.OpUseItem)
	}

	// Verify item ID
	useItem, ok := sent[0].data.(model.UseItem)
	if !ok {
		t.Fatal("data should be UseItem type")
	}
	if useItem.ItemID != "item-001" {
		t.Errorf("ItemID = %s, expected item-001", useItem.ItemID)
	}
}

func TestHandleAvailableUseSkill(t *testing.T) {
	mockSocket := NewMockSocketClient()
	logger := nakama.NewLogger(false)
	player := NewAutoPlayPlayerStandalone(mockSocket, "user-001", logger)

	// Set currentPlayerID
	player.mu.Lock()
	player.currentPlayerID = "player-001"
	player.playerID = "player-001"
	player.mu.Unlock()

	// Simulate Available with skill available
	available := model.Available{
		Items:       []model.Item{},
		CanUseSkill: true,
		DiceType:    "normal",
	}

	ctx := context.Background()
	data, _ := json.Marshal(available)
	player.handleAvailable(ctx, data)

	// Verify UseSkill was sent
	sent := mockSocket.GetSentMessages()
	if len(sent) != 1 {
		t.Fatalf("Sent messages count = %d, expected 1", len(sent))
	}
	if sent[0].opCode != nakama.OpUseSkill {
		t.Errorf("OpCode = %d, expected OpUseSkill (%d)", sent[0].opCode, nakama.OpUseSkill)
	}
}

func TestHandleDecisionRequest(t *testing.T) {
	mockSocket := NewMockSocketClient()
	logger := nakama.NewLogger(false)
	player := NewAutoPlayPlayerStandalone(mockSocket, "user-001", logger)

	// Simulate DecisionRequest
	decision := model.Decision{
		ID:      "dec-001",
		Prompt:  "选择一个选项",
		Options: []model.Option{
			{ID: "opt-1", Label: "选项1"},
			{ID: "opt-2", Label: "选项2"},
		},
	}

	ctx := context.Background()
	data, _ := json.Marshal(decision)
	player.handleDecisionRequest(ctx, data)

	// Verify UserChoice was sent with choice 0
	sent := mockSocket.GetSentMessages()
	if len(sent) != 1 {
		t.Fatalf("Sent messages count = %d, expected 1", len(sent))
	}
	if sent[0].opCode != nakama.OpUserChoice {
		t.Errorf("OpCode = %d, expected OpUserChoice (%d)", sent[0].opCode, nakama.OpUserChoice)
	}

	userChoice, ok := sent[0].data.(model.UserChoice)
	if !ok {
		t.Fatal("data should be UserChoice type")
	}
	if userChoice.DecisionID != "dec-001" {
		t.Errorf("DecisionID = %s, expected dec-001", userChoice.DecisionID)
	}
	if userChoice.Choice != 0 {
		t.Errorf("Choice = %d, expected 0 (first option)", userChoice.Choice)
	}
}

func TestHandleMiniGameStart(t *testing.T) {
	mockSocket := NewMockSocketClient()
	logger := nakama.NewLogger(false)
	player := NewAutoPlayPlayerStandalone(mockSocket, "user-001", logger)

	// Simulate MiniGameStart with user-001 at index 0
	start := model.MiniGameStart{
		GameType: "dice_race",
		Players:  []string{"user-001", "user-002", "user-003"},
	}

	ctx := context.Background()
	data, _ := json.Marshal(start)
	player.handleMiniGameStart(ctx, data)

	// Verify MiniGameResultSubmit was sent
	sent := mockSocket.GetSentMessages()
	if len(sent) != 1 {
		t.Fatalf("Sent messages count = %d, expected 1", len(sent))
	}
	if sent[0].opCode != nakama.OpMiniGameResultSubmit {
		t.Errorf("OpCode = %d, expected OpMiniGameResultSubmit (%d)", sent[0].opCode, nakama.OpMiniGameResultSubmit)
	}

	submit, ok := sent[0].data.(model.MiniGameResultSubmit)
	if !ok {
		t.Fatal("data should be MiniGameResultSubmit type")
	}
	// user-001 is at index 0, so rank should be 1
	if submit.Rank != 1 {
		t.Errorf("Rank = %d, expected 1 (index 0 + 1)", submit.Rank)
	}
}

func TestHandleMiniGameStartDifferentIndex(t *testing.T) {
	mockSocket := NewMockSocketClient()
	logger := nakama.NewLogger(false)
	player := NewAutoPlayPlayerStandalone(mockSocket, "user-002", logger)

	// user-002 is at index 1
	start := model.MiniGameStart{
		GameType: "dice_race",
		Players:  []string{"user-001", "user-002", "user-003"},
	}

	ctx := context.Background()
	data, _ := json.Marshal(start)
	player.handleMiniGameStart(ctx, data)

	sent := mockSocket.GetSentMessages()
	submit, ok := sent[0].data.(model.MiniGameResultSubmit)
	if !ok {
		t.Fatal("data should be MiniGameResultSubmit type")
	}
	// user-002 is at index 1, so rank should be 2
	if submit.Rank != 2 {
		t.Errorf("Rank = %d, expected 2 (index 1 + 1)", submit.Rank)
	}
}

func TestHandleGameOver(t *testing.T) {
	mockSocket := NewMockSocketClient()
	logger := nakama.NewLogger(false)
	player := NewAutoPlayPlayerStandalone(mockSocket, "user-001", logger)

	// Simulate GameOver
	gameOver := model.GameOver{
		WinnerID: "player-001",
		Stats: []model.PlayerStats{
			{PlayerID: "player-001", RoundsWon: 3},
		},
	}

	ctx := context.Background()
	data, _ := json.Marshal(gameOver)
	player.handleGameOver(ctx, data)

	// Verify gameOverChan signaled
	select {
	case <-player.GameOverChan():
		// Success
	case <-time.After(100 * time.Millisecond):
		t.Error("GameOverChan should be signaled")
	}
}

func TestHandleMiniGameResult(t *testing.T) {
	mockSocket := NewMockSocketClient()
	logger := nakama.NewLogger(false)
	player := NewAutoPlayPlayerStandalone(mockSocket, "user-001", logger)

	// Set playerID
	player.mu.Lock()
	player.playerID = "player-001"
	player.mu.Unlock()

	// Simulate MiniGameResult
	result := model.MiniGameResult{
		Rankings: []model.RankingEntry{
			{PlayerID: "player-001", Rank: 1},
			{PlayerID: "player-002", Rank: 2},
		},
	}

	ctx := context.Background()
	data, _ := json.Marshal(result)
	player.handleMiniGameResult(ctx, data)

	// No error means success
}

func TestHandleFullSync(t *testing.T) {
	mockSocket := NewMockSocketClient()
	logger := nakama.NewLogger(false)
	player := NewAutoPlayPlayerStandalone(mockSocket, "user-001", logger)

	// Simulate FullSync (same structure as StateSync)
	stateSync := model.StateSync{
		GlobalState: "turn_loop",
		Round:       2,
		Turn:        5,
	}

	ctx := context.Background()
	data, _ := json.Marshal(stateSync)
	player.handleFullSync(ctx, data)

	// Verify globalState updated
	if player.GlobalState() != "turn_loop" {
		t.Errorf("GlobalState = %s, expected turn_loop", player.GlobalState())
	}
}

func TestHandleActionRejected(t *testing.T) {
	mockSocket := NewMockSocketClient()
	logger := nakama.NewLogger(false)
	player := NewAutoPlayPlayerStandalone(mockSocket, "user-001", logger)

	// Simulate ActionRejected
	rejected := model.ActionRejected{
		OpCode:  100,
		Reason:  "invalid_state",
		Message: "当前状态不允许此操作",
	}

	ctx := context.Background()
	data, _ := json.Marshal(rejected)
	player.handleActionRejected(ctx, data)

	// Verify rejection tracked
	if player.Rejections() != 1 {
		t.Errorf("Rejections = %d, expected 1", player.Rejections())
	}

	last := player.LastRejection()
	if last == nil {
		t.Fatal("LastRejection should be set")
	}
	if last.OpCode != 100 {
		t.Errorf("OpCode = %d, expected 100", last.OpCode)
	}
}

func TestHandleTurnSync(t *testing.T) {
	mockSocket := NewMockSocketClient()
	logger := nakama.NewLogger(false)
	player := NewAutoPlayPlayerStandalone(mockSocket, "user-001", logger)

	// Initial turns completed
	initial := player.TurnsCompleted()

	// Simulate TurnSync message
	ctx := context.Background()
	player.handleMessage(ctx, &nakama.SocketMessage{
		OpCode: nakama.OpTurnSync,
		Data:   []byte(`{"round": 1, "turn": 0}`),
	})

	// Verify turns completed incremented
	if player.TurnsCompleted() != initial+1 {
		t.Errorf("TurnsCompleted = %d, expected %d", player.TurnsCompleted(), initial+1)
	}
}

func TestHandleUnknownOpCode(t *testing.T) {
	mockSocket := NewMockSocketClient()
	logger := nakama.NewLogger(true)
	player := NewAutoPlayPlayerStandalone(mockSocket, "user-001", logger)

	// Simulate unknown OpCode
	ctx := context.Background()
	player.handleMessage(ctx, &nakama.SocketMessage{
		OpCode: 999,
		Data:   []byte(`{"test": "data"}`),
	})

	// Should not panic, messagesReceived should increment
	if player.MessagesReceived() != 1 {
		t.Errorf("MessagesReceived = %d, expected 1", player.MessagesReceived())
	}
}

// ========== ScenarioConfig Tests ==========

func TestScenarioConfig(t *testing.T) {
	config := ScenarioConfig{
		PlayersCount: 4,
		MatchName:    "test_match",
		MaxTurns:     50,
		TimeoutSec:   180,
		Mode:         "nakama",
	}

	if config.PlayersCount != 4 {
		t.Errorf("PlayersCount = %d, expected 4", config.PlayersCount)
	}
	if config.MatchName != "test_match" {
		t.Errorf("MatchName = %s, expected test_match", config.MatchName)
	}
	if config.Mode != "nakama" {
		t.Errorf("Mode = %s, expected nakama", config.Mode)
	}
}

// ========== Result Tests ==========

func TestResultJSON(t *testing.T) {
	result := Result{
		Success:          true,
		FailureReason:    "",
		Duration:         10 * time.Second,
		MessagesReceived: 100,
		TurnsCompleted:   20,
		GlobalState:      "game_over",
		Rejections:       0,
		LastError:        "",
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var parsed Result
	err = json.Unmarshal(data, &parsed)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if parsed.Success != result.Success {
		t.Errorf("Success = %v, expected %v", parsed.Success, result.Success)
	}
	if parsed.MessagesReceived != result.MessagesReceived {
		t.Errorf("MessagesReceived = %d, expected %d", parsed.MessagesReceived, result.MessagesReceived)
	}
}

func TestResultFailure(t *testing.T) {
	result := Result{
		Success:          false,
		FailureReason:    "timeout",
		Duration:         180 * time.Second,
		MessagesReceived: 50,
		TurnsCompleted:   5,
		GlobalState:      "turn_loop",
		Rejections:       2,
		LastError:        "context deadline exceeded",
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var parsed Result
	err = json.Unmarshal(data, &parsed)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if parsed.Success != false {
		t.Errorf("Success = %v, expected false", parsed.Success)
	}
	if parsed.FailureReason != "timeout" {
		t.Errorf("FailureReason = %s, expected timeout", parsed.FailureReason)
	}
}

func TestResultWithRejections(t *testing.T) {
	result := Result{
		Success:    true,
		Rejections: 3,
	}

	data, _ := json.Marshal(result)
	var parsed Result
	json.Unmarshal(data, &parsed)

	if parsed.Rejections != 3 {
		t.Errorf("Rejections = %d, expected 3", parsed.Rejections)
	}
}

// ========== Concurrent Access Tests ==========

func TestAutoPlayPlayerConcurrentAccess(t *testing.T) {
	mockSocket := NewMockSocketClient()
	logger := nakama.NewLogger(false)
	player := NewAutoPlayPlayerStandalone(mockSocket, "user-001", logger)

	// Concurrent access to state variables
	var wg sync.WaitGroup
	wg.Add(10)

	for i := 0; i < 5; i++ {
		go func() {
			defer wg.Done()
			player.mu.Lock()
			player.messagesReceived++
			player.mu.Unlock()
		}()
	}

	for i := 0; i < 5; i++ {
		go func() {
			defer wg.Done()
			_ = player.MessagesReceived()
		}()
	}

	wg.Wait()

	if player.MessagesReceived() != 5 {
		t.Errorf("MessagesReceived = %d, expected 5", player.MessagesReceived())
	}
}