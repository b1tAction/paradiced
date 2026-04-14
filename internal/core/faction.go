package core

import "github.com/b1tAction/fated/pkg/protocol"

// Faction represents player faction (Four Divine Beasts).
// Type alias to protocol.Faction - single source of definition.
type Faction = protocol.Faction

// Faction constants are defined in pkg/protocol/player.go.
// Re-export for convenience in core package.
const (
	FactionQingLong = protocol.FactionQingLong // QingLong青龙 (East) - 行迹
	FactionZhuQue   = protocol.FactionZhuQue   // ZhuQue朱雀 (South) - 离火
	FactionBaiHu    = protocol.FactionBaiHu    // BaiHu白虎 (West) - 劫运
	FactionXuanWu   = protocol.FactionXuanWu   // XuanWu玄武 (North) - 鎮厄
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
func GetFactionSkillDesc(f Faction) string {
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
