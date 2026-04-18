package action

import (
	"testing"

	"github.com/b1tAction/paradiced/internal/core"
	"github.com/b1tAction/paradiced/internal/event"
	"github.com/b1tAction/paradiced/pkg/constants"
	"github.com/b1tAction/paradiced/pkg/id"
)

// ========== ActionType Tests ==========

func TestActionTypeString(t *testing.T) {
	tests := []struct {
		at       ActionType
		expected string
	}{
		{ActionDamage, "damage"},
		{ActionHeal, "heal"},
		{ActionModifyLP, "modify_lp"},
		{ActionMove, "move"},
		{ActionAddBuff, "add_buff"},
		{ActionRemoveBuff, "remove_buff"},
		{ActionRespawn, "respawn"},
		{ActionSkipTurn, "skip_turn"},
		{ActionDrawEvent, "draw_event"},
		{ActionTeleport, "teleport"},
		{ActionStealBuff, "steal_buff"},
		{ActionFellDown, "fell_down"},
		{ActionUnknown, "unknown"},
	}

	for _, tt := range tests {
		// ActionType is now a string, so direct comparison
		if string(tt.at) != tt.expected {
			t.Errorf("ActionType(%s) = %s, want %s", tt.at, tt.at, tt.expected)
		}
	}
}

// ========== Queue Tests ==========

func TestQueuePushPop(t *testing.T) {
	q := NewQueue()

	if q.Len() != 0 {
		t.Error("New queue should be empty")
	}
	if !q.IsEmpty() {
		t.Error("New queue should report empty")
	}

	// Create mock actions for testing
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	action1 := NewDamageAction(player, 10, "test1")
	action2 := NewHealAction(player, 5, "test2")

	q.Push(action1)
	q.Push(action2)

	if q.Len() != 2 {
		t.Errorf("Queue length should be 2, got %d", q.Len())
	}

	// Pop should return first pushed
	popped := q.Pop()
	if popped.Source() != "test1" {
		t.Errorf("First pop should be test1, got %s", popped.Source())
	}

	if q.Len() != 1 {
		t.Errorf("Queue length should be 1 after pop, got %d", q.Len())
	}

	// Peek should return remaining without removing
	peeked := q.Peek()
	if peeked.Source() != "test2" {
		t.Errorf("Peek should be test2, got %s", peeked.Source())
	}
	if q.Len() != 1 {
		t.Error("Peek should not remove item")
	}

	// Pop remaining
	q.Pop()
	if !q.IsEmpty() {
		t.Error("Queue should be empty after popping all")
	}

	// Pop empty queue
	nilAction := q.Pop()
	if nilAction != nil {
		t.Error("Pop on empty queue should return nil")
	}
}

func TestQueueClear(t *testing.T) {
	q := NewQueue()
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	q.Push(NewDamageAction(player, 10, "test"))
	q.Push(NewHealAction(player, 5, "test"))

	q.Clear()

	if q.Len() != 0 {
		t.Error("Clear should empty queue")
	}
	if !q.IsEmpty() {
		t.Error("Clear should make queue empty")
	}
}

// ========== DamageAction Tests ==========

func TestDamageAction(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player.HP = 100

	action := NewDamageAction(player, 20, "Event_Trap")

	// Verify Action interface implementation
	if action.Type() != ActionDamage {
		t.Errorf("Type should be ActionDamage, got %s", action.Type())
	}
	if !action.CanModify() {
		t.Error("DamageAction should be modifiable by default")
	}
	if action.Source() != "Event_Trap" {
		t.Errorf("Source should be Event_Trap, got %s", action.Source())
	}
	if action.Target() != player.ID.UUID() {
		t.Errorf("Target should be %s, got %s", player.ID.UUID(), action.Target())
	}

	// Execute with minimal context
	ctx := NewActionContext(nil, nil, nil, nil)
	err := action.Execute(ctx)
	if err != nil {
		t.Errorf("Execute failed: %v", err)
	}

	if player.HP != 80 {
		t.Errorf("HP should be 80 after damage, got %d", player.HP)
	}

	// Verify log entry
	entry := action.LogEntry()
	if entry.Type != constants.EntryTypeAction {
		t.Errorf("Log type should be action, got %s", entry.Type)
	}
	if entry.ActionType != "damage" {
		t.Errorf("Log ActionType should be damage, got %s", entry.ActionType)
	}
	if entry.Metadata.GetIntOrDefault("hp_change", 0) != -20 {
		t.Errorf("Log delta should be -20, got %d", entry.Metadata.GetIntOrDefault("hp_change", 0))
	}
}

