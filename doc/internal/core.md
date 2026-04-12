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
├── buff.go           # Buff系统（类型/定义/注册表）
├── item.go           # Item系统（类型/定义/注册表）
├── event.go          # Event系统（类型/定义/注册表）
├── player.go         # Player结构（HP/LP/Buffs/Items）
├── *_test.go         # 单元测试
├── README.md         # 包文档
```

## 数据类型

### Evaluation (评分系统)

```go
type Evaluation int  // 0-100

// 分类边界
EvaluationBadThreshold     = 40   // 恶性上限（≤40）
EvaluationNeutralThreshold = 65   // 中性上限（≤65）
// Evaluation > 65 为良性

// 预定义常量
EvaluationVeryBad   = 10   // 极恶
EvaluationBad       = 25   // 较恶
EvaluationMildBad   = 35   // 轻恶
EvaluationNeutral   = 50   // 中性
EvaluationMixed     = 55   // 混合
EvaluationMildGood  = 70   // 轻良
EvaluationGood      = 80   // 较良
EvaluationVeryGood  = 90   // 极良
EvaluationExcellent = 100  // 最佳
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
    UserID      string   // 玩家UUID
    Faction     Faction  // 阵营
    Position    int      // 当前位置
    HP          int      // 血量
    LP          int      // 幸运值（影响随机事件）
    Inventory   []*Item  // 道具栏
    ActiveBuffs []*Buff  // 持续状态
    IsDead      bool     // 是否死亡
    SkipTurn    bool     // 是否跳过回合
    ChargeCount int      // 充能计数（青龙/玄武）
    FireCounter int      // 离火计数（朱雀）
}
```

## Buff/Item/Event 定义

每种类型都有完整的定义结构，包含 Phase、Priority、NeedConfirm 字段：

```go
type BuffDefinition struct {
    Type        BuffType
    Name        string
    Eval        Evaluation
    Phase       event.Phase      // 触发时机
    Priority    int              // 执行优先级
    NeedConfirm bool             // 是否需要用户确认（默认false）
    Duration    int              // 默认持续时间
    LPPerTurn   int              // 每回合 LP 变化
    HPPerTurn   int              // 每回合 HP 变化
    Special     string           // 特殊效果标记
}

type ItemDefinition struct {
    Type        ItemType
    Name        string
    Eval        Evaluation
    Phase       event.Phase      // 可使用时机
    Priority    int              // 执行优先级
    NeedConfirm bool             // 是否需要用户确认（默认true）
    TargetSelf  bool             // 可对自己使用
    TargetOther bool             // 可对他人使用
    Range       int              // 有效范围
    BuffType    BuffType         // 赋予的 Buff 类型
}
```

## 核心功能

### 1. 数值逻辑

```go
// 扣血（注意：回城逻辑由 engine 包处理）
player.ApplyDamage(amount) error

// 回血
player.Heal(amount) error

// 修改幸运值（范围限制 0~8）
player.ModifyLP(amount)
```

**隐匿免疫**：ApplyDamage 时检查 Hidden Buff，自动免疫伤害。

### 2. 移动逻辑

```go
player.Move(newPosition, maxLength)  // 移动到指定位置
player.Respawn(respawnPos)           // 复活回城
```

### 3. Buff 管理

```go
player.AddBuff(buff)              // 添加 Buff
player.RemoveBuff(buffType)       // 移除指定类型 Buff
player.HasBuff(buffType)          // 检查是否有 Buff
player.GetBuff(buffType)          // 获取 Buff 实例
player.TickBuffs()                // 更新持续时间，返回失效的 Buff
player.ClearNegativeBuffs()       // 清除所有负面 Buff
```

**特殊规则**：
- `Duration == -1` 表示永久 Buff（如朱雀离火）
- 隐匿状态下免疫负面 Buff 和伤害

### 4. 道具管理

```go
player.AddItem(item)              // 添加道具
player.RemoveItem(itemID)         // 移除道具
player.GetItem(itemID)            // 获取道具
player.HasItem(itemType)          // 检查是否有指定类型道具
```

## 与其他包的关系

- **pkg/event**: Phase 类型依赖
- **internal/engine**: 使用 core 的数据类型，处理 EventBus 订阅和 Phase 触发
- **internal/gamemap**: 无直接依赖

## 注意事项

- 阵营被动技能触发逻辑已迁移到 engine 包，通过 EventBus + Decision 系统处理
- ApplyDamage 不再包含 MapEngine 参数，回城逻辑由 engine 包统一处理
- Player 结构增加了 FireCounter 字段用于朱雀离火计数

## 测试覆盖

测试文件位于 internal/core/*_test.go：

| 测试文件 | 覆盖内容 |
|---------|---------|
| buff_test.go | Evaluation、BuffType、Buff 实例、BuffDefinition、BuffRegistry |
| item_test.go | ItemType、Item 实例、ItemDefinition、ItemRegistry |
| event_test.go | EventType、EventDefinition、EventRegistry |
| player_test.go | Player 创建、数值逻辑、移动、Buff/道具管理、阵营特性 |

测试覆盖率：~90% statements