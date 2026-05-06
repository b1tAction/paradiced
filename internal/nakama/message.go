// Package nakama provides Nakama match handler implementation for Paradiced.
package nakama

import (
	"encoding/json"
	"strconv"

	"github.com/b1tAction/paradiced/internal/engine/hsm"
	internalnet "github.com/b1tAction/paradiced/internal/net"
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
		return h.handleUseSkill(sender, data)
	case int64(pkgnet.OpUserChoice):
		return h.handleUserChoice(sender, data)
	case int64(pkgnet.OpKickPlayer):
		return h.handleKickPlayer(sender, data)
	case int64(pkgnet.OpMiniGameDataSubmit):
		return h.handleMiniGameDataSubmit(sender, data)
	case int64(pkgnet.OpStartGame):
		return h.handleStartGame(sender)
	case int64(pkgnet.OpRoundReady):
		return h.handleRoundReady(sender)
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
	OpCode   string `json:"op_code"`
	ItemID   string `json:"item_id"`
	TargetID string `json:"target_id"` // Target player UUID (for targeted items like ReverseClock, AnyDoor)
}

// UserChoiceResponse represents a user choice response.
type UserChoiceResponse struct {
	OpCode     string `json:"op_code"`
	DecisionID string `json:"decision_id"`
	Choice     int    `json:"choice"`
}

// MiniGameDataSubmitRequest represents mini-game data submission from client.
// Client submits game_data (not rank); server calculates ranking using RankCalculator.
type MiniGameDataSubmitRequest struct {
	OpCode   string                 `json:"op_code"`
	GameType string                 `json:"game_type"`
	GameData map[string]interface{} `json:"game_data"`
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
		return h.handleUseSkill(sender, data)
	case strconv.FormatInt(int64(pkgnet.OpUserChoice), 10):
		return h.handleUserChoice(sender, data)
	case strconv.FormatInt(int64(pkgnet.OpMiniGameDataSubmit), 10):
		return h.handleMiniGameDataSubmit(sender, data)
	case strconv.FormatInt(int64(pkgnet.OpRoundReady), 10):
		return h.handleRoundReady(sender)
	default:
		h.logWarn("Unknown opcode in payload", "sender", sender, "op_code", opCode)
		// Unknown opcode, ignore
		return nil
	}
}

// handleRollDice handles dice roll request.
// Steps are now calculated by RollDiceAction inside HSM, not by DiceManager.
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

	// Create builder for context
	builder := internalnet.NewBuilder(h.hsm)

	// Create state context for HSM
	ctx := hsm.NewStateContext().
		WithHSM(h.hsm).
		WithPlayer(player).
		WithBroadcast(NewNakamaBroadcastAdapter(h)).
		WithBuilder(builder)

	// Call HSM's OnRollDice method (no steps parameter - calculated internally by RollDiceAction)
	h.logDebug("handleRollDice: calling OnRollDice")
	err := h.hsm.OnRollDice(ctx)
	if err != nil {
		logger.logError("OpRollDice", sender, err)
		return err
	}

	// Check if state execution produced an error
	if ctx.Error != nil {
		logger.logError("OpRollDice", sender, ctx.Error)
		h.logError("handleRollDice: state execution failed", "sender", sender, "error", ctx.Error)
		return h.sendActionRejectedWithCode(sender, pkgnet.OpRollDice, ErrorCodeForError(ctx.Error), ctx.Error.Error())
	}

	logger.logResponse("OpRollDice", sender, "dice rolled successfully")
	return nil
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
	builder := internalnet.NewBuilder(h.hsm)

	// Create state context for HSM
	ctx := hsm.NewStateContext().
		WithHSM(h.hsm).
		WithPlayer(player).
		WithBroadcast(NewNakamaBroadcastAdapter(h)).
		WithBuilder(builder)

	// Propagate target_id to StateContext if provided (for targeted items like ReverseClock, AnyDoor)
	if req.TargetID != "" {
		ctx.SetString("use_item_target_id", req.TargetID)
		h.logDebug("handleUseItem: target specified", "target_id", req.TargetID)
	}

	// Call HSM's OnUseItem method
	h.logDebug("handleUseItem: calling OnUseItem")
	err := h.hsm.OnUseItem(req.ItemID, ctx)
	if err != nil {
		logger.logError("OpUseItem", sender, err)
		h.logError("handleUseItem: OnUseItem failed", "sender", sender, "error", err)
		return err
	}

	// Check if state execution produced an error
	if ctx.Error != nil {
		logger.logError("OpUseItem", sender, ctx.Error)
		h.logError("handleUseItem: state execution failed", "sender", sender, "error", ctx.Error)
		return h.sendActionRejectedWithCode(sender, pkgnet.OpUseItem, ErrorCodeForError(ctx.Error), ctx.Error.Error())
	}

	logger.logResponse("OpUseItem", sender, "item used successfully")
	return nil
}

