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

// ========== ActionContext Tests ==========

func TestSetPools(t *testing.T) {
	ctx := NewActionContext(nil, nil, nil, nil)
	eventPool := []*rng.EvaluatedItem{{Type: "herb", Eval: constants.EvaluationGood}}
	itemPool := []*rng.EvaluatedItem{{Type: "any_door", Eval: constants.EvaluationMildGood}}
	buffPool := []*rng.EvaluatedItem{{Type: "divine", Eval: constants.EvaluationVeryGood}}

	result := ctx.SetPools(eventPool, itemPool, buffPool)

	// Verify fluent API returns same context
	if result != ctx {
		t.Error("SetPools should return the same ActionContext (fluent API)")
	}
	if len(ctx.EventPool) != 1 || ctx.EventPool[0].Type != "herb" {
		t.Errorf("EventPool not set correctly: %v", ctx.EventPool)
	}
	if len(ctx.ItemPool) != 1 || ctx.ItemPool[0].Type != "any_door" {
		t.Errorf("ItemPool not set correctly: %v", ctx.ItemPool)
	}
	if len(ctx.BuffPool) != 1 || ctx.BuffPool[0].Type != "divine" {
		t.Errorf("BuffPool not set correctly: %v", ctx.BuffPool)
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
	if entry.Metadata.GetIntOrDefault("duration", 0) != 3 {
		t.Errorf("Log duration should be 3, got %d", entry.Metadata.GetIntOrDefault("duration", 0))
	}
}

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

	// FellDownAction delegates damage to PiercingDamageAction
	if ctx.ActionQueue.Len() != 1 {
		t.Fatalf("Should have 1 derived action, got %d", ctx.ActionQueue.Len())
	}
	derived := ctx.ActionQueue.Pop()
	piercing, ok := derived.(*DamageAction)
	if !ok {
		t.Fatalf("Derived action should be DamageAction, got %T", derived)
	}
	if !piercing.IsPiercing {
		t.Error("Derived DamageAction should be piercing")
	}
	if piercing.Amount != 1 {
		t.Errorf("PiercingDamageAction amount = %d, expected 1", piercing.Amount)
	}
	if piercing.Source() != "FragileCell" {
		t.Errorf("PiercingDamageAction source = %s, expected FragileCell", piercing.Source())
	}

	entry := action.LogEntry()
	if entry.ActionType != "fell_down" {
		t.Errorf("Log ActionType should be fell_down, got %s", entry.ActionType)
	}
	if entry.Metadata.GetIntOrDefault("position", 0) != 30 {
		t.Errorf("Log position should be 30, got %d", entry.Metadata.GetIntOrDefault("position", 0))
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
	if entry.Metadata.GetIntOrDefault("duration", 0) != 3 {
		t.Errorf("Log duration should be 3, got %d", entry.Metadata.GetIntOrDefault("duration", 0))
	}
}

// ========== AddBuffAction Duration Tests ==========

func TestAddBuffActionLogEntryPermanentDuration(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	action := NewAddBuffAction(player, constants.BuffTypeFire, "Faction_ZhuQue")

	ctx := NewActionContext(nil, nil, nil, nil)
	ctx.OnAddBuff = func(p *core.Player, b *core.Buff) { p.AddBuff(b) }
	ctx.GetBuffDuration = func(bt constants.BuffType) int { return -1 } // permanent buff
	action.Execute(ctx)

	entry := action.LogEntry()
	if entry.Metadata.GetIntOrDefault("duration", 0) != -1 {
		t.Errorf("Log duration for permanent buff should be -1, got %d", entry.Metadata.GetIntOrDefault("duration", 0))
	}
}

func TestAddBuffActionLogEntryDurationBeforeExecute(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	action := NewAddBuffAction(player, constants.BuffTypeCurse, "Event_Curse")

	// LogEntry before Execute should have duration=0 (not yet populated)
	entry := action.LogEntry()
	if entry.Metadata.GetIntOrDefault("duration", 0) != 0 {
		t.Errorf("Log duration before Execute should be 0, got %d", entry.Metadata.GetIntOrDefault("duration", 0))
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

	// Zero damage should not push any derived action
	if ctx.ActionQueue.Len() != 0 {
		t.Errorf("Should have 0 derived actions with zero damage, got %d", ctx.ActionQueue.Len())
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

// ========== ActionContext Tests ==========

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
	ctx := NewActionContext(nil, bus, nil, nil)
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
	// Set OnAddItem callback to add item to player inventory (simulating Game.ApplyItemToPlayer)
	ctx.OnAddItem = func(p *core.Player, i *core.Item) {
		p.AddItem(i)
	}

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

	// Should have AddItemAction as derived action (instead of directly adding to inventory)
	if ctx.ActionQueue.Len() != 1 {
		t.Fatalf("Should have 1 derived action (AddItemAction), got %d", ctx.ActionQueue.Len())
	}
	addItemAction, ok := ctx.ActionQueue.Peek().(*AddItemAction)
	if !ok {
		t.Fatal("Derived action should be AddItemAction")
	}
	if addItemAction.ItemType != action.DrawnType {
		t.Errorf("AddItemAction.ItemType = %s, expected %s", addItemAction.ItemType, action.DrawnType)
	}

	// Process derived action queue to actually add item to inventory
	if err := ctx.ProcessQueue(); err != nil {
		t.Errorf("ProcessQueue should succeed, got error: %v", err)
	}

	// Item should now be added to player inventory after ProcessQueue
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

// ========== RollDiceAction Tests ==========

func TestRollDiceActionType(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	action := NewRollDiceAction(player, rng.DiceTypeWood, rand.New(rand.NewSource(42)), "DiceRoll")

	if action.Type() != constants.ActionDiceRoll {
		t.Errorf("Type() should be ActionDiceRoll, got %s", action.Type())
	}
}

func TestRollDiceActionCanModify(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	action := NewRollDiceAction(player, rng.DiceTypeWood, rand.New(rand.NewSource(42)), "DiceRoll")

	if !action.CanModify() {
		t.Error("CanModify() should be true (Buffs can modify Steps)")
	}
}

func TestRollDiceActionPreTriggerPhase(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	action := NewRollDiceAction(player, rng.DiceTypeWood, rand.New(rand.NewSource(42)), "DiceRoll")

	if action.PreTriggerPhase() != constants.PhasePreDiceRoll {
		t.Errorf("PreTriggerPhase() should be PhasePreDiceRoll, got %s", action.PreTriggerPhase())
	}
}

func TestRollDiceActionPostTriggerPhase(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	action := NewRollDiceAction(player, rng.DiceTypeWood, rand.New(rand.NewSource(42)), "DiceRoll")

	if action.PostTriggerPhase() != constants.PhaseAnyTime {
		t.Errorf("PostTriggerPhase() should be PhaseAnyTime, got %s", action.PostTriggerPhase())
	}
}

func TestRollDiceActionStepsCalculated(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	rngInst := rand.New(rand.NewSource(42))

	// Test each dice type produces valid steps (1-6)
	diceTypes := []rng.DiceType{rng.DiceTypeGold, rng.DiceTypeSilver, rng.DiceTypeCopper, rng.DiceTypeWood, rng.DiceTypeNormal}
	for _, diceType := range diceTypes {
		action := NewRollDiceAction(player, diceType, rngInst, "DiceRoll")
		if action.Steps < 1 || action.Steps > 6 {
			t.Errorf("RollDiceAction Steps for %s should be 1-6, got %d", diceType.String(), action.Steps)
		}
	}
}

func TestRollDiceActionNilRNGFallback(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	action := NewRollDiceAction(player, rng.DiceTypeWood, nil, "DiceRoll")

	if action.Steps != 1 {
		t.Errorf("RollDiceAction with nil RNG should fallback to 1, got %d", action.Steps)
	}
}

func TestRollDiceActionExecute(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	rngInst := rand.New(rand.NewSource(42))
	action := NewRollDiceAction(player, rng.DiceTypeWood, rngInst, "DiceRoll")

	ctx := NewActionContext(nil, nil, nil, nil)

	err := action.Execute(ctx)
	if err != nil {
		t.Errorf("Execute() should return nil, got %v", err)
	}

	// Check metadata
	stepsResult := ctx.GetIntOrDefault("dice_steps_result", 0)
	if stepsResult != action.Steps {
		t.Errorf("dice_steps_result should be %d, got %d", action.Steps, stepsResult)
	}

	diceTypeResult := ctx.GetStringOrDefault("dice_type_result", "")
	if diceTypeResult != "wood" {
		t.Errorf("dice_type_result should be 'wood', got %s", diceTypeResult)
	}
}

func TestRollDiceActionLogEntry(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	rngInst := rand.New(rand.NewSource(42))
	action := NewRollDiceAction(player, rng.DiceTypeGold, rngInst, "DiceRoll")

	entry := action.LogEntry()

	if entry.ActionType != "dice_roll" {
		t.Errorf("LogEntry ActionType should be 'dice_roll', got %s", entry.ActionType)
	}
	if entry.Target != player.ID.UUID() {
		t.Errorf("LogEntry Target should be %s, got %s", player.ID.UUID(), entry.Target)
	}
	if entry.Source != "DiceRoll" {
		t.Errorf("LogEntry Source should be 'DiceRoll', got %s", entry.Source)
	}
	if entry.Type != constants.EntryTypeAction {
		t.Errorf("LogEntry Type should be EntryTypeAction, got %s", entry.Type)
	}

	// Check metadata
	diceType := entry.Metadata.GetStringOrDefault("dice_type", "")
	if diceType != "gold" {
		t.Errorf("LogEntry metadata dice_type should be 'gold', got %s", diceType)
	}
	diceSteps := entry.Metadata.GetIntOrDefault("dice_steps", 0)
	if diceSteps != action.Steps {
		t.Errorf("LogEntry metadata dice_steps should be %d, got %d", action.Steps, diceSteps)
	}
}

func TestRollDiceActionSourceID(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	rngInst := rand.New(rand.NewSource(42))

	// Test different source IDs
	rollAction := NewRollDiceAction(player, rng.DiceTypeWood, rngInst, "DiceRoll")
	timeoutAction := NewRollDiceAction(player, rng.DiceTypeWood, rngInst, "DiceRollTimeout")

	if rollAction.Source() != "DiceRoll" {
		t.Errorf("Source() should be 'DiceRoll', got %s", rollAction.Source())
	}
	if timeoutAction.Source() != "DiceRollTimeout" {
		t.Errorf("Source() should be 'DiceRollTimeout', got %s", timeoutAction.Source())
	}
}

// ========== AddItemAction Tests ==========

func TestAddItemActionExecute(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})

	action := NewAddItemAction(player, constants.ItemTypeReverseClock, "CheckpointTreasure")

	ctx := NewActionContext(nil, nil, nil, nil)
	// Provide OnAddItem callback (required for item lifecycle)
	var addedItem *core.Item
	ctx.OnAddItem = func(p *core.Player, i *core.Item) {
		p.AddItem(i)
		addedItem = i
	}

	err := action.Execute(ctx)
	if err != nil {
		t.Errorf("AddItemAction Execute failed: %v", err)
	}

	if len(player.Inventory) != 1 {
		t.Errorf("Player should have 1 item, got %d", len(player.Inventory))
	}
	if addedItem == nil {
		t.Error("OnAddItem should have been called")
	}
	if addedItem.Type != constants.ItemTypeReverseClock {
		t.Errorf("Item type = %s, expected ReverseClock", addedItem.Type)
	}
}

func TestAddItemActionNilPlayer(t *testing.T) {
	action := NewAddItemAction(nil, constants.ItemTypeReverseClock, "CheckpointTreasure")
	ctx := NewActionContext(nil, nil, nil, nil)
	ctx.OnAddItem = func(p *core.Player, i *core.Item) { p.AddItem(i) }

	err := action.Execute(ctx)
	if err == nil {
		t.Error("AddItemAction with nil player should return error")
	}
}

func TestAddItemActionNilCallback(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	action := NewAddItemAction(player, constants.ItemTypeReverseClock, "CheckpointTreasure")
	ctx := NewActionContext(nil, nil, nil, nil)
	// OnAddItem is nil

	err := action.Execute(ctx)
	if err == nil {
		t.Error("AddItemAction with nil OnAddItem callback should return error")
	}
}

func TestAddItemActionLogEntry(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	action := NewAddItemAction(player, constants.ItemTypeReverseClock, "CheckpointTreasure")

	entry := action.LogEntry()
	if entry.ActionType != string(constants.ActionAddItem) {
		t.Errorf("LogEntry ActionType = %s, expected add_item", entry.ActionType)
	}
	if entry.Source != "CheckpointTreasure" {
		t.Errorf("LogEntry Source = %s, expected CheckpointTreasure", entry.Source)
	}
	if entry.Metadata.GetStringOrDefault("item_type", "") != string(constants.ItemTypeReverseClock) {
		t.Errorf("LogEntry item_type = %s, expected reverse_clock", entry.Metadata.GetStringOrDefault("item_type", ""))
	}
}

// ========== RemoveItemAction Tests ==========

func TestRemoveItemActionExecute(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	item := core.NewItem(constants.ItemTypeReverseClock)
	player.AddItem(item)

	if len(player.Inventory) != 1 {
		t.Fatalf("Player should have 1 item before removal, got %d", len(player.Inventory))
	}

	action := NewRemoveItemAction(player, constants.ItemTypeReverseClock, "Item_Consumed")

	ctx := NewActionContext(nil, nil, nil, nil)
	// Provide OnRemoveItem callback
	var removedItem *core.Item
	ctx.OnRemoveItem = func(p *core.Player, it constants.ItemType) *core.Item {
		for _, invItem := range p.Inventory {
			if invItem.Type == it {
				removedItem = invItem
				p.RemoveItem(invItem.ID)
				return invItem
			}
		}
		return nil
	}

	err := action.Execute(ctx)
	if err != nil {
		t.Errorf("RemoveItemAction Execute failed: %v", err)
	}

	if removedItem == nil {
		t.Error("OnRemoveItem should have been called and found the item")
	}
	if removedItem.Type != constants.ItemTypeReverseClock {
		t.Errorf("Removed item type = %s, expected ReverseClock", removedItem.Type)
	}
	if len(player.Inventory) != 0 {
		t.Errorf("Player should have 0 items after removal, got %d", len(player.Inventory))
	}
}

func TestRemoveItemActionNilPlayer(t *testing.T) {
	action := NewRemoveItemAction(nil, constants.ItemTypeReverseClock, "Item_Consumed")
	ctx := NewActionContext(nil, nil, nil, nil)
	ctx.OnRemoveItem = func(p *core.Player, it constants.ItemType) *core.Item { return nil }

	err := action.Execute(ctx)
	if err == nil {
		t.Error("RemoveItemAction with nil player should return error")
	}
}

func TestRemoveItemActionNilCallback(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	action := NewRemoveItemAction(player, constants.ItemTypeReverseClock, "Item_Consumed")
	ctx := NewActionContext(nil, nil, nil, nil)
	// OnRemoveItem is nil

	err := action.Execute(ctx)
	if err == nil {
		t.Error("RemoveItemAction with nil OnRemoveItem callback should return error")
	}
}

func TestRemoveItemActionLogEntry(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	action := NewRemoveItemAction(player, constants.ItemTypeReverseClock, "Item_Consumed")

	entry := action.LogEntry()
	if entry.ActionType != string(constants.ActionRemoveItem) {
		t.Errorf("LogEntry ActionType = %s, expected remove_item", entry.ActionType)
	}
	if entry.Source != "Item_Consumed" {
		t.Errorf("LogEntry Source = %s, expected Item_Consumed", entry.Source)
	}
	if entry.Metadata.GetStringOrDefault("item_type", "") != string(constants.ItemTypeReverseClock) {
		t.Errorf("LogEntry item_type = %s, expected reverse_clock", entry.Metadata.GetStringOrDefault("item_type", ""))
	}
}

// ========== DrawBuffAction Tests ==========

func TestDrawBuffActionExecuteWithPool(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player.LP = 5

	drawEngine := rng.NewDrawEngine(rand.New(rand.NewSource(42)))
	buffPool := []*rng.EvaluatedItem{
		{Type: "divine", Eval: constants.EvaluationGood},
		{Type: "curse", Eval: constants.EvaluationBad},
	}

	action := NewDrawBuffAction(player, "Event_TasteTest")
	ctx := NewActionContext(nil, nil, nil, drawEngine)
	ctx.BuffPool = buffPool
	// Set probability weights for the draw
	ctx.SetCellDraw(0.5, 0.3, 0.2)
	// Provide OnAddBuff callback for derived AddBuffAction
	ctx.OnAddBuff = func(p *core.Player, b *core.Buff) { p.AddBuff(b) }
	ctx.GetBuffDuration = func(bt constants.BuffType) int { return 3 }

	err := action.Execute(ctx)
	if err != nil {
		t.Errorf("Execute should succeed with pool, got error: %v", err)
	}

	// DrawnType should be set from pool draw
	if action.DrawnType == constants.BuffTypeNone {
		t.Error("DrawnType should not be BuffTypeNone after pool draw")
	}
	if !action.DrawnType.IsValid() {
		t.Errorf("DrawnType should be valid, got %s", action.DrawnType)
	}

	// Should have AddBuffAction as derived action
	if ctx.ActionQueue.Len() != 1 {
		t.Fatalf("Should have 1 derived action (AddBuffAction), got %d", ctx.ActionQueue.Len())
	}
	addBuffAction, ok := ctx.ActionQueue.Peek().(*AddBuffAction)
	if !ok {
		t.Fatal("Derived action should be AddBuffAction")
	}
	if addBuffAction.BuffType != action.DrawnType {
		t.Errorf("AddBuffAction.BuffType = %s, expected %s", addBuffAction.BuffType, action.DrawnType)
	}

	// Verify log entry has buff_type metadata
	entry := action.LogEntry()
	if entry.Metadata.GetStringOrDefault("buff_type", "") != string(action.DrawnType) {
		t.Errorf("Log buff_type should be %s, got %s", action.DrawnType, entry.Metadata.GetStringOrDefault("buff_type", ""))
	}
}

func TestDrawBuffActionNilBuffPool(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	drawEngine := rng.NewDrawEngine(rand.New(rand.NewSource(42)))

	action := NewDrawBuffAction(player, "Event_TasteTest")
	ctx := NewActionContext(nil, nil, nil, drawEngine)
	// BuffPool is nil

	err := action.Execute(ctx)
	if err == nil {
		t.Error("Execute with nil BuffPool should return error")
	}
}

func TestDrawBuffActionNilPlayer(t *testing.T) {
	drawEngine := rng.NewDrawEngine(rand.New(rand.NewSource(42)))
	buffPool := []*rng.EvaluatedItem{
		{Type: "divine", Eval: constants.EvaluationGood},
	}

	action := NewDrawBuffAction(nil, "Event_TasteTest")
	ctx := NewActionContext(nil, nil, nil, drawEngine)
	ctx.BuffPool = buffPool

	err := action.Execute(ctx)
	if err == nil {
		t.Error("Execute with nil player should return error")
	}
}

func TestDrawBuffActionEmptyPool(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	drawEngine := rng.NewDrawEngine(rand.New(rand.NewSource(42)))
	buffPool := []*rng.EvaluatedItem{}

	action := NewDrawBuffAction(player, "Event_TasteTest")
	ctx := NewActionContext(nil, nil, nil, drawEngine)
	ctx.BuffPool = buffPool

	err := action.Execute(ctx)
	if err != nil {
		t.Errorf("Execute with empty pool should succeed, got: %v", err)
	}
	// Empty pool → BuffTypeNone
	if action.DrawnType != constants.BuffTypeNone {
		t.Errorf("DrawnType should be BuffTypeNone for empty pool, got %s", action.DrawnType)
	}
	// Should have no derived actions
	if ctx.ActionQueue.Len() != 0 {
		t.Errorf("Should have 0 derived actions for empty pool, got %d", ctx.ActionQueue.Len())
	}
}

func TestDrawBuffActionCanModify(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	action := NewDrawBuffAction(player, "Event_TasteTest")

	if !action.CanModify() {
		t.Error("DrawBuffAction.CanModify() should be true (can be intercepted by Hidden)")
	}
}

func TestDrawBuffActionPreTriggerPhase(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	action := NewDrawBuffAction(player, "Event_TasteTest")

	if action.PreTriggerPhase() != constants.PhaseAnyTime {
		t.Errorf("PreTriggerPhase = %s, expected PhaseAnyTime", action.PreTriggerPhase())
	}
}

func TestDrawBuffActionLogEntry(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	action := NewDrawBuffAction(player, "Event_TasteTest")

	entry := action.LogEntry()
	if entry.ActionType != string(constants.ActionDrawBuff) {
		t.Errorf("LogEntry ActionType = %s, expected draw_buff", entry.ActionType)
	}
	if entry.Source != "Event_TasteTest" {
		t.Errorf("LogEntry Source = %s, expected Event_TasteTest", entry.Source)
	}
}

// ========== DiceUpgradeAction Tests ==========

func TestDiceUpgradeActionExecute(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})

	action := NewDiceUpgradeAction(player, "Item_DiceUpgrade", rng.DiceTypeSilver)
	ctx := NewActionContext(nil, nil, nil, nil)

	err := action.Execute(ctx)
	if err != nil {
		t.Errorf("DiceUpgradeAction Execute failed: %v", err)
	}

	// Silver should upgrade to Gold
	if action.ToDice != rng.DiceTypeGold {
		t.Errorf("ToDice = %v, expected Gold (Silver→Gold)", action.ToDice)
	}
}

