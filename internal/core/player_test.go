package core

import (
	"testing"

	"github.com/b1tAction/paradiced/pkg/constants"
	"github.com/b1tAction/paradiced/pkg/id"
)

// ========== ParseFaction Tests ==========

func TestParseFaction(t *testing.T) {
	tests := []struct {
		input    string
		expected constants.Faction
	}{
		{"qing_long", constants.FactionQingLong},
		{"zhu_que", constants.FactionZhuQue},
		{"bai_hu", constants.FactionBaiHu},
		{"xuan_wu", constants.FactionXuanWu},
		{"unknown", constants.FactionNone},
		{"", constants.FactionNone},
	}

	for _, tt := range tests {
		result := constants.ParseFaction(tt.input)
		if result != tt.expected {
			t.Errorf("ParseFaction(%s) = %s, expected %s", tt.input, result, tt.expected)
		}
	}
}

// ========== Faction IsValid Tests ==========

func TestFactionIsValid(t *testing.T) {
	validFactions := []constants.Faction{constants.FactionQingLong, constants.FactionZhuQue, constants.FactionBaiHu, constants.FactionXuanWu}
	for _, f := range validFactions {
		if !f.IsValid() {
			t.Errorf("Faction(%s).IsValid() should be true", f)
		}
	}

	invalidFactions := []constants.Faction{constants.Faction("invalid"), constants.Faction("random")}
	for _, f := range invalidFactions {
		if f.IsValid() {
			t.Errorf("Faction(%s).IsValid() should be false", f)
		}
	}
}

// ========== Player Tests ==========

func TestNewPlayer(t *testing.T) {
	testID := id.NewPlayerID()
	config := PlayerConfig{
		ID:       testID,
		Faction:  constants.FactionQingLong,
		MaxHP:    10,
		MaxLP:    5,
		StartPos: 0,
	}

	player := NewPlayer(config)
	if player.ID != testID {
		t.Errorf("player.ID = %s, expected %s", player.ID.UUID(), testID.UUID())
	}
	if player.Faction != constants.FactionQingLong {
		t.Errorf("player.Faction = %s, expected QingLong", player.Faction)
	}
	if player.HP != 6 {
		t.Errorf("player.HP = %d, expected 6 (InitHP default)", player.HP)
	}
	if player.LP != 4 {
		t.Errorf("player.LP = %d, expected 4 (InitLP default)", player.LP)
	}
	if player.MaxHP != 10 {
		t.Errorf("player.MaxHP = %d, expected 10", player.MaxHP)
	}
	if player.InitHP != 6 {
		t.Errorf("player.InitHP = %d, expected 6 (InitHP default)", player.InitHP)
	}
	if player.Position != 0 {
		t.Errorf("player.Position = %d, expected 0", player.Position)
	}
	if player.IsDead {
		t.Error("player should not be dead initially")
	}
}

func TestNewPlayerZhuQue(t *testing.T) {
	config := PlayerConfig{
		ID:      id.NewPlayerID(),
		Faction: constants.FactionZhuQue,
	}

	player := NewPlayer(config)
	// Note: ZhuQue Fire buff is now added by game.InitializePlayerFactionBuffs()
	// during WaitingForHostState.Exit(), not in NewPlayer. This keeps core layer pure.
	// The test here verifies player creation works for ZhuQue faction.
	if player.Faction != constants.FactionZhuQue {
		t.Error("Player faction should be ZhuQue")
	}
	// Fire buff will be added later via engine layer
}

func TestNewPlayerDefaultConfig(t *testing.T) {
	config := PlayerConfig{
		ID:      id.NewPlayerID(),
		Faction: constants.FactionBaiHu,
		MaxHP:   0, // Use default
		MaxLP:   0,
	}

	player := NewPlayer(config)
	if player.HP != DefaultPlayerConfig.InitHP {
		t.Errorf("player.HP = %d, expected default %d", player.HP, DefaultPlayerConfig.InitHP)
	}
	if player.MaxHP != DefaultPlayerConfig.MaxHP {
		t.Errorf("player.MaxHP = %d, expected default %d", player.MaxHP, DefaultPlayerConfig.MaxHP)
	}
	if player.InitHP != DefaultPlayerConfig.InitHP {
		t.Errorf("player.InitHP = %d, expected default %d", player.InitHP, DefaultPlayerConfig.InitHP)
	}
	if player.LP != DefaultPlayerConfig.InitLP {
		t.Errorf("player.LP = %d, expected default %d", player.LP, DefaultPlayerConfig.InitLP)
	}
}

// ========== HP/LP Tests ==========