// handleUseSkill handles faction skill usage request.
func (h *NakamaMatchHandler) handleUseSkill(sender string, data []byte) error {
	logger := NewLogger(h)
	logger.logRequest("handleUseSkill", sender, data)

	// Parse UseSkill request (may contain target_id for BaiHu)
	var req pkgnet.UseSkill
	if data != nil && len(data) > 0 {
		if err := json.Unmarshal(data, &req); err != nil {
			h.logError("handleUseSkill: failed to parse request", "sender", sender, "error", err)
			logger.logError("OpUseSkill", sender, err)
			return h.sendActionRejectedWithCode(sender, pkgnet.OpUseSkill, constants.ErrInvalidParameter, "Invalid request format")
		}
	}

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

	// ZhuQue has no charge-based skill
	faction := player.GetFaction()
	if faction == constants.FactionZhuQue {
		logger.logReject("OpUseSkill", sender, constants.ErrConditionNotMet, "faction_no_skill", "ZhuQue faction has no charge-based skill")
		return h.sendActionRejectedWithCode(sender, pkgnet.OpUseSkill, constants.ErrConditionNotMet, "ZhuQue faction has no charge-based skill")
	}

	// Check if player has charge available
	chargeCount := player.GetChargeCount()
	logger.logValidation(sender, "charge_available", chargeCount >= 1, "charge_count", chargeCount)

	if chargeCount < 1 {
		logger.logReject("OpUseSkill", sender, constants.ErrConditionNotMet, "skill_not_ready", "Skill charge not ready")
		return h.sendActionRejectedWithCode(sender, pkgnet.OpUseSkill, constants.ErrConditionNotMet, "Skill charge not ready")
	}

	// BaiHu requires target_id validation
	if faction == constants.FactionBaiHu {
		if req.TargetID == "" {
			logger.logReject("OpUseSkill", sender, constants.ErrInvalidParameter, "target_required", "BaiHu skill requires target player")
			return h.sendActionRejectedWithCode(sender, pkgnet.OpUseSkill, constants.ErrInvalidParameter, "BaiHu skill requires target player")
		}
		// Validate target exists by UUID
		game := h.hsm.GetGame()
		targetFound := false
		if game != nil {
			for _, p := range game.Players {
				if p.ID.UUID() == req.TargetID {
					targetFound = true
					break
				}
			}
		}
		if !targetFound {
			logger.logReject("OpUseSkill", sender, constants.ErrPlayerNotFound, "target_not_found", "Target player not found")
			return h.sendActionRejectedWithCode(sender, pkgnet.OpUseSkill, constants.ErrPlayerNotFound, "Target player not found")
		}
	}

	// Set target_id in StateContext for HSM OnUseSkill to read
	ctx := hsm.NewStateContext().
		WithHSM(h.hsm).
		WithPlayer(player).
		WithBroadcast(NewNakamaBroadcastAdapter(h))
	if req.TargetID != "" {
		ctx.SetString("use_skill_target_id", req.TargetID)
	}

	// Delegate to HSM.OnUseSkill for faction-specific buff application
	if err := h.hsm.OnUseSkill(ctx); err != nil {
		h.logError("handleUseSkill: OnUseSkill failed", "sender", sender, "error", err)
		logger.logError("OpUseSkill", sender, err)
		return h.sendActionRejectedWithCode(sender, pkgnet.OpUseSkill, constants.ErrHSMError, "Skill execution failed")
	}

	h.logInfo("handleUseSkill: skill used successfully", "sender", sender, "faction", faction)

	// Broadcast state sync to reflect buff application and charge change
	builder := internalnet.NewBuilder(h.hsm)
	broadcastCtx := hsm.NewStateContext().
		WithHSM(h.hsm).
		WithPlayer(player).
		WithBroadcast(NewNakamaBroadcastAdapter(h)).
		WithBuilder(builder)

	if broadcastCtx.Builder != nil {
		stateSync := broadcastCtx.Builder.BuildStateSync()
		broadcastCtx.Broadcast.BroadcastStateSync(stateSync)

		// Re-send Available to current player (still in MainAction state)
		diceType := broadcastCtx.GetDiceType(player.ID.UUID())
		broadcastCtx.Builder.SetDiceType(diceType.String())
		available := broadcastCtx.Builder.BuildAvailable()
		broadcastCtx.Broadcast.SendAvailable(player.ID.UUID(), available)
	}

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

	// Check if HSM is waiting for decision
	if !h.hsm.IsWaiting() {
		logger.logReject("OpUserChoice", sender, constants.ErrInvalidState, "no_pending_decision", "No pending decision")
		return h.sendActionRejectedWithCode(sender, pkgnet.OpUserChoice, constants.ErrInvalidState, "No pending decision")
	}

	// Create state context for HSM
	ctx := hsm.NewStateContext().
		WithHSM(h.hsm).
		WithPlayer(player)

	// Notify HSM about user choice
	h.logDebug("handleUserChoice: calling OnUserChoice", "choice", req.Choice)
	err := h.hsm.OnUserChoice(req.Choice, ctx)
	if err != nil {
		logger.logError("OpUserChoice", sender, err)
		h.logError("handleUserChoice: OnUserChoice failed", "sender", sender, "error", err)
		return h.sendActionRejectedWithCode(sender, pkgnet.OpUserChoice, constants.ErrInternal, err.Error())
	}

	// Check if state execution produced an error
	if ctx.Error != nil {
		logger.logError("OpUserChoice", sender, ctx.Error)
		h.logError("handleUserChoice: state execution failed", "sender", sender, "error", ctx.Error)
		return h.sendActionRejectedWithCode(sender, pkgnet.OpUserChoice, constants.ErrInternal, ctx.Error.Error())
	}

	logger.logResponse("OpUserChoice", sender, "choice submitted")
	_ = req.DecisionID // Placeholder - could be used for validation

	return nil
}

