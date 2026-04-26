# Package net - 网络消息协议层

本包定义客户端 - 服务器交互的消息协议，用于 Paradiced 游戏的权威服务器通信。

## 设计目标

1. **状态同步**：向客户端广播当前游戏状态，包含增量 LogEntry 数据
2. **断线重连**：支持玩家重新连接后恢复游戏状态（含当前回合完整 LogEntry）
3. **决策请求**：等待玩家输入（投骰子、使用道具、选择选项）
4. **错误反馈**：通过 ActionRejected 返回标准化错误码

## 核心设计

### StateSync 包含增量 LogEntry

`StateSync` 通过 `Entries` 字段携带增量 `gamelog.LogEntry` 数据。每次 HSM 状态转换时，`BuildStateSync()` 自动获取自上次广播以来的新 LogEntry，客户端按顺序渲染动画。

**增量机制**：
- `GameLog.GetNewEntries()` 返回自上次 `MarkBroadcasted()` 以来新增的 LogEntry
- `Builder.BuildStateSync()` 调用 `GetNewEntries()` + `MarkBroadcasted()` 实现增量同步
- `Builder.BuildFullSyncStateSync()` 调用 `GetAllCurrentEntries()` 返回当前回合全部 LogEntry（用于断线重连）
- `Entries` 字段使用 `omitempty`，当无新 LogEntry 时 JSON 中不包含此字段

**设计理由**：
- StateSync 既承载状态信息又承载动画数据，减少客户端消息量
- 增量 Entries 避免重复发送已广播的 LogEntry
- FullSync 通过同一个 StateSync 结构携带完整回合数据，无需额外协议

### Metadata 字段契约

客户端根据 `action_type` 判断应解析哪些 metadata 字段：

| action_type | metadata fields |
|-------------|-----------------|
| `damage` | `hp_change: int`, `blocked_by?: string`, `piercing?: bool` |
| `heal` | `hp_change: int` |
| `modify_lp` | `lp_change: int` |
| `move` | `steps: int`, `start_pos: int`, `end_pos: int`, `path: []int` |
| `add_buff` | `buff_type: string`, `duration: int` |
| `remove_buff` | `buff_type: string` |
| `teleport` | `from_pos: int`, `to_pos: int` |
| `steal_buff` | `stolen_by: string`, `buff_type: string` |
| `respawn` | `checkpoint_pos: int` |
| `fell_down` | `position: int`, `hp_change: int` |
| `draw_event` | `event_type: string`, `event_name: string` |
| `dice_roll` | `dice_type: string`, `dice_steps: int` |
| `boss_damage` | `damage: int`, `is_crit: bool`, `boss_remaining_hp: int` |
| `boss_attack` | `attack_type: string`, `damage: int`, `target: string` |
| `boss_skill` | `skill_type: string`, `targets: string` |
| `state` | `from: string`, `to: string` |

完整契约见：[doc/metadata/logentry.md](../../doc/metadata/logentry.md)

## 架构定位

