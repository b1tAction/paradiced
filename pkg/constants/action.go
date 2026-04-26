// Package constants provides unified enum type definitions.
package constants

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
	// ActionDrawItem represents drawing a random item (e.g. from CheckPoint treasure).
	ActionDrawItem ActionType = "draw_item"
	// ActionDeath represents player death event for client animation.
	ActionDeath ActionType = "death"
	// ActionBossDamage represents player attacking boss (damage to boss player).
	ActionBossDamage ActionType = "boss_damage"
	// ActionBossAttack represents boss attacking a player (normal/crit).
	ActionBossAttack ActionType = "boss_attack"
	// ActionBossSkill represents boss using a skill (AOE damage, buff, heal).
	ActionBossSkill ActionType = "boss_skill"
	// ActionDiceRoll represents dice roll result for client animation (interceptable via PhasePreDiceRoll).
	ActionDiceRoll ActionType = "dice_roll"
	// ActionUnknown represents an unknown action type.
	ActionUnknown ActionType = "unknown"
)

// IsValid checks if ActionType is valid.
func (at ActionType) IsValid() bool {
	switch at {
	case ActionDamage, ActionHeal, ActionModifyLP, ActionMove,
		ActionAddBuff, ActionRemoveBuff, ActionRespawn, ActionSkipTurn,
		ActionDrawEvent, ActionTeleport, ActionStealBuff, ActionFellDown,
		ActionDrawItem, ActionDeath, ActionBossDamage, ActionBossAttack,
		ActionBossSkill, ActionDiceRoll:
		return true
	default:
		return false
	}
}

// ParseActionType converts a string to ActionType.
// Returns ActionUnknown if the string is not a valid action type.
func ParseActionType(s string) ActionType {
	switch s {
	case "damage":
		return ActionDamage
	case "heal":
		return ActionHeal
	case "modify_lp":
		return ActionModifyLP
	case "move":
		return ActionMove
	case "add_buff":
		return ActionAddBuff
	case "remove_buff":
		return ActionRemoveBuff
	case "respawn":
		return ActionRespawn
	case "skip_turn":
		return ActionSkipTurn
	case "draw_event":
		return ActionDrawEvent
	case "teleport":
		return ActionTeleport
	case "steal_buff":
		return ActionStealBuff
	case "fell_down":
		return ActionFellDown
	case "draw_item":
		return ActionDrawItem
	case "death":
		return ActionDeath
	case "boss_damage":
		return ActionBossDamage
	case "boss_attack":
		return ActionBossAttack
	case "boss_skill":
		return ActionBossSkill
	case "dice_roll":
		return ActionDiceRoll
	default:
		return ActionUnknown
	}
}