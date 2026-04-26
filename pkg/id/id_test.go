package id

import (
	"encoding/json"
	"sync"
	"testing"

	"github.com/b1tAction/paradiced/pkg/constants"
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

	// Empty string
	_, err = ParseID("test", "")
	if err == nil {
		t.Error("Parse empty string should return error")
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

func TestMustParseIDEmptyPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustParseID with empty string should panic")
		}
	}()
	MustParseID("test", "")
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

func TestIDUnmarshalJSONEmpty(t *testing.T) {
	var id ID
	id.prefix = "test"
	err := json.Unmarshal([]byte(`""`), &id)
	if err != nil {
		t.Errorf("UnmarshalJSON empty string failed: %v", err)
	}
	if !id.IsZero() {
		t.Error("UnmarshalJSON empty string should result in zero ID")
	}
}

func TestIDUnmarshalJSONInvalid(t *testing.T) {
	var id ID
	id.prefix = "test"

	// Invalid JSON format (not a string)
	err := json.Unmarshal([]byte(`123`), &id)
	if err == nil {
		t.Error("UnmarshalJSON with number should return error")
	}

	// Invalid JSON format (object)
	err = json.Unmarshal([]byte(`{"key":"value"}`), &id)
	if err == nil {
		t.Error("UnmarshalJSON with object should return error")
	}

	// Invalid UUID string
	err = json.Unmarshal([]byte(`"not-a-uuid"`), &id)
	if err == nil {
		t.Error("UnmarshalJSON with invalid UUID should return error")
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

func TestIDStringZero(t *testing.T) {
	id := ZeroID("test")
	// Zero ID should return "prefix-nil"
	s := id.String()
	if s != "test-nil" {
		t.Errorf("ZeroID.String() = %s, expected 'test-nil'", s)
	}

	// Non-zero ID should return "prefix-uuid"
	id2 := NewID("test")
	s2 := id2.String()
	if !stringsContains(s2, "test-") {
		t.Errorf("NewID.String() should contain 'test-', got %s", s2)
	}
	if stringsContains(s2, "nil") {
		t.Errorf("NewID.String() should not contain 'nil', got %s", s2)
	}
}

func TestIDUUIDZero(t *testing.T) {
	id := ZeroID("test")
	// Zero ID should return empty string
	u := id.UUID()
	if u != "" {
		t.Errorf("ZeroID.UUID() = %s, expected empty string", u)
	}

	// Non-zero ID should return 36-char UUID
	id2 := NewID("test")
	u2 := id2.UUID()
	if u2 == "" {
		t.Error("NewID.UUID() should not be empty")
	}
	if len(u2) != 36 {
		t.Errorf("NewID.UUID() length = %d, expected 36", len(u2))
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

func TestPlayerIDIsBoss(t *testing.T) {
	// Boss UUID player should be detected
	bossID := MustParsePlayerID(constants.BossPlayerUUID)
	if !bossID.IsBoss() {
		t.Error("PlayerID with Boss UUID should return IsBoss=true")
	}

	// Normal player should not be detected
	normalID := NewPlayerID()
	if normalID.IsBoss() {
		t.Error("Normal PlayerID should return IsBoss=false")
	}

	// Zero player ID should not be boss
	zeroID := ZeroPlayerID()
	if zeroID.IsBoss() {
		t.Error("Zero PlayerID should return IsBoss=false")
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

func TestTestUUID(t *testing.T) {
	u1 := TestUUID(1)
	if len(u1) != 36 {
		t.Errorf("TestUUID(1) length = %d, expected 36", len(u1))
	}
	// Should end with the index
	if u1[24:] != "000000000001" {
		t.Errorf("TestUUID(1) suffix = %s, expected 000000000001", u1[24:])
	}

	u0 := TestUUID(0)
	if u0[24:] != "000000000000" {
		t.Errorf("TestUUID(0) suffix = %s, expected 000000000000", u0[24:])
	}

	u999 := TestUUID(999)
	if u999[24:] != "000000000999" {
		t.Errorf("TestUUID(999) suffix = %s, expected 000000000999", u999[24:])
	}

	// Should produce valid UUIDs usable for parsing
	parsed, err := ParsePlayerID(TestUUID(42))
	if err != nil {
		t.Errorf("ParsePlayerID(TestUUID(42)) failed: %v", err)
	}
	if parsed.UUID() != TestUUID(42) {
		t.Errorf("Parsed UUID = %s, expected %s", parsed.UUID(), TestUUID(42))
	}
}

// ========== Parse Functions Tests ==========

func TestParsePlayerID(t *testing.T) {
	original := NewPlayerID()
	uuidStr := original.UUID()

	// Parse pure UUID
	parsed, err := ParsePlayerID(uuidStr)
	if err != nil {
		t.Errorf("ParsePlayerID pure UUID failed: %v", err)
	}
	if parsed.UUID() != uuidStr {
		t.Errorf("Parsed UUID mismatch: %s vs %s", parsed.UUID(), uuidStr)
	}

	// Parse prefixed format
	parsed2, err := ParsePlayerID("player-" + uuidStr)
	if err != nil {
		t.Errorf("ParsePlayerID prefixed failed: %v", err)
	}
	if parsed2.UUID() != uuidStr {
		t.Errorf("Parsed2 UUID mismatch: %s vs %s", parsed2.UUID(), uuidStr)
	}
}

func TestParsePlayerIDInvalid(t *testing.T) {
	_, err := ParsePlayerID("invalid")
	if err == nil {
		t.Error("ParsePlayerID with invalid should return error")
	}
}

func TestMustParsePlayerIDPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustParsePlayerID with invalid should panic")
		}
	}()
	MustParsePlayerID("invalid")
}

func TestZeroPlayerID(t *testing.T) {
	zero := ZeroPlayerID()
	if !zero.IsZero() {
		t.Error("ZeroPlayerID should be zero")
	}
	if zero.IsValid() {
		t.Error("ZeroPlayerID should not be valid")
	}
}

func TestPlayerIDUnmarshalJSON(t *testing.T) {
	uuidStr := "0194fdc2-fa2f-7cc3-95c0-18c0c013c0be"
	jsonStr := `"` + uuidStr + `"`

	var id PlayerID
	err := json.Unmarshal([]byte(jsonStr), &id)
	if err != nil {
		t.Errorf("PlayerID UnmarshalJSON failed: %v", err)
	}
	if id.UUID() != uuidStr {
		t.Errorf("PlayerID UnmarshalJSON UUID = %s, expected %s", id.UUID(), uuidStr)
	}
	// Check prefix is set correctly
	if !stringsContains(id.String(), "player-") {
		t.Errorf("PlayerID String should contain 'player-', got %s", id.String())
	}
}

func TestPlayerIDUnmarshalJSONInvalid(t *testing.T) {
	var id PlayerID

	// Invalid JSON format (not a string)
	err := json.Unmarshal([]byte(`123`), &id)
	if err == nil {
		t.Error("PlayerID UnmarshalJSON with number should return error")
	}

	// Empty string
	err = json.Unmarshal([]byte(`""`), &id)
	if err != nil {
		t.Errorf("PlayerID UnmarshalJSON empty string failed: %v", err)
	}
	if !id.IsZero() {
		t.Error("PlayerID UnmarshalJSON empty string should result in zero ID")
	}

	// Invalid UUID
	err = json.Unmarshal([]byte(`"not-a-uuid"`), &id)
	if err == nil {
		t.Error("PlayerID UnmarshalJSON with invalid UUID should return error")
	}
}

func TestParseBuffID(t *testing.T) {
	original := NewBuffID()
	uuidStr := original.UUID()

	parsed, err := ParseBuffID(uuidStr)
	if err != nil {
		t.Errorf("ParseBuffID failed: %v", err)
	}
	if parsed.UUID() != uuidStr {
		t.Errorf("Parsed UUID mismatch: %s vs %s", parsed.UUID(), uuidStr)
	}
}

func TestMustParseBuffIDPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustParseBuffID with invalid should panic")
		}
	}()
	MustParseBuffID("invalid")
}

