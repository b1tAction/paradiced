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
    
    Game        protocol.Game      // 游戏实例
    EventBus    *event.EventBus    // 事件总线（拦截支持）
    MapEngine   protocol.MapEngine // 地图引擎
    ActionQueue *Queue             // 派生Action队列
}
```

---

## 字段契约表

### 拦截通信字段

| 字段 | 类型 | 来源 | 用途 | 目标 |
|------|------|------|------|------|
| `current_action` | ExecutableAction | ExecuteAction | 当前执行的Action | EventBus Context |
| `action_context` | *ActionContext | ExecuteAction | ActionContext引用 | EventBus Context |

### MoveAction 路径数据字段

| 字段 | 类型 | 来源 | 用途 | 目标 |
|------|------|------|------|------|
| `target_pos` | int | HSM (TurnMovingState) | 移动目标位置（CalculatePath结果） | MoveAction.Execute() |
| `path` | []int | HSM (TurnMovingState) | 移动路径（CalculatePath结果） | MoveAction.LogEntry() |

### 使用场景

这些字段在 `ExecuteAction()` 中设置，传递给 EventBus PreTrigger/PostTrigger：

```go
func (ctx *ActionContext) ExecuteAction(action ExecutableAction) error {
    // PreTrigger phase
    triggerCtx := event.NewContext(ctx.Game.GetCurrentPlayer())
    triggerCtx.Set("current_action", action)   // 当前Action
    triggerCtx.Set("action_context", ctx)       // ActionContext引用
    
    ctx.EventBus.Publish(prePhase, action.Target(), triggerCtx)
    
    // Handler 可通过 action_context 访问游戏数据
    // Handler 可通过 current_action 获取Action详情
}
```

---

## Handler 访问示例

Handler 可以通过 `action_context` 字段访问游戏数据：

```go
func someHandler(ctx *event.Context, player protocol.Player) {
    // 获取当前Action
    currentAction, ok := ctx.Get("current_action")
    if ok {
        action := currentAction.(ExecutableAction)
        // 获取Action类型、目标等
    }
    
    // 获取ActionContext
    actionCtx, ok := ctx.Get("action_context")
    if ok {
        actx := actionCtx.(*action.ActionContext)
        // 访问游戏数据
        gameLog := actx.GetGameLog()
        // 添加派生Action
        actx.PushDerivedAction(NewHealAction(...))
    }
}
```

---

## 派生Action队列

ActionContext 不直接使用Metadata存储派生Action，而是使用内置的 `ActionQueue`：

```go
// 添加派生Action
func (ctx *ActionContext) PushDerivedAction(action ExecutableAction)

// 处理队列
func (ctx *ActionContext) ProcessQueue()
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
- 执行过程中的临时标记

新增字段时：
1. 确保字段名语义明确
2. 在此契约文档更新
3. 确保不与EventBus Context.Metadata冲突

---

## 相关文档

- [internal/engine/action/README.md](../../internal/engine/action/README.md) - Action系统文档
- [internal/event/context.go](../../internal/event/context.go) - EventBus Context
- [internal/engine/action/context.go](../../internal/engine/action/context.go) - ActionContext实现