func TestApplyDamage(t *testing.T) {
	player := NewPlayer(PlayerConfig{
		ID:      id.NewPlayerID(),
		Faction: constants.FactionQingLong,
		MaxHP:   10,
	})
	player.Position = 20

	// Normal damage
	err := player.ApplyDamage(3)
	if err != nil {
		t.Fatalf("ApplyDamage failed: %v", err)
	}
	if player.IsDead {
		t.Error("player should not be dead")
	}
	if player.HP != 3 {
		t.Errorf("player.HP = %d, expected 3", player.HP)
	}
}

func TestApplyDamageDeath(t *testing.T) {
	player := NewPlayer(PlayerConfig{
		ID:      id.NewPlayerID(),
		Faction: constants.FactionQingLong,
		MaxHP:   10,
	})
	player.Position = 25

	// Damage to death
	err := player.ApplyDamage(15)
	if err != nil {
		t.Fatalf("ApplyDamage failed: %v", err)
	}
	if !player.IsDead {
		t.Error("player should be dead")
	}
	if player.HP != 0 {
		t.Errorf("player.HP = %d, expected 0", player.HP)
	}
	// Note: respawn logic handled by engine package, player only marks death status
}

func TestApplyDamageNegative(t *testing.T) {
	player := NewPlayer(DefaultPlayerConfig)

	err := player.ApplyDamage(-1)
	if err == nil {
		t.Error("ApplyDamage with negative amount should return error")
	}
}

func TestHeal(t *testing.T) {
	player := NewPlayer(PlayerConfig{ID: id.NewPlayerID(), MaxHP: 10})
	player.HP = 5

	err := player.Heal(3)
	if err != nil {
		t.Fatalf("Heal failed: %v", err)
	}
	if player.HP != 8 {
		t.Errorf("player.HP = %d, expected 8", player.HP)
	}

	// Negative heal
	err = player.Heal(-1)
	if err == nil {
		t.Error("Heal with negative amount should return error")
	}
}

func TestModifyLP(t *testing.T) {
	player := NewPlayer(PlayerConfig{ID: id.NewPlayerID(), MaxLP: 5})

	// Increase LP (InitLP=4, MaxLP=5)
	player.ModifyLP(2)
	if player.LP != 5 {
		t.Errorf("player.LP = %d, expected 5 (capped at MaxLP)", player.LP)
	}

	// Already at max, stays at MaxLP
	player.ModifyLP(5)
	if player.LP != 5 {
		t.Errorf("player.LP = %d, expected 5 (max)", player.LP)
	}

	// Decrease LP
	player.ModifyLP(-3)
	if player.LP != 2 {
		t.Errorf("player.LP = %d, expected 2", player.LP)
	}

	// Lower limit
	player.ModifyLP(-10)
	if player.LP != 0 {
		t.Errorf("player.LP = %d, expected 0 (min)", player.LP)
	}
}

// ========== Movement Tests ==========

func TestMove(t *testing.T) {
	player := NewPlayer(PlayerConfig{ID: id.NewPlayerID()})

	err := player.Move(10, 50)
	if err != nil {
		t.Fatalf("Move failed: %v", err)
	}
	if player.Position != 10 {
		t.Errorf("player.Position = %d, expected 10", player.Position)
	}

	// Move beyond end
	err = player.Move(100, 50)
	if err != nil {
		t.Fatalf("Move failed: %v", err)
	}
	if player.Position != 49 {
		t.Errorf("player.Position = %d, expected 49 (end)", player.Position)
	}

	// Negative position
	err = player.Move(-1, 50)
	if err == nil {
		t.Error("Move to negative position should return error")
	}
}

func TestRespawn(t *testing.T) {
	player := NewPlayer(PlayerConfig{ID: id.NewPlayerID(), InitHP: 6, MaxHP: 10})
	player.HP = 0
	player.IsDead = true
	player.Position = 50

	err := player.Respawn(10)
	if err != nil {
		t.Fatalf("Respawn failed: %v", err)
	}
	if player.HP != 6 {
		t.Errorf("player.HP = %d, expected %d (p.InitHP)", player.HP, 6)
	}
	if player.IsDead {
		t.Error("player should not be dead after respawn")
	}
	if player.Position != 10 {
		t.Errorf("player.Position = %d, expected 10", player.Position)
	}
}

// ========== Buff Management Tests ==========

