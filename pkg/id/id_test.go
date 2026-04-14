package id

import (
	"encoding/json"
	"sync"
	"testing"
)

// ========== Base ID Tests ==========

func TestNewID(t *testing.T) {
	id := NewID("test")

	// Check UUID format
	if id.UUID() == "" {
		t.Error("UUID should not be empty")
	}
	if len(id.UUID()) != 36 {
		t.Errorf("UUID length = %d, expected 36", len(id.UUID()))
	}

	// Check String format (prefix-uuid)
	expectedLen := len("test-") + 36
	if len(id.String()) != expectedLen {
		t.Errorf("String length = %d, expected %d", len(id.String()), expectedLen)
	}
}

func TestIDIsZero(t *testing.T) {
	zero := ZeroID("test")
	if !zero.IsZero() {
		t.Error("ZeroID should be zero")
	}
	if zero.IsValid() {
		t.Error("ZeroID should not be valid")
	}

	normal := NewID("test")
	if normal.IsZero() {
		t.Error("NewID should not be zero")
	}
	if !normal.IsValid() {
		t.Error("NewID should be valid")
	}
}

func TestIDEqual(t *testing.T) {
	id1 := NewID("test")
	id2 := NewID("test")

	// Different UUIDs should not be equal
	if id1.Equal(id2) {
		t.Error("Different IDs should not be equal")
	}

	// Same UUID should be equal
	id3 := id1
	if !id1.Equal(id3) {
		t.Error("Same ID should be equal")
	}
}

// ========== Parse Tests ==========

func TestParseID(t *testing.T) {
	// Create a known ID
	id := NewID("test")
	uuidStr := id.UUID()

	// Parse pure UUID
	parsed, err := ParseID("test", uuidStr)
	if err != nil {
		t.Errorf("Parse pure UUID failed: %v", err)
	}
	if !parsed.Equal(id) {
		t.Error("Parsed ID should equal original")
	}

	// Parse prefixed format
	parsed2, err := ParseID("test", "test-"+uuidStr)
	if err != nil {
		t.Errorf("Parse prefixed format failed: %v", err)
	}
	if !parsed2.Equal(id) {
		t.Error("Parsed prefixed ID should equal original")
	}
}

func TestParseIDInvalid(t *testing.T) {
	_, err := ParseID("test", "invalid")
	if err == nil {
		t.Error("Parse invalid string should return error")
	}
}

func TestMustParseIDPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustParseID with invalid should panic")
		}
	}()
	MustParseID("test", "invalid")
}

// ========== JSON Tests ==========

func TestIDMarshalJSON(t *testing.T) {
	id := NewID("test")
	data, err := json.Marshal(id)
	if err != nil {
		t.Errorf("MarshalJSON failed: %v", err)
	}

	// Should be pure UUID, not prefixed
	expected := `"` + id.UUID() + `"`
	if string(data) != expected {
		t.Errorf("MarshalJSON = %s, expected %s", string(data), expected)
	}
}

func TestIDUnmarshalJSON(t *testing.T) {
	uuidStr := "0194fdc2-fa2f-7cc3-95c0-18c0c013c0be"
	jsonStr := `"` + uuidStr + `"`

	var id ID
	id.prefix = "test" // Set prefix before unmarshal
	err := json.Unmarshal([]byte(jsonStr), &id)
	if err != nil {
		t.Errorf("UnmarshalJSON failed: %v", err)
	}
	if id.UUID() != uuidStr {
		t.Errorf("UnmarshalJSON UUID = %s, expected %s", id.UUID(), uuidStr)
	}
}

func TestIDMarshalJSONZero(t *testing.T) {
	id := ZeroID("test")
	data, err := json.Marshal(id)
	if err != nil {
		t.Errorf("MarshalJSON zero failed: %v", err)
	}
	if string(data) != `"\"\""` && string(data) != `""` {
		t.Errorf("MarshalJSON zero = %s, expected empty string", string(data))
	}
}

// ========== Typed ID Tests ==========

func TestPlayerID(t *testing.T) {
	id := NewPlayerID()

	// Check prefix
	if !stringsContains(id.String(), "player-") {
		t.Errorf("PlayerID.String() should contain 'player-', got %s", id.String())
	}

	// Check JSON output is pure UUID
	data, _ := json.Marshal(id)
	if stringsContains(string(data), "player-") {
		t.Errorf("PlayerID JSON should not contain prefix, got %s", string(data))
	}
}

func TestBuffID(t *testing.T) {
	id := NewBuffID()
	if !stringsContains(id.String(), "buff-") {
		t.Errorf("BuffID.String() should contain 'buff-', got %s", id.String())
	}
}

func TestItemID(t *testing.T) {
	id := NewItemID()
	if !stringsContains(id.String(), "item-") {
		t.Errorf("ItemID.String() should contain 'item-', got %s", id.String())
	}
}

func TestGameID(t *testing.T) {
	id := NewGameID()
	if !stringsContains(id.String(), "game-") {
		t.Errorf("GameID.String() should contain 'game-', got %s", id.String())
	}
}

func TestSubscriptionID(t *testing.T) {
	id := NewSubscriptionID()
	if !stringsContains(id.String(), "sub-") {
		t.Errorf("SubscriptionID.String() should contain 'sub-', got %s", id.String())
	}
}

func TestDecisionID(t *testing.T) {
	id := NewDecisionID()
	if !stringsContains(id.String(), "dec-") {
		t.Errorf("DecisionID.String() should contain 'dec-', got %s", id.String())
	}
}

// ========== Type Safety Tests ==========

func TestTypeSafety(t *testing.T) {
	playerID := NewPlayerID()
	buffID := NewBuffID()

	// Different types should have different prefixes
	if playerID.prefix == buffID.prefix {
		t.Error("PlayerID and BuffID should have different prefixes")
	}

	// String output should be distinguishable
	if playerID.String() == buffID.String() {
		t.Error("PlayerID.String() and BuffID.String() should be different")
	}
}

// ========== Concurrent Safety ==========

func TestIDConcurrentGeneration(t *testing.T) {
	count := 100
	ids := make(chan PlayerID, count)
	var wg sync.WaitGroup

	for i := 0; i < count; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ids <- NewPlayerID()
		}()
	}

	wg.Wait()
	close(ids)

	// Check all IDs are unique
	seen := make(map[string]bool)
	for id := range ids {
		uuid := id.UUID()
		if seen[uuid] {
			t.Errorf("Duplicate UUID generated: %s", uuid)
		}
		seen[uuid] = true
	}
}

// ========== JSON Roundtrip Tests ==========

func TestPlayerIDJSONRoundtrip(t *testing.T) {
	original := NewPlayerID()
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var parsed PlayerID
	err = json.Unmarshal(data, &parsed)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// UUID should match
	if parsed.UUID() != original.UUID() {
		t.Errorf("Roundtrip UUID mismatch: %s vs %s", parsed.UUID(), original.UUID())
	}

	// Prefix should be set correctly
	if parsed.String() != "player-"+original.UUID() {
		t.Errorf("Parsed prefix not set correctly: %s", parsed.String())
	}
}

// Helper function
func stringsContains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr
}