// handleMiniGameDataSubmit handles mini-game data submission from client.
// Client submits game_data (score/time etc), server calculates ranking via RankCalculator.
func (h *NakamaMatchHandler) handleMiniGameDataSubmit(sender string, data []byte) error {
	logger := NewLogger(h)
	logger.logRequest("handleMiniGameDataSubmit", sender, data)

	// Parse mini-game data submission
	var req MiniGameDataSubmitRequest
	if err := json.Unmarshal(data, &req); err != nil {
		h.logError("handleMiniGameDataSubmit: failed to parse request", "sender", sender, "error", err)
		logger.logError("OpMiniGameDataSubmit", sender, err)
		return err
	}

	h.logDebug("handleMiniGameDataSubmit: parsed request", "sender", sender, "game_type", req.GameType, "game_data_keys", len(req.GameData))

	// Get player
	player := h.GetPlayer(sender)
	if player == nil {
		logger.logValidation(sender, "player_exists", false, "sender", sender)
		logger.logReject("OpMiniGameDataSubmit", sender, constants.ErrPlayerNotFound, "player_not_found", "Unknown player")
		return h.sendActionRejectedWithCode(sender, pkgnet.OpMiniGameDataSubmit, constants.ErrPlayerNotFound, "Unknown player")
	}

	logger.logValidation(sender, "player_exists", true, "player_id", player.ID.UUID())

	// Check if in RoundMiniGame state
	globalState := h.hsm.GetGlobalStateID()
	logger.logState(sender, globalState.String(), hsm.StateRoundMiniGame.String())

	if globalState != hsm.StateRoundMiniGame {
		logger.logValidation(sender, "state_check", false, "global_state", globalState.String())
		logger.logReject("OpMiniGameDataSubmit", sender, constants.ErrInvalidState, "invalid_state", "Not in mini-game state")
		return h.sendActionRejectedWithCode(sender, pkgnet.OpMiniGameDataSubmit, constants.ErrInvalidState, "Not in mini-game state")
	}

	// Create builder for context
	builder := internalnet.NewBuilder(h.hsm)

	// Create state context for HSM
	ctx := hsm.NewStateContext().
		WithHSM(h.hsm).
		WithPlayer(player).
		WithBroadcast(NewNakamaBroadcastAdapter(h)).
		WithBuilder(builder)

	// Call HSM's OnMiniGameDataSubmit method
	h.logDebug("handleMiniGameDataSubmit: calling OnMiniGameDataSubmit", "sender", sender, "game_type", req.GameType)
	err := h.hsm.OnMiniGameDataSubmit(player.ID.UUID(), req.GameType, req.GameData, ctx)
	if err != nil {
		logger.logError("OpMiniGameDataSubmit", sender, err)
		h.logError("handleMiniGameDataSubmit: OnMiniGameDataSubmit failed", "sender", sender, "error", err)
		return h.sendActionRejectedWithCode(sender, pkgnet.OpMiniGameDataSubmit, constants.ErrInternal, err.Error())
	}

	// Check if state execution produced an error
	if ctx.Error != nil {
		logger.logError("OpMiniGameDataSubmit", sender, ctx.Error)
		h.logError("handleMiniGameDataSubmit: state execution failed", "sender", sender, "error", ctx.Error)
		return h.sendActionRejectedWithCode(sender, pkgnet.OpMiniGameDataSubmit, constants.ErrInternal, ctx.Error.Error())
	}

	// If all players submitted and rankings calculated, assign dice types.
	// Check if state transitioned to RoundPrep (all data received).
	if h.hsm.GetGlobalStateID() == hsm.StateRoundPrep {
		// Get player rank from context and assign dice
		rank := ctx.GetMiniGameRank(player.ID.UUID())
		h.diceMgr.AssignDice(sender, rank)
	}

	logger.logResponse("OpMiniGameDataSubmit", sender, "data submitted")
	return nil
}

