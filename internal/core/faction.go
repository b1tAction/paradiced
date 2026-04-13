package core

// Faction represents player faction (Four Divine Beasts).
type Faction int

const (
	FactionQingLong Faction = iota // QingLong青龙 (East) - 行迹
	FactionZhuQue                  // ZhuQue朱雀 (South) - 离火
	FactionBaiHu                   // BaiHu白虎 (West) - 劫运
	FactionXuanWu                  // XuanWu玄武 (North) - 鎮厄
)

// String returns the faction name.
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

// IsValid checks if the faction is valid.
func (f Faction) IsValid() bool {
	return f >= FactionQingLong && f <= FactionXuanWu
}

// GetFactionNames returns all faction names in Chinese.
func (f Faction) GetFactionNames() map[Faction]string {
	return map[Faction]string{
		FactionQingLong: "青龙",
		FactionZhuQue:   "朱雀",
		FactionBaiHu:    "白虎",
		FactionXuanWu:   "玄武",
	}
}

// GetChineseName returns the faction Chinese name.
func (f Faction) GetChineseName() string {
	names := f.GetFactionNames()
	if name, ok := names[f]; ok {
		return name
	}
	return "未知"
}

// GetFactionSkillName returns the faction passive skill name.
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

// GetFactionSkillDesc returns the faction passive skill description.
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