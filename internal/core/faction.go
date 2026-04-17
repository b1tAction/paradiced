package core

import "github.com/b1tAction/paradiced/pkg/constants"

// Faction represents player faction (Four Divine Beasts).
// Use constants.Faction directly - single source of definition.

// GetFactionNames returns all faction names in Chinese.
func GetFactionNames() map[constants.Faction]string {
	return map[constants.Faction]string{
	 constants.FactionQingLong: "青龙",
	 constants.FactionZhuQue:   "朱雀",
	 constants.FactionBaiHu:    "白虎",
	 constants.FactionXuanWu:   "玄武",
	}
}

// GetFactionSkillName returns the faction passive skill name.
func GetFactionSkillName(f constants.Faction) string {
	return f.GetSkillName()
}

// GetFactionSkillDesc returns the faction passive skill description.
func GetFactionSkillDesc(f constants.Faction) string {
	return f.GetSkillDesc()
}