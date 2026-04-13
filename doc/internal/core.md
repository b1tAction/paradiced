# internal/core - Core Data Structures

核心数据结构包，定义游戏的基础类型和数据结构。

## 概述

internal/core 是《命运骰子》游戏的核心数据模块，负责数据结构定义、评分系统、阵营系统、Buff/道具/事件管理和玩家基础逻辑。

此包无外部依赖（仅依赖 pkg/event），可独立使用。

## 文件结构

```
internal/core/
├── evaluation.go     # 评分系统（0-100）
├── faction.go        # 阵营定义（四神兽）
├── buff.go           # Buff系统（类型/定义/注册表，支持多Phase）
├── item.go           # Item系统（类型/定义/注册表）
├── event.go          # Event系统（类型/定义/注册表）
├── player.go         # Player结构（HP/LP/Buffs/Items）
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

### BuffDefinition（支持多Phase）

```go
type BuffDefinition struct {
    Type        BuffType
    Name        string
    Eval        Evaluation
    Desc        string
    Duration    int
    HPPerTurn   int         // 每回合 HP 变化
    LPPerTurn   int         // 每回合 LP 变化
    Special     string      // 特殊效果标记
    Phases      []event.Phase // 触发时机列表（支持多Phase）
    Priority    int         // 执行优先级
    NeedConfirm bool        // 是否需要用户确认（默认false）
}

// 方法
func (def *BuffDefinition) GetPhases() []event.Phase
func (def *BuffDefinition) HasPhase(phase event.Phase) bool
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
| buff_test.go | Evaluation、BuffType、Buff 实例、BuffDefinition、多Phase支持、SubscriptionIDs |
| item_test.go | ItemType、Item 实例、ItemDefinition、ItemRegistry |
| event_test.go | EventType、EventDefinition、EventRegistry |
| player_test.go | Player 创建、数值逻辑、移动、Buff/道具管理、阵营特性 |

测试覆盖率：~93% statements