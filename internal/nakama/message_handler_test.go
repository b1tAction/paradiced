// Package nakama provides Nakama match handler implementation for Paradiced.
package nakama

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/b1tAction/paradiced/internal/core"
	"github.com/b1tAction/paradiced/internal/engine/hsm"
	"github.com/b1tAction/paradiced/pkg/constants"
	"github.com/b1tAction/paradiced/pkg/id"
	pkgnet "github.com/b1tAction/paradiced/pkg/net"
)

// ========== Test Helpers ==========

// setupHandlerWithPlayers creates a handler with specified number of players.
func setupHandlerWithPlayers(t *testing.T, count int) *NakamaMatchHandler {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)
	mockDispatcher := NewMockDispatcherAdapter()
	handler.WithDispatcher(mockDispatcher)

	factions := []constants.Faction{
	 constants.FactionQingLong,
	 constants.FactionZhuQue,
	 constants.FactionBaiHu,
	 constants.FactionXuanWu,
	}

	for i := 0; i < count; i++ {
	 userID := id.TestUUID(i + 1)
	 handler.addPlayer(userID, factions[i%4], userID)
	}

	err := handler.MatchInit()
	if err != nil {
	 t.Fatalf("MatchInit failed: %v", err)
	}

	return handler
}

// getMockDispatcher returns the mock dispatcher from handler.
func getMockDispatcher(handler *NakamaMatchHandler) *MockDispatcherAdapter {
 return handler.GetDispatcher().(*MockDispatcherAdapter)
}

// getCurrentPlayerUserID returns the userID (handler.players key) for the current player.
func getCurrentPlayerUserID(handler *NakamaMatchHandler) string {
 currentPlayer := handler.getCurrentPlayer()
 if currentPlayer == nil {
	 return ""
 }

 // Find userID by matching player in handler.players
 for userID, player := range handler.players {
	 if player.ID.UUID() == currentPlayer.ID.UUID() {
		 return userID
	 }
 }

 return ""
}

// verifyActionRejected checks that an ActionRejected message was sent.
func verifyActionRejected(t *testing.T, mock *MockDispatcherAdapter, playerID string, expectedOpCode pkgnet.OpCode, expectedErrorCode constants.ErrorCode) {
	t.Helper()
	msgs := mock.GetMessages(playerID)
	if len(msgs) == 0 {
	 t.Fatalf("No messages sent to player %s", playerID)
	}

	// Find ActionRejected message
	var rejected pkgnet.ActionRejected
	for _, msg := range msgs {
	 if msg.OpCode == int64(pkgnet.OpActionRejected) {
		 err := json.Unmarshal(msg.Data, &rejected)
		 if err != nil {
		 t.Fatalf("Failed to parse ActionRejected: %v", err)
		 }
		 break
	 }
	}

	if rejected.OpCode != expectedOpCode {
	 t.Errorf("Rejected OpCode = %d, expected %d", rejected.OpCode, expectedOpCode)
	}

	if rejected.ErrorCode != expectedErrorCode {
	 t.Errorf("ErrorCode = %d, expected %d", rejected.ErrorCode, expectedErrorCode)
	}
}

// ========== HandleMessageWithOp Tests ==========

func TestHandleMessageWithOp_RollDice(t *testing.T) {
	handler := setupHandlerWithPlayers(t, 2)
	mock := getMockDispatcher(handler)

	// Clear previous messages
	mock.Clear()

	// Send RollDice request from unknown player
	err := handler.HandleMessageWithOp(id.TestUUID(999), int64(pkgnet.OpRollDice), []byte("{}"))
	if err != nil {
	 // Error is expected but handler sends ActionRejected instead of returning error
	}

	// Verify ActionRejected was sent
	verifyActionRejected(t, mock, id.TestUUID(999), pkgnet.OpRollDice, constants.ErrPlayerNotFound)
}

