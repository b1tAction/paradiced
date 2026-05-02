// Package constants provides unified enum type definitions.
package constants

import "testing"

// ========== BuffType Tests ==========

func TestBuffTypeIsValid(t *testing.T) {
	validTypes := []BuffType{
		BuffTypeCurse, BuffTypeLost, BuffTypeCorrupt, BuffTypePoison,
		BuffTypeHidden, BuffTypeDivine, BuffTypeRain, BuffTypeExorcism, BuffTypeFire,
		BuffTypeThorns, BuffTypeDeathMark,
	}
	for _, bt := range validTypes {
		if !bt.IsValid() {
			t.Errorf("BuffType(%s).IsValid() should be true", bt)
		}
	}

	invalidTypes := []BuffType{BuffTypeNone, "invalid", ""}
	for _, bt := range invalidTypes {
		if bt.IsValid() {
			t.Errorf("BuffType(%s).IsValid() should be false", bt)
		}
	}
}

func TestBuffTypeIsPositive(t *testing.T) {
	positiveTypes := []BuffType{
		BuffTypeDivine, BuffTypeHidden, BuffTypeRain, BuffTypeExorcism, BuffTypeFire,
	}
	for _, bt := range positiveTypes {
		if !bt.IsPositive() {
			t.Errorf("BuffType(%s).IsPositive() should be true", bt)
		}
	}

	nonPositiveTypes := []BuffType{
		BuffTypeCurse, BuffTypeLost, BuffTypeCorrupt, BuffTypePoison,
		BuffTypeThorns, BuffTypeNone,
	}
	for _, bt := range nonPositiveTypes {
		if bt.IsPositive() {
			t.Errorf("BuffType(%s).IsPositive() should be false", bt)
		}
	}
}

func TestBuffTypeIsNegative(t *testing.T) {
	negativeTypes := []BuffType{
		BuffTypeCurse, BuffTypeLost, BuffTypeCorrupt, BuffTypePoison,
	}
	for _, bt := range negativeTypes {
		if !bt.IsNegative() {
			t.Errorf("BuffType(%s).IsNegative() should be true", bt)
		}
	}

	nonNegativeTypes := []BuffType{
		BuffTypeDivine, BuffTypeHidden, BuffTypeRain, BuffTypeExorcism, BuffTypeFire,
		BuffTypeThorns, BuffTypeNone,
	}
	for _, bt := range nonNegativeTypes {
		if bt.IsNegative() {
			t.Errorf("BuffType(%s).IsNegative() should be false", bt)
		}
	}
}

func TestBuffTypeIsBoss(t *testing.T) {
	bossTypes := []BuffType{BuffTypeThorns, BuffTypeDeathMark}
	for _, bt := range bossTypes {
		if !bt.IsBoss() {
			t.Errorf("BuffType(%s).IsBoss() should be true", bt)
		}
	}

	nonBossTypes := []BuffType{
		BuffTypeCurse, BuffTypeLost, BuffTypeCorrupt, BuffTypePoison,
		BuffTypeHidden, BuffTypeDivine, BuffTypeRain, BuffTypeExorcism, BuffTypeFire,
	}
	for _, bt := range nonBossTypes {
		if bt.IsBoss() {
			t.Errorf("BuffType(%s).IsBoss() should be false", bt)
		}
	}
}

func TestBuffTypeIsHidden(t *testing.T) {
	hiddenTypes := []BuffType{BuffTypeDeathMark}
	for _, bt := range hiddenTypes {
		if !bt.IsHidden() {
			t.Errorf("BuffType(%s).IsHidden() should be true", bt)
		}
	}

	visibleTypes := []BuffType{
		BuffTypeCurse, BuffTypeLost, BuffTypeCorrupt, BuffTypePoison,
		BuffTypeHidden, BuffTypeDivine, BuffTypeRain, BuffTypeExorcism, BuffTypeFire,
		BuffTypeThorns,
	}
	for _, bt := range visibleTypes {
		if bt.IsHidden() {
			t.Errorf("BuffType(%s).IsHidden() should be false", bt)
		}
	}
}

func TestBuffTypeIsDraw(t *testing.T) {
	// IsDraw = true means participates in lottery pool draws
	drawTypes := []BuffType{
		BuffTypeCurse, BuffTypeLost, BuffTypeCorrupt, BuffTypePoison,
		BuffTypeHidden, BuffTypeDivine, BuffTypeRain, BuffTypeExorcism,
	}
	for _, bt := range drawTypes {
		if !bt.IsDraw() {
			t.Errorf("BuffType(%s).IsDraw() should be true", bt)
		}
	}

	// IsDraw = false means excluded from lottery pools (IsBoss, IsHidden, or IsFaction)
	noDrawTypes := []BuffType{BuffTypeThorns, BuffTypeDeathMark, BuffTypeFire}
	for _, bt := range noDrawTypes {
		if bt.IsDraw() {
			t.Errorf("BuffType(%s).IsDraw() should be false", bt)
		}
	}
}

func TestBuffTypeIsFaction(t *testing.T) {
	factionTypes := []BuffType{BuffTypeFire}
	for _, bt := range factionTypes {
		if !bt.IsFaction() {
			t.Errorf("BuffType(%s).IsFaction() should be true", bt)
		}
	}

	nonFactionTypes := []BuffType{
		BuffTypeCurse, BuffTypeLost, BuffTypeCorrupt, BuffTypePoison,
		BuffTypeHidden, BuffTypeDivine, BuffTypeRain, BuffTypeExorcism,
		BuffTypeThorns, BuffTypeDeathMark,
	}
	for _, bt := range nonFactionTypes {
		if bt.IsFaction() {
			t.Errorf("BuffType(%s).IsFaction() should be false", bt)
		}
	}
}

