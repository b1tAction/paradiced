# Action 实现包

internal/engine/action 包提供了 Action 接口的具体实现。

## 文件结构

| 文件 | 说明 |
|------|------|
| `action.go` | Action接口定义，ActionType常量 |
| `types.go` | 具体Action类型（DamageAction, HealAction等） |
| `context.go` | ActionContext执行上下文 |
| `queue.go` | Queue衍生动作队列 |

## Action 接口

```go
type Action interface {
    Type() constants.ActionType
    CanModify() bool
    Source() string
    Target() string           // 目标玩家UUID
    TargetPlayer() *core.Player // 目标玩家实例（用于创建event.Context）
    PreTriggerPhase() constants.Phase
    PostTriggerPhase() constants.Phase
    Execute(ctx *ActionContext) error
    LogEntry() gamelog.LogEntry
}
```

**TargetPlayer() 方法**：返回 Action 的目标玩家实例。ExecuteAction 在 PreTrigger/PostTrigger 阶段使用 `action.TargetPlayer()` 创建 `event.NewContext(action.TargetPlayer())`，确保 Handler 收到正确的 Player 实例。

**设计原则**：Action struct 中目标玩家字段使用私有名 `targetPlayer`，通过公开方法 `TargetPlayer()` 访问。这避免了 Go 中字段名和方法名冲突的问题。

## ActionType 常量

ActionType 使用 snake_case string 类型（定义在 pkg/constants）：

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
    ActionDrawItem   ActionType = "draw_item"
    ActionDeath      ActionType = "death"
    ActionBossDamage ActionType = "boss_damage"  // Player attacks Boss
    ActionBossAttack ActionType = "boss_attack"  // Boss attacks player
    ActionBossSkill  ActionType = "boss_skill"   // Boss uses skill
    ActionUnknown    ActionType = "unknown"
)
```

## Action类型详解

### DamageAction

```go
type DamageAction struct {
    targetPlayer *core.Player   // 私有字段，通过TargetPlayer()方法访问
    SourceID     string         // "Event_Trap", "Buff_Corrupt"
    Amount       int            // 可被拦截器修改
    IsPiercing   bool           // true则不可拦截
    BlockedBy    string         // 拦截器设置的阻断来源
}
```

- `CanModify() = !IsPiercing && Amount > 0`
- `TargetPlayer()` returns `targetPlayer`
- 被隐匿、护盾等拦截时设置 `Amount = 0`

### HealAction

```go
type HealAction struct {
    targetPlayer *core.Player   // 私有字段
    SourceID     string         // "Buff_Rain", "Item_HealingPotion"
    Amount       int
}
```

- `CanModify() = Amount > 0`

### ModifyLPAction

```go
type ModifyLPAction struct {
    targetPlayer *core.Player   // 私有字段
    SourceID     string         // "Buff_Divine", "Buff_Curse"
    Amount       int            // +1 or -1
}
```

- `CanModify() = false` - LP修改不可拦截

### MoveAction

```go
type MoveAction struct {
    targetPlayer *core.Player   // 私有字段
    Steps        int            // 可为负数（迷途反向）
    SourceID     string         // "DiceRoll"
    TargetPos    int            // 执行后设置
    Path         []int          // 计算后的路径
    Overtaken    []*core.Player // 反超的玩家（白虎劫运）
}
```

- `CanModify() = Steps != 0`
- 迷途Buff可设置 `Steps = -Steps` 反向移动

### AddBuffAction / RemoveBuffAction

```go
type AddBuffAction struct {
    targetPlayer *core.Player    // 私有字段
    BuffType     constants.BuffType
    Duration     int
    SourceID     string
}

