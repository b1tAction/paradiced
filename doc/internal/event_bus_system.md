# EventBus + Decision 系统实现文档

## 概述

EventBus + Decision 系统是《派乐代》游戏的统一触发机制框架，为所有Buff、道具、阵营被动提供Phase分类和用户确认支持。

## 设计目标

1. **统一Phase系统**：Buff/道具按触发时机分类
2. **用户确认机制**：支持玩家决定是否使用道具或触发技能
3. **多Phase支持**：一个Buff可以注册多个触发时机
4. **策略模式**：通过EventHandler实现高度定制化的Buff效果
5. **Buff生命周期事件**：支持Applied/Removed事件广播
6. **易于扩展**：新增Buff/道具只需声明Phase，无需改核心代码
7. **维护友好**：模块解耦，改动集中

## 文件结构

```
pkg/constants/
├── phase.go          # Phase枚举定义（触发时机）

internal/event/
├── bus.go            # EventBus结构和方法
├── decision.go       # Decision和Option结构
├── context.go        # Context结构
├── event_test.go     # 单元测试

pkg/gamelog/
├── entry.go          # EntryType枚举、LogEntry结构（使用util.Metadata）
├── segment.go        # TurnSegment回合分段
├── log.go            # GameLog全局日志管理器
├── log_test.go       # 单元测试

pkg/action/
├── action.go         # ActionType（string类型，snake_case命名）、Action接口

internal/core/
├── buff.go           # Buff结构 + BuffDefinition（静态元数据）
├── item.go           # Item结构 + ItemDefinition（静态元数据）
├── game_event.go     # GameEvent结构 + EventDefinition（静态元数据）
├── player.go         # Player结构（HP/LP/Buffs/Items）
├── player_test.go    # 单元测试

internal/engine/
├── game.go           # Game实例（EventBus/玩家管理/订阅，ApplyBuffToPlayer/RemoveBuffFromPlayer，GameLog）
├── buff_registry.go  # Buff Registry + HandlerConfig + handlers
├── item_registry.go  # Item Registry + HandlerConfig + handlers
├── event_registry.go # Event Registry + HandlerConfig + handlers
├── action/           # Action系统实现（DamageAction、HealAction、RespawnAction等）
│   ├── action.go     # Action接口（含TargetPlayer方法）
│   ├── types.go      # 具体Action类型实现
│   ├── context.go    # ActionContext（与全局GameLog集成）
│   ├── queue.go      # 衍生动作队列
├── hsm/              # 分层状态机（状态转换、Phase触发）
│   ├── hsm.go        # HSM主结构（状态转换日志记录）
│   ├── turn_states.go # 回合状态（StartTurn/EndTurn、RespawnAction/FellDownAction）
```

## Phase枚举

**设计原则：谁产生时机，谁发布Phase**

Phase 定义在 `pkg/constants/phase.go`，使用 string 类型：

```go
type Phase string  // snake_case values for JSON compatibility

const (
    // ========== HSM发布的Phase（状态时机） ==========
    // 这些Phase由HSM状态机Enter()方法发布
    PhaseBeforeTurn  Phase = "before_turn"  // TurnUpkeep.Enter() - 回合开始前（神眷/诅咒 LP±1, 离火每4回合）
    PhaseOnLand      Phase = "on_land"      // TurnLanded.Enter() - 落地后（落地事件、格子效果）
    PhaseAfterTurn   Phase = "after_turn"   // TurnEnd.Enter() - 回合结束后（甘霖/腐化 HP±1, TickDuration）

    // ========== Action发布的Phase（动作时机） ==========
    // 这些Phase由ActionContext.ExecuteAction()发布
    PhasePreDamage     Phase = "pre_damage"     // DamageAction.Execute() - 伤害应用前（隐匿、护盾拦截）
    PhasePreEvent      Phase = "pre_event"      // DrawEventAction.Execute() - 事件触发前（辟邪、玄武）
    PhasePreMove       Phase = "pre_move"       // MoveAction.Execute() - 移动前（迷途反向）
    PhaseOnBuffApplied Phase = "on_buff_applied" // AddBuffAction.Execute() - Buff添加后（入场效果、连锁反应）
    PhaseOnBuffRemoved Phase = "on_buff_removed" // RemoveBuffAction.Execute() - Buff移除前（亡语）

    // ========== 特殊Phase ==========
    PhaseAnyTime  Phase = "any_time"  // 任何时候可用（道具主动使用）- 玩家手动触发
    PhaseItemUsed Phase = "item_used" // 道具主动使用时触发 - game.UseItem()
)
```