// ========== EventType Tests ==========

func TestEventTypeIsValid(t *testing.T) {
	validTypes := []EventType{
		EventTypeHerb, EventTypeMilkTea, EventTypeRelic, EventTypeDivineBless,
		EventTypeExchange, EventTypeHiddenBuff, EventTypeTasteTest,
		EventTypeMosquito, EventTypeGhostHit, EventTypeDogPoop,
		EventTypeThief, EventTypeCurseBuddha, EventTypeLostWay, EventTypeThunder,
	}
	for _, et := range validTypes {
		if !et.IsValid() {
			t.Errorf("EventType(%s).IsValid() should be true", et)
		}
	}

	invalidTypes := []EventType{EventTypeNone, "invalid", ""}
	for _, et := range invalidTypes {
		if et.IsValid() {
			t.Errorf("EventType(%s).IsValid() should be false", et)
		}
	}
}

// ========== ItemType Tests ==========

func TestItemTypeIsValid(t *testing.T) {
	validTypes := []ItemType{
		ItemTypeReverseClock, ItemTypeAnyDoor, ItemTypeDiceUpgrade,
	}
	for _, it := range validTypes {
		if !it.IsValid() {
			t.Errorf("ItemType(%s).IsValid() should be true", it)
		}
	}

	invalidTypes := []ItemType{ItemTypeNone, "invalid", ""}
	for _, it := range invalidTypes {
		if it.IsValid() {
			t.Errorf("ItemType(%s).IsValid() should be false", it)
		}
	}
}

// ========== Phase Tests ==========

func TestPhaseIsValid(t *testing.T) {
	validPhases := []Phase{
		PhaseBeforeTurn, PhaseOnLand, PhaseAfterTurn,
		PhasePreDamage, PhasePreEvent, PhasePreMove, PhasePreRespawn,
		PhasePreBuffApplied, PhasePostBuffApplied,
		PhasePreBuffRemoved, PhasePostBuffRemoved,
		PhasePreAction, PhasePreDiceRoll,
		PhaseAnyTime, PhaseItemUsed,
	}
	for _, p := range validPhases {
		if !p.IsValid() {
			t.Errorf("Phase(%s).IsValid() should be true", p)
		}
	}

	invalidPhases := []Phase{"invalid", ""}
	for _, p := range invalidPhases {
		if p.IsValid() {
			t.Errorf("Phase(%s).IsValid() should be false", p)
		}
	}
}

func TestPhaseNeedsSubscription(t *testing.T) {
	// AnyTime does not need subscription
	if PhaseAnyTime.NeedsSubscription() {
		t.Error("PhaseAnyTime.NeedsSubscription() should be false")
	}

	// All other phases need subscription
	needsSubscription := []Phase{
		PhaseBeforeTurn, PhaseOnLand, PhaseAfterTurn,
		PhasePreDamage, PhasePreEvent, PhasePreMove, PhasePreRespawn,
		PhasePreBuffApplied, PhasePostBuffApplied,
		PhasePreBuffRemoved, PhasePostBuffRemoved,
		PhasePreAction, PhasePreDiceRoll, PhaseItemUsed,
	}
	for _, p := range needsSubscription {
		if !p.NeedsSubscription() {
			t.Errorf("Phase(%s).NeedsSubscription() should be true", p)
		}
	}
}

func TestPhaseIsHSMPublished(t *testing.T) {
	hsmPhases := []Phase{PhaseBeforeTurn, PhaseOnLand, PhaseAfterTurn}
	for _, p := range hsmPhases {
		if !p.IsHSMPublished() {
			t.Errorf("Phase(%s).IsHSMPublished() should be true", p)
		}
	}

	nonHSMPublished := []Phase{
		PhasePreDamage, PhasePreEvent, PhasePreMove, PhasePreRespawn,
		PhasePreBuffApplied, PhasePostBuffApplied,
		PhasePreBuffRemoved, PhasePostBuffRemoved, PhaseAnyTime, PhaseItemUsed,
	}
	for _, p := range nonHSMPublished {
		if p.IsHSMPublished() {
			t.Errorf("Phase(%s).IsHSMPublished() should be false", p)
		}
	}
}

func TestPhaseIsActionPublished(t *testing.T) {
	actionPhases := []Phase{
		PhasePreDamage, PhasePreEvent, PhasePreMove, PhasePreRespawn,
		PhasePreBuffApplied, PhasePostBuffApplied,
		PhasePreBuffRemoved, PhasePostBuffRemoved,
		PhasePreAction, PhasePreDiceRoll,
	}
	for _, p := range actionPhases {
		if !p.IsActionPublished() {
			t.Errorf("Phase(%s).IsActionPublished() should be true", p)
		}
	}

	nonActionPublished := []Phase{
		PhaseBeforeTurn, PhaseOnLand, PhaseAfterTurn, PhaseAnyTime, PhaseItemUsed,
	}
	for _, p := range nonActionPublished {
		if p.IsActionPublished() {
			t.Errorf("Phase(%s).IsActionPublished() should be false", p)
		}
	}
}

// ========== Faction Tests ==========

func TestFactionIsValid(t *testing.T) {
	validFactions := []Faction{
		FactionQingLong, FactionZhuQue, FactionBaiHu, FactionXuanWu,
	}
	for _, f := range validFactions {
		if !f.IsValid() {
			t.Errorf("Faction(%s).IsValid() should be true", f)
		}
	}

	invalidFactions := []Faction{FactionNone, "invalid", ""}
	for _, f := range invalidFactions {
		if f.IsValid() {
			t.Errorf("Faction(%s).IsValid() should be false", f)
		}
	}
}

