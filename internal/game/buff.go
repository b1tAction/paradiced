package game

// ========== Buff 属性分类 ==========

// EventAttribute 事件/BUFF的属性分类
type EventAttribute int

const (
	AttributeGood     EventAttribute = iota // 良性（正面效果）
	AttributeNeutral                        // 中性（混合/随机效果）
	AttributeBad                            // 恶性（负面效果）
)

// String 返回属性名称
func (ea EventAttribute) String() string {
	names := map[EventAttribute]string{
		AttributeGood:     "Good",
		AttributeNeutral:  "Neutral",
		AttributeBad:      "Bad",
	}
	if name, ok := names[ea]; ok {
		return name
	}
	return "Unknown"
}

// IsValid 检查属性是否有效
func (ea EventAttribute) IsValid() bool {
	return ea >= AttributeGood && ea <= AttributeBad
}

// IsPositive 判断是否为正面属性
func (ea EventAttribute) IsPositive() bool {
	return ea == AttributeGood
}

// IsNegative 判断是否为负面属性
func (ea EventAttribute) IsNegative() bool {
	return ea == AttributeBad
}

// ========== Buff 类型定义 ==========

// BuffType Buff 类型
type BuffType int

const (
	BuffTypeNone BuffType = iota
	// 负性 Buff
	BuffTypeCurse    // 诅咒：接下来3回合LP-1
	BuffTypeLost     // 迷途：下1回合朝反方向移动
	BuffTypeCorrupt  // 腐化：接下来4回合每2回合HP-1
	BuffTypePoison   // 毒瘴：接下来3回合每回合受一次恶性随机事件
	// 正性 Buff
	BuffTypeDivine   // 神眷：接下来3回合LP+1
	BuffTypeHidden   // 隐匿：接下来3回合免疫任意事件、BUFF或道具
	BuffTypeRain     // 甘霖：接下来4回合每2回合HP+1
	BuffTypeExorcism // 辟邪：接下来5回合无视毒瘴buff
	BuffTypeFire     // 离火：朱雀阵营初始增益，每4回合LP+1
)