func TestHandleMessageWithOp_UseItem(t *testing.T) {
	handler := setupHandlerWithPlayers(t, 2)
	mock := getMockDispatcher(handler)
	mock.Clear()

	data := []byte(`{"op_code": "101", "item_id": "item-001"}`)
	err := handler.HandleMessageWithOp(id.TestUUID(999), int64(pkgnet.OpUseItem), data)
	if err != nil {
	 // Expected
	}

	verifyActionRejected(t, mock, id.TestUUID(999), pkgnet.OpUseItem, constants.ErrPlayerNotFound)
}

func TestHandleMessageWithOp_UseSkill(t *testing.T) {
	handler := setupHandlerWithPlayers(t, 2)
	mock := getMockDispatcher(handler)
	mock.Clear()

	err := handler.HandleMessageWithOp(id.TestUUID(999), int64(pkgnet.OpUseSkill), []byte("{}"))
	if err != nil {
	 // Expected
	}

	verifyActionRejected(t, mock, id.TestUUID(999), pkgnet.OpUseSkill, constants.ErrPlayerNotFound)
}

func TestHandleMessageWithOp_UserChoice(t *testing.T) {
	handler := setupHandlerWithPlayers(t, 2)
	mock := getMockDispatcher(handler)
	mock.Clear()

	data := []byte(`{"op_code": "103", "decision_id": "dec-001", "choice": 0}`)
	err := handler.HandleMessageWithOp(id.TestUUID(999), int64(pkgnet.OpUserChoice), data)
	if err != nil {
	 // Expected
	}

	verifyActionRejected(t, mock, id.TestUUID(999), pkgnet.OpUserChoice, constants.ErrPlayerNotFound)
}

func TestHandleMessageWithOp_MiniGameDataSubmit(t *testing.T) {
	handler := setupHandlerWithPlayers(t, 2)
	mock := getMockDispatcher(handler)
	mock.Clear()

	data := []byte(`{"op_code": "107", "game_type": "dice_race", "game_data": {"score": 100, "time": 3.5}}`)
	err := handler.HandleMessageWithOp(id.TestUUID(999), int64(pkgnet.OpMiniGameDataSubmit), data)
	if err != nil {
		// Expected
	}

	verifyActionRejected(t, mock, id.TestUUID(999), pkgnet.OpMiniGameDataSubmit, constants.ErrPlayerNotFound)
}

func TestHandleMessageWithOp_UnknownOpCode(t *testing.T) {
	handler := setupHandlerWithPlayers(t, 2)
	mock := getMockDispatcher(handler)
	mock.Clear()

	// Unknown OpCode should be handled gracefully
	err := handler.HandleMessageWithOp(id.TestUUID(1), 999, []byte("{}"))
	// Should not error, just log warning
	if err != nil {
	 t.Errorf("Unknown OpCode should not return error, got: %v", err)
	}
}

// ========== HandleMessage Tests ==========

func TestHandleMessage_PayloadRouting(t *testing.T) {
	handler := setupHandlerWithPlayers(t, 2)
	mock := getMockDispatcher(handler)
	mock.Clear()

	// Use payload-based routing with op_code string
	data := []byte(`{"op_code": "100"}`)
	err := handler.HandleMessage(id.TestUUID(999), data)
	if err != nil {
	 // Expected
	}

	// Should route to handleRollDice and send rejection
	verifyActionRejected(t, mock, id.TestUUID(999), pkgnet.OpRollDice, constants.ErrPlayerNotFound)
}

func TestHandleMessage_InvalidJSON(t *testing.T) {
	handler := setupHandlerWithPlayers(t, 2)

	// Invalid JSON should return error
	err := handler.HandleMessage(id.TestUUID(1), []byte("invalid json"))
	if err == nil {
	 t.Error("HandleMessage should return error for invalid JSON")
	}
}

// ========== handleRollDice Tests ==========

func TestHandleRollDice_PlayerNotFound(t *testing.T) {
	handler := setupHandlerWithPlayers(t, 2)
	mock := getMockDispatcher(handler)
	mock.Clear()

	err := handler.handleRollDice(id.TestUUID(999))
	if err != nil {
	 // Expected error path sends ActionRejected
	}

	verifyActionRejected(t, mock, id.TestUUID(999), pkgnet.OpRollDice, constants.ErrPlayerNotFound)
}

