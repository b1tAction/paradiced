# 客户端-服务器交互协议层设计

## 概述

本文档描述 Paradiced 游戏客户端-服务器交互协议层的设计与实现。基于 Nakama 权威服务器架构，提供状态同步、回合同步、决策请求等核心功能。

## 架构分层

```
┌──────────────────────────────────────────────────────────────────┐
│                    Nakama Match Handler Layer                      │
│  (实现 BroadcastAdapter 接口，处理客户端连接、消息路由、状态广播)    │
├──────────────────────────────────────────────────────────────────┤
│                    pkg/net 协议层                                  │
│  - OpCode: 消息操作码                                              │
│  - Message: 基础消息结构                                           │
│  - StateSync: 同步数据结构（含增量 LogEntry）                                │
│  - Decision: 决策请求结构                                          │
│  - BroadcastAdapter: 广播抽象接口                                  │
├──────────────────────────────────────────────────────────────────┤
│                    internal/net 构建层                             │
│  - Builder: 将内部数据转换为协议数据                                │
│  - 使用 pkg/rng.DiceManager 进行骰子计算                           │
├──────────────────────────────────────────────────────────────────┤
│                    游戏核心层 (现有)                               │
│  - HSM: 状态机控制                                                 │
│  - Action: 效果执行                                                │
│  - GameLog: 日志记录                                               │
│  - EventBus: 事件订阅                                              │
└──────────────────────────────────────────────────────────────────┘
```

## 设计原则

1. **权威服务器模式**：所有计算在服务端完成，客户端只负责渲染和发送请求
2. **抽象接口设计**：`pkg/net.BroadcastAdapter` 定义抽象接口，不引入 Nakama SDK 依赖
3. **增量 LogEntry 模式**：StateSync 通过 `Entries` 字段携带增量 LogEntry，客户端顺序渲染
4. **Metadata 契约**：LogEntry.Metadata 字段遵循 [doc/metadata/logentry.md](../metadata/logentry.md) 契约
5. **命名统一**：所有同步数据结构使用无后缀命名（StateSync、Player、Buff）

## 消息操作码

### Server → Client (1-99)

| OpCode | 名称 | 数据类型 | 说明 |
|--------|------|----------|------|
| 1 | `OpStateSync` | `StateSync` | 状态同步（含增量 LogEntry） |
| 3 | `OpDecisionRequest` | `Decision` | 决策请求 |
| 4 | `OpAvailable` | `Available` | 可用操作列表（道具/技能） |
| 5 | `OpMiniGameStart` | `MiniGameStart` | 小游戏开始 |
| 6 | `OpMiniGameResult` | `MiniGameResult` | 小游戏结果广播 |
| 7 | `OpGameOver` | `GameOver` | 游戏结束 |
| 8 | `OpFullSync` | `StateSync` | 完整同步（断线重连，含当前回合全部 LogEntry） |

### Client → Server (100+)

| OpCode | 名称 | 数据类型 | 说明 |
|--------|------|----------|------|
| 100 | `OpRollDice` | `RollDice` | 投骰子请求（服务器计算） |
| 101 | `OpUseItem` | `UseItem` | 使用道具 |
| 102 | `OpUseSkill` | `UseSkill` | 使用阵营技能 |
| 103 | `OpUserChoice` | `UserChoice` | 决策选择回复 |
| 104 | `OpMiniGameDataSubmit` | `MiniGameDataSubmit` | 小游戏数据提交（服务器计算排名） |

## 核心数据结构

### StateSync

完整游戏状态同步，含增量 LogEntry 数据。每次 HSM 状态转换时，`BuildStateSync()` 获取自上次广播以来的新 LogEntry：

```go
type StateSync struct {
    GlobalState string    `json:"global_state"`  // "turn_loop", "round_mini_game"
    TurnState   string    `json:"turn_state"`    // "main_action", "turn_moving"
    TurnPlayer  string    `json:"turn_player"`   // 当前回合玩家ID
    Round       int       `json:"round"`
    Turn        int       `json:"turn"`
    Paused      bool      `json:"paused"`        // 等待决策中
    Players     []Player  `json:"players"`
    Map         MapInfo   `json:"map"`
    Entries     []gamelog.LogEntry `json:"entries,omitempty"` // 增量 LogEntry
}
```

**Entries 增量机制**：
- 每次 `BuildStateSync()` 调用 `GameLog.GetNewEntries()` + `MarkBroadcasted()` 获取增量 LogEntry
- `omitempty` 确保无新 LogEntry 时 JSON 不包含此字段
- 断线重连使用 `BuildFullSyncStateSync()` 获取当前回合全部 LogEntry

### LogEntry 结构

```go
type LogEntry struct {
    Timestamp  time.Time         `json:"timestamp"`
    Type       constants.EntryType `json:"type"`        // "action", "state"
    ActionType string            `json:"action_type,omitempty"` // "damage", "move", "heal"
    Target     string            `json:"target,omitempty"`
    Delta      int               `json:"delta,omitempty"`     // HP/LP变化值
    Source     string            `json:"source,omitempty"`
    Metadata   *util.Metadata    `json:"metadata,omitempty"`  // 效果详情
}
```

