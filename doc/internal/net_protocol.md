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
3. **Action列表模式**：使用 TurnSync 包含 Action 数组，客户端顺序渲染
4. **字段展平**：Action 字段展平便于客户端直接使用，不嵌套 Metadata
5. **命名统一**：所有同步数据结构使用无后缀命名（StateSync、Player、Buff）

## 消息操作码

### Server → Client (1-99)

| OpCode | 名称 | 数据类型 | 说明 |
|--------|------|----------|------|
| 1 | `OpStateSync` | `StateSync` | 状态同步（进入新状态） |
| 2 | `OpTurnSync` | `TurnSync` | 回合内Action效果列表 |
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

回合内所有效果同步，使用Action列表供客户端顺序渲染：

```go
type TurnSync struct {
    Round   int      `json:"round"`
    Turn    int      `json:"turn"`
    Player  string   `json:"player"`    // 回合玩家ID
    Actions []Action `json:"actions"`   // 效果列表（客户端顺序播放）
}
```

**设计理由**：客户端循环遍历 Actions 数组，按顺序播放每个效果的动画。展平字段便于直接使用，无需解析嵌套 Metadata。

### Action

单个效果，所有字段展平便于客户端渲染：

```go
type Action struct {
    Type   string `json:"type"`            // "damage", "move", "heal", "add_buff"
    Target string `json:"target"`          // 目标玩家ID
    Source string `json:"source"`          // 来源（Buff/Item/Event名称）

    // Type-specific fields (omitempty - 客户端判断非空即可使用)
    Delta       int    `json:"delta,omitempty"`       // HP/LP变化值
    Path        []int  `json:"path,omitempty"`        // 移动路径
    BuffType    string `json:"buff_type,omitempty"`   // Buff类型
    Duration    int    `json:"duration,omitempty"`    // Buff持续时间
    EventType   string `json:"event_type,omitempty"`  // 事件类型
    EventName   string `json:"event_name,omitempty"`  // 事件显示名
    Position    int    `json:"position,omitempty"`    // 位置（teleport/respawn）
    StolenFrom  string `json:"stolen_from,omitempty"` // 被偷玩家（白虎劫运）
    StolenBuff  string `json:"stolen_buff,omitempty"` // 被偷Buff类型
    DiceType    string `json:"dice_type,omitempty"`   // 骰子类型
    DiceSteps   int    `json:"dice_steps,omitempty"`  // 骰子步数
    FallDamage  int    `json:"fall_damage,omitempty"` // 落坑伤害
    FromState   string `json:"from_state,omitempty"`  // 状态转换：原状态
    ToState     string `json:"to_state,omitempty"`    // 状态转换：新状态
}
```

客户端渲染逻辑：

```javascript
for (const action of turnSync.actions) {
    switch (action.type) {
        case "damage":
            playDamageAnimation(action.target, action.delta, action.source);
            break;
        case "move":
            playMoveAnimation(action.target, action.path);
            break;
        case "add_buff":
            playBuffAnimation(action.target, action.buff_type, action.duration);
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

抽象广播接口，供 HSM 和 ActionContext 使用：

```go
type BroadcastAdapter interface {
    // 状态广播
    BroadcastStateSync(stateSync *StateSync) error
    BroadcastTurnSync(turnSync *TurnSync) error
    
    // 单播
    SendDecision(playerID string, decision *Decision) error
    SendAvailable(playerID string, available *Available) error
    SendFullSync(playerID string, state *StateSync, turn *TurnSync) error
    
    // 小游戏
    BroadcastMiniGameStart(start *MiniGameStart) error
    BroadcastMiniGameResult(result *MiniGameResult) error
    
    // 游戏结束
    BroadcastGameOver(over *GameOver) error
}
```

### MockBroadcastAdapter

测试实现，捕获所有广播调用：

```go
type MockBroadcastAdapter struct {
    StateSyncs     []*StateSync
    TurnSyncs      []*TurnSync
    Decisions      map[string]*Decision
    Availables     map[string]*Available
    MiniGameStarts []*MiniGameStart
    MiniGameResults []*MiniGameResult
    GameOvers      []*GameOver
    FullSyncs      map[string]*FullSync
}

// 使用示例
mock := NewMockBroadcastAdapter()
mock.BroadcastStateSync(stateSync)
mock.BroadcastTurnSync(turnSync)

// 检查捕获的消息
if mock.StateSyncs[0].GlobalState == "turn_loop" { ... }
if mock.TurnSyncs[0].Actions[0].Type == "damage" { ... }

mock.Clear() // 清空所有捕获的消息
```

## 交互流程

### 回合主流程

```
【TurnUpkeep Enter】
→ Broadcast(OpStateSync, StateSync{global=turn_loop, turn=turn_upkeep})
→ execute BeforeTurn Buffs → GameLog记录
→ 回合结束时 Broadcast(OpTurnSync, TurnSync{actions=[modify_lp/heal/damage]})
→ 检查决策队列 → 若有: SendToPlayer(playerID, OpDecisionRequest, Decision)

【MainAction】
→ Broadcast(OpStateSync, StateSync{turn=main_action})
→ SendToPlayer(playerID, OpAvailable, Available{items, skill, dice_type})
→ 等待客户端 OpRollDice 或 OpUseItem

【Client: OpRollDice】
→ HandleMessage() → handleRollDice()
→ 使用 rng.DiceManager 根据玩家骰子类型计算
→ hsm.OnRollDice(steps, ctx)
→ GameLog记录 dice_roll
→ 自动进入 TurnMoving

