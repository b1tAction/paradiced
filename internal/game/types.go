package game

// Faction 玩家阵营（四神兽）
type Faction int

const (
	FactionQingLong Faction = iota // 青龙（东方）- 行迹
	FactionZhuQue                  // 朱雀（南方）- 离火
	FactionBaiHu                   // 白虎（西方）- 劫运
	FactionXuanWu                  // 玄武（北方）- 镇厄
)

// String 返回阵营名称
func (f Faction) String() string {
	names := map[Faction]string{
		FactionQingLong: "QingLong",
		FactionZhuQue:   "ZhuQue",
		FactionBaiHu:    "BaiHu",
		FactionXuanWu:   "XuanWu",
	}
	if name, ok := names[f]; ok {
		return name
	}
	return "Unknown"
}

// IsValid 检查阵营是否有效
func (f Faction) IsValid() bool {
	return f >= FactionQingLong && f <= FactionXuanWu
}

// BuffType Buff 类型
type BuffType int

const (
	BuffTypeNone BuffType = iota
	// 负性 Buff
	BuffTypeCurse     // 诅咒：接下来3回合lp-1
	BuffTypeLost      // 迷途：下1回合朝反方向移动
	BuffTypeCorrupt   // 腐化：接下来4回合每2回合hp-1
	BuffTypePoison    // 毒瘴：接下来3回合每回合受一次恶性随机事件
	// 正性 Buff
	BuffTypeDivine    // 神眷：接下来3回合lp+1
	BuffTypeHidden    // 隐匿：接下来3回合免疫任意事件、BUFF或道具
	BuffTypeRain      // 甘霖：接下来4回合每2回合hp+1
	BuffTypeExorcism  // 辟邪：接下来5回合无视毒瘴buff
	BuffTypeFire      // 离火：朱雀阵营初始增益，每4回合lp+1
)

// String 返回 Buff 类型名称
func (bt BuffType) String() string {
	names := map[BuffType]string{
		BuffTypeNone:      "None",
		BuffTypeCurse:     "Curse",
		BuffTypeLost:      "Lost",
		BuffTypeCorrupt:   "Corrupt",
		BuffTypePoison:    "Poison",
		BuffTypeDivine:    "Divine",
		BuffTypeHidden:    "Hidden",
		BuffTypeRain:      "Rain",
		BuffTypeExorcism:  "Exorcism",
		BuffTypeFire:      "Fire",
	}
	if name, ok := names[bt]; ok {
		return name
	}
	return "Unknown"
}

// IsPositive 判断是否为正面 Buff
func (bt BuffType) IsPositive() bool {
	return bt == BuffTypeDivine || bt == BuffTypeHidden ||
		bt == BuffTypeRain || bt == BuffTypeExorcism || bt == BuffTypeFire
}

// Buff 持续状态
type Buff struct {
	Type     BuffType `json:"type"`     // Buff类型
	Duration int      `json:"duration"` // 持续回合数
	Charge   int      `json:"charge"`   // 充能次数（用于青龙/玄武被动）
}

// NewBuff 创建新的 Buff
func NewBuff(buffType BuffType, duration int) *Buff {
	return &Buff{
		Type:     buffType,
		Duration: duration,
		Charge:   0,
	}
}

// IsActive 检查 Buff 是否仍然生效
// Duration == -1 表示永久生效
func (b *Buff) IsActive() bool {
	return b.Duration > 0 || b.Duration == -1 || b.Charge > 0
}

// TickDuration 减少持续时间
// 永久 Buff (Duration == -1) 不会减少
func (b *Buff) TickDuration() bool {
	if b.Duration > 0 {
		b.Duration--
	}
	return b.IsActive()
}

// ItemType 道具类型
type ItemType int

const (
	ItemTypeNone ItemType = iota
	ItemTypeReverseClock  // 反方向的钟：给予指定玩家迷途buff
	ItemTypeAnyDoor       // 任意门：去到30格内指定玩家身边
	ItemTypeDiceSwap      // 骰子交换
	ItemTypeDiceUpgrade   // 骰子升级卡
)

// String 返回道具类型名称
func (it ItemType) String() string {
	names := map[ItemType]string{
		ItemTypeNone:         "None",
		ItemTypeReverseClock: "ReverseClock",
		ItemTypeAnyDoor:      "AnyDoor",
		ItemTypeDiceSwap:     "DiceSwap",
		ItemTypeDiceUpgrade:  "DiceUpgrade",
	}
	if name, ok := names[it]; ok {
		return name
	}
	return "Unknown"
}

// Item 道具
type Item struct {
	Type     ItemType `json:"type"`     // 道具类型
	ID       string   `json:"id"`       // 道具唯一ID
	Usable   bool     `json:"usable"`   // 是否可用
	TargetID string   `json:"target_id"` // 目标玩家ID（可选）
}

// NewItem 创建新道具
func NewItem(itemType ItemType, id string) *Item {
	return &Item{
		Type:   itemType,
		ID:     id,
		Usable: true,
	}
}