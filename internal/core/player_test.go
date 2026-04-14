package core

import (
	"testing"

	"github.com/b1tAction/paradiced/pkg/id"
)

// ========== Faction Tests ==========

func TestFactionString(t *testing.T) {
	tests := []struct {
		f        Faction
		expected string
	}{
		{FactionQingLong, "QingLong"},
		{FactionZhuQue, "ZhuQue"},
		{FactionBaiHu, "BaiHu"},
		{FactionXuanWu, "XuanWu"},
		{Faction(999), "Unknown"},
	}

	for _, tt := range tests {
		result := tt.f.String()
		if result != tt.expected {
			t.Errorf("Faction(%d).String() = %s, expected %s", tt.f, result, tt.expected)
		}
	}
}

func TestFactionIsValid(t *testing.T) {
	validFactions := []Faction{FactionQingLong, FactionZhuQue, FactionBaiHu, FactionXuanWu}
	for _, f := range validFactions {
		if !f.IsValid() {
			t.Errorf("Faction(%d).IsValid() should be true", f)
		}
	}

	invalidFactions := []Faction{Faction(-1), Faction(100)}
	for _, f := range invalidFactions {
		if f.IsValid() {
			t.Errorf("Faction(%d).IsValid() should be false", f)
		}
	}
}

// ========== Player Tests ==========

func TestNewPlayer(t *testing.T) {
	testID := id.NewPlayerID()
	config := PlayerConfig{
		ID:       testID,
		Faction:  FactionQingLong,
		MaxHP:    10,
		MaxLP:    5,
		StartPos: 0,
	}

	player := NewPlayer(config)
	if player.ID != testID {
		t.Errorf("player.ID = %s, expected %s", player.ID.UUID(), testID.UUID())
	}
	if player.Faction != FactionQingLong {
		t.Errorf("player.Faction = %d, expected QingLong", player.Faction)
	}
	if player.HP != 10 {
		t.Errorf("player.HP = %d, expected 10", player.HP)
	}
	if player.LP != 5 {
		t.Errorf("player.LP = %d, expected 5", player.LP)
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
		Faction: FactionZhuQue,
	}

	player := NewPlayer(config)
	// 朱雀玩家应该携带离火 Buff
	if !player.HasBuff(BuffTypeFire) {
		t.Error("ZhuQue player should have Fire buff initially")
	}
}

func TestNewPlayerDefaultConfig(t *testing.T) {
	config := PlayerConfig{
		ID:      id.NewPlayerID(),
		Faction: FactionBaiHu,
		MaxHP:   0, // 使用默认值
		MaxLP:   0,
	}

	player := NewPlayer(config)
	if player.HP != DefaultPlayerConfig.MaxHP {
		t.Errorf("player.HP = %d, expected default %d", player.HP, DefaultPlayerConfig.MaxHP)
	}
	if player.LP != DefaultPlayerConfig.MaxLP {
		t.Errorf("player.LP = %d, expected default %d", player.LP, DefaultPlayerConfig.MaxLP)
	}
}

// ========== HP/LP Tests ==========

func TestApplyDamage(t *testing.T) {
	player := NewPlayer(PlayerConfig{
		ID:      id.NewPlayerID(),
		Faction: FactionQingLong,
		MaxHP:   10,
	})
	player.Position = 20

	// 正常扣血
	err := player.ApplyDamage(3)
	if err != nil {
		t.Fatalf("ApplyDamage failed: %v", err)
	}
	if player.IsDead {
		t.Error("player should not be dead")
	}
	if player.HP != 7 {
		t.Errorf("player.HP = %d, expected 7", player.HP)
	}
}

