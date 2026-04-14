# EventBus + Decision 系统实现文档

## 概述

EventBus + Decision 系统是《命运骰子》游戏的统一触发机制框架，为所有Buff、道具、阵营被动提供Phase分类和用户确认支持。

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
pkg/event/
├── phase.go          # Phase枚举定义（9种触发时机）
├── bus.go            # EventBus结构和方法
├── decision.go       # Decision和Option结构
├── context.go        # Context结构
├── bus_test.go       # 单元测试

pkg/gamelog/
├── entry.go          # EntryType枚举、LogEntry结构（使用util.Metadata）
├── segment.go        # TurnSegment回合分段
├── log.go            # GameLog全局日志管理器
├── log_test.go       # 单元测试

pkg/action/
├── action.go         # ActionType（string类型，snake_case命名）、Action接口

internal/core/
├── evaluation.go     # 评分系统（0-100）
├── faction.go        # 阵营定义（四神兽）
├── buff.go           # Buff系统（类型/定义/注册表，支持多Phase）
├── item.go           # Item系统（类型/定义/注册表）
├── event.go          # Event系统（类型/定义/注册表）
├── player.go         # Player结构（HP/LP/Buffs/Items）

internal/engine/
├── game.go           # Game实例（EventBus/玩家管理/订阅，ApplyBuffToPlayer/RemoveBuffFromPlayer，GameLog）
├── action/           # Action系统实现（DamageAction、HealAction、RespawnAction等）
│   ├── action.go     # ExecutableAction接口
│   ├── types.go      # 具体Action类型实现
│   ├── context.go    # ActionContext（与全局GameLog集成）
│   ├── queue.go      # 衍生动作队列
├── hsm/              # 分层状态机（状态转换、Phase触发）
│   ├── hsm.go        # HSM主结构（状态转换日志记录）
│   ├── turn_states.go # 回合状态（StartTurn/EndTurn、RespawnAction/FellDownAction）
├── handlers.go       # EventHandler策略注册表
```

## Phase枚举

**设计原则：谁产生时机，谁发布Phase**

```go
type Phase int

