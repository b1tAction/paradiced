package util

import (
	"testing"
)

// ========== Initialization Tests ==========

func TestNewMetadata(t *testing.T) {
	m := NewMetadata()
	if m == nil {
		t.Fatal("NewMetadata should return non-nil")
	}
	if m.values == nil {
		t.Error("values map should be initialized")
	}
	if m.Size() != 0 {
		t.Errorf("new metadata size = %d, expected 0", m.Size())
	}
}

// ========== Set/Get Basic Tests ==========

func TestSetAndGet(t *testing.T) {
	m := NewMetadata()

	m.Set("key1", 100)
	m.Set("key2", "hello")
	m.Set("key3", true)

	val1, ok1 := m.Get("key1")
	if !ok1 {
		t.Error("key1 should exist")
	}
	if val1 != 100 {
		t.Errorf("key1 = %v, expected 100", val1)
	}

	val2, ok2 := m.Get("key2")
	if !ok2 {
		t.Error("key2 should exist")
	}
	if val2 != "hello" {
		t.Errorf("key2 = %v, expected hello", val2)
	}

	// Get nonexistent key
	val, ok := m.Get("nonexistent")
	if ok {
		t.Error("nonexistent key should not exist")
	}
	if val != nil {
		t.Errorf("nonexistent key value = %v, expected nil", val)
	}
}

func TestSetOverwrite(t *testing.T) {
	m := NewMetadata()

	m.Set("key", 100)
	m.Set("key", 200)

	val, err := m.GetInt("key")
	if err != nil {
		t.Fatalf("GetInt failed: %v", err)
	}
	if val != 200 {
		t.Errorf("overwritten key = %d, expected 200", val)
	}
}

// ========== GetInt/SetInt Tests ==========

func TestGetInt(t *testing.T) {
	m := NewMetadata()

	// Set and get integer
	m.SetInt("count", 42)
	val, err := m.GetInt("count")
	if err != nil {
		t.Fatalf("GetInt failed: %v", err)
	}
	if val != 42 {
		t.Errorf("GetInt = %d, expected 42", val)
	}

	// Get nonexistent key returns error
	_, err = m.GetInt("nonexistent")
	if err == nil {
		t.Error("GetInt nonexistent should return error")
	}

	// Type mismatch returns error
	m.Set("string_key", "not an int")
	_, err = m.GetInt("string_key")
	if err == nil {
		t.Error("GetInt with wrong type should return error")
	}
}

func TestGetIntOrDefault(t *testing.T) {
	m := NewMetadata()

	// Existing key returns actual value
	m.SetInt("existing", 10)
	if m.GetIntOrDefault("existing", 5) != 10 {
		t.Errorf("GetIntOrDefault existing = %d, expected 10", m.GetIntOrDefault("existing", 5))
	}

	// Nonexistent key returns default
	if m.GetIntOrDefault("nonexistent", 5) != 5 {
		t.Errorf("GetIntOrDefault nonexistent = %d, expected 5", m.GetIntOrDefault("nonexistent", 5))
	}

	// Type mismatch returns default
	m.Set("string_key", "not an int")
	if m.GetIntOrDefault("string_key", 5) != 5 {
		t.Errorf("GetIntOrDefault with wrong type = %d, expected 5", m.GetIntOrDefault("string_key", 5))
	}
}

// ========== GetBool/SetBool Tests ==========

func TestGetBool(t *testing.T) {
	m := NewMetadata()

	m.SetBool("flag_true", true)
	m.SetBool("flag_false", false)

	val, err := m.GetBool("flag_true")
	if err != nil {
		t.Fatalf("GetBool failed: %v", err)
	}
	if !val {
		t.Error("GetBool flag_true should be true")
	}

	val, err = m.GetBool("flag_false")
	if err != nil {
		t.Fatalf("GetBool failed: %v", err)
	}
	if val {
		t.Error("GetBool flag_false should be false")
	}

	// Nonexistent key returns error
	_, err = m.GetBool("nonexistent")
	if err == nil {
		t.Error("GetBool nonexistent should return error")
	}

	// Type mismatch returns error
	m.Set("int_key", 100)
	_, err = m.GetBool("int_key")
	if err == nil {
		t.Error("GetBool with wrong type should return error")
	}
}

func TestGetBoolOrDefault(t *testing.T) {
	m := NewMetadata()

	m.SetBool("flag", true)
	if !m.GetBoolOrDefault("flag", false) {
		t.Error("GetBoolOrDefault flag should be true")
	}

	if m.GetBoolOrDefault("nonexistent", true) != true {
		t.Error("GetBoolOrDefault nonexistent should return default true")
	}
}