func TestPiercingDamageAction(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player.HP = 100

	action := NewPiercingDamageAction(player, 30, "Boss")

	if action.CanModify() {
		t.Error("Piercing damage should not be modifiable")
	}
	if !action.IsPiercing {
		t.Error("IsPiercing should be true")
	}

	ctx := NewActionContext(nil, nil, nil, nil)
	action.Execute(ctx)

	if player.HP != 70 {
		t.Errorf("HP should be 70, got %d", player.HP)
	}
}

func TestBlockedDamageAction(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player.HP = 100

	action := NewDamageAction(player, 20, "Event_Trap")
	action.Amount = 0 // Blocked by interceptor
	action.BlockedBy = "Buff_Hidden"

	ctx := NewActionContext(nil, nil, nil, nil)
	action.Execute(ctx)

	if player.HP != 100 {
		t.Errorf("Blocked damage should not affect HP, got %d", player.HP)
	}
}

// ========== HealAction Tests ==========

func TestHealAction(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID(), MaxHP: 100})
	player.HP = 50

	action := NewHealAction(player, 30, "Buff_Rain")

	if action.Type() != ActionHeal {
		t.Errorf("Type should be ActionHeal, got %s", action.Type())
	}
	if !action.CanModify() {
		t.Error("HealAction should be modifiable")
	}

	ctx := NewActionContext(nil, nil, nil, nil)
	action.Execute(ctx)

	if player.HP != 80 {
		t.Errorf("HP should be 80 after heal, got %d", player.HP)
	}

	entry := action.LogEntry()
	if entry.Metadata.GetIntOrDefault("hp_change", 0) != 30 {
		t.Errorf("Log delta should be 30, got %d", entry.Metadata.GetIntOrDefault("hp_change", 0))
	}
}

// ========== ModifyLPAction Tests ==========

func TestModifyLPAction(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player.LP = 5

	// LP+1
	actionPlus := NewModifyLPAction(player, 1, "Buff_Divine")
	if actionPlus.CanModify() {
		t.Error("ModifyLPAction should not be modifiable")
	}

	ctx := NewActionContext(nil, nil, nil, nil)
	actionPlus.Execute(ctx)

	if player.LP != 6 {
		t.Errorf("LP should be 6, got %d", player.LP)
	}

	// LP-1
	actionMinus := NewModifyLPAction(player, -1, "Buff_Curse")
	actionMinus.Execute(ctx)

	if player.LP != 5 {
		t.Errorf("LP should be 5, got %d", player.LP)
	}
}

// ========== AddBuffAction Tests ==========

func TestAddBuffAction(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})

	action := NewAddBuffAction(player, constants.BuffTypeDivine, 3, "Event_DivineGift")

	if action.Type() != ActionAddBuff {
		t.Errorf("Type should be ActionAddBuff, got %s", action.Type())
	}
	if action.CanModify() {
		t.Error("AddBuffAction should not be modifiable")
	}
	if action.BuffType != constants.BuffTypeDivine {
		t.Errorf("BuffType should be Divine, got %s", string(action.BuffType))
	}

	ctx := NewActionContext(nil, nil, nil, nil)
	action.Execute(ctx)

	if len(player.ActiveBuffs) != 1 {
		t.Errorf("Player should have 1 buff, got %d", len(player.ActiveBuffs))
	}
	if player.ActiveBuffs[0].Type != constants.BuffTypeDivine {
		t.Error("Buff type should be Divine")
	}
	if player.ActiveBuffs[0].Duration != 3 {
		t.Errorf("Buff duration should be 3, got %d", player.ActiveBuffs[0].Duration)
	}

	entry := action.LogEntry()
	if entry.ActionType != "add_buff" {
		t.Errorf("Log ActionType should be add_buff, got %s", entry.ActionType)
	}
}