func TestAllFactions(t *testing.T) {
	factions := AllFactions()
	if len(factions) != 4 {
		t.Errorf("AllFactions() returned %d factions, expected 4", len(factions))
	}

	expected := []Faction{FactionQingLong, FactionZhuQue, FactionBaiHu, FactionXuanWu}
	for i, f := range factions {
		if f != expected[i] {
			t.Errorf("AllFactions()[%d] = %s, expected %s", i, f, expected[i])
		}
	}
}

// ========== CellType Tests ==========

func TestCellTypeIsValid(t *testing.T) {
	validTypes := []CellType{
		CellTypeNormal, CellTypeFragile, CellTypeFog, CellTypeCheckpoint, CellTypeBoss,
	}
	for _, ct := range validTypes {
		if !ct.IsValid() {
			t.Errorf("CellType(%s).IsValid() should be true", ct)
		}
	}

	invalidTypes := []CellType{CellTypeNone, ""}
	for _, ct := range invalidTypes {
		if ct.IsValid() {
			t.Errorf("CellType(%s).IsValid() should be false", ct)
		}
	}
}

func TestCellTypeIsSpecial(t *testing.T) {
	specialTypes := []CellType{
		CellTypeFragile, CellTypeFog, CellTypeCheckpoint, CellTypeBoss,
	}
	for _, ct := range specialTypes {
		if !ct.IsSpecial() {
			t.Errorf("CellType(%s).IsSpecial() should be true", ct)
		}
	}

	nonSpecialTypes := []CellType{CellTypeNormal, CellTypeNone}
	for _, ct := range nonSpecialTypes {
		if ct.IsSpecial() {
			t.Errorf("CellType(%s).IsSpecial() should be false", ct)
		}
	}
}

// ========== StateID Tests ==========

func TestStateIDIsValid(t *testing.T) {
	validStates := []StateID{
		StateMatchInit, StateRoundMiniGame, StateRoundPrep, StateTurnLoop,
		StateBossBattle, StateGameOver,
		StateTurnUpkeep, StateMainAction, StateTurnMoving, StateTurnLanded,
		StateTurnEvent, StateTurnEnd,
		StateWaitDecision,
	}
	for _, sid := range validStates {
		if !sid.IsValid() {
			t.Errorf("StateID(%s).IsValid() should be true", sid)
		}
	}

	invalidStates := []StateID{StateNone, StateInvalid, ""}
	for _, sid := range invalidStates {
		if sid.IsValid() {
			t.Errorf("StateID(%s).IsValid() should be false", sid)
		}
	}
}

func TestStateIDIsGlobalState(t *testing.T) {
	globalStates := []StateID{
		StateMatchInit, StateRoundMiniGame, StateRoundPrep, StateTurnLoop,
		StateBossBattle, StateGameOver,
	}
	for _, sid := range globalStates {
		if !sid.IsGlobalState() {
			t.Errorf("StateID(%s).IsGlobalState() should be true", sid)
		}
	}

	nonGlobalStates := []StateID{
		StateTurnUpkeep, StateMainAction, StateTurnMoving, StateTurnLanded,
		StateTurnEvent, StateTurnEnd, StateWaitDecision,
		StateNone, StateInvalid,
	}
	for _, sid := range nonGlobalStates {
		if sid.IsGlobalState() {
			t.Errorf("StateID(%s).IsGlobalState() should be false", sid)
		}
	}
}

func TestStateIDIsTurnState(t *testing.T) {
	turnStates := []StateID{
		StateTurnUpkeep, StateMainAction, StateTurnMoving, StateTurnLanded,
		StateTurnEvent, StateTurnEnd,
	}
	for _, sid := range turnStates {
		if !sid.IsTurnState() {
			t.Errorf("StateID(%s).IsTurnState() should be true", sid)
		}
	}

	nonTurnStates := []StateID{
		StateMatchInit, StateRoundMiniGame, StateRoundPrep, StateTurnLoop,
		StateBossBattle, StateGameOver, StateWaitDecision,
		StateNone, StateInvalid,
	}
	for _, sid := range nonTurnStates {
		if sid.IsTurnState() {
			t.Errorf("StateID(%s).IsTurnState() should be false", sid)
		}
	}
}

func TestStateIDIsInterruptState(t *testing.T) {
	if !StateWaitDecision.IsInterruptState() {
		t.Error("StateWaitDecision.IsInterruptState() should be true")
	}

	nonInterruptStates := []StateID{
		StateMatchInit, StateRoundMiniGame, StateRoundPrep, StateTurnLoop,
		StateBossBattle, StateGameOver,
		StateTurnUpkeep, StateMainAction, StateTurnMoving, StateTurnLanded,
		StateTurnEvent, StateTurnEnd,
		StateNone, StateInvalid,
	}
	for _, sid := range nonInterruptStates {
		if sid.IsInterruptState() {
			t.Errorf("StateID(%s).IsInterruptState() should be false", sid)
		}
	}
}

func TestStateIDLayer(t *testing.T) {
	tests := []struct {
		state    StateID
		expected int
	}{
		{StateMatchInit, 1},
		{StateRoundMiniGame, 1},
		{StateRoundPrep, 1},
		{StateTurnLoop, 1},
		{StateBossBattle, 1},
		{StateGameOver, 1},
		{StateTurnUpkeep, 2},
		{StateMainAction, 2},
		{StateTurnMoving, 2},
		{StateTurnLanded, 2},
		{StateTurnEvent, 2},
		{StateTurnEnd, 2},
		{StateWaitDecision, 3},
		{StateNone, 0},
		{StateInvalid, 0},
	}
	for _, tt := range tests {
		result := tt.state.Layer()
		if result != tt.expected {
			t.Errorf("StateID(%s).Layer() = %d, expected %d", tt.state, result, tt.expected)
		}
	}
}

