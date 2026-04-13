package engine

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/b1tAction/Fated/internal/core"
)

// SyncType represents the type of sync message.
type SyncType int

const (
	SyncTypeFull SyncType = iota  // Complete game state
	SyncTypeDelta                  // Changes since last sync
	SyncTypeEvent                  // Event notification
	SyncTypeDecision               // Decision request
)

// String returns the string representation of SyncType.
func (st SyncType) String() string {
	names := map[SyncType]string{
		SyncTypeFull:     "FullSync",
		SyncTypeDelta:    "DeltaSync",
		SyncTypeEvent:    "EventSync",
		SyncTypeDecision: "DecisionSync",
	}
	if name, ok := names[st]; ok {
		return name
	}
	return "Unknown"
}

// SyncMessage represents a sync message to be sent to clients.
type SyncMessage struct {
	Type      SyncType   `json:"type"`
	Data      []byte     `json:"data"`      // JSON encoded payload
	Timestamp time.Time  `json:"timestamp"`
	GameID    string     `json:"game_id"`
	TargetID  string     `json:"target_id"` // Target player (empty for broadcast)
}

// SyncCheckpoint represents a checkpoint for delta sync.
type SyncCheckpoint struct {
	Round           int            `json:"round"`
	Turn            int            `json:"turn"`
	PlayerPositions map[string]int `json:"player_positions"`
	PlayerHPs       map[string]int `json:"player_hps"`
	PlayerLPs       map[string]int `json:"player_lps"`
	PlayerBuffs     map[string][]string `json:"player_buffs"` // playerID -> buff types
	PlayerItems     map[string]int `json:"player_items"` // playerID -> item count
}

// FullSyncPayload represents full sync data.
type FullSyncPayload struct {
	GameID    string             `json:"game_id"`
	Round     int                `json:"round"`
	Turn      int                `json:"turn"`
	Phase     string             `json:"phase"`
	Players   []*PlayerSyncData  `json:"players"`
	Waiting   bool               `json:"waiting"`
}

// PlayerSyncData represents player data for sync.
type PlayerSyncData struct {
	UserID      string        `json:"user_id"`
	Faction     string        `json:"faction"`
	Position    int           `json:"position"`
	HP          int           `json:"hp"`
	LP          int           `json:"lp"`
	IsDead      bool          `json:"is_dead"`
	SkipTurn    bool          `json:"skip_turn"`
	ActiveBuffs []*BuffSyncData `json:"active_buffs"`
	Inventory   []*ItemSyncData `json:"inventory"`
}

// BuffSyncData represents buff data for sync.
type BuffSyncData struct {
	Type     string `json:"type"`
	Duration int    `json:"duration"`
}

// ItemSyncData represents item data for sync.
type ItemSyncData struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

// DeltaSyncPayload represents delta sync data.
type DeltaSyncPayload struct {
	GameID         string                    `json:"game_id"`
	ChangedPlayers map[string]*PlayerSyncData `json:"changed_players"` // playerID -> changes
	NewRound       int                       `json:"new_round,omitempty"`
	NewTurn        int                       `json:"new_turn,omitempty"`
	PhaseChanged   bool                      `json:"phase_changed,omitempty"`
	NewPhase       string                    `json:"new_phase,omitempty"`
}

// EventSyncPayload represents event notification data.
type EventSyncPayload struct {
	GameID     string `json:"game_id"`
	EventType  string `json:"event_type"`
	PlayerID   string `json:"player_id"`
	EventName  string `json:"event_name"`
	EventDesc  string `json:"event_desc"`
	HPChange   int    `json:"hp_change,omitempty"`
	LPChange   int    `json:"lp_change,omitempty"`
	BuffApplied string `json:"buff_applied,omitempty"`
	ItemGained string `json:"item_gained,omitempty"`
}

// DecisionSyncPayload represents decision request data.
type DecisionSyncPayload struct {
	GameID      string           `json:"game_id"`
	DecisionID  string           `json:"decision_id"`
	PlayerID    string           `json:"player_id"`
	Prompt      string           `json:"prompt"`
	Options     []*OptionSyncData `json:"options"`
	Timeout     int              `json:"timeout"` // seconds, 0 for no timeout
}

// OptionSyncData represents option data for sync.
type OptionSyncData struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// StateSync manages state synchronization to clients.
type StateSync struct {
	Game      *Game
	LastSync  *SyncCheckpoint
}

