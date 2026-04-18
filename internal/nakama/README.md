# Package internal/nakama - Nakama Match Handler 实现

本包实现 Paradiced 游戏的 Nakama Match Handler，负责权威服务器的游戏生命周期管理、消息路由和客户端通信。

## 设计目标

1. **SDK 隔离**：通过 `DispatcherAdapter` 接口隔离 Nakama SDK 依赖
2. **可测试性**：使用 `MockDispatcherAdapter` 实现无真实 Nakama 服务器的测试
3. **架构解耦**：广播逻辑与核心游戏逻辑分离
4. **断线重连**：支持玩家临时断线后重新连接

## 文件说明

| 文件 | 说明 |
|------|------|
| `handler.go` | NakamaMatchHandler 主结构、配置管理 |
| `dispatcher.go` | DispatcherAdapter 接口定义、BroadcastRecord/MessageRecord |
| `dispatcher_mock.go` | MockDispatcherAdapter 测试实现（消息捕获） |
| `dispatcher_real.go` | RealDispatcherAdapter 生产实现（真实 Nakama SDK） |
| `adapter.go` | NakamaMatchAdapter/NakamaMatchHandlerWrapper（Nakama 包装器） |
| `broadcast.go` | NakamaBroadcastAdapter 实现 pkg/net.BroadcastAdapter |
| `lifecycle.go` | MatchInit/MatchLoop/MatchStop、addPlayer/assignFactions |
| `message.go` | HandleMessage 消息路由、各 OpCode 处理器 |
| `presence.go` | HandlePresenceJoin/Leave、玩家连接管理、断线重连 |

## DispatcherAdapter 接口

SDK 隔离接口，提供消息发送能力：

```go
type DispatcherAdapter interface {
    BroadcastMessage(opCode int64, data []byte) error
    SendMessage(playerID string, opCode int64, data []byte) error
}
```

## 使用示例

### 创建 Match Handler

```go
import "github.com/b1tAction/paradiced/internal/nakama"

// 创建 Handler
handler := nakama.NewNakamaMatchHandler("match-001", 12345, 4, 100)

// 设置 Dispatcher
mockDispatcher := nakama.NewMockDispatcherAdapter()
handler.WithDispatcher(mockDispatcher)

// 添加玩家
handler.addPlayer("user-001", protocol.FactionQingLong)
handler.addPlayer("user-002", protocol.FactionZhuQue)

// 初始化游戏
handler.MatchInit()

// 运行游戏循环
handler.MatchLoop(100 * time.Millisecond)
```

### 测试 Mock 使用

```go
func TestBroadcast(t *testing.T) {
    handler := nakama.NewNakamaMatchHandler("match-001", 12345, 4, 100)
    mockDispatcher := nakama.NewMockDispatcherAdapter()
    handler.WithDispatcher(mockDispatcher)

    // 执行操作...

    // 验证广播消息
    broadcasts := mockDispatcher.GetBroadcasts()
    if len(broadcasts) == 0 {
        t.Error("expected broadcast")
    }

    // 解析消息数据
    var stateSync net.StateSync
    mockDispatcher.ParseBroadcastData(0, &stateSync)

    // 断言...
    mockDispatcher.Clear() // 清空捕获
}
```

### 处理客户端消息

```go
// 客户端发送投骰子请求
data, _ := json.Marshal(RollDiceRequest{OpCode: "1"})
handler.HandleMessage("user-001", data)

// 客户端发送道具使用请求
data, _ := json.Marshal(UseItemRequest{
    OpCode: "2",
    ItemID: "item-uuid",
})
handler.HandleMessage("user-001", data)
```

## 消息 OpCode 路由

### Server → Client (1-99)

| OpCode | 处理方法 | 说明 |
|--------|----------|------|
| 1 (OpStateSync) | broadcastStateSync | 状态同步 |
| 2 (OpTurnSync) | broadcastTurnSync | 回合效果列表 |
| 3 (OpDecisionRequest) | sendDecision | 决策请求 |
| 4 (OpAvailable) | sendAvailable | 可用操作列表 |
| 5 (OpMiniGameStart) | broadcastMiniGameStart | 小游戏开始 |
| 6 (OpMiniGameResult) | broadcastMiniGameResult | 小游戏结果 |
| 7 (OpGameOver) | broadcastGameOver | 游戏结束 |
| 8 (OpFullSync) | sendFullSync | 完整同步（重连） |
| 9 (OpActionRejected) | sendActionRejected | 动作拒绝（新增） |

### Client → Server (100+)

| OpCode (Client→Server) | 处理方法 | 说明 |
|------------------------|----------|------|
| 100 (RollDice) | handleRollDice | 投骰子 |
| 101 (UseItem) | handleUseItem | 使用道具 |
| 102 (UseSkill) | handleUseSkill | 阵营技能 |
| 103 (UserChoice) | handleUserChoice | 决策回复 |
| 104 (MiniGameResultSubmit) | handleMiniGameResult | 小游戏排名 |

## ActionRejected 拒绝反馈

当客户端发送无效请求时，服务器发送 `OpActionRejected` 消息：

```go
// pkg/net/sync.go
type ActionRejected struct {
    OpCode  OpCode `json:"op_code"`   // 被拒绝的操作码
    Reason  string `json:"reason"`    // 拒绝原因
    Message string `json:"message"`   // 人类可读的错误信息
}
```

