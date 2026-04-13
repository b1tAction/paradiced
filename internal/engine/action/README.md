# Action 实现包

internal/engine/action 包提供了 Action 接口的具体实现。

## 文件结构

| 文件 | 说明 |
|------|------|
| `action.go` | ExecutableAction接口定义，类型别名 |
| `types.go` | 具体Action类型（DamageAction, HealAction等） |
| `context.go` | ActionContext执行上下文 |
| `queue.go` | Queue衍生动作队列 |
| `turn_event_log.go` | TurnEventLog事件日志 |
| `types_test.go` | 测试文件 |

## Action类型详解

### DamageAction

```go
type DamageAction struct {
    TargetPlayer *core.Player
    SourceID     string      // "Event_Trap", "Buff_Corrupt"
    Amount       int         // 可被拦截器修改
    IsPiercing   bool        // true则不可拦截
    BlockedBy    string      // 拦截器设置的阻断来源
}
```

- `CanModify() = !IsPiercing && Amount > 0`
- 被隐匿、护盾等拦截时设置 `Amount = 0`

### HealAction

```go
type HealAction struct {
    TargetPlayer *core.Player
    SourceID     string      // "Buff_Rain", "Item_HealingPotion"
    Amount       int
}
```

- `CanModify() = Amount > 0`

### ModifyLPAction

```go
type ModifyLPAction struct {
    TargetPlayer *core.Player
    SourceID     string      // "Buff_Divine", "Buff_Curse"
    Amount       int         // +1 or -1
}
```

- `CanModify() = false` - LP修改不可拦截

### MoveAction

```go
type MoveAction struct {
    TargetPlayer *core.Player
    Steps        int         // 可为负数（迷途反向）
    SourceID     string      // "DiceRoll"
    TargetPos    int         // 执行后设置
    Path         []int       // 计算后的路径
    Overtaken    []*core.Player // 反超的玩家（白虎劫运）
}
```

- `CanModify() = Steps != 0`
- 迷途Buff可设置 `Steps = -Steps` 反向移动

### AddBuffAction / RemoveBuffAction

```go
type AddBuffAction struct {
    TargetPlayer *core.Player
    BuffType     buff.BuffType
    Duration     int
    SourceID     string
}

type RemoveBuffAction struct {
    TargetPlayer *core.Player
    BuffType     buff.BuffType
    SourceID     string
}
```

- `CanModify() = false` - Buff操作不可拦截

### TeleportAction

```go
type TeleportAction struct {
    TargetPlayer *core.Player
    TargetPos    int
    SourceID     string      // "Item_AnyDoor"
}
```

- 用于任意门等道具

### StealBuffAction

```go
type StealBuffAction struct {
    TargetPlayer *core.Player  // 被偷取者
    SourcePlayer *core.Player  // 偷取者（白虎玩家）
    SourceID     string        // "Faction_BaiHu"
    StolenBuff   *core.Buff    // 执行后设置
}
```

- 白虎"劫运"阵营被动

## ActionContext

```go
type ActionContext struct {
    *util.Metadata              // 嵌入，支持扩展存储

    Game        GameInterface   // 游戏接口（避免循环依赖）
    EventBus    *event.EventBus // 用于拦截
    MapEngine   MapEngineInterface // 用于移动计算
    ActionQueue *Queue          // 衍生动作队列
    EventLog    *TurnEventLog   // 事件日志
}
```

### ExecuteAction 流程

1. 检查 `CanModify()`，若可拦截则 Publish 到 EventBus
2. 执行 `Execute(ctx)`
3. 记录 `LogEntry()` 到 EventLog
4. 处理 ActionQueue 中的衍生动作

### 接口定义（避免循环依赖）

```go
type GameInterface interface {
    GetCurrentPlayer() *core.Player
    GetPlayer(id string) *core.Player
    GetPlayers() []*core.Player
}

type MapEngineInterface interface {
    CalculatePath(startPos int, steps int) (PathResultInterface, error)
}
```

## Queue

```go
type Queue struct {
    items []ExecutableAction
}

func (q *Queue) Push(action ExecutableAction)
func (q *Queue) Pop() ExecutableAction
func (q *Queue) IsEmpty() bool
```

用于处理衍生动作：
- 陷阱触发 → DamageAction
- 落地事件 → AddBuffAction

## 测试

```bash
go test ./internal/engine/action/... -v
```