// NewStateSync creates a new state sync manager.
func NewStateSync(game *Game) *StateSync {
	return &StateSync{
		Game:     game,
		LastSync: nil,
	}
}

// SyncFull generates a full sync message.
func (ss *StateSync) SyncFull() (*SyncMessage, error) {
	payload := &FullSyncPayload{
		GameID:  ss.Game.ID,
		Round:   ss.Game.State.Round,
		Turn:    ss.Game.State.Turn,
		Phase:   ss.Game.State.CurrentPhase,
		Waiting: ss.Game.State.Waiting,
		Players: make([]*PlayerSyncData, 0, len(ss.Game.Players)),
	}

	for _, player := range ss.Game.Players {
		playerData := ss.createPlayerSyncData(player)
		payload.Players = append(payload.Players, playerData)
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	// Update checkpoint
	ss.LastSync = ss.createCheckpoint()

	return &SyncMessage{
		Type:      SyncTypeFull,
		Data:      data,
		Timestamp: time.Now(),
		GameID:    ss.Game.ID,
	}, nil
}

// SyncDelta generates a delta sync message (changes since last sync).
func (ss *StateSync) SyncDelta() (*SyncMessage, error) {
	if ss.LastSync == nil {
		// No previous sync, do full sync instead
		return ss.SyncFull()
	}

	payload := &DeltaSyncPayload{
		GameID:         ss.Game.ID,
		ChangedPlayers: make(map[string]*PlayerSyncData),
	}

	// Check for round/turn changes
	if ss.Game.State.Round != ss.LastSync.Round {
		payload.NewRound = ss.Game.State.Round
	}
	if ss.Game.State.Turn != ss.LastSync.Turn {
		payload.NewTurn = ss.Game.State.Turn
	}

	// Check for player changes
	for _, player := range ss.Game.Players {
		changed := false

		// Check position
		if lastPos, ok := ss.LastSync.PlayerPositions[player.UserID]; ok {
			if player.Position != lastPos {
				changed = true
			}
		} else {
			changed = true
		}

		// Check HP
		if lastHP, ok := ss.LastSync.PlayerHPs[player.UserID]; ok {
			if player.HP != lastHP {
				changed = true
			}
		} else {
			changed = true
		}

		// Check LP
		if lastLP, ok := ss.LastSync.PlayerLPs[player.UserID]; ok {
			if player.LP != lastLP {
				changed = true
			}
		} else {
			changed = true
		}

		// Check buffs
		if lastBuffs, ok := ss.LastSync.PlayerBuffs[player.UserID]; ok {
			currentBuffTypes := make([]string, 0, len(player.ActiveBuffs))
			for _, buff := range player.ActiveBuffs {
				currentBuffTypes = append(currentBuffTypes, buff.Type.String())
			}
			if len(currentBuffTypes) != len(lastBuffs) {
				changed = true
			}
		} else {
			changed = true
		}

		if changed {
			payload.ChangedPlayers[player.UserID] = ss.createPlayerSyncData(player)
		}
	}

	// If no changes, return nil
	if len(payload.ChangedPlayers) == 0 && payload.NewRound == 0 && payload.NewTurn == 0 {
		return nil, nil
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	// Update checkpoint
	ss.LastSync = ss.createCheckpoint()

	return &SyncMessage{
		Type:      SyncTypeDelta,
		Data:      data,
		Timestamp: time.Now(),
		GameID:    ss.Game.ID,
	}, nil
}

// SyncEvent generates an event notification message.
func (ss *StateSync) SyncEvent(eventType string, playerID string, data map[string]interface{}) (*SyncMessage, error) {
	payload := &EventSyncPayload{
		GameID:    ss.Game.ID,
		EventType: eventType,
		PlayerID:  playerID,
	}

	// Populate payload based on data
	if eventName, ok := data["event_name"].(string); ok {
		payload.EventName = eventName
	}
	if eventDesc, ok := data["event_desc"].(string); ok {
		payload.EventDesc = eventDesc
	}
	if hpChange, ok := data["hp_change"].(int); ok {
		payload.HPChange = hpChange
	}
	if lpChange, ok := data["lp_change"].(int); ok {
		payload.LPChange = lpChange
	}
	if buffApplied, ok := data["buff_applied"].(string); ok {
		payload.BuffApplied = buffApplied
	}
	if itemGained, ok := data["item_gained"].(string); ok {
		payload.ItemGained = itemGained
	}

	eventData, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	return &SyncMessage{
		Type:      SyncTypeEvent,
		Data:      eventData,
		Timestamp: time.Now(),
		GameID:    ss.Game.ID,
		TargetID:  playerID,
	}, nil
}

// SyncDecision generates a decision request message.
func (ss *StateSync) SyncDecision(decisionID string, playerID string, prompt string, options []string, timeout int) (*SyncMessage, error) {
	payload := &DecisionSyncPayload{
		GameID:     ss.Game.ID,
		DecisionID: decisionID,
		PlayerID:   playerID,
		Prompt:     prompt,
		Options:    make([]*OptionSyncData, 0, len(options)),
		Timeout:    timeout,
	}

	for i, opt := range options {
		payload.Options = append(payload.Options, &OptionSyncData{
			ID:    fmt.Sprintf("%d", i),
			Label: opt,
		})
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	return &SyncMessage{
		Type:      SyncTypeDecision,
		Data:      data,
		Timestamp: time.Now(),
		GameID:    ss.Game.ID,
		TargetID:  playerID,
	}, nil
}

// createPlayerSyncData creates sync data for a player.
func (ss *StateSync) createPlayerSyncData(player *core.Player) *PlayerSyncData {
	data := &PlayerSyncData{
		UserID:   player.UserID,
		Faction:  player.Faction.String(),
		Position: player.Position,
		HP:       player.HP,
		LP:       player.LP,
		IsDead:   player.IsDead,
		SkipTurn: player.SkipTurn,
		ActiveBuffs: make([]*BuffSyncData, 0, len(player.ActiveBuffs)),
		Inventory:   make([]*ItemSyncData, 0, len(player.Inventory)),
	}

	for _, buff := range player.ActiveBuffs {
		data.ActiveBuffs = append(data.ActiveBuffs, &BuffSyncData{
			Type:     buff.Type.String(),
			Duration: buff.Duration,
		})
	}

	for _, item := range player.Inventory {
		data.Inventory = append(data.Inventory, &ItemSyncData{
			Type: item.Type.String(),
			ID:   item.ID,
		})
	}

	return data
}

// createCheckpoint creates a checkpoint for delta comparison.
func (ss *StateSync) createCheckpoint() *SyncCheckpoint {
	checkpoint := &SyncCheckpoint{
		Round:           ss.Game.State.Round,
		Turn:            ss.Game.State.Turn,
		PlayerPositions: make(map[string]int),
		PlayerHPs:       make(map[string]int),
		PlayerLPs:       make(map[string]int),
		PlayerBuffs:     make(map[string][]string),
		PlayerItems:     make(map[string]int),
	}

	for _, player := range ss.Game.Players {
		checkpoint.PlayerPositions[player.UserID] = player.Position
		checkpoint.PlayerHPs[player.UserID] = player.HP
		checkpoint.PlayerLPs[player.UserID] = player.LP

		buffTypes := make([]string, 0, len(player.ActiveBuffs))
		for _, buff := range player.ActiveBuffs {
			buffTypes = append(buffTypes, buff.Type.String())
		}
		checkpoint.PlayerBuffs[player.UserID] = buffTypes
		checkpoint.PlayerItems[player.UserID] = len(player.Inventory)
	}

	return checkpoint
}

// ResetCheckpoint resets the checkpoint (forces full sync on next delta).
func (ss *StateSync) ResetCheckpoint() {
	ss.LastSync = nil
}

// GetLastSyncTime returns the timestamp of last checkpoint.
func (ss *StateSync) GetLastSyncTime() time.Time {
	if ss.LastSync == nil {
		return time.Time{}
	}
	return time.Time{} // SyncCheckpoint doesn't store timestamp, return zero
}

// SyncAllPlayers broadcasts sync message to all players.
func (ss *StateSync) SyncAllPlayers() ([]*SyncMessage, error) {
	msg, err := ss.SyncFull()
	if err != nil {
		return nil, err
	}

	// For broadcast, create message for each player
	msgs := make([]*SyncMessage, 0, len(ss.Game.Players))
	for _, player := range ss.Game.Players {
		msgCopy := *msg
		msgCopy.TargetID = player.UserID
		msgs = append(msgs, &msgCopy)
	}

	return msgs, nil
}

// SyncPlayer sends sync message to specific player.
func (ss *StateSync) SyncPlayer(playerID string) (*SyncMessage, error) {
	player := ss.Game.GetPlayer(playerID)
	if player == nil {
		return nil, errors.New("player not found")
	}

	msg, err := ss.SyncFull()
	if err != nil {
		return nil, err
	}

	msg.TargetID = playerID
	return msg, nil
}