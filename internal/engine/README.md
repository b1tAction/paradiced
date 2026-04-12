# internal/engine - Game Engine

游戏引擎包，管理游戏流程和事件触发。

## 功能

### Game 实例

- EventBus 管理
- 玩家添加/移除
- Buff/道具订阅管理
- 回合轮转

### StateMachine

- Phase 触发
- 用户决策等待
- 状态流转

## 文件结构

```
internal/engine/
├── game.go         # Game 实例和 EventBus 管理
└── state_machine.go # Phase 触发状态机
```

## Phase 触发流程

```go
// 创建游戏和状态机
game := engine.NewGame("game-001")
sm := engine.NewStateMachine(game)

// 添加玩家
player := core.NewPlayer(config)
game.AddPlayer(player)

// 执行回合流程
// 1. BeforeTurn
if sm.TriggerPhaseAndWait(event.PhaseBeforeTurn, player) {
    // 等待用户确认骰子升级卡等
}

// 2. OnMove
sm.ExecuteOnMovePhase(player)

// 3. OnLand
if sm.TriggerPhaseAndWait(event.PhaseOnLand, player) {
    // 等待用户确认任意门等
}

// 4. PreEvent
sm.ExecutePreEventPhase(player)

// 5. AfterTurn
sm.ExecuteAfterTurnPhase(player)
```

## 与其他包的关系

- `internal/core`: Player, Buff, Item 类型
- `pkg/event`: EventBus, Decision, Phase