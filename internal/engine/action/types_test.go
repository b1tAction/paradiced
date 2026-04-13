package action

import (
	"testing"

	"github.com/b1tAction/Fated/internal/core"
	"github.com/b1tAction/Fated/internal/core/buff"
)

// ========== ActionType Tests ==========

func TestActionTypeString(t *testing.T) {
	tests := []struct {
		at       ActionType
		expected string
	}{
		{ActionDamage, "Damage"},
		{ActionHeal, "Heal"},
		{ActionModifyLP, "ModifyLP"},
		{ActionMove, "Move"},
		{ActionAddBuff, "AddBuff"},
		{ActionRemoveBuff, "RemoveBuff"},
		{ActionRespawn, "Respawn"},
		{ActionSkipTurn, "SkipTurn"},
		{ActionDrawEvent, "DrawEvent"},
		{ActionTeleport, "Teleport"},
		{ActionStealBuff, "StealBuff"},
		{ActionType(999), "Unknown"},
	}

	for _, tt := range tests {
		if tt.at.String() != tt.expected {
			t.Errorf("ActionType(%d).String() = %s, want %s", tt.at, tt.at.String(), tt.expected)
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
	player := core.NewPlayer(core.PlayerConfig{UserID: "p1"})
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
	player := core.NewPlayer(core.PlayerConfig{UserID: "p1"})
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

// ========== TurnEventLog Tests ==========

func TestTurnEventLogAddEntry(t *testing.T) {
	log := NewTurnEventLog()

	if log.Len() != 0 {
		t.Error("New log should be empty")
	}

	entry1 := TurnEventLogEntry{
		Type:   "HPChange",
		Target: "p1",
		Delta:  -10,
		Source: "Event_Trap",
	}

	entry2 := TurnEventLogEntry{
		Type:   "LPChange",
		Target: "p1",
		Delta:  1,
		Source: "Buff_Divine",
	}

	log.AddEntry(entry1)
	log.AddEntry(entry2)

	if log.Len() != 2 {
		t.Errorf("Log length should be 2, got %d", log.Len())
	}

	entries := log.Entries()
	if len(entries) != 2 {
		t.Errorf("Entries length should be 2, got %d", len(entries))
	}
	if entries[0].Type != "HPChange" {
		t.Errorf("First entry type should be HPChange, got %s", entries[0].Type)
	}
	if entries[1].Type != "LPChange" {
		t.Errorf("Second entry type should be LPChange, got %s", entries[1].Type)
	}
}

func TestTurnEventLogToJSON(t *testing.T) {
	log := NewTurnEventLog()
	log.AddEntry(TurnEventLogEntry{
		Type:   "Move",
		Target: "p1",
		Delta:  5,
		Source: "DiceRoll",
	})

	jsonBytes, err := log.ToJSON()
	if err != nil {
		t.Errorf("ToJSON failed: %v", err)
	}

	// Verify JSON contains expected fields
	jsonStr := string(jsonBytes)
	if jsonStr == "" {
		t.Error("JSON should not be empty")
	}
}

func TestTurnEventLogClear(t *testing.T) {
	log := NewTurnEventLog()
	log.AddEntry(TurnEventLogEntry{Type: "Test", Target: "p1", Delta: 0, Source: "test"})
	log.Clear()

	if log.Len() != 0 {
		t.Error("Clear should empty log")
	}
}

// ========== DamageAction Tests ==========

func TestDamageAction(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{UserID: "p1"})
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
	if action.Target() != "p1" {
		t.Errorf("Target should be p1, got %s", action.Target())
	}

	// Execute with minimal context
	ctx := NewActionContext(nil, nil, nil)
	err := action.Execute(ctx)
	if err != nil {
		t.Errorf("Execute failed: %v", err)
	}

	if player.HP != 80 {
		t.Errorf("HP should be 80 after damage, got %d", player.HP)
	}

	// Verify log entry
	entry := action.LogEntry()
	if entry.Type != "HPChange" {
		t.Errorf("Log type should be HPChange, got %s", entry.Type)
	}
	if entry.Delta != -20 {
		t.Errorf("Log delta should be -20, got %d", entry.Delta)
	}
}

func TestPiercingDamageAction(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{UserID: "p1"})
	player.HP = 100

	action := NewPiercingDamageAction(player, 30, "Boss")

	if action.CanModify() {
		t.Error("Piercing damage should not be modifiable")
	}
	if !action.IsPiercing {
		t.Error("IsPiercing should be true")
	}

	ctx := NewActionContext(nil, nil, nil)
	action.Execute(ctx)

	if player.HP != 70 {
		t.Errorf("HP should be 70, got %d", player.HP)
	}
}

func TestBlockedDamageAction(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{UserID: "p1"})
	player.HP = 100

	action := NewDamageAction(player, 20, "Event_Trap")
	action.Amount = 0 // Blocked by interceptor
	action.BlockedBy = "Buff_Hidden"

	ctx := NewActionContext(nil, nil, nil)
	action.Execute(ctx)

	if player.HP != 100 {
		t.Errorf("Blocked damage should not affect HP, got %d", player.HP)
	}
}

