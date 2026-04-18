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

	// OpTurnSync broadcasts turn/phase action list for client rendering.
	// Data: TurnSync (contains Actions array)
	// Client loops through actions and plays animations sequentially.
	OpTurnSync OpCode = 2

	// OpDecisionRequest requests user decision input (dice roll, item selection, etc).
	// Data: Decision
	OpDecisionRequest OpCode = 3

	// OpAvailable broadcasts available actions for current player (items, skills, dice type).
	// Data: Available
	OpAvailable OpCode = 4

	// OpMiniGameStart notifies all players that mini-game phase is starting.
	// Data: MiniGameStart (includes participating player IDs)
	OpMiniGameStart OpCode = 5

	// OpMiniGameResult broadcasts mini-game ranking results.
	// Data: MiniGameResult (contains Rankings array)
	OpMiniGameResult OpCode = 6

	// OpGameOver broadcasts game end with winner and statistics.
	// Data: GameOver
	OpGameOver OpCode = 7

	// OpFullSync sends complete game state for reconnecting players.
	// Data: StateSync (complete snapshot) + TurnSync (current turn actions)
	OpFullSync OpCode = 8

	// OpActionRejected notifies client that their action was rejected.
	// Data: ActionRejected (includes reason and original opcode)
	OpActionRejected OpCode = 9

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

	// OpMiniGameResultSubmit submits mini-game ranking result (deprecated - server calculates).
	// Data: MiniGameResultSubmit
	OpMiniGameResultSubmit OpCode = 104
)

// String returns the opcode name for logging and debugging.
func (op OpCode) String() string {
	names := map[OpCode]string{
		OpStateSync:            "state_sync",
		OpTurnSync:             "turn_sync",
		OpDecisionRequest:      "decision_request",
		OpAvailable:            "available",
		OpMiniGameStart:        "mini_game_start",
		OpMiniGameResult:       "mini_game_result",
		OpGameOver:             "game_over",
		OpFullSync:             "full_sync",
		OpActionRejected:       "action_rejected",
		OpRollDice:             "roll_dice",
		OpUseItem:              "use_item",
		OpUseSkill:             "use_skill",
		OpUserChoice:           "user_choice",
		OpMiniGameResultSubmit: "mini_game_result_submit",
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