func TestZeroBuffID(t *testing.T) {
	zero := ZeroBuffID()
	if !zero.IsZero() {
		t.Error("ZeroBuffID should be zero")
	}
}

func TestBuffIDUnmarshalJSON(t *testing.T) {
	uuidStr := "0194fdc2-fa2f-7cc3-95c0-18c0c013c0be"
	var id BuffID
	err := json.Unmarshal([]byte(`"`+uuidStr+`"`), &id)
	if err != nil {
		t.Errorf("BuffID UnmarshalJSON failed: %v", err)
	}
	if id.UUID() != uuidStr {
		t.Errorf("BuffID UUID mismatch: %s vs %s", id.UUID(), uuidStr)
	}
}

func TestBuffIDUnmarshalJSONInvalid(t *testing.T) {
	var id BuffID

	// Invalid JSON format
	err := json.Unmarshal([]byte(`123`), &id)
	if err == nil {
		t.Error("BuffID UnmarshalJSON with number should return error")
	}

	// Empty string
	err = json.Unmarshal([]byte(`""`), &id)
	if err != nil {
		t.Errorf("BuffID UnmarshalJSON empty string failed: %v", err)
	}
	if !id.IsZero() {
		t.Error("BuffID UnmarshalJSON empty string should result in zero ID")
	}
}