**常见拒绝原因**：

| Reason | 说明 | 触发场景 |
|--------|------|----------|
| `not_current_player` | 非当前回合玩家 | 其他玩家试图掷骰子/使用道具 |
| `invalid_state` | 无效的游戏状态 | 在 MainAction 阶段发送 MoveRequest |
| `item_not_found` | 道具不存在 | 使用不存在的道具 ID |
| `item_not_usable` | 道具不可用 | 道具已使用过或非 PhaseAnyTime |
| `skill_not_ready` | 技能未就绪 | 充能不足或技能冷却中 |
| `invalid_choice` | 无效的决策选择 | 决策 ID 不匹配或选项超出范围 |

**使用示例**：

```go
// internal/nakama/message.go
func (h *NakamaMatchHandler) handleRollDice(sender string) error {
    player := h.getPlayerByUserID(sender)
    currentPlayer := h.getCurrentPlayer()

    if player != currentPlayer {
        // 发送拒绝消息
        return h.sendActionRejected(sender, pkgnet.ActionRejected{
            OpCode:  pkgnet.OpRollDice,
            Reason:  "not_current_player",
            Message: "当前不是你的回合",
        })
    }

    // 正常处理...
}
```

## 日志调试

### 开发环境日志

在开发环境中，可以通过 `--verbose` 参数启用详细日志：

```bash
# CLI 客户端
pdcli playtest run --players 2 --verbose

# Nakama 服务器日志
docker logs -f nakama
```

### 结构化日志格式

```
INFO [Nakama] MatchInit: match_id=match-001, players=4
DEBUG [Nakama] handleRollDice: player=user-001, dice_type=gold
INFO [HSM] State transition: from=main_action, to=turn_moving
DEBUG [Action] Executing: type=move, target=user-001, steps=5
INFO [Nakama] BroadcastTurnSync: entries=3, round=1, turn=0
```

### 使用 zap 日志记录器

```go
// internal/nakama/logger.go (待实现)
type Logger struct {
    *zap.Logger
    matchID string
}

func NewLogger(matchID string, verbose bool) *Logger {
    config := zap.NewDevelopmentConfig()
    if !verbose {
        config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
    }
    logger, _ := config.Build()
    return &Logger{
        Logger:  logger,
        matchID: matchID,
    }
}

func (l *Logger) Info(msg string, fields ...zap.Field) {
    l.Logger.Info(msg, append([]zap.Field{zap.String("match_id", l.matchID)}, fields...)...)
}
```

## NakamaBroadcastAdapter

实现 `pkg/net.BroadcastAdapter` 接口，连接 HSM 与客户端：

```go
broadcast := nakama.NewNakamaBroadcastAdapter(handler)

// 广播状态同步
broadcast.BroadcastStateSync(stateSync)

// 广播回合 Action 列表
broadcast.BroadcastTurnSync(turnSync)

// 发送决策请求
broadcast.SendDecision(playerID, decision)

// 发送可用操作
broadcast.SendAvailable(playerID, available)
```

## Match 生命周期

```
MatchInit → 初始化 Game/HSM/MapEngine → HSM 启动
MatchLoop → 每帧更新 → HSM.Update → 状态转换
MatchStop → 停止 HSM → 清理资源
```

## RealDispatcherAdapter

生产环境使用真实 Nakama SDK：

```go
// 创建适配器（需要真实 Nakama Match）
adapter := NewNakamaMatchAdapter(match)
dispatcher := NewRealDispatcherAdapter(ctx, adapter)
handler.WithDispatcher(dispatcher)

// 支持的方法
dispatcher.BroadcastMessage(opCode, data)  // 广播消息
dispatcher.SendMessage(playerID, opCode, data)  // 发送给特定玩家
dispatcher.UpdatePresence(userID, presence)  // 更新连接状态
dispatcher.RemovePresence(userID)  // 移除连接
dispatcher.RefreshPresences()  // 刷新连接列表
```

## 断线重连

玩家断线时不会立即从游戏移除，而是标记为 disconnected 状态：

```go
// 玩家断线
handler.HandlePresenceLeave("user-001")
// 玩家仍在 players map 中，但 disconnected["user-001"] = true

// 玩家重连
handler.HandlePresenceJoin("user-001", nil)
// disconnected["user-001"] = false
// 发送完整状态同步 (FullSync) 给重连玩家
```

检查玩家连接状态：

```go
if handler.IsPlayerConnected(userID) {
    // 玩家在线
}

connectedPlayers := handler.GetConnectedPlayers()
// 返回当前在线玩家列表
```

## 与其他包的关系

- `pkg/net`: 实现 BroadcastAdapter 接口
- `pkg/rng`: DiceManager 骰子计算
- `internal/engine`: Game 实例、HSM 状态机
- `internal/gamemap`: MapEngine 地图计算
- `internal/core`: Player 数据结构

## 测试覆盖率

67.9% statements

## 相关文档

- [doc/internal/nakama.md](../../doc/internal/nakama.md) - Nakama 集成设计文档
- [pkg/net/README.md](../../pkg/net/README.md) - 协议层接口
- [internal/engine/hsm/README.md](../engine/hsm/README.md) - HSM 状态机