# pkg/util/metadata - Metadata 类型安全动态数据容器

## 概述

`pkg/util/metadata` 是一个类型安全的键值存储容器，用于解决以下问题：

1. `Context.Data` 为 `interface{}` 类型太过简陋，缺乏类型安全
2. `Player` 中存有 `FireCounter`、`ChargeCount` 等与核心实体无关的数据
3. 未来可以扩展更多动态数据

通过 Go 的**匿名嵌入（Struct Embedding）**特性，让 `Context`、`Player` 等结构体直接继承 `Metadata` 的所有方法。

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

### 类型安全读取

| 方法 | 说明 |
|------|------|
| `GetInt(key string) int` | 安全获取整型，不存在返回 0 |
| `GetIntOrDefault(key, default) int` | 安全获取整型，带默认值 |
| `GetBool(key string) bool` | 安全获取布尔值 |
| `GetString(key string) string` | 安全获取字符串 |
| `GetFloat64(key string) float64` | 安全获取浮点数 |
| `Get(key string) (interface{}, bool)` | 获取原始值 |

### 类型安全写入（链式调用）

| 方法 | 说明 |
|------|------|
| `SetInt(key, value) *Metadata` | 设置整型值 |
| `SetBool(key, value) *Metadata` | 设置布尔值 |
| `SetString(key, value) *Metadata` | 设置字符串 |
| `Set(key, value) *Metadata` | 设置任意类型值 |

### 辅助方法

| 方法 | 说明 |
|------|------|
| `HasKey(key) bool` | 检查键是否存在 |
| `Delete(key)` | 删除键 |
| `Clear()` | 清空所有键 |
| `Keys() []string` | 返回所有键名 |
| `Size() int` | 返回键数量 |
| `Clone() *Metadata` | 克隆（独立副本） |
| `IncrementInt(key, delta) int` | 递增整型值 |
| `DecrementInt(key, delta) int` | 递减整型值 |
| `Merge(other *Metadata) *Metadata` | 合并另一个 Metadata |

---

## 使用示例

### 基本使用

```go
m := util.NewMetadata()

// 设置值
m.SetInt("count", 10)
m.SetString("name", "test")
m.SetBool("active", true)

// 链式调用
m.SetInt("turn", 1).SetString("event", "fog")

// 获取值
count := m.GetInt("count")          // 10
name := m.GetString("name")         // "test"
active := m.GetBool("active")       // true

// 带默认值获取
val := m.GetIntOrDefault("missing", 5)  // 5

// 递增
m.IncrementInt("counter", 1)  // 返回递增后的值
```

### 嵌入到结构体

```go
import "github.com/b1tAction/Fated/pkg/util"

type Player struct {
    UserID   string
    Faction  Faction
    // ... 其他核心字段
    *util.Metadata  // 匿名嵌入
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

### 向后兼容方法

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

---

## 已迁移的数据

### Player

| 原字段 | Metadata 键 | 便捷方法 |
|--------|------------|---------|
| `ChargeCount int` | `charge_count` | `GetChargeCount/SetChargeCount` |
| `FireCounter int` | `fire_counter` | `GetFireCounter/SetFireCounter` |

### Context

| 原字段 | Metadata 键 | 便捷方法 |
|--------|------------|---------|
| `Data interface{}` | `data` | `WithData/GetData` |

---

## 设计理念

### 为什么使用 Metadata

1. **DRY 原则**：类型转换、安全校验集中在 `Metadata`，一处添加方法多处受益。
2. **扁平序列化**：JSON 序列化干净，前端解析无压力。
3. **架构统一**：`Context`（瞬态）、`Player`（持久）用同一组件管理动态数据。
4. **灵活扩展**：未来 Cell 的 `FellDown`、`Interrupted` 等状态也可迁移。

### 键名约定

- 使用 `snake_case` 格式
- 例如：`fire_counter`、`charge_count`、`turn_count`

---

## 测试覆盖

`pkg/util/metadata_test.go` 包含全面测试：

- 初始化测试
- Set/Get 基本操作
- 类型安全读取测试
- HasKey/Delete/Clear 测试
- Clone 独立性测试
- IncrementInt/DecrementInt 测试
- Merge 测试
- 链式调用测试

---

## 相关文档

- [core.md](./core.md) - Player/Buff 结构体定义
- [event_bus_system.md](./event_bus_system.md) - Context 使用方式