func TestAddBuff(t *testing.T) {
	player := NewPlayer(PlayerConfig{ID: id.NewPlayerID()})
	buff := NewBuff(constants.BuffTypeCurse, 3)

	err := player.AddBuff(buff)
	if err != nil {
		t.Fatalf("AddBuff failed: %v", err)
	}
	if len(player.ActiveBuffs) != 1 {
		t.Errorf("ActiveBuffs count = %d, expected 1", len(player.ActiveBuffs))
	}
	if !player.HasBuff(constants.BuffTypeCurse) {
		t.Error("player should have Curse buff")
	}
}

func TestAddBuffNil(t *testing.T) {
	player := NewPlayer(PlayerConfig{ID: id.NewPlayerID()})
	err := player.AddBuff(nil)
	if err == nil {
		t.Error("AddBuff nil should return error")
	}
}

func TestAddBuffNoHiddenImmunity(t *testing.T) {
	// AddBuff no longer has hardcoded Hidden immunity.
	// Hidden immunity is now managed by handleHiddenImmune handler (PhasePreBuffApplied).
	player := NewPlayer(PlayerConfig{ID: id.NewPlayerID()})
	player.AddBuff(NewBuff(constants.BuffTypeHidden, 3))

	// Negative buff should now be added (no hardcoded immunity in AddBuff)
	err := player.AddBuff(NewBuff(constants.BuffTypeCurse, 3))
	if err != nil {
		t.Fatalf("AddBuff failed: %v", err)
	}
	if !player.HasBuff(constants.BuffTypeCurse) {
		t.Error("Curse buff should be added (AddBuff no longer has Hidden immunity)")
	}

	// Positive buff should be added
	err = player.AddBuff(NewBuff(constants.BuffTypeDivine, 3))
	if err != nil {
		t.Fatalf("AddBuff failed: %v", err)
	}
	if !player.HasBuff(constants.BuffTypeDivine) {
		t.Error("player should receive positive buff")
	}
}

func TestRemoveBuff(t *testing.T) {
	player := NewPlayer(PlayerConfig{ID: id.NewPlayerID()})
	player.AddBuff(NewBuff(constants.BuffTypeCurse, 3))
	player.AddBuff(NewBuff(constants.BuffTypeDivine, 3))

	removed := player.RemoveBuff(constants.BuffTypeCurse)
	if !removed {
		t.Error("RemoveBuff should return true")
	}
	if player.HasBuff(constants.BuffTypeCurse) {
		t.Error("player should not have Curse buff after removal")
	}
	if !player.HasBuff(constants.BuffTypeDivine) {
		t.Error("player should still have Divine buff")
	}

	// Remove non-existent buff
	removed = player.RemoveBuff(constants.BuffTypeFire)
	if removed {
		t.Error("RemoveBuff non-existent buff should return false")
	}
}

func TestGetBuff(t *testing.T) {
	player := NewPlayer(PlayerConfig{ID: id.NewPlayerID()})
	player.AddBuff(NewBuff(constants.BuffTypeCurse, 3))

	buff := player.GetBuff(constants.BuffTypeCurse)
	if buff == nil {
		t.Error("GetBuff should return buff")
	}
	if buff.Duration != 3 {
		t.Errorf("buff.Duration = %d, expected 3", buff.Duration)
	}

	// Get non-existent buff
	buff = player.GetBuff(constants.BuffTypeFire)
	if buff != nil {
		t.Error("GetBuff non-existent buff should return nil")
	}
}

func TestTickBuffsEligibleDecrements(t *testing.T) {
	// Buffs marked tickEligible (by MarkAllBuffsTickEligible at TurnUpkeep)
	// should be decremented at TurnEnd. Curse (Duration=1) expires.
	player := NewPlayer(PlayerConfig{ID: id.NewPlayerID()})
	curseBuff := NewBuff(constants.BuffTypeCurse, 1) // Duration=1
	divineBuff := NewBuff(constants.BuffTypeDivine, 3)

	player.AddBuff(curseBuff)
	player.AddBuff(divineBuff)

	// Mark all eligible (simulates TurnUpkeep step)
	player.MarkAllBuffsTickEligible()

	// TickBuffs: curse expires (Duration 1→0), divine decrements (3→2)
	expired := player.TickBuffs()
	if len(expired) != 1 {
		t.Errorf("expired count = %d, expected 1", len(expired))
	}
	if player.HasBuff(constants.BuffTypeCurse) {
		t.Error("Curse buff should be expired after tick")
	}
	if !player.HasBuff(constants.BuffTypeDivine) {
		t.Error("Divine buff should still be active")
	}
	if divineBuff.Duration != 2 {
		t.Errorf("Divine Duration = %d, expected 2", divineBuff.Duration)
	}
}

