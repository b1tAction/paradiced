package core

import (
	"errors"
	"fmt"
)

// Player 玩家结构体
type Player struct {
	UserID      string   `json:"user_id"`      // 玩家UUID
	Faction     Faction  `json:"faction"`      // 阵营
	Position    int      `json:"position"`     // 当前位置
	HP          int      `json:"hp"`           // 血量
	LP          int      `json:"lp"`           // 幸运值（影响随机事件）
	Inventory   []*Item  `json:"inventory"`    // 道具栏
	ActiveBuffs []*Buff  `json:"active_buffs"` // 持续状态
	IsDead      bool     `json:"is_dead"`      // 是否死亡
	SkipTurn    bool     `json:"skip_turn"`    // 是否跳过回合
	ChargeCount int      `json:"charge_count"` // 充能计数（青龙/玄武）
	FireCounter int      `json:"fire_counter"` // 离火计数（朱雀）
}

// PlayerConfig 玩家配置
type PlayerConfig struct {
	UserID   string
	Faction  Faction
	MaxHP    int
	MaxLP    int
	StartPos int
}

// DefaultPlayerConfig 默认玩家配置
var DefaultPlayerConfig = PlayerConfig{
	MaxHP:    6,
	MaxLP:    10,
	StartPos: 0,
}

// NewPlayer 创建新玩家
func NewPlayer(config PlayerConfig) *Player {
	if config.MaxHP <= 0 {
		config.MaxHP = DefaultPlayerConfig.MaxHP
	}
	if config.MaxLP <= 0 {
		config.MaxLP = DefaultPlayerConfig.MaxLP
	}

	player := &Player{
		UserID:      config.UserID,
		Faction:     config.Faction,
		Position:    config.StartPos,
		HP:          config.MaxHP,
		LP:          config.MaxLP,
		Inventory:   make([]*Item, 0),
		ActiveBuffs: make([]*Buff, 0),
		IsDead:      false,
		SkipTurn:    false,
		ChargeCount: 0,
		FireCounter: 0,
	}

	// 朱雀阵营初始携带离火Buff
	if config.Faction == FactionZhuQue {
		player.AddBuff(NewBuff(BuffTypeFire, -1))
	}

	return player
}

// ========== 数值逻辑 ==========

// ApplyDamage 扣血并检测是否死亡
// 注意：回城逻辑由 engine 包处理，这里只负责扣血
func (p *Player) ApplyDamage(amount int) error {
	if amount < 0 {
		return errors.New("damage amount cannot be negative")
	}

	// 隐匿状态下免疫伤害
	if p.HasBuff(BuffTypeHidden) {
		return nil
	}

	p.HP -= amount

	if p.HP <= 0 {
		p.HP = 0
		p.IsDead = true
	}

	return nil
}

// Heal 回血
func (p *Player) Heal(amount int) error {
	if amount < 0 {
		return errors.New("heal amount cannot be negative")
	}
	p.HP += amount
	return nil
}

// ModifyLP 修改幸运值
func (p *Player) ModifyLP(amount int) {
	p.LP += amount
	if p.LP < 0 {
		p.LP = 0
	}
	if p.LP > 8 {
		p.LP = 8
	}
}

// ========== 移动逻辑 ==========

// Move 移动玩家到指定位置
func (p *Player) Move(newPosition int, maxLength int) error {
	if newPosition < 0 {
		return errors.New("position cannot be negative")
	}
	if newPosition >= maxLength {
		newPosition = maxLength - 1
	}
	p.Position = newPosition
	return nil
}

// Respawn 复活回城
func (p *Player) Respawn(respawnPos int) error {
	if respawnPos < 0 {
		return errors.New("respawn position cannot be negative")
	}
	p.Position = respawnPos
	p.HP = DefaultPlayerConfig.MaxHP
	p.IsDead = false
	p.SkipTurn = false
	return nil
}

// ========== Buff 管理 ==========

// AddBuff 添加 Buff
func (p *Player) AddBuff(buff *Buff) error {
	if buff == nil {
		return errors.New("buff cannot be nil")
	}
	if p.HasBuff(BuffTypeHidden) && !buff.Type.IsPositive() {
		return nil
	}
	p.ActiveBuffs = append(p.ActiveBuffs, buff)
	return nil
}