// String 返回 Buff 类型名称
func (bt BuffType) String() string {
	names := map[BuffType]string{
		BuffTypeNone:     "None",
		BuffTypeCurse:    "Curse",
		BuffTypeLost:     "Lost",
		BuffTypeCorrupt:  "Corrupt",
		BuffTypePoison:   "Poison",
		BuffTypeDivine:   "Divine",
		BuffTypeHidden:   "Hidden",
		BuffTypeRain:     "Rain",
		BuffTypeExorcism: "Exorcism",
		BuffTypeFire:     "Fire",
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

// GetBuffAttribute 获取 Buff 的属性分类
func (bt BuffType) GetBuffAttribute() EventAttribute {
	positiveBuffs := []BuffType{
		BuffTypeDivine, BuffTypeHidden, BuffTypeRain, BuffTypeExorcism, BuffTypeFire,
	}
	negativeBuffs := []BuffType{
		BuffTypeCurse, BuffTypeLost, BuffTypeCorrupt, BuffTypePoison,
	}

	for _, b := range positiveBuffs {
		if bt == b {
			return AttributeGood
		}
	}
	for _, b := range negativeBuffs {
		if bt == b {
			return AttributeBad
		}
	}
	return AttributeNeutral
}

// ========== Buff 实例 ==========

// Buff 持续状态实例
type Buff struct {
	Type     BuffType `json:"type"`     // Buff类型
	Duration int      `json:"duration"` // 持续回合数
	Charge   int      `json:"charge"`   // 充能次数（用于青龙/玄武被动）
}

// NewBuff 创建新的 Buff 实例
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

// ========== Buff 静态定义 ==========

// BuffDefinition Buff定义（静态配置）
type BuffDefinition struct {
	Type      BuffType       `json:"type"`       // Buff类型
	Attribute EventAttribute `json:"attribute"`  // Buff属性（良性/恶性）
	Name      string         `json:"name"`       // Buff名称（中文）
	Desc      string         `json:"desc"`       // Buff描述
	Duration  int            `json:"duration"`   // 持续回合数（-1表示永久）
	HPPerTurn int            `json:"hp_per_turn"` // 每回合HP变化
	LPPerTurn int            `json:"lp_per_turn"` // 每回合LP变化
	Special   string         `json:"special"`    // 特殊效果描述
}

// GetBuffDefinition 获取 Buff 的完整定义
func (bt BuffType) GetBuffDefinition() *BuffDefinition {
	definitions := map[BuffType]*BuffDefinition{
		BuffTypeCurse: {
			Type:      BuffTypeCurse,
			Attribute: AttributeBad,
			Name:      "诅咒",
			Desc:      "接下来3回合LP-1",
			Duration:  3,
			LPPerTurn: -1,
		},
		BuffTypeDivine: {
			Type:      BuffTypeDivine,
			Attribute: AttributeGood,
			Name:      "神眷",
			Desc:      "接下来3回合LP+1",
			Duration:  3,
			LPPerTurn: 1,
		},
		BuffTypeHidden: {
			Type:      BuffTypeHidden,
			Attribute: AttributeGood,
			Name:      "隐匿",
			Desc:      "接下来3回合免疫任意事件、BUFF或道具的影响",
			Duration:  3,
			Special:   "immune",
		},
		BuffTypeLost: {
			Type:      BuffTypeLost,
			Attribute: AttributeBad,
			Name:      "迷途",
			Desc:      "下1回合朝反方向移动",
			Duration:  1,
			Special:   "reverse",
		},
		BuffTypeCorrupt: {
			Type:      BuffTypeCorrupt,
			Attribute: AttributeBad,
			Name:      "腐化",
			Desc:      "接下来4回合每2回合HP-1",
			Duration:  4,
			HPPerTurn: -1, // 实际是每2回合生效
		},
		BuffTypeRain: {
			Type:      BuffTypeRain,
			Attribute: AttributeGood,
			Name:      "甘霖",
			Desc:      "接下来4回合每2回合HP+1",
			Duration:  4,
			HPPerTurn: 1, // 实际是每2回合生效
		},
		BuffTypeExorcism: {
			Type:      BuffTypeExorcism,
			Attribute: AttributeGood,
			Name:      "辟邪",
			Desc:      "接下来5回合无视毒瘴buff",
			Duration:  5,
			Special:   "immune_poison",
		},
		BuffTypePoison: {
			Type:      BuffTypePoison,
			Attribute: AttributeBad,
			Name:      "毒瘴",
			Desc:      "接下来3回合每回合受一次恶性随机事件影响",
			Duration:  3,
			Special:   "bad_event_per_turn",
		},
		BuffTypeFire: {
			Type:      BuffTypeFire,
			Attribute: AttributeGood,
			Name:      "离火",
			Desc:      "朱雀阵营增益，每4回合LP+1",
			Duration:  -1, // 永久
			Special:   "zhuque_passive",
		},
	}

	if def, ok := definitions[bt]; ok {
		return def
	}
	return nil
}

// ========== Buff 注册表 ==========

// BuffRegistry Buff注册表
type BuffRegistry struct {
	AllBuffs  []BuffType `json:"all_buffs"`  // 所有Buff
	GoodBuffs []BuffType `json:"good_buffs"` // 良性Buff
	BadBuffs  []BuffType `json:"bad_buffs"`  // 恶性Buff
}

// NewBuffRegistry 创建Buff注册表
func NewBuffRegistry() *BuffRegistry {
	return &BuffRegistry{
		AllBuffs: []BuffType{
			BuffTypeCurse, BuffTypeDivine, BuffTypeHidden, BuffTypeLost,
			BuffTypeCorrupt, BuffTypeRain, BuffTypeExorcism, BuffTypePoison, BuffTypeFire,
		},
		GoodBuffs: []BuffType{
			BuffTypeDivine, BuffTypeHidden, BuffTypeRain, BuffTypeExorcism, BuffTypeFire,
		},
		BadBuffs: []BuffType{
			BuffTypeCurse, BuffTypeLost, BuffTypeCorrupt, BuffTypePoison,
		},
	}
}

// GetBuffsByAttribute 按属性获取Buff列表
func (br *BuffRegistry) GetBuffsByAttribute(attr EventAttribute) []BuffType {
	switch attr {
	case AttributeGood:
		return br.GoodBuffs
	case AttributeBad:
		return br.BadBuffs
	}
	return br.AllBuffs
}

// GetAllBuffDefinitions 获取所有Buff定义
func (br *BuffRegistry) GetAllBuffDefinitions() []*BuffDefinition {
	defs := make([]*BuffDefinition, 0, len(br.AllBuffs))
	for _, bt := range br.AllBuffs {
		def := bt.GetBuffDefinition()
		if def != nil {
			defs = append(defs, def)
		}
	}
	return defs
}

// GetGoodBuffDefinitions 获取良性Buff定义
func (br *BuffRegistry) GetGoodBuffDefinitions() []*BuffDefinition {
	defs := make([]*BuffDefinition, 0, len(br.GoodBuffs))
	for _, bt := range br.GoodBuffs {
		def := bt.GetBuffDefinition()
		if def != nil {
			defs = append(defs, def)
		}
	}
	return defs
}

// GetBadBuffDefinitions 获取恶性Buff定义
func (br *BuffRegistry) GetBadBuffDefinitions() []*BuffDefinition {
	defs := make([]*BuffDefinition, 0, len(br.BadBuffs))
	for _, bt := range br.BadBuffs {
		def := bt.GetBuffDefinition()
		if def != nil {
			defs = append(defs, def)
		}
	}
	return defs
}