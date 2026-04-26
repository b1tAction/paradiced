// Package player provides interactive player functionality for CLI.
package player

import (
	"context"

	"github.com/b1tAction/paradiced/internal/cli/model"
)

// PlayerUIAdapter defines the interface for player UI interactions.
// CLI implements this with simple text output.
// TUI can implement this with rich terminal UI frameworks like bubbletea or tview.
type PlayerUIAdapter interface {
	// OnStateSync displays game state update.
	// Called when server sends StateSync message.
	OnStateSync(ctx context.Context, state *model.StateSync)

	// OnAvailable prompts user to choose an action.
	// Called when server sends Available message (player's turn).
	// Returns the user's chosen action.
	OnAvailable(ctx context.Context, available *model.Available) PlayerAction

	// OnMiniGameStart prompts user for mini-game participation.
	// Called when server sends MiniGameStart message.
	// Returns the game_data the user wants to submit.
	OnMiniGameStart(ctx context.Context, start *model.MiniGameStart) map[string]interface{}

	// OnMiniGameResult displays mini-game result.
	// Called when server sends MiniGameResult message.
	OnMiniGameResult(ctx context.Context, result *model.MiniGameResult)

	// OnGameOver displays game over information.
	// Called when server sends GameOver message.
	OnGameOver(ctx context.Context, gameOver *model.GameOver)

	// OnActionRejected displays action rejection notification.
	// Called when server sends ActionRejected message.
	OnActionRejected(ctx context.Context, rejected *model.ActionRejected)

	// OnFullSync displays full sync (reconnection).
	// Called when server sends FullSync message.
	OnFullSync(ctx context.Context, state *model.StateSync)

	// OnError displays error message.
	OnError(err error)

	// OnWaiting displays waiting status.
	// Called during room waiting phase or other waiting scenarios.
	OnWaiting(message string, playerCount int, maxPlayers int)

	// OnConnected displays connection success.
	// Called when WebSocket connection is established.
	OnConnected(matchID string)

	// OnPlayerJoined displays player join notification.
	// Called when another player joins the room.
	OnPlayerJoined(userID string, faction string)

	// OnMatchCreated displays match creation success.
	// Called when match is created successfully.
	OnMatchCreated(matchID string)

	// OnWaitingSync displays waiting room status for the host.
	// Called when server sends WaitingSync message (host sees player list and can start).
	// Returns true if host wants to start the game, false to keep waiting.
	OnWaitingSync(ctx context.Context, waiting *model.WaitingSync) bool

	// OnStartGameAck displays game start acknowledgment with map configuration.
	// Called when server broadcasts StartGameAck after host starts the game.
	OnStartGameAck(ctx context.Context, ack *model.StartGameAck)

	// Clear clears the UI state (for reconnection/reset).
	Clear()
}

// ViewStatusAdapter provides interface for viewing game status.
// This is a subset of PlayerUIAdapter for status-only operations.
type ViewStatusAdapter interface {
	// GetCurrentState returns the current game state.
	GetCurrentState() *model.StateSync

	// GetMyPlayerID returns the current player's game ID.
	GetMyPlayerID() string

	// GetUserID returns the Nakama user ID.
	GetUserID() string
}