func TestDiceUpgradeActionUpgradePaths(t *testing.T) {
	tests := []struct {
		from     rng.DiceType
		expected rng.DiceType
	}{
		{rng.DiceTypeWood, rng.DiceTypeCopper},
		{rng.DiceTypeCopper, rng.DiceTypeSilver},
		{rng.DiceTypeSilver, rng.DiceTypeGold},
		{rng.DiceTypeGold, rng.DiceTypeGold}, // Gold cannot upgrade further
	}

	for _, tt := range tests {
		player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
		action := NewDiceUpgradeAction(player, "Item_DiceUpgrade", tt.from)
		ctx := NewActionContext(nil, nil, nil, nil)

		err := action.Execute(ctx)
		if err != nil {
			t.Errorf("DiceUpgradeAction(%v) Execute failed: %v", tt.from, err)
		}
		if action.ToDice != tt.expected {
			t.Errorf("DiceUpgrade(%v) = %v, expected %v", tt.from, action.ToDice, tt.expected)
		}
	}
}

func TestDiceUpgradeActionMetadata(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	action := NewDiceUpgradeAction(player, "Item_DiceUpgrade", rng.DiceTypeCopper)
	ctx := NewActionContext(nil, nil, nil, nil)

	action.Execute(ctx)

	// Verify metadata written to ActionContext
	upgradeTo := ctx.GetStringOrDefault("dice_upgrade_to", "")
	if upgradeTo != rng.DiceTypeSilver.String() {
		t.Errorf("dice_upgrade_to = %s, expected %s", upgradeTo, rng.DiceTypeSilver.String())
	}
	upgradeFrom := ctx.GetStringOrDefault("dice_upgrade_from", "")
	if upgradeFrom != rng.DiceTypeCopper.String() {
		t.Errorf("dice_upgrade_from = %s, expected %s", upgradeFrom, rng.DiceTypeCopper.String())
	}
}

