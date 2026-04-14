# pkg/handler - Effect Handler Types

效果处理器统一类型定义。

## 概述

`pkg/handler` 提供统一的 `EffectHandler` 类型，用于 Buff/Item/Event/Faction 等所有效果源的处理函数签名。

## 核心类型

### EffectHandler

```go
type EffectHandler func(phase event.Phase, ctx *event.Context)
```

所有效果处理器共享此签名，通过 `ctx.AddDerivedAction()` 生成新 Action。

## 使用方式

### Buff Handler

```go
// 注册 Buff 时传入 handler
GlobalBuffRegistry.RegisterBuff(&BuffDefinition{
    Type:   BuffTypeUndying,
    Phases: []event.Phase{event.PhasePreRespawn},
    // ...
}, handleUndying)

func handleUndying(phase event.Phase, ctx *event.Context) {
    if phase != event.PhasePreRespawn {
        return
    }
    
    ctx.SetBool("action_blocked", true)
    player := ctx.Player.(*core.Player)
    ctx.AddDerivedAction(NewHealAction(player, player.MaxHP, "Buff_Undying"))
    ctx.AddDerivedAction(NewRemoveBuffAction(player, BuffTypeUndying, "Buff_Undying"))
}
```

### Item Handler

```go
GlobalItemRegistry.RegisterItem(&ItemDefinition{
    Type:   ItemTypeAnyDoor,
    Phases: []event.Phase{event.PhaseOnLand},
    // ...
}, handleAnyDoor)

func handleAnyDoor(phase event.Phase, ctx *event.Context) {
    if phase != event.PhaseOnLand {
        return
    }
    // 处理任意门逻辑...
}
```

### Event Handler

```go
GlobalEventRegistry.RegisterEvent(&EventDefinition{
    Type: EventTypeThunder,
    // ...
}, handleThunder)

func handleThunder(phase event.Phase, ctx *event.Context) {
    // 雷劫处理逻辑...
}
```

## Handler 行为模式

### 1. 拦截模式

阻断当前 Action，生成替代 Action：

```go
func InterceptHandler(phase event.Phase, ctx *event.Context) {
    ctx.SetBool("action_blocked", true)  // 阻断原 Action
    
    // 生成替代 Action
    ctx.AddDerivedAction(NewHealAction(...))
    ctx.AddDerivedAction(NewRemoveBuffAction(...))
}
```

### 2. 篡改模式

修改当前 Action 的参数：

```go
func ModifyHandler(phase event.Phase, ctx *event.Context) {
    action := ctx.Get("current_action")
    if moveAction, ok := action.(*MoveAction); ok {
        moveAction.Steps = -moveAction.Steps  // 反向移动
    }
}
```

### 3. 生成模式

生成新的衍生 Action：

```go
func GenerateHandler(phase event.Phase, ctx *event.Context) {
    player := ctx.Player.(*core.Player)
    
    if player.HasBuff(BuffTypeDivine) {
        ctx.AddDerivedAction(NewModifyLPAction(player, 1, "Buff_Divine"))
    }
}
```

## 相关文档

- [pkg/event/README.md](../event/README.md) - EventBus 和 Phase 系统
- [pkg/action/README.md](../action/README.md) - Action 接口层
- [internal/core/buff/README.md](../../internal/core/buff/README.md) - Buff 注册表