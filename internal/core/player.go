package core

import (
	"errors"
	"fmt"

	"github.com/b1tAction/Fated/internal/core/buff"
	"github.com/b1tAction/Fated/internal/core/item"
	"github.com/b1tAction/Fated/pkg/util"
)

// Player represents a player in the game.
type Player struct {
	UserID      string     `json:"user_id"`      // Player UUID
	Faction     Faction    `json:"faction"`      // Faction (阵营)
	Position    int        `json:"position"`     // Current position
	HP          int        `json:"hp"`           // Health points
	LP          int        `json:"lp"`           // Luck points (affects random events)
	Inventory   []*item.Item `json:"inventory"`    // Item inventory
	ActiveBuffs []*buff.Buff `json:"active_buffs"` // Active buffs
	IsDead      bool       `json:"is_dead"`      // Whether player is dead
	SkipTurn    bool       `json:"skip_turn"`    // Whether player skips turn
	*util.Metadata          `json:"metadata"`     // Type-safe dynamic data container
}

// PlayerConfig represents player configuration.
type PlayerConfig struct {
	UserID   string
	Faction  Faction
	MaxHP    int
	MaxLP    int
	StartPos int
}

// DefaultPlayerConfig is the default player configuration.
var DefaultPlayerConfig = PlayerConfig{
	MaxHP:    6,
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

	player := &Player{
		UserID:      config.UserID,
		Faction:     config.Faction,
		Position:    config.StartPos,
		HP:          config.MaxHP,
		LP:          config.MaxLP,
		Inventory:   make([]*item.Item, 0),
		ActiveBuffs: make([]*buff.Buff, 0),
		IsDead:      false,
		SkipTurn:    false,
		Metadata:    util.NewMetadata(),
	}

	// ZhuQue朱雀 faction starts with Fire离火 buff
	if config.Faction == FactionZhuQue {
		player.AddBuff(buff.NewBuff(buff.BuffTypeFire, -1))
	}

	return player
}

// ========== Getter Methods (implement protocol.Player) ==========

// GetUserID returns the player's user ID.
func (p *Player) GetUserID() string { return p.UserID }

// GetHP returns the player's current HP.
func (p *Player) GetHP() int { return p.HP }

// GetLP returns the player's current LP.
func (p *Player) GetLP() int { return p.LP }

// GetPosition returns the player's current position.
func (p *Player) GetPosition() int { return p.Position }

// GetFaction returns the player's faction.
func (p *Player) GetFaction() Faction { return p.Faction }

// ========== HP/LP Logic ==========

// ApplyDamage deducts HP and checks for death.
// Note: Respawn logic is handled by engine package, this only deducts HP.
func (p *Player) ApplyDamage(amount int) error {
	if amount < 0 {
		return errors.New("damage amount cannot be negative")
	}

	// Hidden隐匿 buff grants damage immunity
	if p.HasBuff(buff.BuffTypeHidden) {
		return nil
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
		return errors.New("heal amount cannot be negative")
	}
	p.HP += amount
	return nil
}

// ModifyLP modifies luck points with range limit (0~8).
func (p *Player) ModifyLP(amount int) {
	p.LP += amount
	if p.LP < 0 {
		p.LP = 0
	}
	if p.LP > 8 {
		p.LP = 8
	}
}

// ========== Movement Logic ==========

// Move moves the player to a specified position.
func (p *Player) Move(newPosition int, maxLength int) error {
	if newPosition < 0 {
		return errors.New("position cannot be negative")
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
		return errors.New("respawn position cannot be negative")
	}
	p.Position = respawnPos
	p.HP = DefaultPlayerConfig.MaxHP
	p.IsDead = false
	p.SkipTurn = false
	return nil
}

// ========== Buff Management ==========