func TestTickBuffsNotEligibleNoDecrement(t *testing.T) {
	// Buffs NOT marked tickEligible should NOT be decremented at TurnEnd.
	// This simulates a buff added after TurnUpkeep (e.g., by another player's item
	// targeting this player) — it survives this turn and gets ticked next turn.
	player := NewPlayer(PlayerConfig{ID: id.NewPlayerID()})
	buff := NewBuff(constants.BuffTypeCurse, 1) // Duration=1, tickEligible=false (default)

	player.AddBuff(buff)
	expired := player.TickBuffs()

	// Buff should NOT expire because it's not tickEligible
	if len(expired) != 0 {
		t.Errorf("expired count = %d, expected 0 (not eligible, not ticked)", len(expired))
	}
	if !player.HasBuff(constants.BuffTypeCurse) {
		t.Error("Curse buff should still be active (not ticked this turn)")
	}
	if buff.Duration != 1 {
		t.Errorf("Duration = %d, expected 1 (not decremented)", buff.Duration)
	}
}

func TestTickBuffsMixedEligibility(t *testing.T) {
	// Mix of eligible and not-eligible buffs: only eligible ones are decremented
	player := NewPlayer(PlayerConfig{ID: id.NewPlayerID()})
	existingBuff := NewBuff(constants.BuffTypeCurse, 2)
	existingBuff.MarkTickEligible() // Already marked eligible
	newBuff := NewBuff(constants.BuffTypeDivine, 3) // Not yet eligible

	player.AddBuff(existingBuff)
	player.AddBuff(newBuff)

	// TickBuffs: only existingBuff (eligible) is decremented
	expired := player.TickBuffs()
	if len(expired) != 0 {
		t.Errorf("expired count = %d, expected 0", len(expired))
	}
	if existingBuff.Duration != 1 {
		t.Errorf("existing buff Duration = %d, expected 1 (ticked)", existingBuff.Duration)
	}
	if newBuff.Duration != 3 {
		t.Errorf("new buff Duration = %d, expected 3 (not ticked)", newBuff.Duration)
	}
}

func TestClearNegativeBuffs(t *testing.T) {
	player := NewPlayer(PlayerConfig{ID: id.NewPlayerID()})
	player.AddBuff(NewBuff(constants.BuffTypeCurse, 3))
	player.AddBuff(NewBuff(constants.BuffTypePoison, 3))
	player.AddBuff(NewBuff(constants.BuffTypeDivine, 3))

	count := player.ClearNegativeBuffs()
	if count != 2 {
		t.Errorf("cleared count = %d, expected 2", count)
	}
	if player.HasBuff(constants.BuffTypeCurse) || player.HasBuff(constants.BuffTypePoison) {
		t.Error("negative buffs should be cleared")
	}
	if !player.HasBuff(constants.BuffTypeDivine) {
		t.Error("positive buff should remain")
	}
}

// ========== Item Management Tests ==========

func TestAddItem(t *testing.T) {
	player := NewPlayer(PlayerConfig{ID: id.NewPlayerID()})
	item := NewItem(constants.ItemTypeReverseClock)

	err := player.AddItem(item)
	if err != nil {
		t.Fatalf("AddItem failed: %v", err)
	}
	if len(player.Inventory) != 1 {
		t.Errorf("Inventory count = %d, expected 1", len(player.Inventory))
	}
	if !player.HasItem(constants.ItemTypeReverseClock) {
		t.Error("player should have ReverseClock item")
	}
}

func TestAddItemNil(t *testing.T) {
	player := NewPlayer(PlayerConfig{ID: id.NewPlayerID()})
	err := player.AddItem(nil)
	if err == nil {
		t.Error("AddItem nil should return error")
	}
}

func TestRemoveItem(t *testing.T) {
	player := NewPlayer(PlayerConfig{ID: id.NewPlayerID()})
	item1 := NewItem(constants.ItemTypeReverseClock)
	item2 := NewItem(constants.ItemTypeAnyDoor)
	player.AddItem(item1)
	player.AddItem(item2)

	removed, err := player.RemoveItem(item1.ID)
	if err != nil {
		t.Fatalf("RemoveItem failed: %v", err)
	}
	if removed.ID != item1.ID {
		t.Errorf("removed.ID = %s, expected %s", removed.ID, item1.ID)
	}
	if player.HasItem(constants.ItemTypeReverseClock) {
		t.Error("player should not have ReverseClock after removal")
	}

	// Remove non-existent item
	_, err = player.RemoveItem(id.NewItemID())
	if err == nil {
		t.Error("RemoveItem non-existent should return error")
	}
}

