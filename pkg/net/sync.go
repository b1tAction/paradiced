// Package net provides network message protocol definitions for client-server communication.
// This package defines the message opcodes, data structures, and abstract handler interface
// for implementing authoritative server communication (e.g., Nakama Match Handler).
package net

import (
	"time"

	"github.com/b1tAction/paradiced/pkg/constants"
	"github.com/b1tAction/paradiced/pkg/gamelog"
)

// StateSync represents complete game state for synchronization.
// Broadcast when entering a new state or for reconnecting players.
type StateSync struct {
	// GlobalState is the current global state (Layer 1).
	// Values: "MatchInit", "WaitingForHost", "RoundMiniGame", "RoundPrep", "TurnLoop", "RoundEndWait", "GameOver"
	GlobalState string `json:"global_state"`

	// TurnState is the current turn state (Layer 2), empty if not in turn loop.
	// Values: "TurnUpkeep", "MainAction", "TurnMoving", "TurnLanded", "TurnDraw", "TurnBossBattle", "TurnEnd"
	TurnState string `json:"turn_state"`

	// CurrentPlayerID is the player ID whose turn is active.
	// Matches core.Player.ID.UUID() format.
	CurrentPlayerID string `json:"current_player_id"`

	// Round is the current round number.
	Round int `json:"round"`

	// Turn is the current turn index within the round (0-3 for 4 players).
	Turn int `json:"turn"`

	// Paused indicates the game is waiting for a decision (interrupt state).
	Paused bool `json:"paused"`

	// Players contains all player state snapshots.
	Players []Player `json:"players"`

	// Map contains the map information for client rendering.
	Map MapInfo `json:"map"`
}

// TurnSync represents all events for a turn/phase.
// Broadcast after executing effects, client renders entries sequentially.
//
// LogEntry.Metadata Field Contract:
// Each action_type has specific metadata fields that client should parse.
//
// | action_type   | metadata fields                                                      | client usage                |
// |---------------|----------------------------------------------------------------------|----------------------------|
// | damage        | hp_change: int, blocked_by?: string, piercing?: bool                 | 显示伤害数值、阻挡来源、穿透效果 |
// | heal          | hp_change: int                                                       | 显示治疗数值动画             |
// | modify_lp     | lp_change: int                                                       | 显示LP变化数值动画           |
// | move          | steps: int, start_pos: int, end_pos: int, path: []int                | 显示移动路径动画             |
// | add_buff      | buff_type: string, duration: int                                     | 显示获得Buff动画及持续时间   |
// | remove_buff   | buff_type: string                                                    | 显示移除Buff动画           |
// | teleport      | from_pos: int, to_pos: int                                           | 显示传送动画               |
// | steal_buff    | stolen_by: string, buff_type: string                                 | 显示白虎劫运动画           |
// | respawn       | checkpoint_pos: int                                                  | 显示重生动画               |
// | fell_down     | position: int, hp_change: int                                        | 显示落坑动画及坠落伤害       |
// | draw_event    | event_type: string                                                 | 显示抽取事件动画           |
// | draw_item     | item_type: string                                                  | 显示道具获取动画           |
// | dice_roll     | dice_type: string, dice_steps: int                                   | 显示骰子动画               |
// | state         | from: string, to: string                                             | 状态转换记录               |
//
// Client rendering example (TypeScript):
//
//	for (const entry of turnSync.entries) {
//	    switch (entry.action_type) {
//	        case "move":
//	            const path = entry.metadata?.path || [];
//	            playMoveAnimation(entry.target, path);
//	            break;
//	        case "add_buff":
//	            const buffType = entry.metadata?.buff_type;
//	            playBuffGainAnimation(entry.target, buffType);
//	            break;
//	    }
//	}
type TurnSync struct {
	// Round is the current round number.
	Round int `json:"round"`

	// Turn is the current turn index.
	Turn int `json:"turn"`

	// CurrentPlayerID is the player ID whose turn is active.
	// Matches core.Player.ID.UUID() format.
	CurrentPlayerID string `json:"current_player_id"`

	// Entries contains all log entries for this turn/phase.
	// Directly uses gamelog.LogEntry (no conversion to Action).
	Entries []gamelog.LogEntry `json:"entries"`
}

// MapInfo represents map data for client synchronization.
type MapInfo struct {
	// Length is the total number of cells in the map.
	Length int `json:"length"`

	// Cells contains cell information for the map.
	Cells []CellInfo `json:"cells"`
}

// CellInfo represents a single map cell for client synchronization.
type CellInfo struct {
	// Index is the cell position (0 to Length-1).
	Index int `json:"index"`

	// CellType is the cell type identifier (snake_case).
	// Values: "normal", "fragile", "fog", "checkpoint", "boss", "event"
	CellType string `json:"cell_type"`

	// EventID is the bound event ID for CellTypeEvent cells.
	EventID string `json:"event_id,omitempty"`

	// IsBroken indicates whether a Fragile cell is broken.
	IsBroken bool `json:"is_broken,omitempty"`
}

