# internal/core - Core Data Structures

核心数据结构包，定义游戏的基础类型和数据结构。

## 概述

internal/core 是《命运骰子》游戏的核心数据模块，负责数据结构定义、评分系统、阵营系统、Buff/道具/事件管理和玩家基础逻辑。

此包无外部依赖（仅依赖 pkg/event），可独立使用。

## 文件结构

```
internal/core/
├── evaluation.go       # 评分系统（0-100）
├── faction.go          # 阵营定义（四神兽）
├── registry.go         # 统一注册表（单一数据源）
├── special_effect.go   # 特殊效果枚举
├── buff_init.go        # Buff 注册初始化
├── event_init.go       # Event 注册初始化
├── item_init.go        # Item 注册初始化
├── buff.go             # Buff 枚举和实例结构
├── item.go             # Item 枚举和实例结构
├── event.go            # Event 枚举和定义结构
├── player.go           # Player结构（HP/LP/Buffs/Items）
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

## Buff/Item/Event 定义

### 统一注册表（GlobalRegistry）

所有定义在包初始化时统一注册到 `GlobalRegistry`，实现单一数据源：

```go
// 在 buff_init.go 的 init() 中注册
GlobalRegistry.RegisterBuff(&BuffDefinition{
    Type:          BuffTypeFire,
    Eval:          EvaluationGood,
    EnglishName:   "Fire",           // 用于 String()
    Name:          "离火",           // 中文显示名
    Desc:          "朱雀阵营增益...",
    Duration:      -1,
    SpecialEffect: SpecialZhuQuePassive,  // 枚举替代字符串
    Phases:        []event.Phase{event.PhaseBeforeTurn},
    Priority:      10,
}, handleZhuQueFire)  // Handler 在注册时传入
```

**注册表自动生成**：
- `String()` 映射（使用 EnglishName）
- `Evaluation` 映射
- 分类列表（Good/Bad/Neutral）

### BuffDefinition（支持多Phase）

```go
type BuffDefinition struct {
    Type          BuffType
    Eval          Evaluation
    EnglishName   string        // 英文标识符（用于 String()）
    Name          string        // 中文名称（用于显示）
    Desc          string
    Duration      int
    HPPerTurn     int           // 每回合 HP 变化
    LPPerTurn     int           // 每回合 LP 变化
    SpecialEffect SpecialEffect // 特殊效果枚举（替代字符串标记）
    Phases        []event.Phase // 触发时机列表（支持多Phase）
    Priority      int           // 执行优先级
    NeedConfirm   bool          // 是否需要用户确认（默认false）
}

// 方法
func (def *BuffDefinition) GetPhases() []event.Phase
func (def *BuffDefinition) HasPhase(phase event.Phase) bool
```

### 访问方式（函数式API）

```go
// 获取定义
def := core.GetBuffDefinition(buffType)

// 获取字符串标识
name := buffType.String()  // 返回 EnglishName

// 获取评估分数
eval := core.GetBuffEvaluation(buffType)

// 获取自定义 Handler
handler := core.GetBuffHandler(buffType)

// 获取分类列表
goodBuffs := core.GetBuffTypesByCategory("Good")
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

| 测试文件 | 覆盖内容 |
|---------|---------|
| registry.go | 注册表初始化、定义注册、查询方法、分类生成 |
| special_effect.go | SpecialEffect 枚举、类型判断方法 |
| buff_test.go | BuffType、Buff 实例、BuffDefinition、多Phase支持 |
| item_test.go | ItemType、Item 实例、ItemDefinition |
| event_test.go | EventType、EventDefinition |
| player_test.go | Player 创建、数值逻辑、移动、Buff/道具管理、阵营特性 |

## 新增内容流程

添加新的 Buff/Event/Item 只需：

1. 在枚举定义中添加常量
2. 在对应 `_init.go` 文件中添加注册

```go
// buff.go - 添加枚举
const (
    ...
    BuffTypeNewBuff  // 新Buff
)

// buff_init.go - 注册定义
GlobalRegistry.RegisterBuff(&BuffDefinition{
    Type:        BuffTypeNewBuff,
    Eval:        EvaluationGood,
    EnglishName: "NewBuff",
    Name:        "新Buff",
    ...
}, nil)
```

测试覆盖率：~93% statements