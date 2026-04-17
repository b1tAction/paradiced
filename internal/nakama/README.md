# Package internal/nakama - Nakama Match Handler 实现

本包实现 Paradiced 游戏的 Nakama Match Handler，负责权威服务器的游戏生命周期管理、消息路由和客户端通信。

## 设计目标

1. **SDK 隔离**：通过 `DispatcherAdapter` 接口隔离 Nakama SDK 依赖
2. **可测试性**：使用 `MockDispatcherAdapter` 实现无真实 Nakama 服务器的测试
3. **架构解耦**：广播逻辑与核心游戏逻辑分离

## 文件说明

| 文件 | 说明 |
|------|------|
| `handler.go` | NakamaMatchHandler 主结构、配置管理 |
| `dispatcher.go` | DispatcherAdapter 接口定义、BroadcastRecord/MessageRecord |
| `dispatcher_mock.go` | MockDispatcherAdapter 测试实现（消息捕获） |
| `broadcast.go` | NakamaBroadcastAdapter 实现 pkg/net.BroadcastAdapter |
| `lifecycle.go` | MatchInit/MatchLoop/MatchStop、addPlayer/assignFactions |
| `message.go` | HandleMessage 消息路由、各 OpCode 处理器 |
| `presence.go` | HandlePresenceJoin/Leave、玩家连接管理 |

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

| OpCode (Client→Server) | 处理方法 | 说明 |
|------------------------|----------|------|
| 100 (RollDice) | handleRollDice | 投骰子 |
| 101 (UseItem) | handleUseItem | 使用道具 |
| 102 (UseSkill) | handleUseSkill | 阵营技能 |
| 103 (UserChoice) | handleUserChoice | 决策回复 |
| 104 (MiniGameResultSubmit) | handleMiniGameResult | 小游戏排名 |

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