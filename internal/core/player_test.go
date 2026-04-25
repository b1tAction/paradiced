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
	// during match initialization, not in NewPlayer. This keeps core layer pure.
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

func TestApplyDamageHiddenImmune(t *testing.T) {
	player := NewPlayer(PlayerConfig{ID: id.NewPlayerID()})
	player.AddBuff(NewBuff(constants.BuffTypeHidden, 3))

	err := player.ApplyDamage(5)
	if err != nil {
		t.Fatalf("ApplyDamage failed: %v", err)
	}
	if player.IsDead {
		t.Error("player with Hidden buff should be immune to damage")
	}
	// HP should remain initial value
	if player.HP != DefaultPlayerConfig.InitHP {
		t.Errorf("player.HP = %d, expected %d (immune)", player.HP, DefaultPlayerConfig.InitHP)
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
	player := NewPlayer(PlayerConfig{ID: id.NewPlayerID(), MaxHP: 10})
	player.HP = 0
	player.IsDead = true
	player.Position = 50

	err := player.Respawn(10)
	if err != nil {
		t.Fatalf("Respawn failed: %v", err)
	}
	if player.HP != 10 {
		t.Errorf("player.HP = %d, expected %d (p.MaxHP)", player.HP, 10)
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

func TestTickBuffs(t *testing.T) {
	player := NewPlayer(PlayerConfig{ID: id.NewPlayerID()})
	curseBuff := NewBuff(constants.BuffTypeCurse, 1) // Duration=1
	divineBuff := NewBuff(constants.BuffTypeDivine, 3)

	player.AddBuff(curseBuff)
	player.AddBuff(divineBuff)

	// First TickBuffs call: marks eligible, no decrement (simulates first turn-end)
	expired := player.TickBuffs()
	if len(expired) != 0 {
		t.Errorf("first tick expired count = %d, expected 0 (new buffs not yet eligible)", len(expired))
	}

	// Second TickBuffs call: now eligible, curse expires (Duration 1→0)
	expired = player.TickBuffs()
	if len(expired) != 1 {
		t.Errorf("second tick expired count = %d, expected 1", len(expired))
	}
	if player.HasBuff(constants.BuffTypeCurse) {
		t.Error("Curse buff should be expired after second tick")
	}
	if !player.HasBuff(constants.BuffTypeDivine) {
		t.Error("Divine buff should still be active")
	}
}

func TestTickBuffsNewBuffNotTicked(t *testing.T) {
	// Buffs acquired mid-turn: first TickBuffs call marks eligible without decrement
	player := NewPlayer(PlayerConfig{ID: id.NewPlayerID()})
	buff := NewBuff(constants.BuffTypeCurse, 1) // Duration=1, tickEligible=false (default)

	player.AddBuff(buff)
	expired := player.TickBuffs()

	// Buff should NOT expire because first call only marks eligible
	if len(expired) != 0 {
		t.Errorf("expired count = %d, expected 0 (new buff not ticked)", len(expired))
	}
	if !player.HasBuff(constants.BuffTypeCurse) {
		t.Error("Curse buff should still be active (not ticked this turn)")
	}
	if buff.Duration != 1 {
		t.Errorf("Duration = %d, expected 1 (not decremented)", buff.Duration)
	}
}

func TestTickBuffsOnlyTickEligible(t *testing.T) {
	// Buff that has already been marked eligible (first call done) should be ticked,
	// while newly added buff should only be marked eligible without decrement
	player := NewPlayer(PlayerConfig{ID: id.NewPlayerID()})
	existingBuff := NewBuff(constants.BuffTypeCurse, 2)
	newBuff := NewBuff(constants.BuffTypeDivine, 3)

	player.AddBuff(existingBuff)
	player.AddBuff(newBuff)

	// First call: marks both eligible, no decrement
	expired := player.TickBuffs()
	if len(expired) != 0 {
		t.Errorf("expired count = %d, expected 0", len(expired))
	}

	// Second call: decrements both
	expired = player.TickBuffs()
	if len(expired) != 0 {
		t.Errorf("expired count = %d, expected 0", len(expired))
	}
	if existingBuff.Duration != 1 {
		t.Errorf("existing buff Duration = %d, expected 1 (ticked)", existingBuff.Duration)
	}
	if newBuff.Duration != 2 {
		t.Errorf("new buff Duration = %d, expected 2 (ticked)", newBuff.Duration)
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