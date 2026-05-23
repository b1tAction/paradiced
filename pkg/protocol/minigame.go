package protocol

import (
	"time"

	"github.com/b1tAction/paradiced/pkg/constants"
	pkgnet "github.com/b1tAction/paradiced/pkg/net"
)

// OnlineMiniGameProvider abstracts mini-game service operations.
// Implementations connect to external game servers (e.g. Colyseus)
// to create rooms and provide connection info for clients.
//
// Usage: injected into RoundMiniGameState via WithProvider().
// When a MiniGameType.IsOnline() type is selected and provider is available,
// the HSM calls CreateRoom to establish an online game session.
// After the game completes, the mini-game service calls Nakama RPC,
// which sends a MatchSignal to the running match. MatchLoop then
// applies the rankings via OnMiniGameResult().
type OnlineMiniGameProvider interface {
	// CreateRoom creates a game room on the mini-game service.
	// gameType: which mini-game to play (must be IsOnline())
	// playerIDs: participating player UUIDs for identity mapping
	// Returns MiniGameConn with URL, RoomID, and per-player tokens for clients to join.
	CreateRoom(gameType constants.MiniGameType, playerIDs []string) (*pkgnet.MiniGameConn, error)

	// DestroyRoom cleans up a game room after result is received.
	// Called after Nakama receives the ranking callback from the mini-game service.
	DestroyRoom(roomID string) error

	// GetTimeout returns the maximum wait duration for this mini-game type.
	// Used by MatchLoop for fallback if no result received within timeout.
	GetTimeout(gameType constants.MiniGameType) time.Duration
}