// Package net provides network message protocol definitions for client-server communication.
package net

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/b1tAction/paradiced/pkg/constants"
	"github.com/b1tAction/paradiced/pkg/gamelog"
	"github.com/b1tAction/paradiced/pkg/util"
)

func TestStateSyncJSON(t *testing.T) {
	sync := &StateSync{
		GlobalState:   "turn_loop",
		TurnState:     "main_action",
		CurrentPlayerID: "player-abc123",
		Round:         1,
		Turn:          0,
		Paused:        false,
		Players: []Player{
			{PlayerID: "player-abc123", Faction: "qing_long", Position: 10, HP: 6, LP: 5},
		},
	}

	jsonBytes, err := json.Marshal(sync)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	jsonStr := string(jsonBytes)

	// Verify field names are correct
	if !strings.Contains(jsonStr, `"global_state"`) {
		t.Error("JSON should contain 'global_state' field")
	}
	if !strings.Contains(jsonStr, `"turn_state"`) {
		t.Error("JSON should contain 'turn_state' field")
	}
	if !strings.Contains(jsonStr, `"current_player_id"`) {
		t.Error("JSON should contain 'current_player_id' field")
	}
	if !strings.Contains(jsonStr, `"players"`) {
		t.Error("JSON should contain 'players' field")
	}
}

func TestTurnSyncWithLogEntries(t *testing.T) {
	// Create LogEntries using gamelog
	entry1 := gamelog.NewActionEntry("damage", "player-abc123", "Cell_Fragile")

	entry2Meta := util.NewMetadata()
	entry2Meta.Set("path", []int{10, 11, 12, 13})
	entry2 := gamelog.NewActionEntryWithMetadata("move", "player-abc123", "DiceRoll", entry2Meta)

	sync := &TurnSync{
		Round:             1,
		Turn:              0,
		CurrentPlayerID:   "player-abc123",
		Entries:           []gamelog.LogEntry{entry1, entry2},
	}

	jsonBytes, err := json.Marshal(sync)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	var parsed TurnSync
	err = json.Unmarshal(jsonBytes, &parsed)
	if err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}
	if parsed.Round != 1 {
		t.Errorf("parsed.Round = %d, want 1", parsed.Round)
	}
	if len(parsed.Entries) != 2 {
		t.Errorf("len(parsed.Entries) = %d, want 2", len(parsed.Entries))
	}
	if parsed.Entries[0].ActionType != "damage" {
		t.Errorf("parsed.Entries[0].ActionType = %s, want damage", parsed.Entries[0].ActionType)
	}
	if parsed.Entries[1].ActionType != "move" {
		t.Errorf("parsed.Entries[1].ActionType = %s, want move", parsed.Entries[1].ActionType)
	}
}

func TestTurnSyncJSONFieldNames(t *testing.T) {
	sync := &TurnSync{
		Round:           1,
		Turn:            0,
		CurrentPlayerID: "player-abc123",
		Entries:         []gamelog.LogEntry{},
	}

	jsonBytes, err := json.Marshal(sync)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	jsonStr := string(jsonBytes)

	// Verify field names are correct (entries not actions)
	if !strings.Contains(jsonStr, `"entries"`) {
		t.Error("JSON should contain 'entries' field (not 'actions')")
	}
	if strings.Contains(jsonStr, `"actions"`) {
		t.Error("JSON should NOT contain 'actions' field (use 'entries' now)")
	}
}

func TestLogEntryMetadataSerialization(t *testing.T) {
	// Test that LogEntry.Metadata serializes correctly
	meta := util.NewMetadata()
	meta.SetString("buff_type", "divine")
	meta.SetInt("duration", 3)

	entry := gamelog.NewActionEntryWithMetadata("add_buff", "player-001", "Buff_Divine", meta)

	jsonBytes, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	jsonStr := string(jsonBytes)
	if !strings.Contains(jsonStr, `"metadata"`) {
		t.Error("JSON should contain 'metadata' field")
	}
	if !strings.Contains(jsonStr, `"buff_type":"divine"`) {
		t.Error("JSON metadata should contain buff_type: divine")
	}
	if !strings.Contains(jsonStr, `"duration":3`) {
		t.Error("JSON metadata should contain duration: 3")
	}

	// Deserialize and verify
	var parsed gamelog.LogEntry
	err = json.Unmarshal(jsonBytes, &parsed)
	if err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}
	if parsed.Metadata == nil {
		t.Fatal("parsed.Metadata should not be nil")
	}
	buffType := parsed.Metadata.GetStringOrDefault("buff_type", "")
	if buffType != "divine" {
		t.Errorf("buff_type = %s, want divine", buffType)
	}
	duration := parsed.Metadata.GetIntOrDefault("duration", 0)
	if duration != 3 {
		t.Errorf("duration = %d, want 3", duration)
	}
}

