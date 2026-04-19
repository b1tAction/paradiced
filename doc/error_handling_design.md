# 错误处理设计文档

## 1. 概述

本文档描述了 Paradiced 项目的统一错误处理系统设计，包括错误类型定义、错误包装、错误传递和错误日志记录的最佳实践。

## 2. 错误处理架构

```
┌─────────────────────────────────────────────────────────────┐
│                      Nakama Protocol Layer                   │
│  - 边界日志记录                                              │
│  - ErrorCode 转换                                            │
│  - ActionRejected 响应                                        │
├─────────────────────────────────────────────────────────────┤
│                      HSM Layer                               │
│  - StateContext.Error 状态错误捕获                            │
│  - HSMError 包装                                             │
│  - 错误向上传递                                              │
├─────────────────────────────────────────────────────────────┤
│                      Service/Core Layer                      │
│  - pkg/errors 统一错误类型                                    │
│  - 错误包装 (Wrap/Wrapf)                                     │
│  - 具体错误类型 (ValidationError, InternalError, etc.)       │
└─────────────────────────────────────────────────────────────┘
```

## 3. 错误类型系统

### 3.1 pkg/errors - 通用错误类型

位于 `pkg/errors/errors.go`，提供项目通用的错误类型：

#### InternalError（内部错误）
- **用途**: 包装内部服务器错误，添加组件上下文
- **场景**: 服务层、Repository 层、底层系统调用
- **示例**:
```go
err := errors.NewInternalError("HSM", "Update", underlyingErr).
    WithContext("state", "TurnUpkeep")
```

#### ValidationError（验证错误）
- **用途**: 客户端输入验证失败
- **场景**: 参数校验、状态检查
- **示例**:
```go
if playerID == "" {
    return errors.NewValidationError("player_id", "", "must be non-empty")
}
```

#### StateExecutionError（状态执行错误）
- **用途**: HSM 状态执行失败
- **场景**: State.Enter/Update/Exit phase 错误
- **示例**:
```go
err := errors.NewStateExecutionError("TurnUpkeep", "Enter", underlyingErr)
```

#### ActionExecutionError（Action 执行错误）
- **用途**: Action 执行失败
- **场景**: DamageAction、HealAction 等执行错误
- **示例**:
```go
err := errors.NewActionExecutionError("damage", playerID, "target is dead", underlyingErr)
```

### 3.2 pkg/errors - HSM 错误类型

`pkg/errors` 包提供 HSM 特定的错误类型：

#### HSMError
- **用途**: HSM 状态机执行错误，带有状态上下文
- **字段**: StateID, Layer, Phase, Err, Message
- **位置**: `pkg/errors/hsm_error.go`
- **示例**:
```go
err := errors.NewHSMError("TurnUpkeep", 2, "Enter", underlyingErr, "phase execution failed")
```

#### WrapHSMError
- **用途**: 为错误添加 HSM 上下文
- **示例**:
```go
if err != nil {
    return errors.WrapHSMError(err, "TurnMoving", 2, "Enter", "move action failed")
}
```

#### StateError (internal/engine/hsm)
- **用途**: HSM 状态特定错误
- **位置**: `internal/engine/hsm/turn_states.go`
- **说明**: 由于依赖 `StateID` 类型，保留在 hsm 包中
- **示例**:
```go
ctx.Error = hsm.NewStateError(StateTurnUpkeep, "player is nil")
```

### 3.3 StateContext.Error - 状态错误捕获

`StateContext` 有 `Error` 字段用于捕获状态执行过程中的错误：

```go
func (s *TurnUpkeepState) Enter(ctx *StateContext) {
    player := ctx.Player
    if player == nil {
        ctx.Error = NewStateError(StateTurnUpkeep, "player is nil")
        return
    }
    // ...
}
```

## 4. 错误处理流程

### 4.1 错误产生（Service/Core 层）

```go
// internal/core/player.go
func (p *Player) TakeDamage(amount int) error {
    if amount < 0 {
        return errors.NewValidationError("damage_amount", amount, "must be non-negative")
    }
    if p.HP <= 0 {
        return errors.NewActionExecutionError("damage", p.ID.UUID(), "player already dead", nil)
    }
    p.HP -= amount
    return nil
}
```

### 4.2 错误包装（HSM 层）

```go
// internal/engine/hsm/turn_states.go
func (s *TurnUpkeepState) Enter(ctx *StateContext) {
    // ...
    if err := s.executePhase(ctx); err != nil {
        // 包装错误并设置到上下文
        ctx.Error = errors.WrapHSMError(err, "TurnUpkeep", 2, "Enter", "phase execution failed")
        return
    }

    // 或使用 StateError（简单状态错误）
    if player == nil {
        ctx.Error = hsm.NewStateError(StateTurnUpkeep, "player is nil")
        return
    }
}
```

### 4.3 错误处理与日志记录（Nakama 层）

