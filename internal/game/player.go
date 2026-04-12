package game

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
	SkipTurn    bool     `json:"skip_turn"`    // 是否跳过回合（冰冻/晕眩）
	ChargeCount int      `json:"charge_count"` // 充能计数（青龙/玄武）
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
	MaxHP:    10,
	MaxLP:    5,
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
	}

	// 朱雀阵营初始携带离火Buff
	if config.Faction == FactionZhuQue {
		player.AddBuff(NewBuff(BuffTypeFire, -1)) // -1 表示永久生效
	}

	return player
}

// ========== 数值逻辑 ==========

// ApplyDamage 扣血并检测是否死亡
// 返回值：是否死亡，是否触发回城
func (p *Player) ApplyDamage(amount int, engine *MapEngine) (isDead bool, respawnPos int, err error) {
	if amount < 0 {
		return false, p.Position, errors.New("damage amount cannot be negative")
	}

	// 隐匿状态下免疫伤害
	if p.HasBuff(BuffTypeHidden) {
		return false, p.Position, nil
	}

	p.HP -= amount

	if p.HP <= 0 {
		p.HP = 0
		p.IsDead = true
		// 回城到最近的检查点
		respawnPos = engine.GetLastCheckpoint(p.Position)
		return true, respawnPos, nil
	}

	return false, p.Position, nil
}

// Heal 回血
func (p *Player) Heal(amount int) error {
	if amount < 0 {
		return errors.New("heal amount cannot be negative")
	}
	p.HP += amount
	// HP 上限为初始值（可通过配置扩展）
	return nil
}

// ModifyLP 修改幸运值
func (p *Player) ModifyLP(amount int) {
	p.LP += amount
	// LP 范围限制 [0, 8]
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
	p.HP = DefaultPlayerConfig.MaxHP // 回血到初始值
	p.IsDead = false
	p.SkipTurn = false
	// 移除部分 Buff（可选：死亡后清除所有负面buff）
	return nil
}

// ========== Buff 管理 ==========

// AddBuff 添加 Buff
func (p *Player) AddBuff(buff *Buff) error {
	if buff == nil {
		return errors.New("buff cannot be nil")
	}
	// 隐匿状态下免疫 Buff
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
// 返回已失效的 Buff 列表
func (p *Player) TickBuffs() []*Buff {
	var expired []*Buff
	for i, buff := range p.ActiveBuffs {
		if !buff.TickDuration() {
			expired = append(expired, buff)
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

// ========== 阵营被动技能 ==========

// FactionSkill 阵营被动技能接口
type FactionSkill interface {
	CanActivate(player *Player) bool
	Activate(player *Player, event *GameEvent) bool
	GetCharge() int
}

// QingLongPassive 青龙被动：行迹（每5回合获得充能，无视负面地形）
type QingLongPassive struct {
	Charge int
}

func (ql *QingLongPassive) CanActivate(player *Player) bool {
	return ql.Charge > 0
}

func (ql *QingLongPassive) Activate(player *Player, event *GameEvent) bool {
	if ql.Charge > 0 {
		ql.Charge--
		return true
	}
	return false
}

func (ql *QingLongPassive) GetCharge() int {
	return ql.Charge
}

// ZhuQuePassive 朱雀被动：离火（每4回合幸运值+1，最高8）
type ZhuQuePassive struct{}

func (zq *ZhuQuePassive) CanActivate(player *Player) bool {
	return player.LP < 8
}

func (zq *ZhuQuePassive) Activate(player *Player, event *GameEvent) bool {
	player.ModifyLP(1)
	return true
}

func (zq *ZhuQuePassive) GetCharge() int {
	return 0
}

// BaiHuPassive 白虎被动：劫运（反超他人时偷取 Buff）
type BaiHuPassive struct{}

func (bh *BaiHuPassive) CanActivate(player *Player) bool {
	return true
}

func (bh *BaiHuPassive) Activate(player *Player, event *GameEvent) bool {
	if event == nil || event.Target == nil {
		return false
	}
	// 从目标身上偷取一个 Buff
	target := event.Target
	if len(target.ActiveBuffs) > 0 {
		buff := target.ActiveBuffs[0]
		target.RemoveBuff(buff.Type)
		player.AddBuff(buff)
		return true
	}
	return false
}

func (bh *BaiHuPassive) GetCharge() int {
	return 0
}

// XuanWuPassive 玄武被动：镇厄（每5回合获得充能，可抵消恶性事件）
type XuanWuPassive struct {
	Charge int
}

func (xw *XuanWuPassive) CanActivate(player *Player) bool {
	return xw.Charge > 0
}

func (xw *XuanWuPassive) Activate(player *Player, event *GameEvent) bool {
	if xw.Charge > 0 && event != nil {
		event.IsCancel = true
		xw.Charge--
		return true
	}
	return false
}

func (xw *XuanWuPassive) GetCharge() int {
	return xw.Charge
}

// TriggerFactionSkill 触发阵营被动技能
func (p *Player) TriggerFactionSkill(event *GameEvent) bool {
	// 根据阵营选择对应的被动技能
	switch p.Faction {
	case FactionQingLong:
		// 青龙：检查是否有充能
		if p.ChargeCount > 0 {
			p.ChargeCount--
			return true
		}
	case FactionZhuQue:
		// 朱雀：自动生效（已在 NewPlayer 中添加离火 Buff）
		return true
	case FactionBaiHu:
		// 白虎：反超时触发（需要在事件中处理）
		if event != nil && event.Type == EventOnOvertake {
			passive := &BaiHuPassive{}
			return passive.Activate(p, event)
		}
	case FactionXuanWu:
		// 玄武：检查是否有充能，抵消恶性事件
		if p.ChargeCount > 0 && event != nil && event.Type == EventPreBadEvent {
			event.IsCancel = true
			p.ChargeCount--
			return true
		}
	}
	return false
}

// UpdateCharge 更新充能计数（回合结束时调用）
func (p *Player) UpdateCharge() {
	switch p.Faction {
	case FactionQingLong:
		// 青龙：每5回合获得充能
		p.ChargeCount++
		if p.ChargeCount >= 5 {
			p.ChargeCount = 0
			// TODO: 触发充能获得
		}
	case FactionXuanWu:
		// 玄武：每5回合获得充能
		p.ChargeCount++
		if p.ChargeCount >= 5 {
			p.ChargeCount = 0
			// TODO: 触发充能获得
		}
	}
}

// ========== 游戏事件系统 ==========

// EventPhase 事件阶段
type EventPhase string

const (
	EventPreBadEvent EventPhase = "PreBadEvent" // 触发恶性事件前
	EventOnOvertake  EventPhase = "OnOvertake"  // 发生反超后
	EventPreDamage   EventPhase = "PreDamage"   // 扣血前（护盾类道具）
	EventOnMove      EventPhase = "OnMove"      // 每次移动一格时触发
)

// GameEvent 游戏事件
type GameEvent struct {
	Type     EventPhase `json:"type"`     // 事件类型
	Source   *Player    `json:"source"`   // 触发事件的玩家
	Target   *Player    `json:"target"`   // 目标玩家
	Payload  interface{} `json:"payload"` // 事件数据
	IsCancel bool       `json:"is_cancel"` // 是否被取消/拦截
}

// DispatchEvent 分发事件到玩家的 Hooks
func (p *Player) DispatchEvent(event *GameEvent) {
	// 检查隐匿状态
	if p.HasBuff(BuffTypeHidden) {
		event.IsCancel = true
		return
	}

	// 触发阵营被动技能
	p.TriggerFactionSkill(event)

	// TODO: 触发道具的 Hook（如护盾类道具）
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