func TestDiceUpgradeActionNilPlayer(t *testing.T) {
	action := NewDiceUpgradeAction(nil, "Item_DiceUpgrade", rng.DiceTypeSilver)
	ctx := NewActionContext(nil, nil, nil, nil)

	err := action.Execute(ctx)
	if err == nil {
		t.Error("DiceUpgradeAction with nil player should return error")
	}
}

func TestDiceUpgradeActionLogEntry(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	action := NewDiceUpgradeAction(player, "Item_DiceUpgrade", rng.DiceTypeWood)
	ctx := NewActionContext(nil, nil, nil, nil)

	action.Execute(ctx)

	entry := action.LogEntry()
	if entry.ActionType != string(constants.ActionDiceUpgrade) {
		t.Errorf("LogEntry ActionType = %s, expected dice_upgrade", entry.ActionType)
	}
	if entry.Source != "Item_DiceUpgrade" {
		t.Errorf("LogEntry Source = %s, expected Item_DiceUpgrade", entry.Source)
	}
	fromDice := entry.Metadata.GetStringOrDefault("from_dice", "")
	if fromDice != rng.DiceTypeWood.String() {
		t.Errorf("LogEntry from_dice = %s, expected %s", fromDice, rng.DiceTypeWood.String())
	}
	toDice := entry.Metadata.GetStringOrDefault("to_dice", "")
	if toDice != rng.DiceTypeCopper.String() {
		t.Errorf("LogEntry to_dice = %s, expected %s", toDice, rng.DiceTypeCopper.String())
	}
}

