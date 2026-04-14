// Package constants provides unified enum type definitions.
package constants

// Faction defines player faction type.
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