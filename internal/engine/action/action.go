package action

import "github.com/b1tAction/Fated/pkg/action"

// ActionType is alias to pkg/action.ActionType for convenience.
type ActionType = action.ActionType

// TurnEventLogEntry is alias to pkg/action.TurnEventLogEntry for convenience.
type TurnEventLogEntry = action.TurnEventLogEntry

// ActionType constants (imported from pkg/action)
const (
	ActionDamage     = action.ActionDamage
	ActionHeal       = action.ActionHeal
	ActionModifyLP   = action.ActionModifyLP
	ActionMove       = action.ActionMove
	ActionAddBuff    = action.ActionAddBuff
	ActionRemoveBuff = action.ActionRemoveBuff
	ActionRespawn    = action.ActionRespawn
	ActionSkipTurn   = action.ActionSkipTurn
	ActionDrawEvent  = action.ActionDrawEvent
	ActionTeleport   = action.ActionTeleport
	ActionStealBuff  = action.ActionStealBuff
)

// ExecutableAction extends pkg/action.Action with execution and logging capabilities.
// Concrete action types implement this interface.
type ExecutableAction interface {
	action.Action // Embed base Action interface

	// Execute performs the action on the game state.
	// Called after interception phase completes.
	Execute(ctx *ActionContext) error

	// LogEntry generates an event log entry for client animation.
	LogEntry() TurnEventLogEntry
}