const (
    // ========== HSM发布的Phase（状态时机） ==========
    // 这些Phase由HSM状态机Enter()方法发布
    PhaseBeforeTurn  Phase = iota  // TurnUpkeep.Enter() - 回合开始前（神眷/诅咒 LP±1, 离火每4回合）
    PhaseOnLand                    // TurnLanded.Enter() - 落地后（落地事件、格子效果）
    PhaseAfterTurn                 // TurnEnd.Enter() - 回合结束后（甘霖/腐化 HP±1, TickDuration）

    // ========== Action发布的Phase（动作时机） ==========
    // 这些Phase由ActionContext.ExecuteAction()发布
    PhasePreDamage    // DamageAction.Execute() - 伤害应用前（隐匿、护盾拦截）
    PhasePreEvent     // DrawEventAction.Execute() - 事件触发前（辟邪、玄武）
    PhasePreMove      // MoveAction.Execute() - 移动前（迷途反向）
    PhaseOnBuffApplied  // AddBuffAction.Execute() - Buff添加后（入场效果、连锁反应）
    PhaseOnBuffRemoved  // RemoveBuffAction.Execute() - Buff移除前（亡语）

    // ========== 特殊Phase ==========
    PhaseAnyTime   // 任何时候可用（道具主动使用）- 玩家手动触发
    PhaseItemUsed  // 道具主动使用时触发 - game.UseItem()
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

BuffDefinition 现在支持多Phase触发：
```go
type BuffDefinition struct {
    Type        BuffType
    Phases      []event.Phase  // 支持多Phase触发
    Priority    int
    NeedConfirm bool
    // ...
}
```

Buff 实例存储多个订阅ID：
```go
type Buff struct {
    Type            BuffType
    SubscriptionIDs []string  // 多订阅ID
    // ...
}
```

## Buff生命周期管理

通过 `ApplyBuffToPlayer` 和 `RemoveBuffFromPlayer` 方法管理完整的Buff生命周期：

```go
// ApplyBuffToPlayer 流程（先订阅，后广播）
func (g *Game) ApplyBuffToPlayer(player, buff) {
    1. player.AddBuff(buff)      // 底层数据添加
    2. g.SubscribeBuff(player, buff) // 挂载到EventBus
    3. g.BroadcastBuffApplied(player, buff) // 广播Applied事件
}

// RemoveBuffFromPlayer 流程（先广播，后取消订阅）
func (g *Game) RemoveBuffFromPlayer(player, buff) {
    1. g.BroadcastBuffRemoved(player, buff) // 广播Removed事件
    2. g.UnsubscribeBuff(buff)  // 取消订阅
    3. player.RemoveBuff(buff.Type) // 底层数据移除
}
```

**关键设计**：
- Applied：订阅发生在广播之前，Buff可以听到自己的Applied事件
- Removed：广播发生在取消订阅之前，Buff可以听到自己的removed事件

## EventHandler策略模式

通过策略模式实现高度定制化的Buff效果：

```go
// EventHandler 是定制化的Buff处理函数（无返回值）
// Handler 使用 ctx.AddDerivedAction() 添加衍生 Action
type EventHandler func(phase event.Phase, ctx *event.Context)

// BuffHandlers 注册表
var BuffHandlers = map[core.BuffType]EventHandler{
    core.BuffTypeFire: handleZhuQueFire,  // 朱雀离火：每4回合LP+1
    // 更多处理器...
}
```

**Handler 使用示例**：

```go
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

// 不死Buff：拦截死亡，原地复活（多个衍生Action）
func UndyingHandler(phase event.Phase, ctx *event.Context) {
    if phase != event.PhasePreRespawn {
        return
    }
    
    // 拦截 Respawn
    ctx.SetBool("action_blocked", true)
    
    // 添加多个衍生 Action
    player := ctx.Player.(*core.Player)
    ctx.AddDerivedAction(NewHealAction(player, player.MaxHP, "Buff_Undying"))
    ctx.AddDerivedAction(NewRemoveBuffAction(player, BuffTypeUndying, "Buff_Undying"))
}
```

**优势**：
1. **数据行为分离**：BuffDefinition保持纯数据，可序列化
2. **万能拦截器**：通过修改ctx.Data实现各种机制
3. **消灭特判代码**：阵营逻辑成为Buff处理器
4. **多Action支持**：一个Handler可生成多个衍生Action

## Buff/Item与Phase对应

| Buff | Phases | NeedConfirm | 说明 |
|------|--------|-------------|------|
| 神眷 | [BeforeTurn] | false | 自动LP+1 |
| 诅咒 | [BeforeTurn] | false | 自动LP-1 |
| 迷途 | [PreMove] | false | 自动反向 |
| 隐匿 | [PreDamage] | false | 自动免疫（高优先级） |
| 辟邪 | [PreEvent] | false | 自动免疫毒瘴 |
| 甘霖 | [AfterTurn] | false | 每2回合HP+1 |
| 腐化 | [AfterTurn] | false | 每2回合HP-1 |
| 毒瘴 | [BeforeTurn] | false | 每回合恶性事件 |
| 离火 | [BeforeTurn] | false | 每4回合LP+1（定制处理器） |

| Item | Phase | NeedConfirm | 说明 |
|------|-------|-------------|------|
| 反方向的钟 | AnyTime | true | 主动使用，需确认目标 |
| 任意门 | ItemUsed | true | 落地后使用，需确认目标 |
| 骰子交换 | AnyTime | true | 主动使用，需确认目标 |
| 骰子升级卡 | BeforeTurn | true | 回合开始前，需确认 |

## 测试覆盖

### 测试覆盖率统计
```
pkg/event:       91.9% statements
internal/core:   93.4% statements
internal/engine: 91.8% statements
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
func (ctx *ActionContext) ExecuteAction(action ExecutableAction) error {
    // 1. PreTrigger阶段 - 发布Phase供拦截
    if action.PreTriggerPhase() != PhaseAnyTime {
        ctx.EventBus.Publish(action.PreTriggerPhase(), action.Target(), ctx)
    }
    
    // 2. 执行 Action
    action.Execute(ctx)
    
    // 3. PostTrigger阶段 - 发布Phase供生命周期事件
    if action.PostTriggerPhase() != PhaseAnyTime {
        ctx.EventBus.Publish(action.PostTriggerPhase(), action.Target(), ctx)
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
    ctx.Game.Log.StartTurn(ctx.Game.State.Round, ctx.Game.State.Turn, player.UserID)
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