// ========== EntryType Tests ==========

func TestEntryTypeIsValid(t *testing.T) {
	validTypes := []EntryType{
		EntryTypeAction, EntryTypeState, EntryTypeMiniGame, EntryTypeBoss, EntryTypeDecision,
	}
	for _, et := range validTypes {
		if !et.IsValid() {
			t.Errorf("EntryType(%s).IsValid() should be true", et)
		}
	}

	invalidTypes := []EntryType{"invalid", ""}
	for _, et := range invalidTypes {
		if et.IsValid() {
			t.Errorf("EntryType(%s).IsValid() should be false", et)
		}
	}
}

// ========== ActionSource Tests ==========

func TestActionSourceIsValid(t *testing.T) {
	validSources := []ActionSource{
		SourceBuffDivine, SourceBuffDivineRemoval,
		SourceBuffCurse, SourceBuffCurseRemoval,
		SourceBuffRain, SourceBuffCorrupt, SourceBuffFire, SourceBuffUndying,
		SourceBuffExpiry, SourceBuffHidden, SourceBuffPoison,
		SourceBuffThornsReflect,
		SourceItemReverseClock, SourceItemReverseClockBuff, SourceItemAnyDoor,
		SourceItemHealingPotion, SourceItemDiceUpgrade, SourceItemConsumed,
		SourceEventTrap, SourceEventHerb, SourceEventThunder, SourceEventMilkTea,
		SourceEventMosquito, SourceEventGhostHit, SourceEventDogPoop,
		SourceEventRelic, SourceEventExchange, SourceEventTasteTest,
		SourceEventThief, SourceEventDivineBless, SourceEventCurseBuddha,
		SourceEventHiddenBuff, SourceEventLostWay,
		SourceFactionBaiHu, SourceFactionQingLong,
		SourceSystemDice, SourceSystemDiceRoll, SourceSystemDiceRollFellDown,
		SourceSystemDiceRollCheckpoint, SourceSystemRespawn, SourceSystemFell,
		SourceSystemCellDraw, SourceSystemCheckpointTreasure,
		SourceSystemPoisonBadEvent, SourceSystemBossAttackRespawn,
		SourceSystemBossSkillRespawn, SourceSystemTurnEndRespawn,
		SourceDeathRespawn, SourceFragileCell,
		SourceBossNormal, SourceBossCrit, SourceBossDamage,
		SourceBossSkillThunder, SourceBossSkillCurse, SourceBossSkillLost,
		SourceBossSkillRest, SourceBossSkillThorns,
	}
	for _, as := range validSources {
		if !as.IsValid() {
			t.Errorf("ActionSource(%s).IsValid() should be true", as)
		}
	}

	invalidSources := []ActionSource{""}
	for _, as := range invalidSources {
		if as.IsValid() {
			t.Errorf("ActionSource(%s).IsValid() should be false", as)
		}
	}
}

func TestActionSourceIsBuff(t *testing.T) {
	buffSources := []ActionSource{
		SourceBuffDivine, SourceBuffDivineRemoval,
		SourceBuffCurse, SourceBuffCurseRemoval,
		SourceBuffRain, SourceBuffCorrupt, SourceBuffFire, SourceBuffUndying,
		SourceBuffExpiry, SourceBuffHidden, SourceBuffPoison,
		SourceBuffThornsReflect,
	}
	for _, as := range buffSources {
		if !as.IsBuff() {
			t.Errorf("ActionSource(%s).IsBuff() should be true", as)
		}
	}

	nonBuffSources := []ActionSource{
		SourceItemReverseClock, SourceEventTrap, SourceFactionBaiHu,
		SourceSystemDice, SourceDeathRespawn,
	}
	for _, as := range nonBuffSources {
		if as.IsBuff() {
			t.Errorf("ActionSource(%s).IsBuff() should be false", as)
		}
	}
}

func TestActionSourceIsItem(t *testing.T) {
	itemSources := []ActionSource{
		SourceItemReverseClock, SourceItemReverseClockBuff, SourceItemAnyDoor,
		SourceItemHealingPotion, SourceItemDiceUpgrade, SourceItemConsumed,
	}
	for _, as := range itemSources {
		if !as.IsItem() {
			t.Errorf("ActionSource(%s).IsItem() should be true", as)
		}
	}

	nonItemSources := []ActionSource{
		SourceBuffDivine, SourceEventTrap, SourceFactionBaiHu, SourceSystemDice,
	}
	for _, as := range nonItemSources {
		if as.IsItem() {
			t.Errorf("ActionSource(%s).IsItem() should be false", as)
		}
	}
}

func TestActionSourceIsEvent(t *testing.T) {
	eventSources := []ActionSource{
		SourceEventTrap, SourceEventHerb, SourceEventThunder, SourceEventMilkTea,
		SourceEventMosquito, SourceEventGhostHit, SourceEventDogPoop,
		SourceEventRelic, SourceEventExchange, SourceEventTasteTest,
		SourceEventThief, SourceEventDivineBless, SourceEventCurseBuddha,
		SourceEventHiddenBuff, SourceEventLostWay,
	}
	for _, as := range eventSources {
		if !as.IsEvent() {
			t.Errorf("ActionSource(%s).IsEvent() should be true", as)
		}
	}

	nonEventSources := []ActionSource{
		SourceBuffDivine, SourceItemReverseClock, SourceFactionBaiHu, SourceSystemDice,
	}
	for _, as := range nonEventSources {
		if as.IsEvent() {
			t.Errorf("ActionSource(%s).IsEvent() should be false", as)
		}
	}
}