// Player represents a player state snapshot for synchronization.
// Builder extracts known keys from core.Player.Metadata into typed fields.
type Player struct {
	// PlayerID is the player's game internal ID (UUID format).
	// For Nakama: equals Nakama session.UserID (UUID format) - no separate ClientID.
	// Matches core.Player.ID.UUID() format.
	PlayerID string `json:"player_id"`

	// DisplayName is the user-provided display name for UI rendering.
	// Falls back to PlayerID (UUID) if not provided.
	DisplayName string `json:"display_name"`

	// Faction is the player's faction (snake_case: "qing_long", "zhu_que", "bai_hu", "xuan_wu").
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

	// IsBoss indicates this is the Boss special player.
	// Only true for the Boss entity at the end of the Players list.
	IsBoss bool `json:"is_boss,omitempty"`
}

// Buff represents a buff state for synchronization.
// Includes display name for client UI.
type Buff struct {
	// Type is the buff type identifier (snake_case).
	// Values: "divine", "curse", "fire", "lost", "hidden", "rain", "corrupt", "exorcism", "poison"
	Type string `json:"type"`

	// Name is the buff Chinese display name for client UI.
	// Values: "神眷", "诅咒", "离火", "迷途", "隐匿", "甘霖", "腐化", "辟邪", "毒瘴"
	Name string `json:"name"`

	// Duration is the remaining turn count. -1 for permanent buffs (离火).
	Duration int `json:"duration"`
}