【TurnMoving Enter】
→ Broadcast(OpStateSync, StateSync{turn=turn_moving})
→ execute MoveAction → GameLog记录 move
→ 路径效果: GameLog记录 fell_down, steal_buff...
→ 若有决策(任意门): SendToPlayer(OpDecisionRequest)

【TurnLanded Enter】
→ Broadcast(OpStateSync, StateSync{turn=turn_landed})
→ execute OnLand effects → GameLog记录

【TurnEvent Enter】
→ Broadcast(OpStateSync, StateSync{turn=turn_event})
→ execute DrawEventAction → GameLog记录 draw_event
→ execute event effects → GameLog记录 add_buff, damage...

【TurnEnd Enter】
→ Broadcast(OpStateSync, StateSync{turn=turn_end})
→ execute AfterTurn Buffs → GameLog记录
→ tickBuffs → GameLog记录 remove_buff
→ Broadcast(OpTurnSync, TurnSync{actions=[本回合所有效果]})
→ 结束回合 → 下一玩家
```

### Decision处理流程

```
【服务端生成Decision】
→ HSM检测到 ctx.Decisions 不为空
→ PushInterrupt(WaitDecision)
→ SendToPlayer(playerID, OpDecisionRequest, Decision)

【客户端等待】
→ 显示决策UI（根据Context区分来源）
→ 等待玩家选择或超时

【Client: OpUserChoice】
→ HandleMessage() → handleUserChoice(decisionID, choice)
→ hsm.OnUserChoice(choice, ctx)
→ PopInterrupt → 恢复原状态继续执行
→ GameLog记录决策结果效果
```

### 断线重连流程

```
【玩家断线】
→ HandlePresenceLeave(userID)
→ 若当前回合是该玩家 → 决策超时自动处理（HSM已有30秒超时）

【玩家重连】
→ HandlePresenceJoin(userID, sessionID)
→ SendToPlayer(userID, OpFullSync, FullSync{state, turn})
→ 发送当前回合的完整 GameLog（转换为 TurnSync）
→ 若有等待该玩家的决策 → resendDecisionRequest()
```

## Builder 使用

Builder 负责将内部数据结构转换为 `pkg/net` 协议数据：

```go
import (
    internalnet "github.com/b1tAction/paradiced/internal/net"
    pkgnet "github.com/b1tAction/paradiced/pkg/net"
    "github.com/b1tAction/paradiced/pkg/rng"
)

// 创建构建器
builder := internalnet.NewBuilder(hsm, game)

// 构建状态同步
stateSync := builder.BuildStateSync()

// 构建回合同步（从GameLog提取所有Action）
turnSync := builder.BuildTurnSync()

// 构建玩家快照
players := builder.BuildPlayers()

// 构建可用操作列表
builder.SetDiceType(rng.DiceTypeGold) // 使用 pkg/rng.DiceType 枚举
available := builder.BuildAvailable(player)

// 构建完整同步（断线重连）
stateSync, turnSync := builder.BuildFullSync()
```

### buildAction 字段映射

| ActionType | 提取字段 | 说明 |
|------------|----------|------|
| `damage`/`heal`/`modify_lp` | `delta` | HP/LP变化值 |
| `move` | `path`, `dice_steps`, `dice_type` | 移动路径和骰子信息 |
| `add_buff` | `buff_type`, `duration` | 添加Buff |
| `remove_buff` | `buff_type` | 移除Buff |
| `draw_event` | `event_type`, `event_name` | 抽取事件 |
| `teleport` | `position` | 传送位置 |
| `steal_buff` | `stolen_buff`, `stolen_from` | 白虎劫运 |
| `fell_down` | `position`, `fall_damage` | 落坑 |
| `respawn` | `position` | 重生位置 |
| `dice_roll` | `dice_type`, `dice_steps` | 投骰子 |
| `state` | `from_state`, `to_state` | 状态转换 |

## Faction SnakeCase 转换

`protocol.Faction` 使用 `SnakeCase()` 方法获取 snake_case 值用于 JSON：

```go
// protocol/player.go
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

实现 `BroadcastAdapter` 接口：

```go
type NakamaBroadcastAdapter struct {
    matchID   string
    nakama    *nakama.Server
}

func (a *NakamaBroadcastAdapter) BroadcastStateSync(stateSync *pkgnet.StateSync) error {
    msg := pkgnet.MustNewMessage(pkgnet.OpStateSync, stateSync)
    return a.nakama.BroadcastMessage(a.matchID, msg)
}

func (a *NakamaBroadcastAdapter) BroadcastTurnSync(turnSync *pkgnet.TurnSync) error {
    msg := pkgnet.MustNewMessage(pkgnet.OpTurnSync, turnSync)
    return a.nakama.BroadcastMessage(a.matchID, msg)
}

func (a *NakamaBroadcastAdapter) SendDecision(playerID string, decision *pkgnet.Decision) error {
    msg := pkgnet.MustNewMessage(pkgnet.OpDecisionRequest, decision)
    return a.nakama.SendToPlayer(a.matchID, playerID, msg)
}

// ... 其他方法实现
```

## 相关文档

- [pkg/net/README.md](../../pkg/net/README.md) - 协议层包文档
- [internal/net/README.md](../../internal/net/README.md) - 构建层包文档
- [pkg/rng/README.md](../../pkg/rng/README.md) - RNG引擎和骰子类型
- [internal/engine/hsm/README.md](../engine/hsm/README.md) - HSM状态机
- [pkg/gamelog/README.md](../../pkg/gamelog/README.md) - 游戏日志系统