// ========== RemoveBuffAction Tests ==========

func TestRemoveBuffAction(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player.AddBuff(core.NewBuff(constants.BuffTypeCurse, 2))
	player.AddBuff(core.NewBuff(constants.BuffTypeDivine, 3))

	if len(player.ActiveBuffs) != 2 {
		t.Error("Setup: player should have 2 buffs")
	}

	action := NewRemoveBuffAction(player, constants.BuffTypeCurse, "Manual_Remove")

	ctx := NewActionContext(nil, nil, nil, nil)
	action.Execute(ctx)

	if len(player.ActiveBuffs) != 1 {
		t.Errorf("Player should have 1 buff after removal, got %d", len(player.ActiveBuffs))
	}
	if player.ActiveBuffs[0].Type != constants.BuffTypeDivine {
		t.Error("Remaining buff should be Divine")
	}
}

// ========== TeleportAction Tests ==========

func TestTeleportAction(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player.Position = 5

	action := NewTeleportAction(player, 20, "Item_AnyDoor")

	if action.Type() != ActionTeleport {
		t.Errorf("Type should be ActionTeleport, got %s", action.Type())
	}
	if action.TargetPos != 20 {
		t.Errorf("TargetPos should be 20, got %d", action.TargetPos)
	}

	ctx := NewActionContext(nil, nil, nil, nil)
	action.Execute(ctx)

	if player.Position != 20 {
		t.Errorf("Position should be 20, got %d", player.Position)
	}

	entry := action.LogEntry()
	if entry.ActionType != "teleport" {
		t.Errorf("Log ActionType should be teleport, got %s", entry.ActionType)
	}
}

// ========== StealBuffAction Tests ==========

func TestStealBuffAction(t *testing.T) {
	target := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	source := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	target.AddBuff(core.NewBuff(constants.BuffTypeCurse, 2))
	target.AddBuff(core.NewBuff(constants.BuffTypeDivine, 3))

	action := NewStealBuffAction(target, source, "Faction_BaiHu")

	if action.Type() != ActionStealBuff {
		t.Errorf("Type should be ActionStealBuff, got %s", action.Type())
	}
	if action.SourcePlayer != source {
		t.Errorf("SourcePlayer should be source, got %s", action.SourcePlayer.ID.UUID())
	}

	ctx := NewActionContext(nil, nil, nil, nil)
	action.Execute(ctx)

	// Target should lose a buff
	if len(target.ActiveBuffs) != 1 {
		t.Errorf("Target should have 1 buff, got %d", len(target.ActiveBuffs))
	}

	// Source should gain a buff
	if len(source.ActiveBuffs) != 1 {
		t.Errorf("Source should have 1 buff, got %d", len(source.ActiveBuffs))
	}

	// Verify stolen buff was transferred
	if action.StolenBuff == nil {
		t.Error("StolenBuff should be set after execution")
	}
}

func TestStealBuffActionNoBuffs(t *testing.T) {
	target := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	source := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})

	action := NewStealBuffAction(target, source, "Faction_BaiHu")

	ctx := NewActionContext(nil, nil, nil, nil)
	action.Execute(ctx)

	// Should not fail when target has no buffs
	if len(source.ActiveBuffs) != 0 {
		t.Error("Source should have no buffs when target has none")
	}
	if action.StolenBuff != nil {
		t.Error("StolenBuff should be nil when no buffs to steal")
	}
}

// ========== RespawnAction Tests ==========

