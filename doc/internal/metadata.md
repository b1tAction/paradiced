# pkg/util/metadata - Type-Safe Dynamic Data Container

## Overview

`pkg/util/metadata` is a type-safe key-value storage container that solves these problems:

1. `Context.Data` as `interface{}` is too simplistic, lacking type safety
2. `Player` contains `FireCounter`, `ChargeCount` and other data unrelated to core entity
3. Future extensibility for more dynamic data

Through Go's **anonymous embedding (Struct Embedding)** feature, `Context`, `Player` and other structs directly inherit all methods from `Metadata`.

---

## File Structure

```
pkg/util/
├── metadata.go      # Metadata core implementation
├── metadata_test.go # Test file
```

---

## Metadata Struct

```go
type Metadata struct {
    values map[string]interface{}
}

func NewMetadata() *Metadata
```

---

## Core Methods

### Type-Safe Reading (Returns Error)

| Method | Description |
|--------|-------------|
| `GetInt(key string) (int, error)` | Gets integer, returns error if key not found or type mismatch |
| `GetBool(key string) (bool, error)` | Gets boolean, returns error if key not found or type mismatch |
| `GetString(key string) (string, error)` | Gets string, returns error if key not found or type mismatch |
| `GetFloat64(key string) (float64, error)` | Gets float64, returns error if key not found or type mismatch |
| `Get(key string) (interface{}, bool)` | Gets raw value, returns (nil, false) if not found |

### Type-Safe Reading (With Default - No Error)

| Method | Description |
|--------|-------------|
| `GetIntOrDefault(key, default) int` | Gets integer, returns default if key not found or type mismatch |
| `GetBoolOrDefault(key, default) bool` | Gets boolean, returns default if key not found or type mismatch |
| `GetStringOrDefault(key, default) string` | Gets string, returns default if key not found or type mismatch |
| `GetFloat64OrDefault(key, default) float64` | Gets float64, returns default if key not found or type mismatch |

### Type-Safe Writing (Chainable)

| Method | Description |
|--------|-------------|
| `SetInt(key, value) *Metadata` | Sets integer value |
| `SetBool(key, value) *Metadata` | Sets boolean value |
| `SetString(key, value) *Metadata` | Sets string value |
| `Set(key, value) *Metadata` | Sets any type value |

### Utility Methods

| Method | Description |
|--------|-------------|
| `HasKey(key) bool` | Checks if key exists |
| `Delete(key)` | Deletes key |
| `Clear()` | Clears all keys |
| `Keys() []string` | Returns all key names |
| `Size() int` | Returns key count |
| `Clone() *Metadata` | Clones (independent copy) |
| `IncrementInt(key, delta) int` | Increments integer value |
| `DecrementInt(key, delta) int` | Decrements integer value |
| `Merge(other *Metadata) *Metadata` | Merges another Metadata |

---

## Usage Examples

### Basic Usage

```go
m := util.NewMetadata()

// Set values
m.SetInt("count", 10)
m.SetString("name", "test")
m.SetBool("active", true)

// Chain calls
m.SetInt("turn", 1).SetString("event", "fog")

// Get values with error handling
count, err := m.GetInt("count")
if err != nil {
    // Handle error: key not found or type mismatch
}

// Or use GetIntOrDefault for graceful handling
count := m.GetIntOrDefault("count", 0)          // 10
name := m.GetStringOrDefault("name", "")        // "test"
active := m.GetBoolOrDefault("active", false)   // true

// Default value for missing key
val := m.GetIntOrDefault("missing", 5)  // 5

// Increment
m.IncrementInt("counter", 1)  // Returns new value
```

### Embedding in Struct

```go
import "github.com/b1tAction/Fated/pkg/util"

type Player struct {
    UserID   string
    Faction  Faction
    // ... other core fields
    *util.Metadata  // Anonymous embedding
}

func NewPlayer(config PlayerConfig) *Player {
    return &Player{
        UserID:   config.UserID,
        Faction:  config.Faction,
        Metadata: util.NewMetadata(),
    }
}

// Usage example
player := NewPlayer(config)
player.SetInt("fire_counter", 0)
player.IncrementInt("fire_counter", 1)
```

### Convenience Methods for Known Data

For data with known purposes, add convenience methods:

```go
// Player convenience methods
func (p *Player) GetFireCounter() int {
    return p.GetIntOrDefault("fire_counter", 0)
}

func (p *Player) SetFireCounter(count int) {
    p.SetInt("fire_counter", count)
}

func (p *Player) IncrementFireCounter() int {
    return p.IncrementInt("fire_counter", 1)
}
```

---

## Migrated Data

### Player

| Original Field | Metadata Key | Convenience Methods |
|---------------|--------------|---------------------|
| `ChargeCount int` | `charge_count` | `GetChargeCount/SetChargeCount` |
| `FireCounter int` | `fire_counter` | `GetFireCounter/SetFireCounter` |

### Context

| Original Field | Metadata Key | Convenience Methods |
|---------------|--------------|---------------------|
| `Data interface{}` | `data` | `WithData/GetData` |

---

## Design Philosophy

### Why Use Metadata

1. **DRY Principle**: Type conversion and safety checks centralized in `Metadata`, one place to add methods benefits all.
2. **Flat Serialization**: JSON serialization is clean, frontend parsing is easy.
3. **Unified Architecture**: `Context` (transient) and `Player` (persistent) managed by same component.
4. **Flexible Extension**: Future Cell states like `FellDown`, `Interrupted` can be migrated.

### Key Naming Convention

- Use `snake_case` format
- Examples: `fire_counter`, `charge_count`, `turn_count`

---

## Test Coverage

`pkg/util/metadata_test.go` includes comprehensive tests:

- Initialization tests
- Set/Get basic operations
- Type-safe reading tests (with error handling)
- GetOrDefault tests
- HasKey/Delete/Clear tests
- Clone independence tests
- IncrementInt/DecrementInt tests
- Merge tests
- Chainable calls tests

---

## Related Documentation

- [core.md](./core.md) - Player/Buff struct definitions
- [event_bus_system.md](./event_bus_system.md) - Context usage