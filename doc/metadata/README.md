# Metadata 契约文档

本目录包含所有使用 `util.Metadata` 的类型的字段契约定义。

## 概述

`pkg/util/metadata` 提供类型安全的键值存储容器，被多个核心类型嵌入使用。为确保前后端数据一致性，所有 Metadata 字段的使用必须遵循本目录定义的契约。

**重要**：
- 新增 Metadata 字段时，必须在对应文档中更新契约
- 客户端开发应参考 `logentry.md` 了解客户端可见字段
- 内部通信字段（Handler/Context）仅供后端开发参考

## 文件说明

| 文件 | 类型 | 可见性 | 说明 |
|------|------|--------|------|
| `logentry.md` | `gamelog.LogEntry.Metadata` | 客户端可见 | Action效果详情，通过TurnSync发送给客户端渲染 |
| `player.md` | `core.Player.Metadata` | 客户端可见 | 玩家动态属性（阵营特定） |
| `round_data.md` | `Game.RoundData` | 内部 | 回合级持久数据（每回合清空） |
| `event_context.md` | `event.Context.Metadata` | 内部 | EventBus Handler之间传递意图信号 |
| `hsm_context.md` | `hsm.StateContext.Metadata` | 内部 | HSM状态之间传递数据（tick级） |
| `action_context.md` | `action.ActionContext.Metadata` | 内部 | Action执行上下文数据 |
| `buff.md` | `core.Buff.Metadata` | 内部 | Buff实例状态存储（如everyNTurns计数器） |

## 数据分层设计

| 层级 | 存储位置 | 生命周期 | 适用数据 |
|-----|---------|---------|---------|
| **瞬时层** | StateContext.Metadata | 单 tick | Enter→Update→Exit 通信 |
| **回合层** | Game.RoundData | 一回合 | 跨状态但回合结束时清理 |
| **核心层** | Game/HSM 字段 | 整个游戏 | 类型安全的核心数据 |

## 维护指南

### 新增 Metadata 字段时

1. **确定字段归属**：
   - 需要发送给客户端 → 更新 `logentry.md`
   - 玩家动态属性 → 更新 `player.md`
   - 回合级持久数据 → 更新 `round_data.md`
   - Handler通信 → 更新 `event_context.md`
   - HSM状态通信 → 更新 `hsm_context.md`
   - Action执行 → 更新 `action_context.md`

2. **更新契约表格**：
   - 添加字段名、类型、用途说明
   - 注明来源和目标（内部通信）

3. **客户端同步**（仅LogEntry.Metadata）：
   - 更新 TypeScript 类型定义
   - 更新客户端渲染逻辑

### 字段命名约定

- 使用 `snake_case` 格式
- 键名应具有语义（如 `buff_type` 而非 `type`）
- 避免缩写（如 `fire_counter` 而非 `fc`）
- 动态键使用拼接格式（如 `result_{playerID}`）

## 相关文档

- [doc/internal/metadata.md](../internal/metadata.md) - Metadata工具类使用说明
- [pkg/util/metadata.go](../../pkg/util/metadata.go) - 核心实现
- [doc/internal/net_protocol.md](../internal/net_protocol.md) - 网络协议层设计