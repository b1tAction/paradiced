package game

import "slices"

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

// ========== 事件定义 ==========

// EventType 事件类型枚举
type EventType int

const (
	EventTypeNone EventType = iota

	// 良性事件 (Good)
	EventTypeHerb          // 采集到草药：HP+1
	EventTypeMilkTea       // 捡到奶茶：LP+1
	EventTypeRelic         // 捡到勇士的圣遗物：道具抽奖
	EventTypeDivineBless   // 受到天使眷顾：获得神眷Buff

	// 中性事件 (Neutral)
	EventTypeExchange      // 交换：与随机玩家交换位置
	EventTypeHiddenBuff    // 麻了：获得隐匿Buff
	EventTypeTasteTest     // 这是什么？尝一口：获得腐化/甘霖Buff（随机）

	// 恶性事件 (Bad)
	EventTypeMosquito      // 被蚊虫叮咬：HP-1
	EventTypeGhostHit      // 偶遇孤魂野鬼：HP-1
	EventTypeDogPoop       // 踩到了狗屎：LP-1
	EventTypeThief         // 啊？！贼：随机丢失道具
	EventTypeCurseBuddha   // 虔诚拜三拜：获得诅咒Buff
	EventTypeLostWay       // 迷途：获得迷途Buff
	EventTypeThunder       // 雷劫：HP归零
)

// String 返回事件类型名称
func (et EventType) String() string {
	names := map[EventType]string{
		EventTypeNone:        "None",
		EventTypeHerb:        "Herb",
		EventTypeMilkTea:     "MilkTea",
		EventTypeRelic:       "Relic",
		EventTypeDivineBless: "DivineBless",
		EventTypeExchange:    "Exchange",
		EventTypeHiddenBuff:  "HiddenBuff",
		EventTypeTasteTest:   "TasteTest",
		EventTypeMosquito:    "Mosquito",
		EventTypeGhostHit:    "GhostHit",
		EventTypeDogPoop:     "DogPoop",
		EventTypeThief:       "Thief",
		EventTypeCurseBuddha: "CurseBuddha",
		EventTypeLostWay:     "LostWay",
		EventTypeThunder:     "Thunder",
	}
	if name, ok := names[et]; ok {
		return name
	}
	return "Unknown"
}

// EventDefinition 事件定义（静态配置）
type EventDefinition struct {
	Type       EventType     `json:"type"`       // 事件类型
	Attribute  EventAttribute `json:"attribute"` // 事件属性（良性/中性/恶性）
	Name       string        `json:"name"`       // 事件名称（中文）
	Desc       string        `json:"desc"`       // 事件描述
	HPChange   int           `json:"hp_change"`  // HP变化值
	LPChange   int           `json:"lp_change"`  // LP变化值
	BuffType   BuffType      `json:"buff_type"`  // 获得的Buff类型（0表示无）
	ItemAction string        `json:"item_action"` // 道具行为（gain/lose/draw）
}

// GetEventAttribute 获取事件的属性分类
func (et EventType) GetAttribute() EventAttribute {
	goodEvents := []EventType{
		EventTypeHerb, EventTypeMilkTea, EventTypeRelic, EventTypeDivineBless,
	}
	neutralEvents := []EventType{
		EventTypeExchange, EventTypeHiddenBuff, EventTypeTasteTest,
	}
	badEvents := []EventType{
		EventTypeMosquito, EventTypeGhostHit, EventTypeDogPoop,
		EventTypeThief, EventTypeCurseBuddha, EventTypeLostWay, EventTypeThunder,
	}

	for _, e := range goodEvents {
		if et == e {
			return AttributeGood
		}
	}
	for _, e := range neutralEvents {
		if et == e {
			return AttributeNeutral
		}
	}
	for _, e := range badEvents {
		if et == e {
			return AttributeBad
		}
	}
	return AttributeNeutral
}

