# Package errors - 统一错误处理

提供统一的错误类型定义和包装工具，用于 Paradiced 项目的错误处理。

## 设计理念

1. **错误分类**：不同类型的错误使用不同的错误类型
2. **错误包装**：使用 `Wrap`/`Wrapf` 为底层错误添加上下文
3. **错误传递**：错误应该向上传递，不要吞掉
4. **边界日志**：在边界层（Nakama 层）记录错误日志

## 错误类型

### InternalError

内部服务器错误，带有组件上下文信息：

```go
err := errors.NewInternalError("HSM", "Update", underlyingErr)
// 输出：[HSM] Update failed: underlying error
```

支持添加额外上下文：

```go
err := errors.NewInternalError("HSM", "Transition", underlyingErr).
    WithContext("from_state", "main_action").
    WithContext("to_state", "turn_moving")
// 输出：[HSM] Transition failed: underlying error (context: map[from_state:main_action to_state:turn_moving])
```

### ValidationError

客户端输入验证失败：

```go
err := errors.NewValidationError("player_id", "", "must be non-empty")
// 输出：validation error: field 'player_id' - must be non-empty (got: )
```

### StateExecutionError

HSM 状态执行失败：

```go
err := errors.NewStateExecutionError("TurnUpkeep", "Enter", underlyingErr)
// 输出：state TurnUpkeep Enter phase error: underlying error
```

### ActionExecutionError

Action 执行失败：

```go
err := errors.NewActionExecutionError("damage", "player-001", "target is dead", underlyingErr)
// 输出：action damage on target player-001 failed (target is dead): underlying error
```

## 使用示例

### 服务层

```go
func (s *GameService) RollDice(playerID string) error {
    // 验证输入
    if playerID == "" {
        return errors.NewValidationError("player_id", "", "must be non-empty")
    }

    // 查找玩家
    player := s.GetPlayer(playerID)
    if player == nil {
        return errors.NewInternalError("GameService", "GetPlayer", nil).
            WithContext("player_id", playerID)
    }

    // 执行操作
    result, err := s.diceMgr.Roll(playerID)
    if err != nil {
        return errors.Wrap(err, "DiceManager", "Roll")
    }

    return nil
}
```

### 处理器层

```go
func (h *NakamaMatchHandler) handleRollDice(sender string) error {
    err := h.gameService.RollDice(sender)
    if err != nil {
        // 检查是否为验证错误
        var ve *errors.ValidationError
        if errors.As(err, &ve) {
            // 返回 400 给客户端
            return h.sendActionRejectedWithCode(sender, pkgnet.OpRollDice,
                constants.ErrInvalidParameter, ve.Error())
        }

        // 检查是否为内部错误
        var ie *errors.InternalError
        if errors.As(err, &ie) {
            // 记录详细日志
            h.logger.Error("RollDice failed",
                "component", ie.Component,
                "operation", ie.Operation,
                "context", ie.Context)
        }

        // 返回 500 给客户端
        return h.sendActionRejectedWithCode(sender, pkgnet.OpRollDice,
            constants.ErrInternal, "Internal server error")
    }
    return nil
}
```

### HSM 状态层

```go
func (s *TurnUpkeepState) Enter(ctx *hsm.StateContext) {
    player := ctx.Player
    if player == nil {
        // 设置状态错误
        ctx.Error = hsm.NewStateError(StateTurnUpkeep, "player is nil")
        return
    }

    // 执行可能失败的操作
    if err := s.executePhase(ctx); err != nil {
        // 包装错误并设置到上下文
        ctx.Error = hsm.WrapError(err, StateTurnUpkeep, 2, "Enter", "phase execution failed")
        return
    }
}
```

## 错误处理最佳实践

1. **使用具体错误类型**：不要总是返回 `fmt.Errorf`，使用适当的错误类型
2. **添加上下文**：使用 `Wrap`/`Wrapf` 为错误添加组件和操作信息
3. **不要吞掉错误**：错误应该向上传递，除非你知道如何处理它
4. **在边界记录日志**：在 Nakama 层记录错误详情，返回简洁消息给客户端
5. **错误检查**：使用 `errors.As` 进行类型断言，进行适当的错误处理

## 与 HSM 错误系统集成

`internal/engine/hsm` 包有自己的 `HSMError` 和 `StateError` 类型，用于 HSM 特定的错误场景：

```go
// HSM 层返回 HSMError
err := hsm.WrapError(underlyingErr, StateTurnUpkeep, 2, "Enter", "phase execution failed")

// Nakama 层检查并转换为 InternalError
var hsmErr *hsm.HSMError
if errors.As(err, &hsmErr) {
    return errors.Wrap(hsmErr, "HSM", hsmErr.StateID.String())
}
```

## 与 ErrorCode 系统集成

ErrorCode 系统 (`pkg/constants.ErrorCode`) 用于客户端 - 服务器通信的标准化错误码：

```go
// 内部错误处理
err := errors.NewInternalError("HSM", "Update", underlyingErr)

// 转换为 ErrorCode 返回给客户端
var errorCode constants.ErrorCode
switch {
case errors.As(err, &validationErr):
    errorCode = constants.ErrInvalidParameter
case errors.As(err, &notFoundErr):
    errorCode = constants.ErrPlayerNotFound
default:
    errorCode = constants.ErrInternal
}

// 发送 ActionRejected
return h.sendActionRejectedWithCode(sender, opCode, errorCode, err.Error())
```

## 相关文档

- [pkg/constants/README.md](../constants/README.md) - ErrorCode 错误码系统
- [internal/engine/hsm/README.md](../../internal/engine/hsm/README.md) - HSM 状态机
- [internal/nakama/README.md](../../internal/nakama/README.md) - Nakama 协议层
