// Package constants provides unified enum type definitions.
package constants

import "testing"

// ========== BuffType Tests ==========

func TestBuffTypeIsValid(t *testing.T) {
	validTypes := []BuffType{
		BuffTypeCurse, BuffTypeLost, BuffTypeCorrupt, BuffTypePoison,
		BuffTypeHidden, BuffTypeDivine, BuffTypeRain, BuffTypeExorcism, BuffTypeFire,
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
		BuffTypeNone,
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
		BuffTypeNone,
	}
	for _, bt := range nonNegativeTypes {
		if bt.IsNegative() {
			t.Errorf("BuffType(%s).IsNegative() should be false", bt)
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
		ItemTypeReverseClock, ItemTypeAnyDoor, ItemTypeDiceSwap, ItemTypeDiceUpgrade,
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
		PhaseOnBuffApplied, PhaseOnBuffRemoved,
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
		PhaseOnBuffApplied, PhaseOnBuffRemoved, PhaseItemUsed,
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
		PhaseOnBuffApplied, PhaseOnBuffRemoved, PhaseAnyTime, PhaseItemUsed,
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
		PhaseOnBuffApplied, PhaseOnBuffRemoved,
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
		SourceBuffDivine, SourceBuffCurse, SourceBuffRain, SourceBuffCorrupt,
		SourceItemReverseClock, SourceItemAnyDoor,
		SourceEventTrap, SourceEventHerb,
		SourceFactionBaiHu, SourceFactionQingLong,
		SourceSystemDice, SourceSystemRespawn,
		SourceDeathRespawn, SourceFragileCell,
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
		SourceBuffDivine, SourceBuffCurse, SourceBuffRain, SourceBuffCorrupt,
		SourceBuffFire, SourceBuffUndying,
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
		SourceItemReverseClock, SourceItemAnyDoor, SourceItemHealingPotion,
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

func TestActionSourceIsSystem(t *testing.T) {
	systemSources := []ActionSource{
		SourceSystemDice, SourceSystemRespawn, SourceSystemFell,
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

// ========== SpecialEffect Tests ==========

func TestSpecialEffectIsValid(t *testing.T) {
	validEffects := []SpecialEffect{
		SpecialImmune, SpecialReverse, SpecialImmunePoison, SpecialBadEvent,
		SpecialZhuQuePassive, SpecialTeleport, SpecialDiceSwap, SpecialDiceUpgrade,
		SpecialGiveLost, SpecialDrawItem, SpecialLoseItem, SpecialSwapPosition,
		SpecialRandomBuff,
	}
	for _, se := range validEffects {
		if !se.IsValid() {
			t.Errorf("SpecialEffect(%s).IsValid() should be true", se)
		}
	}

	invalidEffects := []SpecialEffect{SpecialNone, ""}
	for _, se := range invalidEffects {
		if se.IsValid() {
			t.Errorf("SpecialEffect(%s).IsValid() should be false", se)
		}
	}
}

func TestSpecialEffectIsBuffEffect(t *testing.T) {
	buffEffects := []SpecialEffect{
		SpecialImmune, SpecialReverse, SpecialImmunePoison, SpecialBadEvent,
		SpecialZhuQuePassive,
	}
	for _, se := range buffEffects {
		if !se.IsBuffEffect() {
			t.Errorf("SpecialEffect(%s).IsBuffEffect() should be true", se)
		}
	}

	nonBuffEffects := []SpecialEffect{
		SpecialTeleport, SpecialDiceSwap, SpecialDiceUpgrade, SpecialGiveLost,
		SpecialDrawItem, SpecialLoseItem, SpecialSwapPosition, SpecialRandomBuff,
		SpecialNone,
	}
	for _, se := range nonBuffEffects {
		if se.IsBuffEffect() {
			t.Errorf("SpecialEffect(%s).IsBuffEffect() should be false", se)
		}
	}
}

func TestSpecialEffectIsItemEffect(t *testing.T) {
	itemEffects := []SpecialEffect{
		SpecialTeleport, SpecialDiceSwap, SpecialDiceUpgrade, SpecialGiveLost,
	}
	for _, se := range itemEffects {
		if !se.IsItemEffect() {
			t.Errorf("SpecialEffect(%s).IsItemEffect() should be true", se)
		}
	}

	nonItemEffects := []SpecialEffect{
		SpecialImmune, SpecialReverse, SpecialImmunePoison, SpecialBadEvent,
		SpecialZhuQuePassive, SpecialDrawItem, SpecialLoseItem,
		SpecialSwapPosition, SpecialRandomBuff, SpecialNone,
	}
	for _, se := range nonItemEffects {
		if se.IsItemEffect() {
			t.Errorf("SpecialEffect(%s).IsItemEffect() should be false", se)
		}
	}
}

func TestSpecialEffectIsEventEffect(t *testing.T) {
	eventEffects := []SpecialEffect{
		SpecialDrawItem, SpecialLoseItem, SpecialSwapPosition, SpecialRandomBuff,
	}
	for _, se := range eventEffects {
		if !se.IsEventEffect() {
			t.Errorf("SpecialEffect(%s).IsEventEffect() should be true", se)
		}
	}

	nonEventEffects := []SpecialEffect{
		SpecialImmune, SpecialReverse, SpecialImmunePoison, SpecialBadEvent,
		SpecialZhuQuePassive, SpecialTeleport, SpecialDiceSwap,
		SpecialDiceUpgrade, SpecialGiveLost, SpecialNone,
	}
	for _, se := range nonEventEffects {
		if se.IsEventEffect() {
			t.Errorf("SpecialEffect(%s).IsEventEffect() should be false", se)
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