| Phase | 发布者 | 说明 | 需订阅EventBus |
|-------|--------|------|---------------|
| BeforeTurn | HSM | 回合开始前 | ✓ |
| OnLand | HSM | 落地后 | ✓ |
| AfterTurn | HSM | 回合结束后 | ✓ |
| PreDamage | Action | 受伤前 | ✓ |
| PreEvent | Action | 事件触发前 | ✓ |
| PreMove | Action | 移动前 | ✓ |
| OnBuffApplied | Action | Buff添加后 | ✓ |
| OnBuffRemoved | Action | Buff移除前 | ✓ |
| AnyTime | - | 任何时候可用 | ❌（主动触发） |
| ItemUsed | Game | 道具使用时 | ✓ |

## 多Phase支持

Buff 采用 **Definition + HandlerConfig 分离架构**，支持多Phase触发：

```go
// BuffDefinition 只包含静态元数据（可序列化）
type BuffDefinition struct {
    Type        BuffType
    Eval        constants.Evaluation  // 评估分数
    EnglishName string                // 英文名（snake_case）
    Name        string                // 中文显示名
    Desc        string                // 描述
    Duration    int                   // 默认持续时间
}

// BuffHandlerConfig 包含执行配置和效果处理函数
type BuffHandlerConfig struct {
    Phases      []constants.Phase     // 支持多Phase触发
    Priority    int                   // 执行优先级
    NeedConfirm bool                  // 是否需要用户确认
    Handler     EffectHandler         // 效果处理函数
}
```

注册示例：
```go
GlobalBuffRegistry.RegisterBuff(&BuffDefinition{
    Type:        constants.BuffTypeFire,
    Eval:        constants.EvaluationGood,
    EnglishName: "Fire",
    Name:        "离火",
    Desc:        "朱雀阵营增益，每4回合LP+1",
    Duration:    -1,
}, &BuffHandlerConfig{
    Phases:      []constants.Phase{constants.PhaseBeforeTurn},
    Priority:    10,
    NeedConfirm: false,
    Handler:     handleZhuQueFire,
})
```

Buff 实例存储多个订阅ID：
```go
type Buff struct {
    Type            BuffType
    SubscriptionIDs []string  // 多订阅ID（UUID字符串）
    // ...
}
```

## Buff生命周期管理

通过 `ApplyBuffToPlayer` 和 `RemoveBuffFromPlayer` 方法管理 Buff 的数据层和 EventBus 订阅：

```go
// ApplyBuffToPlayer 流程
func (g *Game) ApplyBuffToPlayer(player, buff) {
    1. player.AddBuff(buff)      // 底层数据添加
    2. g.SubscribeBuff(player, buff) // 挂载到EventBus
}

// RemoveBuffFromPlayer 流程
func (g *Game) RemoveBuffFromPlayer(player, buff) {
    1. g.UnsubscribeBuff(buff)  // 取消订阅
    2. player.RemoveBuff(buff.Type) // 底层数据移除
}
```

**Buff 生命周期 Phase 发布**：

Buff 的生命周期 Phase（`PhasePostBuffApplied`、`PhasePreBuffRemoved`）由 **Action 系统**负责发布：

- `AddBuffAction.PostTriggerPhase()` = `PhasePostBuffApplied` - Buff 添加后发布
- `RemoveBuffAction.PreTriggerPhase()` = `PhasePreBuffRemoved` - Buff 移除前发布

这确保了：
- Phase 发布与实际效果执行在同一 ActionContext 中
- 所有 Buff 生命周期事件通过统一的 Action 执行流程触发
- GameLog 自动记录 Buff 添加/移除操作

## EffectHandler策略模式

通过策略模式实现高度定制化的Buff效果。Handler 在 BuffHandlerConfig 中注册：

