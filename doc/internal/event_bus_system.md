# EventBus + Decision 系统实现文档

## 概述

EventBus + Decision 系统是《命运骰子》游戏的统一触发机制框架，为所有Buff、道具、阵营被动提供Phase分类和用户确认支持。

## 设计目标

1. **统一Phase系统**：Buff/道具按触发时机分类
2. **用户确认机制**：支持玩家决定是否使用道具或触发技能
3. **多Phase支持**：一个Buff可以注册多个触发时机
4. **策略模式**：通过EventHandler实现高度定制化的Buff效果
5. **Buff生命周期事件**：支持Applied/Removed事件广播
6. **易于扩展**：新增Buff/道具只需声明Phase，无需改核心代码
7. **维护友好**：模块解耦，改动集中

## 文件结构

```
pkg/event/
├── phase.go          # Phase枚举定义（9种触发时机）
├── bus.go            # EventBus结构和方法
├── decision.go       # Decision和Option结构
├── context.go        # Context结构
├── bus_test.go       # 单元测试

internal/core/
├── evaluation.go     # 评分系统（0-100）
├── faction.go        # 阵营定义（四神兽）
├── buff.go           # Buff系统（类型/定义/注册表，支持多Phase）
├── item.go           # Item系统（类型/定义/注册表）
├── event.go          # Event系统（类型/定义/注册表）
├── player.go         # Player结构（HP/LP/Buffs/Items）

internal/engine/
├── game.go           # Game实例（EventBus/玩家管理/订阅，ApplyBuffToPlayer/RemoveBuffFromPlayer）
├── state_machine.go  # StateMachine（Phase触发/决策等待）
├── handlers.go       # EventHandler策略注册表
```

## Phase枚举

```go
type Phase int

const (
    PhaseBeforeTurn  Phase = iota  // 回合开始前（神眷、诅咒 LP±1）
    PhaseOnMove                    // 移动时（迷途反向）
    PhaseOnLand                    // 落地后（任意门、落点事件）
    PhasePreEvent                  // 事件触发前（辟邪、玄武、护盾道具）
    PhasePreDamage                 // 受伤前（隐匿、护盾）
    PhaseAfterTurn                 // 回合结束后（甘霖/腐化 HP±1）
    PhaseAnyTime                   // 任何时候可用（道具主动使用）
    PhaseOnBuffApplied             // Buff被挂载时触发（生命周期事件）
    PhaseOnBuffRemoved             // Buff被移除时触发（生命周期事件）
)
```

| Phase | 说明 | 需订阅EventBus |
|-------|------|---------------|
| BeforeTurn | 回合开始前 | ✓ |
| OnMove | 移动过程中 | ✓ |
| OnLand | 落地后 | ✓ |
| PreEvent | 事件触发前 | ✓ |
| PreDamage | 受伤前 | ✓ |
| AfterTurn | 回合结束后 | ✓ |
| AnyTime | 任何时候可用 | ❌（主动触发） |
| OnBuffApplied | Buff被挂载时 | ✓ |
| OnBuffRemoved | Buff被移除时 | ✓ |

## 多Phase支持

BuffDefinition 现在支持多Phase触发：
```go
type BuffDefinition struct {
    Type        BuffType
    Phases      []event.Phase  // 支持多Phase触发
    Priority    int
    NeedConfirm bool
    // ...
}
```

Buff 实例存储多个订阅ID：
```go
type Buff struct {
    Type            BuffType
    SubscriptionIDs []string  // 多订阅ID
    // ...
}
```

## Buff生命周期管理

通过 `ApplyBuffToPlayer` 和 `RemoveBuffFromPlayer` 方法管理完整的Buff生命周期：

```go
// ApplyBuffToPlayer 流程（先订阅，后广播）
func (g *Game) ApplyBuffToPlayer(player, buff) {
    1. player.AddBuff(buff)      // 底层数据添加
    2. g.SubscribeBuff(player, buff) // 挂载到EventBus
    3. g.BroadcastBuffApplied(player, buff) // 广播Applied事件
}

// RemoveBuffFromPlayer 流程（先广播，后取消订阅）
func (g *Game) RemoveBuffFromPlayer(player, buff) {
    1. g.BroadcastBuffRemoved(player, buff) // 广播Removed事件
    2. g.UnsubscribeBuff(buff)  // 取消订阅
    3. player.RemoveBuff(buff.Type) // 底层数据移除
}
```

**关键设计**：
- Applied：订阅发生在广播之前，Buff可以听到自己的Applied事件
- Removed：广播发生在取消订阅之前，Buff可以听到自己的removed事件

## EventHandler策略模式

通过策略模式实现高度定制化的Buff效果：

```go
// EventHandler 是定制化的Buff处理函数
type EventHandler func(phase event.Phase, ctx *event.Context)

// BuffHandlers 注册表
var BuffHandlers = map[core.BuffType]EventHandler{
    core.BuffTypeFire: handleZhuQueFire,  // 朱雀离火：每4回合LP+1
    // 更多处理器...
}
```

**优势**：
1. **数据行为分离**：BuffDefinition保持纯数据，可序列化
2. **万能拦截器**：通过修改ctx.Data实现各种机制
3. **消灭特判代码**：阵营逻辑成为Buff处理器

## Buff/Item与Phase对应

| Buff | Phases | NeedConfirm | 说明 |
|------|--------|-------------|------|
| 神眷 | [BeforeTurn] | false | 自动LP+1 |
| 诅咒 | [BeforeTurn] | false | 自动LP-1 |
| 迷途 | [OnMove] | false | 自动反向 |
| 隐匿 | [PreDamage] | false | 自动免疫（高优先级） |
| 辟邪 | [PreEvent] | false | 自动免疫毒瘴 |
| 甘霖 | [AfterTurn] | false | 每2回合HP+1 |
| 腐化 | [AfterTurn] | false | 每2回合HP-1 |
| 毒瘴 | [BeforeTurn] | false | 每回合恶性事件 |
| 离火 | [BeforeTurn] | false | 每4回合LP+1（定制处理器） |

| Item | Phase | NeedConfirm | 说明 |
|------|-------|-------------|------|
| 反方向的钟 | AnyTime | true | 主动使用，需确认目标 |
| 任意门 | OnLand | true | 落地后，需确认目标 |
| 骰子交换 | AnyTime | true | 主动使用，需确认目标 |
| 骰子升级卡 | BeforeTurn | true | 回合开始前，需确认 |

## 测试覆盖

### 测试覆盖率统计
```
pkg/event:       91.9% statements
internal/core:   93.4% statements
internal/engine: 91.8% statements
```

## 后续扩展

1. **更多EventHandler**：为复杂Buff添加定制处理器
2. **多Phase Buff**：实现如"白虎劫运"等多触发时机Buff
3. **Buff联动**：监听OnBuffApplied/Removed实现联动效果
4. **UI集成**：Decision Prompt格式规范
5. **超时处理**：客户端无响应时自动执行默认选项