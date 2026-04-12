package game

// ========== Item 类型定义 ==========

// ItemType 道具类型
type ItemType int

const (
	ItemTypeNone ItemType = iota
	ItemTypeReverseClock // 反方向的钟：给予指定玩家迷途buff
	ItemTypeAnyDoor      // 任意门：去到30格内指定玩家身边
	ItemTypeDiceSwap     // 骰子交换
	ItemTypeDiceUpgrade  // 骰子升级卡
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

// IsValid 检查道具类型是否有效
func (it ItemType) IsValid() bool {
	return it > ItemTypeNone && it <= ItemTypeDiceUpgrade
}

// ========== Item 实例 ==========

// Item 道具实例
type Item struct {
	Type     ItemType `json:"type"`      // 道具类型
	ID       string   `json:"id"`        // 道具唯一ID
	Usable   bool     `json:"usable"`    // 是否可用
	TargetID string   `json:"target_id"` // 目标玩家ID（可选）
}

// NewItem 创建新道具实例
func NewItem(itemType ItemType, id string) *Item {
	return &Item{
		Type:   itemType,
		ID:     id,
		Usable: true,
	}
}

// ========== Item 静态定义 ==========

// ItemDefinition 道具定义（静态配置）
type ItemDefinition struct {
	Type        ItemType       `json:"type"`         // 道具类型
	Attribute   EventAttribute `json:"attribute"`    // 道具属性（良性/中性/恶性）
	Name        string         `json:"name"`         // 道具名称（中文）
	Desc        string         `json:"desc"`         // 道具描述
	TargetSelf  bool           `json:"target_self"`  // 是否对自己使用
	TargetOther bool           `json:"target_other"` // 是否对他人使用
	BuffType    BuffType       `json:"buff_type"`    // 给予的Buff类型
	Range       int            `json:"range"`        // 作用范围（格子数）
}

// GetItemDefinition 获取道具的完整定义
func (it ItemType) GetItemDefinition() *ItemDefinition {
	definitions := map[ItemType]*ItemDefinition{
		ItemTypeReverseClock: {
			Type:        ItemTypeReverseClock,
			Attribute:   AttributeBad,
			Name:        "反方向的钟",
			Desc:        "给予指定玩家迷途Buff",
			TargetSelf:  false,
			TargetOther: true,
			BuffType:    BuffTypeLost,
		},
		ItemTypeAnyDoor: {
			Type:        ItemTypeAnyDoor,
			Attribute:   AttributeNeutral,
			Name:        "任意门",
			Desc:        "去到30格内指定玩家身边",
			TargetSelf:  false,
			TargetOther: true,
			Range:       30,
		},
		ItemTypeDiceSwap: {
			Type:        ItemTypeDiceSwap,
			Attribute:   AttributeNeutral,
			Name:        "骰子交换",
			Desc:        "与指定玩家交换骰子等级",
			TargetSelf:  false,
			TargetOther: true,
		},
		ItemTypeDiceUpgrade: {
			Type:        ItemTypeDiceUpgrade,
			Attribute:   AttributeGood,
			Name:        "骰子升级卡",
			Desc:        "将当前骰子升级为更高等级",
			TargetSelf:  true,
			TargetOther: false,
		},
	}

	if def, ok := definitions[it]; ok {
		return def
	}
	return nil
}

// ========== Item 注册表 ==========

// ItemRegistry 道具注册表
type ItemRegistry struct {
	AllItems   []ItemType `json:"all_items"`   // 所有道具
	GoodItems  []ItemType `json:"good_items"`  // 良性道具
	NeutralItems []ItemType `json:"neutral_items"` // 中性道具
	BadItems   []ItemType `json:"bad_items"`   // 恶性道具
}

// NewItemRegistry 创建道具注册表
func NewItemRegistry() *ItemRegistry {
	return &ItemRegistry{
		AllItems: []ItemType{
			ItemTypeReverseClock, ItemTypeAnyDoor, ItemTypeDiceSwap, ItemTypeDiceUpgrade,
		},
		GoodItems: []ItemType{
			ItemTypeDiceUpgrade,
		},
		NeutralItems: []ItemType{
			ItemTypeAnyDoor, ItemTypeDiceSwap,
		},
		BadItems: []ItemType{
			ItemTypeReverseClock,
		},
	}
}

// GetItemsByAttribute 按属性获取道具列表
func (ir *ItemRegistry) GetItemsByAttribute(attr EventAttribute) []ItemType {
	switch attr {
	case AttributeGood:
		return ir.GoodItems
	case AttributeNeutral:
		return ir.NeutralItems
	case AttributeBad:
		return ir.BadItems
	}
	return ir.AllItems
}

// GetAllItemDefinitions 获取所有道具定义
func (ir *ItemRegistry) GetAllItemDefinitions() []*ItemDefinition {
	defs := make([]*ItemDefinition, 0, len(ir.AllItems))
	for _, it := range ir.AllItems {
		def := it.GetItemDefinition()
		if def != nil {
			defs = append(defs, def)
		}
	}
	return defs
}

// ========== Item 属性方法 ==========

// GetItemAttribute 获取道具的属性分类
func (it ItemType) GetItemAttribute() EventAttribute {
	goodItems := []ItemType{ItemTypeDiceUpgrade}
	neutralItems := []ItemType{ItemTypeAnyDoor, ItemTypeDiceSwap}
	badItems := []ItemType{ItemTypeReverseClock}

	for _, i := range goodItems {
		if it == i {
			return AttributeGood
		}
	}
	for _, i := range neutralItems {
		if it == i {
			return AttributeNeutral
		}
	}
	for _, i := range badItems {
		if it == i {
			return AttributeBad
		}
	}
	return AttributeNeutral
}