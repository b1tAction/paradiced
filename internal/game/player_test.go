package game

import (
	"testing"
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

// ========== Buff Tests ==========

func TestBuffTypeString(t *testing.T) {
	tests := []struct {
		bt       BuffType
		expected string
	}{
		{BuffTypeCurse, "Curse"},
		{BuffTypeDivine, "Divine"},
		{BuffTypeFire, "Fire"},
		{BuffType(999), "Unknown"},
	}

	for _, tt := range tests {
		result := tt.bt.String()
		if result != tt.expected {
			t.Errorf("BuffType(%d).String() = %s, expected %s", tt.bt, result, tt.expected)
		}
	}
}

func TestBuffTypeIsPositive(t *testing.T) {
	positiveBuffs := []BuffType{BuffTypeDivine, BuffTypeHidden, BuffTypeRain, BuffTypeExorcism, BuffTypeFire}
	for _, bt := range positiveBuffs {
		if !bt.IsPositive() {
			t.Errorf("BuffType(%d).IsPositive() should be true", bt)
		}
	}

	negativeBuffs := []BuffType{BuffTypeCurse, BuffTypeLost, BuffTypeCorrupt, BuffTypePoison}
	for _, bt := range negativeBuffs {
		if bt.IsPositive() {
			t.Errorf("BuffType(%d).IsPositive() should be false", bt)
		}
	}
}

func TestNewBuff(t *testing.T) {
	buff := NewBuff(BuffTypeCurse, 3)
	if buff.Type != BuffTypeCurse {
		t.Errorf("buff.Type = %d, expected Curse", buff.Type)
	}
	if buff.Duration != 3 {
		t.Errorf("buff.Duration = %d, expected 3", buff.Duration)
	}
	if buff.Charge != 0 {
		t.Errorf("buff.Charge = %d, expected 0", buff.Charge)
	}
}

func TestBuffIsActive(t *testing.T) {
	// 有持续时间的 Buff
	buff1 := NewBuff(BuffTypeCurse, 3)
	if !buff1.IsActive() {
		t.Error("buff with duration > 0 should be active")
	}

	// 无持续时间的 Buff
	buff2 := NewBuff(BuffTypeCurse, 0)
	if buff2.IsActive() {
		t.Error("buff with duration = 0 should not be active")
	}

	// 有充能的 Buff
	buff3 := NewBuff(BuffTypeFire, 0)
	buff3.Charge = 1
	if !buff3.IsActive() {
		t.Error("buff with charge > 0 should be active")
	}
}

func TestBuffTickDuration(t *testing.T) {
	buff := NewBuff(BuffTypeCurse, 3)

	// 第一次 tick
	if !buff.TickDuration() {
		t.Error("buff should still be active after first tick")
	}
	if buff.Duration != 2 {
		t.Errorf("buff.Duration = %d, expected 2", buff.Duration)
	}

	// 继续 tick 直到失效
	buff.TickDuration()
	buff.TickDuration()
	if buff.IsActive() {
		t.Error("buff should not be active after all ticks")
	}
}

// ========== Item Tests ==========

func TestItemTypeString(t *testing.T) {
	tests := []struct {
		it       ItemType
		expected string
	}{
		{ItemTypeReverseClock, "ReverseClock"},
		{ItemTypeAnyDoor, "AnyDoor"},
		{ItemType(999), "Unknown"},
	}

	for _, tt := range tests {
		result := tt.it.String()
		if result != tt.expected {
			t.Errorf("ItemType(%d).String() = %s, expected %s", tt.it, result, tt.expected)
		}
	}
}

func TestNewItem(t *testing.T) {
	item := NewItem(ItemTypeReverseClock, "item-001")
	if item.Type != ItemTypeReverseClock {
		t.Errorf("item.Type = %d, expected ReverseClock", item.Type)
	}
	if item.ID != "item-001" {
		t.Errorf("item.ID = %s, expected item-001", item.ID)
	}
	if !item.Usable {
		t.Error("item should be usable initially")
	}
}

// ========== Player Tests ==========

func TestNewPlayer(t *testing.T) {
	config := PlayerConfig{
		UserID:   "player-001",
		Faction:  FactionQingLong,
		MaxHP:    10,
		MaxLP:    5,
		StartPos: 0,
	}

	player := NewPlayer(config)
	if player.UserID != "player-001" {
		t.Errorf("player.UserID = %s, expected player-001", player.UserID)
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
		UserID:  "player-002",
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
		UserID:  "player-003",
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
	engine := NewMapEngine(50)
	engine.SetCellType(10, CellTypeCheckpoint)

	player := NewPlayer(PlayerConfig{
		UserID:  "player-001",
		Faction: FactionQingLong,
		MaxHP:   10,
	})
	player.Position = 20

	// 正常扣血
	isDead, respawnPos, err := player.ApplyDamage(3, engine)
	if err != nil {
		t.Fatalf("ApplyDamage failed: %v", err)
	}
	if isDead {
		t.Error("player should not be dead")
	}
	if player.HP != 7 {
		t.Errorf("player.HP = %d, expected 7", player.HP)
	}
	if respawnPos != 20 {
		t.Errorf("respawnPos = %d, expected 20", respawnPos)
	}
}

func TestApplyDamageDeath(t *testing.T) {
	engine := NewMapEngine(50)
	engine.SetCellType(10, CellTypeCheckpoint)

	player := NewPlayer(PlayerConfig{
		UserID:  "player-001",
		Faction: FactionQingLong,
		MaxHP:   10,
	})
	player.Position = 25

	// 扣血致死
	isDead, respawnPos, err := player.ApplyDamage(15, engine)
	if err != nil {
		t.Fatalf("ApplyDamage failed: %v", err)
	}
	if !isDead {
		t.Error("player should be dead")
	}
	if player.HP != 0 {
		t.Errorf("player.HP = %d, expected 0", player.HP)
	}
	if !player.IsDead {
		t.Error("player.IsDead should be true")
	}
	// 应该回城到最近的检查点
	if respawnPos != 10 {
		t.Errorf("respawnPos = %d, expected 10 (checkpoint)", respawnPos)
	}
}

func TestApplyDamageNegative(t *testing.T) {
	engine := NewMapEngine(10)
	player := NewPlayer(DefaultPlayerConfig)

	_, _, err := player.ApplyDamage(-1, engine)
	if err == nil {
		t.Error("ApplyDamage with negative amount should return error")
	}
}

func TestApplyDamageHiddenImmune(t *testing.T) {
	engine := NewMapEngine(10)
	player := NewPlayer(PlayerConfig{UserID: "test"})
	player.AddBuff(NewBuff(BuffTypeHidden, 3))

	isDead, _, err := player.ApplyDamage(5, engine)
	if err != nil {
		t.Fatalf("ApplyDamage failed: %v", err)
	}
	if isDead {
		t.Error("player with Hidden buff should be immune to damage")
	}
	if player.HP != 10 {
		t.Errorf("player.HP = %d, expected 10 (immune)", player.HP)
	}
}

func TestHeal(t *testing.T) {
	player := NewPlayer(PlayerConfig{UserID: "test", MaxHP: 10})
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
	player := NewPlayer(PlayerConfig{UserID: "test", MaxLP: 5})

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
	player := NewPlayer(PlayerConfig{UserID: "test"})

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
	player := NewPlayer(PlayerConfig{UserID: "test", MaxHP: 10})
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
	player := NewPlayer(PlayerConfig{UserID: "test"})
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
	player := NewPlayer(PlayerConfig{UserID: "test"})
	err := player.AddBuff(nil)
	if err == nil {
		t.Error("AddBuff nil should return error")
	}
}

func TestAddBuffHiddenImmune(t *testing.T) {
	player := NewPlayer(PlayerConfig{UserID: "test"})
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
	player := NewPlayer(PlayerConfig{UserID: "test"})
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
	player := NewPlayer(PlayerConfig{UserID: "test"})
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
	player := NewPlayer(PlayerConfig{UserID: "test"})
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
	player := NewPlayer(PlayerConfig{UserID: "test"})
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
	player := NewPlayer(PlayerConfig{UserID: "test"})
	item := NewItem(ItemTypeReverseClock, "item-001")

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
	player := NewPlayer(PlayerConfig{UserID: "test"})
	err := player.AddItem(nil)
	if err == nil {
		t.Error("AddItem nil should return error")
	}
}

func TestRemoveItem(t *testing.T) {
	player := NewPlayer(PlayerConfig{UserID: "test"})
	player.AddItem(NewItem(ItemTypeReverseClock, "item-001"))
	player.AddItem(NewItem(ItemTypeAnyDoor, "item-002"))

	removed, err := player.RemoveItem("item-001")
	if err != nil {
		t.Fatalf("RemoveItem failed: %v", err)
	}
	if removed.ID != "item-001" {
		t.Errorf("removed.ID = %s, expected item-001", removed.ID)
	}
	if player.HasItem(ItemTypeReverseClock) {
		t.Error("player should not have ReverseClock after removal")
	}

	// 移除不存在的道具
	_, err = player.RemoveItem("item-999")
	if err == nil {
		t.Error("RemoveItem non-existent should return error")
	}
}

func TestGetItem(t *testing.T) {
	player := NewPlayer(PlayerConfig{UserID: "test"})
	player.AddItem(NewItem(ItemTypeReverseClock, "item-001"))

	item := player.GetItem("item-001")
	if item == nil {
		t.Error("GetItem should return item")
	}

	item = player.GetItem("item-999")
	if item != nil {
		t.Error("GetItem non-existent should return nil")
	}
}

// ========== Faction Skill Tests ==========

func TestTriggerFactionSkillQingLong(t *testing.T) {
	player := NewPlayer(PlayerConfig{
		UserID:  "test",
		Faction: FactionQingLong,
	})
	player.ChargeCount = 1

	// 有充能时可以触发
	result := player.TriggerFactionSkill(nil)
	if !result {
		t.Error("QingLong with charge should be able to trigger skill")
	}
	if player.ChargeCount != 0 {
		t.Errorf("ChargeCount = %d, expected 0", player.ChargeCount)
	}

	// 无充能时不能触发
	result = player.TriggerFactionSkill(nil)
	if result {
		t.Error("QingLong without charge should not be able to trigger skill")
	}
}

func TestTriggerFactionSkillZhuQue(t *testing.T) {
	player := NewPlayer(PlayerConfig{
		UserID:  "test",
		Faction: FactionZhuQue,
		MaxLP:   5,
	})

	// 朱雀被动是离火 Buff，已在创建时添加
	if !player.HasBuff(BuffTypeFire) {
		t.Error("ZhuQue should have Fire buff")
	}
}

func TestTriggerFactionSkillBaiHu(t *testing.T) {
	player := NewPlayer(PlayerConfig{
		UserID:  "test",
		Faction: FactionBaiHu,
	})
	target := NewPlayer(PlayerConfig{UserID: "target"})
	target.AddBuff(NewBuff(BuffTypeCurse, 3))

	event := &GameEvent{
		Type:   EventOnOvertake,
		Source: player,
		Target: target,
	}

	// 白虎反超时偷取 Buff
	result := player.TriggerFactionSkill(event)
	if !result {
		t.Error("BaiHu should be able to trigger skill on overtake")
	}
	if !player.HasBuff(BuffTypeCurse) {
		t.Error("BaiHu should have stolen Curse buff")
	}
	if target.HasBuff(BuffTypeCurse) {
		t.Error("target should not have Curse buff after steal")
	}
}

func TestTriggerFactionSkillXuanWu(t *testing.T) {
	player := NewPlayer(PlayerConfig{
		UserID:  "test",
		Faction: FactionXuanWu,
	})
	player.ChargeCount = 1

	event := &GameEvent{
		Type: EventPreBadEvent,
	}

	// 玄武有充能时可以抵消恶性事件
	result := player.TriggerFactionSkill(event)
	if !result {
		t.Error("XuanWu with charge should be able to trigger skill")
	}
	if !event.IsCancel {
		t.Error("event should be cancelled")
	}
	if player.ChargeCount != 0 {
		t.Errorf("ChargeCount = %d, expected 0", player.ChargeCount)
	}

	// 无充能时不能抵消
	event2 := &GameEvent{Type: EventPreBadEvent}
	player.ChargeCount = 0
	result = player.TriggerFactionSkill(event2)
	if result {
		t.Error("XuanWu without charge should not trigger")
	}
	if event2.IsCancel {
		t.Error("event should not be cancelled without charge")
	}
}

// ========== Event System Tests ==========

func TestDispatchEventHidden(t *testing.T) {
	player := NewPlayer(PlayerConfig{UserID: "test"})
	player.AddBuff(NewBuff(BuffTypeHidden, 3))

	event := &GameEvent{
		Type: EventPreBadEvent,
	}

	player.DispatchEvent(event)
	if !event.IsCancel {
		t.Error("Hidden buff should cancel event")
	}
}

// ========== Helper Methods Tests ==========

func TestPlayerClone(t *testing.T) {
	original := NewPlayer(PlayerConfig{
		UserID:  "test",
		Faction: FactionQingLong,
	})
	original.AddItem(NewItem(ItemTypeReverseClock, "item-001"))
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
	player := NewPlayer(PlayerConfig{
		UserID:  "player-001",
		Faction: FactionQingLong,
		MaxHP:   10,
		MaxLP:   5,
	})
	player.Position = 20
	player.AddItem(NewItem(ItemTypeReverseClock, "item-001"))
	player.AddBuff(NewBuff(BuffTypeCurse, 3))

	str := player.String()
	expected := "Player{ID: player-001, Faction: QingLong, Pos: 20, HP: 10, LP: 5, Buffs: 1, Items: 1}"
	if str != expected {
		t.Errorf("String() = %s, expected %s", str, expected)
	}
}

func TestPlayerIsAlive(t *testing.T) {
	player := NewPlayer(PlayerConfig{UserID: "test", MaxHP: 10})

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
	player := NewPlayer(PlayerConfig{UserID: "test", MaxHP: 10})

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