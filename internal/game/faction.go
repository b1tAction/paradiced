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

// GetFactionNames 获取所有阵营名称（中文）
func (f Faction) GetFactionNames() map[Faction]string {
	return map[Faction]string{
		FactionQingLong: "青龙",
		FactionZhuQue:   "朱雀",
		FactionBaiHu:    "白虎",
		FactionXuanWu:   "玄武",
	}
}

// GetChineseName 获取阵营中文名称
func (f Faction) GetChineseName() string {
	names := f.GetFactionNames()
	if name, ok := names[f]; ok {
		return name
	}
	return "未知"
}

// GetFactionSkillName 获取阵营被动技能名称
func (f Faction) GetFactionSkillName() string {
	skills := map[Faction]string{
		FactionQingLong: "行迹",
		FactionZhuQue:   "离火",
		FactionBaiHu:    "劫运",
		FactionXuanWu:   "镇厄",
	}
	if name, ok := skills[f]; ok {
		return name
	}
	return "未知"
}

// GetFactionSkillDesc 获取阵营被动技能描述
func (f Faction) GetFactionSkillDesc() string {
	descs := map[Faction]string{
		FactionQingLong: "每5回合获得充能，使用后1回合内无视负面地形（迷雾debuff与Fragile）",
		FactionZhuQue:   "每4回合幸运值+1，最高不超过8点",
		FactionBaiHu:    "反超其他玩家时随机从该玩家身上偷取一个Buff",
		FactionXuanWu:   "每5回合获得充能，可以抵消一次任意恶性事件",
	}
	if desc, ok := descs[f]; ok {
		return desc
	}
	return "未知"
}