func TestLogEntryMetadataWithPath(t *testing.T) {
	// Test path array serialization
	meta := util.NewMetadata()
	meta.Set("path", []int{10, 11, 12, 13, 14, 15})

	entry := gamelog.NewActionEntryWithMetadata("move", "player-001", "DiceRoll", meta)

	jsonBytes, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	jsonStr := string(jsonBytes)
	if !strings.Contains(jsonStr, `"path":[10,11,12,13,14,15]`) {
		t.Errorf("JSON should contain path array, got: %s", jsonStr)
	}
}

func TestLogEntryMetadataOmitempty(t *testing.T) {
	// Create entry with nil metadata (not using NewActionEntry)
	entry := gamelog.LogEntry{
		Type:       constants.EntryTypeAction,
		ActionType: "damage",
		Target:     "player-001",
		Source:     "Cell_Fragile",
		Metadata:   nil, // Explicitly nil
	}

	jsonBytes, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	jsonStr := string(jsonBytes)
	if strings.Contains(jsonStr, `"metadata"`) {
		t.Error("JSON should NOT contain 'metadata' field when nil (omitempty)")
	}
}

func TestLogEntryMetadataEmptyMap(t *testing.T) {
	// NewActionEntry creates an empty but non-nil Metadata
	// This will serialize as "metadata":{} not omitted
	entry := gamelog.NewActionEntry("damage", "player-001", "Cell_Fragile")

	jsonBytes, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	jsonStr := string(jsonBytes)
	// Empty metadata will serialize as "metadata":{} (not omitted because pointer is non-nil)
	if !strings.Contains(jsonStr, `"metadata":{}`) {
		t.Errorf("JSON should contain 'metadata':{} for empty non-nil Metadata, got: %s", jsonStr)
	}
}

func TestBuffWithName(t *testing.T) {
	buff := Buff{Type: "divine", Name: "神眷", Duration: 3}
	jsonBytes, err := json.Marshal(buff)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	jsonStr := string(jsonBytes)
	if !strings.Contains(jsonStr, `"type":"divine"`) {
		t.Error("JSON should contain type: divine")
	}
	if !strings.Contains(jsonStr, `"name":"神眷"`) {
		t.Error("JSON should contain name: 神眷")
	}
}

func TestBuffPermanent(t *testing.T) {
	buff := Buff{Type: "fire", Name: "离火", Duration: -1}
	jsonBytes, err := json.Marshal(buff)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	jsonStr := string(jsonBytes)
	if !strings.Contains(jsonStr, `"duration":-1`) {
		t.Error("JSON should contain duration: -1 for permanent buff")
	}
}

func TestItemWithName(t *testing.T) {
	item := Item{ID: "item-abc", Type: "reverse_clock", Name: "反方向的钟"}
	jsonBytes, err := json.Marshal(item)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	jsonStr := string(jsonBytes)
	if !strings.Contains(jsonStr, `"id":"item-abc"`) {
		t.Error("JSON should contain id: item-abc")
	}
	if !strings.Contains(jsonStr, `"name":"反方向的钟"`) {
		t.Error("JSON should contain name: 反方向的钟")
	}
}