```go
// internal/nakama/message.go
func (h *NakamaMatchHandler) handleRollDice(sender string) error {
    logger := NewLogger(h)

    // 验证玩家
    player := h.GetPlayer(sender)
    if player == nil {
        logger.logReject("OpRollDice", sender, constants.ErrPlayerNotFound, "player_not_found", "Player not found")
        return h.sendActionRejectedWithCode(sender, pkgnet.OpRollDice, constants.ErrPlayerNotFound, "Player not found")
    }

    // 调用 HSM
    err := h.hsm.OnRollDice(steps, ctx)
    if err != nil {
        // 检查错误类型
        var hsmErr *errors.HSMError
        if errors.As(err, &hsmErr) {
            // 记录详细的 HSM 错误日志
            logger.logError("OpRollDice", sender, err,
                "state", hsmErr.StateID,
                "layer", hsmErr.Layer,
                "phase", hsmErr.Phase)
        }

        var validationErr *errors.ValidationError
        if errors.As(err, &validationErr) {
            // 返回 400 级别的错误
            return h.sendActionRejectedWithCode(sender, pkgnet.OpRollDice,
                constants.ErrInvalidParameter, validationErr.Error())
        }

        // 检查 StateError
        var stateErr *hsm.StateError
        if errors.As(err, &stateErr) {
            logger.logError("OpRollDice", sender, stateErr)
            return h.sendActionRejectedWithCode(sender, pkgnet.OpRollDice,
                constants.ErrInternal, stateErr.Error())
        }

        // 默认返回 500 级别错误
        return h.sendActionRejectedWithCode(sender, pkgnet.OpRollDice,
            constants.ErrInternal, "Internal server error")
    }

    return nil
}
```

## 5. ErrorCode 映射

`pkg/constants/error_code.go` 定义了发送给客户端的错误码：

| ErrorCode | HTTP 等价 | 用途 |
|-----------|-----------|------|
| `ErrInvalidParameter` (1001) | 400 | 参数验证失败 |
| `ErrInvalidState` (1002) | 400 | 状态不允许操作 |
| `ErrConditionNotMet` (1003) | 400 | 条件未满足 |
| `ErrPlayerNotFound` (2001) | 404 | 玩家不存在 |
| `ErrItemNotFound` (2002) | 404 | 道具不存在 |
| `ErrBuffNotFound` (2003) | 404 | Buff 不存在 |
| `ErrInternal` (3001) | 500 | 内部服务器错误 |
| `ErrNotCurrentTurn` (3002) | 403 | 非当前回合 |

### 错误类型到 ErrorCode 的映射

```go
// internal/nakama/error_handler.go (建议新增)
func ErrorCodeForError(err error) constants.ErrorCode {
    // 验证错误 -> 1xxx
    var validationErr *errors.ValidationError
    if errors.As(err, &validationErr) {
        return constants.ErrInvalidParameter
    }

    // HSM 错误 -> 根据具体情况
    var hsmErr *hsm.HSMError
    if errors.As(err, &hsmErr) {
        if hsmErr.Phase == "Enter" && strings.Contains(hsmErr.Err.Error(), "player is nil") {
            return constants.ErrPlayerNotFound
        }
        return constants.ErrInternal
    }

    // 未找到错误
    if strings.Contains(err.Error(), "not found") {
        return constants.ErrPlayerNotFound
    }

    // 状态错误
    if strings.Contains(err.Error(), "invalid state") {
        return constants.ErrInvalidState
    }

    // 默认内部错误
    return constants.ErrInternal
}
```

## 6. 最佳实践

### 6.1 错误创建

✅ **推荐**:
```go
// 使用具体的错误类型
return errors.NewValidationError("player_id", "", "must be non-empty")

// 包装底层错误
return errors.Wrap(underlyingErr, "GameService", "RollDice")

// 添加上下文
return errors.NewInternalError("HSM", "Update", err).
    WithContext("state", "TurnUpkeep").
    WithContext("round", game.Round)
```

❌ **不推荐**:
```go
// 直接使用 fmt.Errorf
return fmt.Errorf("failed to roll dice")

// 吞掉错误
if err := doSomething(); err != nil {
    log.Println("error:", err)
    // 没有返回错误
}

// 返回 nil 而不处理
if validationFailed {
    return nil // 应该返回错误
}
```

### 6.2 错误检查

✅ **推荐**:
```go
// 使用 errors.As 进行类型检查
var validationErr *errors.ValidationError
if errors.As(err, &validationErr) {
    // 处理验证错误
    return h.sendActionRejectedWithCode(sender, opCode,
        constants.ErrInvalidParameter, validationErr.Error())
}

// 检查 StateContext.Error
if ctx.Error != nil {
    logger.Error("State execution failed", "error", ctx.Error)
    return h.sendActionRejectedWithCode(sender, opCode,
        constants.ErrInternal, ctx.Error.Error())
}
```