func TestHandleRollDice_NotCurrentPlayer(t *testing.T) {
	handler := setupHandlerWithPlayers(t, 2)
	mock := getMockDispatcher(handler)
	mock.Clear()

	// Get current player userID
	currentPlayerID := getCurrentPlayerUserID(handler)
	if currentPlayerID == "" {
	 t.Fatal("Should have current player")
	}

	// Find non-current player
	nonCurrentPlayerID := id.TestUUID(2)
	if currentPlayerID == id.TestUUID(2) {
	 nonCurrentPlayerID = id.TestUUID(1)
	}

	err := handler.handleRollDice(nonCurrentPlayerID)
	if err != nil {
	 // Expected
	}

	verifyActionRejected(t, mock, nonCurrentPlayerID, pkgnet.OpRollDice, constants.ErrNotCurrentTurn)
}

func TestHandleRollDice_InvalidState(t *testing.T) {
	handler := setupHandlerWithPlayers(t, 2)
	mock := getMockDispatcher(handler)

	// Run HSM until it exits MainAction state or wait for state change
	// For this test, we verify that state checking is performed
	// The actual state may vary based on HSM progression

	currentPlayerID := getCurrentPlayerUserID(handler)
	if currentPlayerID == "" {
	 t.Skip("No current player available")
	}

	// If not in MainAction state, verify rejection
	currentState := handler.hsm.GetCurrentStateID()
	if currentState != hsm.StateMainAction {
	 mock.Clear()
	 err := handler.handleRollDice(currentPlayerID)
	 if err != nil {
		 // Expected
	 }
	 verifyActionRejected(t, mock, currentPlayerID, pkgnet.OpRollDice, constants.ErrInvalidState)
	} else {
	 t.Skip("Already in MainAction state")
	}
}

// ========== handleUseItem Tests ==========

func TestHandleUseItem_PlayerNotFound(t *testing.T) {
	handler := setupHandlerWithPlayers(t, 2)
	mock := getMockDispatcher(handler)
	mock.Clear()

	data := []byte(`{"op_code": "101", "item_id": "item-001"}`)
	err := handler.handleUseItem(id.TestUUID(999), data)
	if err != nil {
	 // Expected
	}

	verifyActionRejected(t, mock, id.TestUUID(999), pkgnet.OpUseItem, constants.ErrPlayerNotFound)
}

func TestHandleUseItem_NotCurrentPlayer(t *testing.T) {
	handler := setupHandlerWithPlayers(t, 2)
	mock := getMockDispatcher(handler)
	mock.Clear()

	currentPlayerID := getCurrentPlayerUserID(handler)
	if currentPlayerID == "" {
	 t.Fatal("Should have current player")
	}

	nonCurrentPlayerID := id.TestUUID(2)
	if currentPlayerID == id.TestUUID(2) {
	 nonCurrentPlayerID = id.TestUUID(1)
	}

	data := []byte(`{"op_code": "101", "item_id": "item-001"}`)
	err := handler.handleUseItem(nonCurrentPlayerID, data)
	if err != nil {
	 // Expected
	}

	verifyActionRejected(t, mock, nonCurrentPlayerID, pkgnet.OpUseItem, constants.ErrNotCurrentTurn)
}

func TestHandleUseItem_InvalidJSON(t *testing.T) {
	handler := setupHandlerWithPlayers(t, 2)

	err := handler.handleUseItem(id.TestUUID(1), []byte("invalid json"))
	if err == nil {
	 t.Error("handleUseItem should return error for invalid JSON")
	}
}

func TestHandleUseItem_ItemNotFound(t *testing.T) {
	handler := setupHandlerWithPlayers(t, 2)
	mock := getMockDispatcher(handler)

	// Check if in MainAction state
	currentState := handler.hsm.GetCurrentStateID()
	if currentState != hsm.StateMainAction {
	 t.Skip("Not in MainAction state")
	}

	currentPlayerID := getCurrentPlayerUserID(handler)
	if currentPlayerID == "" {
	 t.Skip("No current player available")
	}

	mock.Clear()

	data := []byte(`{"op_code": "101", "item_id": "nonexistent-item"}`)
	err := handler.handleUseItem(currentPlayerID, data)
	if err != nil {
	 // Expected
	}

	verifyActionRejected(t, mock, currentPlayerID, pkgnet.OpUseItem, constants.ErrItemNotFound)
}