type RemoveBuffAction struct {
    targetPlayer *core.Player    // 私有字段
    BuffType     constants.BuffType
    SourceID     string
}
```

- `CanModify() = false` - Buff操作不可拦截

### TeleportAction

```go
type TeleportAction struct {
    targetPlayer *core.Player   // 私有字段
    TargetPos    int
    SourceID     string         // "Item_AnyDoor"
}
```

- 用于任意门等道具

### StealBuffAction

```go
type StealBuffAction struct {
    targetPlayer *core.Player   // 被偷取者（私有字段）
    SourcePlayer *core.Player   // 偷取者（白虎玩家）
    SourceID     string         // "Faction_BaiHu"
    StolenBuff   *core.Buff     // 执行后设置
}
```

- 白虎"劫运"阵营被动

### RespawnAction

```go
type RespawnAction struct {
    targetPlayer  *core.Player  // 私有字段
    CheckpointPos int           // 检查点位置
    SourceID      string        // "DeathRespawn", "FragileRespawn"
}
```

- 玩家死亡重生时使用

### FellDownAction

```go
type FellDownAction struct {
    targetPlayer *core.Player   // 私有字段
    Position     int            // 坔落位置
    Damage       int            // 坔落伤害
    SourceID     string         // "FragileCell"
}
```

- Fragile块坠落时使用

### DeathAction

```go
type DeathAction struct {
    targetPlayer *core.Player   // 私有字段
    SourceID     string         // 死亡来源
    Position     int            // 死亡位置
}
```

- `TargetPlayer()` returns `targetPlayer` (死亡玩家)

### BossDamageAction

```go
type BossDamageAction struct {
    SourcePlayer  *core.Player  // Player attacking the boss
    targetPlayer  *core.Player  // Boss player receiving damage（私有字段）
    Damage        int           // Damage amount (dice steps, x2 if crit)
    IsCrit        bool          // Whether this is a critical hit
    SourceID      string        // "boss_damage"
}
```

- `CanModify() = false` - Boss damage cannot be intercepted
- `PreTriggerPhase() = PhasePreDamage` - Thorns handler intercepts at PreDamage on BossPlayer (publishes PhasePreDamage to Boss, Thorns Buff handler pushes derived BossAttackAction for reflect)
- `PostTriggerPhase() = PhaseAnyTime` - No post-trigger for boss damage
- `TargetPlayer()` returns `targetPlayer` (Boss player)
- Used when player is on Boss cell and rolls dice

### BossAttackAction

```go
type BossAttackAction struct {
    SourcePlayer *core.Player         // Boss player (attacker)
    targetPlayer *core.Player         // Player receiving damage（私有字段）
    Damage       int                  // 1 for normal, 2 for crit
    AttackType   constants.BossAttackType // "normal"/"crit"/"skill"
    SourceID     string               // "boss_normal"/"boss_crit"
}
```

- `CanModify() = false` - Boss attack cannot be modified
- `PreTriggerPhase() = PhasePreDamage` - Can be intercepted by 隐匿 Buff
- `TargetPlayer()` returns `targetPlayer` (target player)
- Used in Boss counter-attack (normal/crit attacks on players)

### BossSkillAction

```go
type BossSkillAction struct {
    SourcePlayer *core.Player         // Boss player
    SkillType    constants.BossSkillType // "thunder"/"curse"/"lost"/"rest"
    TargetIDs    []string             // Target player IDs
    SourceID     string               // "boss_skill_thunder" etc.
    Targets      []*core.Player       // Target players
}
```

- `CanModify() = false` - Boss skills cannot be intercepted
- `PreTriggerPhase() = PhaseAnyTime` - Boss skills cannot be intercepted
- `TargetPlayer()` returns `SourcePlayer` (Boss player as actor)
- LogEntry record only; actual skill effect handled by BossRegistry skill handlers

## ActionContext

```go
type ActionContext struct {
    *util.Metadata              // 嵌入，支持扩展存储

    Game          protocol.Game     // 游戏接口（获取全局日志）
    EventBus      *event.EventBus   // 用于拦截
    MapEngine     *gamemap.MapEngine // 用于移动计算（直接类型）
    DrawEngine    *rng.DrawEngine   // 用于随机抽取（事件、Buff、道具）
    EventPool     []*rng.EvaluatedItem // 事件池（DrawEventAction）
    ItemPool      []*rng.EvaluatedItem // 道具池（DrawItemAction）
    ActionQueue   *Queue            // 衍生动作队列
    ProbGood      float64           // Good池概率权重
    ProbNeutral   float64           // Neutral池概率权重
    ProbBad       float64           // Bad池概率权重

    // Buff lifecycle callbacks - injected by HSM layer
    OnAddBuff    func(player *core.Player, buff *core.Buff)
    OnRemoveBuff func(player *core.Player, buffType constants.BuffType) *core.Buff
    GetBuffDuration func(buffType constants.BuffType) int
}
```

### ExecuteAction 流程

**设计原则：谁产生时机，谁发布Phase**

1. **PhasePreAction 死亡拦截** - 如果 Action 的 `TargetPlayer` 已死亡（`IsDead=true`），且不是 RemoveBuffAction/RespawnAction，则发布 `PhasePreAction` 供 DeathMark 阻断
   - 使用 `action.TargetPlayer()` 获取目标玩家
   - 使用 `event.NewContext(action.TargetPlayer())` 创建触发上下文
2. **PreTrigger阶段** - 发布Phase供拦截（如 `PhasePreDamage`、`PhasePreMove`）
   - 若 `PreTriggerPhase() != PhaseAnyTime`，则 Publish 到 EventBus
   - 使用 `event.NewContext(action.TargetPlayer())` 创建触发上下文
   - 检查 `action_blocked` 标志，若被阻断则跳过执行
3. 执行 `Execute(ctx)`
4. **PostTrigger阶段** - 发布Phase供生命周期事件（如 `PhasePostBuffApplied`、`PhasePreBuffRemoved`）
   - 使用 `event.NewContext(action.TargetPlayer())` 创建触发上下文
   - AddBuffAction 设置 `applied_buff_type`；RemoveBuffAction 设置 `removed_buff_type`
5. 记录 `LogEntry()` 到全局 GameLog（通过 `protocol.Game.GetGameLog()`）
6. 处理 ActionQueue 中的衍生动作

**关键变更**：PreTrigger/PostTrigger 阶段使用 `action.TargetPlayer()` 创建 `event.NewContext()`，而非之前已删除的 `ctx.CurrentPlayer`。这确保 Handler 总能收到正确的 Player 实例。

### Phase方法实现

```go
// DamageAction - 伤害前可被拦截
func (a *DamageAction) PreTriggerPhase() constants.Phase {
    if a.IsPiercing { return constants.PhaseAnyTime } // 穿透伤害不触发拦截
    return constants.PhasePreDamage
}
func (a *DamageAction) PostTriggerPhase() constants.Phase { return constants.PhaseAnyTime }