func TestParseItemID(t *testing.T) {
	original := NewItemID()
	uuidStr := original.UUID()

	parsed, err := ParseItemID(uuidStr)
	if err != nil {
		t.Errorf("ParseItemID failed: %v", err)
	}
	if parsed.UUID() != uuidStr {
		t.Errorf("Parsed UUID mismatch")
	}
}

func TestMustParseItemIDPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustParseItemID with invalid should panic")
		}
	}()
	MustParseItemID("invalid")
}

func TestZeroItemID(t *testing.T) {
	zero := ZeroItemID()
	if !zero.IsZero() {
		t.Error("ZeroItemID should be zero")
	}
}

func TestItemIDUnmarshalJSON(t *testing.T) {
	uuidStr := "0194fdc2-fa2f-7cc3-95c0-18c0c013c0be"
	var id ItemID
	err := json.Unmarshal([]byte(`"`+uuidStr+`"`), &id)
	if err != nil {
		t.Errorf("ItemID UnmarshalJSON failed: %v", err)
	}
	if id.UUID() != uuidStr {
		t.Errorf("ItemID UUID mismatch")
	}
}

func TestItemIDUnmarshalJSONInvalid(t *testing.T) {
	var id ItemID

	// Invalid JSON format
	err := json.Unmarshal([]byte(`123`), &id)
	if err == nil {
		t.Error("ItemID UnmarshalJSON with number should return error")
	}

	// Empty string
	err = json.Unmarshal([]byte(`""`), &id)
	if err != nil {
		t.Errorf("ItemID UnmarshalJSON empty string failed: %v", err)
	}
	if !id.IsZero() {
		t.Error("ItemID UnmarshalJSON empty string should result in zero ID")
	}
}

func TestParseGameID(t *testing.T) {
	original := NewGameID()
	uuidStr := original.UUID()

	parsed, err := ParseGameID(uuidStr)
	if err != nil {
		t.Errorf("ParseGameID failed: %v", err)
	}
	if parsed.UUID() != uuidStr {
		t.Errorf("Parsed UUID mismatch")
	}
}

func TestMustParseGameIDPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustParseGameID with invalid should panic")
		}
	}()
	MustParseGameID("invalid")
}

func TestZeroGameID(t *testing.T) {
	zero := ZeroGameID()
	if !zero.IsZero() {
		t.Error("ZeroGameID should be zero")
	}
}

func TestGameIDUnmarshalJSON(t *testing.T) {
	uuidStr := "0194fdc2-fa2f-7cc3-95c0-18c0c013c0be"
	var id GameID
	err := json.Unmarshal([]byte(`"`+uuidStr+`"`), &id)
	if err != nil {
		t.Errorf("GameID UnmarshalJSON failed: %v", err)
	}
	if id.UUID() != uuidStr {
		t.Errorf("GameID UUID mismatch")
	}
}

func TestGameIDUnmarshalJSONInvalid(t *testing.T) {
	var id GameID

	// Invalid JSON format
	err := json.Unmarshal([]byte(`123`), &id)
	if err == nil {
		t.Error("GameID UnmarshalJSON with number should return error")
	}

	// Empty string
	err = json.Unmarshal([]byte(`""`), &id)
	if err != nil {
		t.Errorf("GameID UnmarshalJSON empty string failed: %v", err)
	}
	if !id.IsZero() {
		t.Error("GameID UnmarshalJSON empty string should result in zero ID")
	}
}

func TestParseSubscriptionID(t *testing.T) {
	original := NewSubscriptionID()
	uuidStr := original.UUID()

	parsed, err := ParseSubscriptionID(uuidStr)
	if err != nil {
		t.Errorf("ParseSubscriptionID failed: %v", err)
	}
	if parsed.UUID() != uuidStr {
		t.Errorf("Parsed UUID mismatch")
	}
}

func TestMustParseSubscriptionIDPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustParseSubscriptionID with invalid should panic")
		}
	}()
	MustParseSubscriptionID("invalid")
}

func TestZeroSubscriptionID(t *testing.T) {
	zero := ZeroSubscriptionID()
	if !zero.IsZero() {
		t.Error("ZeroSubscriptionID should be zero")
	}
}