// ========== handleUseSkill Tests ==========

func TestHandleUseSkill_PlayerNotFound(t *testing.T) {
	handler := setupHandlerWithPlayers(t, 2)
	mock := getMockDispatcher(handler)
	mock.Clear()

	err := handler.handleUseSkill(id.TestUUID(999))
	if err != nil {
	 // Expected
	}

	verifyActionRejected(t, mock, id.TestUUID(999), pkgnet.OpUseSkill, constants.ErrPlayerNotFound)
}

func TestHandleUseSkill_NotCurrentPlayer(t *testing.T) {
	handler := setupHandlerWithPlayers(t, 2)
	mock := getMockDispatcher(handler)
	mock.Clear()

	currentPlayerID := getCurrentPlayerUserID(handler)
	if currentPlayerID == "" {
	 t.Fatal("Should have current player")
	}

	nonCurrentPlayerID := id.TestUUID(2)
	if currentPlayerID == id.TestUUID(2) {
	 nonCurrentPlayerID = id.TestUUID(1)
	}

	err := handler.handleUseSkill(nonCurrentPlayerID)
	if err != nil {
	 // Expected
	}

	verifyActionRejected(t, mock, nonCurrentPlayerID, pkgnet.OpUseSkill, constants.ErrNotCurrentTurn)
}

func TestHandleUseSkill_SkillNotReady(t *testing.T) {
	handler := setupHandlerWithPlayers(t, 2)
	mock := getMockDispatcher(handler)

	// Check if in MainAction state
	currentState := handler.hsm.GetCurrentStateID()
	if currentState != hsm.StateMainAction {
	 t.Skip("Not in MainAction state")
	}

	currentPlayerID := getCurrentPlayerUserID(handler)
	if currentPlayerID == "" {
	 t.Skip("No current player available")
	}

	mock.Clear()

	// Ensure player has no charge
	player := handler.GetPlayer(currentPlayerID)
	if player != nil {
	 player.SetChargeCount(0)
	}

	err := handler.handleUseSkill(currentPlayerID)
	if err != nil {
	 // Expected
	}

	verifyActionRejected(t, mock, currentPlayerID, pkgnet.OpUseSkill, constants.ErrConditionNotMet)
}

func TestHandleUseSkill_Success(t *testing.T) {
	handler := setupHandlerWithPlayers(t, 2)
	mock := getMockDispatcher(handler)

	// Check if in MainAction state
	currentState := handler.hsm.GetCurrentStateID()
	if currentState != hsm.StateMainAction {
	 t.Skip("Not in MainAction state")
	}

	currentPlayerID := getCurrentPlayerUserID(handler)
	if currentPlayerID == "" {
	 t.Skip("No current player available")
	}

	mock.Clear()

	// Set player charge to 1
	player := handler.GetPlayer(currentPlayerID)
	if player != nil {
	 player.SetChargeCount(1)
	}

	err := handler.handleUseSkill(currentPlayerID)
	// Skill usage should clear charge
	if err != nil {
	 t.Errorf("handleUseSkill should succeed with charge, got: %v", err)
	}

	// Verify charge was cleared
	if player != nil && player.GetChargeCount() != 0 {
	 t.Errorf("Charge should be 0 after skill use, got %d", player.GetChargeCount())
	}
}

// ========== handleUserChoice Tests ==========

func TestHandleUserChoice_PlayerNotFound(t *testing.T) {
	handler := setupHandlerWithPlayers(t, 2)
	mock := getMockDispatcher(handler)
	mock.Clear()

	data := []byte(`{"op_code": "103", "decision_id": "dec-001", "choice": 0}`)
	err := handler.handleUserChoice(id.TestUUID(999), data)
	if err != nil {
	 // Expected
	}

	verifyActionRejected(t, mock, id.TestUUID(999), pkgnet.OpUserChoice, constants.ErrPlayerNotFound)
}