// ========== HealAction Tests ==========

func TestHealAction(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{UserID: "p1", MaxHP: 100})
	player.HP = 50

	action := NewHealAction(player, 30, "Buff_Rain")

	if action.Type() != ActionHeal {
		t.Errorf("Type should be ActionHeal, got %s", action.Type())
	}
	if !action.CanModify() {
		t.Error("HealAction should be modifiable")
	}

	ctx := NewActionContext(nil, nil, nil)
	action.Execute(ctx)

	if player.HP != 80 {
		t.Errorf("HP should be 80 after heal, got %d", player.HP)
	}

	entry := action.LogEntry()
	if entry.Delta != 30 {
		t.Errorf("Log delta should be 30, got %d", entry.Delta)
	}
}

// ========== ModifyLPAction Tests ==========

func TestModifyLPAction(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{UserID: "p1"})
	player.LP = 5

	// LP+1
	actionPlus := NewModifyLPAction(player, 1, "Buff_Divine")
	if actionPlus.CanModify() {
		t.Error("ModifyLPAction should not be modifiable")
	}

	ctx := NewActionContext(nil, nil, nil)
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
	player := core.NewPlayer(core.PlayerConfig{UserID: "p1"})

	action := NewAddBuffAction(player, buff.BuffTypeDivine, 3, "Event_DivineGift")

	if action.Type() != ActionAddBuff {
		t.Errorf("Type should be ActionAddBuff, got %s", action.Type())
	}
	if action.CanModify() {
		t.Error("AddBuffAction should not be modifiable")
	}
	if action.BuffType != buff.BuffTypeDivine {
		t.Errorf("BuffType should be Divine, got %s", action.BuffType.String())
	}

	ctx := NewActionContext(nil, nil, nil)
	action.Execute(ctx)

	if len(player.ActiveBuffs) != 1 {
		t.Errorf("Player should have 1 buff, got %d", len(player.ActiveBuffs))
	}
	if player.ActiveBuffs[0].Type != buff.BuffTypeDivine {
		t.Error("Buff type should be Divine")
	}
	if player.ActiveBuffs[0].Duration != 3 {
		t.Errorf("Buff duration should be 3, got %d", player.ActiveBuffs[0].Duration)
	}

	entry := action.LogEntry()
	if entry.Type != "BuffAdd" {
		t.Errorf("Log type should be BuffAdd, got %s", entry.Type)
	}
}

// ========== RemoveBuffAction Tests ==========

func TestRemoveBuffAction(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{UserID: "p1"})
	player.AddBuff(buff.NewBuff(buff.BuffTypeCurse, 2))
	player.AddBuff(buff.NewBuff(buff.BuffTypeDivine, 3))

	if len(player.ActiveBuffs) != 2 {
		t.Error("Setup: player should have 2 buffs")
	}

	action := NewRemoveBuffAction(player, buff.BuffTypeCurse, "Manual_Remove")

	ctx := NewActionContext(nil, nil, nil)
	action.Execute(ctx)

	if len(player.ActiveBuffs) != 1 {
		t.Errorf("Player should have 1 buff after removal, got %d", len(player.ActiveBuffs))
	}
	if player.ActiveBuffs[0].Type != buff.BuffTypeDivine {
		t.Error("Remaining buff should be Divine")
	}
}

