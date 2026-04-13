package util

import (
	"errors"
	"fmt"
)

// Metadata is a dynamic attribute bag for type-safe key-value storage.
// It can be embedded anonymously to give any struct type-safe access methods.
type Metadata struct {
	values map[string]interface{}
}

// NewMetadata initializes a new Metadata instance.
func NewMetadata() *Metadata {
	return &Metadata{
		values: make(map[string]interface{}),
	}
}

// Set stores a value with the given key. Returns itself for chaining.
func (m *Metadata) Set(key string, value interface{}) *Metadata {
	m.values[key] = value
	return m
}

// Get retrieves the raw value for a key. Returns (nil, false) if key doesn't exist.
func (m *Metadata) Get(key string) (interface{}, bool) {
	val, ok := m.values[key]
	return val, ok
}

// GetInt retrieves an integer value. Returns error if key doesn't exist or type mismatch.
func (m *Metadata) GetInt(key string) (int, error) {
	val, ok := m.values[key]
	if !ok {
		return 0, fmt.Errorf("key '%s' not found", key)
	}
	i, ok := val.(int)
	if !ok {
		return 0, fmt.Errorf("key '%s' is not int, got %T", key, val)
	}
	return i, nil
}

// GetIntOrDefault retrieves an integer value, returns default if key doesn't exist or type mismatch.
// This method is useful when you want to gracefully handle missing keys.
func (m *Metadata) GetIntOrDefault(key string, defaultValue int) int {
	if val, ok := m.values[key].(int); ok {
		return val
	}
	return defaultValue
}

// SetInt stores an integer value. Returns itself for chaining.
func (m *Metadata) SetInt(key string, value int) *Metadata {
	m.values[key] = value
	return m
}

// GetBool retrieves a boolean value. Returns error if key doesn't exist or type mismatch.
func (m *Metadata) GetBool(key string) (bool, error) {
	val, ok := m.values[key]
	if !ok {
		return false, fmt.Errorf("key '%s' not found", key)
	}
	b, ok := val.(bool)
	if !ok {
		return false, fmt.Errorf("key '%s' is not bool, got %T", key, val)
	}
	return b, nil
}

// GetBoolOrDefault retrieves a boolean value, returns default if key doesn't exist or type mismatch.
func (m *Metadata) GetBoolOrDefault(key string, defaultValue bool) bool {
	if val, ok := m.values[key].(bool); ok {
		return val
	}
	return defaultValue
}

// SetBool stores a boolean value. Returns itself for chaining.
func (m *Metadata) SetBool(key string, value bool) *Metadata {
	m.values[key] = value
	return m
}

// GetString retrieves a string value. Returns error if key doesn't exist or type mismatch.
func (m *Metadata) GetString(key string) (string, error) {
	val, ok := m.values[key]
	if !ok {
		return "", fmt.Errorf("key '%s' not found", key)
	}
	s, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("key '%s' is not string, got %T", key, val)
	}
	return s, nil
}

// GetStringOrDefault retrieves a string value, returns default if key doesn't exist or type mismatch.
func (m *Metadata) GetStringOrDefault(key string, defaultValue string) string {
	if val, ok := m.values[key].(string); ok {
		return val
	}
	return defaultValue
}

// SetString stores a string value. Returns itself for chaining.
func (m *Metadata) SetString(key string, value string) *Metadata {
	m.values[key] = value
	return m
}

// GetFloat64 retrieves a float64 value. Returns error if key doesn't exist or type mismatch.
func (m *Metadata) GetFloat64(key string) (float64, error) {
	val, ok := m.values[key]
	if !ok {
		return 0, fmt.Errorf("key '%s' not found", key)
	}
	f, ok := val.(float64)
	if !ok {
		return 0, fmt.Errorf("key '%s' is not float64, got %T", key, val)
	}
	return f, nil
}

// GetFloat64OrDefault retrieves a float64 value, returns default if key doesn't exist or type mismatch.
func (m *Metadata) GetFloat64OrDefault(key string, defaultValue float64) float64 {
	if val, ok := m.values[key].(float64); ok {
		return val
	}
	return defaultValue
}

// SetFloat64 stores a float64 value. Returns itself for chaining.
func (m *Metadata) SetFloat64(key string, value float64) *Metadata {
	m.values[key] = value
	return m
}

// HasKey checks if a key exists.
func (m *Metadata) HasKey(key string) bool {
	_, ok := m.values[key]
	return ok
}

// Delete removes a key from the metadata.
func (m *Metadata) Delete(key string) {
	delete(m.values, key)
}

// Clear removes all keys from the metadata.
func (m *Metadata) Clear() {
	m.values = make(map[string]interface{})
}

// Size returns the number of keys in the metadata.
func (m *Metadata) Size() int {
	return len(m.values)
}

// Keys returns all key names in the metadata.
func (m *Metadata) Keys() []string {
	keys := make([]string, 0, len(m.values))
	for k := range m.values {
		keys = append(keys, k)
	}
	return keys
}

// Clone creates an independent copy of the metadata.
func (m *Metadata) Clone() *Metadata {
	newValues := make(map[string]interface{}, len(m.values))
	for k, v := range m.values {
		newValues[k] = v
	}
	return &Metadata{values: newValues}
}

// IncrementInt increments an integer value by delta. Returns the new value.
// If key doesn't exist, it starts from 0.
func (m *Metadata) IncrementInt(key string, delta int) int {
	current := m.GetIntOrDefault(key, 0)
	newValue := current + delta
	m.SetInt(key, newValue)
	return newValue
}

// DecrementInt decrements an integer value by delta. Returns the new value.
// If key doesn't exist, it starts from 0.
func (m *Metadata) DecrementInt(key string, delta int) int {
	return m.IncrementInt(key, -delta)
}

// Merge combines another metadata's values into this one.
// Existing keys will be overwritten.
func (m *Metadata) Merge(other *Metadata) *Metadata {
	if other == nil {
		return m
	}
	for k, v := range other.values {
		m.values[k] = v
	}
	return m
}

// ToMap returns a copy of the underlying map (for read-only use).
func (m *Metadata) ToMap() map[string]interface{} {
	result := make(map[string]interface{}, len(m.values))
	for k, v := range m.values {
		result[k] = v
	}
	return result
}

// Common errors
var (
	ErrKeyNotFound   = errors.New("key not found")
	ErrTypeMismatch  = errors.New("type mismatch")
)