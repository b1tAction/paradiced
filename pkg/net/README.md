# Package net - 网络消息协议层

本包定义客户端-服务器交互的消息协议，用于 Fated 游戏的权威服务器通信。

## 设计目标

1. **状态同步**：向客户端广播当前游戏状态
2. **Action同步**：将效果执行结果发送给客户端渲染
3. **决策请求**：等待玩家输入（投骰子、使用道具、选择选项）
4. **断线重连**：支持玩家重新连接后恢复游戏状态

## 架构定位

```
┌──────────────────────────────────────────────────────────────────┐
│                    Nakama Match Handler Layer                      │
│  (实现 MatchHandler 接口，处理客户端连接、消息路由、状态广播)        │
├──────────────────────────────────────────────────────────────────┤
│                    pkg/net 协议层 (本包)                           │
│  - OpCode: 消息操作码                                              │
│  - Message: 基础消息结构                                           │
│  - StateSync/ActionSync: 同步数据结构                              │
│  - Decision: 决策请求结构                                          │
│  - MatchHandler: 抽象接口                                          │
├──────────────────────────────────────────────────────────────────┤
│                    internal/net 构建层                             │
│  - Builder: 将内部数据转换为协议数据                                │
│  - DiceCalculator: 骰子计算（权威服务器）                           │
├──────────────────────────────────────────────────────────────────┤
│                    游戏核心层 (现有)                               │
│  - HSM: 状态机控制                                                 │
│  - Action: 效果执行                                                │
│  - GameLog: 日志记录                                               │
└──────────────────────────────────────────────────────────────────┘
```

## 文件说明

| 文件 | 说明 |
|------|------|
| `opcode.go` | 消息操作码定义（Server→Client: 1-99, Client→Server: 100+） |
| `message.go` | 基础消息结构 `Message` |
| `sync.go` | 状态同步数据结构：`StateSync`, `Player`, `ActionSync`, `Available` |
| `decision.go` | 决策请求/回复结构：`Decision`, `RollDice`, `UseItem`, `UserChoice` |
| `handler.go` | 抽象接口 `MatchHandler` 和配置 `MatchConfig` |

## 消息操作码

### Server → Client (1-99)

| OpCode | 名称 | 数据类型 | 说明 |
|--------|------|----------|------|
| 1 | `OpStateSync` | `StateSync` | 状态同步（进入新状态） |
| 2 | `OpActionSync` | `ActionSync` | Action效果广播 |
| 3 | `OpDecisionRequest` | `Decision` | 决策请求 |
| 4 | `OpAvailable` | `Available` | 可用操作列表 |
| 5 | `OpMiniGameStart` | `MiniGameStart` | 小游戏开始 |
| 6 | `OpGameOver` | `GameOver` | 游戏结束 |
| 7 | `OpFullSync` | `StateSync` | 完整同步（断线重连） |

### Client → Server (100+)

| OpCode | 名称 | 数据类型 | 说明 |
|--------|------|----------|------|
| 100 | `OpRollDice` | `RollDice` | 投骰子请求（服务器计算） |
| 101 | `OpUseItem` | `UseItem` | 使用道具 |
| 102 | `OpUseSkill` | `UseSkill` | 使用阵营技能 |
| 103 | `OpUserChoice` | `UserChoice` | 决策选择回复 |
| 104 | `OpMiniGameResult` | `MiniGameResult` | 小游戏排名提交 |

## 使用示例

### 创建消息

```go
import "pkg/net"

// 创建状态同步消息
stateSync := &net.StateSync{
    GlobalState: "turn_loop",
    TurnState:   "main_action",
    TurnPlayer:  "player_001",
    Round:       1,
    Turn:        0,
    Players:     []net.Player{...},
}
msg, err := net.NewMessage(net.OpStateSync, stateSync)
```

### 解析消息

```go
// 解析消息数据
var stateSync net.StateSync
err := msg.ParseData(&stateSync)
```

## 命名规范

所有同步数据结构使用**无后缀命名**：

- `StateSync`（不是 `StateSyncData`）
- `ActionSync`（不是 `ActionSyncData`）
- `Player`（不是 `PlayerSync`）
- `Buff`（不是 `BuffSync`）
- `Item`（不是 `ItemSync`）

## 与 Nakama 集成

后续集成 Nakama 时：

1. 实现 `MatchHandler` 接口
2. 在 `MatchInit` 中初始化 HSM 和 Game
3. 在 `HandleMessage` 中调用 `hsm.OnRollDice/OnUseItem/OnUserChoice`
4. 使用 `internal/net.Builder` 构建同步数据

## 相关文档

- [internal/net/README.md](../../internal/net/README.md) - 构建器和骰子计算器
- [pkg/protocol/README.md](../protocol/README.md) - 协议接口层
- [internal/engine/hsm/README.md](../../internal/engine/hsm/README.md) - HSM 状态机