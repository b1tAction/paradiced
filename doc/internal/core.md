# internal/core - Core Data Structures

核心数据结构包，定义游戏的基础类型。

## 概述

internal/core 是《派乐代》游戏的纯数据层，只定义结构体和静态元数据。

**架构原则**：
- core 层不依赖任何 internal 包
- Handler、Registry、注册逻辑完全在 engine 层
- Definition（静态元数据）在 core，HandlerConfig（执行配置）在 engine

## 包结构

```
internal/core/
├── buff.go       # Buff 结构 + BuffDefinition（静态元数据）
├── item.go       # Item 结构 + ItemDefinition（静态元数据）
├── game_event.go # GameEvent 结构 + EventDefinition（静态元数据）
├── player.go     # Player 结构（HP/LP/Buffs/Items/Metadata）
├── faction.go    # 阵营辅助函数（GetFactionNames等）
└── player_test.go # Player 单元测试
```

**注意**：
- 所有枚举类型（BuffType、EventType、ItemType、Faction）定义在 `pkg/constants` 包
- 所有 Handler、Registry 定义在 `internal/engine` 包

## 依赖关系

```
pkg/constants → 无依赖
pkg/id       → 无依赖
pkg/util     → 无依赖

internal/core → pkg/constants, pkg/id, pkg/util（纯数据层）
internal/event → pkg/constants, pkg/id, pkg/util, internal/core
internal/engine → internal/core, internal/event, pkg/*（Registry + Handler 层）
```

## 数据类型（定义在 pkg/constants）

### Evaluation (评分系统)

定义在 `pkg/constants/evaluation.go`：

```go
type Evaluation int  // 0-100

// 分类边界
EvaluationBadThreshold     = 40   // 恶性上限（≤40）
EvaluationNeutralThreshold = 65   // 中性上限（≤65）
// Evaluation > 65 为良性
```

### Faction (阵营)

定义在 `pkg/constants/faction.go`：

```go
type Faction string

const (
    FactionQingLong Faction = "qing_long" // 青龙（东方）- 行迹
    FactionZhuQue   Faction = "zhu_que"   // 朱雀（南方）- 离火
    FactionBaiHu    Faction = "bai_hu"    // 白虎（西方）- 劫运
    FactionXuanWu   Faction = "xuan_wu"   // 玄武（北方）- 鎮厄
)

func (f Faction) SnakeCase() string    // 返回 snake_case 值
func (f Faction) GetChineseName() string // 返回中文名
func (f Faction) GetSkillName() string   // 返回技能名
func (f Faction) GetSkillDesc() string   // 返回技能描述
```

### BuffType/EventType/ItemType

分别定义在 `pkg/constants/buff.go`、`event.go`、`item.go`。

## Definition 结构（静态元数据）

### BuffDefinition

```go
type BuffDefinition struct {
    Type        constants.BuffType   `json:"type"`
    Eval        constants.Evaluation `json:"evaluation"`
    EnglishName string               `json:"english_name"`
    Name        string               `json:"name"`
    Desc        string               `json:"desc"`
    Duration    int                  `json:"duration"`      // -1 表示永久
    Hidden      bool                 `json:"hidden"`        // 隐藏Buff：无抽签，不发送给客户端
}
```

### EventDefinition

```go
type EventDefinition struct {
    Type        constants.EventType   `json:"type"`
    Eval        constants.Evaluation  `json:"evaluation"`
    EnglishName string                `json:"english_name"`
    Name        string                `json:"name"`
    Desc        string                `json:"desc"`
}
```

### ItemDefinition

```go
type ItemDefinition struct {
    Type        constants.ItemType   `json:"type"`
    Eval        constants.Evaluation `json:"evaluation"`
    EnglishName string               `json:"english_name"`
    Name        string               `json:"name"`
    Desc        string               `json:"desc"`
    TargetSelf  bool                 `json:"target_self"`   // 可对自己使用
    TargetOther bool                 `json:"target_other"`  // 可对他人使用
    Range       int                  `json:"range"`         // 有效范围
}
```

## 实例结构

### Buff

```go
type Buff struct {
    Type         constants.BuffType `json:"type"`
    ID           id.BuffID          `json:"id"`
    Duration     int                `json:"duration"`       // 剩余回合数，-1表示永久
    tickEligible bool               // 是否应在下回合结束时tick（内部状态）
    *util.Metadata `json:"metadata"` // Per-buff状态存储（如everyNTurns计数器）
}

func NewBuff(buffType constants.BuffType, duration int) *Buff
func NewBuffWithID(buffType constants.BuffType, buffID id.BuffID, duration int) *Buff
func (b *Buff) IsActive() bool
func (b *Buff) TickDuration() bool
```

