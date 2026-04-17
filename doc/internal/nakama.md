# Nakama Match Handler 集成设计文档

## 概述

本文档描述 Paradiced 游戏与 Nakama 权威服务器框架的集成设计。`internal/nakama` 包实现 Match Handler 接口，负责游戏生命周期管理、消息路由和客户端通信。

## 设计目标

1. **SDK 隔离**：通过 `DispatcherAdapter` 接口隔离 Nakama SDK 依赖
2. **可测试性**：使用 `MockDispatcherAdapter` 实现无真实 Nakama 服务器的测试
3. **架构解耦**：广播逻辑与核心游戏逻辑分离
4. **状态集成**：HSM 与 Nakama Match Handler 的生命周期同步

## 架构分层

```
┌──────────────────────────────────────────────────────────────────┐
│                    Nakama Server (外部依赖)                        │
│  - Match 创建/销毁                                                 │
│  - 客户端连接管理                                                   │
│  - 消息路由                                                        │
├──────────────────────────────────────────────────────────────────┤
│                    internal/nakama (本包)                          │
│  - DispatcherAdapter: SDK 隔离接口                                 │
│  - NakamaMatchHandler: Match 生命周期管理                          │
│  - NakamaBroadcastAdapter: 实现 pkg/net.BroadcastAdapter          │
│  - MockDispatcherAdapter: 测试用 Mock 实现                         │
├──────────────────────────────────────────────────────────────────┤
│                    pkg/net 协议层                                  │
│  - BroadcastAdapter: 广播抽象接口                                  │
│  - OpCode/Message: 消息操作码和结构                                │
│  - StateSync/TurnSync: 同步数据结构                                │
├──────────────────────────────────────────────────────────────────┤
│                    游戏核心层                                      │
│  - HSM: 分层状态机                                                 │
│  - Game: 游戏实例                                                  │
│  - EventBus: 事件订阅                                              │
│  - GameLog: 日志记录                                               │
└──────────────────────────────────────────────────────────────────┘
```

## DispatcherAdapter 接口

隔离 Nakama SDK 的核心接口，提供消息发送能力：

```go
// DispatcherAdapter isolates Nakama SDK from our codebase.
type DispatcherAdapter interface {
    // BroadcastMessage sends a message to all players in the match.
    BroadcastMessage(opCode int64, data []byte) error

    // SendMessage sends a message to a specific player.
    SendMessage(playerID string, opCode int64, data []byte) error
}
```

### 设计理由

1. **测试隔离**：测试无需真实 Nakama 服务器
2. **依赖解耦**：核心包不引入 Nakama SDK 依赖
3. **未来扩展**：支持切换其他权威服务器框架

## MockDispatcherAdapter

测试实现，捕获所有消息用于验证：

```go
type MockDispatcherAdapter struct {
    Broadcasts []BroadcastRecord          // 捕获的广播消息
    Messages   map[string][]MessageRecord  // 捕获的单播消息（按玩家ID）
}

// 辅助方法
func (m *MockDispatcherAdapter) GetBroadcasts() []BroadcastRecord
func (m *MockDispatcherAdapter) GetMessages(playerID string) []MessageRecord
func (m *MockDispatcherAdapter) ParseBroadcastData(index int, target interface{}) error
func (m *MockDispatcherAdapter) ParseMessageData(playerID string, index int, target interface{}) error
func (m *MockDispatcherAdapter) CountBroadcasts() int
func (m *MockDispatcherAdapter) CountMessages(playerID string) int
func (m *MockDispatcherAdapter) Clear()
```

### 使用示例

```go
func TestBroadcastStateSync(t *testing.T) {
    handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)
    mockDispatcher := NewMockDispatcherAdapter()
    handler.WithDispatcher(mockDispatcher)

    // ... 初始化和执行逻辑 ...

    // 验证广播消息
    broadcasts := mockDispatcher.GetBroadcasts()
    if len(broadcasts) == 0 {
        t.Error("expected at least one broadcast")
    }

    // 解析广播数据
    var stateSync net.StateSync
    mockDispatcher.ParseBroadcastData(0, &stateSync)

    if stateSync.GlobalState != "turn_loop" {
        t.Errorf("state = %s, want turn_loop", stateSync.GlobalState)
    }
}
```

## NakamaMatchHandler

Match Handler 核心结构，管理游戏生命周期：

```go
type NakamaMatchHandler struct {
    // 核心组件
    game      *engine.Game       // 游戏实例
    hsm       *hsm.HSM           // 分层状态机
    mapEngine *gamemap.MapEngine // 地图引擎
    diceMgr   *rng.DiceManager   // 骰子管理器

    // 消息分发器
    dispatcher DispatcherAdapter // SDK 隔离接口

    // Match 标识
    matchID string // Nakama Match ID

    // 玩家管理
    players    map[string]*core.Player // userID -> Player
    playerList []string                // 按加入顺序的玩家列表

    // 配置
    maxPlayers  int    // 最大玩家数（默认 4）
    mapLength   int    // 地图长度（默认 100）
    randomSeed  int64  // 随机种子
}
```

