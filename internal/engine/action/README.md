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

**设计原则：谁产生时机，谁发布Phase**

1. **PreTrigger阶段** - 发布Phase供拦截（如 `PhasePreDamage`、`PhasePreMove`）
   - 若 `PreTriggerPhase() != PhaseAnyTime`，则 Publish 到 EventBus
   - 检查 `action_blocked` 标志，若被阻断则跳过执行
2. 执行 `Execute(ctx)`
3. **PostTrigger阶段** - 发布Phase供生命周期事件（如 `PhaseOnBuffApplied`、`PhaseOnBuffRemoved`）
4. 记录 `LogEntry()` 到 EventLog
5. 处理 ActionQueue 中的衍生动作

### Phase方法实现

```go
// DamageAction - 伤害前可被拦截
func (a *DamageAction) PreTriggerPhase() event.Phase {
    if a.IsPiercing { return event.PhaseAnyTime } // 穿透伤害不触发拦截
    return event.PhasePreDamage
}
func (a *DamageAction) PostTriggerPhase() event.Phase { return event.PhaseAnyTime }

// MoveAction - 移动前可被篡改
func (a *MoveAction) PreTriggerPhase() event.Phase { return event.PhasePreMove }
func (a *MoveAction) PostTriggerPhase() event.Phase { return event.PhaseAnyTime }

// AddBuffAction - 添加后触发入场效果
func (a *AddBuffAction) PreTriggerPhase() event.Phase { return event.PhaseAnyTime }
func (a *AddBuffAction) PostTriggerPhase() event.Phase { return event.PhaseOnBuffApplied }

// RemoveBuffAction - 移除前触发亡语
func (a *RemoveBuffAction) PreTriggerPhase() event.Phase { return event.PhaseOnBuffRemoved }
func (a *RemoveBuffAction) PostTriggerPhase() event.Phase { return event.PhaseAnyTime }
```

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