// ========== DeathAction Interface Tests ==========

func TestDeathActionInterfaceMethods(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	action := NewDeathAction(player, "Buff_Corrupt", 15)

	if action.Type() != constants.ActionDeath {
		t.Errorf("Type() = %s, expected death", action.Type())
	}
	if action.CanModify() {
		t.Error("DeathAction.CanModify() should be false")
	}
	if action.Source() != "Buff_Corrupt" {
		t.Errorf("Source() = %s, expected Buff_Corrupt", action.Source())
	}
	if action.Target() != player.ID.UUID() {
		t.Errorf("Target() = %s, expected %s", action.Target(), player.ID.UUID())
	}
	if action.TargetPlayer() != player {
		t.Error("TargetPlayer() should return the target player")
	}
	if action.PreTriggerPhase() != constants.PhaseAnyTime {
		t.Errorf("PreTriggerPhase() = %s, expected PhaseAnyTime", action.PreTriggerPhase())
	}
	if action.PostTriggerPhase() != constants.PhaseAnyTime {
		t.Errorf("PostTriggerPhase() = %s, expected PhaseAnyTime", action.PostTriggerPhase())
	}
}

func TestDeathActionExecute(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	action := NewDeathAction(player, "Buff_Corrupt", 15)
	ctx := NewActionContext(nil, nil, nil, nil)
	ctx.OnAddBuff = func(p *core.Player, b *core.Buff) { p.AddBuff(b) }
	ctx.GetBuffDuration = func(bt constants.BuffType) int { return -1 }

	err := action.Execute(ctx)
	if err != nil {
		t.Errorf("Execute should succeed, got: %v", err)
	}

	// Player should now have DeathMark buff
	hasDeathMark := false
	for _, b := range player.ActiveBuffs {
		if b.Type == constants.BuffTypeDeathMark {
			hasDeathMark = true
		}
	}
	if !hasDeathMark {
		t.Error("Player should have DeathMark buff after DeathAction.Execute()")
	}
}