```
┌──────────────────────────────────────────────────────────────────┐
│                    Nakama Match Handler Layer                      │
│  (实现 BroadcastAdapter 接口，处理客户端连接、消息路由、状态广播)    │
├──────────────────────────────────────────────────────────────────┤
│                    pkg/net 协议层 (本包)                           │
│  - OpCode: 消息操作码                                              │
│  - Message: 基础消息结构                                           │
│  - StateSync: 同步数据结构（含增量 LogEntry）                           │
│  - Decision: 决策请求结构                                          │
│  - BroadcastAdapter: 广播抽象接口                                  │
│  - ActionRejected: 动作拒绝（带错误码）                            │
├──────────────────────────────────────────────────────────────────┤
│                    internal/net 构建层                             │
│  - Builder: 将内部数据转换为协议数据                                │
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
| `sync.go` | 状态同步数据结构：`StateSync`, `Player`, `ActionRejected` |
| `decision.go` | 决策请求/回复结构：`Decision`, `Option`, `RollDice`, `UseItem`, `UserChoice` |
| `broadcast.go` | 广播抽象接口 `BroadcastAdapter` 和测试实现 `MockBroadcastAdapter` |

## 消息操作码

### Server → Client (1-99)

| OpCode | 名称 | 数据类型 | 说明 |
|--------|------|----------|------|
| 1 | `OpStateSync` | `StateSync` | 状态同步（含增量 LogEntry） |
| 3 | `OpDecisionRequest` | `Decision` | 决策请求 |
| 4 | `OpAvailable` | `Available` | 可用操作列表 |
| 5 | `OpMiniGameStart` | `MiniGameStart` | 小游戏开始 |
| 6 | `OpMiniGameResult` | `MiniGameResult` | 小游戏结果广播 |
| 7 | `OpGameOver` | `GameOver` | 游戏结束 |
| 8 | `OpFullSync` | `StateSync` | 完整同步（断线重连，含当前回合全部 LogEntry） |
| 9 | `OpActionRejected` | `ActionRejected` | 动作拒绝（带错误码） |
| 10 | `OpWaitingSync` | `StateSync` | 等待同步（断线重连等待期间） |
| 11 | `OpStartGameAck` | `StartGameAck` | 游戏开始确认（含地图配置） |

### Client → Server (100+)

| OpCode | 名称 | 数据类型 | 说明 |
|--------|------|----------|------|
| 100 | `OpRollDice` | `RollDice` | 投骰子请求（服务器计算） |
| 101 | `OpUseItem` | `UseItem` | 使用道具 |
| 102 | `OpUseSkill` | `UseSkill` | 使用阵营技能 |
| 103 | `OpUserChoice` | `UserChoice` | 决策选择回复 |
| 105 | `OpStartGame` | `StartGame` | 房主开始游戏 |
| 106 | `OpRoundReady` | `RoundReady` | 回合就绪确认 |
| 107 | `OpMiniGameDataSubmit` | `MiniGameDataSubmit` | 小游戏数据提交（服务器计算排名） |

## 核心数据结构

### StateSync

完整游戏状态同步，用于客户端状态更新、动画渲染和断线重连：

```go
type StateSync struct {
    GlobalState     string    `json:"global_state"`  // "turn_loop", "round_mini_game"
    TurnState       string    `json:"turn_state"`    // "main_action", "turn_moving"
    CurrentPlayerID string    `json:"current_player_id"`   // 当前回合玩家 ID
    Round           int       `json:"round"`
    Turn            int       `json:"turn"`
    Paused          bool      `json:"paused"`        // 等待决策中
    Players         []Player  `json:"players"`
    Map             MapInfo   `json:"map"`
    Entries         []gamelog.LogEntry `json:"entries,omitempty"` // 增量 LogEntry（omitempty: nil/空时不包含）
}
```

**Entries 增量机制**：
- 每次 `BuildStateSync()` 获取自上次广播以来的新 LogEntry
- 调用后自动 `MarkBroadcasted()`，下次广播只返回新增部分
- `omitempty` 确保无新 LogEntry 时 JSON 不包含空数组
- 断线重连使用 `BuildFullSyncStateSync()` 获取当前回合全部 LogEntry

### Player

玩家状态快照：

```go
type Player struct {
    PlayerID    string `json:"player_id"`   // 玩家游戏内部 ID（直接等于 Nakama userID）
    DisplayName string `json:"display_name"` // 用户显示名称（Boss: "Boss"，其他: fallback PlayerID）
    Faction     string `json:"faction"`      // snake_case: "qing_long", "zhu_que"
    Position    int    `json:"position"`
    HP          int    `json:"hp"`
    LP          int    `json:"lp"`
    Buffs       []Buff `json:"buffs"`        // 带 Name 的 Buff 列表
    Items       []Item `json:"items"`        // 带 Name 的 Item 列表
    Charge      int    `json:"charge"`       // 阵营充能数（青龙/玄武）
    FireCounter int    `json:"fire_counter"` // 朱雀火计数
    IsDead      bool   `json:"is_dead"`
    SkipTurn    bool   `json:"skip_turn"`
    IsBoss      bool   `json:"is_boss,omitempty"` // Boss特殊玩家标识（仅Boss为true）
}
```

### Buff/Item（带显示名）

```go
type Buff struct {
    Type     string `json:"type"`     // "divine", "curse"
    Name     string `json:"name"`     // "神眷", "诅咒"（从定义获取）
    Duration int    `json:"duration"` // -1 表示永久
}