func TestActionSourceIsFaction(t *testing.T) {
	factionSources := []ActionSource{
		SourceFactionBaiHu, SourceFactionQingLong,
	}
	for _, as := range factionSources {
		if !as.IsFaction() {
			t.Errorf("ActionSource(%s).IsFaction() should be true", as)
		}
	}

	nonFactionSources := []ActionSource{
		SourceBuffDivine, SourceItemReverseClock, SourceEventTrap, SourceSystemDice,
	}
	for _, as := range nonFactionSources {
		if as.IsFaction() {
			t.Errorf("ActionSource(%s).IsFaction() should be false", as)
		}
	}
}

func TestActionSourceIsBoss(t *testing.T) {
	bossSources := []ActionSource{
		SourceBossNormal, SourceBossCrit, SourceBossDamage,
		SourceBossSkillThunder, SourceBossSkillCurse, SourceBossSkillLost,
		SourceBossSkillRest, SourceBossSkillThorns,
	}
	for _, as := range bossSources {
		if !as.IsBoss() {
			t.Errorf("ActionSource(%s).IsBoss() should be true", as)
		}
	}

	nonBossSources := []ActionSource{
		SourceBuffDivine, SourceBuffThornsReflect, SourceItemReverseClock, SourceEventTrap,
		SourceFactionBaiHu, SourceSystemDice,
	}
	for _, as := range nonBossSources {
		if as.IsBoss() {
			t.Errorf("ActionSource(%s).IsBoss() should be false", as)
		}
	}
}

// ========== BossType Tests ==========

func TestBossTypeIsValid(t *testing.T) {
	if !BossTypeBeast.IsValid() {
		t.Error("BossTypeBeast.IsValid() should be true")
	}

	invalidTypes := []BossType{"invalid", ""}
	for _, bt := range invalidTypes {
		if bt.IsValid() {
			t.Errorf("BossType(%s).IsValid() should be false", bt)
		}
	}
}

func TestBossSkillTypeIsValid(t *testing.T) {
	validTypes := []BossSkillType{
		BossSkillThunder, BossSkillCurse, BossSkillLost, BossSkillRest, BossSkillThorns,
	}
	for _, bst := range validTypes {
		if !bst.IsValid() {
			t.Errorf("BossSkillType(%s).IsValid() should be true", bst)
		}
	}

	invalidTypes := []BossSkillType{"invalid", ""}
	for _, bst := range invalidTypes {
		if bst.IsValid() {
			t.Errorf("BossSkillType(%s).IsValid() should be false", bst)
		}
	}
}

func TestParseBossSkillType(t *testing.T) {
	tests := []struct {
		input    string
		expected BossSkillType
	}{
		{"thunder", BossSkillThunder},
		{"curse", BossSkillCurse},
		{"lost", BossSkillLost},
		{"rest", BossSkillRest},
		{"thorns", BossSkillThorns},
		{"invalid", BossSkillType("invalid")},
	}
	for _, tt := range tests {
		result := ParseBossSkillType(tt.input)
		if result != tt.expected {
			t.Errorf("ParseBossSkillType(%s) = %s, want %s", tt.input, result, tt.expected)
		}
	}
}

func TestBossAttackTypeIsValid(t *testing.T) {
	validTypes := []BossAttackType{BossAttackNormal, BossAttackCrit}
	for _, bat := range validTypes {
		if !bat.IsValid() {
			t.Errorf("BossAttackType(%s).IsValid() should be true", bat)
		}
	}

	invalidTypes := []BossAttackType{"invalid", "", "skill"}
	for _, bat := range invalidTypes {
		if bat.IsValid() {
			t.Errorf("BossAttackType(%s).IsValid() should be false", bat)
		}
	}
}

// ========== DrawType Tests ==========

func TestDrawTypeIsValid(t *testing.T) {
	validTypes := []DrawType{DrawTypeNone, DrawTypeEvent, DrawTypeItem}
	for _, dt := range validTypes {
		if !dt.IsValid() {
			t.Errorf("DrawType(%s).IsValid() should be true", dt)
		}
	}

	invalidTypes := []DrawType{"invalid"}
	for _, dt := range invalidTypes {
		if dt.IsValid() {
			t.Errorf("DrawType(%s).IsValid() should be false", dt)
		}
	}
}

func TestParseDrawType(t *testing.T) {
	tests := []struct {
		input    string
		expected DrawType
	}{
		{"none", DrawTypeNone},
		{"None", DrawTypeNone},
		{"event", DrawTypeEvent},
		{"Event", DrawTypeEvent},
		{"item", DrawTypeItem},
		{"Item", DrawTypeItem},
		{"invalid", DrawTypeNone},
	}
	for _, tt := range tests {
		result := ParseDrawType(tt.input)
		if result != tt.expected {
			t.Errorf("ParseDrawType(%s) = %s, want %s", tt.input, result, tt.expected)
		}
	}
}
func TestActionSourceIsSystem(t *testing.T) {
	systemSources := []ActionSource{
		SourceSystemDice, SourceSystemDiceRoll, SourceSystemDiceRollFellDown,
		SourceSystemDiceRollCheckpoint, SourceSystemRespawn, SourceSystemFell,
		SourceSystemCellDraw, SourceSystemCheckpointTreasure,
		SourceSystemPoisonBadEvent, SourceSystemBossAttackRespawn,
		SourceSystemBossSkillRespawn, SourceSystemTurnEndRespawn,
		SourceDeathRespawn, SourceFragileCell,
	}
	for _, as := range systemSources {
		if !as.IsSystem() {
			t.Errorf("ActionSource(%s).IsSystem() should be true", as)
		}
	}

	nonSystemSources := []ActionSource{
		SourceBuffDivine, SourceItemReverseClock, SourceEventTrap, SourceFactionBaiHu,
	}
	for _, as := range nonSystemSources {
		if as.IsSystem() {
			t.Errorf("ActionSource(%s).IsSystem() should be false", as)
		}
	}
}

