package engine

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/b1tAction/Fated/internal/core"
)

// FlowSnapshot represents a saved state of the turn flow for recovery.
type FlowSnapshot struct {
	GameID          string              `json:"game_id"`
	Round           int                 `json:"round"`
	Turn            int                 `json:"turn"`
	CurrentStep     TurnStep            `json:"current_step"`
	PlayerID        string              `json:"player_id"`
	WaitingDecisions []*DecisionSnapshot `json:"waiting_decisions"`
	PlayerSnapshots []*PlayerSnapshot   `json:"player_snapshots"`
	Timestamp       time.Time           `json:"timestamp"`
}

// DecisionSnapshot represents a saved decision state.
type DecisionSnapshot struct {
	DecisionID    string `json:"decision_id"`
	Prompt        string `json:"prompt"`
	SourceID      string `json:"source_id"`
	SourceType    string `json:"source_type"`
	DefaultChoice int    `json:"default_choice"`
}

// PlayerSnapshot represents a saved player state.
type PlayerSnapshot struct {
	UserID      string          `json:"user_id"`
	Faction     core.Faction    `json:"faction"`
	Position    int             `json:"position"`
	HP          int             `json:"hp"`
	LP          int             `json:"lp"`
	IsDead      bool            `json:"is_dead"`
	SkipTurn    bool            `json:"skip_turn"`
	Inventory   []*ItemSnapshot `json:"inventory"`
	ActiveBuffs []*BuffSnapshot `json:"active_buffs"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// ItemSnapshot represents a saved item state.
type ItemSnapshot struct {
	Type           core.ItemType `json:"type"`
	ID             string        `json:"id"`
	Usable         bool          `json:"usable"`
	TargetID       string        `json:"target_id"`
	SubscriptionID string        `json:"subscription_id"`
}

// BuffSnapshot represents a saved buff state.
type BuffSnapshot struct {
	Type            core.BuffType `json:"type"`
	ID              string        `json:"id"`
	Duration        int           `json:"duration"`
	Charge          int           `json:"charge"`
	SubscriptionIDs []string      `json:"subscription_ids"`
}

// NewFlowSnapshot creates a new flow snapshot.
func NewFlowSnapshot(gameID string) *FlowSnapshot {
	return &FlowSnapshot{
		GameID:          gameID,
		WaitingDecisions: make([]*DecisionSnapshot, 0),
		PlayerSnapshots: make([]*PlayerSnapshot, 0),
		Timestamp:       time.Now(),
	}
}

// ToJSON serializes snapshot to JSON.
func (fs *FlowSnapshot) ToJSON() ([]byte, error) {
	return json.Marshal(fs)
}

// FromJSON deserializes snapshot from JSON.
func (fs *FlowSnapshot) FromJSON(data []byte) error {
	return json.Unmarshal(data, fs)
}

// CreatePlayerSnapshot creates a snapshot of player state.
func CreatePlayerSnapshot(player *core.Player) *PlayerSnapshot {
	snapshot := &PlayerSnapshot{
		UserID:   player.UserID,
		Faction:  player.Faction,
		Position: player.Position,
		HP:       player.HP,
		LP:       player.LP,
		IsDead:   player.IsDead,
		SkipTurn: player.SkipTurn,
		Inventory: make([]*ItemSnapshot, 0, len(player.Inventory)),
		ActiveBuffs: make([]*BuffSnapshot, 0, len(player.ActiveBuffs)),
		Metadata:    player.Metadata.ToMap(),
	}

	for _, item := range player.Inventory {
		snapshot.Inventory = append(snapshot.Inventory, &ItemSnapshot{
			Type:           item.Type,
			ID:             item.ID,
			Usable:         item.Usable,
			TargetID:       item.TargetID,
			SubscriptionID: item.SubscriptionID,
		})
	}

	for _, buff := range player.ActiveBuffs {
		snapshot.ActiveBuffs = append(snapshot.ActiveBuffs, &BuffSnapshot{
			Type:            buff.Type,
			ID:              buff.ID,
			Duration:        buff.Duration,
			Charge:          buff.Charge,
			SubscriptionIDs: buff.SubscriptionIDs,
		})
	}

	return snapshot
}

// RestorePlayer restores player state from snapshot.
func RestorePlayer(snapshot *PlayerSnapshot) *core.Player {
	player := &core.Player{
		UserID:      snapshot.UserID,
		Faction:     snapshot.Faction,
		Position:    snapshot.Position,
		HP:          snapshot.HP,
		LP:          snapshot.LP,
		IsDead:      snapshot.IsDead,
		SkipTurn:    snapshot.SkipTurn,
		Inventory:   make([]*core.Item, 0, len(snapshot.Inventory)),
		ActiveBuffs: make([]*core.Buff, 0, len(snapshot.ActiveBuffs)),
	}

	for _, itemSnap := range snapshot.Inventory {
		player.Inventory = append(player.Inventory, &core.Item{
			Type:           itemSnap.Type,
			ID:             itemSnap.ID,
			Usable:         itemSnap.Usable,
			TargetID:       itemSnap.TargetID,
			SubscriptionID: itemSnap.SubscriptionID,
		})
	}

	for _, buffSnap := range snapshot.ActiveBuffs {
		player.ActiveBuffs = append(player.ActiveBuffs, &core.Buff{
			Type:            buffSnap.Type,
			ID:              buffSnap.ID,
			Duration:        buffSnap.Duration,
			Charge:          buffSnap.Charge,
			SubscriptionIDs: buffSnap.SubscriptionIDs,
		})
	}

	return player
}

// SnapshotManager manages flow snapshots for save/load operations.
type SnapshotManager struct {
	snapshots map[string]*FlowSnapshot // gameID -> latest snapshot
}

// NewSnapshotManager creates a new snapshot manager.
func NewSnapshotManager() *SnapshotManager {
	return &SnapshotManager{
		snapshots: make(map[string]*FlowSnapshot),
	}
}

// Save saves a snapshot for a game.
func (sm *SnapshotManager) Save(snapshot *FlowSnapshot) error {
	if snapshot == nil {
		return errors.New("snapshot is nil")
	}
	if snapshot.GameID == "" {
		return errors.New("game id is empty")
	}
	sm.snapshots[snapshot.GameID] = snapshot
	return nil
}

// Load loads the latest snapshot for a game.
func (sm *SnapshotManager) Load(gameID string) (*FlowSnapshot, error) {
	snapshot, ok := sm.snapshots[gameID]
	if !ok {
		return nil, errors.New("snapshot not found")
	}
	return snapshot, nil
}

// Delete removes a snapshot for a game.
func (sm *SnapshotManager) Delete(gameID string) {
	delete(sm.snapshots, gameID)
}

// List returns all game IDs with saved snapshots.
func (sm *SnapshotManager) List() []string {
	ids := make([]string, 0, len(sm.snapshots))
	for id := range sm.snapshots {
		ids = append(ids, id)
	}
	return ids
}

// HasSnapshot checks if a game has a saved snapshot.
func (sm *SnapshotManager) HasSnapshot(gameID string) bool {
	_, ok := sm.snapshots[gameID]
	return ok
}