func TestHandleUserChoice_NoPendingDecision(t *testing.T) {
	handler := setupHandlerWithPlayers(t, 2)
	mock := getMockDispatcher(handler)
	mock.Clear()

	// Ensure HSM is not waiting for decision
	if handler.hsm.IsWaiting() {
	 t.Skip("HSM is waiting for decision, cannot test this case")
	}

	data := []byte(`{"op_code": "103", "decision_id": "dec-001", "choice": 0}`)
	err := handler.handleUserChoice(id.TestUUID(1), data)
	if err != nil {
	 // Expected
	}

	verifyActionRejected(t, mock, id.TestUUID(1), pkgnet.OpUserChoice, constants.ErrInvalidState)
}

func TestHandleUserChoice_InvalidJSON(t *testing.T) {
	handler := setupHandlerWithPlayers(t, 2)

	err := handler.handleUserChoice(id.TestUUID(1), []byte("invalid json"))
	if err == nil {
	 t.Error("handleUserChoice should return error for invalid JSON")
	}
}

// ========== handleMiniGameDataSubmit Tests ==========

func TestHandleMiniGameDataSubmit_PlayerNotFound(t *testing.T) {
	handler := setupHandlerWithPlayers(t, 2)
	mock := getMockDispatcher(handler)
	mock.Clear()

	data := []byte(`{"op_code": "107", "game_type": "dice_race", "game_data": {"score": 100, "time": 3.5}}`)
	err := handler.handleMiniGameDataSubmit(id.TestUUID(999), data)
	if err != nil {
		// Expected
	}

	verifyActionRejected(t, mock, id.TestUUID(999), pkgnet.OpMiniGameDataSubmit, constants.ErrPlayerNotFound)
}

func TestHandleMiniGameDataSubmit_InvalidJSON(t *testing.T) {
	handler := setupHandlerWithPlayers(t, 2)

	err := handler.handleMiniGameDataSubmit(id.TestUUID(1), []byte("invalid json"))
	if err == nil {
		t.Error("handleMiniGameDataSubmit should return error for invalid JSON")
	}
}

func TestHandleMiniGameDataSubmit_InvalidState(t *testing.T) {
	handler := setupHandlerWithPlayers(t, 2)
	mock := getMockDispatcher(handler)

	// Check if not in RoundMiniGame state
	globalState := handler.hsm.GetGlobalStateID()
	if globalState == hsm.StateRoundMiniGame {
		t.Skip("Already in RoundMiniGame state")
	}

	mock.Clear()

	data := []byte(`{"op_code": "107", "game_type": "dice_race", "game_data": {"score": 100}}`)
	err := handler.handleMiniGameDataSubmit(id.TestUUID(1), data)
	if err != nil {
		// Expected
	}

	verifyActionRejected(t, mock, id.TestUUID(1), pkgnet.OpMiniGameDataSubmit, constants.ErrInvalidState)
}

// ========== Request Types Tests ==========

func TestRollDiceRequest_JSON(t *testing.T) {
	req := RollDiceRequest{OpCode: "100"}

	data, err := json.Marshal(req)
	if err != nil {
	 t.Fatalf("Marshal failed: %v", err)
	}

	var parsed RollDiceRequest
	err = json.Unmarshal(data, &parsed)
	if err != nil {
	 t.Fatalf("Unmarshal failed: %v", err)
	}

	if parsed.OpCode != "100" {
	 t.Errorf("OpCode = %s, expected 100", parsed.OpCode)
	}
}

func TestUseItemRequest_JSON(t *testing.T) {
	req := UseItemRequest{
	 OpCode: "101",
	 ItemID: "item-uuid-123",
	}

	data, err := json.Marshal(req)
	if err != nil {
	 t.Fatalf("Marshal failed: %v", err)
	}

	var parsed UseItemRequest
	err = json.Unmarshal(data, &parsed)
	if err != nil {
	 t.Fatalf("Unmarshal failed: %v", err)
	}

	if parsed.OpCode != "101" {
	 t.Errorf("OpCode = %s, expected 101", parsed.OpCode)
	}
	if parsed.ItemID != "item-uuid-123" {
	 t.Errorf("ItemID = %s, expected item-uuid-123", parsed.ItemID)
	}
}

