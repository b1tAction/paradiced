// Package core provides core data structures for the Paradiced game.
package core

import (
	"fmt"

	"github.com/b1tAction/paradiced/pkg/constants"
	pkgerrors "github.com/b1tAction/paradiced/pkg/errors"
	"github.com/b1tAction/paradiced/pkg/id"
	"github.com/b1tAction/paradiced/pkg/util"
)

// Player represents a player in the game.
type Player struct {
	ID             id.PlayerID       `json:"id"`           // Player unique identifier (UUID v7)
	Faction        constants.Faction `json:"faction"`      // Faction (阵营)
	Position       int               `json:"position"`     // Current position
	HP             int               `json:"hp"`           // Health points
	LP             int               `json:"lp"`           // Luck points (affects random events)
	MaxHP		   int				 `json:"max_hp"`	   // Maximum health points
	MaxLP		   int				 `json:"max_lp"`	   // Maximum luck points
	InitHP		   int				 `json:"init_hp"`	   // Initial health points (used for respawn reset)
	Inventory      []*Item           `json:"inventory"`    // Item inventory
	ActiveBuffs    []*Buff           `json:"active_buffs"` // Active buffs
	IsDead         bool              `json:"is_dead"`      // Whether player is dead
	SkipTurn       bool              `json:"skip_turn"`    // Whether player skips turn
	*util.Metadata `json:"metadata"` // Type-safe dynamic data container
}

// PlayerConfig represents player configuration.
type PlayerConfig struct {
	ID       id.PlayerID
	Faction  constants.Faction
	InitHP	 int
	InitLP	 int
	MaxHP    int
	MaxLP    int
	StartPos int
}

// DefaultPlayerConfig is the default player configuration.
var DefaultPlayerConfig = PlayerConfig{
	InitHP:   6,
	InitLP:   4,
	MaxHP:    8,
	MaxLP:    8, // Consistent with ModifyLP LP range limit
	StartPos: 0,
}

// NewPlayer creates a new player.
func NewPlayer(config PlayerConfig) *Player {
	if config.MaxHP <= 0 {
		config.MaxHP = DefaultPlayerConfig.MaxHP
	}
	if config.MaxLP <= 0 {
		config.MaxLP = DefaultPlayerConfig.MaxLP
	}
	if config.InitHP <= 0 {
		config.InitHP = DefaultPlayerConfig.InitHP
	}
	if config.InitLP <= 0 {
		config.InitLP = DefaultPlayerConfig.InitLP
	}
	// Ensure InitHP does not exceed MaxHP
	if config.InitHP > config.MaxHP {
		config.InitHP = config.MaxHP
	}

	// Generate ID if not provided
	if config.ID.IsZero() {
		config.ID = id.NewPlayerID()
	}

	player := &Player{
		ID:          config.ID,
		Faction:     config.Faction,
		Position:    config.StartPos,
		HP:          config.InitHP,
		LP:          config.InitLP,
		MaxHP: 		 config.MaxHP,
		MaxLP: 		 config.MaxLP,
		InitHP:      config.InitHP,
		Inventory:   make([]*Item, 0),
		ActiveBuffs: make([]*Buff, 0),
		IsDead:      false,
		SkipTurn:    false,
		Metadata:    util.NewMetadata(),
	}

	return player
}

// ========== Getter Methods ==========

// GetID returns the player's ID.
func (p *Player) GetID() id.PlayerID { return p.ID }

// GetIDString returns the player's ID as pure UUID string (for protocol compatibility).
func (p *Player) GetIDString() string { return p.ID.UUID() }

// GetHP returns the player's current HP.
func (p *Player) GetHP() int { return p.HP }

// GetLP returns the player's current LP.
func (p *Player) GetLP() int { return p.LP }

// GetPosition returns the player's current position.
func (p *Player) GetPosition() int { return p.Position }

// GetFaction returns the player's faction.
func (p *Player) GetFaction() constants.Faction { return p.Faction }

// ========== HP/LP Logic ==========

// ApplyDamage deducts HP and checks for death.
// Note: Respawn logic is handled by engine package, this only deducts HP.
func (p *Player) ApplyDamage(amount int) error {
	if amount < 0 {
		return pkgerrors.NewValidationError("damage_amount", amount, "must be non-negative")
	}

	p.HP -= amount

	if p.HP <= 0 {
		p.HP = 0
		p.IsDead = true
	}

	return nil
}

// Heal restores HP.
func (p *Player) Heal(amount int) error {
	if amount < 0 {
		return pkgerrors.NewValidationError("heal_amount", amount, "must be non-negative")
	}
	p.HP += amount
	if (p.HP > p.MaxHP) {
		p.HP = p.MaxHP
	}
	return nil
}

// ModifyLP modifies luck points with range limit (0~8).
func (p *Player) ModifyLP(amount int) {
	p.LP += amount
	if p.LP < 0 {
		p.LP = 0
	}
	if p.LP > p.MaxLP {
		p.LP = p.MaxLP
	}
}

// ========== Movement Logic ==========

