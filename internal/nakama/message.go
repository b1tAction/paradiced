// Package nakama provides Nakama match handler implementation for Paradiced.
package nakama

import (
	"encoding/json"
	"strconv"

	"github.com/b1tAction/paradiced/internal/engine/hsm"
	"github.com/b1tAction/paradiced/internal/net"
	"github.com/b1tAction/paradiced/pkg/constants"
	pkgnet "github.com/b1tAction/paradiced/pkg/net"
)

// HandleMessageWithOp processes incoming messages using Nakama envelope opcode first.
// This is the primary path for realtime match data where opcode is provided by transport.
func (h *NakamaMatchHandler) HandleMessageWithOp(sender string, opCode int64, data []byte) error {
	h.logDebug("HandleMessageWithOp called", "sender", sender, "op_code", opCode, "data_len", len(data))

	switch opCode {
	case int64(pkgnet.OpRollDice):
		return h.handleRollDice(sender)
	case int64(pkgnet.OpUseItem):
		return h.handleUseItem(sender, data)
	case int64(pkgnet.OpUseSkill):
		return h.handleUseSkill(sender)
	case int64(pkgnet.OpUserChoice):
		return h.handleUserChoice(sender, data)
	case int64(pkgnet.OpMiniGameResultSubmit):
		return h.handleMiniGameResult(sender, data)
	default:
		h.logWarn("Unknown opcode received", "sender", sender, "op_code", opCode)
	}

	// Fallback to payload-based routing for compatibility with older message format.
	return h.HandleMessage(sender, data)
}

// Message types for client requests (not defined in pkg/net, define here)
// These are the request structures for client messages.

// Message types for client requests (not defined in pkg/net, define here)
// These are the request structures for client messages.

// RollDiceRequest represents a dice roll request from client.
type RollDiceRequest struct {
	OpCode string `json:"op_code"`
}

// UseItemRequest represents an item usage request from client.
type UseItemRequest struct {
	OpCode string `json:"op_code"`
	ItemID string `json:"item_id"`
}

// UseSkillRequest represents a faction skill usage request from client.
type UseSkillRequest struct {
	OpCode string `json:"op_code"`
}

// UserChoiceResponse represents a user choice response.
type UserChoiceResponse struct {
	OpCode     string `json:"op_code"`
	DecisionID string `json:"decision_id"`
	Choice     int    `json:"choice"`
}

// MiniGameResultSubmit represents mini-game result submission.
type MiniGameResultSubmit struct {
	OpCode string `json:"op_code"`
	Rank   int    `json:"rank"`
}

// HandleMessage processes incoming messages from clients.
// Called by Nakama when a player sends a message.
func (h *NakamaMatchHandler) HandleMessage(sender string, data []byte) error {
	h.logDebug("HandleMessage called (fallback routing)", "sender", sender, "data", string(data))

	// Parse message opcode first
	var base struct {
		OpCode string `json:"op_code"`
	}
	if err := json.Unmarshal(data, &base); err != nil {
		h.logError("Failed to parse message opcode", "sender", sender, "error", err)
		return err
	}

	// Route to appropriate handler
	opCode := base.OpCode
	h.logDebug("Rerouting by payload opcode", "sender", sender, "op_code", opCode)
	switch opCode {
	case strconv.FormatInt(int64(pkgnet.OpRollDice), 10):
		return h.handleRollDice(sender)
	case strconv.FormatInt(int64(pkgnet.OpUseItem), 10):
		return h.handleUseItem(sender, data)
	case strconv.FormatInt(int64(pkgnet.OpUseSkill), 10):
		return h.handleUseSkill(sender)
	case strconv.FormatInt(int64(pkgnet.OpUserChoice), 10):
		return h.handleUserChoice(sender, data)
	case strconv.FormatInt(int64(pkgnet.OpMiniGameResultSubmit), 10):
		return h.handleMiniGameResult(sender, data)
	default:
		h.logWarn("Unknown opcode in payload", "sender", sender, "op_code", opCode)
		// Unknown opcode, ignore
		return nil
	}
}

