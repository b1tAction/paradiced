# Action 包

Action系统是游戏效果的核心抽象层，所有副作用（Buff/Item/Event/Faction被动）都通过Action执行，支持拦截和全局日志。

## 设计理念

1. **统一接口** - 所有效果源使用相同的 `Action` 签名
2. **拦截机制** - Buffs/Items可以篡改Action（如隐匿免疫伤害、迷途反向移动）
3. **全局日志** - 通过 GameLog 记录所有事件，供客户端播放动画
4. **衍生动作** - Action执行时可生成新的衍生Action

## 核心类型

### ActionType 字符串类型

使用 snake_case 命名，便于 JSON 序列化：

```go
type ActionType string

const (
    ActionDamage     ActionType = "damage"      // HP减少（可拦截）
    ActionHeal       ActionType = "heal"        // HP恢复
    ActionModifyLP   ActionType = "modify_lp"   // LP修改
    ActionMove       ActionType = "move"        // 移动（可拦截）
    ActionAddBuff    ActionType = "add_buff"    // 添加Buff
    ActionRemoveBuff ActionType = "remove_buff" // 移除Buff
    ActionRespawn    ActionType = "respawn"     // 重生
    ActionSkipTurn   ActionType = "skip_turn"   // 跳过回合
    ActionDrawEvent  ActionType = "draw_event"  // 抽取事件
    ActionTeleport   ActionType = "teleport"    // 传送
    ActionStealBuff  ActionType = "steal_buff"  // 偷取Buff
    ActionFellDown   ActionType = "fell_down"   // Fragile坠落
    ActionUnknown    ActionType = "unknown"     // 未知类型
)
```

### Action 接口

```go
type Action interface {
    Type() ActionType            // 动作类型（返回 snake_case string）
    CanModify() bool             // 是否可被拦截篡改
    Source() string              // 来源标识（BuffID/ItemID等）
    Target() string              // 目标玩家ID
    PreTriggerPhase() event.Phase  // 执行前触发Phase（用于拦截）
    PostTriggerPhase() event.Phase // 执行后触发Phase（用于生命周期事件）
}
```

### Phase 设计原则

**谁产生时机，谁发布Phase：**
- HSM发布状态时机Phase（BeforeTurn, OnLand, AfterTurn）
- Action发布动作时机Phase（PreDamage, PreEvent, PreMove, OnBuffApplied, OnBuffRemoved）

```go
// HSM发布的Phase（状态时机）
PhaseBeforeTurn  // TurnUpkeep.Enter() - 回合开始前
PhaseOnLand      // TurnLanded.Enter() - 落地后
PhaseAfterTurn   // TurnEnd.Enter() - 回合结束后

// Action发布的Phase（动作时机）
PhasePreDamage     // DamageAction - 伤害应用前（隐匿、护盾拦截）
PhasePreEvent      // DrawEventAction - 事件触发前（辟邪）
PhasePreMove       // MoveAction - 移动前（迷途反向）
PhaseOnBuffApplied // AddBuffAction - Buff添加后（入场效果）
PhaseOnBuffRemoved // RemoveBuffAction - Buff移除前（亡语）
```

### LogEntry 接口

```go
type LogEntry interface {
    LogEntry() gamelog.LogEntry  // 生成日志条目
}
```

## 使用示例

### 创建DamageAction

```go
player := core.NewPlayer(core.PlayerConfig{UserID: "p1"})
action := NewDamageAction(player, 10, "Event_Trap")

// 执行（自动记录到全局 GameLog）
ctx := NewActionContext(game, bus, mapEngine)
ctx.ExecuteAction(action)

// 查看全局日志
segment := game.Log.GetSegment(1, 0)
for _, entry := range segment.Entries {
    fmt.Printf("%s: %s -> %d\n", entry.ActionType, entry.Target, entry.Delta)
}
```

### EffectHandler 示例