// ========== GetString/SetString Tests ==========

func TestGetString(t *testing.T) {
	m := NewMetadata()

	m.SetString("name", "test")
	val, err := m.GetString("name")
	if err != nil {
		t.Fatalf("GetString failed: %v", err)
	}
	if val != "test" {
		t.Errorf("GetString = %s, expected test", val)
	}

	// Nonexistent key returns error
	_, err = m.GetString("nonexistent")
	if err == nil {
		t.Error("GetString nonexistent should return error")
	}

	// Type mismatch returns error
	m.Set("int_key", 100)
	_, err = m.GetString("int_key")
	if err == nil {
		t.Error("GetString with wrong type should return error")
	}
}

func TestGetStringOrDefault(t *testing.T) {
	m := NewMetadata()

	m.SetString("name", "test")
	if m.GetStringOrDefault("name", "default") != "test" {
		t.Errorf("GetStringOrDefault name = %s, expected test", m.GetStringOrDefault("name", "default"))
	}

	if m.GetStringOrDefault("nonexistent", "default") != "default" {
		t.Errorf("GetStringOrDefault nonexistent = %s, expected default", m.GetStringOrDefault("nonexistent", "default"))
	}
}

// ========== GetFloat64/SetFloat64 Tests ==========

func TestGetFloat64(t *testing.T) {
	m := NewMetadata()

	m.SetFloat64("ratio", 3.14)
	val, err := m.GetFloat64("ratio")
	if err != nil {
		t.Fatalf("GetFloat64 failed: %v", err)
	}
	if val != 3.14 {
		t.Errorf("GetFloat64 = %f, expected 3.14", val)
	}

	// Nonexistent key returns error
	_, err = m.GetFloat64("nonexistent")
	if err == nil {
		t.Error("GetFloat64 nonexistent should return error")
	}
}

func TestGetFloat64OrDefault(t *testing.T) {
	m := NewMetadata()

	m.SetFloat64("ratio", 3.14)
	if m.GetFloat64OrDefault("ratio", 1.0) != 3.14 {
		t.Errorf("GetFloat64OrDefault ratio = %f, expected 3.14", m.GetFloat64OrDefault("ratio", 1.0))
	}

	if m.GetFloat64OrDefault("nonexistent", 1.0) != 1.0 {
		t.Errorf("GetFloat64OrDefault nonexistent = %f, expected 1.0", m.GetFloat64OrDefault("nonexistent", 1.0))
	}
}

// ========== HasKey/Delete/Clear Tests ==========

func TestHasKey(t *testing.T) {
	m := NewMetadata()

	m.Set("existing", 100)

	if !m.HasKey("existing") {
		t.Error("existing key should be found")
	}
	if m.HasKey("nonexistent") {
		t.Error("nonexistent key should not be found")
	}
}

func TestDelete(t *testing.T) {
	m := NewMetadata()

	m.Set("key", 100)
	if !m.HasKey("key") {
		t.Fatal("key should exist before delete")
	}

	m.Delete("key")
	if m.HasKey("key") {
		t.Error("key should not exist after delete")
	}

	// Deleting nonexistent key doesn't panic
	m.Delete("nonexistent")
}

func TestClear(t *testing.T) {
	m := NewMetadata()

	m.Set("key1", 100)
	m.Set("key2", 200)
	m.Set("key3", 300)

	if m.Size() != 3 {
		t.Fatalf("size = %d, expected 3 before clear", m.Size())
	}

	m.Clear()

	if m.Size() != 0 {
		t.Errorf("size = %d, expected 0 after clear", m.Size())
	}
	if m.HasKey("key1") {
		t.Error("key1 should not exist after clear")
	}
}

// ========== Keys Tests ==========

func TestKeys(t *testing.T) {
	m := NewMetadata()

	m.Set("a", 1)
	m.Set("b", 2)
	m.Set("c", 3)

	keys := m.Keys()
	if len(keys) != 3 {
		t.Errorf("keys count = %d, expected 3", len(keys))
	}

	// Verify all keys are in the returned list
	keySet := make(map[string]bool)
	for _, k := range keys {
		keySet[k] = true
	}
	for _, expected := range []string{"a", "b", "c"} {
		if !keySet[expected] {
			t.Errorf("key %s not found in Keys()", expected)
		}
	}
}

// ========== Clone Tests ==========

