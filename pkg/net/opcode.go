// Package net provides network message protocol definitions for client-server communication.
// This package defines the message opcodes, data structures, and abstract handler interface
// for implementing authoritative server communication (e.g., Nakama Match Handler).
package net

// OpCode defines message operation codes.
// Server -> Client: 1-99
// Client -> Server: 100+
type OpCode int64

const (
	// ========== Server -> Client Messages ==========

	// OpStateSync broadcasts current game state when entering a new state.
	// Data: StateSync
	OpStateSync OpCode = 1

	// OpActionSync broadcasts action execution results for client rendering.
	// Data: ActionSync
	OpActionSync OpCode = 2

	// OpDecisionRequest requests user decision input (dice roll, item selection, etc).
	// Data: Decision
	OpDecisionRequest OpCode = 3

	// OpAvailable broadcasts available actions for current player (items, skills, dice type).
	// Data: Available
	OpAvailable OpCode = 4

	// OpMiniGameStart notifies all players that mini-game phase is starting.
	// Data: MiniGameStart
	OpMiniGameStart OpCode = 5

	// OpGameOver broadcasts game end with winner and statistics.
	// Data: GameOver
	OpGameOver OpCode = 6

	// OpFullSync sends complete game state for reconnecting players.
	// Data: StateSync (complete snapshot)
	OpFullSync OpCode = 7

	// ========== Client -> Server Messages ==========

	// OpRollDice requests dice roll calculation from server.
	// Data: RollDice (empty, server calculates based on player's dice type)
	OpRollDice OpCode = 100

	// OpUseItem requests item usage.
	// Data: UseItem
	OpUseItem OpCode = 101

	// OpUseSkill requests faction skill activation.
	// Data: UseSkill (empty, server checks player's faction and charge status)
	OpUseSkill OpCode = 102

	// OpUserChoice sends user decision response.
	// Data: UserChoice
	OpUserChoice OpCode = 103

	// OpMiniGameResult submits mini-game ranking result.
	// Data: MiniGameResult
	OpMiniGameResult OpCode = 104
)

// String returns the opcode name for logging and debugging.
func (op OpCode) String() string {
	names := map[OpCode]string{
		OpStateSync:       "state_sync",
		OpActionSync:      "action_sync",
		OpDecisionRequest: "decision_request",
		OpAvailable:       "available",
		OpMiniGameStart:   "mini_game_start",
		OpGameOver:        "game_over",
		OpFullSync:        "full_sync",
		OpRollDice:        "roll_dice",
		OpUseItem:         "use_item",
		OpUseSkill:        "use_skill",
		OpUserChoice:      "user_choice",
		OpMiniGameResult:  "mini_game_result",
	}
	if name, ok := names[op]; ok {
		return name
	}
	return "unknown"
}

// IsServerToClient returns true if this opcode is from server to client.
func (op OpCode) IsServerToClient() bool {
	return op >= 1 && op <= 99
}

// IsClientToServer returns true if this opcode is from client to server.
func (op OpCode) IsClientToServer() bool {
	return op >= 100
}