**Buff.Metadata 字段契约**（内部使用，不发送给客户端）：

| 字段 | 类型 | 来源 | 用途 |
|------|------|------|------|
| `buff_turn_counter` | int | everyNTurns handler | 甘霖/腐化每N回合触发计数器 |

### Item

```go
type Item struct {
    Type          constants.ItemType `json:"type"`
    ID            id.ItemID          `json:"id"`
    SubscriptionID string            `json:"subscription_id"`
}

func NewItem(itemType constants.ItemType) *Item
```

### GameEvent

```go
type GameEvent struct {
    Type constants.EventType `json:"type"`
}

func NewGameEvent(eventType constants.EventType) *GameEvent
```

### Player

```go
type Player struct {
    ID          id.PlayerID       `json:"id"`           // 玩家唯一标识
    Faction     constants.Faction `json:"faction"`      // 阵营
    Position    int               `json:"position"`     // 当前位置
    HP          int               `json:"hp"`           // 血量（默认最大6）
    LP          int               `json:"lp"`           // 幸运值（范围0~8）
    Inventory   []*Item           `json:"inventory"`    // 道具栏
    ActiveBuffs []*Buff           `json:"active_buffs"` // 持续状态
    IsDead      bool              `json:"is_dead"`      // 是否死亡
    SkipTurn    bool              `json:"skip_turn"`    // 是否跳过回合
    *util.Metadata                 `json:"metadata"`     // 类型安全的动态数据容器
}

func NewPlayer(config PlayerConfig) *Player
```

**Metadata 契约**：详见 [doc/metadata/player.md](../metadata/player.md)。

## 核心功能

### 1. 数值逻辑

```go
// 扣血（隐匿免疫）
player.ApplyDamage(amount) error

// 回血
player.Heal(amount) error

// 修改幸运值（范围限制 0~8）
player.ModifyLP(amount)
```

### 2. Buff 管理

```go
player.AddBuff(buff)                    // 添加 Buff
player.RemoveBuff(buffType)             // 移除指定类型 Buff
player.HasBuff(constants.BuffTypeFire)  // 检查是否有 Buff
player.GetBuff(constants.BuffTypeFire)  // 获取 Buff 实例
player.TickBuffs()                      // 更新持续时间，返回失效的 Buff
```

### 3. Metadata 方法

```go
// 阵营特定属性
player.GetChargeCount()     // 青龙/玄武充能数
player.SetChargeCount(count)
player.IncrementChargeCount()

player.GetFireCounter()     // 朱雀火计数
player.SetFireCounter(count)
player.IncrementFireCounter()
```

## Registry 和 Handler（在 engine 层）

Definition 在 core 层定义，但 Registry 和 Handler 在 engine 层：

```go
// internal/engine/buff_registry.go
type BuffHandlerConfig struct {
    Phases      []constants.Phase
    Priority    int
    NeedConfirm bool
    Handler     EffectHandler  // 定义在 engine 层
}

type BuffRegistry struct {
    defs    map[constants.BuffType]*core.BuffDefinition  // 使用 core 定义
    configs map[constants.BuffType]*BuffHandlerConfig    // engine 自己的配置
}

// 获取 Definition（来自 core）
func GetBuffDefinition(bt constants.BuffType) *core.BuffDefinition

// 获取 HandlerConfig（来自 engine）
func GetBuffHandlerConfig(bt constants.BuffType) *BuffHandlerConfig
```

## 新增内容流程

添加新的 Buff/Event/Item 需要：

1. 在 `pkg/constants` 包添加枚举常量
2. 在 `internal/core` 包添加 Definition 结构（如果需要新字段）
3. 在 `internal/engine` 包的 registry.go 中注册 Definition + HandlerConfig

```go
// pkg/constants/buff.go - 添加枚举
const (
    ...
    BuffTypeNewBuff BuffType = "new_buff"
)

// internal/engine/buff_registry.go - 注册定义和配置
GlobalBuffRegistry.RegisterBuff(&core.BuffDefinition{
    Type:        constants.BuffTypeNewBuff,
    Eval:        constants.EvaluationGood,
    EnglishName: "NewBuff",
    Name:        "新Buff",
    Desc:        "新Buff描述",
    Duration:    3,
}, &BuffHandlerConfig{
    Phases:      []constants.Phase{constants.PhaseBeforeTurn},
    Priority:    50,
    NeedConfirm: false,
    Handler:     createModifyLPHandler(1),
})
```

## 相关文档

- [pkg/constants/README.md](../../pkg/constants/README.md) - 统一枚举类型
- [internal/engine/README.md](../engine/README.md) - Registry + Handler 层
- [doc/metadata/player.md](../metadata/player.md) - Player.Metadata 契约