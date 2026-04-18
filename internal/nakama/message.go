// Package nakama provides Nakama match handler implementation for Paradiced.
package nakama

import (
	"encoding/json"
	"strconv"

	"github.com/b1tAction/paradiced/internal/engine/hsm"
	"github.com/b1tAction/paradiced/internal/net"
	pkgnet "github.com/b1tAction/paradiced/pkg/net"
)

// HandleMessageWithOp processes incoming messages using Nakama envelope opcode first.
// This is the primary path for realtime match data where opcode is provided by transport.
func (h *NakamaMatchHandler) HandleMessageWithOp(sender string, opCode int64, data []byte) error {
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
	// Parse message opcode first
	var base struct {
		OpCode string `json:"op_code"`
	}
	if err := json.Unmarshal(data, &base); err != nil {
		return err
	}

	// Route to appropriate handler
	opCode := base.OpCode
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
		// Unknown opcode, ignore
		return nil
	}
}

// handleRollDice handles dice roll request.
func (h *NakamaMatchHandler) handleRollDice(sender string) error {
	// Get current player
	player := h.GetPlayer(sender)
	if player == nil {
		// Unknown player - send rejection
		return h.sendActionRejected(sender, pkgnet.ActionRejected{
			OpCode:  pkgnet.OpRollDice,
			Reason:  "player_not_found",
			Message: "未知玩家",
		})
	}

	// Check if player is current turn player
	currentPlayer := h.getCurrentPlayer()
	if currentPlayer == nil {
		return h.sendActionRejected(sender, pkgnet.ActionRejected{
			OpCode:  pkgnet.OpRollDice,
			Reason:  "no_current_player",
			Message: "当前没有回合玩家",
		})
	}
	if player != currentPlayer {
		return h.sendActionRejected(sender, pkgnet.ActionRejected{
			OpCode:  pkgnet.OpRollDice,
			Reason:  "not_current_player",
			Message: "当前不是你的回合",
		})
	}

	// Check if in MainAction state
	currentState := h.hsm.GetCurrentStateID()
	if currentState != hsm.StateMainAction {
		return h.sendActionRejected(sender, pkgnet.ActionRejected{
			OpCode:  pkgnet.OpRollDice,
			Reason:  "invalid_state",
			Message: "当前状态不能掷骰子",
		})
	}

	// Roll dice using dice manager
	steps := h.diceMgr.RollSpecialDice(sender)

	// Create builder for context
	builder := net.NewBuilder(h.hsm)

	// Create state context for HSM
	ctx := hsm.NewStateContext().
		WithHSM(h.hsm).
		WithPlayer(player).
		WithBroadcast(NewNakamaBroadcastAdapter(h)).
		WithBuilder(builder)

	// Call HSM's OnRollDice method
	return h.hsm.OnRollDice(steps, ctx)
}

// handleUseItem handles item usage request.
func (h *NakamaMatchHandler) handleUseItem(sender string, data []byte) error {
	// Parse use item request
	var req UseItemRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return err
	}

	// Get current player
	player := h.GetPlayer(sender)
	if player == nil {
		return h.sendActionRejected(sender, pkgnet.ActionRejected{
			OpCode:  pkgnet.OpUseItem,
			Reason:  "player_not_found",
			Message: "未知玩家",
		})
	}

	// Check if player is current turn player
	currentPlayer := h.getCurrentPlayer()
	if currentPlayer == nil {
		return h.sendActionRejected(sender, pkgnet.ActionRejected{
			OpCode:  pkgnet.OpUseItem,
			Reason:  "no_current_player",
			Message: "当前没有回合玩家",
		})
	}
	if player != currentPlayer {
		return h.sendActionRejected(sender, pkgnet.ActionRejected{
			OpCode:  pkgnet.OpUseItem,
			Reason:  "not_current_player",
			Message: "当前不是你的回合",
		})
	}

	// Check if in MainAction state
	if h.hsm.GetCurrentStateID() != hsm.StateMainAction {
		return h.sendActionRejected(sender, pkgnet.ActionRejected{
			OpCode:  pkgnet.OpUseItem,
			Reason:  "invalid_state",
			Message: "当前状态不能使用道具",
		})
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
		return h.sendActionRejected(sender, pkgnet.ActionRejected{
			OpCode:  pkgnet.OpUseItem,
			Reason:  "item_not_found",
			Message: "道具不存在",
		})
	}

	// Create builder for context
	builder := net.NewBuilder(h.hsm)

	// Create state context for HSM
	ctx := hsm.NewStateContext().
		WithHSM(h.hsm).
		WithPlayer(player).
		WithBroadcast(NewNakamaBroadcastAdapter(h)).
		WithBuilder(builder)

	// Call HSM's OnUseItem method
	return h.hsm.OnUseItem(req.ItemID, ctx)
}

