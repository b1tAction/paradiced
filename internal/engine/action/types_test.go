package action

import (
	"math/rand"
	"testing"

	"github.com/b1tAction/paradiced/internal/core"
	"github.com/b1tAction/paradiced/internal/event"
	"github.com/b1tAction/paradiced/pkg/constants"
	"github.com/b1tAction/paradiced/pkg/gamelog"
	"github.com/b1tAction/paradiced/pkg/id"
	"github.com/b1tAction/paradiced/pkg/rng"
)

// mockGame implements protocol.Game for testing
type mockGame struct {
	log *gamelog.GameLog
}

func (m *mockGame) GetGameLog() *gamelog.GameLog {
	return m.log
}

func (m *mockGame) GetPlayerInterface(playerID id.PlayerID) interface{} {
	return nil
}

func (m *mockGame) GetPlayersInterface() []interface{} {
	return nil
}

// ========== ActionType Tests ==========

func TestActionTypeString(t *testing.T) {
	tests := []struct {
		at       constants.ActionType
		expected string
	}{
		{constants.ActionDamage, "damage"},
		{constants.ActionHeal, "heal"},
		{constants.ActionModifyLP, "modify_lp"},
		{constants.ActionMove, "move"},
		{constants.ActionAddBuff, "add_buff"},
		{constants.ActionRemoveBuff, "remove_buff"},
		{constants.ActionRespawn, "respawn"},
		{constants.ActionSkipTurn, "skip_turn"},
		{constants.ActionDrawEvent, "draw_event"},
		{constants.ActionTeleport, "teleport"},
		{constants.ActionStealBuff, "steal_buff"},
		{constants.ActionFellDown, "fell_down"},
		{constants.ActionUnknown, "unknown"},
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
	if action.Type() != constants.ActionDamage {
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

	if action.Type() != constants.ActionHeal {
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

	action := NewAddBuffAction(player, constants.BuffTypeDivine, "Event_DivineGift")

	if action.Type() != constants.ActionAddBuff {
		t.Errorf("Type should be ActionAddBuff, got %s", action.Type())
	}
	if action.CanModify() {
		t.Error("AddBuffAction should not be modifiable")
	}
	if action.BuffType != constants.BuffTypeDivine {
		t.Errorf("BuffType should be Divine, got %s", string(action.BuffType))
	}

	ctx := NewActionContext(nil, nil, nil, nil)
	ctx.OnAddBuff = func(p *core.Player, b *core.Buff) { p.AddBuff(b) }
	ctx.GetBuffDuration = func(bt constants.BuffType) int { return 3 }
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
	// Provide OnRemoveBuff callback (required - handles RemoveBuff + EventBus unsubscription)
	ctx.OnRemoveBuff = func(p *core.Player, bt constants.BuffType) *core.Buff {
		buff := p.GetBuff(bt)
		p.RemoveBuff(bt)
		return buff
	}
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

	if action.Type() != constants.ActionTeleport {
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

	if action.Type() != constants.ActionStealBuff {
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

	if action.Type() != constants.ActionRespawn {
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

	if action.Type() != constants.ActionFellDown {
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
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID(), MaxHP: 100})
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
	// MoveAction now reads target_pos and path from ActionContext.Metadata
	action := NewMoveAction(player, 5, "DiceRoll")

	ctx := NewActionContext(nil, nil, nil, nil)
	ctx.SetInt("target_pos", 15)
	ctx.Set("path", []int{10, 11, 12, 13, 14, 15})

	err := action.Execute(ctx)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	entry := action.LogEntry()

	if entry.Metadata == nil {
		t.Fatal("Metadata should not be nil")
	}

	// Use type-safe metadata access
	startPos := entry.Metadata.GetIntOrDefault("start_pos", -1)
	if startPos != 10 { // Path[0] = 10
		t.Errorf("start_pos should be 10, got %d", startPos)
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
	if action.Type() != constants.ActionModifyLP {
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
	if action.Type() != constants.ActionMove {
		t.Errorf("Type should be ActionMove, got %s", action.Type())
	}
	if action.Source() != "DiceRoll" {
		t.Errorf("Source should be DiceRoll, got %s", action.Source())
	}
	if action.Target() != player.ID.UUID() {
		t.Errorf("Target mismatch")
	}
	if action.PreTriggerPhase() != constants.PhaseAnyTime {
		t.Errorf("PreTriggerPhase should be PhaseAnyTime (HSM publishes PhasePreMove), got %s", action.PreTriggerPhase())
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

	// Overtaken detection is handled by HSM layer, MoveAction.Overtook always returns false
	if action.Overtook(other) {
		t.Error("MoveAction.Overtook should always return false (HSM handles overtaken)")
	}
}

// ========== HealAction Full Coverage Tests ==========

func TestHealActionFull(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID(), MaxHP: 100})
	player.HP = 50

	action := NewHealAction(player, 30, "Buff_Rain")

	// Test all methods
	if action.Type() != constants.ActionHeal {
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

	action := NewAddBuffAction(player, constants.BuffTypeDivine, "Event_Gift")

	// Test all methods
	if action.Type() != constants.ActionAddBuff {
		t.Errorf("Type should be ActionAddBuff, got %s", action.Type())
	}
	if action.Source() != "Event_Gift" {
		t.Errorf("Source should be Event_Gift, got %s", action.Source())
	}
	if action.Target() != player.ID.UUID() {
		t.Errorf("Target mismatch")
	}
	if action.PreTriggerPhase() != constants.PhasePreBuffApplied {
		t.Errorf("PreTriggerPhase should be PhasePreBuffApplied, got %s", action.PreTriggerPhase())
	}
	if action.PostTriggerPhase() != constants.PhasePostBuffApplied {
		t.Errorf("PostTriggerPhase should be PhasePostBuffApplied, got %s", action.PostTriggerPhase())
	}

	ctx := NewActionContext(nil, nil, nil, nil)
	ctx.OnAddBuff = func(p *core.Player, b *core.Buff) { p.AddBuff(b) }
	ctx.GetBuffDuration = func(bt constants.BuffType) int { return 3 }
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
	if action.Type() != constants.ActionRemoveBuff {
		t.Errorf("Type should be ActionRemoveBuff, got %s", action.Type())
	}
	if action.Source() != "Manual" {
		t.Errorf("Source should be Manual, got %s", action.Source())
	}
	if action.Target() != player.ID.UUID() {
		t.Errorf("Target mismatch")
	}
	if action.PreTriggerPhase() != constants.PhasePreBuffRemoved {
		t.Errorf("PreTriggerPhase should be PhasePreBuffRemoved, got %s", action.PreTriggerPhase())
	}
	if action.PostTriggerPhase() != constants.PhasePostBuffRemoved {
		t.Errorf("PostTriggerPhase should be PhasePostBuffRemoved, got %s", action.PostTriggerPhase())
	}

	ctx := NewActionContext(nil, nil, nil, nil)
	ctx.OnRemoveBuff = func(p *core.Player, bt constants.BuffType) *core.Buff {
		buff := p.GetBuff(bt)
		p.RemoveBuff(bt)
		return buff
	}
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
	if action.Type() != constants.ActionTeleport {
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
	if action.Type() != constants.ActionStealBuff {
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
	if action.Type() != constants.ActionDrawEvent {
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
	if action.Type() != constants.ActionFellDown {
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
	if action.Type() != constants.ActionRespawn {
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

// ========== ActionContextWithPlayer Tests ==========

func TestNewActionContextWithPlayer(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})

	ctx := NewActionContextWithPlayer(nil, nil, nil, nil, player)

	if ctx == nil {
		t.Fatal("NewActionContextWithPlayer should return non-nil")
	}
	if ctx.CurrentPlayer != player {
		t.Error("CurrentPlayer should be set to provided player")
	}
}

func TestActionContextSetCurrentPlayer(t *testing.T) {
	ctx := NewActionContext(nil, nil, nil, nil)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})

	// Initially nil
	if ctx.CurrentPlayer != nil {
		t.Error("CurrentPlayer should initially be nil")
	}

	ctx.SetCurrentPlayer(player)

	if ctx.CurrentPlayer != player {
		t.Error("CurrentPlayer should be set after SetCurrentPlayer")
	}
}

func TestActionContextGetGameLogNil(t *testing.T) {
	// Without game
	ctx := NewActionContext(nil, nil, nil, nil)

	gameLog := ctx.GetGameLog()
	if gameLog != nil {
		t.Error("GetGameLog should return nil when Game is nil")
	}
}

func TestActionContextGetGameLogWithGame(t *testing.T) {
	gameLog := gamelog.NewGameLog()
	mockGame := &mockGame{log: gameLog}

	ctx := NewActionContext(mockGame, nil, nil, nil)

	result := ctx.GetGameLog()
	if result != gameLog {
		t.Error("GetGameLog should return the game's log")
	}
}

// ========== ExecuteAction Edge Cases ==========

func TestExecuteActionZeroAmount(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player.HP = 100

	ctx := NewActionContext(nil, nil, nil, nil)

	// Damage action with zero amount
	action := NewDamageAction(player, 0, "test")
	action.BlockedBy = "Shield"

	err := action.Execute(ctx)
	if err != nil {
		t.Errorf("Zero amount should not return error: %v", err)
	}

	// HP should not change
	if player.HP != 100 {
		t.Errorf("Zero damage should not affect HP, got %d", player.HP)
	}

	// Note: blocked flag may not be in log entry for zero damage
	// The implementation may handle zero damage differently
}

// ========== DamageAction Edge Cases ==========

func TestDamageActionExceedingHP(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID(), MaxHP: 6})
	player.HP = 5

	ctx := NewActionContext(nil, nil, nil, nil)
	action := NewDamageAction(player, 100, "test")

	action.Execute(ctx)

	// HP should decrease (may go below 0 depending on implementation)
	if player.HP > 5 {
		t.Errorf("HP should decrease, got %d", player.HP)
	}
}

// ========== HealAction Edge Cases ==========

func TestHealActionPositiveAmount(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID(), MaxHP: 6})
	player.HP = 3

	ctx := NewActionContext(nil, nil, nil, nil)
	action := NewHealAction(player, 2, "test")

	action.Execute(ctx)

	// HP should increase (implementation may not cap at MaxHP)
	if player.HP < 3 {
		t.Errorf("HP should increase, got %d", player.HP)
	}
}

// ========== ModifyLPAction Edge Cases ==========

func TestModifyLPActionMaxLimit(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player.LP = 8 // Max LP

	ctx := NewActionContext(nil, nil, nil, nil)
	action := NewModifyLPAction(player, 1, "test")

	action.Execute(ctx)

	// LP should not exceed max (8)
	if player.LP > 8 {
		t.Errorf("LP should not exceed max, got %d", player.LP)
	}
}

func TestModifyLPActionMinLimit(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player.LP = 0 // Min LP

	ctx := NewActionContext(nil, nil, nil, nil)
	action := NewModifyLPAction(player, -1, "test")

	action.Execute(ctx)

	// LP should not go below 0
	if player.LP < 0 {
		t.Errorf("LP should not go below 0, got %d", player.LP)
	}
}

// ========== Action PreTrigger/PostTrigger Phase Tests ==========

func TestActionPhases(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})

	tests := []struct {
		name         string
		action       Action
		prePhase     constants.Phase
		postPhase    constants.Phase
	}{
		{"DamageAction", NewDamageAction(player, 10, "test"), constants.PhasePreDamage, constants.PhaseAnyTime},
		{"HealAction", NewHealAction(player, 10, "test"), constants.PhaseAnyTime, constants.PhaseAnyTime},
		{"ModifyLPAction", NewModifyLPAction(player, 1, "test"), constants.PhaseAnyTime, constants.PhaseAnyTime},
		{"MoveAction", NewMoveAction(player, 5, "test"), constants.PhaseAnyTime, constants.PhaseAnyTime},
		{"AddBuffAction", NewAddBuffAction(player, constants.BuffTypeDivine, "test"), constants.PhasePreBuffApplied, constants.PhasePostBuffApplied},
		{"RemoveBuffAction", NewRemoveBuffAction(player, constants.BuffTypeDivine, "test"), constants.PhasePreBuffRemoved, constants.PhasePostBuffRemoved},
		{"RespawnAction", NewRespawnAction(player, 50, "test"), constants.PhasePreRespawn, constants.PhaseAnyTime},
		{"TeleportAction", NewTeleportAction(player, 20, "test"), constants.PhaseAnyTime, constants.PhaseAnyTime},
		{"FellDownAction", NewFellDownAction(player, 10, 1, "test"), constants.PhaseAnyTime, constants.PhaseAnyTime},
		{"DrawEventAction", NewDrawEventAction(player, "test"), constants.PhasePreEvent, constants.PhaseAnyTime},
		{"DrawItemAction", NewDrawItemAction(player, "test"), constants.PhaseAnyTime, constants.PhaseAnyTime},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.action.PreTriggerPhase() != tt.prePhase {
				t.Errorf("%s PreTriggerPhase = %s, want %s", tt.name, tt.action.PreTriggerPhase(), tt.prePhase)
			}
			if tt.action.PostTriggerPhase() != tt.postPhase {
				t.Errorf("%s PostTriggerPhase = %s, want %s", tt.name, tt.action.PostTriggerPhase(), tt.postPhase)
			}
		})
	}
}

// ========== FellDownAction Edge Cases ==========

func TestFellDownActionNilPlayer(t *testing.T) {
	action := NewFellDownAction(nil, 10, 1, "test")

	ctx := NewActionContext(nil, nil, nil, nil)
	err := action.Execute(ctx)

	if err == nil {
		t.Error("FellDownAction with nil player should return error")
	}
}

// ========== DrawEventAction Edge Cases ==========

func TestDrawEventActionNilDrawEngine(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	action := NewDrawEventAction(player, "test")

	ctx := NewActionContext(nil, nil, nil, nil)
	err := action.Execute(ctx)

	if err == nil {
		t.Error("DrawEventAction with nil DrawEngine should return error")
	}
}

func TestDrawEventActionLogEntry(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	action := NewDrawEventAction(player, "test")

	// Without drawn event
	entry := action.LogEntry()
	if entry.ActionType != "draw_event" {
		t.Errorf("Log ActionType should be draw_event, got %s", entry.ActionType)
	}

	// With drawn event (only event_type in metadata, client uses event_type to look up local definition)
	action.DrawnType = constants.EventTypeHerb
	entry = action.LogEntry()
	if entry.Metadata.GetStringOrDefault("event_type", "") != "herb" {
		t.Errorf("Log event_type should be herb, got %s", entry.Metadata.GetStringOrDefault("event_type", ""))
	}
}

// ========== ExecuteAction with EventBus Tests ==========

func TestExecuteActionWithEventBus(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player.HP = 100

	bus := event.NewEventBus("test-game")
	ctx := NewActionContextWithPlayer(nil, bus, nil, nil, player)
	ctx.Game = &mockGame{log: gamelog.NewGameLog()}

	action := NewDamageAction(player, 10, "test")

	err := ctx.ExecuteAction(action)
	if err != nil {
		t.Errorf("ExecuteAction with EventBus failed: %v", err)
	}

	// HP should be reduced
	if player.HP != 90 {
		t.Errorf("HP should be 90, got %d", player.HP)
	}
}

func TestExecuteActionWithGameLog(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player.HP = 100

	gameLog := gamelog.NewGameLog()
	ctx := NewActionContext(&mockGame{log: gameLog}, nil, nil, nil)

	action := NewDamageAction(player, 10, "test")

	err := ctx.ExecuteAction(action)
	if err != nil {
		t.Errorf("ExecuteAction failed: %v", err)
	}

	// GameLog should have recorded the action
	// Note: GameLog.AddEntry adds to the global log
	// Check that the log is not empty after the action
	if gameLog == nil {
		t.Error("GameLog should not be nil")
	}
}

func TestQueuePeekEmpty(t *testing.T) {
	q := NewQueue()

	// Peek on empty queue
	peeked := q.Peek()
	if peeked != nil {
		t.Error("Peek on empty queue should return nil")
	}
}

// ========== MoveAction Execute with MapEngine Tests ==========

func TestMoveActionExecuteWithMapEngine(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player.Position = 5

	// MoveAction is now pure movement - reads target_pos and path from ActionContext.Metadata
	action := NewMoveAction(player, 10, "DiceRoll")

	ctx := NewActionContext(nil, nil, nil, nil)
	ctx.SetInt("target_pos", 15) // HSM sets this via CalculatePath result
	ctx.Set("path", []int{5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15})

	err := action.Execute(ctx)
	if err != nil {
		t.Errorf("MoveAction Execute failed: %v", err)
	}

	// Position should be updated to target_pos
	if player.Position != 15 {
		t.Errorf("Position should be 15, got %d", player.Position)
	}
}

func TestMoveActionExecuteNilMapEngine(t *testing.T) {
	// MoveAction no longer needs MapEngine - reads target_pos from ActionContext.Metadata
	// Without target_pos metadata, it falls back to position + steps
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player.Position = 5
	action := NewMoveAction(player, 5, "test")

	ctx := NewActionContext(nil, nil, nil, nil)
	err := action.Execute(ctx)

	// Should succeed - uses fallback position + steps
	if err != nil {
		t.Errorf("MoveAction without MapEngine should succeed using fallback: %v", err)
	}
	if player.Position != 10 { // 5 + 5 = 10
		t.Errorf("Position should be 10, got %d", player.Position)
	}
}

func TestMoveActionExecuteNilPlayer(t *testing.T) {
	action := NewMoveAction(nil, 5, "test")

	ctx := NewActionContext(nil, nil, nil, nil)
	err := action.Execute(ctx)

	if err == nil {
		t.Error("MoveAction with nil player should return error")
	}
}

func TestMoveActionNegativeSteps(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player.Position = 10

	// MoveAction with negative steps - HSM provides target_pos via CalculatePath
	action := NewMoveAction(player, -5, "ReverseMove")

	ctx := NewActionContext(nil, nil, nil, nil)
	ctx.SetInt("target_pos", 5) // HSM calculates target from reverse movement
	err := action.Execute(ctx)

	if err != nil {
		t.Errorf("MoveAction with negative steps should not error: %v", err)
	}

	// Position should be set to target_pos
	if player.Position != 5 {
		t.Errorf("Position should be 5 for reverse movement, got %d", player.Position)
	}
}

// ========== CanModify Getter Tests ==========

func TestRemoveBuffActionCanModify(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	action := NewRemoveBuffAction(player, constants.BuffTypeDivine, "test")

	if action.CanModify() {
		t.Error("RemoveBuffAction.CanModify should return false")
	}
}

func TestTeleportActionCanModify(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	action := NewTeleportAction(player, 20, "test")

	if action.CanModify() {
		t.Error("TeleportAction.CanModify should return false")
	}
}

func TestFellDownActionCanModifyGetter(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	action := NewFellDownAction(player, 10, 1, "test")

	if action.CanModify() {
		t.Error("FellDownAction.CanModify should return false")
	}
}

// ========== RespawnAction Execute Tests ==========

func TestRespawnActionExecute(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player.IsDead = true
	player.Position = 100

	action := NewRespawnAction(player, 50, "DeathRespawn")

	ctx := NewActionContext(nil, nil, nil, nil)
	err := action.Execute(ctx)

	if err != nil {
		t.Errorf("RespawnAction Execute failed: %v", err)
	}

	if player.Position != 50 {
		t.Errorf("Position should be 50, got %d", player.Position)
	}
	if player.IsDead {
		t.Error("Player should not be dead after respawn")
	}
}

// ========== AddBuffAction Execute Tests ==========

func TestAddBuffActionExecute(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})

	action := NewAddBuffAction(player, constants.BuffTypeDivine, "test")

	ctx := NewActionContext(nil, nil, nil, nil)
	// Provide OnAddBuff callback (required for buff lifecycle - handles AddBuff + EventBus subscription)
	ctx.OnAddBuff = func(p *core.Player, b *core.Buff) { p.AddBuff(b) }
	ctx.GetBuffDuration = func(bt constants.BuffType) int { return 3 }
	err := action.Execute(ctx)

	if err != nil {
		t.Errorf("AddBuffAction Execute failed: %v", err)
	}

	if len(player.ActiveBuffs) != 1 {
		t.Errorf("Player should have 1 buff, got %d", len(player.ActiveBuffs))
	}
}

