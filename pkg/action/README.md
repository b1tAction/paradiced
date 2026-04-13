# Action 包

Action系统是游戏效果的核心抽象层，所有副作用（Buff/Item/Event/Faction被动）都通过Action执行，支持拦截和事件日志。

## 设计理念

1. **统一接口** - 所有效果源使用相同的 `EffectHandler` 签名
2. **拦截机制** - Buffs/Items可以篡改Action（如隐匿免疫伤害、迷途反向移动）
3. **事件日志** - 生成TurnEventLog供客户端播放动画
4. **衍生动作** - Action执行时可生成新的衍生Action

## 核心类型

### ActionType 枚举

```go
type ActionType int

const (
    ActionDamage     // HP减少（可拦截）
    ActionHeal       // HP恢复
    ActionModifyLP   // LP修改
    ActionMove       // 移动（可拦截）
    ActionAddBuff    // 添加Buff
    ActionRemoveBuff // 移除Buff
    ActionRespawn    // 重生
    ActionSkipTurn   // 跳过回合
    ActionDrawEvent  // 抽取事件
    ActionTeleport   // 传送
    ActionStealBuff  // 偷取Buff
)
```

### Action 接口

```go
type Action interface {
    Type() ActionType    // 动作类型
    CanModify() bool     // 是否可被拦截篡改
    Source() string      // 来源标识（BuffID/ItemID等）
    Target() string      // 目标玩家ID
}
```

### TurnEventLogEntry

```go
type TurnEventLogEntry struct {
    Type     string      // "HPChange", "LPChange", "Move"等
    Target   string      // 目标玩家ID
    Delta    int         // 变化量（负数表示减少）
    Source   string      // 来源标识
    Metadata interface{} // 额外数据（路径、Buff类型等）
}
```

## 使用示例

### 创建DamageAction

```go
player := core.NewPlayer(core.PlayerConfig{UserID: "p1"})
action := NewDamageAction(player, 10, "Event_Trap")

// 执行
ctx := NewActionContext(game, bus, mapEngine)
ctx.ExecuteAction(action)

// 查看日志
for _, entry := range ctx.EventLog.Entries() {
    fmt.Printf("%s: %s -> %d\n", entry.Type, entry.Target, entry.Delta)
}
```

### EffectHandler 示例

```go
// 诅咒Buff：每回合LP-1
func CurseHandler(phase event.Phase, ctx *event.Context) Action {
    if phase != event.PhaseBeforeTurn {
        return nil
    }
    player := ctx.Player.(*core.Player)
    return &ModifyLPAction{
        TargetPlayer: player,
        Amount:       -1,
        SourceID:     "Buff_Curse",
    }
}

// 迷途Buff：反向移动
func LostHandler(phase event.Phase, ctx *event.Context) Action {
    if phase != event.PhaseOnMove {
        return nil
    }
    action := ctx.Get("current_action")
    if moveAction, ok := action.(*MoveAction); ok {
        moveAction.Steps = -moveAction.Steps // 篡改
    }
    return nil
}

// 隐匿Buff：免疫伤害
func HiddenHandler(phase event.Phase, ctx *event.Context) Action {
    if phase != event.PhasePreDamage {
        return nil
    }
    action := ctx.Get("current_action")
    if dmgAction, ok := action.(*DamageAction); ok {
        dmgAction.Amount = 0 // 篡改：伤害归零
        dmgAction.BlockedBy = "Buff_Hidden"
    }
    return nil
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
记录LogEntry（BlockedBy = "Buff_Hidden"）
```

## 包结构

```
pkg/action/
├── action.go            # 核心接口定义

internal/engine/action/
├── action.go            # ExecutableAction接口
├── types.go             # 具体Action类型实现
├── context.go           # ActionContext（执行上下文）
├── queue.go             # Queue（衍生动作队列）
├── turn_event_log.go    # TurnEventLog
└── types_test.go        # 测试
```

## 与其他系统集成

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