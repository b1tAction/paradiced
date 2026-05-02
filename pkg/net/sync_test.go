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
			{PlayerID: "player-abc123", Faction: "qing_long", Position: 10, HP: 6, MaxHP: 8, LP: 5},
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

func TestStateSyncWithLogEntries(t *testing.T) {
	// Create LogEntries using gamelog
	entry1 := gamelog.NewActionEntry("damage", "player-abc123", "Cell_Fragile")

	entry2Meta := util.NewMetadata()
	entry2Meta.Set("path", []int{10, 11, 12, 13})
	entry2 := gamelog.NewActionEntryWithMetadata("move", "player-abc123", "DiceRoll", entry2Meta)

	sync := &StateSync{
		GlobalState:     "turn_loop",
		TurnState:       "turn_end",
		CurrentPlayerID: "player-abc123",
		Round:           1,
		Turn:            0,
		Entries:         []gamelog.LogEntry{entry1, entry2},
	}

	jsonBytes, err := json.Marshal(sync)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	var parsed StateSync
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

func TestStateSyncEntriesJSONFieldNames(t *testing.T) {
	entry := gamelog.NewActionEntry("damage", "player-001", "Test")
	sync := &StateSync{
		GlobalState:     "turn_loop",
		TurnState:       "turn_end",
		CurrentPlayerID: "player-abc123",
		Round:           1,
		Turn:            0,
		Entries:         []gamelog.LogEntry{entry},
	}

	jsonBytes, err := json.Marshal(sync)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	jsonStr := string(jsonBytes)

	// Verify entries field name is correct when populated
	if !strings.Contains(jsonStr, `"entries"`) {
		t.Error("JSON should contain 'entries' field when populated")
	}
}

func TestStateSyncEntriesOmitempty(t *testing.T) {
	// StateSync with nil Entries - field should be omitted in JSON
	syncNil := &StateSync{
		GlobalState: "turn_loop",
		Round:       1,
		Turn:        0,
		// Entries is nil
	}

	jsonBytes, err := json.Marshal(syncNil)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	jsonStr := string(jsonBytes)
	if strings.Contains(jsonStr, `"entries"`) {
		t.Errorf("JSON should NOT contain 'entries' field when nil (omitempty), got: %s", jsonStr)
	}

	// StateSync with empty Entries slice - also omitted (omitempty)
	syncEmpty := &StateSync{
		GlobalState: "turn_loop",
		Round:       1,
		Turn:        0,
		Entries:     []gamelog.LogEntry{},
	}

	jsonBytes, err = json.Marshal(syncEmpty)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	jsonStr = string(jsonBytes)
	if strings.Contains(jsonStr, `"entries"`) {
		t.Errorf("JSON should NOT contain 'entries' field when empty slice (omitempty), got: %s", jsonStr)
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
		MaxHP:       8,
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
			{PlayerID: "player-001", DisplayName: "Alice", Rank: 1},
			{PlayerID: "player-002", DisplayName: "Bob", Rank: 2},
		},
	}

	jsonBytes, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	jsonStr := string(jsonBytes)
	if !strings.Contains(jsonStr, `"display_name":"Alice"`) {
		t.Errorf("JSON should contain display_name: Alice, got: %s", jsonStr)
	}
	if !strings.Contains(jsonStr, `"display_name":"Bob"`) {
		t.Errorf("JSON should contain display_name: Bob, got: %s", jsonStr)
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
	if parsed.Rankings[0].DisplayName != "Alice" {
		t.Errorf("parsed.Rankings[0].DisplayName = %s, want Alice", parsed.Rankings[0].DisplayName)
	}
	if parsed.Rankings[1].DisplayName != "Bob" {
		t.Errorf("parsed.Rankings[1].DisplayName = %s, want Bob", parsed.Rankings[1].DisplayName)
	}
}

func TestGameOver(t *testing.T) {
	over := GameOver{
		WinnerID:          "player-001",
		WinnerDisplayName: "Alice",
		Stats: []PlayerStats{
			{PlayerID: "player-001", DisplayName: "Alice", RoundsWon: 3, EventsDrawn: 5, ItemsUsed: 2},
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
	if parsed.WinnerDisplayName != "Alice" {
		t.Errorf("parsed.WinnerDisplayName = %s, want Alice", parsed.WinnerDisplayName)
	}
	if parsed.Stats[0].DisplayName != "Alice" {
		t.Errorf("parsed.Stats[0].DisplayName = %s, want Alice", parsed.Stats[0].DisplayName)
	}
}

func TestGameOverWithEmptyDisplayName(t *testing.T) {
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
	// WinnerID should remain intact (not overwritten)
	if parsed.WinnerID != "player-001" {
		t.Errorf("parsed.WinnerID = %s, want player-001", parsed.WinnerID)
	}
	// DisplayName fields should be empty strings (default)
	if parsed.WinnerDisplayName != "" {
		t.Errorf("parsed.WinnerDisplayName = %s, want empty string", parsed.WinnerDisplayName)
	}
	if parsed.Stats[0].DisplayName != "" {
		t.Errorf("parsed.Stats[0].DisplayName = %s, want empty string", parsed.Stats[0].DisplayName)
	}
	// PlayerID should remain intact
	if parsed.Stats[0].PlayerID != "player-001" {
		t.Errorf("parsed.Stats[0].PlayerID = %s, want player-001", parsed.Stats[0].PlayerID)
	}
}

func TestGameOverJSONContainsDisplayNameFields(t *testing.T) {
	over := GameOver{
		WinnerID:          "player-001",
		WinnerDisplayName: "Alice",
		Stats: []PlayerStats{
			{PlayerID: "player-001", DisplayName: "Alice", RoundsWon: 3, EventsDrawn: 5, ItemsUsed: 2},
			{PlayerID: "player-002", DisplayName: "Bob", RoundsWon: 1, EventsDrawn: 4, ItemsUsed: 1},
		},
	}

	jsonBytes, err := json.Marshal(over)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	// Verify JSON contains both ID and DisplayName fields
	jsonStr := string(jsonBytes)
	if !strings.Contains(jsonStr, `"winner_id":"player-001"`) {
		t.Error("JSON should contain winner_id field with UUID value")
	}
	if !strings.Contains(jsonStr, `"winner_display_name":"Alice"`) {
		t.Error("JSON should contain winner_display_name field")
	}
	if !strings.Contains(jsonStr, `"player_id":"player-001"`) {
		t.Error("JSON should contain player_id field with UUID value")
	}
	if !strings.Contains(jsonStr, `"display_name":"Alice"`) {
		t.Error("JSON should contain display_name field in PlayerStats")
	}
	// Verify PlayerID is NOT replaced with DisplayName
	if strings.Contains(jsonStr, `"player_id":"Alice"`) {
		t.Error("JSON should NOT have display_name as player_id value (UUID must be preserved)")
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

func TestPlayerIsBossField(t *testing.T) {
	// Normal player: IsBoss=false, omitempty means field not in JSON
	normalPlayer := Player{
		PlayerID: "player-001",
		Faction:  "qing_long",
		HP:       6,
		MaxHP:    8,
		LP:       5,
		IsBoss:   false,
	}

	jsonBytes, err := json.Marshal(normalPlayer)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}
	jsonStr := string(jsonBytes)
	if strings.Contains(jsonStr, "is_boss") {
		t.Errorf("Normal player JSON should not contain is_boss field (omitempty), got: %s", jsonStr)
	}

	// Boss player: IsBoss=true, should appear in JSON
	bossPlayer := Player{
		PlayerID:    "beeeeeef-beef-beef-beef-beeeeeeeeeef",
		Faction:     "",
		HP:          50,
		MaxHP:       50,
		LP:          0,
		IsBoss:      true,
		IsDead:      false,
	}

	jsonBytes, err = json.Marshal(bossPlayer)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}
	jsonStr = string(jsonBytes)
	if !strings.Contains(jsonStr, `"is_boss":true`) {
		t.Errorf("Boss player JSON should contain is_boss:true, got: %s", jsonStr)
	}

	// Roundtrip: Boss player unmarshals correctly
	var parsed Player
	err = json.Unmarshal(jsonBytes, &parsed)
	if err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}
	if !parsed.IsBoss {
		t.Errorf("parsed.IsBoss = false, want true")
	}
	if parsed.HP != 50 {
		t.Errorf("parsed.HP = %d, want 50", parsed.HP)
	}
	if parsed.PlayerID != "beeeeeef-beef-beef-beef-beeeeeeeeeef" {
		t.Errorf("parsed.PlayerID = %s, want beeeeeef-beef-beef-beef-beeeeeeeeeef", parsed.PlayerID)
	}

	// Roundtrip: Normal player with explicit is_boss:false in JSON
	jsonWithBossFalse := `{"player_id":"p1","faction":"zhu_que","hp":6,"lp":5,"is_boss":false}`
	var parsedNormal Player
	err = json.Unmarshal([]byte(jsonWithBossFalse), &parsedNormal)
	if err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}
	if parsedNormal.IsBoss {
		t.Errorf("parsedNormal.IsBoss = true, want false")
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

func TestMiniGameStartWithConnection(t *testing.T) {
	conn := &MiniGameConn{
		URL:    "ws://minigame.example.com",
		RoomID: "room-abc123",
		Token:  "token-xyz789",
	}
	start := MiniGameStart{
		GameType:   "coin_flip",
		Players:    []string{"p1", "p2"},
		Connection: conn,
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
	if parsed.GameType != "coin_flip" {
		t.Errorf("parsed.GameType = %s, want coin_flip", parsed.GameType)
	}
	if parsed.Connection == nil {
		t.Fatal("parsed.Connection should not be nil")
	}
	if parsed.Connection.URL != "ws://minigame.example.com" {
		t.Errorf("parsed.Connection.URL = %s, want ws://minigame.example.com", parsed.Connection.URL)
	}
	if parsed.Connection.RoomID != "room-abc123" {
		t.Errorf("parsed.Connection.RoomID = %s, want room-abc123", parsed.Connection.RoomID)
	}
}

func TestMiniGameStartWithoutConnection(t *testing.T) {
	start := MiniGameStart{
		GameType: "dice_race",
		Players:  []string{"p1", "p2"},
		// Connection is nil (frontend-driven mode)
	}

	jsonBytes, err := json.Marshal(start)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	// Verify connection field is omitted in JSON (omitempty)
	jsonStr := string(jsonBytes)
	if strings.Contains(jsonStr, "connection") {
		t.Errorf("JSON should not contain 'connection' field when nil, got: %s", jsonStr)
	}

	var parsed MiniGameStart
	err = json.Unmarshal(jsonBytes, &parsed)
	if err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}
	if parsed.Connection != nil {
		t.Error("parsed.Connection should be nil for frontend-driven mode")
	}
}

func TestMiniGameDataSubmitJSON(t *testing.T) {
	submit := MiniGameDataSubmit{
		GameType: "dice_race",
		GameData: map[string]interface{}{
			"score": 150.0,
			"time":  3.5,
		},
	}

	jsonBytes, err := json.Marshal(submit)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	var parsed MiniGameDataSubmit
	err = json.Unmarshal(jsonBytes, &parsed)
	if err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}
	if parsed.GameType != "dice_race" {
		t.Errorf("parsed.GameType = %s, want dice_race", parsed.GameType)
	}
	score, ok := parsed.GameData["score"]
	if !ok {
		t.Error("parsed.GameData should contain 'score' key")
	}
	if score != 150.0 {
		t.Errorf("parsed.GameData['score'] = %v, want 150.0", score)
	}
}

func TestMiniGameConnJSON(t *testing.T) {
	conn := MiniGameConn{
		URL:    "ws://minigame.example.com",
		RoomID: "room-abc123",
		Token:  "token-xyz789",
	}

	jsonBytes, err := json.Marshal(conn)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	var parsed MiniGameConn
	err = json.Unmarshal(jsonBytes, &parsed)
	if err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}
	if parsed.URL != "ws://minigame.example.com" {
		t.Errorf("parsed.URL = %s, want ws://minigame.example.com", parsed.URL)
	}
	if parsed.RoomID != "room-abc123" {
		t.Errorf("parsed.RoomID = %s, want room-abc123", parsed.RoomID)
	}
	if parsed.Token != "token-xyz789" {
		t.Errorf("parsed.Token = %s, want token-xyz789", parsed.Token)
	}
}

func TestRankingEntryWithGameData(t *testing.T) {
	entry := RankingEntry{
		PlayerID:    "player-001",
		DisplayName: "Alice",
		Rank:        1,
		GameData: map[string]interface{}{
			"dice1": 3,
			"dice2": 5,
			"score": 8,
		},
	}

	jsonBytes, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	jsonStr := string(jsonBytes)

	// Verify game_data field is present with correct content
	if !strings.Contains(jsonStr, `"game_data"`) {
		t.Error("JSON should contain 'game_data' field when GameData is populated")
	}
	if !strings.Contains(jsonStr, `"dice1"`) {
		t.Error("JSON game_data should contain 'dice1' key")
	}
	if !strings.Contains(jsonStr, `"score"`) {
		t.Error("JSON game_data should contain 'score' key")
	}
	if !strings.Contains(jsonStr, `"display_name":"Alice"`) {
		t.Error("JSON should contain display_name: Alice")
	}

	// Roundtrip: unmarshal and verify GameData content
	var parsed RankingEntry
	err = json.Unmarshal(jsonBytes, &parsed)
	if err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}
	if parsed.PlayerID != "player-001" {
		t.Errorf("parsed.PlayerID = %s, want player-001", parsed.PlayerID)
	}
	if parsed.DisplayName != "Alice" {
		t.Errorf("parsed.DisplayName = %s, want Alice", parsed.DisplayName)
	}
	if parsed.Rank != 1 {
		t.Errorf("parsed.Rank = %d, want 1", parsed.Rank)
	}
	if parsed.GameData == nil {
		t.Fatal("parsed.GameData should not be nil")
	}
	dice1, ok := parsed.GameData["dice1"]
	if !ok {
		t.Error("parsed.GameData should contain 'dice1' key")
	}
	// JSON numbers unmarshal as float64 by default
	if dice1 != float64(3) {
		t.Errorf("parsed.GameData['dice1'] = %v, want 3", dice1)
	}
	score, ok := parsed.GameData["score"]
	if !ok {
		t.Error("parsed.GameData should contain 'score' key")
	}
	if score != float64(8) {
		t.Errorf("parsed.GameData['score'] = %v, want 8", score)
	}
}

func TestRankingEntryGameDataOmitempty(t *testing.T) {
	// RankingEntry without GameData — field should be omitted in JSON
	entry := RankingEntry{
		PlayerID:    "player-001",
		DisplayName: "Bob",
		Rank:        2,
		GameData:    nil,
	}

	jsonBytes, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	jsonStr := string(jsonBytes)
	if strings.Contains(jsonStr, `"game_data"`) {
		t.Errorf("JSON should NOT contain 'game_data' field when nil (omitempty), got: %s", jsonStr)
	}

	// Roundtrip: unmarshal and verify GameData is nil
	var parsed RankingEntry
	err = json.Unmarshal(jsonBytes, &parsed)
	if err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}
	if parsed.GameData != nil {
		t.Errorf("parsed.GameData should be nil, got: %v", parsed.GameData)
	}
	if parsed.DisplayName != "Bob" {
		t.Errorf("parsed.DisplayName = %s, want Bob", parsed.DisplayName)
	}
}
