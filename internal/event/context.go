package event

import (
	"github.com/b1tAction/paradiced/internal/core"
	"github.com/b1tAction/paradiced/pkg/constants"
	"github.com/b1tAction/paradiced/pkg/util"
)

// Context is the execution context containing all information when a Phase triggers.
type Context struct {
	Player         *core.Player     `json:"player"`          // Player triggering the Phase (concrete type)
	GameEvent      interface{}     `json:"game_event"`      // Related game event (optional)
	GameState      *GameState      `json:"game_state"`      // Game state (optional)
	Choice         int             `json:"choice"`          // User's selected option index
	DerivedActions []interface{}   `json:"derived_actions"` // Actions to execute after handler
	Errors         []error         `json:"errors"`          // Errors collected from handlers
	*util.Metadata `json:"metadata"` // Type-safe dynamic data container
}

// GameState is a snapshot of game state.
type GameState struct {
	Round        int             `json:"round"`
	Turn         int             `json:"turn"`
	CurrentPhase constants.Phase `json:"current_phase"`
	AllPlayers   []*core.Player  `json:"all_players"` // Concrete type
}

// NewContext creates a new context.
func NewContext(player *core.Player) *Context {
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

// WithData sets additional data.
func (c *Context) WithData(data interface{}) *Context {
	c.Set("data", data)
	return c
}

// GetData gets additional data.
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

// Clone clones the context.
func (c *Context) Clone() *Context {
	clonedErrors := make([]error, len(c.Errors))
	copy(clonedErrors, c.Errors)
	return &Context{
		Player:         c.Player,
		GameEvent:      c.GameEvent,
		GameState:      c.GameState,
		Choice:         c.Choice,
		DerivedActions: c.DerivedActions,
		Errors:         clonedErrors,
		Metadata:       c.Metadata.Clone(),
	}
}

// AddDerivedAction adds an Action to be executed after the current handler completes.
func (c *Context) AddDerivedAction(a interface{}) {
	c.DerivedActions = append(c.DerivedActions, a)
}

// ClearDerivedActions removes all derived actions.
func (c *Context) ClearDerivedActions() {
	c.DerivedActions = nil
}

// GetDerivedActions returns all derived actions.
func (c *Context) GetDerivedActions() []interface{} {
	return c.DerivedActions
}

// AddError adds an error to the error list.
func (c *Context) AddError(err error) {
	if err == nil {
		return
	}
	c.Errors = append(c.Errors, err)
}

// HasError returns true if any errors were recorded.
func (c *Context) HasError() bool {
	return len(c.Errors) > 0
}

// GetErrors returns all recorded errors.
func (c *Context) GetErrors() []error {
	return c.Errors
}

// FirstError returns the first error, or nil if no errors.
func (c *Context) FirstError() error {
	if len(c.Errors) == 0 {
		return nil
	}
	return c.Errors[0]
}

// ClearErrors removes all recorded errors.
func (c *Context) ClearErrors() {
	c.Errors = nil
}

// Clear resets the context to default values.
func (c *Context) Clear() {
	c.GameEvent = nil
	c.GameState = nil
	c.Choice = 0
	c.DerivedActions = nil
	c.Errors = nil
	c.Metadata.Clear()
}