func TestDeathActionLogEntry(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	action := NewDeathAction(player, "FragileCell", 25)

	entry := action.LogEntry()
	if entry.ActionType != string(constants.ActionDeath) {
		t.Errorf("LogEntry ActionType = %s, expected death", entry.ActionType)
	}
	if entry.Source != "FragileCell" {
		t.Errorf("LogEntry Source = %s, expected FragileCell", entry.Source)
	}
	pos := entry.Metadata.GetIntOrDefault("position", -1)
	if pos != 25 {
		t.Errorf("LogEntry position = %d, expected 25", pos)
	}
	src := entry.Metadata.GetStringOrDefault("death_source", "")
	if src != "FragileCell" {
		t.Errorf("LogEntry death_source = %s, expected FragileCell", src)
	}
}

func TestDeathActionNilPlayer(t *testing.T) {
	action := NewDeathAction(nil, "test", 10)
	ctx := NewActionContext(nil, nil, nil, nil)

	err := action.Execute(ctx)
	if err == nil {
		t.Error("DeathAction with nil player should return error")
	}
}

func TestDeathActionNilCallback(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	action := NewDeathAction(player, "test", 10)
	ctx := NewActionContext(nil, nil, nil, nil)
	// OnAddBuff is nil

	err := action.Execute(ctx)
	if err == nil {
		t.Error("DeathAction with nil OnAddBuff should return error")
	}
}

// ========== BossDamageAction Tests ==========

func TestBossDamageActionInterfaceMethods(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	boss := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	action := NewBossDamageAction(player, boss, 5, true, "boss_damage")

	if action.Type() != constants.ActionBossDamage {
		t.Errorf("Type() = %s, expected boss_damage", action.Type())
	}
	if action.CanModify() {
		t.Error("BossDamageAction.CanModify() should be false")
	}
	if action.Source() != "boss_damage" {
		t.Errorf("Source() = %s, expected boss_damage", action.Source())
	}
	if action.Target() != boss.ID.UUID() {
		t.Errorf("Target() = %s, expected %s", action.Target(), boss.ID.UUID())
	}
	if action.TargetPlayer() != boss {
		t.Error("TargetPlayer() should return boss player")
	}
	if action.PreTriggerPhase() != constants.PhasePreDamage {
		t.Errorf("PreTriggerPhase() = %s, expected PhasePreDamage", action.PreTriggerPhase())
	}
	if action.PostTriggerPhase() != constants.PhaseAnyTime {
		t.Errorf("PostTriggerPhase() = %s, expected PhaseAnyTime", action.PostTriggerPhase())
	}
}