### 核心方法

| 方法 | 说明 | 调用时机 |
|------|------|----------|
| `MatchInit()` | 初始化游戏实例和 HSM | 所有玩家加入后 |
| `MatchLoop(delta)` | 每帧更新游戏状态 | Nakama 定时调用 |
| `MatchStop()` | 终止游戏，清理资源 | Match 结束时 |
| `HandleMessage(sender, data)` | 处理客户端消息 | 收到玩家消息时 |
| `HandlePresenceJoin(userID, metadata)` | 处理玩家加入 | 新玩家连接时 |
| `HandlePresenceLeave(userID)` | 处理玩家离开 | 玩家断开时 |

## NakamaBroadcastAdapter

实现 `pkg/net.BroadcastAdapter` 接口，连接 HSM 与客户端：

```go
type NakamaBroadcastAdapter struct {
    handler *NakamaMatchHandler
}

// 实现 BroadcastAdapter 所有方法
func (a *NakamaBroadcastAdapter) BroadcastStateSync(state *net.StateSync) error
func (a *NakamaBroadcastAdapter) BroadcastTurnSync(turn *net.TurnSync) error
func (a *NakamaBroadcastAdapter) SendDecision(playerID string, decision *net.Decision) error
func (a *NakamaBroadcastAdapter) SendAvailable(playerID string, available *net.Available) error
func (a *NakamaBroadcastAdapter) BroadcastMiniGameStart(start *net.MiniGameStart) error
func (a *NakamaBroadcastAdapter) BroadcastMiniGameResult(result *net.MiniGameResult) error
func (a *NakamaBroadcastAdapter) BroadcastGameOver(over *net.GameOver) error
func (a *NakamaBroadcastAdapter) SendFullSync(playerID string, state *net.StateSync, turn *net.TurnSync) error
```

### HSM 集成

StateContext 持有 BroadcastAdapter，状态可直接广播：

```go
// lifecycle.go - MatchInit
func (h *NakamaMatchHandler) MatchInit() error {
    // 创建广播适配器
    broadcastAdapter := NewNakamaBroadcastAdapter(h)

    // 创建 StateContext
    ctx := hsm.NewStateContext().
        WithGame(h.game).
        WithBroadcast(broadcastAdapter)

    // 启动 HSM
    h.hsm.Start(hsm.StateMatchInit, ctx)
}
```

## 消息处理流程

### OpCode 路由

```go
func (h *NakamaMatchHandler) HandleMessage(sender string, data []byte) error {
    var base struct {
        OpCode string `json:"op_code"`
    }
    json.Unmarshal(data, &base)

    switch base.OpCode {
    case strconv.FormatInt(int64(net.OpRollDice), 10):
        return h.handleRollDice(sender)
    case strconv.FormatInt(int64(net.OpUseItem), 10):
        return h.handleUseItem(sender, data)
    case strconv.FormatInt(int64(net.OpUseSkill), 10):
        return h.handleUseSkill(sender)
    case strconv.FormatInt(int64(net.OpUserChoice), 10):
        return h.handleUserChoice(sender, data)
    case strconv.FormatInt(int64(net.OpMiniGameResultSubmit), 10):
        return h.handleMiniGameResult(sender, data)
    default:
        return nil // 未知 OpCode 忽略
    }
}
```

### 投骰子处理

```go
func (h *NakamaMatchHandler) handleRollDice(sender string) error {
    player := h.GetPlayer(sender)
    if player == nil {
        return nil // 未知玩家
    }

    // 检查是否为当前回合玩家
    if h.getCurrentPlayer() != player {
        return nil
    }

    // 检查是否在 MainAction 状态
    if h.hsm.GetCurrentStateID() != hsm.StateMainAction {
        return nil
    }

    // 使用 DiceManager 计算骰子结果
    steps := h.diceMgr.RollSpecialDice(sender)

    // 通知 HSM（实际实现）
    // ctx := hsm.NewStateContext().WithPlayer(player)
    // h.hsm.OnRollDice(steps, ctx)

    return nil
}
```

## 状态同步机制

### 完整流程

