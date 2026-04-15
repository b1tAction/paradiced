# pkg/util/metadata - 类型安全动态数据容器

## 概述

`pkg/util/metadata` 是一个类型安全的键值存储容器，解决了以下问题：

1. `Context.Data` 作为 `interface{}` 太过简陋，缺乏类型安全
2. `Player` 包含 `FireCounter`、`ChargeCount` 等与核心实体不相关的数据
3. 未来需要更灵活的动态数据扩展

通过 Go 的**匿名嵌套（Struct Embedding）**特性，`Context`、`Player` 等结构体可以直接继承 `Metadata` 的所有方法。

---

## 文件结构

```
pkg/util/
├── metadata.go      # Metadata 核心实现
├── metadata_test.go # 测试文件
```

---

## Metadata 结构体

```go
type Metadata struct {
    values map[string]interface{}
}

func NewMetadata() *Metadata
```

---

## 核心方法

### 类型安全读取（返回错误）

| 方法 | 描述 |
|--------|-------------|
| `GetInt(key string) (int, error)` | 获取整数，键不存在或类型不匹配时返回错误 |
| `GetBool(key string) (bool, error)` | 获取布尔值，键不存在或类型不匹配时返回错误 |
| `GetString(key string) (string, error)` | 获取字符串，键不存在或类型不匹配时返回错误 |
| `GetFloat64(key string) (float64, error)` | 获取浮点数，键不存在或类型不匹配时返回错误 |
| `Get(key string) (interface{}, bool)` | 获取原始值，不存在时返回 (nil, false) |

### 类型安全读取（带默认值 - 无错误）

| 方法 | 描述 |
|--------|-------------|
| `GetIntOrDefault(key, default) int` | 获取整数，键不存在或类型不匹配时返回默认值 |
| `GetBoolOrDefault(key, default) bool` | 获取布尔值，键不存在或类型不匹配时返回默认值 |
| `GetStringOrDefault(key, default) string` | 获取字符串，键不存在或类型不匹配时返回默认值 |
| `GetFloat64OrDefault(key, default) float64` | 获取浮点数，键不存在或类型不匹配时返回默认值 |

### 类型安全写入（链式调用）

| 方法 | 描述 |
|--------|-------------|
| `SetInt(key, value) *Metadata` | 设置整数值 |
| `SetBool(key, value) *Metadata` | 设置布尔值 |
| `SetString(key, value) *Metadata` | 设置字符串值 |
| `Set(key, value) *Metadata` | 设置任意类型值 |

### JSON 序列化支持

| 方法 | 描述 |
|--------|-------------|
| `MarshalJSON() ([]byte, error)` | JSON 序列化，输出 `map[string]interface{}` 格式 |
| `UnmarshalJSON(data []byte) error` | JSON 反序列化，自动重建内部 map |
| `FromMap(m map[string]interface{}) *Metadata` | 从 map 创建 Metadata（用于反序列化辅助） |

**JSON 序列化特性**：
- 序列化后所有数值会转换为 JSON 的 `float64` 格式
- `GetIntOrDefault` 已处理此情况，自动将 `float64` 转回 `int`
- 支持嵌套结构（如 LogEntry.Metadata 字段）

### 辅助方法

| 方法 | 描述 |
|--------|-------------|
| `HasKey(key) bool` | 检查键是否存在 |
| `Delete(key)` | 删除键 |
| `Clear()` | 清空所有键 |
| `Keys() []string` | 返回所有键名 |
| `Size() int` | 返回键数量 |
| `Clone() *Metadata` | 克隆（独立副本） |
| `IncrementInt(key, delta) int` | 递增整数值 |
| `DecrementInt(key, delta) int` | 递减整数值 |
| `Merge(other *Metadata) *Metadata` | 合并另一个 Metadata |

---

## 使用示例

### 基本用法

```go
m := util.NewMetadata()

// 设置值
m.SetInt("count", 10)
m.SetString("name", "test")
m.SetBool("active", true)

// 链式调用
m.SetInt("turn", 1).SetString("event", "fog")

// 获取值（带错误处理）
count, err := m.GetInt("count")
if err != nil {
    // 处理错误：键不存在或类型不匹配
}

// 或使用 GetIntOrDefault 进行优雅处理
count := m.GetIntOrDefault("count", 0)          // 10
name := m.GetStringOrDefault("name", "")        // "test"
active := m.GetBoolOrDefault("active", false)   // true

// 键不存在时返回默认值
val := m.GetIntOrDefault("missing", 5)  // 5

// 递增
m.IncrementInt("counter", 1)  // 返回新值
```