// Move moves the player to a specified position.
func (p *Player) Move(newPosition int, maxLength int) error {
	if newPosition < 0 {
		return pkgerrors.NewValidationError("position", newPosition, "must be non-negative")
	}
	if newPosition >= maxLength {
		newPosition = maxLength - 1
	}
	p.Position = newPosition
	return nil
}

// Respawn respawns the player at checkpoint.
func (p *Player) Respawn(respawnPos int) error {
	if respawnPos < 0 {
		return pkgerrors.NewValidationError("respawn_pos", respawnPos, "must be non-negative")
	}
	p.Position = respawnPos
	p.HP = p.InitHP // Reset HP to player's InitHP (not MaxHP)
	p.IsDead = false
	p.SkipTurn = false
	return nil
}

// ========== Buff Management ==========

// AddBuff adds a buff to the player.
func (p *Player) AddBuff(buffInstance *Buff) error {
	if buffInstance == nil {
		return pkgerrors.NewInternalError("Player", "AddBuff", nil).
		WithContext("reason", "buff instance is nil")
	}

	// Check if player already has a buff of the same type
	existing := p.GetBuff(buffInstance.Type)
	if existing != nil {
		// Extend duration: add the new buff's duration to the existing one
		// Permanent buffs (-1) stay permanent regardless
		if existing.Duration != -1 && buffInstance.Duration != -1 {
			existing.Duration += buffInstance.Duration
		}
		// Preserve the existing buff's tickEligible state.
		// If the buff was already marked eligible at TurnUpkeep, it should be
		// decremented at this turn's TurnEnd. The extended duration portion is
		// naturally protected because TickDuration decrements by 1, not proportionally.
		return nil // Don't add new instance, duration was extended
	}

	p.ActiveBuffs = append(p.ActiveBuffs, buffInstance)
	return nil
}

// RemoveBuff removes a buff of specified type.
func (p *Player) RemoveBuff(buffType constants.BuffType) bool {
	for i, b := range p.ActiveBuffs {
		if b.Type == buffType {
			p.ActiveBuffs = append(p.ActiveBuffs[:i], p.ActiveBuffs[i+1:]...)
			return true
		}
	}
	return false
}

// HasBuff checks if player has a buff of specified type.
func (p *Player) HasBuff(buffType constants.BuffType) bool {
	for _, b := range p.ActiveBuffs {
		if b.Type == buffType && b.IsActive() {
			return true
		}
	}
	return false
}

// GetBuff gets the buff of specified type.
func (p *Player) GetBuff(buffType constants.BuffType) *Buff {
	for _, b := range p.ActiveBuffs {
		if b.Type == buffType {
			return b
		}
	}
	return nil
}

// TickBuffs updates all buff durations, returns expired buffs.
// TickBuffDurations decrements duration for tick-eligible buffs and returns expired ones.
// Unlike TickBuffs, this does NOT remove expired buffs from ActiveBuffs —
// removal should be handled by RemoveBuffAction through the Action system.
func (p *Player) TickBuffs() []*Buff {
	var expired []*Buff
	for i := range p.ActiveBuffs {
		if !p.ActiveBuffs[i].TickDuration() {
			expired = append(expired, p.ActiveBuffs[i])
		}
	}
	return expired
}

// MarkAllBuffsTickEligible marks all active buffs as tick-eligible.
// Called at the start of each turn (TurnUpkeep) so that buffs added mid-turn
// (e.g., by another player's item targeting this player) are properly decremented
// at this turn's TurnEnd, instead of getting an extra free turn.
func (p *Player) MarkAllBuffsTickEligible() {
	for i := range p.ActiveBuffs {
		p.ActiveBuffs[i].tickEligible = true
	}
}

// ClearNegativeBuffs clears all negative buffs.
func (p *Player) ClearNegativeBuffs() int {
	count := 0
	for i := len(p.ActiveBuffs) - 1; i >= 0; i-- {
		if !p.ActiveBuffs[i].Type.IsPositive() {
			p.ActiveBuffs = append(p.ActiveBuffs[:i], p.ActiveBuffs[i+1:]...)
			count++
		}
	}
	return count
}

// ========== Item Management ==========

// AddItem adds an item to inventory.
func (p *Player) AddItem(itemInstance *Item) error {
	if itemInstance == nil {
		return pkgerrors.NewInternalError("Player", "AddItem", nil).
		WithContext("reason", "item instance is nil")
	}
	p.Inventory = append(p.Inventory, itemInstance)
	return nil
}

// RemoveItem removes an item from inventory.
func (p *Player) RemoveItem(itemID id.ItemID) (*Item, error) {
	for i, it := range p.Inventory {
		if it.ID.Equal(itemID.ID) {
			removed := p.Inventory[i]
			p.Inventory = append(p.Inventory[:i], p.Inventory[i+1:]...)
			return removed, nil
		}
	}
	return nil, pkgerrors.NewInternalError("Player", "RemoveItem", nil).
		WithContext("reason", "item not found")
}