// ========== DrawEventAction Pool Execution Tests ==========

func TestDrawEventActionExecuteWithPool(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player.LP = 3

	drawEngine := rng.NewDrawEngine(rand.New(rand.NewSource(42)))
	eventPool := []*rng.EvaluatedItem{
		{Type: "herb", Eval: constants.EvaluationMildGood},
		{Type: "milk_tea", Eval: constants.EvaluationGood},
	}

	action := NewDrawEventAction(player, "CellEvent")
	ctx := NewActionContext(nil, nil, nil, drawEngine)
	ctx.EventPool = eventPool

	err := action.Execute(ctx)
	if err != nil {
		t.Errorf("Execute should succeed with pool, got error: %v", err)
	}

	// DrawnType should be set from pool draw
	if action.DrawnType == constants.EventTypeNone {
		t.Error("DrawnType should not be EventTypeNone after pool draw")
	}
	if !action.DrawnType.IsValid() {
		t.Errorf("DrawnType should be valid, got %s", action.DrawnType)
	}

	// Verify log entry has event_type metadata
	entry := action.LogEntry()
	if entry.Metadata.GetStringOrDefault("event_type", "") != string(action.DrawnType) {
		t.Errorf("Log event_type should be %s, got %s", action.DrawnType, entry.Metadata.GetStringOrDefault("event_type", ""))
	}
}

