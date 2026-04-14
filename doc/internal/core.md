# internal/core - Core Data Structures

核心数据结构包，定义游戏的基础类型和数据结构。

## 概述

internal/core 是《命运骰子》游戏的核心数据模块，采用**Direct Import模式**设计，支持独立子包导入。

## 包结构

```
internal/core/
├── types/              # 共享基础类型（无依赖）
│   ├── evaluation.go   # 评分系统（0-100）
│   └── special_effect.go # 特殊效果枚举
├── buff/               # Buff 子包
│   ├── buff.go         # BuffType、Buff、BuffDefinition、Registry
│   └── init.go         # init() + registerAllBuffs() + handlers
├── event/              # Event 子包
│   ├── event.go        # EventType、EventDefinition、Registry
│   └── init.go         # init() + registerAllEvents()
├── item/               # Item 子包
│   ├── item.go         # ItemType、Item、ItemDefinition、Registry
│   └── init.go         # init() + registerAllItems()
├── player.go           # Player结构（HP/LP/Buffs/Items）
├── faction.go          # 阵营定义（四神兽）
└── init.go             # 统一入口（重导出所有类型）
```

## Direct Import 模式

```go
// 只需要 Buff（自动初始化）
import "github.com/b1tAction/fated/internal/core/buff"
buff.GetBuffDefinition(buff.BuffTypeFire)

// 只需要 Event（自动初始化）
import "github.com/b1tAction/fated/internal/core/event"
event.GetEventDefinition(event.EventTypeHerb)

// 需要完整游戏逻辑（自动初始化所有子包）
import "github.com/b1tAction/fated/internal/core"
core.GetBuffDefinition(core.BuffTypeFire)
```

## 依赖关系

```
types/    ← 独立，无外部依赖
buff/     ← import types
event/    ← import buff, types (EventDefinition.BuffType)
item/     ← import buff, types (ItemDefinition.BuffType)
core/     ← import buff, event, item, types（重导出）
engine/   ← import core
```

## 数据类型

### Evaluation (评分系统)

```go
type Evaluation int  // 0-100

// 分类边界
EvaluationBadThreshold     = 40   // 恶性上限（≤40）
EvaluationNeutralThreshold = 65   // 中性上限（≤65）
// Evaluation > 65 为良性

// LP 范围限制
// LP 最大值为 8，最小值为 0
```

### Faction (阵营)

```go
type Faction int

const (
    FactionQingLong Faction = iota // 青龙（东方）- 行迹
    FactionZhuQue                  // 朱雀（南方）- 离火
    FactionBaiHu                   // 白虎（西方）- 劫运
    FactionXuanWu                  // 玄武（北方）- 鎮厄
)
```

### Player (玩家)

```go
type Player struct {
    UserID      string     // 玩家UUID
    Faction     Faction    // 阵营
    Position    int        // 当前位置
    HP          int        // 血量（默认最大6）
    LP          int        // 幸运值（范围0~8，影响随机事件）
    Inventory   []*Item    // 道具栏
    ActiveBuffs []*Buff    // 持续状态
    IsDead      bool       // 是否死亡
    SkipTurn    bool       // 是否跳过回合
    *util.Metadata          // 类型安全的动态数据容器
}

// Metadata 存储的键名约定：
// - "charge_count": 充能计数（青龙/玄武阵营）
// - "fire_counter": 离火计数（朱雀阵营）
// - 其他自定义数据可通过 SetInt/SetString 等方法添加

// 便捷方法（向后兼容）：
func (p *Player) GetChargeCount() int
func (p *Player) SetChargeCount(count int)
func (p *Player) GetFireCounter() int
func (p *Player) SetFireCounter(count int)
func (p *Player) IncrementFireCounter() int
```

## 统一注册表

每个子包有独立的 Registry，在各自的 init.go 中初始化：

```go
// buff/init.go
var GlobalBuffRegistry *BuffRegistry
func init() {
    GlobalBuffRegistry = NewBuffRegistry()
    registerAllBuffs()
}

// event/init.go
var GlobalEventRegistry *EventRegistry
func init() {
    GlobalEventRegistry = NewEventRegistry()
    registerAllEvents()
}

// item/init.go
var GlobalItemRegistry *ItemRegistry
func init() {
    GlobalItemRegistry = NewItemRegistry()
    registerAllItems()
}
```

core/init.go 提供 CombinedRegistry 用于向后兼容：

```go
// 统一入口，重导出所有类型
type CombinedRegistry struct{}
var GlobalRegistry = &CombinedRegistry{}
```

### 访问方式

```go
// 通过子包直接访问（推荐）
buff.GetBuffDefinition(buff.BuffTypeFire)
event.GetEventDefinition(event.EventTypeHerb)

// 通过 core 包访问（向后兼容）
core.GetBuffDefinition(core.BuffTypeFire)
core.GlobalBuffRegistry.GetBuffTypesByEvaluationRange(...)
```

### Buff 实例（支持多订阅）

```go
type Buff struct {
    Type            BuffType
    ID              string      // Buff实例ID
    Duration        int
    Charge          int
    SubscriptionIDs []string    // EventBus订阅ID列表（支持多Phase订阅）
}
```

### ItemDefinition

```go
type ItemDefinition struct {
    Type        ItemType
    Name        string
    Desc        string
    TargetSelf  bool
    TargetOther bool
    BuffType    BuffType
    Range       int
    Phase       event.Phase // 可使用时机
    Priority    int
    NeedConfirm bool        // 默认true
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
player.AddBuff(buff)              // 添加 Buff
player.RemoveBuff(buffType)       // 移除指定类型 Buff
player.HasBuff(buffType)          // 检查是否有 Buff
player.GetBuff(buffType)          // 获取 Buff 实例
player.TickBuffs()                // 更新持续时间，返回失效的 Buff
```

**特殊规则**：
- `Duration == -1` 表示永久 Buff（如朱雀离火）
- 隐匿状态下免疫负面 Buff 和伤害

### 3. 阵营特性

```go
// 朱雀玩家初始携带离火 Buff
if config.Faction == FactionZhuQue {
    player.AddBuff(NewBuff(BuffTypeFire, -1))
}
```

## 测试覆盖

| 子包 | 测试文件 | 覆盖内容 |
|------|---------|---------|
| types/ | evaluation_test.go | Evaluation 评分系统 |
| types/ | special_effect_test.go | SpecialEffect 枚举 |
| buff/ | buff_test.go | BuffType、Buff实例、BuffDefinition、多Phase |
| event/ | event_test.go | EventType、EventDefinition |
| item/ | item_test.go | ItemType、Item实例、ItemDefinition |
| core | player_test.go | Player 创建、数值逻辑、Buff/道具管理 |

## 新增内容流程

添加新的 Buff/Event/Item 只需：

1. 在子包的枚举定义中添加常量
2. 在子包的 init.go 中添加注册

```go
// buff/buff.go - 添加枚举
const (
    ...
    BuffTypeNewBuff  // 新Buff
)

// buff/init.go - 注册定义
GlobalBuffRegistry.RegisterBuff(&BuffDefinition{
    Type:        BuffTypeNewBuff,
    Eval:        types.EvaluationGood,
    EnglishName: "NewBuff",
    Name:        "新Buff",
    ...
}, nil)
```

测试覆盖率：~93% statements