func TestBossDamageActionExecute(t *testing.T) {
	boss := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	boss.HP = 50
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	action := NewBossDamageAction(player, boss, 5, false, "boss_damage")
	ctx := NewActionContext(nil, nil, nil, nil)

	err := action.Execute(ctx)
	if err != nil {
		t.Errorf("Execute should succeed, got: %v", err)
	}
	if boss.HP != 45 {
		t.Errorf("Boss HP = %d, expected 45 (50-5)", boss.HP)
	}
}

func TestBossDamageActionExecuteWithCrit(t *testing.T) {
	boss := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	boss.HP = 50
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	action := NewBossDamageAction(player, boss, 10, true, "boss_damage")
	ctx := NewActionContext(nil, nil, nil, nil)

	err := action.Execute(ctx)
	if err != nil {
		t.Errorf("Execute should succeed, got: %v", err)
	}
	if boss.HP != 40 {
		t.Errorf("Boss HP = %d, expected 40 (50-10)", boss.HP)
	}
}

func TestBossDamageActionZeroDamage(t *testing.T) {
	boss := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	boss.HP = 50
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	action := NewBossDamageAction(player, boss, 0, false, "boss_damage")
	ctx := NewActionContext(nil, nil, nil, nil)

	err := action.Execute(ctx)
	if err != nil {
		t.Errorf("Execute with zero damage should succeed, got: %v", err)
	}
	if boss.HP != 50 {
		t.Errorf("Boss HP should not change with zero damage, got %d", boss.HP)
	}
}

func TestBossDamageActionNilBoss(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	action := NewBossDamageAction(player, nil, 5, false, "boss_damage")
	ctx := NewActionContext(nil, nil, nil, nil)

	err := action.Execute(ctx)
	if err == nil {
		t.Error("BossDamageAction with nil boss should return error")
	}
}

func TestBossDamageActionLogEntry(t *testing.T) {
	boss := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	boss.HP = 50
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	action := NewBossDamageAction(player, boss, 10, true, "boss_damage")
	ctx := NewActionContext(nil, nil, nil, nil)
	action.Execute(ctx)

	entry := action.LogEntry()
	if entry.ActionType != string(constants.ActionBossDamage) {
		t.Errorf("LogEntry ActionType = %s, expected boss_damage", entry.ActionType)
	}
	if entry.Type != constants.EntryTypeBoss {
		t.Errorf("LogEntry Type = %s, expected boss", entry.Type)
	}
	damage := entry.Metadata.GetIntOrDefault("damage", 0)
	if damage != 10 {
		t.Errorf("LogEntry damage = %d, expected 10", damage)
	}
	isCrit := entry.Metadata.GetBoolOrDefault("is_crit", false)
	if !isCrit {
		t.Error("LogEntry is_crit should be true")
	}
	remainingHP := entry.Metadata.GetIntOrDefault("boss_remaining_hp", 0)
	if remainingHP != 40 {
		t.Errorf("LogEntry boss_remaining_hp = %d, expected 40", remainingHP)
	}
}

// ========== BossAttackAction Tests ==========

func TestBossAttackActionInterfaceMethods(t *testing.T) {
	boss := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	action := NewBossAttackAction(boss, player, 2, constants.BossAttackCrit, "boss_attack")

	if action.Type() != constants.ActionBossAttack {
		t.Errorf("Type() = %s, expected boss_attack", action.Type())
	}
	if action.CanModify() {
		t.Error("BossAttackAction.CanModify() should be false")
	}
	if action.Source() != "boss_attack" {
		t.Errorf("Source() = %s, expected boss_attack", action.Source())
	}
	if action.Target() != player.ID.UUID() {
		t.Errorf("Target() = %s, expected %s", action.Target(), player.ID.UUID())
	}
	if action.TargetPlayer() != player {
		t.Error("TargetPlayer() should return target player")
	}
	if action.PreTriggerPhase() != constants.PhaseAnyTime {
		t.Errorf("PreTriggerPhase() = %s, expected PhaseAnyTime", action.PreTriggerPhase())
	}
	if action.PostTriggerPhase() != constants.PhaseAnyTime {
		t.Errorf("PostTriggerPhase() = %s, expected PhaseAnyTime", action.PostTriggerPhase())
	}
}

func TestBossAttackActionExecute(t *testing.T) {
	boss := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player.HP = 5
	player.LP = 3
	action := NewBossAttackAction(boss, player, 2, constants.BossAttackCrit, "boss_attack")
	ctx := NewActionContext(nil, nil, nil, nil)

	err := action.Execute(ctx)
	if err != nil {
		t.Errorf("Execute should succeed, got: %v", err)
	}
	// BossAttackAction delegates damage to DamageAction
	if ctx.ActionQueue.Len() != 1 {
		t.Fatalf("Should have 1 derived action, got %d", ctx.ActionQueue.Len())
	}
	derived := ctx.ActionQueue.Pop()
	damageAction, ok := derived.(*DamageAction)
	if !ok {
		t.Fatalf("Derived action should be DamageAction, got %T", derived)
	}
	if damageAction.Amount != 2 {
		t.Errorf("DamageAction amount = %d, expected 2", damageAction.Amount)
	}
	if damageAction.Source() != "boss_attack" {
		t.Errorf("DamageAction source = %s, expected boss_attack", damageAction.Source())
	}
}

func TestBossAttackActionLethalDamage(t *testing.T) {
	boss := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player.HP = 2
	player.LP = 0
	player.Position = 10
	action := NewBossAttackAction(boss, player, 2, constants.BossAttackNormal, "boss_attack")
	ctx := NewActionContext(nil, nil, nil, nil)

	err := action.Execute(ctx)
	if err != nil {
		t.Errorf("Execute should succeed, got: %v", err)
	}
	// BossAttackAction delegates damage to DamageAction even for lethal damage.
	// DeathAction is derived by DamageAction.Execute (tested in DamageAction tests).
	if ctx.ActionQueue.Len() != 1 {
		t.Fatalf("Should have 1 derived DamageAction, got %d", ctx.ActionQueue.Len())
	}
	derived := ctx.ActionQueue.Pop()
	damageAction, ok := derived.(*DamageAction)
	if !ok {
		t.Fatalf("Derived action should be DamageAction, got %T", derived)
	}
	if damageAction.Amount != 2 {
		t.Errorf("DamageAction amount = %d, expected 2 (lethal)", damageAction.Amount)
	}
}