// GetEventDefinition 获取事件的完整定义
func (et EventType) GetEventDefinition() *EventDefinition {
	definitions := map[EventType]*EventDefinition{
		EventTypeHerb: {
			Type:      EventTypeHerb,
			Attribute: AttributeGood,
			Name:      "采集到草药",
			Desc:      "在路边发现了草药，恢复了体力",
			HPChange:  1,
		},
		EventTypeMilkTea: {
			Type:      EventTypeMilkTea,
			Attribute: AttributeGood,
			Name:      "捡到奶茶",
			Desc:      "捡到了一杯奶茶，一口就吃到了猪猪欸",
			LPChange:  1,
		},
		EventTypeRelic: {
			Type:       EventTypeRelic,
			Attribute:  AttributeGood,
			Name:       "捡到勇士的圣遗物",
			Desc:       "发现了古老圣遗物，获得一次道具抽奖机会",
			ItemAction: "draw",
		},
		EventTypeDivineBless: {
			Type:      EventTypeDivineBless,
			Attribute: AttributeGood,
			Name:      "受到天使眷顾",
			Desc:      "天使的祝福降临，获得神眷Buff",
			BuffType:  BuffTypeDivine,
		},
		EventTypeExchange: {
			Type:      EventTypeExchange,
			Attribute: AttributeNeutral,
			Name:      "交换",
			Desc:      "命运之手将你与另一位玩家交换位置",
		},
		EventTypeHiddenBuff: {
			Type:      EventTypeHiddenBuff,
			Attribute: AttributeNeutral,
			Name:      "麻了",
			Desc:      "身体麻木，获得隐匿Buff",
			BuffType:  BuffTypeHidden,
		},
		EventTypeTasteTest: {
			Type:      EventTypeTasteTest,
			Attribute: AttributeNeutral,
			Name:      "这是什么？尝一口",
			Desc:      "发现神秘物质，尝试后获得随机效果",
			// BuffType 在执行时随机决定（腐化或甘霖）
		},
		EventTypeMosquito: {
			Type:      EventTypeMosquito,
			Attribute: AttributeBad,
			Name:      "被蚊虫叮咬",
			Desc:      "丛林中的蚊虫叮咬了你",
			HPChange:  -1,
		},
		EventTypeGhostHit: {
			Type:      EventTypeGhostHit,
			Attribute: AttributeBad,
			Name:      "偶遇孤魂野鬼",
			Desc:      "被野鬼打了一闷棍",
			HPChange:  -1,
		},
		EventTypeDogPoop: {
			Type:      EventTypeDogPoop,
			Attribute: AttributeBad,
			Name:      "踩到了狗屎",
			Desc:      "运气糟糕的一天",
			LPChange:  -1,
		},
		EventTypeThief: {
			Type:       EventTypeThief,
			Attribute:  AttributeBad,
			Name:       "啊？！贼",
			Desc:       "遭遇盗贼，随机丢失一个道具",
			ItemAction: "lose",
		},
		EventTypeCurseBuddha: {
			Type:      EventTypeCurseBuddha,
			Attribute: AttributeBad,
			Name:      "虔诚拜三拜",
			Desc:      "拜路边的野佛，获得诅咒Buff",
			BuffType:  BuffTypeCurse,
		},
		EventTypeLostWay: {
			Type:      EventTypeLostWay,
			Attribute: AttributeBad,
			Name:      "迷途",
			Desc:      "迷失方向，获得迷途Buff",
			BuffType:  BuffTypeLost,
		},
		EventTypeThunder: {
			Type:      EventTypeThunder,
			Attribute: AttributeBad,
			Name:      "雷劫",
			Desc:      "天雷降临，HP归零",
			HPChange:  -999, // 特殊值表示归零
		},
	}

	if def, ok := definitions[et]; ok {
		return def
	}
	return nil
}

// ========== BUFF 属性定义 ==========

// GetBuffAttribute 获取 Buff 的属性分类
func (bt BuffType) GetBuffAttribute() EventAttribute {
	positiveBuffs := []BuffType{
		BuffTypeDivine, BuffTypeHidden, BuffTypeRain, BuffTypeExorcism, BuffTypeFire,
	}
	negativeBuffs := []BuffType{
		BuffTypeCurse, BuffTypeLost, BuffTypeCorrupt, BuffTypePoison,
	}

	if slices.Contains(positiveBuffs, bt) {
			return AttributeGood
		}
	if slices.Contains(negativeBuffs, bt) {
			return AttributeBad
		}
	return AttributeNeutral
}

// BuffDefinition Buff定义（静态配置）
type BuffDefinition struct {
	Type       BuffType      `json:"type"`       // Buff类型
	Attribute  EventAttribute `json:"attribute"` // Buff属性（良性/恶性）
	Name       string        `json:"name"`       // Buff名称（中文）
	Desc       string        `json:"desc"`       // Buff描述
	Duration   int           `json:"duration"`   // 持续回合数（-1表示永久）
	HPPerTurn  int           `json:"hp_per_turn"` // 每回合HP变化
	LPPerTurn  int           `json:"lp_per_turn"` // 每回合LP变化
	Special    string        `json:"special"`    // 特殊效果描述
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

// ========== 事件注册表 ==========

// EventRegistry 事件注册表（用于 RNG 抽卡）
type EventRegistry struct {
	AllEvents    []EventType `json:"all_events"`    // 所有事件
	GoodEvents   []EventType `json:"good_events"`   // 良性事件
	NeutralEvents []EventType `json:"neutral_events"` // 中性事件
	BadEvents    []EventType `json:"bad_events"`    // 恶性事件
}

// NewEventRegistry 创建事件注册表
func NewEventRegistry() *EventRegistry {
	return &EventRegistry{
		AllEvents: []EventType{
			EventTypeHerb, EventTypeMilkTea, EventTypeRelic, EventTypeDivineBless,
			EventTypeExchange, EventTypeHiddenBuff, EventTypeTasteTest,
			EventTypeMosquito, EventTypeGhostHit, EventTypeDogPoop,
			EventTypeThief, EventTypeCurseBuddha, EventTypeLostWay, EventTypeThunder,
		},
		GoodEvents: []EventType{
			EventTypeHerb, EventTypeMilkTea, EventTypeRelic, EventTypeDivineBless,
		},
		NeutralEvents: []EventType{
			EventTypeExchange, EventTypeHiddenBuff, EventTypeTasteTest,
		},
		BadEvents: []EventType{
			EventTypeMosquito, EventTypeGhostHit, EventTypeDogPoop,
			EventTypeThief, EventTypeCurseBuddha, EventTypeLostWay, EventTypeThunder,
		},
	}
}

// GetEventsByAttribute 按属性获取事件列表
func (er *EventRegistry) GetEventsByAttribute(attr EventAttribute) []EventType {
	switch attr {
	case AttributeGood:
		return er.GoodEvents
	case AttributeNeutral:
		return er.NeutralEvents
	case AttributeBad:
		return er.BadEvents
	}
	return er.AllEvents
}

// BuffRegistry Buff注册表
type BuffRegistry struct {
	AllBuffs     []BuffType `json:"all_buffs"`     // 所有Buff
	GoodBuffs    []BuffType `json:"good_buffs"`    // 良性Buff
	BadBuffs     []BuffType `json:"bad_buffs"`     // 恶性Buff
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