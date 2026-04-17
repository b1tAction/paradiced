package core

import "github.com/b1tAction/paradiced/pkg/constants"

// Faction represents player faction (Four Divine Beasts).
// Type alias to constants.Faction - single source of definition.
type Faction = constants.Faction

// Faction constants are defined in pkg/constants/faction.go.
// Re-export for convenience in core package.
const (
	FactionQingLong = constants.FactionQingLong // QingLong青龙 (East) - 行迹
	FactionZhuQue   = constants.FactionZhuQue   // ZhuQue朱雀 (South) - 离火
	FactionBaiHu    = constants.FactionBaiHu    // BaiHu白虎 (West) - 劫运
	FactionXuanWu   = constants.FactionXuanWu   // XuanWu玄武 (North) - 鎮厄
)

// GetFactionNames returns all faction names in Chinese.
func GetFactionNames() map[Faction]string {
	return map[Faction]string{
		FactionQingLong: "青龙",
		FactionZhuQue:   "朱雀",
		FactionBaiHu:    "白虎",
		FactionXuanWu:   "玄武",
	}
}

// GetFactionSkillName returns the faction passive skill name.
func GetFactionSkillName(f Faction) string {
	return f.GetSkillName()
}

// GetFactionSkillDesc returns the faction passive skill description.
func GetFactionSkillDesc(f Faction) string {
	return f.GetSkillDesc()
}
