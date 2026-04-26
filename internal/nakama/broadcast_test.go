// Package nakama provides Nakama match handler implementation for Paradiced.
package nakama

import (
	"testing"

	"github.com/b1tAction/paradiced/pkg/constants"
	"github.com/b1tAction/paradiced/pkg/gamelog"
	"github.com/b1tAction/paradiced/pkg/net"
)

func TestBroadcastStateSync(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)
	mockDispatcher := NewMockDispatcherAdapter()
	handler.WithDispatcher(mockDispatcher)

	broadcastAdapter := NewNakamaBroadcastAdapter(handler)

	stateSync := &net.StateSync{
		GlobalState:   "turn_loop",
		Round:         1,
		Turn:          0,
		CurrentPlayerID: "player-001",
	}

	err := broadcastAdapter.BroadcastStateSync(stateSync)
	if err != nil {
		t.Fatalf("BroadcastStateSync error: %v", err)
	}

	if mockDispatcher.CountBroadcasts() != 1 {
		t.Errorf("broadcasts count = %d, want 1", mockDispatcher.CountBroadcasts())
	}

	// Verify OpCode
	broadcasts := mockDispatcher.GetBroadcasts()
	if broadcasts[0].OpCode != int64(net.OpStateSync) {
		t.Errorf("OpCode = %d, want %d", broadcasts[0].OpCode, int64(net.OpStateSync))
	}

	// Parse and verify data
	var parsed net.StateSync
	err = mockDispatcher.ParseBroadcastData(0, &parsed)
	if err != nil {
		t.Fatalf("ParseBroadcastData error: %v", err)
	}

	if parsed.GlobalState != "turn_loop" {
		t.Errorf("parsed.GlobalState = %s, want turn_loop", parsed.GlobalState)
	}

	if parsed.Round != 1 {
		t.Errorf("parsed.Round = %d, want 1", parsed.Round)
	}
}

func TestBroadcastStateSyncWithEntries(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)
	mockDispatcher := NewMockDispatcherAdapter()
	handler.WithDispatcher(mockDispatcher)

	broadcastAdapter := NewNakamaBroadcastAdapter(handler)

	entry := gamelog.NewActionEntry("damage", "player-001", "Cell_Fragile")
	stateSync := &net.StateSync{
		GlobalState:     "turn_end",
		TurnState:       "turn_end",
		Round:           1,
		Turn:            0,
		CurrentPlayerID: "player-001",
		Entries:         []gamelog.LogEntry{entry},
	}

	err := broadcastAdapter.BroadcastStateSync(stateSync)
	if err != nil {
		t.Fatalf("BroadcastStateSync error: %v", err)
	}

	if mockDispatcher.CountBroadcasts() != 1 {
		t.Errorf("broadcasts count = %d, want 1", mockDispatcher.CountBroadcasts())
	}

	// Verify OpCode
	broadcasts := mockDispatcher.GetBroadcasts()
	if broadcasts[0].OpCode != int64(net.OpStateSync) {
		t.Errorf("OpCode = %d, want %d", broadcasts[0].OpCode, int64(net.OpStateSync))
	}

	// Parse and verify data
	var parsed net.StateSync
	err = mockDispatcher.ParseBroadcastData(0, &parsed)
	if err != nil {
		t.Fatalf("ParseBroadcastData error: %v", err)
	}

	if parsed.CurrentPlayerID != "player-001" {
		t.Errorf("parsed.CurrentPlayerID = %s, want player-001", parsed.CurrentPlayerID)
	}
	if len(parsed.Entries) != 1 {
		t.Errorf("len(parsed.Entries) = %d, want 1", len(parsed.Entries))
	}
	if parsed.Entries[0].ActionType != "damage" {
		t.Errorf("parsed.Entries[0].ActionType = %s, want damage", parsed.Entries[0].ActionType)
	}
}

func TestSendDecision(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)
	mockDispatcher := NewMockDispatcherAdapter()
	handler.WithDispatcher(mockDispatcher)

	broadcastAdapter := NewNakamaBroadcastAdapter(handler)

	decision := &net.Decision{
		ID:      "dec-001",
		Prompt:  "选择目标",
		Context: "Item_AnyDoor",
	}

	err := broadcastAdapter.SendDecision("player-001", decision)
	if err != nil {
		t.Fatalf("SendDecision error: %v", err)
	}

	if mockDispatcher.CountMessages("player-001") != 1 {
		t.Errorf("messages count = %d, want 1", mockDispatcher.CountMessages("player-001"))
	}

	// Verify OpCode
	messages := mockDispatcher.GetMessages("player-001")
	if messages[0].OpCode != int64(net.OpDecisionRequest) {
		t.Errorf("OpCode = %d, want %d", messages[0].OpCode, int64(net.OpDecisionRequest))
	}

	// Parse and verify data
	var parsed net.Decision
	err = mockDispatcher.ParseMessageData("player-001", 0, &parsed)
	if err != nil {
		t.Fatalf("ParseMessageData error: %v", err)
	}

	if parsed.ID != "dec-001" {
		t.Errorf("parsed.ID = %s, want dec-001", parsed.ID)
	}
}