// handleRoundReady handles client's round-ready signal.
// Called when client finishes rendering current round and is ready for next.
func (h *NakamaMatchHandler) handleRoundReady(sender string) error {
	logger := NewLogger(h)
	logger.logRequest("handleRoundReady", sender, nil)

	// Get player
	player := h.GetPlayer(sender)
	if player == nil {
		logger.logReject("OpRoundReady", sender, constants.ErrPlayerNotFound, "player_not_found", "Unknown player")
		return h.sendActionRejectedWithCode(sender, pkgnet.OpRoundReady, constants.ErrPlayerNotFound, "Unknown player")
	}

	// Check if in RoundEndWait state
	globalState := h.hsm.GetGlobalStateID()
	logger.logState(sender, globalState.String(), hsm.StateRoundEndWait.String())

	if globalState != hsm.StateRoundEndWait {
		logger.logReject("OpRoundReady", sender, constants.ErrInvalidState, "invalid_state", "Not in round-end wait state")
		return h.sendActionRejectedWithCode(sender, pkgnet.OpRoundReady, constants.ErrInvalidState, "Not in round-end wait state")
	}

	// Create builder for context
	builder := internalnet.NewBuilder(h.hsm)

	// Create state context for HSM
	ctx := hsm.NewStateContext().
		WithHSM(h.hsm).
		WithPlayer(player).
		WithBroadcast(NewNakamaBroadcastAdapter(h)).
		WithBuilder(builder)

	// Call HSM's OnRoundReady method
	err := h.hsm.OnRoundReady(player.ID.UUID(), ctx)
	if err != nil {
		logger.logError("OpRoundReady", sender, err)
		return err
	}

	// Check if state execution produced an error
	if ctx.Error != nil {
		logger.logError("OpRoundReady", sender, ctx.Error)
		return h.sendActionRejectedWithCode(sender, pkgnet.OpRoundReady, ErrorCodeForError(ctx.Error), ctx.Error.Error())
	}

	logger.logResponse("OpRoundReady", sender, "round ready signal received")
	return nil
}

// handleStartGame handles start game request from host.
func (h *NakamaMatchHandler) handleStartGame(sender string) error {
	logger := NewLogger(h)
	logger.logRequest("handleStartGame", sender, nil)

	// Validate: only host can start
	if sender != h.hostUserID {
		logger.logReject("OpStartGame", sender, constants.ErrConditionNotMet, "not_host", "Only host can start the game")
		return h.sendActionRejectedWithCode(sender, pkgnet.OpStartGame, constants.ErrConditionNotMet, "Only host can start the game")
	}

	// Validate: minimum 2 players
	playerCount := len(h.playerList)
	if playerCount < 2 {
		logger.logReject("OpStartGame", sender, constants.ErrConditionNotMet, "not_enough_players", "Need at least 2 players to start")
		return h.sendActionRejectedWithCode(sender, pkgnet.OpStartGame, constants.ErrConditionNotMet, "Need at least 2 players to start")
	}

	// Check if in WaitingForHost state
	globalState := h.hsm.GetGlobalStateID()
	logger.logState(sender, globalState.String(), hsm.StateWaitingForHost.String())

	if globalState != hsm.StateWaitingForHost {
		logger.logReject("OpStartGame", sender, constants.ErrInvalidState, "invalid_state", "Not in waiting state")
		return h.sendActionRejectedWithCode(sender, pkgnet.OpStartGame, constants.ErrInvalidState, "Not in waiting state")
	}

	h.logInfo("handleStartGame: host starting game", "host", sender, "player_count", playerCount)

	// Signal HSM to start game by setting the start flag in handler
	// This flag is persisted across MatchLoop ticks
	h.startRequested = true

	// Broadcast StartGameAck to all players with map configuration
	if h.mapConfig != nil {
		broadcastAdapter := NewNakamaBroadcastAdapter(h)
		definitions := internalnet.BuildDefinitionsConfig()
		ack := &pkgnet.StartGameAck{
			MapConfig:    *h.mapConfig,
			Definitions: definitions,
		}
		broadcastAdapter.BroadcastStartGameAck(ack)
		h.logInfo("handleStartGame: StartGameAck broadcasted", "map_length", h.mapConfig.Length, "cells", len(h.mapConfig.Cells))
	}

	logger.logResponse("OpStartGame", sender, "game starting")
	return nil
}

