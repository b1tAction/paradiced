package protocol

import "github.com/b1tAction/paradiced/pkg/id"

// Faction represents player's faction (Four Divine Beasts 阵营).
// Defined in protocol to avoid circular dependency.
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

// ========== Player Interfaces ==========

// PlayerReader defines read-only methods for Player.
// Used by handlers that only need to inspect player state.
type PlayerReader interface {
	GetID() id.PlayerID
	GetIDString() string // Pure UUID string for protocol compatibility
	GetHP() int
	GetLP() int
	GetPosition() int
	GetFaction() Faction
	IsAlive() bool
	CanAct() bool
}

// PlayerWriter defines write methods for Player.
// Used by handlers that modify player state.
// Note: Buff operations (AddBuff/RemoveBuff) should use Action system instead.
type PlayerWriter interface {
	ModifyLP(amount int)
	Heal(amount int) error
	ApplyDamage(amount int) error
	Move(newPosition int, maxLength int) error
	Respawn(respawnPos int) error
}

// Player combines Reader and Writer interfaces.
// Full interface for actions that need both read and write access.
type Player interface {
	PlayerReader
	PlayerWriter

	// Metadata methods (for Fire counter, Charge count, etc.)
	GetFireCounter() int
	SetFireCounter(count int)
	IncrementFireCounter() int
	GetChargeCount() int
	SetChargeCount(count int)
	IncrementChargeCount() int
}

// PlayerLite is a minimal interface for simple Buff handlers.
// Only includes methods commonly needed by simple Buff effects.
type PlayerLite interface {
	ModifyLP(amount int)
	GetFireCounter() int
	SetFireCounter(count int)
	IncrementFireCounter() int
}