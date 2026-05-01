# ActionContext.Metadata 契约

`action.ActionContext.Metadata` 用于Action执行上下文数据，**不发送给客户端**。

**位置**：`internal/engine/action/context.go`

**可见性**：内部（仅后端使用）

---

## 概述

ActionContext 是Action执行的核心上下文，嵌入Metadata用于扩展存储：

```go
// internal/engine/action/context.go
type ActionContext struct {
    *util.Metadata // 嵌入扩展存储

    Game          protocol.Game      // 游戏实例
    EventBus      *event.EventBus    // 事件总线（拦截支持）
    MapEngine     *gamemap.MapEngine // 地图引擎
    DrawEngine    *rng.DrawEngine    // 随机抽取引擎
    EventPool     []*rng.EvaluatedItem // 事件池
    ItemPool      []*rng.EvaluatedItem // 道具池
    BuffPool      []*rng.EvaluatedItem // Buff池（仅IsDraw()的BuffType）
    ActionQueue   *Queue             // 派生Action队列
    ProbGood      float64            // Good池概率权重
    ProbNeutral   float64            // Neutral池概率权重
    ProbBad       float64            // Bad池概率权重

    // Buff lifecycle callbacks - injected by HSM layer
    OnAddBuff    func(player *core.Player, buff *core.Buff)
    OnRemoveBuff func(player *core.Player, buffType constants.BuffType) *core.Buff
    GetBuffDuration func(buffType constants.BuffType) int

    // Item lifecycle callbacks - injected by HSM layer
    OnAddItem    func(player *core.Player, item *core.Item)
    OnRemoveItem func(player *core.Player, itemType constants.ItemType) *core.Item
}
```

---

## 字段契约表

### 拦截通信字段

| 字段 | 类型 | 来源 | 用途 | 目标 |
|------|------|------|------|------|
| `current_action` | Action | ExecuteAction | 当前执行的Action | EventBus Context |
| `action_context` | *ActionContext | ExecuteAction | ActionContext引用 | EventBus Context |

### Buff 时效通信字段

| 字段 | 类型 | 来源 | 用途 | 目标 |
|------|------|------|------|------|
| `buff_duration_extended` | bool | AddBuffAction.Execute | Buff时效已延长（非新应用） | ExecuteAction PostTrigger |

### MoveAction 路径数据字段

| 字段 | 类型 | 来源 | 用途 | 目标 |
|------|------|------|------|------|
| `target_pos` | int | HSM (TurnMovingState) | 移动目标位置（CalculatePath结果） | MoveAction.Execute() |
| `path` | []int | HSM (TurnMovingState) | 移动路径（CalculatePath结果） | MoveAction.LogEntry() |

### DiceUpgradeAction 骰子升级字段

| 字段 | 类型 | 来源 | 用途 | 目标 |
|------|------|------|------|------|
| `dice_upgrade_from` | string | DiceUpgradeAction.Execute | 原始骰子类型 | DiceUpgradeAction.LogEntry() |
| `dice_upgrade_to` | string | DiceUpgradeAction.Execute | 升级后骰子类型 | DiceUpgradeAction.LogEntry()、HSM层读取升级目标 |

### 使用场景

这些字段在 `ExecuteAction()` 中设置，传递给 EventBus。除了 PreTrigger/PostTrigger 外，
Step 0 (PhasePreAction) 也使用这些字段进行死亡拦截：

```go
func (ctx *ActionContext) ExecuteAction(action Action) error {
    // Step 0: PhasePreAction - 死亡拦截
    // DeathMark handler 根据 current_action 判断是否阻拦：
    // - RespawnAction: 不阻拦（必须执行）
    // - RemoveBuffAction(DeathMark): 不阻拦（移除自身不应阻拦）
    // - 其他Action: 阻拦
    if targetPlayer != nil && targetPlayer.IsDead {
        preCtx := event.NewContext(targetPlayer)
        preCtx.Set("current_action", action)
        preCtx.Set("action_context", ctx)
        ctx.EventBus.Publish(constants.PhasePreAction, targetPlayer.ID.UUID(), preCtx)
        // 检查 PhasePreAction handler 错误
        if preCtx.HasError() {
            return preCtx.FirstError()
        }
        if preCtx.GetBoolOrDefault("action_blocked", false) {
            // 阻拦时仍收集衍生Action
            return nil
        }
    }

    // PreTrigger phase - 使用 action.TargetPlayer() 创建触发上下文
    triggerCtx := event.NewContext(action.TargetPlayer())
    triggerCtx.Set("current_action", action)   // 当前Action
    triggerCtx.Set("action_context", ctx)       // ActionContext引用

    ctx.EventBus.Publish(prePhase, action.Target(), triggerCtx)

    // PostTrigger phase - 同样使用 action.TargetPlayer()
    triggerCtx := event.NewContext(action.TargetPlayer())
    triggerCtx.Set("current_action", action)
    triggerCtx.Set("action_context", ctx)
    // AddBuffAction 设置 applied_buff_type
    // RemoveBuffAction 设置 removed_buff_type

    ctx.EventBus.Publish(postPhase, action.Target(), triggerCtx)
}
```

---

## Handler 访问示例

Handler 可以通过 `action_context` 字段访问游戏数据。`getActionCtxFromEventCtx` 辅助函数现在返回 `(ActionContext, error)`，若缺失则返回错误而非静默 nil：

```go
func someHandler(phase constants.Phase, ctx *event.Context) error {
    if ctx == nil {
        return fmt.Errorf("handler: event context is nil")
    }
    if ctx.Player == nil {
        return fmt.Errorf("handler: player is nil in event context")
    }

    // 获取ActionContext（必须存在，否则返回错误）
    actionCtx, err := getActionCtxFromEventCtx(ctx)
    if err != nil {
        return err
    }
    _ = actionCtx // ActionContext used for derived action processing

    // 获取当前Action
    currentAction, ok := ctx.Get("current_action")
    if ok {
        action := currentAction.(Action)
        // 获取Action类型、目标等
    }

    // 添加派生Action
    ctx.AddDerivedAction(engineaction.NewModifyLPAction(ctx.Player, 1, string(constants.SourceBuffDivine)))
}
```

---

## 派生Action队列

ActionContext 不直接使用Metadata存储派生Action，而是使用内置的 `ActionQueue`：

```go
// 添加派生Action
func (ctx *ActionContext) PushDerivedAction(action Action)

// 处理队列
func (ctx *ActionContext) ProcessQueue() error
```

派生Action来源：
1. EventBus Handler 通过 `ctx.AddDerivedAction()` 添加
2. `ExecuteAction()` 收集并推送到 ActionQueue
3. `ProcessQueue()` 递归执行派生Action

---

## Clear 方法

ActionContext.Clear() 重置上下文：

```go
func (ctx *ActionContext) Clear() {
    ctx.ActionQueue.Clear()
    ctx.Metadata.Clear()
}
```

每回合开始时调用，清除上一回合的临时数据。

---

## 扩展说明

ActionContext.Metadata 主要用于存储临时上下文数据：
- 当前Action信息（传递给EventBus）
- 执行过程中的临时标记（如 `buff_duration_extended`）

新增字段时：
1. 确保字段名语义明确
2. 在此契约文档更新
3. 确保不与EventBus Context.Metadata冲突

---

## 相关文档

- [internal/engine/action/README.md](../../internal/engine/action/README.md) - Action系统文档
- [doc/metadata/event_context.md](event_context.md) - EventBus Context.Metadata 契约
- [internal/engine/action/context.go](../../internal/engine/action/context.go) - ActionContext实现