func TestSendAvailable(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)
	mockDispatcher := NewMockDispatcherAdapter()
	handler.WithDispatcher(mockDispatcher)

	broadcastAdapter := NewNakamaBroadcastAdapter(handler)

	available := &net.Available{
		Items:       []net.Item{{ID: "item-001", Type: "any_door", Name: "任意门"}},
		CanUseSkill: true,
		DiceType:    "gold",
	}

	err := broadcastAdapter.SendAvailable("player-001", available)
	if err != nil {
		t.Fatalf("SendAvailable error: %v", err)
	}

	if mockDispatcher.CountMessages("player-001") != 1 {
		t.Errorf("messages count = %d, want 1", mockDispatcher.CountMessages("player-001"))
	}

	// Verify OpCode
	messages := mockDispatcher.GetMessages("player-001")
	if messages[0].OpCode != int64(net.OpAvailable) {
		t.Errorf("OpCode = %d, want %d", messages[0].OpCode, int64(net.OpAvailable))
	}

	// Parse and verify data
	var parsed net.Available
	err = mockDispatcher.ParseMessageData("player-001", 0, &parsed)
	if err != nil {
		t.Fatalf("ParseMessageData error: %v", err)
	}

	if parsed.DiceType != "gold" {
		t.Errorf("parsed.DiceType = %s, want gold", parsed.DiceType)
	}

	if parsed.CanUseSkill != true {
		t.Errorf("parsed.CanUseSkill = %v, want true", parsed.CanUseSkill)
	}
}

func TestBroadcastMiniGameStart(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)
	mockDispatcher := NewMockDispatcherAdapter()
	handler.WithDispatcher(mockDispatcher)

	broadcastAdapter := NewNakamaBroadcastAdapter(handler)

	start := &net.MiniGameStart{
		GameType: "dice_race",
		Players:  []string{"p1", "p2", "p3", "p4"},
	}

	err := broadcastAdapter.BroadcastMiniGameStart(start)
	if err != nil {
		t.Fatalf("BroadcastMiniGameStart error: %v", err)
	}

	if mockDispatcher.CountBroadcasts() != 1 {
		t.Errorf("broadcasts count = %d, want 1", mockDispatcher.CountBroadcasts())
	}

	// Verify OpCode
	broadcasts := mockDispatcher.GetBroadcasts()
	if broadcasts[0].OpCode != int64(net.OpMiniGameStart) {
		t.Errorf("OpCode = %d, want %d", broadcasts[0].OpCode, int64(net.OpMiniGameStart))
	}

	// Parse and verify data
	var parsed net.MiniGameStart
	err = mockDispatcher.ParseBroadcastData(0, &parsed)
	if err != nil {
		t.Fatalf("ParseBroadcastData error: %v", err)
	}

	if parsed.GameType != "dice_race" {
		t.Errorf("parsed.GameType = %s, want dice_race", parsed.GameType)
	}
}

func TestBroadcastMiniGameResult(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)
	mockDispatcher := NewMockDispatcherAdapter()
	handler.WithDispatcher(mockDispatcher)

	// Add players with display names for DisplayName injection testing
	// Use valid UUID format for userID since it's now parsed as PlayerID
	userID1 := "00000000-0000-0000-0000-000000000001"
	userID2 := "00000000-0000-0000-0000-000000000002"
	p1 := handler.addPlayer(userID1, constants.FactionQingLong, "Alice")
	p2 := handler.addPlayer(userID2, constants.FactionZhuQue, "Bob")

	broadcastAdapter := NewNakamaBroadcastAdapter(handler)

	result := &net.MiniGameResult{
		Rankings: []net.RankingEntry{
			{PlayerID: p1.ID.UUID(), Rank: 1},
			{PlayerID: p2.ID.UUID(), Rank: 2},
		},
	}

	err := broadcastAdapter.BroadcastMiniGameResult(result)
	if err != nil {
		t.Fatalf("BroadcastMiniGameResult error: %v", err)
	}

	if mockDispatcher.CountBroadcasts() != 1 {
		t.Errorf("broadcasts count = %d, want 1", mockDispatcher.CountBroadcasts())
	}

	// Verify OpCode
	broadcasts := mockDispatcher.GetBroadcasts()
	if broadcasts[0].OpCode != int64(net.OpMiniGameResult) {
		t.Errorf("OpCode = %d, want %d", broadcasts[0].OpCode, int64(net.OpMiniGameResult))
	}

	// Verify DisplayName injection (PlayerID is no longer converted, it equals userID)
	// PlayerID should remain the same (UUID format)
	if result.Rankings[0].PlayerID != userID1 {
		t.Errorf("Rankings[0].PlayerID = %s, want %s", result.Rankings[0].PlayerID, userID1)
	}
	if result.Rankings[0].DisplayName != "Alice" {
		t.Errorf("Rankings[0].DisplayName = %s, want Alice", result.Rankings[0].DisplayName)
	}
	if result.Rankings[1].PlayerID != userID2 {
		t.Errorf("Rankings[1].PlayerID = %s, want %s", result.Rankings[1].PlayerID, userID2)
	}
	if result.Rankings[1].DisplayName != "Bob" {
		t.Errorf("Rankings[1].DisplayName = %s, want Bob", result.Rankings[1].DisplayName)
	}
}