// Item represents an item state for synchronization.
// Includes display name for client UI.
type Item struct {
	// ID is the unique item instance ID.
	ID string `json:"id"`

	// Type is the item type identifier (snake_case).
	// Values: "reverse_clock", "any_door", "dice_swap", "dice_upgrade"
	Type string `json:"type"`

	// Name is the item Chinese display name for client UI.
	// Values: "反方向的钟", "任意门", "骰子交换", "骰子升级卡"
	Name string `json:"name"`
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
// Sent to all players when entering mini-game phase.
type MiniGameStart struct {
	// GameType identifies the mini-game type for client to load.
	GameType string `json:"game_type"`

	// Players contains all participating player IDs.
	Players []string `json:"players"`

	// Connection contains MiniGame Service connection info (nil for frontend-driven mode).
	Connection *MiniGameConn `json:"connection,omitempty"`
}

// MiniGameConn represents connection info for real-time competitive mini-game service.
// Only present when MiniGame Service (Colyseus) is used; nil for frontend-driven mode.
type MiniGameConn struct {
	// URL is the MiniGame Service WebSocket URL.
	URL string `json:"url"`

	// RoomID is the Colyseus room ID.
	RoomID string `json:"room_id"`

	// Token is the authentication token for MiniGame Service.
	Token string `json:"token"`
}

// MiniGameDataSubmit represents client mini-game data submission.
// Client submits game_data (not rank); server calculates ranking using RankCalculator.
type MiniGameDataSubmit struct {
	// GameType identifies the mini-game type (must match MiniGameStart.GameType).
	GameType string `json:"game_type"`

	// GameData contains raw mini-game performance data (score, time, etc).
	GameData map[string]interface{} `json:"game_data"`
}

// MiniGameResult represents mini-game ranking result notification.
// Broadcast after mini-game completes.
type MiniGameResult struct {
	// Rankings contains player rankings (sorted by rank 1-4).
	Rankings []RankingEntry `json:"rankings"`
}

// RankingEntry represents a single player's mini-game ranking.
type RankingEntry struct {
	// PlayerID is the player's game internal ID.
	// Matches core.Player.ID.UUID() format.
	PlayerID string `json:"player_id"`

	// DisplayName is the user-provided display name for UI rendering.
	// Falls back to PlayerID (UUID) if not provided.
	DisplayName string `json:"display_name"`

	// Rank is the player's ranking (1-4).
	Rank int `json:"rank"`
}

// GameOver represents game end notification.
type GameOver struct {
	// WinnerID is the winning player's game internal ID.
	// Matches core.Player.ID.UUID() format.
	WinnerID string `json:"winner_id"`

	// Stats contains end-game statistics for all players.
	Stats []PlayerStats `json:"stats"`
}

// PlayerStats represents a player's end-game statistics.
type PlayerStats struct {
	// PlayerID is the player's game internal ID.
	// Matches core.Player.ID.UUID() format.
	PlayerID string `json:"player_id"`

	// RoundsWon is the number of rounds won (mini-game ranking 1).
	RoundsWon int `json:"rounds_won"`

	// EventsDrawn is the number of random events drawn.
	EventsDrawn int `json:"events_drawn"`

	// ItemsUsed is the number of items consumed.
	ItemsUsed int `json:"items_used"`
}

// FullSync represents complete sync data for reconnecting players.
type FullSync struct {
	// State is the current game state.
	State *StateSync `json:"state"`

	// Turn is the current turn's log entries.
	Turn *TurnSync `json:"turn"`
}

// ActionRejected notifies client that their action was rejected.
// Sent when client sends invalid request (wrong player, invalid state, etc).
type ActionRejected struct {
	// OpCode is the rejected operation code.
	OpCode OpCode `json:"op_code"`

	// ErrorCode is the standardized error code for client-side handling.
	// Defined in pkg/constants/error_code.go
	ErrorCode constants.ErrorCode `json:"error_code"`

	// Reason explains why the action was rejected.
	// Common values: "not_current_player", "invalid_state", "item_not_found", "skill_not_ready"
	Reason string `json:"reason"`

	// Message is a human-readable error message for debugging.
	Message string `json:"message"`
}

// WaitingSync represents waiting status before game starts.
// Broadcast to host when players join/leave in WaitingForHost state.
type WaitingSync struct {
	// MatchID is the Nakama match ID.
	MatchID string `json:"match_id"`

	// HostUserID is the user ID of the host (first player).
	HostUserID string `json:"host_user_id"`

	// Players contains list of joined players with their factions.
	Players []WaitingPlayer `json:"players"`

	// PlayerCount is the current number of players.
	PlayerCount int `json:"player_count"`

	// MinPlayers is the minimum players needed to start (2).
	MinPlayers int `json:"min_players"`

	// MaxPlayers is the maximum players allowed (4).
	MaxPlayers int `json:"max_players"`

	// CanStart indicates if game can be started (player_count >= min_players).
	CanStart bool `json:"can_start"`

	// Message is a human-readable status message.
	Message string `json:"message"`
}

// WaitingPlayer represents a player in the waiting room.
type WaitingPlayer struct {
	// UserID is the Nakama user ID.
	UserID string `json:"user_id"`

	// DisplayName is the user-provided display name for UI rendering.
	DisplayName string `json:"display_name"`

	// Faction is the player's chosen faction.
	Faction string `json:"faction"`

	// IsHost indicates if this player is the host.
	IsHost bool `json:"is_host"`
}

// MapConfig represents the complete map configuration for game initialization.
// This type is shared between pkg/net (protocol) and pkg/resource (loading).
// Contains full cell details including draw_type and probabilities needed for client rendering.
type MapConfig struct {
	// Length is the total number of cells in the map.
	Length int `json:"length"`

	// StartIndex is the starting position index.
	StartIndex int `json:"start_index"`

	// EndIndex is the ending position index (boss/bottom cell).
	EndIndex int `json:"end_index"`

	// Cells contains all cell configurations with full details.
	Cells []MapCellConfig `json:"cells"`
}

// MapCellConfig represents a single cell's complete configuration.
// Includes draw_type and probability fields needed for client-side rendering.
type MapCellConfig struct {
	// Index is the cell position (0 to Length-1).
	Index int `json:"index"`

	// CellType is the cell type identifier (snake_case).
	// Values: "normal", "fragile", "fog", "checkpoint", "boss", "event"
	CellType string `json:"cell_type"`

	// IsBroken indicates whether a Fragile cell is broken.
	IsBroken bool `json:"is_broken"`

	// EventID is the bound event ID for Event-type cells.
	EventID string `json:"event_id"`

	// FogActive indicates whether a Fog cell is currently active.
	FogActive bool `json:"fog_active"`

	// DrawType specifies what to draw when landing on this cell.
	// Values: "none", "event", "item"
	DrawType string `json:"draw_type"`

	// ProbGood is the probability weight for drawing a good outcome.
	ProbGood float64 `json:"prob_good"`

	// ProbNeutral is the probability weight for drawing a neutral outcome.
	ProbNeutral float64 `json:"prob_neutral"`

	// ProbBad is the probability weight for drawing a bad outcome.
	ProbBad float64 `json:"prob_bad"`
}

// StartGameAck represents the acknowledgment sent to all players when the host starts the game.
// Contains the full map configuration so clients can render the map before StateSync arrives.
type StartGameAck struct {
	// MapConfig is the complete map configuration for this game session.
	MapConfig MapConfig `json:"map_config"`
}

// NewLogEntry creates a simple log entry for protocol testing.
// For production use, use gamelog.NewActionEntry instead.
func NewLogEntry(actionType string, target string, source string) gamelog.LogEntry {
	return gamelog.LogEntry{
		Timestamp:  time.Now(),
		Type:       constants.EntryTypeAction,
		ActionType: actionType,
		Target:     target,
		Source:     source,
		Metadata:   nil,
	}
}