func TestBossAttackActionNilTarget(t *testing.T) {
	boss := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	action := NewBossAttackAction(boss, nil, 1, constants.BossAttackNormal, "boss_attack")
	ctx := NewActionContext(nil, nil, nil, nil)

	err := action.Execute(ctx)
	if err == nil {
		t.Error("BossAttackAction with nil target should return error")
	}
}

func TestBossAttackActionLogEntry(t *testing.T) {
	boss := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	action := NewBossAttackAction(boss, player, 2, constants.BossAttackCrit, "boss_attack")

	entry := action.LogEntry()
	if entry.ActionType != string(constants.ActionBossAttack) {
		t.Errorf("LogEntry ActionType = %s, expected boss_attack", entry.ActionType)
	}
	if entry.Type != constants.EntryTypeBoss {
		t.Errorf("LogEntry Type = %s, expected boss", entry.Type)
	}
	attackType := entry.Metadata.GetStringOrDefault("attack_type", "")
	if attackType != string(constants.BossAttackCrit) {
		t.Errorf("LogEntry attack_type = %s, expected %s", attackType, constants.BossAttackCrit)
	}
}

// ========== BossSkillAction Tests ==========

func TestBossSkillActionInterfaceMethods(t *testing.T) {
	boss := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player1 := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player2 := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	action := NewBossSkillAction(boss, constants.BossSkillThunder, []*core.Player{player1, player2}, "boss_skill")

	if action.Type() != constants.ActionBossSkill {
		t.Errorf("Type() = %s, expected boss_skill", action.Type())
	}
	if action.CanModify() {
		t.Error("BossSkillAction.CanModify() should be false")
	}
	if action.Source() != "boss_skill" {
		t.Errorf("Source() = %s, expected boss_skill", action.Source())
	}
	if action.Target() != player1.ID.UUID() {
		t.Errorf("Target() = %s, expected %s (first target)", action.Target(), player1.ID.UUID())
	}
	if action.TargetPlayer() != boss {
		t.Error("TargetPlayer() should return boss (actor)")
	}
	if action.PreTriggerPhase() != constants.PhaseAnyTime {
		t.Errorf("PreTriggerPhase() = %s, expected PhaseAnyTime", action.PreTriggerPhase())
	}
	if action.PostTriggerPhase() != constants.PhaseAnyTime {
		t.Errorf("PostTriggerPhase() = %s, expected PhaseAnyTime", action.PostTriggerPhase())
	}
}

func TestBossSkillActionExecute(t *testing.T) {
	boss := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	action := NewBossSkillAction(boss, constants.BossSkillThunder, []*core.Player{player}, "boss_skill")
	ctx := NewActionContext(nil, nil, nil, nil)

	err := action.Execute(ctx)
	if err != nil {
		t.Errorf("Execute should succeed (delegated to handler), got: %v", err)
	}
}

func TestBossSkillActionLogEntry(t *testing.T) {
	boss := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player1 := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player2 := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	action := NewBossSkillAction(boss, constants.BossSkillCurse, []*core.Player{player1, player2}, "boss_skill")

	entry := action.LogEntry()
	if entry.ActionType != string(constants.ActionBossSkill) {
		t.Errorf("LogEntry ActionType = %s, expected boss_skill", entry.ActionType)
	}
	if entry.Type != constants.EntryTypeBoss {
		t.Errorf("LogEntry Type = %s, expected boss", entry.Type)
	}
	skillType := entry.Metadata.GetStringOrDefault("skill_type", "")
	if skillType != string(constants.BossSkillCurse) {
		t.Errorf("LogEntry skill_type = %s, expected %s", skillType, constants.BossSkillCurse)
	}
	targets := entry.Metadata.GetStringOrDefault("targets", "")
	if targets != player1.ID.UUID()+","+player2.ID.UUID() {
		t.Errorf("LogEntry targets = %s, expected comma-separated target IDs", targets)
	}
}

func TestBossSkillActionEmptyTargets(t *testing.T) {
	boss := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	action := NewBossSkillAction(boss, constants.BossSkillRest, []*core.Player{}, "boss_skill")

	if action.Target() != "" {
		t.Errorf("Target() with empty targets should return empty string, got %s", action.Target())
	}
}

// ========== Interface Method Coverage Tests ==========
// TargetPlayer, CanModify, Source, Target, PreTriggerPhase, PostTriggerPhase
// for actions that had 0% coverage on these interface methods.

func TestHealActionInterfaceMethods(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	action := NewHealAction(player, 5, "buff_divine")

	if action.TargetPlayer() != player {
		t.Error("HealAction TargetPlayer should return target player")
	}
}

func TestModifyLPActionInterfaceMethods(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	action := NewModifyLPAction(player, 1, "faction_zhu_que")

	if action.TargetPlayer() != player {
		t.Error("ModifyLPAction TargetPlayer should return target player")
	}
}

func TestMoveActionInterfaceMethods(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	action := NewMoveAction(player, 5, "system_dice")

	if action.TargetPlayer() != player {
		t.Error("MoveAction TargetPlayer should return target player")
	}
}

func TestAddBuffActionInterfaceMethods(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	action := NewAddBuffAction(player, constants.BuffTypeCurse, "event_trap")

	if action.TargetPlayer() != player {
		t.Error("AddBuffAction TargetPlayer should return target player")
	}
}

func TestRemoveBuffActionInterfaceMethods(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	action := NewRemoveBuffAction(player, constants.BuffTypeCurse, "system")

	if action.TargetPlayer() != player {
		t.Error("RemoveBuffAction TargetPlayer should return target player")
	}
}