func TestBroadcastGameOver(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)
	mockDispatcher := NewMockDispatcherAdapter()
	handler.WithDispatcher(mockDispatcher)

	broadcastAdapter := NewNakamaBroadcastAdapter(handler)

	over := &net.GameOver{
		WinnerID: "player-001",
		Stats:    []net.PlayerStats{},
	}

	err := broadcastAdapter.BroadcastGameOver(over)
	if err != nil {
		t.Fatalf("BroadcastGameOver error: %v", err)
	}

	if mockDispatcher.CountBroadcasts() != 1 {
		t.Errorf("broadcasts count = %d, want 1", mockDispatcher.CountBroadcasts())
	}

	// Verify OpCode
	broadcasts := mockDispatcher.GetBroadcasts()
	if broadcasts[0].OpCode != int64(net.OpGameOver) {
		t.Errorf("OpCode = %d, want %d", broadcasts[0].OpCode, int64(net.OpGameOver))
	}

	// Parse and verify data
	var parsed net.GameOver
	err = mockDispatcher.ParseBroadcastData(0, &parsed)
	if err != nil {
		t.Fatalf("ParseBroadcastData error: %v", err)
	}

	if parsed.WinnerID != "player-001" {
		t.Errorf("parsed.WinnerID = %s, want player-001", parsed.WinnerID)
	}
}

func TestSendFullSync(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)
	mockDispatcher := NewMockDispatcherAdapter()
	handler.WithDispatcher(mockDispatcher)

	broadcastAdapter := NewNakamaBroadcastAdapter(handler)

	stateSync := &net.StateSync{GlobalState: "turn_loop", Round: 1}

	err := broadcastAdapter.SendFullSync("player-001", stateSync)
	if err != nil {
		t.Fatalf("SendFullSync error: %v", err)
	}

	if mockDispatcher.CountMessages("player-001") != 1 {
		t.Errorf("messages count = %d, want 1", mockDispatcher.CountMessages("player-001"))
	}

	// Verify OpCode
	messages := mockDispatcher.GetMessages("player-001")
	if messages[0].OpCode != int64(net.OpFullSync) {
		t.Errorf("OpCode = %d, want %d", messages[0].OpCode, int64(net.OpFullSync))
	}

	// Parse and verify data - FullSync now sends plain StateSync
	var parsed net.StateSync
	err = mockDispatcher.ParseMessageData("player-001", 0, &parsed)
	if err != nil {
		t.Fatalf("ParseMessageData error: %v", err)
	}
	if parsed.GlobalState != "turn_loop" {
		t.Errorf("parsed.GlobalState = %s, want turn_loop", parsed.GlobalState)
	}
	if parsed.Round != 1 {
		t.Errorf("parsed.Round = %d, want 1", parsed.Round)
	}
}

func TestBroadcastWithoutDispatcher(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)
	// No dispatcher set

	broadcastAdapter := NewNakamaBroadcastAdapter(handler)

	// All methods should return nil without dispatcher
	err := broadcastAdapter.BroadcastStateSync(&net.StateSync{})
	if err != nil {
		t.Errorf("BroadcastStateSync should return nil without dispatcher, got: %v", err)
	}

	err = broadcastAdapter.SendDecision("player-001", &net.Decision{})
	if err != nil {
		t.Errorf("SendDecision should return nil without dispatcher, got: %v", err)
	}

	err = broadcastAdapter.SendAvailable("player-001", &net.Available{})
	if err != nil {
		t.Errorf("SendAvailable should return nil without dispatcher, got: %v", err)
	}
}