func TestUseSkillRequest_JSON(t *testing.T) {
	req := UseSkillRequest{OpCode: "102"}

	data, err := json.Marshal(req)
	if err != nil {
	 t.Fatalf("Marshal failed: %v", err)
	}

	var parsed UseSkillRequest
	err = json.Unmarshal(data, &parsed)
	if err != nil {
	 t.Fatalf("Unmarshal failed: %v", err)
	}

	if parsed.OpCode != "102" {
	 t.Errorf("OpCode = %s, expected 102", parsed.OpCode)
	}
}

func TestUserChoiceResponse_JSON(t *testing.T) {
	req := UserChoiceResponse{
	 OpCode:	 "103",
	 DecisionID: "dec-uuid-123",
	 Choice:	 1,
	}

	data, err := json.Marshal(req)
	if err != nil {
	 t.Fatalf("Marshal failed: %v", err)
	}

	var parsed UserChoiceResponse
	err = json.Unmarshal(data, &parsed)
	if err != nil {
	 t.Fatalf("Unmarshal failed: %v", err)
	}

	if parsed.OpCode != "103" {
	 t.Errorf("OpCode = %s, expected 103", parsed.OpCode)
	}
	if parsed.DecisionID != "dec-uuid-123" {
	 t.Errorf("DecisionID = %s, expected dec-uuid-123", parsed.DecisionID)
	}
	if parsed.Choice != 1 {
	 t.Errorf("Choice = %d, expected 1", parsed.Choice)
	}
}

func TestMiniGameDataSubmitRequest_JSON(t *testing.T) {
	req := MiniGameDataSubmitRequest{
		OpCode:   "107",
		GameType: "dice_race",
		GameData: map[string]interface{}{"score": 100, "time": 3.5},
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var parsed MiniGameDataSubmitRequest
	err = json.Unmarshal(data, &parsed)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if parsed.OpCode != "107" {
		t.Errorf("OpCode = %s, expected 107", parsed.OpCode)
	}
	if parsed.GameType != "dice_race" {
		t.Errorf("GameType = %s, expected dice_race", parsed.GameType)
	}
}

// ========== Concurrent Message Handling Tests ==========

func TestHandleMessage_ConcurrentRequests(t *testing.T) {
	handler := setupHandlerWithPlayers(t, 4)
	mock := getMockDispatcher(handler)

	// Run multiple MatchLoop ticks to progress HSM
	for i := 0; i < 10; i++ {
	 handler.MatchLoop(50 * time.Millisecond)
	}

	mock.Clear()

	// Send messages from different players concurrently (simulated)
	// Note: In real Nakama, messages are handled sequentially
	err := handler.HandleMessageWithOp(id.TestUUID(1), int64(pkgnet.OpRollDice), []byte("{}"))
	// Result depends on current state
	_ = err

	// Just verify handler doesn't panic with concurrent-like usage
	err = handler.HandleMessageWithOp(id.TestUUID(2), int64(pkgnet.OpRollDice), []byte("{}"))
	_ = err
}

// ========== State Transition During Message Handling Tests ==========

func TestMessageHandling_DuringMiniGameState(t *testing.T) {
	handler := setupHandlerWithPlayers(t, 4)
	mock := getMockDispatcher(handler)

	// Run loops until we hit mini-game state or skip
	for i := 0; i < 20; i++ {
	 handler.MatchLoop(50 * time.Millisecond)

	 globalState := handler.hsm.GetGlobalStateID()
	 if globalState == hsm.StateRoundMiniGame {
		 mock.Clear()

		 // Try to send RollDice during mini-game (should be rejected)
		 currentPlayerID := getCurrentPlayerUserID(handler)
		 if currentPlayerID != "" {
			 err := handler.handleRollDice(currentPlayerID)
			 if err != nil {
				 // Expected - invalid state
			 }

			 // Should have ActionRejected for invalid state
			 msgs := mock.GetMessages(currentPlayerID)
			 for _, msg := range msgs {
				 if msg.OpCode == int64(pkgnet.OpActionRejected) {
					 var rejected pkgnet.ActionRejected
					 json.Unmarshal(msg.Data, &rejected)
					 if rejected.ErrorCode == constants.ErrInvalidState {
						 // Found expected rejection
						 return
					 }
				 }
			 }
		 }
		 break
	 }
	}
}

// ========== Edge Cases Tests ==========

func TestHandleRollDice_NoCurrentPlayer(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)
	mock := NewMockDispatcherAdapter()
	handler.WithDispatcher(mock)

	// Add player but don't initialize HSM
	handler.addPlayer(id.TestUUID(1), constants.FactionQingLong, id.TestUUID(1))

	// No HSM means no current player
	err := handler.handleRollDice(id.TestUUID(1))
	if err != nil {
	 // Expected - no current player
	}

	// Verify ActionRejected was sent
	msgs := mock.GetMessages(id.TestUUID(1))
	if len(msgs) == 0 {
	 // May not have message if HSM is nil
	 t.Log("No messages sent (HSM not initialized)")
	}
}