### 6.3 错误日志

✅ **推荐**:
```go
// 在边界层（Nakama）记录详细日志
logger.Error("RollDice failed",
    "component", ie.Component,
    "operation", ie.Operation,
    "context", ie.Context,
    "player_id", sender)
```

❌ **不推荐**:
```go
// 在服务层记录日志（应该在边界层记录）
func (s *Service) DoSomething() error {
    if err != nil {
        log.Println("error:", err) // 不应该在这里记录
        return err
    }
}
```

## 7. 当前存在的问题与改进计划

### 7.1 已识别的问题

| 问题 | 位置 | 严重性 | 改进建议 |
|------|------|--------|----------|
| `handleUserChoice` 返回 nil 而不处理错误 | `nakama/message.go:331` | 中 | 检查 `ctx.Error` 并返回适当错误 |
| `handleMiniGameResult` 对未知玩家返回 nil | `nakama/message.go:354` | 低 | 考虑返回 `ErrPlayerNotFound` |
| `handleMiniGameResult` 对错误状态返回 nil | `nakama/message.go:366` | 中 | 返回 `ErrInvalidState` |
| `handleUseSkill` 成功后返回 nil 但不广播状态 | `nakama/message.go:294` | 中 | 添加状态广播 |
| HSM `OnUserChoice` 不检查 `ctx.Error` | `hsm/hsm.go:527` | 中 | 添加错误检查 |
| 状态 `Enter` 方法设置 `ctx.Error` 但调用方不检查 | 多处 | 高 | 统一检查模式 |

### 7.2 改进计划

#### Phase 1: 统一错误检查模式（已完成）
- ✅ 创建 `pkg/errors` 包
- ✅ 创建 `internal/engine/hsm/errors.go`
- ✅ 定义统一错误类型

#### Phase 2: 修复 Nakama 层错误处理
- [ ] `handleUserChoice`: 检查 HSM 执行结果和 `ctx.Error`
- [ ] `handleMiniGameResult`: 返回适当的 ErrorCode
- [ ] `handleUseSkill`: 添加状态同步广播
- [ ] 统一验证失败时返回 ErrorCode

#### Phase 3: 修复 HSM 层错误传播
- [ ] `OnUserChoice`: 检查并返回 `ctx.Error`
- [ ] `OnMiniGameResult`: 检查状态执行错误
- [ ] `OnRollDice`/`OnUseItem`: 统一错误返回模式
- [ ] 所有状态 `Enter` 方法后检查 `ctx.Error`

#### Phase 4: 添加错误处理工具
- [ ] `internal/nakama/error_handler.go`: ErrorCode 映射工具
- [ ] `internal/nakama/error_handler_test.go`: 映射测试
- [ ] 更新 `sendActionRejectedWithCode` 使用统一映射

#### Phase 5: 测试与文档
- [ ] 添加错误处理集成测试
- [ ] 更新 `pkg/errors/README.md`
- [ ] 更新 `internal/engine/hsm/README.md`
- [ ] 更新 `internal/nakama/README.md`

## 8. 错误处理流程图

```
客户端请求
    │
    ▼
NakamaMatchHandler
    │
    ├──→ 参数验证 ────→ ValidationError ──→ ErrInvalidParameter (1001)
    │
    ├──→ 状态检查 ────→ InvalidState ─────→ ErrInvalidState (1002)
    │
    ▼
HSM.OnXxx()
    │
    ├──→ State.Enter() ─→ ctx.Error ──→ HSMError
    │
    ▼
返回 error
    │
    ▼
Nakama 错误处理
    │
    ├──→ errors.As(err, &ValidationError) ──→ 1xxx
    ├──→ errors.As(err, &HSMError) ─────────→ 根据情况
    └──→ default ───────────────────────────→ 3001
    │
    ▼
ActionRejected (带 ErrorCode)
    │
    ▼
客户端
```

## 9. 相关文件

- `pkg/errors/errors.go` - 通用错误类型定义
- `pkg/errors/README.md` - 错误包使用文档
- `internal/engine/hsm/errors.go` - HSM 错误类型定义
- `pkg/constants/error_code.go` - ErrorCode 定义
- `pkg/constants/error_code_test.go` - ErrorCode 测试
- `internal/nakama/message.go` - Nakama 消息处理
- `internal/nakama/handler.go` - Nakama Handler

## 10. 总结

本项目错误处理系统分为三层：

1. **pkg/errors**: 通用错误类型（InternalError, ValidationError, etc.）
2. **internal/engine/hsm/errors**: HSM 专用错误类型（HSMError）
3. **pkg/constants**: ErrorCode 错误码（发送给客户端）

核心原则：
- 错误应该向上传递，不要吞掉
- 在边界层（Nakama）记录日志
- 使用 `errors.As` 进行错误类型检查
- 返回适当的 ErrorCode 给客户端
- 使用 `StateContext.Error` 捕获状态执行错误