// GetItem gets an item by ID.
func (p *Player) GetItem(itemID id.ItemID) *Item {
	for _, it := range p.Inventory {
		if it.ID.Equal(itemID.ID) {
			return it
		}
	}
	return nil
}

// HasItem checks if player has an item of specified type.
func (p *Player) HasItem(itemType constants.ItemType) bool {
	for _, it := range p.Inventory {
		if it.Type == itemType {
			return true
		}
	}
	return false
}

// ========== Charge Count Methods (using Metadata) ==========

// GetChargeCount gets the charge count (used by QingLong青龙/XuanWu玄武 faction).
func (p *Player) GetChargeCount() int {
	return p.GetIntOrDefault("charge_count", 0)
}

// SetChargeCount sets the charge count.
func (p *Player) SetChargeCount(count int) {
	p.SetInt("charge_count", count)
}

// IncrementChargeCount increments the charge count, returns new value.
func (p *Player) IncrementChargeCount() int {
	return p.IncrementInt("charge_count", 1)
}

// ========== Charge Turn Counter Methods (using Metadata) ==========

// GetChargeTurnCounter gets the charge turn counter (used for every 2 turns charging).
func (p *Player) GetChargeTurnCounter() int {
	return p.GetIntOrDefault("charge_turn_counter", 0)
}

// SetChargeTurnCounter sets the charge turn counter.
func (p *Player) SetChargeTurnCounter(count int) {
	p.SetInt("charge_turn_counter", count)
}

// IncrementChargeTurnCounter increments the charge turn counter, returns new value.
func (p *Player) IncrementChargeTurnCounter() int {
	return p.IncrementInt("charge_turn_counter", 1)
}

// ========== Fire Counter Methods (using Metadata) ==========

// GetFireCounter gets the fire counter (used by ZhuQue朱雀 faction).
func (p *Player) GetFireCounter() int {
	return p.GetIntOrDefault("fire_counter", 0)
}

// SetFireCounter sets the fire counter.
func (p *Player) SetFireCounter(count int) {
	p.SetInt("fire_counter", count)
}

// IncrementFireCounter increments the fire counter, returns new value.
func (p *Player) IncrementFireCounter() int {
	return p.IncrementInt("fire_counter", 1)
}

// ========== Game Stats Methods (using Metadata) ==========

// GetEventsDrawn returns the number of random events drawn by this player.
func (p *Player) GetEventsDrawn() int {
	return p.GetIntOrDefault("events_drawn", 0)
}

// IncrementEventsDrawn increments the events drawn counter, returns new value.
func (p *Player) IncrementEventsDrawn() int {
	return p.IncrementInt("events_drawn", 1)
}

// GetItemsUsed returns the number of items consumed by this player.
func (p *Player) GetItemsUsed() int {
	return p.GetIntOrDefault("items_used", 0)
}

// IncrementItemsUsed increments the items used counter, returns new value.
func (p *Player) IncrementItemsUsed() int {
	return p.IncrementInt("items_used", 1)
}

// GetRoundsWon returns the number of mini-game rounds won (rank 1) by this player.
func (p *Player) GetRoundsWon() int {
	return p.GetIntOrDefault("rounds_won", 0)
}

// IncrementRoundsWon increments the rounds won counter, returns new value.
func (p *Player) IncrementRoundsWon() int {
	return p.IncrementInt("rounds_won", 1)
}

// ========== Helper Methods ==========

// Clone clones the player (used for testing).
func (p *Player) Clone() *Player {
	inventory := make([]*Item, len(p.Inventory))
	for i, it := range p.Inventory {
		inventory[i] = &Item{
			Type:     it.Type,
			ID:       it.ID,
			Usable:   it.Usable,
			TargetID: it.TargetID,
		}
	}

	buffs := make([]*Buff, len(p.ActiveBuffs))
	for i, b := range p.ActiveBuffs {
		buffs[i] = &Buff{
			Type:         b.Type,
			ID:           b.ID,
			Duration:     b.Duration,
			tickEligible: b.tickEligible,
		}
	}

	return &Player{
		ID:          p.ID,
		Faction:     p.Faction,
		Position:    p.Position,
		HP:          p.HP,
		LP:          p.LP,
		MaxHP: 		 p.MaxHP,
		MaxLP: 		 p.MaxLP,
		InitHP:      p.InitHP,
		Inventory:   inventory,
		ActiveBuffs: buffs,
		IsDead:      p.IsDead,
		SkipTurn:    p.SkipTurn,
		Metadata:    p.Metadata.Clone(),
	}
}

// String returns the player info string.
func (p *Player) String() string {
	return fmt.Sprintf("Player{ID: %s, Faction: %s, Pos: %d, HP: %d, LP: %d, Buffs: %d, Items: %d}",
		p.ID.UUID(), string(p.Faction), p.Position, p.HP, p.LP, len(p.ActiveBuffs), len(p.Inventory))
}

// IsAlive checks if the player is alive.
func (p *Player) IsAlive() bool {
	return !p.IsDead && p.HP > 0
}

// CanAct checks if the player can act.
func (p *Player) CanAct() bool {
	return p.IsAlive() && !p.SkipTurn
}