// ========== TeleportAction Tests ==========

func TestTeleportAction(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{UserID: "p1"})
	player.Position = 5

	action := NewTeleportAction(player, 20, "Item_AnyDoor")

	if action.Type() != ActionTeleport {
		t.Errorf("Type should be ActionTeleport, got %s", action.Type())
	}
	if action.TargetPos != 20 {
		t.Errorf("TargetPos should be 20, got %d", action.TargetPos)
	}

	ctx := NewActionContext(nil, nil, nil)
	action.Execute(ctx)

	if player.Position != 20 {
		t.Errorf("Position should be 20, got %d", player.Position)
	}

	entry := action.LogEntry()
	if entry.Type != "Teleport" {
		t.Errorf("Log type should be Teleport, got %s", entry.Type)
	}
}

// ========== StealBuffAction Tests ==========

func TestStealBuffAction(t *testing.T) {
	target := core.NewPlayer(core.PlayerConfig{UserID: "target"})
	source := core.NewPlayer(core.PlayerConfig{UserID: "source"})
	target.AddBuff(buff.NewBuff(buff.BuffTypeCurse, 2))
	target.AddBuff(buff.NewBuff(buff.BuffTypeDivine, 3))

	action := NewStealBuffAction(target, source, "Faction_BaiHu")

	if action.Type() != ActionStealBuff {
		t.Errorf("Type should be ActionStealBuff, got %s", action.Type())
	}
	if action.SourcePlayer.UserID != "source" {
		t.Errorf("SourcePlayer should be source, got %s", action.SourcePlayer.UserID)
	}

	ctx := NewActionContext(nil, nil, nil)
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
	target := core.NewPlayer(core.PlayerConfig{UserID: "target"})
	source := core.NewPlayer(core.PlayerConfig{UserID: "source"})

	action := NewStealBuffAction(target, source, "Faction_BaiHu")

	ctx := NewActionContext(nil, nil, nil)
	action.Execute(ctx)

	// Should not fail when target has no buffs
	if len(source.ActiveBuffs) != 0 {
		t.Error("Source should have no buffs when target has none")
	}
	if action.StolenBuff != nil {
		t.Error("StolenBuff should be nil when no buffs to steal")
	}
}

// ========== ActionContext Tests ==========

func TestActionContextExecuteAction(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{UserID: "p1"})
	player.HP = 100

	ctx := NewActionContext(nil, nil, nil)
	action := NewDamageAction(player, 10, "test")

	err := ctx.ExecuteAction(action)
	if err != nil {
		t.Errorf("ExecuteAction failed: %v", err)
	}

	// Verify action was executed
	if player.HP != 90 {
		t.Errorf("HP should be 90, got %d", player.HP)
	}

	// Verify log was recorded
	if ctx.EventLog.Len() != 1 {
		t.Errorf("EventLog should have 1 entry, got %d", ctx.EventLog.Len())
	}
}

func TestActionContextProcessQueue(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{UserID: "p1"})
	player.HP = 100

	ctx := NewActionContext(nil, nil, nil)

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

	// EventLog should have 2 entries
	if ctx.EventLog.Len() != 2 {
		t.Errorf("EventLog should have 2 entries, got %d", ctx.EventLog.Len())
	}
}

func TestActionContextClear(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{UserID: "p1"})
	ctx := NewActionContext(nil, nil, nil)

	ctx.PushDerivedAction(NewDamageAction(player, 10, "test"))
	ctx.EventLog.AddEntry(TurnEventLogEntry{Type: "Test"})

	ctx.Clear()

	if !ctx.ActionQueue.IsEmpty() {
		t.Error("Queue should be empty after Clear")
	}
	if ctx.EventLog.Len() != 0 {
		t.Error("EventLog should be empty after Clear")
	}
}

// ========== ActionContext Metadata Tests ==========

func TestActionContextMetadata(t *testing.T) {
	ctx := NewActionContext(nil, nil, nil)

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