// RemoveBuff 移除指定类型的 Buff
func (p *Player) RemoveBuff(buffType BuffType) bool {
	for i, buff := range p.ActiveBuffs {
		if buff.Type == buffType {
			p.ActiveBuffs = append(p.ActiveBuffs[:i], p.ActiveBuffs[i+1:]...)
			return true
		}
	}
	return false
}

// HasBuff 检查是否有指定类型的 Buff
func (p *Player) HasBuff(buffType BuffType) bool {
	for _, buff := range p.ActiveBuffs {
		if buff.Type == buffType && buff.IsActive() {
			return true
		}
	}
	return false
}

// GetBuff 获取指定类型的 Buff
func (p *Player) GetBuff(buffType BuffType) *Buff {
	for _, buff := range p.ActiveBuffs {
		if buff.Type == buffType {
			return buff
		}
	}
	return nil
}

// TickBuffs 更新所有 Buff 的持续时间
func (p *Player) TickBuffs() []*Buff {
	var expired []*Buff
	for i := len(p.ActiveBuffs) - 1; i >= 0; i-- {
		if !p.ActiveBuffs[i].TickDuration() {
			expired = append(expired, p.ActiveBuffs[i])
			p.ActiveBuffs = append(p.ActiveBuffs[:i], p.ActiveBuffs[i+1:]...)
		}
	}
	return expired
}

// ClearNegativeBuffs 清除所有负面 Buff
func (p *Player) ClearNegativeBuffs() int {
	count := 0
	for i := len(p.ActiveBuffs) - 1; i >= 0; i-- {
		if !p.ActiveBuffs[i].Type.IsPositive() {
			p.ActiveBuffs = append(p.ActiveBuffs[:i], p.ActiveBuffs[i+1:]...)
			count++
		}
	}
	return count
}

// ========== 道具管理 ==========

// AddItem 添加道具
func (p *Player) AddItem(item *Item) error {
	if item == nil {
		return errors.New("item cannot be nil")
	}
	p.Inventory = append(p.Inventory, item)
	return nil
}

// RemoveItem 移除道具
func (p *Player) RemoveItem(itemID string) (*Item, error) {
	for i, item := range p.Inventory {
		if item.ID == itemID {
			removed := p.Inventory[i]
			p.Inventory = append(p.Inventory[:i], p.Inventory[i+1:]...)
			return removed, nil
		}
	}
	return nil, errors.New("item not found")
}

// GetItem 获取道具
func (p *Player) GetItem(itemID string) *Item {
	for _, item := range p.Inventory {
		if item.ID == itemID {
			return item
		}
	}
	return nil
}

// HasItem 检查是否有指定类型的道具
func (p *Player) HasItem(itemType ItemType) bool {
	for _, item := range p.Inventory {
		if item.Type == itemType {
			return true
		}
	}
	return false
}

// ========== 辅助方法 ==========

// Clone 克隆玩家（用于测试）
func (p *Player) Clone() *Player {
	inventory := make([]*Item, len(p.Inventory))
	for i, item := range p.Inventory {
		inventory[i] = &Item{
			Type:     item.Type,
			ID:       item.ID,
			Usable:   item.Usable,
			TargetID: item.TargetID,
		}
	}

	buffs := make([]*Buff, len(p.ActiveBuffs))
	for i, buff := range p.ActiveBuffs {
		buffs[i] = &Buff{
			Type:     buff.Type,
			Duration: buff.Duration,
			Charge:   buff.Charge,
		}
	}

	return &Player{
		UserID:      p.UserID,
		Faction:     p.Faction,
		Position:    p.Position,
		HP:          p.HP,
		LP:          p.LP,
		Inventory:   inventory,
		ActiveBuffs: buffs,
		IsDead:      p.IsDead,
		SkipTurn:    p.SkipTurn,
		ChargeCount: p.ChargeCount,
		FireCounter: p.FireCounter,
	}
}

// String 返回玩家信息字符串
func (p *Player) String() string {
	return fmt.Sprintf("Player{ID: %s, Faction: %s, Pos: %d, HP: %d, LP: %d, Buffs: %d, Items: %d}",
		p.UserID, p.Faction.String(), p.Position, p.HP, p.LP, len(p.ActiveBuffs), len(p.Inventory))
}

// IsAlive 检查玩家是否存活
func (p *Player) IsAlive() bool {
	return !p.IsDead && p.HP > 0
}

// CanAct 检查玩家是否可以行动
func (p *Player) CanAct() bool {
	return p.IsAlive() && !p.SkipTurn
}