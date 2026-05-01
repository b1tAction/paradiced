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
    ActionAddItem    ActionType = "add_item"     // Add Item to inventory
    ActionRemoveItem ActionType = "remove_item"  // Remove Item from inventory
    ActionDrawBuff   ActionType = "draw_buff"    // Draw random Buff from pool
    ActionDiceUpgrade ActionType = "dice_upgrade" // Upgrade dice type
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
    SourceID     string         // "DiceRoll" / "DiceRollCheckpoint"
    targetPos    int            // 私有字段，执行后设置
    path         []int          // 私有字段，计算后的路径
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
    Damage       int            // 坔落伤害（传递给衍生 PiercingDamageAction）
    SourceID     string         // "FragileCell"
}
```

- Fragile块坠落时使用
- `Execute()` 不直接扣 HP，而是衍生 `PiercingDamageAction`（不可拦截）来执行伤害
- `LogEntry()` 只记录语义信息（position），hp_change 由 DamageAction LogEntry 承担

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
- `PreTriggerPhase() = PhasePreDamage` - Thorns handler intercepts at PreDamage on BossPlayer and pushes derived `PiercingDamageAction` for reflect
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
- `PreTriggerPhase() = PhaseAnyTime` - No handler intercepts BossAttackAction at PhasePreDamage
- `Execute()` 不直接扣 HP，而是衍生 `DamageAction` 来执行伤害
- `LogEntry()` 只记录语义信息（attack_type, target），hp_change 由 DamageAction LogEntry 承担
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

### AddItemAction

```go
type AddItemAction struct {
    targetPlayer *core.Player    // 私有字段
    ItemType     constants.ItemType
    SourceID     string          // "CheckpointTreasure", "Event_Relic"
}
```

- `CanModify() = false` - Item加入不可拦截
- `PreTriggerPhase() = PhaseAnyTime` - 不触发拦截Phase
- `PostTriggerPhase() = PhaseAnyTime` - 无入场效果
- `Execute()` → 创建 `core.NewItem(ItemType)`，调用 `ctx.OnAddItem(player, item)` callback
- OnAddItem callback 由 HSM 层注入（`game.ApplyItemToPlayer`）：1. `player.AddItem(item)` 2. `game.SubscribeItem(player, item)` EventBus订阅
- DrawItemAction 现不再直接 AddItem，改为 `PushDerivedAction(AddItemAction)` 走完整生命周期

### RemoveItemAction

```go
type RemoveItemAction struct {
    targetPlayer *core.Player    // 私有字段
    ItemType     constants.ItemType
    SourceID     string          // "Item_Consumed", "Event_Thief"
}
```

- `CanModify() = false` - Item移除不可拦截
- `PreTriggerPhase() = PhaseAnyTime` - 不触发拦截Phase
- `PostTriggerPhase() = PhaseAnyTime`
- `Execute()` → 调用 `ctx.OnRemoveItem(player, ItemType)` callback
- OnRemoveItem callback 由 HSM 层注入：在 Inventory 中按 Type 查找，调用 `game.RemoveItemFromPlayer(player, item)`（1. UnsubscribeItem 2. player.RemoveItem）
- 物品使用后自动消耗：`MainActionState.OnUseItem` 执行 Handler 后追加 `RemoveItemAction`

### DrawBuffAction

```go
type DrawBuffAction struct {
    targetPlayer *core.Player    // 私有字段
    SourceID     string          // "Event_TasteTest"
    DrawnType    constants.BuffType  // 抽取结果（Execute后设置）
}
```

- `CanModify() = true`
- `PreTriggerPhase() = PhaseAnyTime` - 抽取本身无需拦截，隐匿拦截点在后续 AddBuffAction 的 `PhasePreBuffApplied`
- `PostTriggerPhase() = PhaseAnyTime` - Buff入场效果由后续 AddBuffAction 的 PostTrigger 处理
- `Execute()` → 使用 `DrawEngine.DrawWithProb(BuffPool, ...)` 从 BuffPool 随机抽取 BuffType
- 抽取后 `PushDerivedAction(AddBuffAction)` → AddBuffAction 走完整 Buff 生命周期
- BuffPool 由 `BuffRegistry.BuildBuffPool()` 构建，仅包含 `IsDraw()` 的 Buff（排除 DeathMark/Thorns）
- 类似 DrawEventAction 的设计：抽取 + DerivedAction 链路

### DiceUpgradeAction

```go
type DiceUpgradeAction struct {
    targetPlayer *core.Player    // 私有字段
    SourceID     string          // "Item_DiceUpgrade"
    FromDice    rng.DiceType     // 原始骰子类型
    ToDice      rng.DiceType     // 升级后类型（Execute时计算）
}
```

- `CanModify() = false` - 骰子升级不可拦截
- `PreTriggerPhase() = PhaseAnyTime`
- `PostTriggerPhase() = PhaseAnyTime`
- `Execute()` → 计算升级路径（Wood→Copper→Silver→Gold），Gold不能再升级
- 升级结果写入 ActionContext Metadata：`dice_upgrade_to`, `dice_upgrade_from`, `dice_upgrade_player`
- HSM 层读取 metadata 更新 DiceManager 中玩家的 DiceType

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
    BuffPool      []*rng.EvaluatedItem // Buff池（DrawBuffAction）
    ActionQueue   *Queue            // 衍生动作队列
    ProbGood      float64           // Good池概率权重
    ProbNeutral   float64           // Neutral池概率权重
    ProbBad       float64           // Bad池概率权重

    // Buff lifecycle callbacks - injected by HSM layer
    OnAddBuff    func(player *core.Player, buff *core.Buff)
    OnRemoveBuff func(player *core.Player, buffType constants.BuffType) *core.Buff
    GetBuffDuration func(buffType constants.BuffType) int

    // Item lifecycle callbacks - injected by HSM layer
    OnAddItem    func(player *core.Player, item *core.Item)
    OnRemoveItem func(player *core.Player, itemType constants.ItemType) *core.Item
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

// AddBuffAction - 添加前可被隐匿拦截
func (a *AddBuffAction) PreTriggerPhase() constants.Phase { return constants.PhasePreBuffApplied }
func (a *AddBuffAction) PostTriggerPhase() constants.Phase { return constants.PhasePostBuffApplied }

// RemoveBuffAction - 移除前触发亡语，移除后触发PostBuffRemoved
func (a *RemoveBuffAction) PreTriggerPhase() constants.Phase { return constants.PhasePreBuffRemoved }
func (a *RemoveBuffAction) PostTriggerPhase() constants.Phase { return constants.PhasePostBuffRemoved }

// DrawBuffAction - 抽取本身无需拦截，后续AddBuffAction处理PhasePreBuffApplied
func (a *DrawBuffAction) PreTriggerPhase() constants.Phase { return constants.PhaseAnyTime }
func (a *DrawBuffAction) PostTriggerPhase() constants.Phase { return constants.PhaseAnyTime }

// AddItemAction - 不可拦截
func (a *AddItemAction) PreTriggerPhase() constants.Phase { return constants.PhaseAnyTime }
func (a *AddItemAction) PostTriggerPhase() constants.Phase { return constants.PhaseAnyTime }

// RemoveItemAction - 不可拦截
func (a *RemoveItemAction) PreTriggerPhase() constants.Phase { return constants.PhaseAnyTime }
func (a *RemoveItemAction) PostTriggerPhase() constants.Phase { return constants.PhaseAnyTime }

// DiceUpgradeAction - 不可拦截
func (a *DiceUpgradeAction) PreTriggerPhase() constants.Phase { return constants.PhaseAnyTime }
func (a *DiceUpgradeAction) PostTriggerPhase() constants.Phase { return constants.PhaseAnyTime }
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