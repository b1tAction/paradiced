package protocol

// Broadcast defines the client communication interface.
// Used by HSM and ActionContext to send sync messages to clients.
// Nakama implementation will implement this interface.
type Broadcast interface {
	// BroadcastStateSync broadcasts state sync to all players.
	// Called when HSM transitions to a new state.
	BroadcastStateSync(state interface{}) error

	// BroadcastTurnSync broadcasts turn action list to all players.
	// Called after executing turn effects.
	BroadcastTurnSync(turn interface{}) error

	// SendDecision sends a decision request to a specific player.
	// Called when HSM enters WaitDecision state.
	SendDecision(playerID string, decision interface{}) error

	// SendAvailable sends available actions to a specific player.
	// Called when entering MainAction state.
	SendAvailable(playerID string, available interface{}) error

	// BroadcastMiniGameStart broadcasts mini-game start notification.
	// Called when entering RoundMiniGame state.
	BroadcastMiniGameStart(start interface{}) error

	// BroadcastMiniGameResult broadcasts mini-game ranking results.
	// Called after mini-game completes.
	BroadcastMiniGameResult(result interface{}) error

	// BroadcastGameOver broadcasts game end notification.
	// Called when entering GameOver state.
	BroadcastGameOver(over interface{}) error

	// SendFullSync sends complete state to a reconnecting player.
	// Called when a player rejoins the match.
	SendFullSync(playerID string, state, turn interface{}) error
}