func TestGetItem(t *testing.T) {
	player := NewPlayer(PlayerConfig{ID: id.NewPlayerID()})
	item := NewItem(constants.ItemTypeReverseClock)
	player.AddItem(item)

	found := player.GetItem(item.ID)
	if found == nil {
		t.Error("GetItem should return item")
	}

	notFound := player.GetItem(id.NewItemID())
	if notFound != nil {
		t.Error("GetItem non-existent should return nil")
	}
}

// ========== Faction Skill Tests ==========

// Note: ZhuQue Fire buff is now added by game.InitializePlayerFactionBuffs()
// during match initialization, not in NewPlayer. This keeps core layer pure.
// Fire buff behavior tests are in engine package.

// Note: faction passive skill trigger logic moved to engine package,
// handled via EventBus + Decision system.

// ========== Helper Methods Tests ==========

func TestPlayerClone(t *testing.T) {
	original := NewPlayer(PlayerConfig{
		ID:      id.NewPlayerID(),
		Faction: constants.FactionQingLong,
	})
	original.AddItem(NewItem(constants.ItemTypeReverseClock))
	original.AddBuff(NewBuff(constants.BuffTypeCurse, 3))
	original.HP = 5

	cloned := original.Clone()

	// Modify clone doesn't affect original
	cloned.HP = 10
	cloned.RemoveBuff(constants.BuffTypeCurse)

	if original.HP != 5 {
		t.Error("original HP should not change")
	}
	if !original.HasBuff(constants.BuffTypeCurse) {
		t.Error("original should still have Curse buff")
	}
	if cloned.HP != 10 {
		t.Error("cloned HP should be 10")
	}
	if cloned.HasBuff(constants.BuffTypeCurse) {
		t.Error("cloned should not have Curse buff after removal")
	}
}

func TestPlayerString(t *testing.T) {
	testID := id.NewPlayerID()
	player := NewPlayer(PlayerConfig{
		ID:      testID,
		Faction: constants.FactionQingLong,
		MaxHP:   10,
		MaxLP:   5,
	})
	player.Position = 20
	player.AddItem(NewItem(constants.ItemTypeReverseClock))
	player.AddBuff(NewBuff(constants.BuffTypeCurse, 3))

	str := player.String()
	expected := "Player{ID: " + testID.UUID() + ", Faction: qing_long, Pos: 20, HP: 6, LP: 4, Buffs: 1, Items: 1}"
	if str != expected {
		t.Errorf("String() = %s, expected %s", str, expected)
	}
}

func TestPlayerIsAlive(t *testing.T) {
	player := NewPlayer(PlayerConfig{ID: id.NewPlayerID(), MaxHP: 10})

	if !player.IsAlive() {
		t.Error("player should be alive initially")
	}

	player.HP = 0
	if player.IsAlive() {
		t.Error("player with HP=0 should not be alive")
	}

	player.HP = 5
	player.IsDead = true
	if player.IsAlive() {
		t.Error("dead player should not be alive")
	}
}

func TestPlayerCanAct(t *testing.T) {
	player := NewPlayer(PlayerConfig{ID: id.NewPlayerID(), MaxHP: 10})

	if !player.CanAct() {
		t.Error("player should be able to act initially")
	}

	player.SkipTurn = true
	if player.CanAct() {
		t.Error("player with SkipTurn should not be able to act")
	}

	player.SkipTurn = false
	player.HP = 0
	if player.CanAct() {
		t.Error("player with HP=0 should not be able to act")
	}
}

// ========== Metadata Tests ==========

func TestPlayerMetadataInitialized(t *testing.T) {
	player := NewPlayer(PlayerConfig{ID: id.NewPlayerID()})

	// Metadata should be initialized
	if player.Metadata == nil {
		t.Error("player Metadata should be initialized")
	}
	if player.Size() != 0 {
		t.Errorf("initial metadata size = %d, expected 0", player.Size())
	}
}

func TestPlayerChargeCount(t *testing.T) {
	player := NewPlayer(PlayerConfig{ID: id.NewPlayerID(), Faction: constants.FactionQingLong})

	// Initial charge count is 0
	if player.GetChargeCount() != 0 {
		t.Errorf("initial charge count = %d, expected 0", player.GetChargeCount())
	}

	// Set charge count
	player.SetChargeCount(5)
	if player.GetChargeCount() != 5 {
		t.Errorf("charge count = %d, expected 5", player.GetChargeCount())
	}

	// Increment charge count
	result := player.IncrementChargeCount()
	if result != 6 {
		t.Errorf("increment result = %d, expected 6", result)
	}
	if player.GetChargeCount() != 6 {
		t.Errorf("charge count after increment = %d, expected 6", player.GetChargeCount())
	}
}

