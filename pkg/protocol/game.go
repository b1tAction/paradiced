package protocol

import (
	"github.com/b1tAction/fated/pkg/gamelog"
	"github.com/b1tAction/fated/pkg/id"
)

// Game defines the interface Game engine needs to expose.
// Used by ActionContext to access game state and players.
// Note: Uses interface{} for Player return types to avoid circular dependency.
// Concrete implementations return their specific Player types.
type Game interface {
	GetCurrentPlayer() interface{}
	GetPlayer(id id.PlayerID) interface{}
	GetPlayers() []interface{}
	GetGameLog() *gamelog.GameLog // Get the global game log for playback
}
