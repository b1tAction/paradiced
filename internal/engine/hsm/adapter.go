package hsm

import (
	"github.com/b1tAction/Fated/pkg/event"
)

// GameAdapter provides an interface for HSM to interact with Game without direct dependency.
// This allows HSM to be tested independently and decouples state logic from game implementation.
type GameAdapter interface {
	// Basic game info
	GetID() string
	GetRound() int
	GetTurn() int
	GetCurrentPhase() string

	// State management
	SetRound(round int)
	SetTurn(turn int)
	SetWaiting(waiting bool)

	// Player management
	GetPlayer(playerID string) PlayerAdapter
	GetCurrentPlayer() PlayerAdapter
	GetAllPlayers() []PlayerAdapter
	NextTurn() // Advance to next player

	// EventBus integration
	GetBus() EventBusAdapter
	PublishPhase(phase event.Phase, playerID string, ctx *StateContext) []*event.Decision

	// Buff/Item subscription
	SubscribeBuff(player PlayerAdapter, buff BuffAdapter)
	UnsubscribeBuff(buff BuffAdapter)
	ApplyBuffToPlayer(player PlayerAdapter, buff BuffAdapter)
	RemoveBuffFromPlayer(player PlayerAdapter, buff BuffAdapter)

	SubscribeItem(player PlayerAdapter, item ItemAdapter)
	UnsubscribeItem(item ItemAdapter)

	// RNG integration
	DrawEvent(lp int) EventAdapter
	DrawItem(lp int) ItemAdapter

	// Map integration
	GetMapEngine() MapEngineAdapter
}

// PlayerAdapter provides an interface for HSM to interact with Player.
type PlayerAdapter interface {
	GetUserID() string
	GetFaction() FactionAdapter
	GetPosition() int
	GetHP() int
	GetLP() int
	IsDead() bool
	CanAct() bool
	GetSkipTurn() bool

	SetPosition(pos int)
	SetHP(hp int)
	SetLP(lp int)
	SetSkipTurn(skip bool)
	SetDead(dead bool)

	// Movement
	Move(pos int, maxLen int)

	// HP/LP modification
	ApplyDamage(amount int)
	Heal(amount int)
	ModifyLP(amount int)

	// Buff management
	AddBuff(buff BuffAdapter)
	RemoveBuff(buffType BuffTypeAdapter)
	HasBuff(buffType BuffTypeAdapter) bool
	GetBuff(buffType BuffTypeAdapter) BuffAdapter
	GetAllBuffs() []BuffAdapter
	TickBuffs() []BuffAdapter // Returns expired buffs
	ClearNegativeBuffs() int

	// Item management
	AddItem(item ItemAdapter)
	RemoveItem(itemID string) ItemAdapter
	GetItem(itemID string) ItemAdapter
	HasItem(itemType ItemTypeAdapter) bool
	GetAllItems() []ItemAdapter

	// Charge/Fire counter (faction-specific)
	GetChargeCount() int
	SetChargeCount(count int)
	IncrementChargeCount() int

	GetFireCounter() int
	SetFireCounter(count int)
	IncrementFireCounter() int

	// Respawn
	Respawn(pos int)

	// Clone for testing
	Clone() PlayerAdapter
}

// EventBusAdapter provides interface for EventBus operations.
type EventBusAdapter interface {
	Publish(phase event.Phase, playerID string, ctx interface{}) []*event.Decision
	Subscribe(phase event.Phase, playerID string, sourceID string, sourceType string, decision *event.Decision)
	Unsubscribe(phase event.Phase, playerID string, sourceID string)
	GetSubscriptionCount() int
}

// BuffAdapter provides interface for Buff operations.
type BuffAdapter interface {
	GetType() BuffTypeAdapter
	GetID() string
	GetDuration() int
	SetDuration(duration int)
	GetCharge() int
	SetCharge(charge int)
	TickDuration() bool // Returns false if expired
	GetSubscriptionIDs() []string
	IsActive() bool
	IsPositive() bool
}

// BuffTypeAdapter provides interface for BuffType operations.
type BuffTypeAdapter interface {
	String() string
	IsPositive() bool
}

// ItemAdapter provides interface for Item operations.
type ItemAdapter interface {
	GetType() ItemTypeAdapter
	GetID() string
	IsUsable() bool
	GetTargetID() string
	GetSubscriptionID() string
}

// ItemTypeAdapter provides interface for ItemType operations.
type ItemTypeAdapter interface {
	String() string
}

// FactionAdapter provides interface for Faction operations.
type FactionAdapter interface {
	String() string
}

// EventAdapter provides interface for Event operations.
type EventAdapter interface {
	GetName() string
	GetDesc() string
	GetEvaluation() int
	Execute(player PlayerAdapter, ctx *StateContext)
}

// MapEngineAdapter provides interface for MapEngine operations.
type MapEngineAdapter interface {
	GetLength() int
	GetCell(pos int) CellAdapter
	CalculatePath(startPos int, steps int) PathResultAdapter
	GetLastCheckpoint(pos int) int
	SetCellType(pos int, cellType CellTypeAdapter)
	ActivateFog(pos int)
	IsFogActivated(pos int) bool
}

// CellAdapter provides interface for Cell operations.
type CellAdapter interface {
	GetPosition() int
	GetType() CellTypeAdapter
	IsFragile() bool
	IsFog() bool
	IsCheckpoint() bool
	IsBoss() bool
	IsBroken() bool
	SetBroken(broken bool)
	GetEventID() string
}

// CellTypeAdapter provides interface for CellType operations.
type CellTypeAdapter interface {
	String() string
}

// PathResultAdapter provides interface for PathResult operations.
type PathResultAdapter interface {
	GetTargetIndex() int
	GetPath() []int
	GetFellDown() bool
	GetReachedEnd() bool
	GetInterrupted() bool
}