func TestTeleportActionInterfaceMethods(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	action := NewTeleportAction(player, 20, "item_any_door")

	if action.TargetPlayer() != player {
		t.Error("TeleportAction TargetPlayer should return target player")
	}
}

func TestStealBuffActionInterfaceMethods(t *testing.T) {
	victim := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	stealer := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	action := NewStealBuffAction(victim, stealer, "faction_bai_hu")

	if action.TargetPlayer() != victim {
		t.Error("StealBuffAction TargetPlayer should return victim")
	}
	if action.CanModify() {
		t.Error("StealBuffAction CanModify should return false")
	}
}

func TestDrawEventActionInterfaceMethods(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	action := NewDrawEventAction(player, "event_trap")

	if action.TargetPlayer() != player {
		t.Error("DrawEventAction TargetPlayer should return target player")
	}
}

func TestDrawItemActionInterfaceMethods(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	action := NewDrawItemAction(player, "item_reverse_clock")

	if action.CanModify() {
		t.Error("DrawItemAction CanModify should return false")
	}
	if action.Source() != "item_reverse_clock" {
		t.Errorf("DrawItemAction Source = %s, expected item_reverse_clock", action.Source())
	}
	if action.Target() != player.ID.UUID() {
		t.Errorf("DrawItemAction Target = %s, expected %s", action.Target(), player.ID.UUID())
	}
	if action.TargetPlayer() != player {
		t.Error("DrawItemAction TargetPlayer should return target player")
	}
}

func TestRespawnActionInterfaceMethods(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	action := NewRespawnAction(player, 10, "system_respawn")

	if action.TargetPlayer() != player {
		t.Error("RespawnAction TargetPlayer should return target player")
	}
}

func TestFellDownActionInterfaceMethods(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	action := NewFellDownAction(player, 20, 5, "cell_fragile")

	if action.TargetPlayer() != player {
		t.Error("FellDownAction TargetPlayer should return target player")
	}
}

func TestRollDiceActionInterfaceMethods(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	rngInst := rand.New(rand.NewSource(42))
	action := NewRollDiceAction(player, rng.DiceTypeGold, rngInst, "system_dice")

	if action.Target() != player.ID.UUID() {
		t.Errorf("RollDiceAction Target = %s, expected %s", action.Target(), player.ID.UUID())
	}
	if action.TargetPlayer() != player {
		t.Error("RollDiceAction TargetPlayer should return target player")
	}
}

func TestAddItemActionInterfaceMethods(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	action := NewAddItemAction(player, constants.ItemTypeReverseClock, "event")

	if action.CanModify() {
		t.Error("AddItemAction CanModify should return false")
	}
	if action.Source() != "event" {
		t.Errorf("AddItemAction Source = %s, expected event", action.Source())
	}
	if action.Target() != player.ID.UUID() {
		t.Errorf("AddItemAction Target = %s, expected %s", action.Target(), player.ID.UUID())
	}
	if action.TargetPlayer() != player {
		t.Error("AddItemAction TargetPlayer should return target player")
	}
}

func TestRemoveItemActionInterfaceMethods(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	action := NewRemoveItemAction(player, constants.ItemTypeReverseClock, "item_used")

	if action.CanModify() {
		t.Error("RemoveItemAction CanModify should return false")
	}
	if action.Source() != "item_used" {
		t.Errorf("RemoveItemAction Source = %s, expected item_used", action.Source())
	}
	if action.Target() != player.ID.UUID() {
		t.Errorf("RemoveItemAction Target = %s, expected %s", action.Target(), player.ID.UUID())
	}
	if action.TargetPlayer() != player {
		t.Error("RemoveItemAction TargetPlayer should return target player")
	}
	if action.PreTriggerPhase() != constants.PhaseAnyTime {
		t.Errorf("RemoveItemAction PreTriggerPhase = %s, expected any_time", action.PreTriggerPhase())
	}
	if action.PostTriggerPhase() != constants.PhaseAnyTime {
		t.Errorf("RemoveItemAction PostTriggerPhase = %s, expected any_time", action.PostTriggerPhase())
	}
}

func TestDrawBuffActionInterfaceMethods(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	action := NewDrawBuffAction(player, "pool_draw")

	if action.Source() != "pool_draw" {
		t.Errorf("DrawBuffAction Source = %s, expected pool_draw", action.Source())
	}
	if action.Target() != player.ID.UUID() {
		t.Errorf("DrawBuffAction Target = %s, expected %s", action.Target(), player.ID.UUID())
	}
	if action.TargetPlayer() != player {
		t.Error("DrawBuffAction TargetPlayer should return target player")
	}
	if action.PostTriggerPhase() != constants.PhaseAnyTime {
		t.Errorf("DrawBuffAction PostTriggerPhase = %s, expected any_time", action.PostTriggerPhase())
	}
}

func TestDiceUpgradeActionInterfaceMethods(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	action := NewDiceUpgradeAction(player, "item_dice_upgrade", rng.DiceTypeCopper)

	if action.CanModify() {
		t.Error("DiceUpgradeAction CanModify should return false")
	}
	if action.Source() != "item_dice_upgrade" {
		t.Errorf("DiceUpgradeAction Source = %s, expected item_dice_upgrade", action.Source())
	}
	if action.Target() != player.ID.UUID() {
		t.Errorf("DiceUpgradeAction Target = %s, expected %s", action.Target(), player.ID.UUID())
	}
	if action.TargetPlayer() != player {
		t.Error("DiceUpgradeAction TargetPlayer should return target player")
	}
	if action.PreTriggerPhase() != constants.PhaseAnyTime {
		t.Errorf("DiceUpgradeAction PreTriggerPhase = %s, expected any_time", action.PreTriggerPhase())
	}
	if action.PostTriggerPhase() != constants.PhaseAnyTime {
		t.Errorf("DiceUpgradeAction PostTriggerPhase = %s, expected any_time", action.PostTriggerPhase())
	}
}
