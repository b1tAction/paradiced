package net

// StateSync represents complete game state for synchronization.
// Broadcast when entering a new state or for reconnecting players.
type StateSync struct {
	// GlobalState is the current global state (Layer 1).
	// Values: "match_init", "round_mini_game", "round_prep", "turn_loop", "boss_battle", "game_over"
	GlobalState string `json:"global_state"`

	// TurnState is the current turn state (Layer 2), empty if not in turn loop.
	// Values: "turn_upkeep", "main_action", "turn_moving", "turn_landed", "turn_event", "turn_end"
	TurnState string `json:"turn_state"`

	// TurnPlayer is the user ID of the player whose turn is active.
	TurnPlayer string `json:"turn_player"`

	// Round is the current round number.
	Round int `json:"round"`

	// Turn is the current turn index within the round (0-3 for 4 players).
	Turn int `json:"turn"`

	// Paused indicates the game is waiting for a decision (interrupt state).
	Paused bool `json:"paused"`

	// Players contains all player state snapshots.
	Players []Player `json:"players"`
}

// Player represents a player state snapshot for synchronization.
// Builder extracts known keys from core.Player.Metadata into typed fields.
type Player struct {
	// UserID is the player's unique identifier.
	UserID string `json:"user_id"`

	// Faction is the player's faction (青龙/朱雀/白虎/玄武).
	Faction string `json:"faction"`

	// Position is the player's current position on the map.
	Position int `json:"position"`

	// HP is the player's current health points.
	HP int `json:"hp"`

	// LP is the player's current luck points (0-8).
	LP int `json:"lp"`

	// Buffs contains the player's active buffs.
	Buffs []Buff `json:"buffs"`

	// Items contains the player's inventory items.
	Items []Item `json:"items"`

	// Charge is the faction skill charge count (青龙行迹/玄武镇厄).
	Charge int `json:"charge"`

	// FireCounter is the 朱雀离火 fire counter (every 4 turns LP+1).
	FireCounter int `json:"fire_counter"`

	// IsDead indicates the player has died and needs respawn.
	IsDead bool `json:"is_dead"`

	// SkipTurn indicates the player will skip this turn.
	SkipTurn bool `json:"skip_turn"`
}

// Buff represents a buff state for synchronization.
type Buff struct {
	// Type is the buff type identifier (snake_case).
	// Values: "divine", "curse", "fire", "lost", "hidden", "rain", "corrupt", "exorcism", "poison"
	Type string `json:"type"`

	// Duration is the remaining turn count. -1 for permanent buffs (离火).
	Duration int `json:"duration"`
}

// Item represents an item state for synchronization.
type Item struct {
	// ID is the unique item instance ID.
	ID string `json:"id"`

	// Type is the item type identifier (snake_case).
	Type string `json:"type"`
}

// ActionSync represents an action execution result for client rendering.
// Maps directly from gamelog.LogEntry.
type ActionSync struct {
	// ActionType is the action type (snake_case).
	// Values: "damage", "heal", "modify_lp", "move", "add_buff", "remove_buff", "respawn", etc.
	ActionType string `json:"action_type"`

	// Target is the player ID affected by this action.
	Target string `json:"target"`

	// Delta is the change amount (negative for damage/LP loss, positive for heal/LP gain).
	Delta int `json:"delta"`

	// Source is the action source identifier (buff ID, item ID, event ID, "System").
	Source string `json:"source"`

	// Metadata contains additional action data (path, position, etc).
	// Converted from util.Metadata.ToMap().
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// Available represents available actions for the current player.
// Sent when entering MainAction state.
type Available struct {
	// Items contains usable items (PhaseAnyTime items).
	Items []Item `json:"items"`

	// CanUseSkill indicates the faction skill is available.
	// True when charge count >= 1 for 青龙/玄武.
	CanUseSkill bool `json:"can_use_skill"`

	// DiceType is the player's dice type based on mini-game ranking.
	// Values: "gold" (rank 1), "silver" (rank 2), "copper" (rank 3), "wood" (rank 4)
	DiceType string `json:"dice_type"`
}

// MiniGameStart represents mini-game start notification.
type MiniGameStart struct {
	// GameType identifies the mini-game type for client to load.
	GameType string `json:"game_type"`
}

// GameOver represents game end notification.
type GameOver struct {
	// WinnerID is the winning player's user ID.
	WinnerID string `json:"winner_id"`

	// Stats contains end-game statistics for all players.
	Stats []PlayerStats `json:"stats"`
}

// PlayerStats represents a player's end-game statistics.
type PlayerStats struct {
	// UserID is the player's user ID.
	UserID string `json:"user_id"`

	// RoundsWon is the number of rounds won (mini-game ranking 1).
	RoundsWon int `json:"rounds_won"`

	// EventsDrawn is the number of random events drawn.
	EventsDrawn int `json:"events_drawn"`

	// ItemsUsed is the number of items consumed.
	ItemsUsed int `json:"items_used"`
}