func TestPlayerWithBuffsAndItems(t *testing.T) {
	player := Player{
		PlayerID:    "player-001",
		Faction:     "zhu_que",
		Position:    25,
		HP:          6,
		LP:          7,
		Buffs:       []Buff{{Type: "fire", Name: "离火", Duration: -1}},
		Items:       []Item{{ID: "item-001", Type: "any_door", Name: "任意门"}},
		FireCounter: 3,
	}

	jsonBytes, err := json.Marshal(player)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	var parsed Player
	err = json.Unmarshal(jsonBytes, &parsed)
	if err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}
	if parsed.PlayerID != "player-001" {
		t.Errorf("parsed.PlayerID = %s, want player-001", parsed.PlayerID)
	}
	if len(parsed.Buffs) != 1 {
		t.Errorf("len(parsed.Buffs) = %d, want 1", len(parsed.Buffs))
	}
	if parsed.Buffs[0].Name != "离火" {
		t.Errorf("parsed.Buffs[0].Name = %s, want 离火", parsed.Buffs[0].Name)
	}
	if len(parsed.Items) != 1 {
		t.Errorf("len(parsed.Items) = %d, want 1", len(parsed.Items))
	}
	if parsed.Items[0].Name != "任意门" {
		t.Errorf("parsed.Items[0].Name = %s, want 任意门", parsed.Items[0].Name)
	}
	if parsed.FireCounter != 3 {
		t.Errorf("parsed.FireCounter = %d, want 3", parsed.FireCounter)
	}
}

func TestMiniGameStartWithPlayers(t *testing.T) {
	start := MiniGameStart{
		GameType: "dice_race",
		Players:  []string{"player-001", "player-002", "player-003", "player-004"},
	}

	jsonBytes, err := json.Marshal(start)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	var parsed MiniGameStart
	err = json.Unmarshal(jsonBytes, &parsed)
	if err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}
	if parsed.GameType != "dice_race" {
		t.Errorf("parsed.GameType = %s, want dice_race", parsed.GameType)
	}
	if len(parsed.Players) != 4 {
		t.Errorf("len(parsed.Players) = %d, want 4", len(parsed.Players))
	}
}

func TestMiniGameResult(t *testing.T) {
	result := MiniGameResult{
		Rankings: []RankingEntry{
			{PlayerID: "player-001", Rank: 1},
			{PlayerID: "player-002", Rank: 2},
		},
	}

	jsonBytes, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	var parsed MiniGameResult
	err = json.Unmarshal(jsonBytes, &parsed)
	if err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}
	if len(parsed.Rankings) != 2 {
		t.Errorf("len(parsed.Rankings) = %d, want 2", len(parsed.Rankings))
	}
	if parsed.Rankings[0].Rank != 1 {
		t.Errorf("parsed.Rankings[0].Rank = %d, want 1", parsed.Rankings[0].Rank)
	}
}

func TestGameOver(t *testing.T) {
	over := GameOver{
		WinnerID: "player-001",
		Stats: []PlayerStats{
			{PlayerID: "player-001", RoundsWon: 3, EventsDrawn: 5, ItemsUsed: 2},
		},
	}

	jsonBytes, err := json.Marshal(over)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	var parsed GameOver
	err = json.Unmarshal(jsonBytes, &parsed)
	if err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}
	if parsed.WinnerID != "player-001" {
		t.Errorf("parsed.WinnerID = %s, want player-001", parsed.WinnerID)
	}
}

func TestAvailableWithItems(t *testing.T) {
	available := Available{
		Items: []Item{
			{ID: "item-001", Type: "reverse_clock", Name: "反方向的钟"},
		},
		CanUseSkill: false,
		DiceType:    "gold",
	}

	jsonBytes, err := json.Marshal(available)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	var parsed Available
	err = json.Unmarshal(jsonBytes, &parsed)
	if err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}
	if len(parsed.Items) != 1 {
		t.Errorf("len(parsed.Items) = %d, want 1", len(parsed.Items))
	}
	if parsed.DiceType != "gold" {
		t.Errorf("parsed.DiceType = %s, want gold", parsed.DiceType)
	}
}

func TestFullSync(t *testing.T) {
	stateSync := &StateSync{
		GlobalState:     "turn_loop",
		TurnState:       "main_action",
		CurrentPlayerID: "player-001",
		Round:           1,
		Turn:            0,
		Players:         []Player{{PlayerID: "player-001", Faction: "qing_long"}},
	}

	turnSync := &TurnSync{
		Round:             1,
		Turn:              0,
		CurrentPlayerID:   "player-001",
		Entries:           []gamelog.LogEntry{gamelog.NewActionEntry("damage", "player-001", "Test")},
	}

	fullSync := &FullSync{
		State: stateSync,
		Turn:  turnSync,
	}

	jsonBytes, err := json.Marshal(fullSync)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	var parsed FullSync
	err = json.Unmarshal(jsonBytes, &parsed)
	if err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}
	if parsed.State.GlobalState != "turn_loop" {
		t.Errorf("parsed.State.GlobalState = %s, want turn_loop", parsed.State.GlobalState)
	}
	if parsed.Turn.Round != 1 {
		t.Errorf("parsed.Turn.Round = %d, want 1", parsed.Turn.Round)
	}
	if len(parsed.Turn.Entries) != 1 {
		t.Errorf("len(parsed.Turn.Entries) = %d, want 1", len(parsed.Turn.Entries))
	}
}

