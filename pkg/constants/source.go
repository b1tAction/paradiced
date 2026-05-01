// Package constants provides unified enum type definitions.
package constants

// ActionSource defines the source of an action.
// Used to identify where the action originated (Buff, Item, Event, System, etc).
type ActionSource string

// ActionSource constants - snake_case values for JSON serialization.
const (
	// Buff sources
	SourceBuffDivine         ActionSource = "buff_divine"
	SourceBuffDivineRemoval  ActionSource = "buff_divine_removal"
	SourceBuffCurse          ActionSource = "buff_curse"
	SourceBuffCurseRemoval   ActionSource = "buff_curse_removal"
	SourceBuffRain           ActionSource = "buff_rain"
	SourceBuffCorrupt        ActionSource = "buff_corrupt"
	SourceBuffFire           ActionSource = "buff_fire"
	SourceBuffUndying        ActionSource = "buff_undying"
	SourceBuffExpiry         ActionSource = "buff_expiry"
	SourceBuffHidden         ActionSource = "buff_hidden"
	SourceBuffPoison         ActionSource = "buff_poison"
	SourceBuffThornsReflect  ActionSource = "buff_thorns"

	// Item sources
	SourceItemReverseClock     ActionSource = "item_reverse_clock"
	SourceItemReverseClockBuff ActionSource = "item_reverse_clock_buff"
	SourceItemAnyDoor          ActionSource = "item_any_door"
	SourceItemHealingPotion    ActionSource = "item_healing_potion"
	SourceItemDiceUpgrade      ActionSource = "item_dice_upgrade"
	SourceItemConsumed         ActionSource = "item_consumed"

	// Event sources
	SourceEventTrap        ActionSource = "event_trap"
	SourceEventHerb        ActionSource = "event_herb"
	SourceEventThunder     ActionSource = "event_thunder"
	SourceEventMilkTea     ActionSource = "event_milk_tea"
	SourceEventMosquito    ActionSource = "event_mosquito"
	SourceEventGhostHit    ActionSource = "event_ghost_hit"
	SourceEventDogPoop     ActionSource = "event_dog_poop"
	SourceEventRelic       ActionSource = "event_relic"
	SourceEventExchange    ActionSource = "event_exchange"
	SourceEventTasteTest   ActionSource = "event_taste_test"
	SourceEventThief       ActionSource = "event_thief"
	SourceEventDivineBless ActionSource = "event_divine_bless"
	SourceEventCurseBuddha ActionSource = "event_curse_buddha"
	SourceEventHiddenBuff  ActionSource = "event_hidden_buff"
	SourceEventLostWay     ActionSource = "event_lost_way"

	// Faction sources
	SourceFactionBaiHu  ActionSource = "faction_bai_hu"  // 劫运
	SourceFactionQingLong ActionSource = "faction_qing_long" // 行迹

	// System sources
	SourceSystemDice                ActionSource = "system_dice"
	SourceSystemDiceRoll            ActionSource = "system_dice_roll"
	SourceSystemDiceRollFellDown    ActionSource = "system_dice_roll_fell_down"
	SourceSystemDiceRollCheckpoint  ActionSource = "system_dice_roll_checkpoint"
	SourceSystemRespawn             ActionSource = "system_respawn"
	SourceSystemFell                ActionSource = "system_fell"
	SourceSystemCellDraw            ActionSource = "system_cell_draw"
	SourceSystemCheckpointTreasure  ActionSource = "system_checkpoint_treasure"
	SourceSystemPoisonBadEvent      ActionSource = "system_poison_bad_event"
	SourceSystemBossAttackRespawn   ActionSource = "system_boss_attack_respawn"
	SourceSystemBossSkillRespawn    ActionSource = "system_boss_skill_respawn"
	SourceSystemTurnEndRespawn      ActionSource = "system_turn_end_respawn"
	SourceDeathRespawn              ActionSource = "death_respawn"
	SourceFragileCell               ActionSource = "fragile_cell"

	// Boss sources
	SourceBossNormal       ActionSource = "boss_normal"       // Boss normal attack
	SourceBossCrit         ActionSource = "boss_crit"         // Boss critical attack
	SourceBossDamage       ActionSource = "boss_damage"       // Player attacks Boss
	SourceBossSkillThunder ActionSource = "boss_skill_thunder" // Boss thunder skill
	SourceBossSkillCurse   ActionSource = "boss_skill_curse"  // Boss curse skill
	SourceBossSkillLost    ActionSource = "boss_skill_lost"   // Boss lost skill
	SourceBossSkillRest    ActionSource = "boss_skill_rest"   // Boss rest skill
	SourceBossSkillThorns  ActionSource = "boss_skill_thorns" // Boss thorns skill
)

// IsValid checks if ActionSource is valid.
func (as ActionSource) IsValid() bool {
	return as != ""
}

// IsBuff checks if source is from Buff.
func (as ActionSource) IsBuff() bool {
	return len(as) > 5 && as[:5] == "buff_"
}

// IsItem checks if source is from Item.
func (as ActionSource) IsItem() bool {
	return len(as) > 5 && as[:5] == "item_"
}

// IsEvent checks if source is from Event.
func (as ActionSource) IsEvent() bool {
	return len(as) > 6 && as[:6] == "event_"
}

// IsFaction checks if source is from Faction.
func (as ActionSource) IsFaction() bool {
	return len(as) > 8 && as[:8] == "faction_"
}

// IsSystem checks if source is from System.
func (as ActionSource) IsSystem() bool {
	return len(as) > 7 && as[:7] == "system_" ||
		as == SourceDeathRespawn || as == SourceFragileCell
}

// IsBoss checks if source is from Boss.
func (as ActionSource) IsBoss() bool {
	return len(as) > 5 && as[:5] == "boss_"
}