func TestMockDispatcherClear(t *testing.T) {
	mockDispatcher := NewMockDispatcherAdapter()

	// Add some messages
	mockDispatcher.BroadcastMessage(1, []byte("test"))
	mockDispatcher.SendMessage("p1", 2, []byte("test"))

	if mockDispatcher.CountBroadcasts() != 1 {
		t.Error("should have 1 broadcast before Clear")
	}

	if mockDispatcher.CountMessages("p1") != 1 {
		t.Error("should have 1 message before Clear")
	}

	// Clear
	mockDispatcher.Clear()

	if mockDispatcher.CountBroadcasts() != 0 {
		t.Errorf("broadcasts count = %d after Clear, want 0", mockDispatcher.CountBroadcasts())
	}

	if mockDispatcher.CountMessages("p1") != 0 {
		t.Errorf("messages count = %d after Clear, want 0", mockDispatcher.CountMessages("p1"))
	}
}

func TestMockDispatcherMultipleCalls(t *testing.T) {
	mockDispatcher := NewMockDispatcherAdapter()

	// Multiple broadcasts
	for i := 0; i < 5; i++ {
		mockDispatcher.BroadcastMessage(int64(i), []byte("data"))
	}

	if mockDispatcher.CountBroadcasts() != 5 {
		t.Errorf("broadcasts count = %d, want 5", mockDispatcher.CountBroadcasts())
	}

	// Multiple messages to same player
	for i := 0; i < 3; i++ {
		mockDispatcher.SendMessage("p1", int64(i), []byte("data"))
	}

	if mockDispatcher.CountMessages("p1") != 3 {
		t.Errorf("messages count = %d, want 3", mockDispatcher.CountMessages("p1"))
	}

	// Messages to different players
	mockDispatcher.SendMessage("p2", 1, []byte("data"))
	mockDispatcher.SendMessage("p3", 1, []byte("data"))

	if mockDispatcher.CountMessages("p2") != 1 {
		t.Errorf("p2 messages count = %d, want 1", mockDispatcher.CountMessages("p2"))
	}

	if mockDispatcher.CountMessages("p3") != 1 {
		t.Errorf("p3 messages count = %d, want 1", mockDispatcher.CountMessages("p3"))
	}
}

func TestSendActionRejected(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)
	mockDispatcher := NewMockDispatcherAdapter()
	handler.WithDispatcher(mockDispatcher)

	broadcastAdapter := NewNakamaBroadcastAdapter(handler)

	rejected := &net.ActionRejected{
		OpCode:    net.OpRollDice,
		ErrorCode: constants.ErrInvalidParameter,
		Message:   "Not your turn",
	}

	err := broadcastAdapter.SendActionRejected("player-001", rejected)
	if err != nil {
		t.Fatalf("SendActionRejected error: %v", err)
	}

	if mockDispatcher.CountMessages("player-001") != 1 {
		t.Errorf("messages count = %d, want 1", mockDispatcher.CountMessages("player-001"))
	}

	// Verify OpCode
	messages := mockDispatcher.GetMessages("player-001")
	if messages[0].OpCode != int64(net.OpActionRejected) {
		t.Errorf("OpCode = %d, want %d", messages[0].OpCode, int64(net.OpActionRejected))
	}

	// Parse and verify data
	var parsed net.ActionRejected
	err = mockDispatcher.ParseMessageData("player-001", 0, &parsed)
	if err != nil {
		t.Fatalf("ParseMessageData error: %v", err)
	}

	if parsed.ErrorCode != constants.ErrInvalidParameter {
		t.Errorf("parsed.ErrorCode = %d, want %d", parsed.ErrorCode, constants.ErrInvalidParameter)
	}

	if parsed.Message != "Not your turn" {
		t.Errorf("parsed.Message = %s, want 'Not your turn'", parsed.Message)
	}

	if parsed.OpCode != net.OpRollDice {
		t.Errorf("parsed.OpCode = %d, want %d", parsed.OpCode, net.OpRollDice)
	}
}

func TestSendActionRejectedWithoutDispatcher(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)
	// No dispatcher set

	broadcastAdapter := NewNakamaBroadcastAdapter(handler)

	rejected := &net.ActionRejected{
		OpCode:    net.OpRollDice,
		ErrorCode: constants.ErrInvalidParameter,
		Message:   "Not your turn",
	}

	err := broadcastAdapter.SendActionRejected("player-001", rejected)
	if err != nil {
		t.Errorf("SendActionRejected should return nil without dispatcher, got: %v", err)
	}
}

func TestResolveRecipientUserID(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)

	broadcastAdapter := NewNakamaBroadcastAdapter(handler)

	// Test with nil handler
	nilAdapter := &NakamaBroadcastAdapter{handler: nil}
	result := nilAdapter.resolveRecipientUserID("player-001")
	if result != "player-001" {
		t.Errorf("resolveRecipientUserID with nil handler should return input, got: %s", result)
	}

	// Test with already known userID
	handler.players["user-001"] = nil
	result = broadcastAdapter.resolveRecipientUserID("user-001")
	if result != "user-001" {
		t.Errorf("resolveRecipientUserID with known userID should return same, got: %s", result)
	}
}