type Item struct {
    ID   string `json:"id"`   // Item 实例 UUID
    Type string `json:"type"` // "any_door", "reverse_clock"
    Name string `json:"name"` // "任意门", "反方向的钟"
}
```

### Decision

决策请求，带 Context 标识来源：

```go
type Decision struct {
    ID      string   `json:"id"`       // 决策 ID
    Prompt  string   `json:"prompt"`   // 提示文本
    Context string   `json:"context"`  // 来源标识："Item_AnyDoor", "Buff_Divine"
    Options []Option `json:"options"`  // 选项列表
    Timeout int      `json:"timeout"`  // 超时秒数
    Default int      `json:"default"`  // 默认选项索引
}

type Option struct {
    ID     string `json:"id"`             // 选项 ID
    Label  string `json:"label"`          // 显示文本
    Effect string `json:"effect,omitempty"` // 效果描述
}
```

### Available

可用操作列表：

```go
type Available struct {
    Items       []Item `json:"items"`         // 可用道具（带 Name）
    CanUseSkill bool   `json:"can_use_skill"` // 阵营技能可用
    DiceType    string `json:"dice_type"`     // "gold", "silver", "copper", "wood"
}
```

### MiniGameStart/MiniGameResult

小游戏相关：

```go
type MiniGameStart struct {
    GameType string   `json:"game_type"` // "dice_race"
    Players  []string `json:"players"`   // 参赛玩家 ID 列表
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

完整同步（断线重连）直接使用 `StateSync` 结构，通过 `BuildFullSyncStateSync()` 获取包含当前回合全部 LogEntry 的 StateSync：

```go
// OpFullSync 发送包含全部当前回合 LogEntry 的 StateSync
stateSync := builder.BuildFullSyncStateSync()
broadcastAdapter.SendFullSync(playerID, stateSync)
```

### ActionRejected

动作拒绝通知（新增 ErrorCode 字段）：

```go
type ActionRejected struct {
    OpCode    OpCode              `json:"op_code"`
    ErrorCode constants.ErrorCode `json:"error_code"` // 错误码
    Reason    string              `json:"reason"`     // 拒绝原因
    Message   string              `json:"message"`    // 人类可读消息
}
```

**错误码分类**：

| 范围 | 分类 |
|------|------|
| `0` | 成功 (ErrOK) |
| `1001-1999` | 验证错误 (Validation Errors) |
| `2001-2999` | 游戏逻辑错误 (Game Logic Errors) |
| `3001-3999` | 系统错误 (System Errors) |
| `4001-4999` | 未找到错误 (Not Found Errors) |

常见错误码：
- `ErrNotCurrentTurn` (1004): 非当前回合玩家
- `ErrInvalidState` (1002): 无效状态
- `ErrPlayerNotFound` (4001): 玩家未找到
- `ErrItemNotFound` (4002): 道具未找到
- `ErrInvalidParameter` (1001): 无效参数
- `ErrConditionNotMet` (1005): 条件未满足

详见 [pkg/constants/README.md](../constants/README.md#ErrorCode---错误码系统)。

## BroadcastAdapter 接口

抽象广播接口，供 HSM 和 ActionContext 使用：

```go
type BroadcastAdapter interface {
    BroadcastStateSync(stateSync *StateSync) error
    SendDecision(playerID string, decision *Decision) error
    SendAvailable(playerID string, available *Available) error
    BroadcastMiniGameStart(start *MiniGameStart) error
    BroadcastMiniGameResult(result *MiniGameResult) error
    BroadcastGameOver(over *GameOver) error
    SendFullSync(playerID string, state *StateSync) error
    BroadcastStartGameAck(ack *StartGameAck) error
}
```

**注意**：`BroadcastTurnSync` 和 `TurnSync` 已移除，LogEntry 数据现在通过 `StateSync.Entries` 增量携带。

### MockBroadcastAdapter

测试实现，捕获所有广播调用：

```go
mock := NewMockBroadcastAdapter()
mock.BroadcastStateSync(stateSync)

// 检查捕获的消息
if mock.StateSyncs[0].GlobalState == "turn_loop" { ... }
if len(mock.StateSyncs[0].Entries) > 0 { ... }

mock.Clear() // 清空所有捕获的消息
```

## 使用示例

### 创建消息

```go
import (
    "github.com/b1tAction/paradiced/pkg/net"
    "github.com/b1tAction/paradiced/pkg/gamelog"
)

// 创建状态同步消息（含增量 LogEntry）
stateSync := &net.StateSync{
    GlobalState:     "turn_loop",
    TurnState:       "turn_landed",
    CurrentPlayerID: "player-001",
    Round:           1,
    Turn:            0,
    Players:         []net.Player{...},
    Entries: []gamelog.LogEntry{
        {
            ActionType: "damage",
            Target:     "player-001",
            Source:     "Cell_Fragile",
        },
        {
            ActionType: "move",
            Target:     "player-001",
            Metadata:   metadata.WithInt("steps", 3),
        },
    },
}
msg, err := net.NewMessage(net.OpStateSync, stateSync)
```

### 解析消息

```go
// 解析消息数据
var stateSync net.StateSync
err := msg.ParseData(&stateSync)

// 遍历 Entries 渲染动画
for _, entry := range stateSync.Entries {
    switch entry.ActionType {
    case "damage":
        // 播放伤害动画，使用 entry.Metadata 中的 hp_change
    case "move":
        // 播放移动动画，使用 entry.Metadata 中的 path
    }
}
```

## 命名规范

所有同步数据结构使用**无后缀命名**：

- `StateSync`（不是 `StateSyncData`）
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

- [internal/net/README.md](../../internal/net/README.md) - 构建器
- [pkg/protocol/README.md](../protocol/README.md) - 协议接口层
- [doc/internal/net_protocol.md](../../doc/internal/net_protocol.md) - 协议层完整设计
- [internal/engine/hsm/README.md](../../internal/engine/hsm/README.md) - HSM 状态机
- [doc/internal/nakama.md](../../doc/internal/nakama.md) - Nakama Match Handler 集成
- [internal/nakama/README.md](../../internal/nakama/README.md) - Nakama 包文档
- [pkg/gamelog/README.md](../gamelog/README.md) - GameLog 系统
- [doc/metadata/logentry.md](../../doc/metadata/logentry.md) - LogEntry.Metadata 契约

## Builder 接口

定义在 `pkg/net/builder.go`，用于构建协议同步消息。接口设计避免 internal/engine/hsm 和 internal/net 之间的循环引用：

```go
type Builder interface {
    BuildStateSync() *StateSync          // 增量 Entries（每次广播后自动 MarkBroadcasted）
    BuildFullSyncStateSync() *StateSync  // 全量当前回合 Entries（断线重连用，不 MarkBroadcasted）
    BuildAvailable() *Available
    SetDiceType(diceType string)
}
```

实现在 `internal/net/builder.go`：

```go
type Builder struct {
    hsm         *hsm.HSM
    turnDiceType rng.DiceType
}

func NewBuilder(hsmInstance *hsm.HSM) *Builder
func (b *Builder) BuildStateSync() *StateSync                   // 增量 Entries
func (b *Builder) BuildFullSyncStateSync() *StateSync           // 全量 Entries（断线重连）
func (b *Builder) BuildAvailable() *Available
func (b *Builder) SetDiceType(diceType string)                  // pkg/net.Builder interface
func (b *Builder) SetDiceTypeFromRng(diceType rng.DiceType)     // internal use
func (b *Builder) BuildAvailableForPlayer(player *core.Player) *Available  // specific player
```

HSM 状态通过 `StateContext.WithBuilder(builder)` 获取 Builder，用于广播消息。

## Nakama 集成实现

`internal/nakama.NakamaBroadcastAdapter` 实现本包的 `BroadcastAdapter` 接口：

```go
// internal/nakama/broadcast.go
type NakamaBroadcastAdapter struct {
    handler *NakamaMatchHandler
}

func (a *NakamaBroadcastAdapter) BroadcastStateSync(state *StateSync) error {
    data, err := json.Marshal(state)
    if err != nil {
        return err
    }
    return a.handler.dispatcher.BroadcastMessage(int64(OpStateSync), data)
}
```

使用 `DispatcherAdapter` 接口隔离 Nakama SDK 依赖，支持无真实服务器的测试。