// handleRollDice handles dice roll request.
func (h *NakamaMatchHandler) handleRollDice(sender string) error {
	logger := NewLogger(h)
	logger.logRequest("handleRollDice", sender, nil)

	// Get current player
	player := h.GetPlayer(sender)
	if player == nil {
		logger.logReject("OpRollDice", sender, constants.ErrPlayerNotFound, "player_not_found", "Unknown player")
		return h.sendActionRejectedWithCode(sender, pkgnet.OpRollDice, constants.ErrPlayerNotFound, "Unknown player")
	}

	// Check if player is current turn player
	currentPlayer := h.getCurrentPlayer()
	if currentPlayer == nil {
		logger.logReject("OpRollDice", sender, constants.ErrInvalidState, "no_current_player", "No current player")
		return h.sendActionRejectedWithCode(sender, pkgnet.OpRollDice, constants.ErrInvalidState, "No current player")
	}

	logger.logPlayer(sender, "roll_dice", player.ID.UUID(), player == currentPlayer)

	if player != currentPlayer {
		logger.logReject("OpRollDice", sender, constants.ErrNotCurrentTurn, "not_current_player", "Not your turn")
		return h.sendActionRejectedWithCode(sender, pkgnet.OpRollDice, constants.ErrNotCurrentTurn, "Not your turn")
	}

	// Check if in MainAction state
	currentState := h.hsm.GetCurrentStateID()
	logger.logState(sender, currentState.String(), hsm.StateMainAction.String())

	if currentState != hsm.StateMainAction {
		logger.logReject("OpRollDice", sender, constants.ErrInvalidState, "invalid_state", "Cannot roll dice in current state")
		return h.sendActionRejectedWithCode(sender, pkgnet.OpRollDice, constants.ErrInvalidState, "Cannot roll dice in current state")
	}

	// Roll dice using dice manager
	steps := h.diceMgr.RollSpecialDice(sender)
	h.logInfo("handleRollDice: dice rolled", "steps", steps)

	// Create builder for context
	builder := net.NewBuilder(h.hsm)

	// Create state context for HSM
	ctx := hsm.NewStateContext().
		WithHSM(h.hsm).
		WithPlayer(player).
		WithBroadcast(NewNakamaBroadcastAdapter(h)).
		WithBuilder(builder)

	// Call HSM's OnRollDice method
	h.logDebug("handleRollDice: calling OnRollDice")
	err := h.hsm.OnRollDice(steps, ctx)
	if err != nil {
		logger.logError("OpRollDice", sender, err)
		return err
	}

	logger.logResponse("OpRollDice", sender, "dice rolled successfully")
	h.logDebug("handleRollDice: OnRollDice returned", "error", err)
	return err
}

