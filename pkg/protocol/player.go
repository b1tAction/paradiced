package protocol

import (
	"github.com/b1tAction/paradiced/pkg/constants"
	"github.com/b1tAction/paradiced/pkg/id"
)

// PlayerReader defines read-only methods for Player.
// Used by handlers that only need to inspect player state.
type PlayerReader interface {
	GetID() id.PlayerID
	GetIDString() string // Pure UUID string for protocol compatibility
	GetHP() int
	GetLP() int
	GetPosition() int
	GetFaction() constants.Faction
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