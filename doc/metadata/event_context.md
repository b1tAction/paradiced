# Context.Metadata 契约（EventBus Handler通信）

`event.Context.Metadata` 用于 Handler 之间传递意图信号，**不发送给客户端**。

**位置**：`internal/event/context.go`

**可见性**：内部（仅后端使用）

---

## 概述

EventBus Handler 通过 Context.Metadata 传递执行意图，下游 Action 根据 Metadata 字段执行具体效果。

```go
// Handler 设置意图信号
func (h *BuffHandler) Execute(ctx *event.Context) {
    ctx.SetInt("hp_change", 1)  // 通知执行HP+1
    ctx.AddDerivedAction(...)    // 生成派生Action
}

// 下游根据意图执行
func ExecuteHPChange(ctx *ActionContext, target *Player) {
    amount := ctx.GetIntOrDefault("hp_change", 0)
    if amount > 0 {
        ExecuteHealAction(target, amount)
    } else {
        ExecuteDamageAction(target, -amount)
    }
}
```

---

## 字段契约表

### HP/LP 变化意图

| 字段 | 类型 | 来源Handler | 用途 | 目标Action |
|------|------|-------------|------|------------|
| `hp_change` | int | Buff/Event Handler | HP变化意图（正数恢复，负数伤害） | HealAction/DamageAction |
| `give_buff_type` | string | Event/Item Handler | 给予Buff类型 | AddBuffAction |
| `give_buff_duration` | int | Event/Item Handler | Buff持续回合 | AddBuffAction |

### Buff 效果信号

| 字段 | 类型 | 来源Buff | 用途 | 目标 |
|------|------|----------|------|------|
| `blocked_by` | string | Buff_Hidden（隐匿） | 阻挡来源标识 | LogEntry.Metadata |
| `action_blocked` | bool | Buff_Hidden / Buff_DeathMark | 动作被阻挡标志 | ActionContext |
| `reverse_movement` | bool | Buff_Lost（迷途） | 反向移动标志（仅日志/调试） | LogEntry.Metadata |
| `current_state` | StepsModifier | HSM (TurnMoving) | 当前移动状态实例（迷途修改Steps） | 迷途handler |
| `draw_bad_event` | bool | Buff_Poison（毒瘴） | 抽取坏事件标志 | DrawEventAction |
| `block_poison_effect` | bool | Buff_Exorcism（辟邪） | 阻挡毒效果标志 | Event Handler |
| `applied_buff_type` | string | ActionContext (PreTrigger) | 被添加的Buff类型标识（用于隐匿Handler判断IsPositive/IsBoss） | Hidden Handler |
| `removed_buff_type` | string | ActionContext (PreTrigger) / HSM (TurnEnd expiry) | 被移除的Buff类型标识 | Divine/Curse Handler |

**反刺(Thorns)反伤机制**：
- Thorns Handler 在 PhasePreDamage（BossPlayer上）触发
- 从 `current_action` 获取 BossDamageAction，计算30%反伤（math.Round）
- 推送衍生 BossAttackAction（Boss→攻击玩家，SourceThornsReflect）
- 衍生 BossAttackAction 走 PhasePreDamage → 隐匿可拦截反伤

**骰子拦截机制 (PhasePreDiceRoll)**：
- RollDiceAction 在构造时计算 Steps（遵循 DamageAction 模式：Amount 构造时设定）
- PhasePreDiceRoll Handler 通过 `current_action` 类型断言为 `*RollDiceAction`，可修改 `Steps` 字段
- 例如"强运"Buff：Handler 断言 `action.(*RollDiceAction)`，将 `Steps` 修改为最大值

### Event 效果信号（DerivedAction 模式）

Event Handler 现通过 `ctx.AddDerivedAction()` 推送具体 Action，而非设置 flag。

| Handler | 之前（flag模式） | 现在（DerivedAction模式） |
|---------|-----------------|--------------------------|
| 遗物(Relic) | `ctx.SetBool("draw_item", true)` | `ctx.AddDerivedAction(NewDrawItemAction(...))` |
| 交换(Exchange) | `ctx.SetBool("swap_position", true)` | `ctx.AddDerivedAction(NewTeleportAction(...))` ×2 |
| 尝一口(TasteTest) | `ctx.SetBool("random_buff", true)` | `ctx.AddDerivedAction(NewDrawBuffAction(...))` |
| 盗贼(Thief) | `ctx.SetBool("lose_item", true)` | `ctx.AddDerivedAction(NewRemoveItemAction(...))` |
| 雷劫(Thunder) | 直接HP归零 | 仍直接执行（无flag，无DerivedAction） |