```
【MatchInit】
→ 初始化 Game、HSM、MapEngine
→ 启动 HSM → 进入 MatchInit 状态
→ 自动转移至 RoundMiniGame

【RoundMiniGame】
→ 广播 MiniGameStart
→ 等待所有玩家提交排名
→ MiniGameResult 广播

【RoundPrep】
→ 根据排名分配骰子等级
→ 进入 TurnLoop

【TurnLoop】
→ 选择当前玩家
→ 进入 TurnUpkeep 子状态

【TurnUpkeep】
→ BroadcastStateSync(turn_upkeep)
→ 触发 BeforeTurn Buff
→ 进入 MainAction

【MainAction】
→ BroadcastStateSync(main_action)
→ SendAvailable(道具/技能/骰子)
→ 等待玩家消息

【Client: OpRollDice】
→ HandleMessage → handleRollDice
→ HSM.OnRollDice → TurnMoving

【TurnMoving】
→ BroadcastStateSync(turn_moving)
→ 执行 MoveAction
→ 路径效果（坠落/反超）

【TurnLanded】
→ BroadcastStateSync(turn_landed)
→ 触发 OnLand 效果

【TurnEvent】
→ BroadcastStateSync(turn_event)
→ 抽取事件、执行效果

【TurnEnd】
→ BroadcastStateSync(turn_end)
→ TickBuffs、阵营充能
→ BroadcastTurnSync(本回合 Action)
→ 下一玩家
```

### 断线重连

```go
// HandlePresenceJoin - 重连逻辑
func (h *NakamaMatchHandler) HandlePresenceJoin(userID string, metadata map[string]string) error {
    // 检查玩家是否已在游戏中
    if h.players[userID] != nil {
        // 重连：发送完整状态
        broadcast := NewNakamaBroadcastAdapter(h)
        stateSync := buildStateSync(h)
        turnSync := buildTurnSync(h)
        broadcast.SendFullSync(userID, stateSync, turnSync)
        return nil
    }

    // 新玩家加入...
}
```

## 文件结构

```
internal/nakama/
├── handler.go          # NakamaMatchHandler 主结构
├── dispatcher.go       # DispatcherAdapter 接口定义
├── dispatcher_mock.go  # MockDispatcherAdapter 测试实现
├── broadcast.go        # NakamaBroadcastAdapter 实现
├── lifecycle.go        # MatchInit/MatchLoop/MatchStop
├── message.go          # HandleMessage 消息路由
├── presence.go         # HandlePresenceJoin/Leave
├── handler_test.go     # Handler 测试
├── broadcast_test.go   # Broadcast 测试
├── lifecycle_test.go   # Lifecycle 测试
├── message_test.go     # Message 测试
├── presence_test.go    # Presence 测试
└── README.md           # 包文档
```

## 测试覆盖

| 测试文件 | 覆盖内容 | 关键场景 |
|----------|----------|----------|
| handler_test.go | Handler 创建、玩家管理、状态获取 | 配置默认值、GetPlayer |
| broadcast_test.go | 所有广播方法 | Mock 消息捕获、数据解析 |
| lifecycle_test.go | MatchInit/Loop/Stop、完整游戏流程 | HSM 启动/停止、状态转换 |
| message_test.go | 消息路由、各 OpCode 处理 | 未知 OpCode、非当前玩家 |
| presence_test.go | 玩家加入/离开、满场 | 重复加入、断线 |

### 测试模式

```go
// 标准测试模式
func TestXxx(t *testing.T) {
    handler := NewNakamaMatchHandler("match-001", 12345, 4, 100)
    mockDispatcher := NewMockDispatcherAdapter()
    handler.WithDispatcher(mockDispatcher)

    // 初始化
    handler.addPlayer("user-001", protocol.FactionQingLong)
    handler.MatchInit()

    // 执行操作...

    // 验证 Mock 捕获
    broadcasts := mockDispatcher.GetBroadcasts()
    // 断言...
}
```

## 与其他包的关系

| 包 | 关系 | 说明 |
|----|------|------|
| `pkg/net` | 实现接口 | 实现 BroadcastAdapter 接口 |
| `pkg/rng` | 使用 | DiceManager 骰子计算 |
| `internal/engine` | 使用 | Game 实例、HSM 状态机 |
| `internal/gamemap` | 使用 | MapEngine 地图计算 |
| `internal/core` | 使用 | Player 数据结构 |

## 后续工作

### Phase 1: 完整集成（当前）

- DispatcherAdapter 接口 ✅
- MockDispatcherAdapter ✅
- NakamaBroadcastAdapter ✅
- 基础测试覆盖 ✅

### Phase 2: Nakama SDK 真实集成

- RealDispatcherAdapter 实现
- Nakama Server 配置
- Match 注册和生命周期

### Phase 3: 高级功能

- 断线重连完整流程
- 超时处理优化
- 性能优化（批量消息）

## 相关文档

- [pkg/net/README.md](../../pkg/net/README.md) - 协议层接口
- [internal/engine/hsm/README.md](../engine/hsm/README.md) - HSM 状态机
- [doc/internal/net_protocol.md](../net_protocol.md) - 协议层设计
- [doc/internal/hsm_design.md](../hsm_design.md) - HSM 设计文档