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
| `action_blocked` | bool | Buff_Hidden | 动作被阻挡标志 | ActionContext |
| `reverse_movement` | bool | Buff_Lost（迷途） | 反向移动标志（仅日志/调试） | LogEntry.Metadata |
| `current_state` | StepsModifier | HSM (TurnMoving) | 当前移动状态实例（迷途修改Steps） | 迷途handler |
| `draw_bad_event` | bool | Buff_Poison（毒瘴） | 抽取坏事件标志 | DrawEventAction |
| `block_poison_effect` | bool | Buff_Exorcism（辟邪） | 阻挡毒效果标志 | Event Handler |
| `buff_turn_counter` | int | Buff_Rain/Corrupt | 计数器（每N回合触发） | Buff Handler |
| `applied_buff_type` | string | ActionContext (PostTrigger) | 被添加的Buff类型标识 | Divine/Curse Handler |
| `removed_buff_type` | string | ActionContext (PreTrigger) / HSM (TurnEnd expiry) | 被移除的Buff类型标识 | Divine/Curse Handler |

### Event 效果信号

| 字段 | 类型 | 来源Event | 用途 | 目标 |
|------|------|----------|------|------|
| `draw_item` | bool | Event Handler | 抽取道具标志 | GiveItemAction |
| `swap_position` | bool | Event Handler | 交换位置标志 | SwapPositionAction |
| `random_buff` | bool | Event Handler | 随机Buff标志 | AddBuffAction |
| `lose_item` | bool | Event Handler | 失去道具标志 | RemoveItemAction |
| `instant_death` | bool | Event Handler | 即死标志 | DeathAction |

### Item 效果信号

| 字段 | 类型 | 来源Item | 用途 | 目标 |
|------|------|----------|------|------|
| `target_id` | string | Item Handler | 目标玩家ID | Decision/Action |
| `teleport_target` | string | Item_AnyDoor（任意门） | 传送目标玩家 | TeleportAction |
| `dice_swap_target` | string | Item Handler | 骰子交换目标 | Decision |
| `dice_upgrade_from` | string | Item Handler | 骰子升级来源类型 | LogEntry.Metadata |
| `current_dice_type` | string | Item Handler | 当前骰子类型 | Item Handler |

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
func rainHandler(phase constants.Phase, ctx *event.Context) {
    // 获取计数器
    counter, _ := ctx.GetInt("buff_turn_counter")
    counter++
    ctx.SetInt("buff_turn_counter", counter)
    
    // 每2回合触发HP+1
    if counter >= 2 {
        ctx.SetInt("hp_change", 1)
        // engine层通过 ctx.AddDerivedAction 添加派生Action
        ctx.SetInt("buff_turn_counter", 0) // 重置
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