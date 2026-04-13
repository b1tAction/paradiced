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

### Handlers

EventHandler 策略注册表，实现定制化的 Buff 效果：

- 朱雀离火：每4回合 LP+1
- 其他 Buff 的默认处理逻辑

## 文件结构

```
internal/engine/
├── game.go           # Game 实例和 EventBus 管理
├── state_machine.go  # Phase 触发状态机
├── handlers.go       # EventHandler 策略注册表
└── integration_test.go # 集成测试
```

## Phase 触发流程

```go
// 使用 StateMachine
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