func TestClone(t *testing.T) {
	m := NewMetadata()
	m.SetInt("count", 10)
	m.SetString("name", "original")

	cloned := m.Clone()

	// Clone is independent
	cloned.SetInt("count", 20)
	cloned.SetString("name", "cloned")

	// Original unaffected
	val, err := m.GetInt("count")
	if err != nil {
		t.Fatalf("GetInt failed: %v", err)
	}
	if val != 10 {
		t.Errorf("original count = %d, expected 10", val)
	}

	sval, err := m.GetString("name")
	if err != nil {
		t.Fatalf("GetString failed: %v", err)
	}
	if sval != "original" {
		t.Errorf("original name = %s, expected original", sval)
	}

	// Clone values correct
	val, err = cloned.GetInt("count")
	if err != nil {
		t.Fatalf("GetInt failed: %v", err)
	}
	if val != 20 {
		t.Errorf("cloned count = %d, expected 20", val)
	}

	sval, err = cloned.GetString("name")
	if err != nil {
		t.Fatalf("GetString failed: %v", err)
	}
	if sval != "cloned" {
		t.Errorf("cloned name = %s, expected cloned", sval)
	}
}

func TestCloneNil(t *testing.T) {
	m := NewMetadata()
	cloned := m.Clone()

	if cloned == nil {
		t.Error("Clone should return non-nil")
	}
	if cloned.Size() != 0 {
		t.Errorf("empty clone size = %d, expected 0", cloned.Size())
	}
}

// ========== IncrementInt/DecrementInt Tests ==========

func TestIncrementInt(t *testing.T) {
	m := NewMetadata()

	// Increment from 0 (key doesn't exist)
	result := m.IncrementInt("counter", 1)
	if result != 1 {
		t.Errorf("first increment = %d, expected 1", result)
	}
	val, err := m.GetInt("counter")
	if err != nil {
		t.Fatalf("GetInt failed: %v", err)
	}
	if val != 1 {
		t.Errorf("counter = %d, expected 1", val)
	}

	// Continue incrementing
	result = m.IncrementInt("counter", 5)
	if result != 6 {
		t.Errorf("second increment = %d, expected 6", result)
	}

	// Increment with negative (effectively decrement)
	result = m.IncrementInt("counter", -2)
	if result != 4 {
		t.Errorf("increment with negative = %d, expected 4", result)
	}
}

func TestDecrementInt(t *testing.T) {
	m := NewMetadata()

	m.SetInt("counter", 10)
	result := m.DecrementInt("counter", 3)
	if result != 7 {
		t.Errorf("decrement = %d, expected 7", result)
	}
}

// ========== Merge Tests ==========

func TestMerge(t *testing.T) {
	m1 := NewMetadata()
	m1.SetInt("a", 1)
	m1.SetInt("b", 2)

	m2 := NewMetadata()
	m2.SetInt("b", 20) // Same key, will be overwritten
	m2.SetInt("c", 3)

	m1.Merge(m2)

	val, err := m1.GetInt("a")
	if err != nil || val != 1 {
		t.Errorf("a = %d, expected 1", val)
	}

	val, err = m1.GetInt("b")
	if err != nil || val != 20 {
		t.Errorf("b = %d, expected 20 (overwritten)", val)
	}

	val, err = m1.GetInt("c")
	if err != nil || val != 3 {
		t.Errorf("c = %d, expected 3", val)
	}
}

func TestMergeNil(t *testing.T) {
	m := NewMetadata()
	m.SetInt("key", 100)

	// Merge nil shouldn't panic
	m.Merge(nil)

	val, err := m.GetInt("key")
	if err != nil || val != 100 {
		t.Errorf("key = %d, expected 100 (unchanged)", val)
	}
}

// ========== ToMap Tests ==========

func TestToMap(t *testing.T) {
	m := NewMetadata()
	m.SetInt("a", 1)
	m.SetString("b", "hello")

	mapCopy := m.ToMap()

	if mapCopy["a"] != 1 {
		t.Errorf("mapCopy[a] = %v, expected 1", mapCopy["a"])
	}
	if mapCopy["b"] != "hello" {
		t.Errorf("mapCopy[b] = %v, expected hello", mapCopy["b"])
	}

	// Modifying copy doesn't affect original
	mapCopy["a"] = 100
	val, err := m.GetInt("a")
	if err != nil || val != 1 {
		t.Errorf("original a = %d, expected 1", val)
	}
}

// ========== Chained Calls Tests ==========