// BossDamageAction - 玩家攻击Boss，PreDamage发布到BossPlayer（Thorns handler可响应）
func (a *BossDamageAction) PreTriggerPhase() constants.Phase { return constants.PhasePreDamage }
func (a *BossDamageAction) PostTriggerPhase() constants.Phase { return constants.PhaseAnyTime }

// MoveAction - 移动前可被篡改
func (a *MoveAction) PreTriggerPhase() constants.Phase { return constants.PhasePreMove }
func (a *MoveAction) PostTriggerPhase() constants.Phase { return constants.PhaseAnyTime }

// AddBuffAction - 添加后触发入场效果
func (a *AddBuffAction) PreTriggerPhase() constants.Phase { return constants.PhaseAnyTime }
func (a *AddBuffAction) PostTriggerPhase() constants.Phase { return constants.PhasePostBuffApplied }

// RemoveBuffAction - 移除前触发亡语
func (a *RemoveBuffAction) PreTriggerPhase() constants.Phase { return constants.PhasePreBuffRemoved }
func (a *RemoveBuffAction) PostTriggerPhase() constants.Phase { return constants.PhaseAnyTime }
```

### LogEntry 方法实现

所有 Action 的 `LogEntry()` 返回 `gamelog.LogEntry`，ActionType 使用 `string(a.Type())`。

## Queue

```go
type Queue struct {
    items []Action
}

func (q *Queue) Push(action Action)
func (q *Queue) Pop() Action
func (q *Queue) IsEmpty() bool
func (q *Queue) Len() int
```

用于处理衍生动作：
- 陷阱触发 → DamageAction
- 落地事件 → AddBuffAction

## 测试

```bash
GOMODCACHE=${workdir}/.gomodcache go test ./internal/engine/action/... -v
```

## Metadata 契约

**重要**：`ActionContext.Metadata` 字段使用遵循契约文档定义。

详见：[doc/metadata/action_context.md](../../../doc/metadata/action_context.md) - ActionContext.Metadata 契约

ActionContext.Metadata 主要用于：
- 存储当前Action信息（传递给EventBus）
- 执行过程中的临时标记（如 `buff_duration_extended`）

**LogEntry.Metadata** 契约：
详见：[doc/metadata/logentry.md](../../../doc/metadata/logentry.md) - LogEntry.Metadata 契约（客户端可见字段）

新增 ActionType 的 Metadata 字段时：
1. 在 LogEntry 契约文档更新表格
2. 同步更新 TypeScript 类型定义
3. 更新 `internal/net/builder.go` 的 `buildAction()` 方法

## 相关文档

- [pkg/constants/README.md](../../../pkg/constants/README.md) - 常量枚举类型
- [pkg/gamelog/README.md](../../../pkg/gamelog/README.md) - GameLog 系统