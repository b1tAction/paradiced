package core

import (
	"errors"
	"fmt"

	"github.com/b1tAction/Fated/pkg/util"
)

// Player represents a player in the game.
type Player struct {
	UserID      string     `json:"user_id"`      // Player UUID
	Faction     Faction    `json:"faction"`      // Faction (阵营)
	Position    int        `json:"position"`     // Current position
	HP          int        `json:"hp"`           // Health points
	LP          int        `json:"lp"`           // Luck points (affects random events)
	Inventory   []*Item    `json:"inventory"`    // Item inventory
	ActiveBuffs []*Buff    `json:"active_buffs"` // Active buffs
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
		Inventory:   make([]*Item, 0),
		ActiveBuffs: make([]*Buff, 0),
		IsDead:      false,
		SkipTurn:    false,
		Metadata:    util.NewMetadata(),
	}

	// ZhuQue朱雀 faction starts with Fire离火 buff
	if config.Faction == FactionZhuQue {
		player.AddBuff(NewBuff(BuffTypeFire, -1))
	}

	return player
}

// ========== HP/LP Logic ==========

// ApplyDamage deducts HP and checks for death.
// Note: Respawn logic is handled by engine package, this only deducts HP.
func (p *Player) ApplyDamage(amount int) error {
	if amount < 0 {
		return errors.New("damage amount cannot be negative")
	}

	// Hidden隐匿 buff grants damage immunity
	if p.HasBuff(BuffTypeHidden) {
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
func (p *Player) AddBuff(buff *Buff) error {
	if buff == nil {
		return errors.New("buff cannot be nil")
	}
	if p.HasBuff(BuffTypeHidden) && !buff.Type.IsPositive() {
		return nil
	}
	p.ActiveBuffs = append(p.ActiveBuffs, buff)
	return nil
}

// RemoveBuff removes a buff of specified type.
func (p *Player) RemoveBuff(buffType BuffType) bool {
	for i, buff := range p.ActiveBuffs {
		if buff.Type == buffType {
			p.ActiveBuffs = append(p.ActiveBuffs[:i], p.ActiveBuffs[i+1:]...)
			return true
		}
	}
	return false
}

// HasBuff checks if player has a buff of specified type.
func (p *Player) HasBuff(buffType BuffType) bool {
	for _, buff := range p.ActiveBuffs {
		if buff.Type == buffType && buff.IsActive() {
			return true
		}
	}
	return false
}

// GetBuff gets the buff of specified type.
func (p *Player) GetBuff(buffType BuffType) *Buff {
	for _, buff := range p.ActiveBuffs {
		if buff.Type == buffType {
			return buff
		}
	}
	return nil
}

// TickBuffs updates all buff durations, returns expired buffs.
func (p *Player) TickBuffs() []*Buff {
	var expired []*Buff
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
func (p *Player) AddItem(item *Item) error {
	if item == nil {
		return errors.New("item cannot be nil")
	}
	p.Inventory = append(p.Inventory, item)
	return nil
}

// RemoveItem removes an item from inventory.
func (p *Player) RemoveItem(itemID string) (*Item, error) {
	for i, item := range p.Inventory {
		if item.ID == itemID {
			removed := p.Inventory[i]
			p.Inventory = append(p.Inventory[:i], p.Inventory[i+1:]...)
			return removed, nil
		}
	}
	return nil, errors.New("item not found")
}

// GetItem gets an item by ID.
func (p *Player) GetItem(itemID string) *Item {
	for _, item := range p.Inventory {
		if item.ID == itemID {
			return item
		}
	}
	return nil
}

// HasItem checks if player has an item of specified type.
func (p *Player) HasItem(itemType ItemType) bool {
	for _, item := range p.Inventory {
		if item.Type == itemType {
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
	inventory := make([]*Item, len(p.Inventory))
	for i, item := range p.Inventory {
		inventory[i] = &Item{
			Type:     item.Type,
			ID:       item.ID,
			Usable:   item.Usable,
			TargetID: item.TargetID,
		}
	}

	buffs := make([]*Buff, len(p.ActiveBuffs))
	for i, buff := range p.ActiveBuffs {
		buffs[i] = &Buff{
			Type:            buff.Type,
			ID:              buff.ID,
			Duration:        buff.Duration,
			Charge:          buff.Charge,
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