func TestRespawnAction(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player.IsDead = true
	player.Position = 100

	action := NewRespawnAction(player, 50, "DeathRespawn")

	if action.Type() != ActionRespawn {
		t.Errorf("Type should be ActionRespawn, got %s", action.Type())
	}
	if action.CanModify() {
		t.Error("RespawnAction should not be modifiable")
	}
	if action.CheckpointPos != 50 {
		t.Errorf("CheckpointPos should be 50, got %d", action.CheckpointPos)
	}

	ctx := NewActionContext(nil, nil, nil, nil)
	action.Execute(ctx)

	if player.Position != 50 {
		t.Errorf("Position should be 50 after respawn, got %d", player.Position)
	}
	if player.IsDead {
		t.Error("Player should not be dead after respawn")
	}

	entry := action.LogEntry()
	if entry.ActionType != "respawn" {
		t.Errorf("Log ActionType should be respawn, got %s", entry.ActionType)
	}
}

// ========== FellDownAction Tests ==========

func TestFellDownAction(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player.HP = 10
	player.Position = 30

	action := NewFellDownAction(player, 30, 1, "FragileCell")

	if action.Type() != ActionFellDown {
		t.Errorf("Type should be ActionFellDown, got %s", action.Type())
	}
	if action.CanModify() {
		t.Error("FellDownAction should not be modifiable")
	}
	if action.Damage != 1 {
		t.Errorf("Damage should be 1, got %d", action.Damage)
	}

	ctx := NewActionContext(nil, nil, nil, nil)
	action.Execute(ctx)

	if player.HP != 9 {
		t.Errorf("HP should be 9 after falling damage, got %d", player.HP)
	}

	entry := action.LogEntry()
	if entry.ActionType != "fell_down" {
		t.Errorf("Log ActionType should be fell_down, got %s", entry.ActionType)
	}
	if entry.Metadata.GetIntOrDefault("hp_change", 0) != -1 {
		t.Errorf("Log delta should be -1, got %d", entry.Metadata.GetIntOrDefault("hp_change", 0))
	}
}

// ========== ActionContext Tests ==========

func TestActionContextExecuteAction(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player.HP = 100

	ctx := NewActionContext(nil, nil, nil, nil)
	action := NewDamageAction(player, 10, "test")

	err := ctx.ExecuteAction(action)
	if err != nil {
		t.Errorf("ExecuteAction failed: %v", err)
	}

	// Verify action was executed
	if player.HP != 90 {
		t.Errorf("HP should be 90, got %d", player.HP)
	}
}

func TestActionContextProcessQueue(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player.HP = 100

	ctx := NewActionContext(nil, nil, nil, nil)

	// Push derived actions
	ctx.PushDerivedAction(NewDamageAction(player, 10, "derived1"))
	ctx.PushDerivedAction(NewHealAction(player, 5, "derived2"))

	if ctx.ActionQueue.Len() != 2 {
		t.Errorf("Queue should have 2 actions, got %d", ctx.ActionQueue.Len())
	}

	// Process queue
	ctx.ProcessQueue()

	// Queue should be empty
	if !ctx.ActionQueue.IsEmpty() {
		t.Error("Queue should be empty after processing")
	}

	// HP should be 100 - 10 + 5 = 95
	if player.HP != 95 {
		t.Errorf("HP should be 95, got %d", player.HP)
	}
}

func TestActionContextClear(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	ctx := NewActionContext(nil, nil, nil, nil)

	ctx.PushDerivedAction(NewDamageAction(player, 10, "test"))

	ctx.Clear()

	if !ctx.ActionQueue.IsEmpty() {
		t.Error("Queue should be empty after Clear")
	}
}

// ========== ActionContext Metadata Tests ==========

func TestActionContextMetadata(t *testing.T) {
	ctx := NewActionContext(nil, nil, nil, nil)

	// Test Metadata functionality
	ctx.SetBool("test_key", true)
	if !ctx.GetBoolOrDefault("test_key", false) {
		t.Error("Metadata should store bool correctly")
	}

	ctx.SetInt("int_key", 42)
	if ctx.GetIntOrDefault("int_key", 0) != 42 {
		t.Error("Metadata should store int correctly")
	}

	ctx.SetString("str_key", "value")
	if ctx.GetStringOrDefault("str_key", "") != "value" {
		t.Error("Metadata should store string correctly")
	}
}

