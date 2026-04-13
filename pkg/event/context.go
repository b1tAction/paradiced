package event

import (
	"github.com/b1tAction/Fated/pkg/util"
)

// Context is the execution context containing all information when a Phase triggers.
// Note: To avoid circular dependencies, Player and GameEvent use interface{} type.
// The concrete type is provided by the caller at runtime.
type Context struct {
	Player     interface{} `json:"player"`     // The player triggering the Phase (concrete type determined by caller)
	GameEvent  interface{} `json:"game_event"` // Related game event (optional)
	GameState  *GameState  `json:"game_state"` // Game state (optional)
	Choice     int         `json:"choice"`     // User's selected option index
	*util.Metadata          `json:"metadata"`   // Type-safe dynamic data container (replaces Data interface{})
}

// GameState is a snapshot of game state (simplified, can be extended later).
type GameState struct {
	Round        int           `json:"round"`         // Current round
	Turn         int           `json:"turn"`          // Current turn
	CurrentPhase Phase         `json:"current_phase"` // Current Phase
	AllPlayers   []interface{} `json:"all_players"`   // All players
}

// NewContext creates a new context.
func NewContext(player interface{}) *Context {
	return &Context{
		Player:   player,
		Metadata: util.NewMetadata(),
	}
}

// WithEvent sets the game event.
func (c *Context) WithEvent(event interface{}) *Context {
	c.GameEvent = event
	return c
}

// WithState sets the game state.
func (c *Context) WithState(state *GameState) *Context {
	c.GameState = state
	return c
}

// WithData sets additional data (backward-compatible method, uses Metadata internally).
// Deprecated: Prefer using Set(key, value) or type-specific SetInt/SetString etc.
func (c *Context) WithData(data interface{}) *Context {
	c.Set("data", data)
	return c
}

// GetData gets additional data (backward-compatible method).
// Deprecated: Prefer using GetInt/GetString/Get based on data type.
func (c *Context) GetData() interface{} {
	if val, ok := c.Get("data"); ok {
		return val
	}
	return nil
}

// WithChoice sets the user's choice.
func (c *Context) WithChoice(choice int) *Context {
	c.Choice = choice
	return c
}

// Clone clones the context (used for testing or when independent copy is needed).
func (c *Context) Clone() *Context {
	return &Context{
		Player:     c.Player,
		GameEvent:  c.GameEvent,
		GameState:  c.GameState,
		Choice:     c.Choice,
		Metadata:   c.Metadata.Clone(),
	}
}