// handleUseSkill handles faction skill usage request.
func (h *NakamaMatchHandler) handleUseSkill(sender string) error {
	// Get current player
	player := h.GetPlayer(sender)
	if player == nil {
		return h.sendActionRejected(sender, pkgnet.ActionRejected{
			OpCode:  pkgnet.OpUseSkill,
			Reason:  "player_not_found",
			Message: "未知玩家",
		})
	}

	// Check if player is current turn player
	currentPlayer := h.getCurrentPlayer()
	if currentPlayer == nil {
		return h.sendActionRejected(sender, pkgnet.ActionRejected{
			OpCode:  pkgnet.OpUseSkill,
			Reason:  "no_current_player",
			Message: "当前没有回合玩家",
		})
	}
	if player != currentPlayer {
		return h.sendActionRejected(sender, pkgnet.ActionRejected{
			OpCode:  pkgnet.OpUseSkill,
			Reason:  "not_current_player",
			Message: "当前不是你的回合",
		})
	}

	// Check if player has charge available
	if player.GetChargeCount() < 1 {
		return h.sendActionRejected(sender, pkgnet.ActionRejected{
			OpCode:  pkgnet.OpUseSkill,
			Reason:  "skill_not_ready",
			Message: "技能充能不足",
		})
	}

	// Notify HSM about skill usage
	// h.hsm.OnUseSkill()

	// Clear charge after use
	player.SetChargeCount(0)

	return nil
}

// handleUserChoice handles user choice response.
func (h *NakamaMatchHandler) handleUserChoice(sender string, data []byte) error {
	// Parse user choice response
	var req UserChoiceResponse
	if err := json.Unmarshal(data, &req); err != nil {
		return err
	}

	// Get current player
	player := h.GetPlayer(sender)
	if player == nil {
		return h.sendActionRejected(sender, pkgnet.ActionRejected{
			OpCode:  pkgnet.OpUserChoice,
			Reason:  "player_not_found",
			Message: "未知玩家",
		})
	}

	// Create state context for HSM
	ctx := hsm.NewStateContext().
		WithHSM(h.hsm).
		WithPlayer(player)

	// Notify HSM about user choice
	h.hsm.OnUserChoice(req.Choice, ctx)

	_ = req.DecisionID // Placeholder - could be used for validation

	return nil
}

// handleMiniGameResult handles mini-game result submission.
func (h *NakamaMatchHandler) handleMiniGameResult(sender string, data []byte) error {
	// Parse mini-game result
	var req MiniGameResultSubmit
	if err := json.Unmarshal(data, &req); err != nil {
		return err
	}

	// Get player
	player := h.GetPlayer(sender)
	if player == nil {
		return nil // Unknown player
	}

	// Check if in RoundMiniGame state
	if h.hsm.GetGlobalStateID() != hsm.StateRoundMiniGame {
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
	return h.hsm.OnMiniGameResult(player.ID.UUID(), req.Rank, ctx)
}