// ========== Evaluation Tests ==========

func TestEvaluationIsValid(t *testing.T) {
	validEvals := []Evaluation{0, 10, 25, 40, 50, 65, 70, 100}
	for _, e := range validEvals {
		if !e.IsValid() {
			t.Errorf("Evaluation(%d).IsValid() should be true", e)
		}
	}

	invalidEvals := []Evaluation{-1, 101, 200}
	for _, e := range invalidEvals {
		if e.IsValid() {
			t.Errorf("Evaluation(%d).IsValid() should be false", e)
		}
	}
}

func TestEvaluationGetCategory(t *testing.T) {
	tests := []struct {
		eval     Evaluation
		expected string
	}{
		{0, "bad"},
		{10, "bad"},
		{25, "bad"},
		{40, "bad"},
		{41, "neutral"},
		{50, "neutral"},
		{55, "neutral"},
		{65, "neutral"},
		{66, "good"},
		{70, "good"},
		{80, "good"},
		{90, "good"},
		{100, "good"},
	}
	for _, tt := range tests {
		result := tt.eval.GetCategory()
		if result != tt.expected {
			t.Errorf("Evaluation(%d).GetCategory() = %s, expected %s", tt.eval, result, tt.expected)
		}
	}
}

func TestEvaluationIsGood(t *testing.T) {
	goodEvals := []Evaluation{66, 70, 80, 90, 100}
	for _, e := range goodEvals {
		if !e.IsGood() {
			t.Errorf("Evaluation(%d).IsGood() should be true", e)
		}
	}

	notGoodEvals := []Evaluation{0, 40, 50, 65}
	for _, e := range notGoodEvals {
		if e.IsGood() {
			t.Errorf("Evaluation(%d).IsGood() should be false", e)
		}
	}
}

func TestEvaluationIsNeutral(t *testing.T) {
	neutralEvals := []Evaluation{41, 50, 55, 65}
	for _, e := range neutralEvals {
		if !e.IsNeutral() {
			t.Errorf("Evaluation(%d).IsNeutral() should be true", e)
		}
	}

	notNeutralEvals := []Evaluation{0, 40, 66, 100}
	for _, e := range notNeutralEvals {
		if e.IsNeutral() {
			t.Errorf("Evaluation(%d).IsNeutral() should be false", e)
		}
	}
}

func TestEvaluationIsBad(t *testing.T) {
	badEvals := []Evaluation{0, 10, 25, 40}
	for _, e := range badEvals {
		if !e.IsBad() {
			t.Errorf("Evaluation(%d).IsBad() should be true", e)
		}
	}

	notBadEvals := []Evaluation{41, 50, 66, 100}
	for _, e := range notBadEvals {
		if e.IsBad() {
			t.Errorf("Evaluation(%d).IsBad() should be false", e)
		}
	}
}

func TestEvaluationCompare(t *testing.T) {
	tests := []struct {
		e1       Evaluation
		e2       Evaluation
		expected int
	}{
		{100, 50, 1},
		{50, 100, -1},
		{50, 50, 0},
		{10, 40, -1},
		{80, 70, 1},
	}
	for _, tt := range tests {
		result := tt.e1.Compare(tt.e2)
		if result != tt.expected {
			t.Errorf("Evaluation(%d).Compare(%d) = %d, expected %d", tt.e1, tt.e2, result, tt.expected)
		}
	}
}

func TestEvaluationConstants(t *testing.T) {
	if EvaluationVeryBad != 10 {
		t.Errorf("EvaluationVeryBad = %d, expected 10", EvaluationVeryBad)
	}
	if EvaluationBad != 25 {
		t.Errorf("EvaluationBad = %d, expected 25", EvaluationBad)
	}
	if EvaluationMildBad != 35 {
		t.Errorf("EvaluationMildBad = %d, expected 35", EvaluationMildBad)
	}
	if EvaluationNeutral != 50 {
		t.Errorf("EvaluationNeutral = %d, expected 50", EvaluationNeutral)
	}
	if EvaluationMixed != 55 {
		t.Errorf("EvaluationMixed = %d, expected 55", EvaluationMixed)
	}
	if EvaluationMildGood != 70 {
		t.Errorf("EvaluationMildGood = %d, expected 70", EvaluationMildGood)
	}
	if EvaluationGood != 80 {
		t.Errorf("EvaluationGood = %d, expected 80", EvaluationGood)
	}
	if EvaluationVeryGood != 90 {
		t.Errorf("EvaluationVeryGood = %d, expected 90", EvaluationVeryGood)
	}
	if EvaluationExcellent != 100 {
		t.Errorf("EvaluationExcellent = %d, expected 100", EvaluationExcellent)
	}
}

func TestEvaluationThresholds(t *testing.T) {
	if EvaluationBadThreshold != 40 {
		t.Errorf("EvaluationBadThreshold = %d, expected 40", EvaluationBadThreshold)
	}
	if EvaluationNeutralThreshold != 65 {
		t.Errorf("EvaluationNeutralThreshold = %d, expected 65", EvaluationNeutralThreshold)
	}
	if EvaluationMin != 0 {
		t.Errorf("EvaluationMin = %d, expected 0", EvaluationMin)
	}
	if EvaluationMax != 100 {
		t.Errorf("EvaluationMax = %d, expected 100", EvaluationMax)
	}
}

// ========== ParseFaction Tests ==========