func TestPlayerFireCounter(t *testing.T) {
	player := NewPlayer(PlayerConfig{ID: id.NewPlayerID(), Faction: constants.FactionZhuQue})

	// Initial fire counter is 0
	if player.GetFireCounter() != 0 {
		t.Errorf("initial fire counter = %d, expected 0", player.GetFireCounter())
	}

	// Set fire counter
	player.SetFireCounter(3)
	if player.GetFireCounter() != 3 {
		t.Errorf("fire counter = %d, expected 3", player.GetFireCounter())
	}

	// Increment fire counter
	result := player.IncrementFireCounter()
	if result != 4 {
		t.Errorf("increment result = %d, expected 4", result)
	}
	if player.GetFireCounter() != 4 {
		t.Errorf("fire counter after increment = %d, expected 4", player.GetFireCounter())
	}
}

func TestPlayerCloneWithMetadata(t *testing.T) {
	original := NewPlayer(PlayerConfig{ID: id.NewPlayerID()})
	original.SetChargeCount(5)
	original.SetFireCounter(3)
	original.SetInt("custom_value", 100)

	cloned := original.Clone()

	// Modify cloned Metadata doesn't affect original
	cloned.SetChargeCount(10)
	cloned.SetFireCounter(0)
	cloned.SetInt("custom_value", 200)

	if original.GetChargeCount() != 5 {
		t.Errorf("original charge count = %d, expected 5", original.GetChargeCount())
	}
	if original.GetFireCounter() != 3 {
		t.Errorf("original fire counter = %d, expected 3", original.GetFireCounter())
	}
	if original.GetIntOrDefault("custom_value", 0) != 100 {
		t.Errorf("original custom_value = %d, expected 100", original.GetIntOrDefault("custom_value", 0))
	}

	// Cloned values should be modified
	if cloned.GetChargeCount() != 10 {
		t.Errorf("cloned charge count = %d, expected 10", cloned.GetChargeCount())
	}
	if cloned.GetFireCounter() != 0 {
		t.Errorf("cloned fire counter = %d, expected 0", cloned.GetFireCounter())
	}
	if cloned.GetIntOrDefault("custom_value", 0) != 200 {
		t.Errorf("cloned custom_value = %d, expected 200", cloned.GetIntOrDefault("custom_value", 0))
	}
}

func TestPlayerMetadataDirectUsage(t *testing.T) {
	player := NewPlayer(PlayerConfig{ID: id.NewPlayerID()})

	// Direct use of Metadata methods
	player.SetInt("turn_count", 10)
	player.SetString("last_event", "fog")
	player.SetBool("has_visited_checkpoint", true)

	if player.GetIntOrDefault("turn_count", 0) != 10 {
		t.Errorf("GetIntOrDefault(\"turn_count\") = %d, expected 10", player.GetIntOrDefault("turn_count", 0))
	}
	if player.GetStringOrDefault("last_event", "") != "fog" {
		t.Errorf("GetStringOrDefault(\"last_event\") = %s, expected fog", player.GetStringOrDefault("last_event", ""))
	}
	if !player.GetBoolOrDefault("has_visited_checkpoint", false) {
		t.Error("GetBoolOrDefault(\"has_visited_checkpoint\") should be true")
	}

	// Chained calls
	player.SetInt("chain1", 1).SetString("chain2", "test").SetBool("chain3", true)
	if !player.HasKey("chain1") || !player.HasKey("chain2") || !player.HasKey("chain3") {
		t.Error("chained keys should all exist")
	}
}

// ========== Game Stats Tests ==========

func TestPlayerEventsDrawn(t *testing.T) {
	player := NewPlayer(PlayerConfig{ID: id.NewPlayerID()})

	// Initial events drawn is 0
	if player.GetEventsDrawn() != 0 {
		t.Errorf("initial events_drawn = %d, expected 0", player.GetEventsDrawn())
	}

	// Increment events drawn
	result := player.IncrementEventsDrawn()
	if result != 1 {
		t.Errorf("increment result = %d, expected 1", result)
	}
	if player.GetEventsDrawn() != 1 {
		t.Errorf("events_drawn after increment = %d, expected 1", player.GetEventsDrawn())
	}

	// Increment multiple times
	player.IncrementEventsDrawn()
	player.IncrementEventsDrawn()
	if player.GetEventsDrawn() != 3 {
		t.Errorf("events_drawn after 3 increments = %d, expected 3", player.GetEventsDrawn())
	}
}