// ========== LogEntry Tests ==========

func TestLogEntryMetadata(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player.Position = 10

	// Test MoveAction LogEntry with metadata
	action := NewMoveAction(player, 5, "DiceRoll")
	action.TargetPos = 15
	action.Path = []int{10, 11, 12, 13, 14, 15}

	entry := action.LogEntry()

	if entry.Metadata == nil {
		t.Fatal("Metadata should not be nil")
	}

	// Use type-safe metadata access
	startPos := entry.Metadata.GetIntOrDefault("start_pos", -1)
	if startPos != 5 { // 10 - 5 = 5
		t.Errorf("start_pos should be 5, got %d", startPos)
	}

	endPos := entry.Metadata.GetIntOrDefault("end_pos", -1)
	if endPos != 15 {
		t.Errorf("end_pos should be 15, got %d", endPos)
	}
}

// ========== DerivedActions Tests ==========

func TestContextDerivedActions(t *testing.T) {
	// Test event.Context.AddDerivedAction functionality
	triggerCtx := event.NewContext(nil)

	// Initially empty
	if len(triggerCtx.GetDerivedActions()) != 0 {
		t.Error("DerivedActions should be empty initially")
	}

	// Add actions
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	healAction := NewHealAction(player, 10, "test")
	removeAction := NewRemoveBuffAction(player, constants.BuffTypeDivine, "test")

	triggerCtx.AddDerivedAction(healAction)
	triggerCtx.AddDerivedAction(removeAction)

	// Should have 2 actions
	if len(triggerCtx.GetDerivedActions()) != 2 {
		t.Errorf("DerivedActions should have 2 actions, got %d", len(triggerCtx.GetDerivedActions()))
	}

	// Clear should remove all
	triggerCtx.ClearDerivedActions()
	if len(triggerCtx.GetDerivedActions()) != 0 {
		t.Error("DerivedActions should be empty after Clear")
	}
}

func TestRespawnActionPreTriggerPhase(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	action := NewRespawnAction(player, 50, "DeathRespawn")

	// RespawnAction now has PhasePreRespawn for interception
	if action.PreTriggerPhase() != constants.PhasePreRespawn {
		t.Errorf("RespawnAction PreTriggerPhase should be PhasePreRespawn, got %s", string(action.PreTriggerPhase()))
	}
}

// ========== ModifyLPAction Full Coverage Tests ==========

func TestModifyLPActionFull(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player.LP = 5

	action := NewModifyLPAction(player, 1, "Buff_Divine")

	// Test all methods
	if action.Type() != ActionModifyLP {
		t.Errorf("Type should be ActionModifyLP, got %s", action.Type())
	}
	if action.Source() != "Buff_Divine" {
		t.Errorf("Source should be Buff_Divine, got %s", action.Source())
	}
	if action.Target() != player.ID.UUID() {
		t.Errorf("Target mismatch")
	}
	if action.PreTriggerPhase() != constants.PhaseAnyTime {
		t.Errorf("PreTriggerPhase should be PhaseAnyTime, got %s", action.PreTriggerPhase())
	}
	if action.PostTriggerPhase() != constants.PhaseAnyTime {
		t.Errorf("PostTriggerPhase should be PhaseAnyTime, got %s", action.PostTriggerPhase())
	}

	ctx := NewActionContext(nil, nil, nil, nil)
	action.Execute(ctx)

	entry := action.LogEntry()
	if entry.ActionType != "modify_lp" {
		t.Errorf("Log ActionType should be modify_lp, got %s", entry.ActionType)
	}
	if entry.Metadata.GetIntOrDefault("lp_change", 0) != 1 {
		t.Errorf("Log lp_change should be 1, got %d", entry.Metadata.GetIntOrDefault("lp_change", 0))
	}
}

func TestModifyLPActionZeroAmount(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player.LP = 5

	action := NewModifyLPAction(player, 0, "Test")

	ctx := NewActionContext(nil, nil, nil, nil)
	action.Execute(ctx)

	// Zero amount should not change LP
	if player.LP != 5 {
		t.Errorf("LP should remain 5, got %d", player.LP)
	}
}

