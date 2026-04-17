# internal/core - Core Data Structures

核心数据结构包，定义游戏的基础类型和数据结构。

## 概述

internal/core 是《派乐代》游戏的核心数据模块，采用**Direct Import模式**设计，支持独立子包导入。

## 包结构

```
internal/core/
├── buff/               # Buff 子包
│   ├── buff.go         # Buff、BuffDefinition、BuffHandlerConfig、Registry
│   └── init.go         # init() + registerAllBuffs() + handlers
├── event/              # Event 子包
│   ├── event.go        # Event、EventDefinition、EventHandlerConfig、Registry
│   └── init.go         # init() + registerAllEvents() + handlers
├── item/               # Item 子包
│   ├── item.go         # Item、ItemDefinition、ItemHandlerConfig、Registry
│   └── init.go         # init() + registerAllItems() + handlers
├── player.go           # Player结构（HP/LP/Buffs/Items/Metadata）
├── faction.go          # 阵营辅助函数（GetFactionNames等）
└── init.go             # 统一入口（重导出函数和类型）
```

**注意**：所有枚举类型（BuffType、EventType、ItemType、Faction）定义在 `pkg/constants` 包。

## Direct Import 模式

```go
// 只需要 Buff（自动初始化）
import "github.com/b1tAction/paradiced/internal/core/buff"
import "github.com/b1tAction/paradiced/pkg/constants"
buff.GetBuffDefinition(constants.BuffTypeFire)
buff.GetBuffHandlerConfig(constants.BuffTypeFire)

// 只需要 Event（自动初始化）
import "github.com/b1tAction/paradiced/internal/core/event"
import "github.com/b1tAction/paradiced/pkg/constants"
event.GetEventDefinition(constants.EventTypeHerb)
event.GetEventHandlerConfig(constants.EventTypeHerb)

// 需要完整游戏逻辑（自动初始化所有子包）
import "github.com/b1tAction/paradiced/internal/core"
import "github.com/b1tAction/paradiced/pkg/constants"
core.GetBuffDefinition(constants.BuffTypeFire)
core.GetBuffHandlerConfig(constants.BuffTypeFire)
```

## 架构原则

### 单一职责原则

每个系统分离为两个结构：

| 系统 | Definition (静态元数据) | HandlerConfig (执行配置) |
|------|------------------------|------------------------|
| Buff | BuffDefinition | BuffHandlerConfig |
| Event | EventDefinition | EventHandlerConfig |
| Item | ItemDefinition | ItemHandlerConfig |

**Definition** 只包含：
- Type（类型标识，来自 constants 包）
- Eval（评估分数）
- EnglishName/Name/Desc（显示信息）
- Duration（持续时间，仅Buff）

**HandlerConfig** 包含：
- Phases/Phase（触发时机）
- Priority（执行优先级）
- NeedConfirm（是否需要确认）
- Handler（效果处理函数）

### 效果逻辑统一由 Handler 管理

所有效果逻辑（HP/LP修改、赋予Buff、特殊效果）都通过 EffectHandler 实现：

```go
type EffectHandler func(phase constants.Phase, ctx *event.Context)
```

Handler 通过 `ctx.SetInt/SetBool/SetString` 信号意图，engine层通过 Action 系统执行。

## 依赖关系

```
constants/  ← 独立，无外部依赖（包含所有枚举类型）
buff/       ← import constants, event, handler
event/      ← import constants, event, handler
item/       ← import constants, event, handler
core/       ← import buff, event, item（重导出函数）
engine/     ← import core, action
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

### Player (玩家)

```go
type Player struct {
    ID          id.PlayerID       // 玩家唯一标识（UUID v7 + prefix）
    Faction     constants.Faction // 阵营
    Position    int               // 当前位置
    HP          int               // 血量（默认最大6）
    LP          int               // 幸运值（范围0~8，影响随机事件）
    Inventory   []*item.Item      // 道具栏
    ActiveBuffs []*buff.Buff      // 持续状态
    IsDead      bool              // 是否死亡
    SkipTurn    bool              // 是否跳过回合
    *util.Metadata               // 类型安全的动态数据容器
}
```

**Metadata 契约**：详见 [doc/metadata/player.md](../metadata/player.md)。

## 统一注册表

每个子包有独立的 Registry，在各自的 init.go 中初始化：

```go
// buff/init.go
var GlobalBuffRegistry *BuffRegistry
func init() {
    GlobalBuffRegistry = NewBuffRegistry()
    registerAllBuffs()
}