func TestNewLogEntry(t *testing.T) {
	entry := NewLogEntry("heal", "player-001", "Event_Herb")

	if entry.ActionType != "heal" {
		t.Errorf("entry.ActionType = %s, want heal", entry.ActionType)
	}
	if entry.Target != "player-001" {
		t.Errorf("entry.Target = %s, want player-001", entry.Target)
	}
	if entry.Source != "Event_Herb" {
		t.Errorf("entry.Source = %s, want Event_Herb", entry.Source)
	}
	if entry.Timestamp.IsZero() || entry.Timestamp.After(time.Now()) {
		t.Errorf("entry.Timestamp should be valid and not future")
	}
}

func TestActionRejectedWithErrorCode(t *testing.T) {
	rejected := &ActionRejected{
		OpCode:    OpRollDice,
		ErrorCode: constants.ErrPlayerNotFound,
		Reason:    "player_not_found",
		Message:   "Unknown player",
	}

	jsonBytes, err := json.Marshal(rejected)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	jsonStr := string(jsonBytes)

	// Verify all fields are present
	if !strings.Contains(jsonStr, `"op_code"`) {
		t.Error("JSON should contain 'op_code' field")
	}
	if !strings.Contains(jsonStr, `"error_code"`) {
		t.Error("JSON should contain 'error_code' field")
	}
	if !strings.Contains(jsonStr, `"reason"`) {
		t.Error("JSON should contain 'reason' field")
	}
	if !strings.Contains(jsonStr, `"message"`) {
		t.Error("JSON should contain 'message' field")
	}
	if !strings.Contains(jsonStr, `"error_code":4001`) {
		t.Errorf("JSON should contain error_code value 4001, got: %s", jsonStr)
	}

	// Deserialize and verify
	var parsed ActionRejected
	err = json.Unmarshal(jsonBytes, &parsed)
	if err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}
	if parsed.OpCode != OpRollDice {
		t.Errorf("parsed.OpCode = %v, want OpRollDice", parsed.OpCode)
	}
	if parsed.ErrorCode != constants.ErrPlayerNotFound {
		t.Errorf("parsed.ErrorCode = %v, want ErrPlayerNotFound", parsed.ErrorCode)
	}
	if parsed.Reason != "player_not_found" {
		t.Errorf("parsed.Reason = %s, want player_not_found", parsed.Reason)
	}
	if parsed.Message != "Unknown player" {
		t.Errorf("parsed.Message = %s, want Unknown player", parsed.Message)
	}
}

func TestActionRejectedDifferentErrorCodes(t *testing.T) {
	tests := []struct {
		name        string
		errorCode   constants.ErrorCode
		expectedVal int
	}{
		{"ErrOK", constants.ErrOK, 0},
		{"ErrInvalidParameter", constants.ErrInvalidParameter, 1001},
		{"ErrInvalidState", constants.ErrInvalidState, 1002},
		{"ErrNotCurrentTurn", constants.ErrNotCurrentTurn, 1004},
		{"ErrPlayerNotFound", constants.ErrPlayerNotFound, 4001},
		{"ErrItemNotFound", constants.ErrItemNotFound, 4002},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rejected := &ActionRejected{
				OpCode:    OpUseItem,
				ErrorCode: tt.errorCode,
				Reason:    tt.errorCode.ToReason(),
				Message:   "test message",
			}

			jsonBytes, err := json.Marshal(rejected)
			if err != nil {
				t.Fatalf("json.Marshal() error: %v", err)
			}

			expectedCode := string(rune(tt.expectedVal))
			_ = expectedCode // avoid unused variable warning

			var parsed ActionRejected
			err = json.Unmarshal(jsonBytes, &parsed)
			if err != nil {
				t.Fatalf("json.Unmarshal() error: %v", err)
			}
			if parsed.ErrorCode != tt.errorCode {
				t.Errorf("parsed.ErrorCode = %v, want %v", parsed.ErrorCode, tt.errorCode)
			}
		})
	}
}