// ========== MoveAction Full Coverage Tests ==========

func TestMoveActionFull(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player.Position = 10

	action := NewMoveAction(player, 5, "DiceRoll")

	// Test all methods
	if action.Type() != ActionMove {
		t.Errorf("Type should be ActionMove, got %s", action.Type())
	}
	if action.Source() != "DiceRoll" {
		t.Errorf("Source should be DiceRoll, got %s", action.Source())
	}
	if action.Target() != player.ID.UUID() {
		t.Errorf("Target mismatch")
	}
	if action.PreTriggerPhase() != constants.PhasePreMove {
		t.Errorf("PreTriggerPhase should be PhasePreMove, got %s", action.PreTriggerPhase())
	}
	if action.PostTriggerPhase() != constants.PhaseAnyTime {
		t.Errorf("PostTriggerPhase should be PhaseAnyTime, got %s", action.PostTriggerPhase())
	}
	if !action.CanModify() {
		t.Error("MoveAction should be modifiable when Steps != 0")
	}
}

func TestMoveActionOvertook(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	other := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})

	action := NewMoveAction(player, 5, "DiceRoll")

	// Initially no overtaken players
	if action.Overtook(other) {
		t.Error("Should not have overtaken initially")
	}

	// Add overtaken player
	action.Overtaken = []*core.Player{other}

	if !action.Overtook(other) {
		t.Error("Should have overtaken the other player")
	}
}

// ========== HealAction Full Coverage Tests ==========

func TestHealActionFull(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID(), MaxHP: 100})
	player.HP = 50

	action := NewHealAction(player, 30, "Buff_Rain")

	// Test all methods
	if action.Type() != ActionHeal {
		t.Errorf("Type should be ActionHeal, got %s", action.Type())
	}
	if action.Source() != "Buff_Rain" {
		t.Errorf("Source should be Buff_Rain, got %s", action.Source())
	}
	if action.Target() != player.ID.UUID() {
		t.Errorf("Target mismatch")
	}
	if action.PreTriggerPhase() != constants.PhaseAnyTime {
		t.Errorf("PreTriggerPhase should be PhaseAnyTime, got %s", action.PreTriggerPhase())
	}
	if action.PostTriggerPhase() != constants.PhaseAnyTime {
		t.Errorf("PostTriggerPhase should be PhaseAnyTime, got %s", action.PostTriggerPhase())
	}

	ctx := NewActionContext(nil, nil, nil, nil)
	action.Execute(ctx)

	entry := action.LogEntry()
	if entry.ActionType != "heal" {
		t.Errorf("Log ActionType should be heal, got %s", entry.ActionType)
	}
}

func TestHealActionZeroAmount(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID(), MaxHP: 100})
	player.HP = 50

	action := NewHealAction(player, 0, "Test")

	ctx := NewActionContext(nil, nil, nil, nil)
	action.Execute(ctx)

	// Zero amount should not change HP
	if player.HP != 50 {
		t.Errorf("HP should remain 50, got %d", player.HP)
	}
}

// ========== AddBuffAction Full Coverage Tests ==========

func TestAddBuffActionFull(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})

	action := NewAddBuffAction(player, constants.BuffTypeDivine, 3, "Event_Gift")

	// Test all methods
	if action.Type() != ActionAddBuff {
		t.Errorf("Type should be ActionAddBuff, got %s", action.Type())
	}
	if action.Source() != "Event_Gift" {
		t.Errorf("Source should be Event_Gift, got %s", action.Source())
	}
	if action.Target() != player.ID.UUID() {
		t.Errorf("Target mismatch")
	}
	if action.PreTriggerPhase() != constants.PhaseAnyTime {
		t.Errorf("PreTriggerPhase should be PhaseAnyTime, got %s", action.PreTriggerPhase())
	}
	if action.PostTriggerPhase() != constants.PhaseOnBuffApplied {
		t.Errorf("PostTriggerPhase should be PhaseOnBuffApplied, got %s", action.PostTriggerPhase())
	}

	ctx := NewActionContext(nil, nil, nil, nil)
	action.Execute(ctx)

	entry := action.LogEntry()
	if entry.ActionType != "add_buff" {
		t.Errorf("Log ActionType should be add_buff, got %s", entry.ActionType)
	}
}