func TestApplyDamageDeath(t *testing.T) {
	player := NewPlayer(PlayerConfig{
		ID:      id.NewPlayerID(),
		Faction: FactionQingLong,
		MaxHP:   10,
	})
	player.Position = 25

	// 扣血致死
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
	// 注意：回城逻辑由 engine 包处理，player 只标记死亡状态
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
	player.AddBuff(NewBuff(BuffTypeHidden, 3))

	err := player.ApplyDamage(5)
	if err != nil {
		t.Fatalf("ApplyDamage failed: %v", err)
	}
	if player.IsDead {
		t.Error("player with Hidden buff should be immune to damage")
	}
	// HP 应保持初始值
	if player.HP != DefaultPlayerConfig.MaxHP {
		t.Errorf("player.HP = %d, expected %d (immune)", player.HP, DefaultPlayerConfig.MaxHP)
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

	// 负数回血
	err = player.Heal(-1)
	if err == nil {
		t.Error("Heal with negative amount should return error")
	}
}

func TestModifyLP(t *testing.T) {
	player := NewPlayer(PlayerConfig{ID: id.NewPlayerID(), MaxLP: 5})

	// 增加 LP
	player.ModifyLP(2)
	if player.LP != 7 {
		t.Errorf("player.LP = %d, expected 7", player.LP)
	}

	// 上限限制
	player.ModifyLP(5)
	if player.LP != 8 {
		t.Errorf("player.LP = %d, expected 8 (max)", player.LP)
	}

	// 减少 LP
	player.ModifyLP(-3)
	if player.LP != 5 {
		t.Errorf("player.LP = %d, expected 5", player.LP)
	}

	// 下限限制
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

	// 移动到终点之外
	err = player.Move(100, 50)
	if err != nil {
		t.Fatalf("Move failed: %v", err)
	}
	if player.Position != 49 {
		t.Errorf("player.Position = %d, expected 49 (end)", player.Position)
	}

	// 负数位置
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
	if player.HP != DefaultPlayerConfig.MaxHP {
		t.Errorf("player.HP = %d, expected %d", player.HP, DefaultPlayerConfig.MaxHP)
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
	buff := NewBuff(BuffTypeCurse, 3)

	err := player.AddBuff(buff)
	if err != nil {
		t.Fatalf("AddBuff failed: %v", err)
	}
	if len(player.ActiveBuffs) != 1 {
		t.Errorf("ActiveBuffs count = %d, expected 1", len(player.ActiveBuffs))
	}
	if !player.HasBuff(BuffTypeCurse) {
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

func TestAddBuffHiddenImmune(t *testing.T) {
	player := NewPlayer(PlayerConfig{ID: id.NewPlayerID()})
	player.AddBuff(NewBuff(BuffTypeHidden, 3))

	// 隐匿状态下免疫负面 Buff
	err := player.AddBuff(NewBuff(BuffTypeCurse, 3))
	if err != nil {
		t.Fatalf("AddBuff failed: %v", err)
	}
	// Curse buff 不应该被添加
	if player.HasBuff(BuffTypeCurse) {
		t.Error("player with Hidden buff should not receive negative buff")
	}

	// 正面 Buff 应该可以添加
	err = player.AddBuff(NewBuff(BuffTypeDivine, 3))
	if err != nil {
		t.Fatalf("AddBuff failed: %v", err)
	}
	if !player.HasBuff(BuffTypeDivine) {
		t.Error("player with Hidden buff should receive positive buff")
	}
}

func TestRemoveBuff(t *testing.T) {
	player := NewPlayer(PlayerConfig{ID: id.NewPlayerID()})
	player.AddBuff(NewBuff(BuffTypeCurse, 3))
	player.AddBuff(NewBuff(BuffTypeDivine, 3))

	removed := player.RemoveBuff(BuffTypeCurse)
	if !removed {
		t.Error("RemoveBuff should return true")
	}
	if player.HasBuff(BuffTypeCurse) {
		t.Error("player should not have Curse buff after removal")
	}
	if !player.HasBuff(BuffTypeDivine) {
		t.Error("player should still have Divine buff")
	}

	// 移除不存在的 Buff
	removed = player.RemoveBuff(BuffTypeFire)
	if removed {
		t.Error("RemoveBuff non-existent buff should return false")
	}
}

func TestGetBuff(t *testing.T) {
	player := NewPlayer(PlayerConfig{ID: id.NewPlayerID()})
	player.AddBuff(NewBuff(BuffTypeCurse, 3))

	buff := player.GetBuff(BuffTypeCurse)
	if buff == nil {
		t.Error("GetBuff should return buff")
	}
	if buff.Duration != 3 {
		t.Errorf("buff.Duration = %d, expected 3", buff.Duration)
	}

	// 获取不存在的 Buff
	buff = player.GetBuff(BuffTypeFire)
	if buff != nil {
		t.Error("GetBuff non-existent buff should return nil")
	}
}

func TestTickBuffs(t *testing.T) {
	player := NewPlayer(PlayerConfig{ID: id.NewPlayerID()})
	player.AddBuff(NewBuff(BuffTypeCurse, 1)) // 只剩1回合
	player.AddBuff(NewBuff(BuffTypeDivine, 3))

	expired := player.TickBuffs()
	if len(expired) != 1 {
		t.Errorf("expired count = %d, expected 1", len(expired))
	}
	if player.HasBuff(BuffTypeCurse) {
		t.Error("Curse buff should be expired")
	}
	if !player.HasBuff(BuffTypeDivine) {
		t.Error("Divine buff should still be active")
	}
}

func TestClearNegativeBuffs(t *testing.T) {
	player := NewPlayer(PlayerConfig{ID: id.NewPlayerID()})
	player.AddBuff(NewBuff(BuffTypeCurse, 3))
	player.AddBuff(NewBuff(BuffTypePoison, 3))
	player.AddBuff(NewBuff(BuffTypeDivine, 3))

	count := player.ClearNegativeBuffs()
	if count != 2 {
		t.Errorf("cleared count = %d, expected 2", count)
	}
	if player.HasBuff(BuffTypeCurse) || player.HasBuff(BuffTypePoison) {
		t.Error("negative buffs should be cleared")
	}
	if !player.HasBuff(BuffTypeDivine) {
		t.Error("positive buff should remain")
	}
}

// ========== Item Management Tests ==========

func TestAddItem(t *testing.T) {
	player := NewPlayer(PlayerConfig{ID: id.NewPlayerID()})
	item := NewItem(ItemTypeReverseClock)

	err := player.AddItem(item)
	if err != nil {
		t.Fatalf("AddItem failed: %v", err)
	}
	if len(player.Inventory) != 1 {
		t.Errorf("Inventory count = %d, expected 1", len(player.Inventory))
	}
	if !player.HasItem(ItemTypeReverseClock) {
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
	item1 := NewItem(ItemTypeReverseClock)
	item2 := NewItem(ItemTypeAnyDoor)
	player.AddItem(item1)
	player.AddItem(item2)

	removed, err := player.RemoveItem(item1.ID)
	if err != nil {
		t.Fatalf("RemoveItem failed: %v", err)
	}
	if removed.ID != item1.ID {
		t.Errorf("removed.ID = %s, expected %s", removed.ID, item1.ID)
	}
	if player.HasItem(ItemTypeReverseClock) {
		t.Error("player should not have ReverseClock after removal")
	}

	// 移除不存在的道具
	_, err = player.RemoveItem(id.NewItemID())
	if err == nil {
		t.Error("RemoveItem non-existent should return error")
	}
}

func TestGetItem(t *testing.T) {
	player := NewPlayer(PlayerConfig{ID: id.NewPlayerID()})
	item := NewItem(ItemTypeReverseClock)
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

func TestTriggerFactionSkillZhuQue(t *testing.T) {
	player := NewPlayer(PlayerConfig{
		ID:      id.NewPlayerID(),
		Faction: FactionZhuQue,
		MaxLP:   5,
	})

	// 朱雀被动是离火 Buff，已在创建时添加
	if !player.HasBuff(BuffTypeFire) {
		t.Error("ZhuQue should have Fire buff")
	}
}

// 注意：阵营被动技能的触发逻辑已迁移到 engine 包，
// 通过 EventBus + Decision 系统处理。

// ========== Helper Methods Tests ==========

func TestPlayerClone(t *testing.T) {
	original := NewPlayer(PlayerConfig{
		ID:      id.NewPlayerID(),
		Faction: FactionQingLong,
	})
	original.AddItem(NewItem(ItemTypeReverseClock))
	original.AddBuff(NewBuff(BuffTypeCurse, 3))
	original.HP = 5

	cloned := original.Clone()

	// 修改克隆不影响原版
	cloned.HP = 10
	cloned.RemoveBuff(BuffTypeCurse)

	if original.HP != 5 {
		t.Error("original HP should not change")
	}
	if !original.HasBuff(BuffTypeCurse) {
		t.Error("original should still have Curse buff")
	}
	if cloned.HP != 10 {
		t.Error("cloned HP should be 10")
	}
	if cloned.HasBuff(BuffTypeCurse) {
		t.Error("cloned should not have Curse buff after removal")
	}
}

func TestPlayerString(t *testing.T) {
	testID := id.NewPlayerID()
	player := NewPlayer(PlayerConfig{
		ID:      testID,
		Faction: FactionQingLong,
		MaxHP:   10,
		MaxLP:   5,
	})
	player.Position = 20
	player.AddItem(NewItem(ItemTypeReverseClock))
	player.AddBuff(NewBuff(BuffTypeCurse, 3))

	str := player.String()
	expected := "Player{ID: " + testID.UUID() + ", Faction: QingLong, Pos: 20, HP: 10, LP: 5, Buffs: 1, Items: 1}"
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

	// Metadata 应该被初始化
	if player.Metadata == nil {
		t.Error("player Metadata should be initialized")
	}
	if player.Size() != 0 {
		t.Errorf("initial metadata size = %d, expected 0", player.Size())
	}
}

func TestPlayerChargeCount(t *testing.T) {
	player := NewPlayer(PlayerConfig{ID: id.NewPlayerID(), Faction: FactionQingLong})

	// 初始充能计数为 0
	if player.GetChargeCount() != 0 {
		t.Errorf("initial charge count = %d, expected 0", player.GetChargeCount())
	}

	// 设置充能计数
	player.SetChargeCount(5)
	if player.GetChargeCount() != 5 {
		t.Errorf("charge count = %d, expected 5", player.GetChargeCount())
	}

	// 递增充能计数
	result := player.IncrementChargeCount()
	if result != 6 {
		t.Errorf("increment result = %d, expected 6", result)
	}
	if player.GetChargeCount() != 6 {
		t.Errorf("charge count after increment = %d, expected 6", player.GetChargeCount())
	}
}

func TestPlayerFireCounter(t *testing.T) {
	player := NewPlayer(PlayerConfig{ID: id.NewPlayerID(), Faction: FactionZhuQue})

	// 初始离火计数为 0
	if player.GetFireCounter() != 0 {
		t.Errorf("initial fire counter = %d, expected 0", player.GetFireCounter())
	}

	// 设置离火计数
	player.SetFireCounter(3)
	if player.GetFireCounter() != 3 {
		t.Errorf("fire counter = %d, expected 3", player.GetFireCounter())
	}

	// 递增离火计数
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