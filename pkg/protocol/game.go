package protocol

// Game defines the interface Game engine needs to expose.
// Used by ActionContext to access game state and players.
type Game interface {
	GetCurrentPlayer() Player
	GetPlayer(id string) Player
	GetPlayers() []Player
}