// handleKickPlayer handles host kick player request from waiting room.
// Only the host can kick, and only before the game starts (WaitingForHost state).
func (h *NakamaMatchHandler) handleKickPlayer(sender string, data []byte) error {
	logger := NewLogger(h)
	logger.logRequest("handleKickPlayer", sender, data)

	// Parse kick player request
	var req pkgnet.KickPlayerRequest
	if err := json.Unmarshal(data, &req); err != nil {
		h.logError("handleKickPlayer: failed to parse request", "sender", sender, "error", err)
		logger.logError("OpKickPlayer", sender, err)
		return h.sendActionRejectedWithCode(sender, pkgnet.OpKickPlayer, constants.ErrInvalidParameter, "Invalid request format")
	}

	// Validate: only host can kick
	if sender != h.hostUserID {
		logger.logReject("OpKickPlayer", sender, constants.ErrNotHost, "not_host", "Only host can kick players")
		return h.sendActionRejectedWithCode(sender, pkgnet.OpKickPlayer, constants.ErrNotHost, "Only host can kick players")
	}

	// Validate: host cannot kick self
	if req.TargetID == h.hostUserID {
		logger.logReject("OpKickPlayer", sender, constants.ErrInvalidParameter, "cannot_kick_self", "Host cannot kick themselves")
		return h.sendActionRejectedWithCode(sender, pkgnet.OpKickPlayer, constants.ErrInvalidParameter, "Host cannot kick themselves")
	}

	// Validate: must be in WaitingForHost state
	if h.hsm == nil || h.hsm.GetGlobalStateID() != hsm.StateWaitingForHost {
		logger.logReject("OpKickPlayer", sender, constants.ErrInvalidState, "invalid_state", "Can only kick in waiting room")
		return h.sendActionRejectedWithCode(sender, pkgnet.OpKickPlayer, constants.ErrInvalidState, "Can only kick in waiting room")
	}

	// Validate: target must exist in match
	targetPlayer := h.players[req.TargetID]
	if targetPlayer == nil {
		logger.logReject("OpKickPlayer", sender, constants.ErrPlayerNotFound, "target_not_found", "Target player not found")
		return h.sendActionRejectedWithCode(sender, pkgnet.OpKickPlayer, constants.ErrPlayerNotFound, "Target player not found")
	}

	h.logInfo("handleKickPlayer: host kicking player", "host", sender, "target", req.TargetID)

	// Remove player from game if game exists
	if h.hsm != nil {
		if game := h.hsm.GetGame(); game != nil {
			game.RemovePlayer(targetPlayer.ID)
		}
	}

	// Remove player from handler state
	delete(h.players, req.TargetID)
	delete(h.disconnected, req.TargetID)

	for i, id := range h.playerList {
		if id == req.TargetID {
			h.playerList = append(h.playerList[:i], h.playerList[i+1:]...)
			break
		}
	}

	// Send kicked notification to the target player
	h.sendActionRejectedWithCode(req.TargetID, pkgnet.OpKickPlayer, constants.ErrKickedByHost, "You have been kicked by the host")

	// Broadcast updated WaitingSync to remaining players
	h.broadcastWaitingSyncToAll()

	h.logInfo("handleKickPlayer: player kicked successfully", "target", req.TargetID, "remaining_players", len(h.playerList))

	logger.logResponse("OpKickPlayer", sender, "player kicked")
	return nil
}