func TestSubscriptionIDUnmarshalJSON(t *testing.T) {
	uuidStr := "0194fdc2-fa2f-7cc3-95c0-18c0c013c0be"
	var id SubscriptionID
	err := json.Unmarshal([]byte(`"`+uuidStr+`"`), &id)
	if err != nil {
		t.Errorf("SubscriptionID UnmarshalJSON failed: %v", err)
	}
	if id.UUID() != uuidStr {
		t.Errorf("SubscriptionID UUID mismatch")
	}
}

func TestSubscriptionIDUnmarshalJSONInvalid(t *testing.T) {
	var id SubscriptionID

	// Invalid JSON format
	err := json.Unmarshal([]byte(`123`), &id)
	if err == nil {
		t.Error("SubscriptionID UnmarshalJSON with number should return error")
	}

	// Empty string
	err = json.Unmarshal([]byte(`""`), &id)
	if err != nil {
		t.Errorf("SubscriptionID UnmarshalJSON empty string failed: %v", err)
	}
	if !id.IsZero() {
		t.Error("SubscriptionID UnmarshalJSON empty string should result in zero ID")
	}
}

func TestParseDecisionID(t *testing.T) {
	original := NewDecisionID()
	uuidStr := original.UUID()

	parsed, err := ParseDecisionID(uuidStr)
	if err != nil {
		t.Errorf("ParseDecisionID failed: %v", err)
	}
	if parsed.UUID() != uuidStr {
		t.Errorf("Parsed UUID mismatch")
	}
}

func TestMustParseDecisionIDPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustParseDecisionID with invalid should panic")
		}
	}()
	MustParseDecisionID("invalid")
}

func TestZeroDecisionID(t *testing.T) {
	zero := ZeroDecisionID()
	if !zero.IsZero() {
		t.Error("ZeroDecisionID should be zero")
	}
}

func TestDecisionIDUnmarshalJSON(t *testing.T) {
	uuidStr := "0194fdc2-fa2f-7cc3-95c0-18c0c013c0be"
	var id DecisionID
	err := json.Unmarshal([]byte(`"`+uuidStr+`"`), &id)
	if err != nil {
		t.Errorf("DecisionID UnmarshalJSON failed: %v", err)
	}
	if id.UUID() != uuidStr {
		t.Errorf("DecisionID UUID mismatch")
	}
}

func TestDecisionIDUnmarshalJSONInvalid(t *testing.T) {
	var id DecisionID

	// Invalid JSON format
	err := json.Unmarshal([]byte(`123`), &id)
	if err == nil {
		t.Error("DecisionID UnmarshalJSON with number should return error")
	}

	// Empty string
	err = json.Unmarshal([]byte(`""`), &id)
	if err != nil {
		t.Errorf("DecisionID UnmarshalJSON empty string failed: %v", err)
	}
	if !id.IsZero() {
		t.Error("DecisionID UnmarshalJSON empty string should result in zero ID")
	}
}

func TestNewEventID(t *testing.T) {
	id := NewEventID()
	if !stringsContains(id.String(), "event-") {
		t.Errorf("EventID.String() should contain 'event-', got %s", id.String())
	}
}

func TestParseEventID(t *testing.T) {
	original := NewEventID()
	uuidStr := original.UUID()

	parsed, err := ParseEventID(uuidStr)
	if err != nil {
		t.Errorf("ParseEventID failed: %v", err)
	}
	if parsed.UUID() != uuidStr {
		t.Errorf("Parsed UUID mismatch")
	}
}

func TestMustParseEventIDPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustParseEventID with invalid should panic")
		}
	}()
	MustParseEventID("invalid")
}

func TestZeroEventID(t *testing.T) {
	zero := ZeroEventID()
	if !zero.IsZero() {
		t.Error("ZeroEventID should be zero")
	}
}

func TestEventIDUnmarshalJSON(t *testing.T) {
	uuidStr := "0194fdc2-fa2f-7cc3-95c0-18c0c013c0be"
	var id EventID
	err := json.Unmarshal([]byte(`"`+uuidStr+`"`), &id)
	if err != nil {
		t.Errorf("EventID UnmarshalJSON failed: %v", err)
	}
	if id.UUID() != uuidStr {
		t.Errorf("EventID UUID mismatch")
	}
}

func TestEventIDUnmarshalJSONInvalid(t *testing.T) {
	var id EventID

	// Invalid JSON format
	err := json.Unmarshal([]byte(`123`), &id)
	if err == nil {
		t.Error("EventID UnmarshalJSON with number should return error")
	}

	// Empty string
	err = json.Unmarshal([]byte(`""`), &id)
	if err != nil {
		t.Errorf("EventID UnmarshalJSON empty string failed: %v", err)
	}
	if !id.IsZero() {
		t.Error("EventID UnmarshalJSON empty string should result in zero ID")
	}
}