func TestChainedCalls(t *testing.T) {
	m := NewMetadata()

	m.SetInt("int", 10).
		SetString("string", "hello").
		SetBool("bool", true)

	val, err := m.GetInt("int")
	if err != nil || val != 10 {
		t.Errorf("int = %d, expected 10", val)
	}

	sval, err := m.GetString("string")
	if err != nil || sval != "hello" {
		t.Errorf("string = %s, expected hello", sval)
	}

	bval, err := m.GetBool("bool")
	if err != nil || !bval {
		t.Error("bool should be true")
	}
}

// ========== Size Tests ==========

func TestSize(t *testing.T) {
	m := NewMetadata()

	if m.Size() != 0 {
		t.Errorf("initial size = %d, expected 0", m.Size())
	}

	m.Set("a", 1)
	m.Set("b", 2)

	if m.Size() != 2 {
		t.Errorf("size after set = %d, expected 2", m.Size())
	}

	m.Delete("a")

	if m.Size() != 1 {
		t.Errorf("size after delete = %d, expected 1", m.Size())
	}
}

// ========== JSON Serialization Tests ==========

func TestMetadataToJSON(t *testing.T) {
	m := NewMetadata()
	m.SetInt("count", 10)
	m.SetString("name", "test")
	m.SetBool("flag", true)

	// ToJSON returns the values map directly
	valuesMap := m.ToJSON()

	if valuesMap["count"] != 10 {
		t.Errorf("count = %v, want 10", valuesMap["count"])
	}
	if valuesMap["name"] != "test" {
		t.Errorf("name = %v, want test", valuesMap["name"])
	}
	if valuesMap["flag"] != true {
		t.Errorf("flag = %v, want true", valuesMap["flag"])
	}
}

func TestMetadataFromMap(t *testing.T) {
	input := map[string]interface{}{
		"count": 42,
		"name":  "from_map",
		"flag":  true,
	}

	m := FromMap(input)

	if m.GetIntOrDefault("count", 0) != 42 {
		t.Errorf("count = %d, want 42", m.GetIntOrDefault("count", 0))
	}
	if m.GetStringOrDefault("name", "") != "from_map" {
		t.Errorf("name = %s, want from_map", m.GetStringOrDefault("name", ""))
	}
	if !m.GetBoolOrDefault("flag", false) {
		t.Error("flag should be true")
	}
}

func TestMetadataMarshalJSON(t *testing.T) {
	m := NewMetadata()
	m.SetInt("value", 100)

	jsonBytes, err := m.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON error: %v", err)
	}

	// Verify can unmarshal back
	m2 := NewMetadata()
	if err := m2.UnmarshalJSON(jsonBytes); err != nil {
		t.Fatalf("UnmarshalJSON error: %v", err)
	}

	if m2.GetIntOrDefault("value", 0) != 100 {
		t.Errorf("value after unmarshal = %d, want 100", m2.GetIntOrDefault("value", 0))
	}
}

func TestMetadataUnmarshalJSON(t *testing.T) {
	m := NewMetadata()
	jsonStr := `{"key1": "value1", "key2": 42, "key3": true}`

	if err := m.UnmarshalJSON([]byte(jsonStr)); err != nil {
		t.Fatalf("UnmarshalJSON error: %v", err)
	}

	if m.GetStringOrDefault("key1", "") != "value1" {
		t.Errorf("key1 = %s, want value1", m.GetStringOrDefault("key1", ""))
	}
	if m.GetIntOrDefault("key2", 0) != 42 {
		t.Errorf("key2 = %d, want 42", m.GetIntOrDefault("key2", 0))
	}
	if !m.GetBoolOrDefault("key3", false) {
		t.Error("key3 should be true")
	}
}

func TestMetadataUnmarshalJSONInvalid(t *testing.T) {
	m := NewMetadata()

	// Invalid JSON should return error
	if err := m.UnmarshalJSON([]byte("invalid json")); err == nil {
		t.Error("UnmarshalJSON should return error for invalid JSON")
	}
}

func TestMetadataToJSONEmpty(t *testing.T) {
	m := NewMetadata()

	valuesMap := m.ToJSON()

	// Empty metadata should produce empty map
	if len(valuesMap) != 0 {
		t.Errorf("empty metadata should have 0 keys, got %d", len(valuesMap))
	}
}

func TestFromMapNil(t *testing.T) {
	m := FromMap(nil)

	if m == nil {
		t.Error("FromMap(nil) should return non-nil Metadata")
	}

	if m.Size() != 0 {
		t.Errorf("FromMap(nil) should return empty Metadata, got size %d", m.Size())
	}
}