**Metadata 契约**：各 ActionType 的 Metadata 字段详见 [doc/metadata/logentry.md](../metadata/logentry.md)。

客户端渲染逻辑：

```javascript
for (const entry of turnSync.entries) {
    switch (entry.action_type) {
        case "damage":
            const blockedBy = entry.metadata?.blocked_by;
            playDamageAnimation(entry.target, entry.delta, entry.source, blockedBy);
            break;
        case "move":
            const path = entry.metadata?.path || [];
            playMoveAnimation(entry.target, path);
            break;
        case "add_buff":
            const buffType = entry.metadata?.buff_type;
            playBuffAnimation(entry.target, buffType);
            break;
        // ...
    }
}
```

### Player

玩家状态快照，Builder 从 `core.Player.Metadata` 提取已知键转为 typed 字段：

```go
type Player struct {
    PlayerID    string `json:"player_id"`
    Faction     string `json:"faction"`      // snake_case: "qing_long", "zhu_que"
    Position    int    `json:"position"`
    HP          int    `json:"hp"`
    LP          int    `json:"lp"`
    Buffs       []Buff `json:"buffs"`        // 带Name的Buff列表
    Items       []Item `json:"items"`        // 带Name的Item列表
    Charge      int    `json:"charge"`       // 阵营充能数（青龙/玄武）
    FireCounter int    `json:"fire_counter"` // 朱雀火计数
    IsDead      bool   `json:"is_dead"`
    SkipTurn    bool   `json:"skip_turn"`
}
```

### Buff/Item（带显示名）

为客户端 UI 提供显示名：

```go
type Buff struct {
    Type     string `json:"type"`     // "divine", "curse"
    Name     string `json:"name"`     // "神眷", "诅咒"（从定义获取）
    Duration int    `json:"duration"` // -1表示永久
}

type Item struct {
    ID   string `json:"id"`   // Item实例UUID
    Type string `json:"type"` // "any_door", "reverse_clock"
    Name string `json:"name"` // "任意门", "反方向的钟"
}
```

### Decision

决策请求，带Context标识来源：

```go
type Decision struct {
    ID      string   `json:"id"`       // 决策ID
    Prompt  string   `json:"prompt"`   // 提示文本
    Context string   `json:"context"`  // 来源标识："Item_AnyDoor", "Buff_Divine"
    Options []Option `json:"options"`  // 选项列表
    Timeout int      `json:"timeout"`  // 超时秒数
    Default int      `json:"default"`  // 默认选项索引
}

type Option struct {
    ID     string `json:"id"`             // 选项ID
    Label  string `json:"label"`          // 显示文本
    Effect string `json:"effect,omitempty"` // 效果描述
}
```

Context 用于客户端区分决策来源，提供不同 UI 样式。

### Available

可用操作列表：

```go
type Available struct {
    Items       []Item `json:"items"`         // 可用道具（带Name）
    CanUseSkill bool   `json:"can_use_skill"` // 阵营技能可用
    DiceType    string `json:"dice_type"`     // "gold", "silver", "copper", "wood"
}
```

**骰子类型说明**：使用 `pkg/rng.DiceType` 枚举，Builder 转换为字符串用于协议。

| DiceType | 字符串值 | 范围 | 权重分布 | 来源 |
|----------|----------|------|----------|------|
| DiceTypeGold | "gold" | 1-6 | 5-6权重70% | 排名 1 |
| DiceTypeSilver | "silver" | 1-6 | 5-6权重50% | 排名 2 |
| DiceTypeCopper | "copper" | 1-6 | 5-6权重40% | 排名 3 |
| DiceTypeWood | "wood" | 1-6 | 均匀分布 | 排名 4 |

### MiniGameStart/MiniGameResult/MiniGameDataSubmit

小游戏相关：

```go
type MiniGameStart struct {
    GameType  string       `json:"game_type"`            // "dice_race", "count_seconds"
    Players   []string     `json:"players"`              // 参赛玩家ID列表
    Connection *MiniGameConn `json:"connection,omitempty"` // 小游戏服务连接信息（前端模式为nil）
}

type MiniGameConn struct {
    URL    string `json:"url"`     // 小游戏服务 WebSocket URL
    RoomID string `json:"room_id"` // Colyseus 房间 ID
    Token  string `json:"token"`   // 认证 Token
}

type MiniGameDataSubmit struct {
    GameType string                 `json:"game_type"` // 必须匹配 MiniGameStart.GameType
    GameData map[string]interface{} `json:"game_data"` // 原始小游戏数据（score/time等）
}

type MiniGameResult struct {
    Rankings []RankingEntry `json:"rankings"`
}

type RankingEntry struct {
    PlayerID    string                 `json:"player_id"`
    DisplayName string                 `json:"display_name"`
    Rank        int                    `json:"rank"`
    GameData    map[string]interface{} `json:"game_data,omitempty"`
}
```

**game_type 可选值与排名规则**：

