package action

import "github.com/b1tAction/Fated/pkg/event"

// ActionType identifies the type of action.
// All game effects (Buff/Item/Event/Faction) generate Actions with specific types.
type ActionType int

const (
	// ActionDamage represents HP reduction (can be intercepted by shields).
	ActionDamage ActionType = iota
	// ActionHeal represents HP restoration.
	ActionHeal
	// ActionModifyLP represents Luck Point modification (+1 or -1).
	ActionModifyLP
	// ActionMove represents player movement on map (can be intercepted by 迷途).
	ActionMove
	// ActionAddBuff represents adding a Buff to player.
	ActionAddBuff
	// ActionRemoveBuff represents removing a Buff from player.
	ActionRemoveBuff
	// ActionRespawn represents player respawn at checkpoint.
	ActionRespawn
	// ActionSkipTurn represents skipping current turn.
	ActionSkipTurn
	// ActionDrawEvent represents drawing a random event.
	ActionDrawEvent
	// ActionTeleport represents instant teleport to specific position.
	ActionTeleport
	// ActionStealBuff represents stealing a Buff from another player.
	ActionStealBuff
)

// String returns the action type name.
func (at ActionType) String() string {
	switch at {
	case ActionDamage:
		return "Damage"
	case ActionHeal:
		return "Heal"
	case ActionModifyLP:
		return "ModifyLP"
	case ActionMove:
		return "Move"
	case ActionAddBuff:
		return "AddBuff"
	case ActionRemoveBuff:
		return "RemoveBuff"
	case ActionRespawn:
		return "Respawn"
	case ActionSkipTurn:
		return "SkipTurn"
	case ActionDrawEvent:
		return "DrawEvent"
	case ActionTeleport:
		return "Teleport"
	case ActionStealBuff:
		return "StealBuff"
	default:
		return "Unknown"
	}
}

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
	PreTriggerPhase() event.Phase

	// PostTriggerPhase returns the Phase to publish AFTER action execution.
	// Used for lifecycle events (e.g., PhaseOnBuffApplied for buff entry effects).
	// Returns PhaseAnyTime if no post-trigger needed.
	PostTriggerPhase() event.Phase
}

// LogEntry is the interface for actions that generate event log entries.
// Implemented by concrete action types in internal/engine/action.
type LogEntry interface {
	// LogEntry generates an event log entry for client animation.
	LogEntry() TurnEventLogEntry
}

// TurnEventLogEntry represents a single event for client animation playback.
// Each entry corresponds to one Action execution result.
type TurnEventLogEntry struct {
	// Type is the event type name ("HPChange", "LPChange", "Move", "BuffAdd", etc.)
	Type string `json:"type"`
	// Target is the player ID affected by this event
	Target string `json:"target"`
	// Delta is the change amount (negative for damage/LP loss, positive for heal/LP gain)
	Delta int `json:"delta"`
	// Source is the source identifier (Buff ID, Item ID, Event ID)
	Source string `json:"source"`
	// Metadata contains additional data (path for Move, buffType for AddBuff, etc.)
	Metadata interface{} `json:"metadata,omitempty"`
}