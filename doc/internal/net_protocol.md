# 客户端-服务器交互协议层设计

## 概述

本文档描述 Fated 游戏客户端-服务器交互协议层的设计与实现。基于 Nakama 权威服务器架构，提供状态同步、Action同步、决策请求等核心功能。

## 架构分层

```
┌──────────────────────────────────────────────────────────────────┐
│                    Nakama Match Handler Layer                      │
│  (实现 MatchHandler 接口，处理客户端连接、消息路由、状态广播)        │
├──────────────────────────────────────────────────────────────────┤
│                    pkg/net 协议层                                  │
│  - OpCode: 消息操作码                                              │
│  - Message: 基础消息结构                                           │
│  - StateSync/ActionSync: 同步数据结构                              │
│  - Decision: 决策请求结构                                          │
│  - MatchHandler: 抽象接口                                          │
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
2. **抽象接口设计**：`pkg/net.MatchHandler` 定义抽象接口，不引入 Nakama SDK 依赖
3. **复用现有组件**：利用 GameLog 的 JSON 序列化、Metadata 的 ToMap 方法
4. **命名统一**：所有同步数据结构使用无后缀命名（StateSync、Player、Buff）

## 消息操作码

### Server → Client (1-99)

| OpCode | 名称 | 数据类型 | 说明 |
|--------|------|----------|------|
| 1 | `OpStateSync` | `StateSync` | 状态同步（进入新状态） |
| 2 | `OpActionSync` | `ActionSync` | Action效果广播 |
| 3 | `OpDecisionRequest` | `Decision` | 决策请求 |
| 4 | `OpAvailable` | `Available` | 可用操作列表（道具/技能） |
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

### Player

玩家状态快照，Builder 从 `core.Player.Metadata` 提取已知键转为 typed 字段：

```go
type Player struct {
    UserID   string `json:"user_id"`
    Faction  string `json:"faction"`
    Position int    `json:"position"`
    HP       int    `json:"hp"`
    LP       int    `json:"lp"`
    Buffs    []Buff `json:"buffs"`
    Items    []Item `json:"items"`
    // Metadata已知键转为typed字段
    Charge      int  `json:"charge"`       // 阵营充能数（青龙/玄武）
    FireCounter int  `json:"fire_counter"` // 朱雀火计数
    IsDead      bool `json:"is_dead"`
    SkipTurn    bool `json:"skip_turn"`
}
```

### ActionSync

效果执行同步，复用 GameLog.LogEntry 结构：

```go
type ActionSync struct {
    ActionType string                 `json:"action_type"` // "damage", "move", "heal"
    Target     string                 `json:"target"`
    Delta      int                    `json:"delta"`       // HP/LP变化值
    Source     string                 `json:"source"`
    Metadata   map[string]interface{} `json:"metadata"`    // 复用 util.Metadata.ToMap()
}
```

### Available

可用操作列表，包含道具、阵营技能和骰子类型：

```go
type Available struct {
    Items       []Item `json:"items"`         // 可用道具
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

## 交互流程

### 回合主流程

```
【TurnUpkeep Enter】
→ Broadcast(OpStateSync, StateSync{global=turn_loop, turn=turn_upkeep})
→ execute BeforeTurn Buffs → Broadcast(OpActionSync, ActionSync{modify_lp/heal/damage})
→ 检查决策队列 → 若有: SendToPlayer(playerID, OpDecisionRequest, Decision)

【MainAction】
→ Broadcast(OpStateSync, StateSync{turn=main_action})
→ SendToPlayer(playerID, OpAvailable, Available{items, skill, dice_type})
→ 等待客户端 OpRollDice 或 OpUseItem

【Client: OpRollDice】
→ HandleMessage() → handleRollDice()
→ 使用 rng.DiceManager 根据玩家骰子类型计算
→ hsm.OnRollDice(steps, ctx)
→ Broadcast(OpActionSync, ActionSync{action_type="dice_roll", delta=steps})
→ 自动进入 TurnMoving

【TurnMoving Enter】
→ Broadcast(OpStateSync, StateSync{turn=turn_moving})
→ execute MoveAction → Broadcast(OpActionSync, ActionSync{move, path=[...]})
→ 路径效果: Broadcast(OpActionSync, ActionSync{fell_down, steal_buff...})
→ 若有决策(任意门): SendToPlayer(OpDecisionRequest)

【TurnLanded Enter】
→ Broadcast(OpStateSync, StateSync{turn=turn_landed})
→ execute OnLand effects → Broadcast(OpActionSync, ...)

【TurnEvent Enter】
→ Broadcast(OpStateSync, StateSync{turn=turn_event})
→ execute DrawEventAction → Broadcast(OpActionSync, ActionSync{draw_event})
→ execute event effects → Broadcast(OpActionSync, ActionSync{add_buff, damage...})

【TurnEnd Enter】
→ Broadcast(OpStateSync, StateSync{turn=turn_end})
→ execute AfterTurn Buffs → Broadcast(OpActionSync, ...)
→ tickBuffs → Broadcast(OpActionSync, ActionSync{remove_buff})
→ 结束回合 → 下一玩家
```

### Decision处理流程

```
【服务端生成Decision】
→ HSM检测到 ctx.Decisions 不为空
→ PushInterrupt(WaitDecision)
→ SendToPlayer(playerID, OpDecisionRequest, Decision)

【客户端等待】
→ 显示决策UI
→ 等待玩家选择或超时

【Client: OpUserChoice】
→ HandleMessage() → handleUserChoice(decisionID, choice)
→ hsm.OnUserChoice(choice, ctx)
→ PopInterrupt → 恢复原状态继续执行
→ Broadcast(OpActionSync, 决策结果效果)
```

### 断线重连流程

```
【玩家断线】
→ HandlePresenceLeave(userID)
→ 若当前回合是该玩家 → 决策超时自动处理（HSM已有30秒超时）

【玩家重连】
→ HandlePresenceJoin(userID, sessionID)
→ SendToPlayer(userID, OpFullSync, StateSync)
→ 发送当前回合的完整 GameLog（转换为 ActionSync 序列）
→ 若有等待该玩家的决策 → resendDecisionRequest()
```

## Builder 使用

Builder 负责将内部数据结构转换为 `pkg/net` 协议数据：

```go
import (
    internalnet "github.com/b1tAction/fated/internal/net"
    pkgnet "github.com/b1tAction/fated/pkg/net"
    "github.com/b1tAction/fated/pkg/rng"
)

// 创建构建器
builder := internalnet.NewBuilder(hsm, game)

// 构建状态同步
stateSync := builder.BuildStateSync()

// 构建玩家快照
players := builder.BuildPlayers()

// 构建Action同步（从GameLog）
entry := gamelog.LogEntry{...}
actionSync := builder.BuildActionSync(entry)

// 构建可用操作列表
builder.SetDiceType(rng.DiceTypeGold) // 使用 pkg/rng.DiceType 枚举
available := builder.BuildAvailable(player)
```

## 骰子计算

使用 `pkg/rng.DiceManager` 进行骰子计算（权威服务器）：

```go
import "github.com/b1tAction/fated/pkg/rng"

// 创建骰子管理器（传入游戏 RNG）
diceMgr := rng.NewDiceManager(game.RNG)

// 根据排名分配骰子类型
diceMgr.AssignDice(playerID, rank) // rank 1-4

// 滚骰子（返回 1-6，使用加权分布）
steps := diceMgr.RollSpecialDice(playerID)
```

## MatchHandler 接口

抽象接口供后续 Nakama 集成实现：

```go
type MatchHandler interface {
    // 生命周期
    MatchInit(ctx context.Context, config MatchConfig) error
    MatchLoop(ctx context.Context) error
    MatchStop(ctx context.Context) error

    // 消息处理
    HandleMessage(sender string, msg Message) error
    Broadcast(opCode OpCode, data interface{}) error
    SendToPlayer(playerID string, opCode OpCode, data interface{}) error

    // 玩家管理
    HandlePresenceJoin(userID string, sessionID string) error
    HandlePresenceLeave(userID string) error

    // 状态获取
    GetCurrentState() StateSync
}
```

## 相关文档

- [pkg/net/README.md](../../pkg/net/README.md) - 协议层包文档
- [internal/net/README.md](../../internal/net/README.md) - 构建层包文档
- [pkg/rng/README.md](../../pkg/rng/README.md) - RNG引擎和骰子类型
- [internal/engine/hsm/README.md](../engine/hsm/README.md) - HSM状态机
- [pkg/gamelog/README.md](../../pkg/gamelog/README.md) - 游戏日志系统