// Package model provides data models for CLI protocol handling.
package model

import (
	"encoding/json"
	"time"

	"github.com/b1tAction/paradiced/pkg/gamelog"
)

// Message is the base message structure for client-server communication.
type Message struct {
	OpCode    int64           `json:"op_code"`
	Timestamp int64           `json:"timestamp"`
	Data      json.RawMessage `json:"data"`
}

// StateSync represents complete game state for synchronization.
type StateSync struct {
	GlobalState     string   `json:"global_state"`
	TurnState       string   `json:"turn_state"`
	CurrentPlayerID string   `json:"current_player_id"` // 当前回合玩家 ID
	Round           int      `json:"round"`
	Turn            int      `json:"turn"`
	Paused          bool     `json:"paused"`
	Players         []Player `json:"players"`
	Map             MapInfo  `json:"map"`
}

// MapInfo represents map data for client synchronization.
type MapInfo struct {
	Length int        `json:"length"`
	Cells  []CellInfo `json:"cells"`
}

// CellInfo represents a single map cell for client synchronization.
type CellInfo struct {
	Index    int    `json:"index"`
	CellType string `json:"cell_type"`
	EventID  string `json:"event_id,omitempty"`
	IsBroken bool   `json:"is_broken,omitempty"`
}

// Player represents a player state snapshot.
type Player struct {
	PlayerID    string `json:"player_id"` // 玩家游戏 ID (直接等于前端 userID)
	DisplayName string `json:"display_name"` // 用户显示名称
	Faction     string `json:"faction"`
	Position    int    `json:"position"`
	HP          int    `json:"hp"`
	LP          int    `json:"lp"`
	Buffs       []Buff `json:"buffs"`
	Items       []Item `json:"items"`
	Charge      int    `json:"charge"`
	FireCounter int    `json:"fire_counter"`
	IsDead      bool   `json:"is_dead"`
	SkipTurn    bool   `json:"skip_turn"`
	IsBoss      bool   `json:"is_boss,omitempty"` // Boss player identification
}

// Buff represents a buff state for synchronization.
type Buff struct {
	Type     string `json:"type"`
	Name     string `json:"name"`
	Duration int    `json:"duration"`
}

// Item represents an item state for synchronization.
type Item struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Name string `json:"name"`
}

// TurnSync represents all events for a turn/phase.
type TurnSync struct {
	Round           int                `json:"round"`
	Turn            int                `json:"turn"`
	CurrentPlayerID string             `json:"current_player_id"` // 当前回合玩家 ID
	Entries         []gamelog.LogEntry `json:"entries"`
}

// Decision represents a decision request sent to client.
type Decision struct {
	ID      string   `json:"id"`
	Prompt  string   `json:"prompt"`
	Context string   `json:"context"`
	Options []Option `json:"options"`
	Timeout int      `json:"timeout"`
	Default int      `json:"default"`
}

// Option represents a decision choice option.
type Option struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Effect string `json:"effect,omitempty"`
}

// Available represents available actions for the current player.
type Available struct {
	Items       []Item `json:"items"`
	CanUseSkill bool   `json:"can_use_skill"`
	DiceType    string `json:"dice_type"`
}

// MiniGameStart represents mini-game start notification.
type MiniGameStart struct {
	GameType string   `json:"game_type"`
	Players  []string `json:"players"`
}

// MiniGameResult represents mini-game ranking result notification.
type MiniGameResult struct {
	Rankings []RankingEntry `json:"rankings"`
}

// RankingEntry represents a single player's mini-game ranking.
type RankingEntry struct {
	PlayerID    string `json:"player_id"`
	DisplayName string `json:"display_name"`
	Rank        int    `json:"rank"`
}

// GameOver represents game end notification.
type GameOver struct {
	WinnerID string        `json:"winner_id"`
	Stats    []PlayerStats `json:"stats"`
}

// PlayerStats represents a player's end-game statistics.
type PlayerStats struct {
	PlayerID    string `json:"player_id"`
	RoundsWon   int    `json:"rounds_won"`
	EventsDrawn int    `json:"events_drawn"`
	ItemsUsed   int    `json:"items_used"`
}

// RollDice represents a dice roll request (client -> server).
type RollDice struct{}

// RoundReady represents a round-ready signal (client -> server).
// Sent after client finishes rendering current round when in RoundEndWait state.
type RoundReady struct{}

// UseItem represents an item usage request.
type UseItem struct {
	ItemID   string `json:"item_id"`
	TargetID string `json:"target_id,omitempty"`
}

// UseSkill represents a faction skill activation request.
type UseSkill struct{}

// UserChoice represents a decision choice response.
type UserChoice struct {
	DecisionID string `json:"decision_id"`
	Choice     int    `json:"choice"`
}

// MiniGameResultSubmit represents mini-game result submission.
type MiniGameResultSubmit struct {
	Rank int `json:"rank"`
}

// ActionRejected represents an action rejection notification (server -> client).
type ActionRejected struct {
	OpCode    int64  `json:"op_code"`
	ErrorCode int    `json:"error_code"` // Standardized error code for client handling
	Reason    string `json:"reason"`
	Message   string `json:"message"`
}

// WaitingSync represents waiting room status before game starts.
type WaitingSync struct {
	MatchID     string          `json:"match_id"`
	HostUserID  string          `json:"host_user_id"`
	Players     []WaitingPlayer `json:"players"`
	PlayerCount int             `json:"player_count"`
	MinPlayers  int             `json:"min_players"`
	MaxPlayers  int             `json:"max_players"`
	CanStart    bool            `json:"can_start"`
	Message     string          `json:"message"`
}

// WaitingPlayer represents a player in the waiting room.
type WaitingPlayer struct {
	UserID      string `json:"user_id"`
	DisplayName string `json:"display_name"` // 用户显示名称
	Faction     string `json:"faction"`
	IsHost      bool   `json:"is_host"`
}

// StartGameAck represents game start acknowledgment with full map configuration.
type StartGameAck struct {
	MapConfig MapConfig `json:"map_config"`
}

// MapConfig represents the complete map configuration for game initialization.
type MapConfig struct {
	Length     int            `json:"length"`
	StartIndex int            `json:"start_index"`
	EndIndex   int            `json:"end_index"`
	Cells      []MapCellConfig `json:"cells"`
}

// MapCellConfig represents a single cell's complete configuration.
type MapCellConfig struct {
	Index       int     `json:"index"`
	CellType    string  `json:"cell_type"`
	IsBroken    bool    `json:"is_broken"`
	EventID     string  `json:"event_id"`
	FogActive   bool    `json:"fog_active"`
	DrawType    string  `json:"draw_type"`
	ProbGood    float64 `json:"prob_good"`
	ProbNeutral float64 `json:"prob_neutral"`
	ProbBad     float64 `json:"prob_bad"`
}

// ParseMessage parses a JSON message into the Message struct.
func ParseMessage(data []byte) (*Message, error) {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

// ParseData parses the message data into the target structure.
func (m *Message) ParseData(target any) error {
	if len(m.Data) == 0 {
		return nil
	}
	return json.Unmarshal(m.Data, target)
}

// LogEntryWithTime extends LogEntry with formatted timestamp.
type LogEntryWithTime struct {
	gamelog.LogEntry
	Time time.Time `json:"time"`
}
