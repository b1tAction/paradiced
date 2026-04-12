# EventBus + Decision 系统实现文档

## 概述

EventBus + Decision 系统是《命运骰子》游戏的统一触发机制框架，为所有Buff、道具、阵营被动提供Phase分类和用户确认支持。

## 设计目标

1. **统一Phase系统**：Buff/道具按触发时机分类
2. **用户确认机制**：支持玩家决定是否使用道具或触发技能
3. **易于扩展**：新增Buff/道具只需声明Phase，无需改核心代码
4. **维护友好**：模块解耦，改动集中

## 文件结构

```
pkg/event/
├── phase.go          # Phase枚举定义
├── bus.go            # EventBus结构和方法
├── decision.go       # Decision和Option结构
├── context.go        # Context结构
└── bus_test.go       # 单元测试

internal/game/
├── buff.go           # BuffDefinition扩展（Phase字段）
├── item.go           # ItemDefinition扩展（Phase字段）
├── game.go           # Game实例（包含EventBus）
├── state_machine.go  # StateMachine（Phase触发、Decision等待）
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
    PhasePassive                   // 永久被动（离火）
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
| Passive | 永久被动 | ❌（特殊处理） |

## Decision结构

```go
type Decision struct {
    ID          string      // 决策ID
    Prompt      string      // 提示文本
    Options     []Option    // 可选项列表
    Priority    int         // 执行优先级
    Timeout     time.Duration // 超时时间
    Default     int         // 超时默认选项
    NeedConfirm bool        // 是否需要用户确认
    Condition   func() bool // 动态判断条件
    OnChoice    func(int, *Context) // 选择后回调
    SourceID    string      // 来源ID（Buff/道具）
    SourceType  string      // 来源类型
}

type Option struct {
    ID     string
    Label  string
    Action func(*Context)
}
```

**NeedConfirm区分**：
- `NeedConfirm=true`：需要用户确认，加入等待列表
- `NeedConfirm=false`：自动执行默认选项

## EventBus结构

```go
type EventBus struct {
    subscriptions map[Phase][]*Subscription
    GameID        string
}

type Subscription struct {
    ID        string    // 订阅ID
    OwnerID   string    // 玩家ID
    SourceID  string    // Buff/道具ID
    SourceType string   // "buff" / "item"
    Priority  int       // 执行优先级
    Decision  *Decision // 预绑定的Decision
}
```

**核心方法**：
- `Subscribe(phase, ownerID, sourceID, sourceType, decision)` → 订阅ID
- `Unsubscribe(subID)` → 取消订阅
- `UnsubscribeBySource(sourceID)` → 移除Buff/道具时批量取消
- `Publish(phase, ownerID, ctx)` → 返回需要确认的Decision列表

## Game实例

```go
type Game struct {
    ID      string
    Bus     *EventBus
    Players []*Player
    State   *GameState
}
```

**订阅管理**：
- 玩家加入时自动订阅Buff/道具
- Buff移除时自动取消订阅
- 道具消耗后自动取消订阅

## StateMachine

```go
type StateMachine struct {
    Game       *Game
    WaitingFor []*Decision  // 等待的决策列表
    CurrentCtx *Context
    FlowState  string       // "idle" / "waiting" / "running"
}
```

**核心流程**：

```go
// 触发Phase
decisions := sm.TriggerPhaseAndWait(PhasePreEvent, player)

if decisions {
    // 有Decision需要确认，进入等待状态
    // 发送Prompt给客户端等待用户输入
    sm.GetCurrentDecision() // 获取当前Decision
} else {
    // 无需确认，流程继续
}

// 用户选择后
sm.OnUserChoice(choice) // 处理用户选择
```

## Buff/Item与Phase对应

| Buff | Phase | NeedConfirm | 说明 |
|------|-------|-------------|------|
| 神眷 | BeforeTurn | false | 自动LP+1 |
| 诅咒 | BeforeTurn | false | 自动LP-1 |
| 迷途 | OnMove | false | 自动反向 |
| 隐匿 | PreDamage | false | 自动免疫（高优先级） |
| 辟邪 | PreEvent | false | 自动免疫毒瘴 |
| 甘霖 | AfterTurn | false | 每2回合HP+1 |
| 腐化 | AfterTurn | false | 每2回合HP-1 |
| 毒瘴 | BeforeTurn | false | 每回合恶性事件（低优先级） |
| 离火 | Passive | false | 每4回合LP+1（特殊计数） |

| Item | Phase | NeedConfirm | 说明 |
|------|-------|-------------|------|
| 反方向的钟 | AnyTime | true | 主动使用，需确认目标 |
| 任意门 | OnLand | true | 落地后，需确认目标 |
| 骰子交换 | AnyTime | true | 主动使用，需确认目标 |
| 骰子升级卡 | BeforeTurn | true | 回合开始前，需确认 |

## 状态机集成示例

```go
// 回合流程
func (sm *StateMachine) ProcessTurn(player *Player) {
    // 1. 回合开始前
    if sm.TriggerPhaseAndWait(PhaseBeforeTurn, player) {
        return // 等待用户确认骰子升级卡等
    }

    // 2. 掷骰子，移动
    steps := rollDice()
    for i := 0; i < steps; i++ {
        // 移动中Phase
        sm.TriggerPhase(PhaseOnMove, player) // 迷途等效果
    }

    // 3. 落地后
    if sm.TriggerPhaseAndWait(PhaseOnLand, player) {
        return // 等待用户确认任意门等
    }

    // 4. 事件触发前
    if sm.TriggerPhaseAndWait(PhasePreEvent, player) {
        return // 等待用户确认护盾道具等
    }

    // 5. 执行事件
    executeEvent(player)

    // 6. 回合结束后
    sm.TriggerPhase(PhaseAfterTurn, player) // 甘霖/腐化效果、TickDuration
}
```

## 测试覆盖

| 测试类 | 覆盖内容 |
|--------|----------|
| PhaseTest | String转换、IsValid、NeedsSubscription |
| DecisionTest | 创建、优先级、超时、条件判断、执行、克隆 |
| EventBusTest | Subscribe、Unsubscribe、Publish、Priority排序 |
| ContextTest | 创建、WithEvent、WithData |

## 后续扩展

1. **Decision执行器**：定义标准化的Buff/道具效果执行逻辑
2. **超时处理**：客户端无响应时自动执行默认选项
3. **UI集成**：Decision Prompt格式规范
4. **Passive计数**：离火等永久Buff的回合计数机制