| game_type | 含义 | game_data 格式 | 排名规则 |
|-----------|------|----------------|----------|
| `dice_race` | 投骰比大小 | `{ dice1: int, dice2: int, score: dice1+dice2 }` | score 降序（越大越好） |
| `count_seconds` | 计秒小游戏 | `{ elapsed: float64, deviation: |elapsed-5.0| }` | deviation 升序（越接近5秒越好） |
| `coin_flip` | 翻硬币 | 未实现，暂不可用 | - |

**MiniGameStart.Connection 说明**：
- `connection` 为 `nil` 表示前端驱动模式（Frontend）：客户端在本地运行小游戏，完成后提交 `game_data`
- `connection` 非 `nil` 表示 RPC 模式：客户端连接到 Colyseus 小游戏服务，服务端直接上报排名

**MiniGameDataSubmit vs 旧 MiniGameResultSubmit**：
- 客户端提交 `game_data`（原始数据），而非 `rank`（排名）
- 服务器通过 `RankCalculator` 根据 `game_type` 的排名规则计算排名

### FullSync

完整同步（断线重连）直接使用 `StateSync` 结构。通过 `BuildFullSyncStateSync()` 获取包含当前回合全部 LogEntry 的 StateSync：

```go
// 断线重连使用 BuildFullSyncStateSync 获取完整状态
stateSync := builder.BuildFullSyncStateSync()
broadcastAdapter.SendFullSync(playerID, stateSync)
```

## BroadcastAdapter 接口

抽象广播接口，供 HSM 和 ActionContext 使用：

```go
type BroadcastAdapter interface {
    // 状态广播（含增量 LogEntry）
    BroadcastStateSync(state *StateSync) error

    // 单播
    SendDecision(playerID string, decision *Decision) error
    SendAvailable(playerID string, available *Available) error
    SendFullSync(playerID string, state *StateSync) error  // 断线重连（含全部当前回合 LogEntry）

    // 小游戏
    BroadcastMiniGameStart(start *MiniGameStart) error
    BroadcastMiniGameResult(result *MiniGameResult) error

    // 游戏结束
    BroadcastGameOver(over *GameOver) error

    // 开始游戏确认
    BroadcastStartGameAck(ack *StartGameAck) error
}
```

**注意**：`BroadcastTurnSync` 和 `TurnSync` 已移除，LogEntry 数据现在通过 `StateSync.Entries` 增量携带。

## Faction SnakeCase 转换

`constants.Faction` 使用 `SnakeCase()` 方法获取 snake_case 值用于 JSON：

```go
// pkg/constants/faction.go
func (f Faction) SnakeCase() string {
    names := map[Faction]string{
        FactionQingLong: "qing_long",
        FactionZhuQue:   "zhu_que",
        FactionBaiHu:    "bai_hu",
        FactionXuanWu:   "xuan_wu",
    }
    return names[f]
}

// internal/net/builder.go
func (b *Builder) BuildPlayer(p *core.Player) pkgnet.Player {
    return pkgnet.Player{
        Faction: p.Faction.SnakeCase(), // "zhu_que" 而非 "ZhuQue"
        ...
    }
}
```

## 骰子计算

使用 `pkg/rng.DiceManager` 进行骰子计算（权威服务器）：

```go
import "github.com/b1tAction/paradiced/pkg/rng"

// 创建骰子管理器（传入游戏 RNG）
diceMgr := rng.NewDiceManager(game.RNG)

// 根据排名分配骰子类型
diceMgr.AssignDice(playerID, rank) // rank 1-4

// 滚骰子（返回 1-6，使用加权分布）
steps := diceMgr.RollSpecialDice(playerID)
```

## 与 Nakama Handler 集成

`internal/nakama/broadcast.go` 实现 BroadcastAdapter 接口：

```go
type NakamaBroadcastAdapter struct {
    handler *NakamaMatchHandler
}

func (a *NakamaBroadcastAdapter) BroadcastStateSync(state *pkgnet.StateSync) error {
    data, _ := json.Marshal(state)
    return a.handler.dispatcher.BroadcastMessage(int64(pkgnet.OpStateSync), data)
}

func (a *NakamaBroadcastAdapter) SendFullSync(playerID string, state *pkgnet.StateSync) error {
    data, _ := json.Marshal(state)
    return a.handler.dispatcher.SendMessage(playerID, int64(pkgnet.OpFullSync), data)
}
```

### 集成架构

```
Nakama Server → NakamaMatchHandler → DispatcherAdapter → pkg/net.BroadcastAdapter
                     ↓
              HSM StateContext.WithBroadcast()
```

完整集成文档见：[doc/internal/nakama.md](../nakama.md)

## 相关文档

- [pkg/net/README.md](../../pkg/net/README.md) - 协议层包文档
- [internal/net/README.md](../../internal/net/README.md) - 构建层包文档
- [pkg/rng/README.md](../../pkg/rng/README.md) - RNG引擎和骰子类型
- [internal/engine/hsm/README.md](../engine/hsm/README.md) - HSM状态机
- [pkg/gamelog/README.md](../../pkg/gamelog/README.md) - 游戏日志系统
- [doc/internal/nakama.md](../nakama.md) - Nakama Match Handler 集成
- [doc/metadata/logentry.md](../metadata/logentry.md) - LogEntry.Metadata 契约