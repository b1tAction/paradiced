# internal/engine - 游戏引擎

游戏引擎包，管理游戏流程、事件触发和回合流转。

## 功能

### Game 实例

- EventBus 管理
- 玩家添加/移除
- Buff/道具订阅管理
- 回合轮转
- Buff生命周期管理（Apply/Remove）
- Action系统集成

### Registry

Buff/Item/Event Registry 管理定义和效果处理器：

- BuffRegistry：Buff 定义 + HandlerConfig + handlers
- ItemRegistry：Item 定义 + HandlerConfig + handlers
- EventRegistry：Event 定义 + HandlerConfig + handlers
- 所有效果通过 Action 系统执行

## 文件结构

```
internal/engine/
├── game.go            # Game 实例和 EventBus 管理
├── buff_registry.go   # Buff Registry + handlers
├── item_registry.go   # Item Registry + handlers
├── event_registry.go  # Event Registry + handlers
├── game_test.go       # Game 单元测试
├── buff_registry_test.go # Buff Registry 单元测试
├── item_registry_test.go # Item Registry 单元测试
├── event_registry_test.go # Event Registry 单元测试
└── action/            # Action 子包
    ├── action.go      # Action 接口定义（含TargetPlayer方法）
    ├── types.go       # 具体Action类型实现
    ├── context.go     # ActionContext 执行上下文
    ├── queue.go       # Queue 衍生动作队列
    └── types_test.go  # Action 类型测试
└── hsm/               # 分层状态机子包
    ├── state.go       # State 接口和全局状态
    ├── context.go     # StateContext
    └── state_test.go  # HSM 测试
```

## Action 系统集成

所有游戏效果（Buff/Item/Event/Faction被动）通过 Action 系统执行：

```
Buff Handler → 返回 Action → ActionContext.ExecuteAction →
  PreTrigger Phase → Execute → PostTrigger Phase → EventLog → ProcessQueue
```

### Action 执行流程

```go
// 创建 ActionContext
actionCtx := action.NewActionContext(game, bus, mapEngine)

// 执行 DamageAction
damageAction := action.NewDamageAction(player, 10, "Event_Trap")
actionCtx.ExecuteAction(damageAction)

// PreTrigger: 发布 PhasePreDamage，隐匿 Buff 可拦截
// Execute: player.ApplyDamage(10)
// PostTrigger: 无（PhaseAnyTime）
// EventLog: 记录 HPChange 事件

// 处理衍生动作
actionCtx.ProcessQueue()
```

## Phase 触发流程

**设计原则：谁产生时机，谁发布 Phase**

| Phase | 发布者 | 触发位置 |
|-------|--------|----------|
| BeforeTurn | HSM | TurnUpkeep.Enter() |
| OnLand | HSM | TurnLanded.Enter() |
| AfterTurn | HSM | TurnEnd.Enter() |
| PreMove | Action | MoveAction.Execute() |
| PreEvent | Action | DrawEventAction.Execute() |
| PreDamage | Action | DamageAction.Execute() |

## 与其他包的关系

- `internal/core`: Player, Buff, Item 类型
- `internal/gamemap`: MapEngine 地图引擎
- `pkg/event`: EventBus, Decision, Phase, Context
- `pkg/protocol`: Player/Game 接口
- `pkg/action`: Action 接口层

## 相关文档

- [action/README.md](action/README.md) - Action 实现详情
- [hsm/README.md](hsm/README.md) - 分层状态机详情

## 测试覆盖率

91.8% statements