```go
// EffectHandler 是统一的效果处理函数签名
// Handler 使用 ctx.SetInt/SetBool/SetString 信号意图，engine层通过Action执行
// 返回 error 用于错误向上传播至 HSM 层
type EffectHandler func(phase constants.Phase, ctx *event.Context) error

// 注册示例：离火 Buff
GlobalBuffRegistry.RegisterBuff(&BuffDefinition{...}, &BuffHandlerConfig{
    Phases:      []constants.Phase{constants.PhaseBeforeTurn},
    Priority:    10,
    NeedConfirm: false,
    Handler:     handleZhuQueFire,  // 朱雀离火：每4回合LP+1
})
```

**Handler 使用示例**：

```go
// 迷途Buff：反向移动（在PreMove时信号）
func handleLostReverse(phase constants.Phase, ctx *event.Context) {
    if phase != constants.PhasePreMove {
        return
    }
    // 信号反向移动意图（engine层执行）
    ctx.SetBool("reverse_movement", true)
}

// 隐匿Buff：免疫伤害（在PreDamage时拦截）
func handleHiddenImmune(phase constants.Phase, ctx *event.Context) {
    if phase != constants.PhasePreDamage {
        return
    }
    // 阻断伤害 Action
    ctx.SetBool("action_blocked", true)
    ctx.SetString("blocked_by", "Buff_Hidden")
}

// 神眷Buff：每回合LP+1
func handleDivineBuff(phase constants.Phase, ctx *event.Context) {
    if phase != constants.PhaseBeforeTurn {
        return
    }
    // 直接修改 LP（无需 Action）
    if ctx.Player != nil {
        ctx.Player.ModifyLP(1)
    }
}

// 离火Buff：朱雀被动，每4回合LP+1
func handleZhuQueFire(phase constants.Phase, ctx *event.Context) {
    if ctx.Player == nil {
        return
    }
    newCount := ctx.Player.IncrementFireCounter()
    if newCount >= 4 {
        ctx.Player.ModifyLP(1)
        ctx.Player.SetFireCounter(0)
    }
}
```

**优势**：
1. **单一职责原则**：Definition 只负责静态元数据，HandlerConfig 负责执行配置和效果逻辑
2. **数据可序列化**：Definition 只有基础字段，可直接 JSON 序列化
3. **消灭特判代码**：阵营逻辑成为 Buff 处理器
4. **统一签名**：所有 Buff/Item/Event 使用相同的 EffectHandler 签名

## Buff/Item与Phase对应

表格中 Phases/Priority/NeedConfirm 来自 HandlerConfig：

| Buff | Phases | Priority | NeedConfirm | 说明 |
|------|--------|----------|-------------|------|
| 神眷 | [BeforeTurn] | 50 | false | 自动LP+1 |
| 诅咒 | [BeforeTurn] | 50 | false | 自动LP-1 |
| 迷途 | [PreMove] | 100 | false | 自动反向 |
| 隐匿 | [PreDamage] | 100 | false | 自动免疫（高优先级） |
| 辟邪 | [PreEvent] | 80 | false | 自动免疫毒瘴 |
| 甘霖 | [AfterTurn] | 50 | false | 每2回合HP+1 |
| 腐化 | [AfterTurn] | 50 | false | 每2回合HP-1 |
| 毒瘴 | [BeforeTurn] | 30 | false | 每回合恶性事件 |
| 离火 | [BeforeTurn] | 10 | false | 每4回合LP+1（定制处理器） |
| 死亡标记 | [PreAction] | 999 | false | 死亡后阻拦后续Action（豁免Respawn/移除自身） |

| Item | Phase | Priority | NeedConfirm | 说明 |
|------|-------|----------|-------------|------|
| 反方向的钟 | AnyTime | 50 | true | 主动使用，需确认目标 |
| 任意门 | OnLand | 60 | true | 落地后使用，需确认目标 |
| 骰子交换 | AnyTime | 40 | true | 主动使用，需确认目标 |
| 骰子升级卡 | BeforeTurn | 70 | true | 回合开始前，需确认 |

## 测试覆盖

### 测试覆盖率统计
```
internal/event:    91.9% statements
internal/core:     93.4% statements
internal/engine:   91.8% statements
```

## 后续扩展

