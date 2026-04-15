# internal/engine/hsm - Hierarchical State Machine

分层状态机（Hierarchical State Machine）实现，用于管理游戏流程的三层状态结构。

## 概述

HSM 包提供三层架构的状态机：
- **Layer 1 (Global States)**: 全局状态，控制游戏大阶段
- **Layer 2 (Turn States)**: 回合状态，控制玩家回合流程
- **Layer 3 (Interrupt States)**: 中断状态，处理需要用户确认的决策

## 三层架构

```
┌─────────────────────────────────────────────────────────┐
│ Layer 1: Global States                                  │
│ MatchInit → RoundMiniGame → RoundPrep → TurnLoop        │
│                                    ↓                     │
│                              BossBattle                   │
│                                    ↓                     │
│                               GameOver                   │
└─────────────────────────────────────────────────────────┘
              │
              ↓
┌─────────────────────────────────────────────────────────┐
│ Layer 2: Turn States (在 TurnLoop 内循环执行)            │
│ TurnUpkeep → MainAction → TurnMoving → TurnLanded       │
│                                            ↓             │
│                                        TurnEvent         │
│                                            ↓             │
│                                         TurnEnd          │
└─────────────────────────────────────────────────────────┘
              │
              ↓ (需要用户决策时)
┌─────────────────────────────────────────────────────────┐
│ Layer 3: Interrupt States                               │
│ WaitDecision (暂停当前状态，等待用户输入)                  │
└─────────────────────────────────────────────────────────┘
```

## 核心文件

| 文件 | 描述 |
|------|------|
| `state.go` | State接口定义、StateContext上下文、Metadata key常量 |
| `state_id.go` | StateID枚举定义和辅助方法 |
| `state_stack.go` | StateStack实现（入栈/出栈机制） |
| `hsm.go` | HSM主结构、状态转移逻辑、Snapshot机制 |
| `adapter.go` | EventBusAdapter、MapEngineAdapter、GameWrapper |
| `global_states.go` | Layer 1 全局状态实现 |
| `turn_states.go` | Layer 2 回合状态实现 |
| `interrupt_states.go` | Layer 3 中断状态实现 |

## State 接口

```go
type State interface {
    ID() StateID
    Enter(ctx *StateContext)
    Update(ctx *StateContext) StateID
    Exit(ctx *StateContext)
    CanTransitionTo(target StateID) bool
}
```

## StateContext

StateContext 提供状态执行的上下文数据，包含：
- **Metadata**: 嵌入的元数据存储（类型安全的键值对）
- **Game**: 游戏实例引用
- **Player**: 当前玩家引用
- **Bus**: EventBus适配器
- **MapEngine**: 地图引擎适配器
- **Phase/PhaseData**: 阶段触发数据
- **Decision/Decisions**: 待处理决策
- **Stack**: 状态栈引用

```go
ctx := NewStateContext().
    WithGame(game).
    WithPlayer(player).
    WithMapEngine(mapAdapter).
    WithBus(busAdapter)

// 使用 Metadata 方法
ctx.SetInt(KeyDiceSteps, 5)
ctx.SetBool(KeySkipTurn, true)
ctx.GetDiceSteps() // 返回 5
```

## Layer 1: Global States

| State | 描述 |
|-------|------|
| MatchInit | 初始化游戏：生成地图、分配阵营 |
| RoundMiniGame | 小游戏阶段：所有玩家参与竞争 |
| RoundPrep | 回合准备：根据排名分配骰子类型 |
| TurnLoop | 回合循环：管理玩家回合轮转 |
| BossBattle | Boss战斗：玩家到达终点触发 |
| GameOver | 游戏结束：结算和排名 |

## Layer 2: Turn States

| State | 描述 | Phase触发 |
|-------|------|-----------|
| TurnUpkeep | 回合准备：检查死亡/跳过、触发BeforeTurn | PhaseBeforeTurn |
| MainAction | 主要行动：等待骰子/道具/技能输入 | - |
| TurnMoving | 移动处理：计算路径、处理Fragile/Fog | PhasePreMove |
| TurnLanded | 落地处理：检查格子类型、触发OnLand | PhaseOnLand |
| TurnEvent | 事件处理：抽取事件、触发PreEvent | PhasePreEvent |
| TurnEnd | 回合结束：触发AfterTurn、TickBuffs | PhaseAfterTurn |

## Layer 3: Interrupt States

| State | 描述 |
|-------|------|
| WaitDecision | 等待决策：暂停当前状态，等待用户选择 |

## Phase 触发机制

设计原则：**谁产生时机，谁发布Phase**

| Phase | 发布者 | 触发位置 |
|-------|--------|----------|
| BeforeTurn | HSM | TurnUpkeep.Enter() |
| OnLand | HSM | TurnLanded.Enter() |
| AfterTurn | HSM | TurnEnd.Enter() |
| PreMove | Action | MoveAction.Execute() |
| PreEvent | Action | DrawEventAction.Execute() |
| PreDamage | Action | DamageAction.Execute() |
| ItemUsed | Game | Game.UseItem() |

## 与 Action 系统集成

Turn States 通过 ActionContext 执行 Action：

```
TurnUpkeep → ExecuteAction(ModifyLPAction) → PreTrigger → Execute → PostTrigger → EventLog
TurnMoving → ExecuteAction(MoveAction) → PreTrigger(PreMove) → Execute → EventLog
TurnEvent → ExecuteAction(DrawEventAction) → PreTrigger(PreEvent) → Execute → EventLog
```

## 使用示例

```go
// 创建 HSM
game := engine.NewGame("match1", 0)
hsm := NewHSM(game)

// 注册状态
RegisterGlobalStates(hsm)
RegisterTurnStates(hsm)
RegisterInterruptStates(hsm)

// 启动 HSM
ctx := NewStateContext().WithGame(game)
hsm.Start(StateMatchInit, ctx)

// 更新循环
for hsm.IsRunning() {
    hsm.Update(ctx)
}

// 处理用户输入
hsm.OnRollDice(6, ctx)     // 骰子输入
hsm.OnUserChoice(1, ctx)   // 决策选择

// 创建 Snapshot（持久化）
snapshot := hsm.CreateSnapshot()

// 从 Snapshot 恢复
hsm.RestoreFromSnapshot(snapshot)
```

## Adapter 系统

HSM 使用 Adapter 模式隔离外部包依赖：

- **EventBusAdapter**: 隔离 pkg/event 包
- **MapEngineAdapter**: 隔离 internal/gamemap 包
- **GameWrapper**: 实现 protocol.Game 接口
- **ProtocolMapEngineWrapper**: 实现 protocol.MapEngine 接口

```go
// 使用 Wrapper 创建 Adapter
busAdapter := NewEventBusWrapper(game.Bus)
mapAdapter := NewMapEngineWrapper(mapEngine)
gameWrapper := NewGameWrapper(game)
```

## Metadata 契约

**重要**：`StateContext.Metadata` 字段使用遵循契约文档定义。

详见：[doc/metadata/hsm_context.md](../../../doc/metadata/hsm_context.md) - StateContext.Metadata 契约（状态机通信字段）

所有 Metadata key 使用预定义常量（KeyXxx），确保类型安全和命名统一。

新增状态标记时：
1. 在 `state.go` 添加常量定义（KeyXxx）
2. 如常用，添加便捷方法（WithXxx/GetXxx/SetXxx/IsXxx）
3. 在契约文档更新表格

## 测试

```bash
go test ./internal/engine/hsm/... -v
```