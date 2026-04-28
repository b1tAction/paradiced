package protocol

import (
	"github.com/b1tAction/paradiced/pkg/gamelog"
	"github.com/b1tAction/paradiced/pkg/id"
)

// Game defines the interface Game engine needs to expose.
// Used by ActionContext to access game data (players, log).
// Note: Uses interface{} for Player return types to avoid circular dependency.
// Round/Turn state is managed by HSM, not exposed here.
// GetCurrentPlayer should be accessed via HSM.GetTurnPlayer().
type Game interface {
	GetPlayerInterface(id id.PlayerID) interface{}
	GetPlayersInterface() []interface{}
	GetGameLog() *gamelog.GameLog // Get the global game log for playback
}
