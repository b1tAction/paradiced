package action

import (
	"github.com/b1tAction/paradiced/pkg/constants"
	"github.com/b1tAction/paradiced/pkg/gamelog"
)

// Action is the standard payload for all game effects.
// Buffs/Items/Events generate Actions instead of directly modifying data.
// This allows interception (e.g., shields blocking damage) and logging for client animation.
//
// Design principle: Who produces the timing, who publishes the Phase.
// - Action publishes action timing Phases (PreTrigger, PostTrigger)
// - HSM publishes state timing Phases (BeforeTurn, OnLand, AfterTurn)
type Action interface {
	// Type returns the action type.
	Type() constants.ActionType

	// CanModify returns true if this action can be intercepted and modified.
	// Actions like Damage/Move can be modified, others like AddBuff might not.
	CanModify() bool

	// Source returns the source identifier (Buff ID, Item ID, Event ID, Faction ID).
	Source() string

	// Target returns the target player ID.
	Target() string

	// PreTriggerPhase returns the Phase to publish BEFORE action execution.
	// Used for interception (e.g., PhasePreDamage for shields/隐匿).
	// Returns PhaseAnyTime if no pre-trigger needed.
	PreTriggerPhase() constants.Phase

	// PostTriggerPhase returns the Phase to publish AFTER action execution.
	// Used for lifecycle events (e.g., PhasePostBuffApplied for buff entry effects).
	// Returns PhaseAnyTime if no post-trigger needed.
	PostTriggerPhase() constants.Phase

	// Execute performs the action on the game state.
	// Called after interception phase completes.
	Execute(ctx *ActionContext) error

	// LogEntry generates a game log entry for client animation.
	LogEntry() gamelog.LogEntry
}
