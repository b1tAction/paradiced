# internal/engine - 游戏引擎

游戏引擎包，管理游戏流程、事件触发和回合流转。

## 功能

### Game 实例

- EventBus 管理
- 玩家添加/移除
- Buff/道具订阅管理
- 回合轮转
- Buff生命周期管理（Apply/Remove）

### StateMachine

- Phase 触发
- 用户决策等待
- 状态流转

### TurnFlow

回合流转控制器，编排完整的回合流程：

- **TurnStep 步骤管理**：Init → Upcheck → BeforeTurn → MainAction → OnMove → OnLand → PreEvent → AfterTurn → Complete
- **用户决策处理**：等待用户输入，超时自动执行默认选项
- **中断恢复**：保存/恢复快照，支持断线重连

### FlowSnapshot

回合快照，用于中断恢复和状态持久化：

- 游戏状态（Round/Turn/CurrentStep）
- 玩家状态（HP/LP/Position/Buffs/Items）
- 待处理决策

### StateSync

状态同步，用于客户端通信：

- **FullSync**：完整游戏状态
- **DeltaSync**：增量更新（仅发送变化部分）
- **EventSync**：事件通知
- **DecisionSync**：决策请求

### Handlers

EventHandler 策略注册表，实现定制化的 Buff 效果：

- 朱雀离火：每4回合 LP+1
- 其他 Buff 的默认处理逻辑

## 文件结构

```
internal/engine/
├── game.go           # Game 实例和 EventBus 管理
├── state_machine.go  # Phase 触发状态机
├── turn_flow.go      # TurnFlow 回合流转控制器
├── snapshot.go       # FlowSnapshot 快照和 SnapshotManager
├── state_sync.go     # StateSync 状态同步
├── handlers.go       # EventHandler 策略注册表
├── integration_test.go # 集成测试
└── turn_flow_test.go # TurnFlow 测试
```

## TurnFlow 使用示例

```go
// 创建游戏和地图引擎
game := engine.NewGame("game-001")
mapEngine := gamemap.NewMapEngine(50)

// 创建 TurnFlow 控制器
tf := engine.NewTurnFlow(game, mapEngine)

// 添加玩家
player := core.NewPlayer(config)
game.AddPlayer(player)

// 执行回合
decisions, err := tf.ExecuteTurn(player)
if len(decisions) > 0 {
    // 需要用户输入
    tf.OnUserChoice(0) // 用户选择第一个选项
}

// 或使用步骤式执行
tf.CurrentStep = engine.StepInit
result := tf.ExecuteStep(engine.StepBeforeTurn, player)

// 中断和恢复
tf.Interrupt()           // 保存快照
tf.ResumeFromInterrupt(tf.SavedSnapshot) // 从快照恢复

// 状态同步
sync := engine.NewStateSync(game)
msg, _ := sync.SyncFull() // 完整同步
msg, _ := sync.SyncDelta() // 增量同步
```

## Phase 触发流程

```go
// 传统方式（StateMachine）
sm := engine.NewStateMachine(game)

// BeforeTurn
if sm.TriggerPhaseAndWait(event.PhaseBeforeTurn, player) {
    // 等待用户确认骰子升级卡等
}

// OnMove
sm.ExecuteOnMovePhase(player)

// OnLand
if sm.TriggerPhaseAndWait(event.PhaseOnLand, player) {
    // 等待用户确认任意门等
}

// PreEvent
sm.ExecutePreEventPhase(player)

// AfterTurn
sm.ExecuteAfterTurnPhase(player)
```

## 与其他包的关系

- `internal/core`: Player, Buff, Item 类型
- `internal/gamemap`: MapEngine 地图引擎
- `pkg/event`: EventBus, Decision, Phase, Context

## 测试覆盖率

91.8% statements