// 注册签名（Definition + HandlerConfig）
GlobalBuffRegistry.RegisterBuff(def, config)
```

### 访问方式

```go
import "github.com/b1tAction/paradiced/pkg/constants"

// 获取静态元数据
buff.GetBuffDefinition(constants.BuffTypeFire)

// 获取执行配置（含Handler）
config := buff.GetBuffHandlerConfig(constants.BuffTypeFire)
config.Phases           // 触发Phase列表
config.Priority         // 执行优先级
config.Handler          // 效果处理函数
config.HasPhase(phase)  // 检查是否支持某Phase
```

## Buff 系统

### BuffDefinition

```go
type BuffDefinition struct {
    Type        constants.BuffType   `json:"type"`
    Eval        constants.Evaluation `json:"evaluation"`
    EnglishName string               `json:"english_name"`
    Name        string               `json:"name"`
    Desc        string               `json:"desc"`
    Duration    int                  `json:"duration"`      // -1 表示永久
}
```

### BuffHandlerConfig

```go
type BuffHandlerConfig struct {
    Phases      []constants.Phase    `json:"phases"`        // 触发Phase列表
    Priority    int                  `json:"priority"`      // 执行优先级
    NeedConfirm bool                 `json:"need_confirm"`  // 是否需要用户确认
    Handler     EffectHandler        `json:"-"`             // 效果处理函数
}
```

### Buff 实例

```go
type Buff struct {
    Type            constants.BuffType
    ID              id.BuffID
    Duration        int
    Charge          int
    SubscriptionIDs []id.SubscriptionID  // 支持多Phase订阅
}
```

## Event 系统

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

### EventHandlerConfig

```go
type EventHandlerConfig struct {
    Handler EffectHandler  `json:"-"`  // 效果处理函数
}
```

## Item 系统

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

### ItemHandlerConfig

```go
type ItemHandlerConfig struct {
    Phase       constants.Phase      `json:"phase"`         // 可使用时机
    Priority    int                  `json:"priority"`      // 执行优先级
    NeedConfirm bool                 `json:"need_confirm"`  // 是否需要用户确认
    Handler     EffectHandler        `json:"-"`             // 效果处理函数
}
```

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

### 3. 阵营特性

```go
import "github.com/b1tAction/paradiced/pkg/constants"

// 朱雀玩家初始携带离火 Buff
if config.Faction == constants.FactionZhuQue {
    player.AddBuff(buff.NewBuff(constants.BuffTypeFire, -1))
}
```

### 4. Metadata 方法

```go
// 阵营特定属性
player.GetChargeCount()     // 青龙/玄武充能数
player.SetChargeCount(count)
player.IncrementChargeCount()

player.GetFireCounter()     // 朱雀火计数
player.SetFireCounter(count)
player.IncrementFireCounter()
```

## 测试覆盖

| 子包 | 测试文件 | 覆盖内容 |
|------|---------|---------|
| buff/ | buff_test.go | Buff实例、BuffDefinition、BuffHandlerConfig |
| event/ | event_test.go | EventDefinition、EventHandlerConfig |
| item/ | item_test.go | Item实例、ItemDefinition、ItemHandlerConfig |
| core | player_test.go | Player 创建、数值逻辑、Buff/道具管理 |

## 新增内容流程

添加新的 Buff/Event/Item 需要：

1. 在 `pkg/constants` 包添加枚举常量
2. 在子包 init.go 中注册 Definition + HandlerConfig

```go
// constants/buff.go - 添加枚举
const (
    ...
    BuffTypeNewBuff BuffType = "new_buff"
)

// buff/init.go - 注册定义和配置
GlobalBuffRegistry.RegisterBuff(&BuffDefinition{
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
- [doc/metadata/player.md](../metadata/player.md) - Player.Metadata 契约