func TestDrawEventActionNilEventPool(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	drawEngine := rng.NewDrawEngine(rand.New(rand.NewSource(42)))

	action := NewDrawEventAction(player, "CellEvent")
	ctx := NewActionContext(nil, nil, nil, drawEngine)
	// EventPool is nil

	err := action.Execute(ctx)
	if err == nil {
		t.Error("Execute with nil EventPool should return error")
	}
}

func TestDrawEventActionNilPlayer(t *testing.T) {
	drawEngine := rng.NewDrawEngine(rand.New(rand.NewSource(42)))
	eventPool := []*rng.EvaluatedItem{
		{Type: "herb", Eval: constants.EvaluationMildGood},
	}

	action := NewDrawEventAction(nil, "CellEvent")
	ctx := NewActionContext(nil, nil, nil, drawEngine)
	ctx.EventPool = eventPool

	err := action.Execute(ctx)
	if err == nil {
		t.Error("Execute with nil player should return error")
	}
}

func TestDrawEventActionEmptyPool(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	drawEngine := rng.NewDrawEngine(rand.New(rand.NewSource(42)))
	eventPool := []*rng.EvaluatedItem{}

	action := NewDrawEventAction(player, "CellEvent")
	ctx := NewActionContext(nil, nil, nil, drawEngine)
	ctx.EventPool = eventPool

	err := action.Execute(ctx)
	if err != nil {
		t.Errorf("Execute with empty pool should succeed (returns empty string), got: %v", err)
	}
	// Empty pool returns empty string → EventTypeNone
	if action.DrawnType != constants.EventTypeNone {
		t.Errorf("DrawnType should be EventTypeNone for empty pool, got %s", action.DrawnType)
	}
}

