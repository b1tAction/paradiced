# Action 实现包

internal/engine/action 包提供了 Action 接口的具体实现。

## 文件结构

| 文件 | 说明 |
|------|------|
| `action.go` | ExecutableAction接口定义，ActionType常量 |
| `types.go` | 具体Action类型（DamageAction, HealAction等） |
| `context.go` | ActionContext执行上下文 |
| `queue.go` | Queue衍生动作队列 |

## ActionType 常量

ActionType 使用 snake_case string 类型（定义在 pkg/action）：

```go
const (
    ActionDamage     ActionType = "damage"
    ActionHeal       ActionType = "heal"
    ActionModifyLP   ActionType = "modify_lp"
    ActionMove       ActionType = "move"
    ActionAddBuff    ActionType = "add_buff"
    ActionRemoveBuff ActionType = "remove_buff"
    ActionRespawn    ActionType = "respawn"
    ActionSkipTurn   ActionType = "skip_turn"
    ActionDrawEvent  ActionType = "draw_event"
    ActionTeleport   ActionType = "teleport"
    ActionStealBuff  ActionType = "steal_buff"
    ActionFellDown   ActionType = "fell_down"
    ActionUnknown    ActionType = "unknown"
)
```

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

### RespawnAction

```go
type RespawnAction struct {
    TargetPlayer  *core.Player // 玩家
    CheckpointPos int          // 检查点位置
    SourceID      string       // "DeathRespawn", "FragileRespawn"
}
```

- 玩家死亡重生时使用

### FellDownAction

```go
type FellDownAction struct {
    TargetPlayer *core.Player // 玩家
    Position     int          // 坠落位置
    Damage       int          // 坠落伤害
    SourceID     string       // "FragileCell"
}
```

- Fragile块坠落时使用

## ActionContext

```go
type ActionContext struct {
    *util.Metadata              // 嵌入，支持扩展存储

    Game        protocol.Game   // 游戏接口（获取全局日志）
    EventBus    *event.EventBus // 用于拦截
    MapEngine   protocol.MapEngine // 用于移动计算
    ActionQueue *Queue          // 衍生动作队列
}
```

### ExecuteAction 流程

**设计原则：谁产生时机，谁发布Phase**

1. **PreTrigger阶段** - 发布Phase供拦截（如 `PhasePreDamage`、`PhasePreMove`）
   - 若 `PreTriggerPhase() != PhaseAnyTime`，则 Publish 到 EventBus
   - 检查 `action_blocked` 标志，若被阻断则跳过执行
2. 执行 `Execute(ctx)`
3. **PostTrigger阶段** - 发布Phase供生命周期事件（如 `PhaseOnBuffApplied`、`PhaseOnBuffRemoved`）
4. 记录 `LogEntry()` 到全局 GameLog（通过 `protocol.Game.GetGameLog()`）
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

### LogEntry 方法实现

所有 Action 的 `LogEntry()` 返回 `gamelog.LogEntry`，ActionType 使用 `string(a.Type())`：

```go
func (a *DamageAction) LogEntry() gamelog.LogEntry {
    metadata := util.NewMetadata()
    metadata.SetString("blocked_by", a.BlockedBy)
    metadata.SetBool("piercing", a.IsPiercing)

    return gamelog.LogEntry{
        Timestamp:  time.Now(),
        Type:       gamelog.EntryTypeAction,
        ActionType: string(a.Type()), // "damage"
        Target:     a.TargetPlayer.UserID,
        Delta:      -a.Amount,
        Source:     a.SourceID,
        Metadata:   metadata,
    }
}
```

### 接口定义（避免循环依赖）

```go
type Game interface {
    GetCurrentPlayer() interface{}
    GetPlayer(id string) interface{}
    GetPlayers() []interface{}
    GetGameLog() *gamelog.GameLog  // 获取全局日志
}

type MapEngine interface {
    CalculatePath(startPos int, steps int) (PathResult, error)
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

## Metadata 契约

**重要**：`ActionContext.Metadata` 字段使用遵循契约文档定义。

详见：[doc/metadata/action_context.md](../../../doc/metadata/action_context.md) - ActionContext.Metadata 契约

ActionContext.Metadata 主要用于：
- 存储当前Action信息（传递给EventBus）
- 执行过程中的临时标记

**LogEntry.Metadata** 契约：
详见：[doc/metadata/logentry.md](../../../doc/metadata/logentry.md) - LogEntry.Metadata 契约（客户端可见字段）

新增 ActionType 的 Metadata 字段时：
1. 在 LogEntry 契约文档更新表格
2. 同步更新 TypeScript 类型定义
3. 更新 `internal/net/builder.go` 的 `buildAction()` 方法

## 相关文档

- [pkg/action/README.md](../../../pkg/action/README.md) - Action 接口层
- [pkg/gamelog/README.md](../../../pkg/gamelog/README.md) - GameLog 系统