```go
// 诅咒Buff：每回合LP-1
func CurseHandler(phase event.Phase, ctx *event.Context) {
    if phase != event.PhaseBeforeTurn {
        return
    }
    player := ctx.Player.(*core.Player)
    ctx.AddDerivedAction(&ModifyLPAction{
        TargetPlayer: player,
        Amount:       -1,
        SourceID:     "Buff_Curse",
    })
}

// 迷途Buff：反向移动（在PreMove时篡改）
func LostHandler(phase event.Phase, ctx *event.Context) {
    if phase != event.PhasePreMove {
        return
    }
    action := ctx.Get("current_action")
    if moveAction, ok := action.(*MoveAction); ok {
        moveAction.Steps = -moveAction.Steps // 篡改
    }
}

// 隐匿Buff：免疫伤害
func HiddenHandler(phase event.Phase, ctx *event.Context) {
    if phase != event.PhasePreDamage {
        return
    }
    action := ctx.Get("current_action")
    if dmgAction, ok := action.(*DamageAction); ok {
        dmgAction.Amount = 0 // 篡改：伤害归零
        dmgAction.BlockedBy = "Buff_Hidden"
    }
}

// 不死Buff：拦截死亡，原地复活（生成多个Action）
func UndyingHandler(phase event.Phase, ctx *event.Context) {
    if phase != event.PhasePreRespawn {
        return
    }
    
    ctx.SetBool("action_blocked", true) // 拦截原Respawn
    
    player := ctx.Player.(*core.Player)
    ctx.AddDerivedAction(NewHealAction(player, player.MaxHP, "Buff_Undying"))
    ctx.AddDerivedAction(NewRemoveBuffAction(player, BuffTypeUndying, "Buff_Undying"))
}
```

## 拦截流程

```
HSM调用 ExecuteAction(DamageAction{Amount: 10})
    ↓
检查 CanModify() = true
    ↓
Publish(PhasePreDamage, playerID, ctx)
    ↓
EventBus遍历订阅者，调用Handler
    ↓
HiddenHandler检测到DamageAction，设置Amount = 0
    ↓
Execute()检测Amount <= 0，跳过伤害
    ↓
记录LogEntry到全局 GameLog（BlockedBy = "Buff_Hidden"）
```

## 包结构

```
pkg/action/
├── action.go            # 核心接口定义（ActionType string）

pkg/gamelog/
├── entry.go             # EntryType, LogEntry
├── segment.go           # TurnSegment
├── log.go               # GameLog

internal/engine/action/
├── action.go            # ExecutableAction接口
├── types.go             # 具体Action类型实现
├── context.go           # ActionContext（执行上下文）
├── queue.go             # Queue（衍生动作队列）
└── types_test.go        # 测试
```

## 与其他系统集成

### GameLog 系统

所有 Action 执行后自动记录到全局 GameLog：

```go
// ActionContext.ExecuteAction() 自动记录
func (ctx *ActionContext) ExecuteAction(action ExecutableAction) error {
    // ... 执行 Action ...
    
    // 自动记录到全局日志
    if ctx.Game != nil {
        ctx.Game.GetGameLog().AddEntry(action.LogEntry())
    }
    return nil
}
```

### Buff系统

BuffHandler签名改为EffectHandler：
```go
type EffectHandler func(phase event.Phase, ctx *event.Context) Action
```

### EventBus系统

新增 `PhaseItemUsed` 用于道具主动使用：
```go
// 玩家使用道具时
ctx.Set("item_id", itemID)
bus.Publish(PhaseItemUsed, playerID, ctx)
```

### HSM系统

StateContext可持有ActionContext引用：
```go
ctx.Set("action_context", actionCtx)
```

## 相关文档

- [pkg/gamelog/README.md](../gamelog/README.md) - GameLog 系统文档
- [internal/engine/action/README.md](../../internal/engine/action/README.md) - Action 实现文档