**保留的旧字段**（仅在非 DerivedAction 场景使用）：

| 字段 | 类型 | 来源Event | 用途 | 目标 |
|------|------|----------|------|------|
| `instant_death` | bool | Event Handler | 即死标志 | DeathAction |

### Item 效果信号

| 字段 | 类型 | 来源Item | 用途 | 目标 |
|------|------|----------|------|------|
| `target_id` | string | Item Handler | 目标玩家ID | Decision/Action |
| `teleport_target` | string | Item_AnyDoor（任意门） | 传送目标玩家 | TeleportAction |
| `current_dice_type` | string | HSM (OnUseItem) → Item Handler | 当前骰子类型 | DiceUpgradeAction |

**DiceSwap 已移除**：`dice_swap_target` 字段不再使用。DiceUpgrade 取代了 DiceSwap 的功能，通过 DiceUpgradeAction 实现骰子升级（Wood→Copper→Silver→Gold）。

**DerivedAction 模式**：
Item Handler 同样使用 DerivedAction 模式：

| Item Handler | DerivedAction | 说明 |
|-------------|---------------|------|
| 任意门(AnyDoor) | `NewTeleportAction(...)` | 传送玩家至目标位置 |
| 骰子升级(DiceUpgrade) | `NewDiceUpgradeAction(...)` | 升级骰子类型，FromDice 从 `current_dice_type` 字段获取 |

---

## Handler 实现示例

### Buff Handler（隐匿）

```go
// internal/engine/buff_registry.go
func hiddenHandler(phase constants.Phase, ctx *event.Context) {
    // 阻挡所有伤害和事件
    ctx.SetBool("action_blocked", true)
    ctx.SetString("blocked_by", "Buff_Hidden")
    
    // 隐匿自身不会收到伤害，但消耗隐匿状态
    // Handler 返回后，Action 检查 action_blocked 决定是否继续执行
}
```

### Buff Handler（甘霖）

```go
// internal/engine/buff_registry.go
func createEveryNTurnsHandler(everyN int, buffType constants.BuffType, innerHandler EffectHandler) EffectHandler {
    return func(phase constants.Phase, ctx *event.Context) error {
        // 计数器存储在 Buff.Metadata（跨回合持久化）
        buff := ctx.Player.GetBuff(buffType)
        counter := buff.GetIntOrDefault("buff_turn_counter", 0)
        counter++
        buff.SetInt("buff_turn_counter", counter)

        // 每2回合触发HP+1
        if counter >= everyN {
            innerHandler(phase, ctx) // 添加 HealAction derived action
            buff.SetInt("buff_turn_counter", 0) // 重置
        }
        return nil
    }
}
```

### Event Handler

```go
// internal/engine/event_registry.go
func herbHandler(phase constants.Phase, ctx *event.Context) {
    // 采集草药：HP+1
    ctx.SetInt("hp_change", 1)
}
```

---

## 与 ActionContext 的关系

ActionContext.ExecuteAction() 将 Context.Metadata 传递给 EventBus：

```go
// internal/engine/action/context.go
func (ctx *ActionContext) ExecuteAction(action Action) error {
    // PreTrigger phase
    triggerCtx := event.NewContext(action.TargetPlayer())
    triggerCtx.Set("current_action", action)
    triggerCtx.Set("action_context", ctx)

    ctx.EventBus.Publish(prePhase, action.Target(), triggerCtx)

    // Check for blocking
    if triggerCtx.GetBoolOrDefault("action_blocked", false) {
        // Action blocked, process derived actions then return
        return nil
    }
}
```

---

## 扩展说明

新增 Handler 信号时：
1. 选择语义明确的字段名（如 `hp_change` 而非 `val`）
2. 在此契约文档更新
3. 确保信号消费方正确解析

---

## 相关文档

- [internal/event/README.md](../../internal/core/README.md) - EventBus系统
- [doc/internal/event_bus_system.md](../internal/event_bus_system.md) - EventBus设计文档
- [internal/engine/buff_registry.go](../../internal/engine/buff_registry.go) - Buff Handler实现
- [internal/engine/event_registry.go](../../internal/engine/event_registry.go) - Event Handler实现
- [internal/engine/item_registry.go](../../internal/engine/item_registry.go) - Item Handler实现