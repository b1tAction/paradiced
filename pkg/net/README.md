# Package net - 网络消息协议层

本包定义客户端-服务器交互的消息协议，用于 Paradiced 游戏的权威服务器通信。

## 设计目标

1. **状态同步**：向客户端广播当前游戏状态
2. **回合同步**：直接使用 GameLog 的 LogEntry 列表（无需转换）
3. **决策请求**：等待玩家输入（投骰子、使用道具、选择选项）
4. **断线重连**：支持玩家重新连接后恢复游戏状态

## 核心设计

### TurnSync 直接使用 LogEntry

```go
// TurnSync 直接包含 gamelog.LogEntry 列表（无需转换为 Action）
type TurnSync struct {
    Round   int                `json:"round"`
    Turn    int                `json:"turn"`
    Player  string             `json:"player"`
    Entries []gamelog.LogEntry `json:"entries"` // 直接发送 GameLog
}
```

**设计理由**：
- GameLog 是权威数据源，无需二次转换
- Metadata 字段契约文档化（见 `doc/internal/metadata.md`）
- 新增 ActionType 无需更新协议层转换逻辑

### Metadata 字段契约

客户端根据 `action_type` 判断应解析哪些 metadata 字段：

| action_type | metadata fields |
|-------------|-----------------|
| `damage` | `blocked_by?: string`, `piercing?: bool` |
| `move` | `path: []int`, `dice_steps: int`, `dice_type: string` |
| `add_buff` | `buff_type: string` |
| `draw_event` | `event_type: string`, `event_name: string` |

完整契约见：[doc/internal/metadata.md](../../doc/internal/metadata.md)

## 架构定位

```
┌──────────────────────────────────────────────────────────────────┐
│                    Nakama Match Handler Layer                      │
│  (实现 BroadcastAdapter 接口，处理客户端连接、消息路由、状态广播)    │
├──────────────────────────────────────────────────────────────────┤
│                    pkg/net 协议层 (本包)                           │
│  - OpCode: 消息操作码                                              │
│  - Message: 基础消息结构                                           │
│  - StateSync/TurnSync: 同步数据结构                                │
│  - Decision: 决策请求结构                                          │
│  - BroadcastAdapter: 广播抽象接口                                  │
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
| `sync.go` | 状态同步数据结构：`StateSync`, `Player`, `TurnSync`, `Action`, `Available` |
| `decision.go` | 决策请求/回复结构：`Decision`, `Option`, `RollDice`, `UseItem`, `UserChoice` |
| `broadcast.go` | 广播抽象接口 `BroadcastAdapter` 和测试实现 `MockBroadcastAdapter` |

## 消息操作码

### Server → Client (1-99)

| OpCode | 名称 | 数据类型 | 说明 |
|--------|------|----------|------|
| 1 | `OpStateSync` | `StateSync` | 状态同步（进入新状态） |
| 2 | `OpTurnSync` | `TurnSync` | 回合内Action效果列表 |
| 3 | `OpDecisionRequest` | `Decision` | 决策请求 |
| 4 | `OpAvailable` | `Available` | 可用操作列表 |
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

客户端循环遍历 Actions 数组，按顺序播放每个效果的动画。

### Action

单个效果，所有字段展平便于客户端渲染：

```go
type Action struct {
    Type   string `json:"type"`            // "damage", "move", "heal", "add_buff"
    Target string `json:"target"`          // 目标玩家ID
    Source string `json:"source"`          // 来源（Buff/Item/Event名称）

    // Type-specific fields (omitempty)
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

### Player

玩家状态快照：

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

### Available

可用操作列表：

```go
type Available struct {
    Items       []Item `json:"items"`         // 可用道具（带Name）
    CanUseSkill bool   `json:"can_use_skill"` // 阵营技能可用
    DiceType    string `json:"dice_type"`     // "gold", "silver", "copper", "wood"
}
```

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
    BroadcastStateSync(stateSync *StateSync) error
    BroadcastTurnSync(turnSync *TurnSync) error
    SendDecision(playerID string, decision *Decision) error
    SendAvailable(playerID string, available *Available) error
    BroadcastMiniGameStart(start *MiniGameStart) error
    BroadcastMiniGameResult(result *MiniGameResult) error
    BroadcastGameOver(over *GameOver) error
    SendFullSync(playerID string, state *StateSync, turn *TurnSync) error
}
```

### MockBroadcastAdapter

测试实现，捕获所有广播调用：

```go
mock := NewMockBroadcastAdapter()
mock.BroadcastStateSync(stateSync)
mock.BroadcastTurnSync(turnSync)

// 检查捕获的消息
if mock.StateSyncs[0].GlobalState == "turn_loop" { ... }
if mock.TurnSyncs[0].Actions[0].Type == "damage" { ... }

mock.Clear() // 清空所有捕获的消息
```

## 使用示例

### 创建消息

```go
import "pkg/net"

// 创建回合同步消息
turnSync := &net.TurnSync{
    Round:   1,
    Turn:    0,
    Player:  "player-001",
    Actions: []net.Action{
        {Type: "damage", Target: "player-001", Delta: -1, Source: "Cell_Fragile"},
        {Type: "move", Target: "player-001", Path: []int{10, 11, 12}, Source: "DiceRoll"},
    },
}
msg, err := net.NewMessage(net.OpTurnSync, turnSync)
```

### 解析消息

```go
// 解析消息数据
var turnSync net.TurnSync
err := msg.ParseData(&turnSync)

// 遍历Actions渲染
for _, action := range turnSync.Actions {
    switch action.Type {
    case "damage":
        // 播放伤害动画，使用action.Delta和action.Source
    case "move":
        // 播放移动动画，使用action.Path
    }
}
```

## 命名规范

所有同步数据结构使用**无后缀命名**：

- `StateSync`（不是 `StateSyncData`）
- `TurnSync`（不是 `TurnSyncData`）
- `Player`（不是 `PlayerSync`）
- `Buff`（不是 `BuffSync`）
- `Item`（不是 `ItemSync`）

Faction 字段使用 **snake_case** 值：

- `"qing_long"`（青龙）
- `"zhu_que"`（朱雀）
- `"bai_hu"`（白虎）
- `"xuan_wu"`（玄武）

## 与 Nakama 集成

后续集成 Nakama 时：

1. 实现 `BroadcastAdapter` 接口
2. 在 `MatchInit` 中初始化 HSM 和 Game
3. 在 `HandleMessage` 中调用 `hsm.OnRollDice/OnUseItem/OnUserChoice`
4. 使用 `internal/net.Builder` 构建同步数据

## 相关文档

- [internal/net/README.md](../../internal/net/README.md) - 构建器和骰子计算器
- [pkg/protocol/README.md](../protocol/README.md) - 协议接口层
- [doc/internal/net_protocol.md](../../doc/internal/net_protocol.md) - 协议层完整设计
- [internal/engine/hsm/README.md](../../internal/engine/hsm/README.md) - HSM 状态机