// handleUseItem handles item usage request.
func (h *NakamaMatchHandler) handleUseItem(sender string, data []byte) error {
	logger := NewLogger(h)
	logger.logRequest("handleUseItem", sender, data)

	// Parse use item request
	var req UseItemRequest
	if err := json.Unmarshal(data, &req); err != nil {
		h.logError("handleUseItem: failed to parse request", "sender", sender, "error", err)
		logger.logError("OpUseItem", sender, err)
		return err
	}

	// Get current player
	player := h.GetPlayer(sender)
	if player == nil {
		logger.logReject("OpUseItem", sender, constants.ErrPlayerNotFound, "player_not_found", "Unknown player")
		return h.sendActionRejectedWithCode(sender, pkgnet.OpUseItem, constants.ErrPlayerNotFound, "Unknown player")
	}

	// Check if player is current turn player
	currentPlayer := h.getCurrentPlayer()
	if currentPlayer == nil {
		logger.logReject("OpUseItem", sender, constants.ErrInvalidState, "no_current_player", "No current player")
		return h.sendActionRejectedWithCode(sender, pkgnet.OpUseItem, constants.ErrInvalidState, "No current player")
	}

	logger.logPlayer(sender, "use_item", player.ID.UUID(), player == currentPlayer)

	if player != currentPlayer {
		logger.logReject("OpUseItem", sender, constants.ErrNotCurrentTurn, "not_current_player", "Not your turn")
		return h.sendActionRejectedWithCode(sender, pkgnet.OpUseItem, constants.ErrNotCurrentTurn, "Not your turn")
	}

	// Check if in MainAction state
	currentState := h.hsm.GetCurrentStateID()
	logger.logState(sender, currentState.String(), hsm.StateMainAction.String())

	if currentState != hsm.StateMainAction {
		logger.logReject("OpUseItem", sender, constants.ErrInvalidState, "invalid_state", "Cannot use item in current state")
		return h.sendActionRejectedWithCode(sender, pkgnet.OpUseItem, constants.ErrInvalidState, "Cannot use item in current state")
	}

	// Find item in player's inventory
	var found bool
	for _, item := range player.Inventory {
		if item.ID.UUID() == req.ItemID {
			found = true
			break
		}
	}

	if !found {
		logger.logValidation(sender, "item_exists", found, "item_id", req.ItemID)
		logger.logReject("OpUseItem", sender, constants.ErrItemNotFound, "item_not_found", "Item not found")
		return h.sendActionRejectedWithCode(sender, pkgnet.OpUseItem, constants.ErrItemNotFound, "Item not found")
	}

	logger.logValidation(sender, "item_exists", found, "item_id", req.ItemID)
	h.logDebug("handleUseItem: validation passed", "sender", sender, "item_id", req.ItemID)

	// Create builder for context
	builder := net.NewBuilder(h.hsm)

	// Create state context for HSM
	ctx := hsm.NewStateContext().
		WithHSM(h.hsm).
		WithPlayer(player).
		WithBroadcast(NewNakamaBroadcastAdapter(h)).
		WithBuilder(builder)

	// Call HSM's OnUseItem method
	h.logDebug("handleUseItem: calling OnUseItem")
	err := h.hsm.OnUseItem(req.ItemID, ctx)
	if err != nil {
		logger.logError("OpUseItem", sender, err)
		h.logError("handleUseItem: OnUseItem failed", "sender", sender, "error", err)
		return err
	}

	logger.logResponse("OpUseItem", sender, "item used successfully")
	return err
}

// handleUseSkill handles faction skill usage request.
func (h *NakamaMatchHandler) handleUseSkill(sender string) error {
	logger := NewLogger(h)
	logger.logRequest("handleUseSkill", sender, nil)

	// Get current player
	player := h.GetPlayer(sender)
	if player == nil {
		logger.logReject("OpUseSkill", sender, constants.ErrPlayerNotFound, "player_not_found", "Unknown player")
		return h.sendActionRejectedWithCode(sender, pkgnet.OpUseSkill, constants.ErrPlayerNotFound, "Unknown player")
	}

	// Check if player is current turn player
	currentPlayer := h.getCurrentPlayer()
	if currentPlayer == nil {
		logger.logReject("OpUseSkill", sender, constants.ErrInvalidState, "no_current_player", "No current player")
		return h.sendActionRejectedWithCode(sender, pkgnet.OpUseSkill, constants.ErrInvalidState, "No current player")
	}

	logger.logPlayer(sender, "use_skill", player.ID.UUID(), player == currentPlayer)

	if player != currentPlayer {
		logger.logReject("OpUseSkill", sender, constants.ErrNotCurrentTurn, "not_current_player", "Not your turn")
		return h.sendActionRejectedWithCode(sender, pkgnet.OpUseSkill, constants.ErrNotCurrentTurn, "Not your turn")
	}

	// Check if player has charge available
	chargeCount := player.GetChargeCount()
	logger.logValidation(sender, "charge_available", chargeCount >= 1, "charge_count", chargeCount)

	if chargeCount < 1 {
		logger.logReject("OpUseSkill", sender, constants.ErrConditionNotMet, "skill_not_ready", "Skill charge not ready")
		return h.sendActionRejectedWithCode(sender, pkgnet.OpUseSkill, constants.ErrConditionNotMet, "Skill charge not ready")
	}

	h.logInfo("handleUseSkill: clearing charge", "sender", sender)
	// Clear charge after use
	player.SetChargeCount(0)

	logger.logResponse("OpUseSkill", sender, "skill used successfully")
	return nil
}