func TestParseFaction(t *testing.T) {
	tests := []struct {
		input    string
		expected Faction
	}{
		{"qing_long", FactionQingLong},
		{"zhu_que", FactionZhuQue},
		{"bai_hu", FactionBaiHu},
		{"xuan_wu", FactionXuanWu},
		{"invalid", FactionNone},
		{"", FactionNone},
		{"QING_LONG", FactionNone}, // case sensitive
	}
	for _, tt := range tests {
		result := ParseFaction(tt.input)
		if result != tt.expected {
			t.Errorf("ParseFaction(%s) = %s, want %s", tt.input, result, tt.expected)
		}
	}
}

func TestFactionGetChineseName(t *testing.T) {
	tests := []struct {
		faction  Faction
		expected string
	}{
		{FactionQingLong, "青龙"},
		{FactionZhuQue, "朱雀"},
		{FactionBaiHu, "白虎"},
		{FactionXuanWu, "玄武"},
		{FactionNone, "未知"},
		{"invalid", "未知"},
	}
	for _, tt := range tests {
		result := tt.faction.GetChineseName()
		if result != tt.expected {
			t.Errorf("Faction(%s).GetChineseName() = %s, want %s", tt.faction, result, tt.expected)
		}
	}
}

func TestFactionGetSkillName(t *testing.T) {
	tests := []struct {
		faction  Faction
		expected string
	}{
		{FactionQingLong, "行迹"},
		{FactionZhuQue, "离火"},
		{FactionBaiHu, "劫运"},
		{FactionXuanWu, "镇厄"},
		{FactionNone, "未知"},
		{"invalid", "未知"},
	}
	for _, tt := range tests {
		result := tt.faction.GetSkillName()
		if result != tt.expected {
			t.Errorf("Faction(%s).GetSkillName() = %s, want %s", tt.faction, result, tt.expected)
		}
	}
}

func TestFactionGetSkillDesc(t *testing.T) {
	tests := []struct {
		faction  Faction
		expected string
	}{
		{FactionQingLong, "每5回合获得充能，使用后1回合内无视负面地形"},
		{FactionZhuQue, "每4回合幸运值+1，最高不超过8点"},
		{FactionBaiHu, "反超其他玩家时随机从该玩家身上偷取一个Buff"},
		{FactionXuanWu, "每5回合获得充能，可以抵消一次任意恶性事件"},
		{FactionNone, "未知"},
		{"invalid", "未知"},
	}
	for _, tt := range tests {
		result := tt.faction.GetSkillDesc()
		if result != tt.expected {
			t.Errorf("Faction(%s).GetSkillDesc() = %s, want %s", tt.faction, result, tt.expected)
		}
	}
}

// ========== ParseBuffType Tests ==========

func TestParseBuffType(t *testing.T) {
	tests := []struct {
		input    string
		expected BuffType
	}{
		{"curse", BuffTypeCurse},
		{"lost", BuffTypeLost},
		{"corrupt", BuffTypeCorrupt},
		{"poison", BuffTypePoison},
		{"hidden", BuffTypeHidden},
		{"divine", BuffTypeDivine},
		{"rain", BuffTypeRain},
		{"exorcism", BuffTypeExorcism},
		{"fire", BuffTypeFire},
		{"invalid", BuffTypeNone},
		{"", BuffTypeNone},
	}
	for _, tt := range tests {
		result := ParseBuffType(tt.input)
		if result != tt.expected {
			t.Errorf("ParseBuffType(%s) = %s, want %s", tt.input, result, tt.expected)
		}
	}
}

// ========== ParseEventType Tests ==========

func TestParseEventType(t *testing.T) {
	tests := []struct {
		input    string
		expected EventType
	}{
		{"herb", EventTypeHerb},
		{"milk_tea", EventTypeMilkTea},
		{"relic", EventTypeRelic},
		{"divine_bless", EventTypeDivineBless},
		{"exchange", EventTypeExchange},
		{"hidden_buff", EventTypeHiddenBuff},
		{"taste_test", EventTypeTasteTest},
		{"mosquito", EventTypeMosquito},
		{"ghost_hit", EventTypeGhostHit},
		{"dog_poop", EventTypeDogPoop},
		{"thief", EventTypeThief},
		{"curse_buddha", EventTypeCurseBuddha},
		{"lost_way", EventTypeLostWay},
		{"thunder", EventTypeThunder},
		{"invalid", EventTypeNone},
		{"", EventTypeNone},
	}
	for _, tt := range tests {
		result := ParseEventType(tt.input)
		if result != tt.expected {
			t.Errorf("ParseEventType(%s) = %s, want %s", tt.input, result, tt.expected)
		}
	}
}

// ========== ParseItemType Tests ==========

func TestParseItemType(t *testing.T) {
	tests := []struct {
		input    string
		expected ItemType
	}{
		{"reverse_clock", ItemTypeReverseClock},
		{"any_door", ItemTypeAnyDoor},
		{"dice_swap", ItemTypeNone}, // DiceSwap removed, should parse as None
		{"dice_upgrade", ItemTypeDiceUpgrade},
		{"invalid", ItemTypeNone},
		{"", ItemTypeNone},
	}
	for _, tt := range tests {
		result := ParseItemType(tt.input)
		if result != tt.expected {
			t.Errorf("ParseItemType(%s) = %s, want %s", tt.input, result, tt.expected)
		}
	}
}

// ========== ParseCellType Tests ==========

func TestParseCellType(t *testing.T) {
	tests := []struct {
		input    string
		expected CellType
	}{
		{"normal", CellTypeNormal},
		{"fragile", CellTypeFragile},
		{"fog", CellTypeFog},
		{"checkpoint", CellTypeCheckpoint},
		{"boss", CellTypeBoss},
		{"invalid", CellTypeNone},
		{"", CellTypeNone},
	}
	for _, tt := range tests {
		result := ParseCellType(tt.input)
		if result != tt.expected {
			t.Errorf("ParseCellType(%s) = %s, want %s", tt.input, result, tt.expected)
		}
	}
}

