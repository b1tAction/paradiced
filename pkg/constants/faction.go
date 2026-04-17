// Package constants provides unified enum type definitions.
package constants

// Faction defines player faction type (Four Divine Beasts).
type Faction string

// Faction constants - snake_case values for JSON serialization.
const (
	FactionNone Faction = "none"

	// Four Divine Beasts
	FactionQingLong Faction = "qing_long" // 青龙 (东方): 行迹
	FactionZhuQue   Faction = "zhu_que"   // 朱雀 (南方): 离火
	FactionBaiHu    Faction = "bai_hu"    // 白虎 (西方): 劫运
	FactionXuanWu   Faction = "xuan_wu"   // 玄武 (北方): 鎮厄
)

// IsValid checks if Faction is valid.
func (f Faction) IsValid() bool {
	return f == FactionQingLong || f == FactionZhuQue ||
		f == FactionBaiHu || f == FactionXuanWu
}

// AllFactions returns all valid factions.
func AllFactions() []Faction {
	return []Faction{FactionQingLong, FactionZhuQue, FactionBaiHu, FactionXuanWu}
}

// String returns the faction name in PascalCase (for logging/debugging).
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

// SnakeCase returns the faction name in snake_case (same as the value).
// This is already the natural value for Faction, so it just returns itself.
func (f Faction) SnakeCase() string {
	return string(f)
}

// GetChineseName returns the faction Chinese name.
func (f Faction) GetChineseName() string {
	names := map[Faction]string{
		FactionQingLong: "青龙",
		FactionZhuQue:   "朱雀",
		FactionBaiHu:    "白虎",
		FactionXuanWu:   "玄武",
	}
	if name, ok := names[f]; ok {
		return name
	}
	return "未知"
}

// GetSkillName returns the faction passive skill name.
func (f Faction) GetSkillName() string {
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

// GetSkillDesc returns the faction passive skill description.
func (f Faction) GetSkillDesc() string {
	descs := map[Faction]string{
		FactionQingLong: "每5回合获得充能，使用后1回合内无视负面地形",
		FactionZhuQue:   "每4回合幸运值+1，最高不超过8点",
		FactionBaiHu:    "反超其他玩家时随机从该玩家身上偷取一个Buff",
		FactionXuanWu:   "每5回合获得充能，可以抵消一次任意恶性事件",
	}
	if desc, ok := descs[f]; ok {
		return desc
	}
	return "未知"
}