// AddBuff adds a buff to the player.
func (p *Player) AddBuff(buffInstance *buff.Buff) error {
	if buffInstance == nil {
		return errors.New("buff cannot be nil")
	}
	if p.HasBuff(buff.BuffTypeHidden) && !buffInstance.Type.IsPositive() {
		return nil
	}
	p.ActiveBuffs = append(p.ActiveBuffs, buffInstance)
	return nil
}

// RemoveBuff removes a buff of specified type.
func (p *Player) RemoveBuff(buffType buff.BuffType) bool {
	for i, b := range p.ActiveBuffs {
		if b.Type == buffType {
			p.ActiveBuffs = append(p.ActiveBuffs[:i], p.ActiveBuffs[i+1:]...)
			return true
		}
	}
	return false
}

// HasBuff checks if player has a buff of specified type.
func (p *Player) HasBuff(buffType buff.BuffType) bool {
	for _, b := range p.ActiveBuffs {
		if b.Type == buffType && b.IsActive() {
			return true
		}
	}
	return false
}

// GetBuff gets the buff of specified type.
func (p *Player) GetBuff(buffType buff.BuffType) *buff.Buff {
	for _, b := range p.ActiveBuffs {
		if b.Type == buffType {
			return b
		}
	}
	return nil
}

// TickBuffs updates all buff durations, returns expired buffs.
func (p *Player) TickBuffs() []*buff.Buff {
	var expired []*buff.Buff
	for i := len(p.ActiveBuffs) - 1; i >= 0; i-- {
		if !p.ActiveBuffs[i].TickDuration() {
			expired = append(expired, p.ActiveBuffs[i])
			p.ActiveBuffs = append(p.ActiveBuffs[:i], p.ActiveBuffs[i+1:]...)
		}
	}
	return expired
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
func (p *Player) AddItem(itemInstance *item.Item) error {
	if itemInstance == nil {
		return errors.New("item cannot be nil")
	}
	p.Inventory = append(p.Inventory, itemInstance)
	return nil
}

// RemoveItem removes an item from inventory.
func (p *Player) RemoveItem(itemID string) (*item.Item, error) {
	for i, it := range p.Inventory {
		if it.ID == itemID {
			removed := p.Inventory[i]
			p.Inventory = append(p.Inventory[:i], p.Inventory[i+1:]...)
			return removed, nil
		}
	}
	return nil, errors.New("item not found")
}

// GetItem gets an item by ID.
func (p *Player) GetItem(itemID string) *item.Item {
	for _, it := range p.Inventory {
		if it.ID == itemID {
			return it
		}
	}
	return nil
}

// HasItem checks if player has an item of specified type.
func (p *Player) HasItem(itemType item.ItemType) bool {
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

// ========== Helper Methods ==========

// Clone clones the player (used for testing).
func (p *Player) Clone() *Player {
	inventory := make([]*item.Item, len(p.Inventory))
	for i, it := range p.Inventory {
		inventory[i] = &item.Item{
			Type:     it.Type,
			ID:       it.ID,
			Usable:   it.Usable,
			TargetID: it.TargetID,
		}
	}

	buffs := make([]*buff.Buff, len(p.ActiveBuffs))
	for i, b := range p.ActiveBuffs {
		buffs[i] = &buff.Buff{
			Type:            b.Type,
			ID:              b.ID,
			Duration:        b.Duration,
			Charge:          b.Charge,
			SubscriptionIDs: make([]string, 0),
		}
	}

	return &Player{
		UserID:      p.UserID,
		Faction:     p.Faction,
		Position:    p.Position,
		HP:          p.HP,
		LP:          p.LP,
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
		p.UserID, p.Faction.String(), p.Position, p.HP, p.LP, len(p.ActiveBuffs), len(p.Inventory))
}

// IsAlive checks if the player is alive.
func (p *Player) IsAlive() bool {
	return !p.IsDead && p.HP > 0
}

// CanAct checks if the player can act.
func (p *Player) CanAct() bool {
	return p.IsAlive() && !p.SkipTurn
}