// ========== RemoveBuffAction Full Coverage Tests ==========

func TestRemoveBuffActionFull(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player.AddBuff(core.NewBuff(constants.BuffTypeCurse, 2))

	action := NewRemoveBuffAction(player, constants.BuffTypeCurse, "Manual")

	// Test all methods
	if action.Type() != ActionRemoveBuff {
		t.Errorf("Type should be ActionRemoveBuff, got %s", action.Type())
	}
	if action.Source() != "Manual" {
		t.Errorf("Source should be Manual, got %s", action.Source())
	}
	if action.Target() != player.ID.UUID() {
		t.Errorf("Target mismatch")
	}
	if action.PreTriggerPhase() != constants.PhaseOnBuffRemoved {
		t.Errorf("PreTriggerPhase should be PhaseOnBuffRemoved, got %s", action.PreTriggerPhase())
	}
	if action.PostTriggerPhase() != constants.PhaseAnyTime {
		t.Errorf("PostTriggerPhase should be PhaseAnyTime, got %s", action.PostTriggerPhase())
	}

	ctx := NewActionContext(nil, nil, nil, nil)
	action.Execute(ctx)

	entry := action.LogEntry()
	if entry.ActionType != "remove_buff" {
		t.Errorf("Log ActionType should be remove_buff, got %s", entry.ActionType)
	}
}

// ========== TeleportAction Full Coverage Tests ==========

func TestTeleportActionFull(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player.Position = 5

	action := NewTeleportAction(player, 20, "Item_Door")

	// Test all methods
	if action.Type() != ActionTeleport {
		t.Errorf("Type should be ActionTeleport, got %s", action.Type())
	}
	if action.Source() != "Item_Door" {
		t.Errorf("Source should be Item_Door, got %s", action.Source())
	}
	if action.Target() != player.ID.UUID() {
		t.Errorf("Target mismatch")
	}
	if action.PreTriggerPhase() != constants.PhaseAnyTime {
		t.Errorf("PreTriggerPhase should be PhaseAnyTime, got %s", action.PreTriggerPhase())
	}
	if action.PostTriggerPhase() != constants.PhaseAnyTime {
		t.Errorf("PostTriggerPhase should be PhaseAnyTime, got %s", action.PostTriggerPhase())
	}

	ctx := NewActionContext(nil, nil, nil, nil)
	action.Execute(ctx)

	entry := action.LogEntry()
	if entry.ActionType != "teleport" {
		t.Errorf("Log ActionType should be teleport, got %s", entry.ActionType)
	}
}

// ========== StealBuffAction Full Coverage Tests ==========

func TestStealBuffActionFull(t *testing.T) {
	target := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	source := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	target.AddBuff(core.NewBuff(constants.BuffTypeDivine, 3))

	action := NewStealBuffAction(target, source, "Faction_BaiHu")

	// Test all methods
	if action.Type() != ActionStealBuff {
		t.Errorf("Type should be ActionStealBuff, got %s", action.Type())
	}
	if action.Source() != "Faction_BaiHu" {
		t.Errorf("Source should be Faction_BaiHu, got %s", action.Source())
	}
	if action.Target() != target.ID.UUID() {
		t.Errorf("Target mismatch")
	}
	if action.PreTriggerPhase() != constants.PhaseAnyTime {
		t.Errorf("PreTriggerPhase should be PhaseAnyTime, got %s", action.PreTriggerPhase())
	}
	if action.PostTriggerPhase() != constants.PhaseAnyTime {
		t.Errorf("PostTriggerPhase should be PhaseAnyTime, got %s", action.PostTriggerPhase())
	}

	ctx := NewActionContext(nil, nil, nil, nil)
	action.Execute(ctx)

	entry := action.LogEntry()
	if entry.ActionType != "steal_buff" {
		t.Errorf("Log ActionType should be steal_buff, got %s", entry.ActionType)
	}
}