func TestPlayerItemsUsed(t *testing.T) {
	player := NewPlayer(PlayerConfig{ID: id.NewPlayerID()})

	// Initial items used is 0
	if player.GetItemsUsed() != 0 {
		t.Errorf("initial items_used = %d, expected 0", player.GetItemsUsed())
	}

	// Increment items used
	result := player.IncrementItemsUsed()
	if result != 1 {
		t.Errorf("increment result = %d, expected 1", result)
	}
	if player.GetItemsUsed() != 1 {
		t.Errorf("items_used after increment = %d, expected 1", player.GetItemsUsed())
	}

	// Increment multiple times
	player.IncrementItemsUsed()
	player.IncrementItemsUsed()
	if player.GetItemsUsed() != 3 {
		t.Errorf("items_used after 3 increments = %d, expected 3", player.GetItemsUsed())
	}
}

func TestPlayerRoundsWon(t *testing.T) {
	player := NewPlayer(PlayerConfig{ID: id.NewPlayerID()})

	// Initial rounds won is 0
	if player.GetRoundsWon() != 0 {
		t.Errorf("initial rounds_won = %d, expected 0", player.GetRoundsWon())
	}

	// Increment rounds won
	result := player.IncrementRoundsWon()
	if result != 1 {
		t.Errorf("increment result = %d, expected 1", result)
	}
	if player.GetRoundsWon() != 1 {
		t.Errorf("rounds_won after increment = %d, expected 1", player.GetRoundsWon())
	}

	// Increment multiple times
	player.IncrementRoundsWon()
	player.IncrementRoundsWon()
	if player.GetRoundsWon() != 3 {
		t.Errorf("rounds_won after 3 increments = %d, expected 3", player.GetRoundsWon())
	}
}

func TestPlayerCloneWithGameStats(t *testing.T) {
	original := NewPlayer(PlayerConfig{ID: id.NewPlayerID()})
	original.IncrementEventsDrawn()
	original.IncrementEventsDrawn()
	original.IncrementItemsUsed()
	original.IncrementRoundsWon()

	cloned := original.Clone()

	// Cloned should have same stats values
	if cloned.GetEventsDrawn() != 2 {
		t.Errorf("cloned events_drawn = %d, expected 2", cloned.GetEventsDrawn())
	}
	if cloned.GetItemsUsed() != 1 {
		t.Errorf("cloned items_used = %d, expected 1", cloned.GetItemsUsed())
	}
	if cloned.GetRoundsWon() != 1 {
		t.Errorf("cloned rounds_won = %d, expected 1", cloned.GetRoundsWon())
	}

	// Modify cloned stats should not affect original
	cloned.IncrementEventsDrawn()
	if original.GetEventsDrawn() != 2 {
		t.Errorf("original events_drawn should remain 2 after cloned increment, got %d", original.GetEventsDrawn())
	}
}

// ========== protocol.Player Getter Tests ==========

func TestPlayerGetID(t *testing.T) {
	testID := id.NewPlayerID()
	player := NewPlayer(PlayerConfig{ID: testID})

	if player.GetID() != testID {
		t.Errorf("GetID() = %v, expected %v", player.GetID(), testID)
	}
}

func TestPlayerGetIDString(t *testing.T) {
	testID := id.NewPlayerID()
	player := NewPlayer(PlayerConfig{ID: testID})

	if player.GetIDString() != testID.UUID() {
		t.Errorf("GetIDString() = %s, expected %s", player.GetIDString(), testID.UUID())
	}
}

func TestPlayerGetHP(t *testing.T) {
	player := NewPlayer(PlayerConfig{ID: id.NewPlayerID(), MaxHP: 10})
	player.HP = 5

	if player.GetHP() != 5 {
		t.Errorf("GetHP() = %d, expected 5", player.GetHP())
	}
}

func TestPlayerGetLP(t *testing.T) {
	player := NewPlayer(PlayerConfig{ID: id.NewPlayerID(), MaxLP: 8})
	player.LP = 3

	if player.GetLP() != 3 {
		t.Errorf("GetLP() = %d, expected 3", player.GetLP())
	}
}

func TestPlayerGetPosition(t *testing.T) {
	player := NewPlayer(PlayerConfig{ID: id.NewPlayerID()})
	player.Position = 20

	if player.GetPosition() != 20 {
		t.Errorf("GetPosition() = %d, expected 20", player.GetPosition())
	}
}

func TestPlayerGetFaction(t *testing.T) {
	player := NewPlayer(PlayerConfig{ID: id.NewPlayerID(), Faction: constants.FactionBaiHu})

	if player.GetFaction() != constants.FactionBaiHu {
		t.Errorf("GetFaction() = %s, expected bai_hu", player.GetFaction())
	}
}