### 嵌入到结构体中

```go
import "github.com/b1tAction/paradiced/pkg/util"

type Player struct {
    UserID   string
    Faction  Faction
    // ... 其他核心字段
    *util.Metadata  // 匿名嵌套
}

func NewPlayer(config PlayerConfig) *Player {
    return &Player{
        UserID:   config.UserID,
        Faction:  config.Faction,
        Metadata: util.NewMetadata(),
    }
}

// 使用示例
player := NewPlayer(config)
player.SetInt("fire_counter", 0)
player.IncrementInt("fire_counter", 1)
```

### 已知数据的便捷方法

对于已知用途的数据，可以添加便捷方法：

```go
// Player 便捷方法
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

### JSON 序列化示例

```go
// 序列化
m := util.NewMetadata()
m.SetInt("hp", 100)
m.SetString("name", "player1")
m.SetBool("active", true)

jsonBytes, err := m.MarshalJSON()
// 输出: {"hp":100,"name":"player1","active":true}

// 反序列化
m2 := util.NewMetadata()
err := m2.UnmarshalJSON(jsonBytes)
hp := m2.GetIntOrDefault("hp", 0)  // 100（自动处理 float64→int 转换）
```

---

## 数据迁移示例

### Player

| 原字段 | Metadata 键名 | 便捷方法 |
|---------------|--------------|---------------------|
| `ChargeCount int` | `charge_count` | `GetChargeCount/SetChargeCount` |
| `FireCounter int` | `fire_counter` | `GetFireCounter/SetFireCounter` |

---

## 设计哲学

### 为什么使用 Metadata

1. **DRY 原则**：类型转换和安全检查集中在 `Metadata` 中，一处添加方法，多处受益。
2. **扁平序列化**：JSON 序列化干净整洁，前端解析轻松。
3. **架构统一**：`Context`（瞬态）和 `Player`（持久态）由同一组件管理。
4. **灵活扩展**：未来 Cell 状态如 `FellDown`、`Interrupted` 可以迁移到此。

### 键名命名约定

- 使用 `snake_case` 格式
- 示例：`fire_counter`、`charge_count`、`turn_count`

---

## Metadata 字段契约

**重要**：所有使用 `util.Metadata` 的类型的字段契约已迁移到独立文档。

详见：[doc/metadata/README.md](../metadata/README.md) - Metadata契约文档总览

| 类型 | 契约文档 | 可见性 |
|------|----------|--------|
| `gamelog.LogEntry.Metadata` | [doc/metadata/logentry.md](../metadata/logentry.md) | **客户端可见** |
| `core.Player.Metadata` | [doc/metadata/player.md](../metadata/player.md) | **客户端可见** |
| `event.Context.Metadata` | [doc/metadata/event_context.md](../metadata/event_context.md) | 内部 |
| `hsm.StateContext.Metadata` | [doc/metadata/hsm_context.md](../metadata/hsm_context.md) | 内部 |
| `action.ActionContext.Metadata` | [doc/metadata/action_context.md](../metadata/action_context.md) | 内部 |

### 新增 Metadata 字段时

1. **确定字段归属**：根据使用类型选择对应契约文档
2. **更新契约表格**：添加字段名、类型、用途说明
3. **客户端同步**（仅 LogEntry.Metadata）：更新 TypeScript 类型定义

---

## 测试覆盖

`pkg/util/metadata_test.go` 包含全面的测试：

- 初始化测试
- Set/Get 基本操作
- 类型安全读取测试（带错误处理）
- GetOrDefault 测试
- HasKey/Delete/Clear 测试
- Clone 独立性测试
- IncrementInt/DecrementInt 测试
- Merge 测试
- 链式调用测试
- JSON 序列化/反序列化测试

---

## 相关文档

- [doc/metadata/README.md](../metadata/README.md) - Metadata契约文档
- [pkg/gamelog/README.md](../../pkg/gamelog/README.md) - GameLog系统
- [internal/core/README.md](../../internal/core/README.md) - Player结构
- [pkg/event/README.md](../../pkg/event/README.md) - EventBus系统
- [internal/engine/hsm/README.md](../../internal/engine/hsm/README.md) - HSM状态机