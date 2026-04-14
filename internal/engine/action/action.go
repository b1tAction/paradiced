package action

import (
	"github.com/b1tAction/paradiced/pkg/action"
	"github.com/b1tAction/paradiced/pkg/gamelog"
)

// ActionType is alias to pkg/action.ActionType for convenience.
type ActionType = action.ActionType

// ActionType constants (imported from pkg/action)
// Using snake_case string values directly.
const (
	ActionDamage     ActionType = action.ActionDamage
	ActionHeal       ActionType = action.ActionHeal
	ActionModifyLP   ActionType = action.ActionModifyLP
	ActionMove       ActionType = action.ActionMove
	ActionAddBuff    ActionType = action.ActionAddBuff
	ActionRemoveBuff ActionType = action.ActionRemoveBuff
	ActionRespawn    ActionType = action.ActionRespawn
	ActionSkipTurn   ActionType = action.ActionSkipTurn
	ActionDrawEvent  ActionType = action.ActionDrawEvent
	ActionTeleport   ActionType = action.ActionTeleport
	ActionStealBuff  ActionType = action.ActionStealBuff
	ActionFellDown   ActionType = action.ActionFellDown
	ActionUnknown    ActionType = action.ActionUnknown
)

// ExecutableAction extends pkg/action.Action with execution and logging capabilities.
// Concrete action types implement this interface.
type ExecutableAction interface {
	action.Action // Embed base Action interface

	// Execute performs the action on the game state.
	// Called after interception phase completes.
	Execute(ctx *ActionContext) error

	// LogEntry generates a game log entry for client animation.
	LogEntry() gamelog.LogEntry
}