1. **更多EventHandler**：为复杂Buff添加定制处理器
2. **多Phase Buff**：实现如"白虎劫运"等多触发时机Buff
3. **Buff联动**：监听OnBuffApplied/Removed实现联动效果
4. **UI集成**：Decision Prompt格式规范
5. **超时处理**：客户端无响应时自动执行默认选项

---

## GameLog 系统集成

### 统一日志记录

所有游戏效果通过 Action 系统执行，自动记录到全局 GameLog：

```go
// ActionContext.ExecuteAction() 流程
func (ctx *ActionContext) ExecuteAction(action Action) error {
    // 1. PreTrigger阶段 - 发布Phase供拦截
    if action.PreTriggerPhase() != PhaseAnyTime {
        triggerCtx := event.NewContext(action.TargetPlayer())
        triggerCtx.Set("current_action", action)
        triggerCtx.Set("action_context", ctx)
        ctx.EventBus.Publish(action.PreTriggerPhase(), action.Target(), triggerCtx)
    }

    // 2. 执行 Action
    action.Execute(ctx)

    // 3. PostTrigger阶段 - 发布Phase供生命周期事件
    if action.PostTriggerPhase() != PhaseAnyTime {
        triggerCtx := event.NewContext(action.TargetPlayer())
        triggerCtx.Set("current_action", action)
        triggerCtx.Set("action_context", ctx)
        ctx.EventBus.Publish(action.PostTriggerPhase(), action.Target(), triggerCtx)
    }

    // 4. 记录到全局日志
    if ctx.Game != nil {
        ctx.Game.GetGameLog().AddEntry(action.LogEntry())
    }

    return nil
}
```

### ActionType 命名规范

ActionType 使用 `string` 类型，采用 `snake_case` 命名，便于 JSON 序列化：

```go
const (
    ActionDamage     ActionType = "damage"      // 伤害
    ActionHeal       ActionType = "heal"        // 治疗
    ActionModifyLP   ActionType = "modify_lp"   // LP修改
    ActionMove       ActionType = "move"        // 移动
    ActionAddBuff    ActionType = "add_buff"    // 添加Buff
    ActionRemoveBuff ActionType = "remove_buff" // 移除Buff
    ActionRespawn    ActionType = "respawn"     // 重生
    ActionSkipTurn   ActionType = "skip_turn"   // 跳过回合
    ActionDrawEvent  ActionType = "draw_event"  // 抽取事件
    ActionTeleport   ActionType = "teleport"    // 传送
    ActionStealBuff  ActionType = "steal_buff"  // 偷取Buff
    ActionFellDown   ActionType = "fell_down"   // Fragile坠落
)
```

### HSM 状态转换日志

HSM 状态转换自动记录到 GameLog：

```go
// HSM.TransitionTo() 流程
func (hsm *HSM) TransitionTo(targetID StateID, ctx *StateContext) error {
    fromID := hsm.GetCurrentStateID()
    
    // 记录状态转换日志
    if hsm.game.Log != nil {
        hsm.game.Log.LogStateTransition(fromID.String(), targetID.String(), getPlayerID(hsm.turnPlayer))
    }
    
    // ... 状态转换逻辑 ...
}
```

### 回合日志分段

HSM 在回合开始/结束时管理日志分段：

```go
// TurnUpkeepState.Enter() - 开始回合日志
func (s *TurnUpkeepState) Enter(ctx *StateContext) {
    ctx.Game.Log.StartTurn(ctx.Game.State.Round, ctx.Game.State.Turn, player.ID.UUID())
    // ...
}

// TurnEndState.Enter() - 结束回合日志
func (s *TurnEndState) Enter(ctx *StateContext) {
    // ...
    ctx.Game.Log.EndTurn()
}
```

### JSON 输出格式

```json
{
  "segments": [
    {
      "round": 1,
      "turn": 0,
      "player_id": "player1",
      "entries": [
        {"type": "action", "action_type": "modify_lp", "delta": 1, "source": "Buff_Divine"},
        {"type": "action", "action_type": "move", "target": "player1", "delta": 5},
        {"type": "action", "action_type": "fell_down", "target": "player1", "delta": -1},
        {"type": "state", "metadata": {"from": "TurnMoving", "to": "TurnEnd"}}
      ]
    }
  ]
}
```