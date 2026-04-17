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
│  - StateSync/TurnSync: 同步数据结构                                │
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
3. **LogEntry 模式**：使用 TurnSync 包含 LogEntry 数组，客户端顺序渲染
4. **Metadata 契约**：LogEntry.Metadata 字段遵循 [doc/metadata/logentry.md](../metadata/logentry.md) 契约
5. **命名统一**：所有同步数据结构使用无后缀命名（StateSync、Player、Buff）

## 消息操作码

### Server → Client (1-99)

| OpCode | 名称 | 数据类型 | 说明 |
|--------|------|----------|------|
| 1 | `OpStateSync` | `StateSync` | 状态同步（进入新状态） |
| 2 | `OpTurnSync` | `TurnSync` | 回合内效果列表（使用LogEntry） |
| 3 | `OpDecisionRequest` | `Decision` | 决策请求 |
| 4 | `OpAvailable` | `Available` | 可用操作列表（道具/技能） |
| 5 | `OpMiniGameStart` | `MiniGameStart` | 小游戏开始 |
| 6 | `OpMiniGameResult` | `MiniGameResult` | 小游戏结果广播 |
| 7 | `OpGameOver` | `GameOver` | 游戏结束 |
| 8 | `OpFullSync` | `FullSync` | 完整同步（断线重连） |

### Client → Server (100+)

| OpCode | 名称 | 数据类型 | 说明 |
|--------|------|----------|------|
| 100 | `OpRollDice` | `RollDice` | 投骰子请求（服务器计算） |
| 101 | `OpUseItem` | `UseItem` | 使用道具 |
| 102 | `OpUseSkill` | `UseSkill` | 使用阵营技能 |
| 103 | `OpUserChoice` | `UserChoice` | 决策选择回复 |
| 104 | `OpMiniGameResultSubmit` | `MiniGameResultSubmit` | 小游戏排名提交 |

## 核心数据结构

### StateSync

完整游戏状态同步，用于客户端状态更新和断线重连：

```go
type StateSync struct {
    GlobalState string    `json:"global_state"`  // "turn_loop", "round_mini_game"
    TurnState   string    `json:"turn_state"`    // "main_action", "turn_moving"
    TurnPlayer  string    `json:"turn_player"`   // 当前回合玩家ID
    Round       int       `json:"round"`
    Turn        int       `json:"turn"`
    Paused      bool      `json:"paused"`        // 等待决策中
    Players     []Player  `json:"players"`
}
```

### TurnSync

回合内所有效果同步，使用 LogEntry 数组供客户端顺序渲染：

```go
type TurnSync struct {
    Round   int               `json:"round"`
    Turn    int               `json:"turn"`
    Player  string            `json:"player"`    // 回合玩家ID
    Entries []gamelog.LogEntry `json:"entries"`  // 效果列表（使用LogEntry）
}
```

**设计理由**：`TurnSync.Entries` 直接使用 `gamelog.LogEntry`，避免额外的 Action 展平结构。客户端根据 `LogEntry.ActionType` 和 `LogEntry.Metadata` 解析效果详情。

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
    UserID      string `json:"user_id"`
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

### MiniGameStart/MiniGameResult

小游戏相关：

```go
type MiniGameStart struct {
    GameType string   `json:"game_type"` // "dice_race"
    Players  []string `json:"players"`   // 参赛玩家ID列表
}

type MiniGameResult struct {
    Rankings []RankingEntry `json:"rankings"`
}

type RankingEntry struct {
    PlayerID string `json:"player_id"`
    Rank     int    `json:"rank"`
}
```

### FullSync

完整同步（断线重连）：

```go
type FullSync struct {
    State *StateSync `json:"state"`
    Turn  *TurnSync  `json:"turn"`
}
```

## BroadcastAdapter 接口

抽象广播接口，供 HSM 和 ActionContext 使用。接口使用 `interface{}` 参数以支持不同实现：

```go
type BroadcastAdapter interface {
    // 状态广播
    BroadcastStateSync(state interface{}) error
    BroadcastTurnSync(turn interface{}) error
    
    // 单播
    SendDecision(playerID string, decision interface{}) error
    SendAvailable(playerID string, available interface{}) error
    SendFullSync(playerID string, state, turn interface{}) error
    
    // 小游戏
    BroadcastMiniGameStart(start interface{}) error
    BroadcastMiniGameResult(result interface{}) error
    
    // 游戏结束
    BroadcastGameOver(over interface{}) error
}
```

**注意**：接口使用 `interface{}` 参数，实现方需要进行类型断言。

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

func (a *NakamaBroadcastAdapter) BroadcastStateSync(state interface{}) error {
    stateSync, ok := state.(*pkgnet.StateSync)
    if !ok {
        return nil // Invalid type, skip
    }
    data, _ := json.Marshal(stateSync)
    return a.handler.dispatcher.BroadcastMessage(int64(pkgnet.OpStateSync), data)
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