// ========== ActionType Tests ==========

// ========== Definition Tests ==========

func TestEventDefinitionIsGood(t *testing.T) {
	tests := []struct {
		eval     Evaluation
		expected bool
	}{
		{EvaluationGood, true},
		{EvaluationVeryGood, true},
		{EvaluationMildGood, true},
		{EvaluationNeutral, false},
		{EvaluationBad, false},
	}
	for _, tt := range tests {
		def := &EventDefinition{Type: EventTypeHerb, Eval: tt.eval}
		if def.IsGood() != tt.expected {
			t.Errorf("EventDefinition.IsGood() with Eval=%d = %v, expected %v", tt.eval, def.IsGood(), tt.expected)
		}
	}
}

func TestEventDefinitionIsBad(t *testing.T) {
	tests := []struct {
		eval     Evaluation
		expected bool
	}{
		{EvaluationBad, true},
		{EvaluationVeryBad, true},
		{EvaluationMildBad, true},
		{EvaluationNeutral, false},
		{EvaluationGood, false},
	}
	for _, tt := range tests {
		def := &EventDefinition{Type: EventTypeMosquito, Eval: tt.eval}
		if def.IsBad() != tt.expected {
			t.Errorf("EventDefinition.IsBad() with Eval=%d = %v, expected %v", tt.eval, def.IsBad(), tt.expected)
		}
	}
}

func TestEventDefinitionIsNeutral(t *testing.T) {
	tests := []struct {
		eval     Evaluation
		expected bool
	}{
		{EvaluationNeutral, true},
		{EvaluationMixed, true},
		{EvaluationGood, false},
		{EvaluationBad, false},
	}
	for _, tt := range tests {
		def := &EventDefinition{Type: EventTypeExchange, Eval: tt.eval}
		if def.IsNeutral() != tt.expected {
			t.Errorf("EventDefinition.IsNeutral() with Eval=%d = %v, expected %v", tt.eval, def.IsNeutral(), tt.expected)
		}
	}
}

func TestBuffDefinitionIsPositive(t *testing.T) {
	tests := []struct {
		eval     Evaluation
		expected bool
	}{
		{EvaluationGood, true},
		{EvaluationVeryGood, true},
		{EvaluationMildGood, true},
		{EvaluationNeutral, false},
		{EvaluationBad, false},
	}
	for _, tt := range tests {
		def := &BuffDefinition{Type: BuffTypeDivine, Eval: tt.eval}
		if def.IsPositive() != tt.expected {
			t.Errorf("BuffDefinition.IsPositive() with Eval=%d = %v, expected %v", tt.eval, def.IsPositive(), tt.expected)
		}
	}
}

func TestBuffDefinitionIsNegative(t *testing.T) {
	tests := []struct {
		eval     Evaluation
		expected bool
	}{
		{EvaluationBad, true},
		{EvaluationVeryBad, true},
		{EvaluationMildBad, true},
		{EvaluationNeutral, false},
		{EvaluationGood, false},
	}
	for _, tt := range tests {
		def := &BuffDefinition{Type: BuffTypeCurse, Eval: tt.eval}
		if def.IsNegative() != tt.expected {
			t.Errorf("BuffDefinition.IsNegative() with Eval=%d = %v, expected %v", tt.eval, def.IsNegative(), tt.expected)
		}
	}
}

func TestActionTypeIsValid(t *testing.T) {
	validTypes := []ActionType{
		ActionDamage, ActionHeal, ActionModifyLP, ActionMove,
		ActionAddBuff, ActionRemoveBuff, ActionRespawn, ActionSkipTurn,
		ActionDrawEvent, ActionTeleport, ActionStealBuff, ActionFellDown,
		ActionDrawItem, ActionDeath, ActionBossDamage, ActionBossAttack,
		ActionBossSkill, ActionDiceRoll,
	}
	for _, at := range validTypes {
		if !at.IsValid() {
			t.Errorf("ActionType(%s).IsValid() should be true", at)
		}
	}

	invalidTypes := []ActionType{ActionUnknown, "invalid", ""}
	for _, at := range invalidTypes {
		if at.IsValid() {
			t.Errorf("ActionType(%s).IsValid() should be false", at)
		}
	}
}

func TestParseActionType(t *testing.T) {
	tests := []struct {
		input    string
		expected ActionType
	}{
		{"damage", ActionDamage},
		{"heal", ActionHeal},
		{"modify_lp", ActionModifyLP},
		{"move", ActionMove},
		{"add_buff", ActionAddBuff},
		{"remove_buff", ActionRemoveBuff},
		{"respawn", ActionRespawn},
		{"skip_turn", ActionSkipTurn},
		{"draw_event", ActionDrawEvent},
		{"teleport", ActionTeleport},
		{"steal_buff", ActionStealBuff},
		{"fell_down", ActionFellDown},
		{"draw_item", ActionDrawItem},
		{"death", ActionDeath},
		{"boss_damage", ActionBossDamage},
		{"boss_attack", ActionBossAttack},
		{"boss_skill", ActionBossSkill},
		{"dice_roll", ActionDiceRoll},
		{"invalid", ActionUnknown},
		{"", ActionUnknown},
	}
	for _, tt := range tests {
		result := ParseActionType(tt.input)
		if result != tt.expected {
			t.Errorf("ParseActionType(%s) = %s, want %s", tt.input, result, tt.expected)
		}
	}
}
