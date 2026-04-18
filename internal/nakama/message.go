// Package nakama provides Nakama match handler implementation for Paradiced.
package nakama

import (
	"encoding/json"
	"strconv"

	"github.com/b1tAction/paradiced/internal/engine/hsm"
	"github.com/b1tAction/paradiced/internal/net"
	pkgnet "github.com/b1tAction/paradiced/pkg/net"
)

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
		return nil // Unknown player
	}

	// Check if player is current turn player
	if h.getCurrentPlayer() != player {
		return nil // Not current player's turn
	}

	// Check if in MainAction state
	currentState := h.hsm.GetCurrentStateID()
	if currentState != hsm.StateMainAction {
		return nil // Not in MainAction state
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
		return nil // Unknown player
	}

	// Check if player is current turn player
	if h.getCurrentPlayer() != player {
		return nil // Not current player's turn
	}

	// Check if in MainAction state
	if h.hsm.GetCurrentStateID() != hsm.StateMainAction {
		return nil // Not in MainAction state
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
		return nil // Item not found
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
		return nil // Unknown player
	}

	// Check if player is current turn player
	if h.getCurrentPlayer() != player {
		return nil // Not current player's turn
	}

	// Check if player has charge available
	if player.GetChargeCount() < 1 {
		return nil // No charge available
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
		return nil // Unknown player
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