// ========== AddBuff Error Path Tests ==========

func TestPlayerAddBuffNil(t *testing.T) {
	player := NewPlayer(PlayerConfig{ID: id.NewPlayerID()})
	err := player.AddBuff(nil)
	if err == nil {
		t.Error("AddBuff(nil) should return error")
	}
}

func TestPlayerAddBuffDurationExtend(t *testing.T) {
	player := NewPlayer(PlayerConfig{ID: id.NewPlayerID()})

	// Add first buff with duration 3
	buff1 := NewBuff(constants.BuffTypeDivine, 3)
	err := player.AddBuff(buff1)
	if err != nil {
		t.Errorf("AddBuff first instance failed: %v", err)
	}

	// Add same buff type again - should extend duration
	buff2 := NewBuff(constants.BuffTypeDivine, 2)
	err = player.AddBuff(buff2)
	if err != nil {
		t.Errorf("AddBuff duration extend failed: %v", err)
	}

	// Should still have only one buff instance
	if len(player.ActiveBuffs) != 1 {
		t.Errorf("After duration extend, buff count = %d, want 1", len(player.ActiveBuffs))
	}

	// Duration should be extended: 3 + 2 = 5
	if player.ActiveBuffs[0].Duration != 5 {
		t.Errorf("Extended duration = %d, want 5", player.ActiveBuffs[0].Duration)
	}

	// tickEligible should NOT be reset on duration extend.
	// The default NewBuff tickEligible=false should remain false after extend.
	if player.ActiveBuffs[0].TickEligible() {
		t.Error("tickEligible should remain false (default NewBuff state) after duration extension")
	}
}

func TestPlayerAddBuffDurationExtendPreservesTickEligible(t *testing.T) {
	// When a buff was already marked tickEligible=true (by MarkAllBuffsTickEligible at TurnUpkeep),
	// duration extension should preserve that state so the buff is still decremented at TurnEnd.
	player := NewPlayer(PlayerConfig{ID: id.NewPlayerID()})

	buff := NewBuff(constants.BuffTypeDivine, 3)
	player.AddBuff(buff)

	// Simulate TurnUpkeep: mark all buffs tick-eligible
	player.MarkAllBuffsTickEligible()

	if !buff.TickEligible() {
		t.Fatal("buff should be tickEligible after MarkAllBuffsTickEligible")
	}

	// Extend duration mid-turn (simulates drawing the same buff again)
	extendBuff := NewBuff(constants.BuffTypeDivine, 2)
	player.AddBuff(extendBuff)

	// tickEligible should still be true after extension
	if !player.ActiveBuffs[0].TickEligible() {
		t.Error("tickEligible should remain true after duration extension, not reset to false")
	}

	// Verify: TickBuffs should decrement the extended buff
	expired := player.TickBuffs()
	if len(expired) != 0 {
		t.Errorf("expected 0 expired buffs (duration 5→4), got %d", len(expired))
	}
	if player.ActiveBuffs[0].Duration != 4 {
		t.Errorf("duration should be 4 (5-1), got %d", player.ActiveBuffs[0].Duration)
	}
}

func TestPlayerAddBuffDurationExtendNotEligible(t *testing.T) {
	// When a buff is newly created mid-turn (tickEligible=false by default),
	// duration extension should preserve tickEligible=false so it won't be
	// decremented at this turn's TurnEnd.
	player := NewPlayer(PlayerConfig{ID: id.NewPlayerID()})

	buff := NewBuff(constants.BuffTypeCurse, 2)
	player.AddBuff(buff)

	// buff is NOT tickEligible (default for NewBuff)
	if buff.TickEligible() {
		t.Fatal("new buff should have tickEligible=false by default")
	}

	// Extend duration mid-turn
	extendBuff := NewBuff(constants.BuffTypeCurse, 2)
	player.AddBuff(extendBuff)

	// tickEligible should still be false after extension
	if player.ActiveBuffs[0].TickEligible() {
		t.Error("tickEligible should remain false after duration extension of a non-eligible buff")
	}

	// Verify: TickBuffs should NOT decrement (not eligible)
	expired := player.TickBuffs()
	if len(expired) != 0 {
		t.Errorf("expected 0 expired buffs (not eligible), got %d", len(expired))
	}
	if player.ActiveBuffs[0].Duration != 4 {
		t.Errorf("duration should remain 4 (not decremented), got %d", player.ActiveBuffs[0].Duration)
	}
}