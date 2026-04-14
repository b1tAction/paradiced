package gamelog

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/b1tAction/Fated/pkg/util"
)

// ========== GameLog Tests ==========

func TestNewGameLog(t *testing.T) {
	log := NewGameLog()
	if log == nil {
		t.Fatal("NewGameLog returned nil")
	}
	if log.Len() != 0 {
		t.Errorf("New log should have 0 entries, got %d", log.Len())
	}
	if log.IsTurnActive() {
		t.Error("New log should not have active turn")
	}
}

func TestStartTurnEndTurn(t *testing.T) {
	log := NewGameLog()

	// Start turn
	log.StartTurn(1, 0, "player1")

	if !log.IsTurnActive() {
		t.Error("Turn should be active after StartTurn")
	}

	// Add entry during turn
	entry := NewActionEntry("Damage", "player1", -5, "Event_Trap")
	log.AddEntry(entry)

	entries := log.GetCurrentTurnEntries()
	if len(entries) != 1 {
		t.Errorf("Should have 1 entry, got %d", len(entries))
	}

	// End turn
	log.EndTurn()

	if log.IsTurnActive() {
		t.Error("Turn should not be active after EndTurn")
	}

	// Check segment
	segments := log.GetTurnSegments()
	if len(segments) != 1 {
		t.Errorf("Should have 1 segment, got %d", len(segments))
	}

	seg := segments[0]
	if seg.Round != 1 {
		t.Errorf("Round should be 1, got %d", seg.Round)
	}
	if seg.Turn != 0 {
		t.Errorf("Turn should be 0, got %d", seg.Turn)
	}
	if seg.PlayerID != "player1" {
		t.Errorf("PlayerID should be player1, got %s", seg.PlayerID)
	}
	if seg.EndTime.IsZero() {
		t.Error("EndTime should be set after EndTurn")
	}
}

func TestAddEntryWithoutActiveTurn(t *testing.T) {
	log := NewGameLog()

	// Add entry without starting turn - should be discarded
	entry := NewActionEntry("Damage", "player1", -5, "Event_Trap")
	log.AddEntry(entry)

	if log.Len() != 0 {
		t.Errorf("Entry should be discarded without active turn, got %d entries", log.Len())
	}
}

func TestMultipleTurns(t *testing.T) {
	log := NewGameLog()

	// Turn 1
	log.StartTurn(1, 0, "player1")
	metadata1 := util.NewMetadata().SetInt("steps", 5)
	entry1 := NewActionEntryWithMetadata("Move", "player1", 5, "DiceRoll", metadata1)
	log.AddEntry(entry1)
	log.EndTurn()

	// Turn 2
	log.StartTurn(1, 1, "player2")
	log.AddEntry(NewActionEntry("Damage", "player2", -3, "Event_Trap"))
	log.AddEntry(NewActionEntry("Heal", "player2", 2, "Buff_Rain"))
	log.EndTurn()

	// Check segments
	segments := log.GetTurnSegments()
	if len(segments) != 2 {
		t.Errorf("Should have 2 segments, got %d", len(segments))
	}

	// Check first segment
	if segments[0].Len() != 1 {
		t.Errorf("First segment should have 1 entry, got %d", segments[0].Len())
	}

	// Check second segment
	if segments[1].Len() != 2 {
		t.Errorf("Second segment should have 2 entries, got %d", segments[1].Len())
	}

	// Total entries
	if log.Len() != 3 {
		t.Errorf("Total entries should be 3, got %d", log.Len())
	}
}