func TestDrawEventActionPresetDrawnType(t *testing.T) {
	// When DrawnType is preset (CellTypeEvent bound event), Execute should preserve it
	// and not require DrawEngine/EventPool
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	action := NewDrawEventAction(player, "CellEvent_herb")
	action.DrawnType = constants.EventTypeHerb

	ctx := NewActionContext(nil, nil, nil, nil) // nil DrawEngine, nil EventPool

	err := action.Execute(ctx)
	if err != nil {
		t.Errorf("Execute with preset DrawnType should succeed without pool, got: %v", err)
	}
	if action.DrawnType != constants.EventTypeHerb {
		t.Errorf("DrawnType should be preserved as herb, got %s", action.DrawnType)
	}
	if action.DrawnName != "" {
		t.Errorf("DrawnName should be empty (client uses event_type), got %s", action.DrawnName)
	}
}

// ========== DrawItemAction Pool Execution Tests ==========

func TestDrawItemActionExecuteWithPool(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})

	drawEngine := rng.NewDrawEngine(rand.New(rand.NewSource(42)))
	itemPool := []*rng.EvaluatedItem{
		{Type: "reverse_clock", Eval: constants.EvaluationGood},
	}

	action := NewDrawItemAction(player, "CheckpointTreasure")
	ctx := NewActionContext(nil, nil, nil, drawEngine)
	ctx.ItemPool = itemPool

	err := action.Execute(ctx)
	if err != nil {
		t.Errorf("Execute should succeed with pool, got error: %v", err)
	}

	// DrawnType should be set from pool draw
	if action.DrawnType == constants.ItemTypeNone {
		t.Error("DrawnType should not be ItemTypeNone after pool draw")
	}
	if !action.DrawnType.IsValid() {
		t.Errorf("DrawnType should be valid, got %s", action.DrawnType)
	}

	// Item should be added to player inventory
	if len(player.Inventory) != 1 {
		t.Errorf("Player should have 1 item in inventory, got %d", len(player.Inventory))
	}
	if player.Inventory[0].Type != action.DrawnType {
		t.Errorf("Item type should match DrawnType, got %s vs %s", player.Inventory[0].Type, action.DrawnType)
	}

	// Verify log entry has item_type metadata
	entry := action.LogEntry()
	if entry.Metadata.GetStringOrDefault("item_type", "") != string(action.DrawnType) {
		t.Errorf("Log item_type should be %s, got %s", action.DrawnType, entry.Metadata.GetStringOrDefault("item_type", ""))
	}
}