// ========== DrawEventAction Tests ==========

func TestDrawEventAction(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})

	action := NewDrawEventAction(player, "EventCard")

	// Test all methods
	if action.Type() != ActionDrawEvent {
		t.Errorf("Type should be ActionDrawEvent, got %s", action.Type())
	}
	if !action.CanModify() {
		t.Error("DrawEventAction should be modifiable")
	}
	if action.Source() != "EventCard" {
		t.Errorf("Source should be EventCard, got %s", action.Source())
	}
	if action.Target() != player.ID.UUID() {
		t.Errorf("Target mismatch")
	}
	if action.PreTriggerPhase() != constants.PhasePreEvent {
		t.Errorf("PreTriggerPhase should be PhasePreEvent, got %s", action.PreTriggerPhase())
	}
	if action.PostTriggerPhase() != constants.PhaseAnyTime {
		t.Errorf("PostTriggerPhase should be PhaseAnyTime, got %s", action.PostTriggerPhase())
	}

	// Execute without DrawEngine (placeholder)
	ctx := NewActionContext(nil, nil, nil, nil)
	action.Execute(ctx)

	entry := action.LogEntry()
	if entry.ActionType != "draw_event" {
		t.Errorf("Log ActionType should be draw_event, got %s", entry.ActionType)
	}
}

// ========== FellDownAction Full Coverage Tests ==========

func TestFellDownActionFull(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player.HP = 10

	action := NewFellDownAction(player, 30, 1, "FragileCell")

	// Test all methods
	if action.Type() != ActionFellDown {
		t.Errorf("Type should be ActionFellDown, got %s", action.Type())
	}
	if action.Source() != "FragileCell" {
		t.Errorf("Source should be FragileCell, got %s", action.Source())
	}
	if action.Target() != player.ID.UUID() {
		t.Errorf("Target mismatch")
	}
	if action.PreTriggerPhase() != constants.PhaseAnyTime {
		t.Errorf("PreTriggerPhase should be PhaseAnyTime, got %s", action.PreTriggerPhase())
	}
	if action.PostTriggerPhase() != constants.PhaseAnyTime {
		t.Errorf("PostTriggerPhase should be PhaseAnyTime, got %s", action.PostTriggerPhase())
	}

	ctx := NewActionContext(nil, nil, nil, nil)
	action.Execute(ctx)

	entry := action.LogEntry()
	if entry.ActionType != "fell_down" {
		t.Errorf("Log ActionType should be fell_down, got %s", entry.ActionType)
	}
}

func TestFellDownActionZeroDamage(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player.HP = 10

	action := NewFellDownAction(player, 30, 0, "FragileCell")

	ctx := NewActionContext(nil, nil, nil, nil)
	action.Execute(ctx)

	// Zero damage should not change HP
	if player.HP != 10 {
		t.Errorf("HP should remain 10, got %d", player.HP)
	}
}

// ========== RespawnAction Full Coverage Tests ==========

func TestRespawnActionFull(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player.IsDead = true

	action := NewRespawnAction(player, 50, "DeathRespawn")

	// Test all methods
	if action.Type() != ActionRespawn {
		t.Errorf("Type should be ActionRespawn, got %s", action.Type())
	}
	if action.Source() != "DeathRespawn" {
		t.Errorf("Source should be DeathRespawn, got %s", action.Source())
	}
	if action.Target() != player.ID.UUID() {
		t.Errorf("Target mismatch")
	}
	if action.PostTriggerPhase() != constants.PhaseAnyTime {
		t.Errorf("PostTriggerPhase should be PhaseAnyTime, got %s", action.PostTriggerPhase())
	}

	ctx := NewActionContext(nil, nil, nil, nil)
	action.Execute(ctx)

	entry := action.LogEntry()
	if entry.ActionType != "respawn" {
		t.Errorf("Log ActionType should be respawn, got %s", entry.ActionType)
	}
}
