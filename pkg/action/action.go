package action

import (
	"github.com/b1tAction/paradiced/pkg/constants"
	"github.com/b1tAction/paradiced/pkg/gamelog"
)

// ActionType identifies the type of action using snake_case string naming.
// All game effects (Buff/Item/Event/Faction) generate Actions with specific types.
type ActionType string

const (
	// ActionDamage represents HP reduction (can be intercepted by shields).
	ActionDamage ActionType = "damage"
	// ActionHeal represents HP restoration.
	ActionHeal ActionType = "heal"
	// ActionModifyLP represents Luck Point modification (+1 or -1).
	ActionModifyLP ActionType = "modify_lp"
	// ActionMove represents player movement on map (can be intercepted by 迷途).
	ActionMove ActionType = "move"
	// ActionAddBuff represents adding a Buff to player.
	ActionAddBuff ActionType = "add_buff"
	// ActionRemoveBuff represents removing a Buff from player.
	ActionRemoveBuff ActionType = "remove_buff"
	// ActionRespawn represents player respawn at checkpoint.
	ActionRespawn ActionType = "respawn"
	// ActionSkipTurn represents skipping current turn.
	ActionSkipTurn ActionType = "skip_turn"
	// ActionDrawEvent represents drawing a random event.
	ActionDrawEvent ActionType = "draw_event"
	// ActionTeleport represents instant teleport to specific position.
	ActionTeleport ActionType = "teleport"
	// ActionStealBuff represents stealing a Buff from another player.
	ActionStealBuff ActionType = "steal_buff"
	// ActionFellDown represents player falling from Fragile cell.
	ActionFellDown ActionType = "fell_down"
	// ActionUnknown represents an unknown action type.
	ActionUnknown ActionType = "unknown"
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
	Type() ActionType

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
	// Used for lifecycle events (e.g., PhaseOnBuffApplied for buff entry effects).
	// Returns PhaseAnyTime if no post-trigger needed.
	PostTriggerPhase() constants.Phase
}

// LogEntry is the interface for actions that generate game log entries.
// Implemented by concrete action types in internal/engine/action.
// Returns gamelog.LogEntry for unified game playback.
type LogEntry interface {
	// LogEntry generates a game log entry for client animation.
	LogEntry() gamelog.LogEntry
}