// handleUserChoice handles user choice response.
func (h *NakamaMatchHandler) handleUserChoice(sender string, data []byte) error {
	logger := NewLogger(h)
	logger.logRequest("handleUserChoice", sender, data)

	// Parse user choice response
	var req UserChoiceResponse
	if err := json.Unmarshal(data, &req); err != nil {
		h.logError("handleUserChoice: failed to parse request", "sender", sender, "error", err)
		logger.logError("OpUserChoice", sender, err)
		return err
	}

	h.logDebug("handleUserChoice: parsed request", "sender", sender, "decision_id", req.DecisionID, "choice", req.Choice)

	// Get current player
	player := h.GetPlayer(sender)
	if player == nil {
		logger.logReject("OpUserChoice", sender, constants.ErrPlayerNotFound, "player_not_found", "Unknown player")
		return h.sendActionRejectedWithCode(sender, pkgnet.OpUserChoice, constants.ErrPlayerNotFound, "Unknown player")
	}

	// Create state context for HSM
	ctx := hsm.NewStateContext().
		WithHSM(h.hsm).
		WithPlayer(player)

	// Notify HSM about user choice
	h.logDebug("handleUserChoice: calling OnUserChoice", "choice", req.Choice)
	h.hsm.OnUserChoice(req.Choice, ctx)

	logger.logResponse("OpUserChoice", sender, "choice submitted")
	_ = req.DecisionID // Placeholder - could be used for validation

	return nil
}

// handleMiniGameResult handles mini-game result submission.
func (h *NakamaMatchHandler) handleMiniGameResult(sender string, data []byte) error {
	logger := NewLogger(h)
	logger.logRequest("handleMiniGameResult", sender, data)

	// Parse mini-game result
	var req MiniGameResultSubmit
	if err := json.Unmarshal(data, &req); err != nil {
		h.logError("handleMiniGameResult: failed to parse request", "sender", sender, "error", err)
		logger.logError("OpMiniGameResultSubmit", sender, err)
		return err
	}

	h.logDebug("handleMiniGameResult: parsed request", "sender", sender, "rank", req.Rank)

	// Get player
	player := h.GetPlayer(sender)
	if player == nil {
		logger.logValidation(sender, "player_exists", false, "sender", sender)
		h.logDebug("handleMiniGameResult: unknown player, ignoring", "sender", sender)
		return nil // Unknown player
	}

	logger.logValidation(sender, "player_exists", true, "player_id", player.ID.UUID())

	// Check if in RoundMiniGame state
	globalState := h.hsm.GetGlobalStateID()
	logger.logState(sender, globalState.String(), hsm.StateRoundMiniGame.String())

	if globalState != hsm.StateRoundMiniGame {
		logger.logValidation(sender, "state_check", false, "global_state", globalState.String())
		h.logWarn("handleMiniGameResult: not in mini-game state, ignoring", "sender", sender, "state", globalState.String())
		return nil // Not in mini-game state
	}

	// Create builder for context
	builder := net.NewBuilder(h.hsm)

	// Create state context for HSM
	ctx := hsm.NewStateContext().
		WithHSM(h.hsm).
		WithPlayer(player).
		WithBroadcast(NewNakamaBroadcastAdapter(h)).
		WithBuilder(builder)

	// Call HSM's OnMiniGameResult method
	h.logDebug("handleMiniGameResult: calling OnMiniGameResult", "sender", sender, "rank", req.Rank)
	err := h.hsm.OnMiniGameResult(player.ID.UUID(), req.Rank, ctx)
	if err != nil {
		logger.logError("OpMiniGameResultSubmit", sender, err)
		h.logError("handleMiniGameResult: OnMiniGameResult failed", "sender", sender, "error", err)
		return err
	}

	logger.logResponse("OpMiniGameResultSubmit", sender, "result submitted")
	return err
}