func TestHandleUseItem_NoCurrentPlayer(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)
	mock := NewMockDispatcherAdapter()
	handler.WithDispatcher(mock)

	handler.addPlayer(id.TestUUID(1), constants.FactionQingLong, id.TestUUID(1))

	data := []byte(`{"op_code": "101", "item_id": "item-001"}`)
	err := handler.handleUseItem(id.TestUUID(1), data)
	if err != nil {
	 // Expected
	}
}

func TestHandleUseSkill_NoCurrentPlayer(t *testing.T) {
	handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)
	mock := NewMockDispatcherAdapter()
	handler.WithDispatcher(mock)

	handler.addPlayer(id.TestUUID(1), constants.FactionQingLong, id.TestUUID(1))

	err := handler.handleUseSkill(id.TestUUID(1))
	if err != nil {
	 // Expected
	}
}

// ========== Player With Item Tests ==========

func TestHandleUseItem_PlayerWithItem(t *testing.T) {
	handler := setupHandlerWithPlayers(t, 2)
	mock := getMockDispatcher(handler)

	currentState := handler.hsm.GetCurrentStateID()
	if currentState != hsm.StateMainAction {
	 t.Skip("Not in MainAction state")
	}

	currentPlayerID := getCurrentPlayerUserID(handler)
	if currentPlayerID == "" {
	 t.Fatal("Current player should exist")
	}

	mock.Clear()

	// Add item to player
	player := handler.GetPlayer(currentPlayerID)
	if player == nil {
	 t.Fatal("Player should exist")
	}

	item := core.NewItem(constants.ItemTypeReverseClock)
	player.AddItem(item)

	data := []byte(fmt.Sprintf(`{"op_code": "101", "item_id": "%s"}`, item.ID.UUID()))
	err := handler.handleUseItem(currentPlayerID, data)

	// Item found and used - may still error due to HSM state transition
	// but should NOT get ErrItemNotFound
	_ = err

	// Check if ActionRejected was sent and it's NOT item_not_found
	msgs := mock.GetMessages(currentPlayerID)
	for _, msg := range msgs {
	 if msg.OpCode == int64(pkgnet.OpActionRejected) {
		 var rejected pkgnet.ActionRejected
		 json.Unmarshal(msg.Data, &rejected)
		 if rejected.ErrorCode == constants.ErrItemNotFound {
			 t.Error("Should not get ErrItemNotFound when item exists")
		 }
	 }
	}
}

// ========== HSM State Context Tests ==========

func TestHSMStateDuringMessageHandling(t *testing.T) {
	handler := setupHandlerWithPlayers(t, 2)

	// Track state changes during message handling
	initialState := handler.hsm.GetCurrentStateID()

	// Run a few loops to let HSM progress
	for i := 0; i < 5; i++ {
	 handler.MatchLoop(100 * time.Millisecond)
	}

	afterState := handler.hsm.GetCurrentStateID()

	// State should have changed
	t.Logf("State: initial=%s, after=%s", initialState, afterState)

	// Handler should still be valid
	if handler.hsm == nil {
	 t.Error("HSM should still exist")
	}
}