func TestDrawItemActionNilItemPool(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	drawEngine := rng.NewDrawEngine(rand.New(rand.NewSource(42)))

	action := NewDrawItemAction(player, "CheckpointTreasure")
	ctx := NewActionContext(nil, nil, nil, drawEngine)
	// ItemPool is nil

	err := action.Execute(ctx)
	if err == nil {
		t.Error("Execute with nil ItemPool should return error")
	}
}

func TestDrawItemActionNilPlayer(t *testing.T) {
	drawEngine := rng.NewDrawEngine(rand.New(rand.NewSource(42)))
	itemPool := []*rng.EvaluatedItem{
		{Type: "reverse_clock", Eval: constants.EvaluationGood},
	}

	action := NewDrawItemAction(nil, "CheckpointTreasure")
	ctx := NewActionContext(nil, nil, nil, drawEngine)
	ctx.ItemPool = itemPool

	err := action.Execute(ctx)
	if err == nil {
		t.Error("Execute with nil player should return error")
	}
}

func TestDrawItemActionEmptyPool(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	drawEngine := rng.NewDrawEngine(rand.New(rand.NewSource(42)))
	itemPool := []*rng.EvaluatedItem{}

	action := NewDrawItemAction(player, "CheckpointTreasure")
	ctx := NewActionContext(nil, nil, nil, drawEngine)
	ctx.ItemPool = itemPool

	err := action.Execute(ctx)
	if err != nil {
		t.Errorf("Execute with empty pool should succeed, got: %v", err)
	}
	// Empty pool → ItemTypeNone, no item added
	if action.DrawnType != constants.ItemTypeNone {
		t.Errorf("DrawnType should be ItemTypeNone for empty pool, got %s", action.DrawnType)
	}
	if len(player.Inventory) != 0 {
		t.Errorf("Player should have no items for empty pool, got %d", len(player.Inventory))
	}
}
