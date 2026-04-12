package game

// ========== Event 类型定义 ==========

// EventType 事件类型枚举
type EventType int

const (
	EventTypeNone EventType = iota

	// 良性事件 (Good)
	EventTypeHerb         // 采集到草药：HP+1
	EventTypeMilkTea      // 捡到奶茶：LP+1
	EventTypeRelic        // 捡到勇士的圣遗物：道具抽奖
	EventTypeDivineBless  // 受到天使眷顾：获得神眷Buff

	// 中性事件 (Neutral)
	EventTypeExchange     // 交换：与随机玩家交换位置
	EventTypeHiddenBuff   // 麻了：获得隐匿Buff
	EventTypeTasteTest    // 这是什么？尝一口：获得腐化/甘霖Buff（随机）

	// 恶性事件 (Bad)
	EventTypeMosquito     // 被蚊虫叮咬：HP-1
	EventTypeGhostHit     // 偶遇孤魂野鬼：HP-1
	EventTypeDogPoop      // 踩到了狗屎：LP-1
	EventTypeThief        // 啊？！贼：随机丢失道具
	EventTypeCurseBuddha  // 虔诚拜三拜：获得诅咒Buff
	EventTypeLostWay      // 迷途：获得迷途Buff
	EventTypeThunder      // 雷劫：HP归零
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

// IsValid 检查事件类型是否有效
func (et EventType) IsValid() bool {
	return et > EventTypeNone && et <= EventTypeThunder
}

// GetEventAttribute 获取事件的属性分类
func (et EventType) GetEventAttribute() EventAttribute {
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

// ========== Event 静态定义 ==========

// EventDefinition 事件定义（静态配置）
type EventDefinition struct {
	Type       EventType      `json:"type"`        // 事件类型
	Attribute  EventAttribute `json:"attribute"`   // 事件属性（良性/中性/恶性）
	Name       string         `json:"name"`        // 事件名称（中文）
	Desc       string         `json:"desc"`        // 事件描述
	HPChange   int            `json:"hp_change"`   // HP变化值
	LPChange   int            `json:"lp_change"`   // LP变化值
	BuffType   BuffType       `json:"buff_type"`   // 获得的Buff类型（0表示无）
	ItemAction string         `json:"item_action"` // 道具行为（gain/lose/draw）
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
			HPChange:  -999,
		},
	}

	if def, ok := definitions[et]; ok {
		return def
	}
	return nil
}

// ========== Event 注册表 ==========

// EventRegistry 事件注册表
type EventRegistry struct {
	AllEvents     []EventType `json:"all_events"`
	GoodEvents    []EventType `json:"good_events"`
	NeutralEvents []EventType `json:"neutral_events"`
	BadEvents     []EventType `json:"bad_events"`
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

// GetAllEventDefinitions 获取所有事件定义
func (er *EventRegistry) GetAllEventDefinitions() []*EventDefinition {
	defs := make([]*EventDefinition, 0, len(er.AllEvents))
	for _, et := range er.AllEvents {
		def := et.GetEventDefinition()
		if def != nil {
			defs = append(defs, def)
		}
	}
	return defs
}

// GetGoodEventDefinitions 获取良性事件定义
func (er *EventRegistry) GetGoodEventDefinitions() []*EventDefinition {
	defs := make([]*EventDefinition, 0, len(er.GoodEvents))
	for _, et := range er.GoodEvents {
		def := et.GetEventDefinition()
		if def != nil {
			defs = append(defs, def)
		}
	}
	return defs
}

// GetBadEventDefinitions 获取恶性事件定义
func (er *EventRegistry) GetBadEventDefinitions() []*EventDefinition {
	defs := make([]*EventDefinition, 0, len(er.BadEvents))
	for _, et := range er.BadEvents {
		def := et.GetEventDefinition()
		if def != nil {
			defs = append(defs, def)
		}
	}
	return defs
}

// GetNeutralEventDefinitions 获取中性事件定义
func (er *EventRegistry) GetNeutralEventDefinitions() []*EventDefinition {
	defs := make([]*EventDefinition, 0, len(er.NeutralEvents))
	for _, et := range er.NeutralEvents {
		def := et.GetEventDefinition()
		if def != nil {
			defs = append(defs, def)
		}
	}
	return defs
}