func TestGetSegment(t *testing.T) {
	log := NewGameLog()

	log.StartTurn(1, 0, "player1")
	log.AddEntry(NewActionEntry("Move", "player1", 5, "DiceRoll"))
	log.EndTurn()

	log.StartTurn(2, 0, "player1")
	log.AddEntry(NewActionEntry("Damage", "player1", -2, "Event_Trap"))
	log.EndTurn()

	// Get segment for round 1, turn 0
	seg := log.GetSegment(1, 0)
	if seg == nil {
		t.Fatal("Segment (1,0) should exist")
	}
	if seg.Len() != 1 {
		t.Errorf("Segment (1,0) should have 1 entry, got %d", seg.Len())
	}

	// Get segment for round 2, turn 0
	seg = log.GetSegment(2, 0)
	if seg == nil {
		t.Fatal("Segment (2,0) should exist")
	}
	if seg.Len() != 1 {
		t.Errorf("Segment (2,0) should have 1 entry, got %d", seg.Len())
	}

	// Get non-existent segment
	seg = log.GetSegment(99, 99)
	if seg != nil {
		t.Error("Non-existent segment should return nil")
	}
}

func TestToJSON(t *testing.T) {
	log := NewGameLog()

	log.StartTurn(1, 0, "player1")
	metadata := util.NewMetadata().SetInt("steps", 5)
	entry := NewActionEntryWithMetadata("Move", "player1", 5, "DiceRoll", metadata)
	log.AddEntry(entry)
	log.EndTurn()

	data, err := log.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	// Parse JSON to verify structure
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}

	segments, ok := result["segments"].([]interface{})
	if !ok {
		t.Fatal("segments should be an array")
	}
	if len(segments) != 1 {
		t.Errorf("Should have 1 segment in JSON, got %d", len(segments))
	}
}

func TestClear(t *testing.T) {
	log := NewGameLog()

	log.StartTurn(1, 0, "player1")
	log.AddEntry(NewActionEntry("Move", "player1", 5, "DiceRoll"))
	log.EndTurn()

	if log.Len() != 1 {
		t.Errorf("Should have 1 entry before clear, got %d", log.Len())
	}

	log.Clear()

	if log.Len() != 0 {
		t.Errorf("Should have 0 entries after clear, got %d", log.Len())
	}
	if len(log.GetTurnSegments()) != 0 {
		t.Error("Segments should be empty after clear")
	}
}

func TestLogStateTransition(t *testing.T) {
	log := NewGameLog()

	log.StartTurn(1, 0, "player1")
	log.LogStateTransition("TurnUpkeep", "MainAction", "player1")
	log.EndTurn()

	entries := log.GetCurrentTurnEntries()
	// After EndTurn, current entries should be nil
	if entries != nil {
		t.Error("GetCurrentTurnEntries should return nil after EndTurn")
	}

	// Check segment entries
	seg := log.GetSegment(1, 0)
	if seg == nil {
		t.Fatal("Segment should exist")
	}
	if seg.Len() != 1 {
		t.Errorf("Should have 1 entry, got %d", seg.Len())
	}

	// Verify entry type
	entry := seg.Entries[0]
	if entry.Type != EntryTypeState {
		t.Errorf("Entry type should be state, got %s", entry.Type)
	}

	// Verify metadata using type-safe access
	if entry.Metadata == nil {
		t.Fatal("Metadata should not be nil")
	}
	from := entry.Metadata.GetStringOrDefault("from", "")
	if from != "TurnUpkeep" {
		t.Errorf("from should be TurnUpkeep, got %s", from)
	}
}

// ========== LogEntry Tests ==========

func TestNewActionEntry(t *testing.T) {
	metadata := util.NewMetadata().SetBool("piercing", true)
	entry := NewActionEntryWithMetadata("Damage", "player1", -5, "Event_Trap", metadata)

	if entry.Type != EntryTypeAction {
		t.Errorf("Type should be action, got %s", entry.Type)
	}
	if entry.ActionType != "Damage" {
		t.Errorf("ActionType should be Damage, got %s", entry.ActionType)
	}
	if entry.Target != "player1" {
		t.Errorf("Target should be player1, got %s", entry.Target)
	}
	if entry.Delta != -5 {
		t.Errorf("Delta should be -5, got %d", entry.Delta)
	}
	if entry.Source != "Event_Trap" {
		t.Errorf("Source should be Event_Trap, got %s", entry.Source)
	}
	if entry.Timestamp.IsZero() {
		t.Error("Timestamp should be set")
	}
	if entry.Metadata == nil {
		t.Error("Metadata should not be nil")
	}
}

