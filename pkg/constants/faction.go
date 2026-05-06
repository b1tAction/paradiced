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

// ParseFaction converts a string to Faction type.
// Returns FactionNone if the string is not a valid faction.
func ParseFaction(s string) Faction {
	switch s {
	case "qing_long":
		return FactionQingLong
	case "zhu_que":
		return FactionZhuQue
	case "bai_hu":
		return FactionBaiHu
	case "xuan_wu":
		return FactionXuanWu
	default:
		return FactionNone
	}
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
		FactionQingLong: "每2回合获得充能，使用后1回合内增益效果翻倍",
		FactionZhuQue:   "每3回合幸运值+1，最高不超过8点",
		FactionBaiHu:    "每2回合获得充能，指定目标玩家，使其增益效果改向自身",
		FactionXuanWu:   "每2回合获得充能，使用后1回合免疫恶性事件和负面Buff",
	}
	if desc, ok := descs[f]; ok {
		return desc
	}
	return "未知"
}