func TestNewStateEntry(t *testing.T) {
	entry := NewStateEntry("TurnUpkeep", "MainAction", "player1")

	if entry.Type != EntryTypeState {
		t.Errorf("Type should be state, got %s", entry.Type)
	}
	if entry.Target != "player1" {
		t.Errorf("Target should be player1, got %s", entry.Target)
	}

	// Use type-safe metadata access
	if entry.Metadata == nil {
		t.Fatal("Metadata should not be nil")
	}
	from := entry.Metadata.GetStringOrDefault("from", "")
	if from != "TurnUpkeep" {
		t.Errorf("from should be TurnUpkeep, got %s", from)
	}
	to := entry.Metadata.GetStringOrDefault("to", "")
	if to != "MainAction" {
		t.Errorf("to should be MainAction, got %s", to)
	}
}

func TestLogEntryJSONSerialization(t *testing.T) {
	metadata := util.NewMetadata().SetInt("checkpoint_pos", 50)
	entry := NewActionEntryWithMetadata("Respawn", "player1", 0, "DeathRespawn", metadata)

	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("JSON marshal failed: %v", err)
	}

	// Verify JSON contains expected fields
	var parsed LogEntry
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}

	if parsed.ActionType != "Respawn" {
		t.Errorf("ActionType should be Respawn, got %s", parsed.ActionType)
	}
	if parsed.Source != "DeathRespawn" {
		t.Errorf("Source should be DeathRespawn, got %s", parsed.Source)
	}

	// Verify metadata deserialization
	if parsed.Metadata == nil {
		t.Fatal("Parsed metadata should not be nil")
	}
	pos := parsed.Metadata.GetIntOrDefault("checkpoint_pos", 0)
	if pos != 50 {
		t.Errorf("checkpoint_pos should be 50, got %d", pos)
	}
}

// ========== TurnSegment Tests ==========

func TestNewTurnSegment(t *testing.T) {
	seg := NewTurnSegment(1, 0, "player1")

	if seg.Round != 1 {
		t.Errorf("Round should be 1, got %d", seg.Round)
	}
	if seg.Turn != 0 {
		t.Errorf("Turn should be 0, got %d", seg.Turn)
	}
	if seg.PlayerID != "player1" {
		t.Errorf("PlayerID should be player1, got %s", seg.PlayerID)
	}
	if seg.Len() != 0 {
		t.Errorf("New segment should have 0 entries, got %d", seg.Len())
	}
	if seg.StartTime.IsZero() {
		t.Error("StartTime should be set")
	}
}

func TestTurnSegmentAddEntry(t *testing.T) {
	seg := NewTurnSegment(1, 0, "player1")

	seg.AddEntry(NewActionEntry("Move", "player1", 5, "DiceRoll"))
	seg.AddEntry(NewActionEntry("Damage", "player1", -2, "Event_Trap"))

	if seg.Len() != 2 {
		t.Errorf("Should have 2 entries, got %d", seg.Len())
	}
}

func TestTurnSegmentEnd(t *testing.T) {
	seg := NewTurnSegment(1, 0, "player1")

	if !seg.EndTime.IsZero() {
		t.Error("EndTime should be zero before End()")
	}

	seg.End()

	if seg.EndTime.IsZero() {
		t.Error("EndTime should be set after End()")
	}

	// Verify EndTime is after StartTime
	if seg.EndTime.Before(seg.StartTime) {
		t.Error("EndTime should be after StartTime")
	}
}

// ========== Concurrency Tests ==========

func TestGameLogConcurrentAccess(t *testing.T) {
	log := NewGameLog()

	// Start multiple goroutines adding entries
	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func(idx int) {
			log.StartTurn(1, idx, "player"+string(rune('0'+idx)))
			log.AddEntry(NewActionEntry("Move", "player"+string(rune('0'+idx)), idx, "DiceRoll"))
			log.EndTurn()
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("Timeout waiting for goroutine")
		}
	}

	// Verify all segments exist
	segments := log.GetTurnSegments()
	if len(segments) != 10 {
		t.Errorf("Should have 10 segments, got %d", len(segments))
	}
}