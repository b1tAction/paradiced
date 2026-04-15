# pkg/gamelog - Game Log System

游戏日志系统，用于 Client 回放动画。

## 概述

GameLog 提供统一的日志记录机制，所有游戏效果（Action 执行、状态转换等）都记录到全局日志中，供 Client 播放动画。

## 核心结构

### EntryType

日志条目类型枚举：

```go
type EntryType string

const (
    EntryTypeAction    EntryType = "action"     // Action 执行
    EntryTypeState     EntryType = "state"      // HSM 状态转换
    EntryTypeMiniGame  EntryType = "mini_game"  // 小游戏结果
    EntryTypeBoss      EntryType = "boss"       // Boss 战斗
    EntryTypeDecision  EntryType = "decision"   // 用户决策
)
```

### LogEntry

单条日志条目，使用 `util.Metadata` 存储元数据：

```go
type LogEntry struct {
    Timestamp  time.Time        `json:"timestamp"`
    Type       EntryType        `json:"type"`
    ActionType string           `json:"action_type,omitempty"`  // snake_case
    Target     string           `json:"target,omitempty"`
    Delta      int              `json:"delta,omitempty"`
    Source     string           `json:"source,omitempty"`
    Metadata   *util.Metadata   `json:"metadata,omitempty"`
}
```

### TurnSegment

回合分段，按回合存储日志条目：

```go
type TurnSegment struct {
    Round     int         `json:"round"`
    Turn      int         `json:"turn"`
    PlayerID  string      `json:"player_id"`
    Entries   []LogEntry  `json:"entries"`
    StartTime time.Time   `json:"start_time"`
    EndTime   time.Time   `json:"end_time,omitempty"`
}
```

### GameLog

全局日志管理器：

```go
type GameLog struct {
    segments []*TurnSegment
    current  *TurnSegment  // 当前回合分段
}
```

## 关键方法

```go
// 创建日志
log := gamelog.NewGameLog()

// 开始回合
log.StartTurn(round, turn, playerID)

// 添加日志条目（由 Action 系统自动调用）
log.AddEntry(entry)

// 结束回合
log.EndTurn()

// 获取分段
segments := log.GetTurnSegments()

// JSON 序列化
json, err := log.ToJSON()

// 快捷方法
log.LogStateTransition(from, to, playerID)
```

## 使用示例

### 在 HSM 中使用

```go
// TurnUpkeepState.Enter() - 开始回合日志
func (s *TurnUpkeepState) Enter(ctx *StateContext) {
    // 开始回合
    ctx.Game.Log.StartTurn(ctx.Game.State.Round, ctx.Game.State.Turn, player.UserID)
    
    // 使用 Action 系统（自动记录日志）
    respawnAction := engineaction.NewRespawnAction(player, checkpoint, "DeathRespawn")
    s.actionCtx.ExecuteAction(respawnAction)
}

// TurnEndState.Enter() - 结束回合日志
func (s *TurnEndState) Enter(ctx *StateContext) {
    // 结束回合
    ctx.Game.Log.EndTurn()
}
```

### 在 Action 中使用

ActionContext 自动将日志写入全局 GameLog：

```go
func (ctx *ActionContext) ExecuteAction(action ExecutableAction) error {
    // ... 执行 Action ...
    
    // 自动记录到全局日志
    if ctx.Game != nil {
        ctx.Game.GetGameLog().AddEntry(action.LogEntry())
    }
    
    return nil
}
```

## JSON 输出格式

```json
{
  "segments": [
    {
      "round": 1,
      "turn": 0,
      "player_id": "player1",
      "start_time": "2026-04-14T10:00:00Z",
      "end_time": "2026-04-14T10:00:30Z",
      "entries": [
        {
          "timestamp": "2026-04-14T10:00:05Z",
          "type": "action",
          "action_type": "modify_lp",
          "target": "player1",
          "delta": 1,
          "source": "Buff_Divine"
        },
        {
          "timestamp": "2026-04-14T10:00:10Z",
          "type": "action",
          "action_type": "respawn",
          "target": "player1",
          "source": "DeathRespawn",
          "metadata": {"checkpoint_pos": 50}
        },
        {
          "timestamp": "2026-04-14T10:00:20Z",
          "type": "state",
          "target": "player1",
          "metadata": {"from": "TurnUpkeep", "to": "MainAction"}
        }
      ]
    }
  ]
}
```

## 设计原则

1. **单一日志源** - Game 持有唯一的 GameLog 实例
2. **分段存储** - 按 Round/Turn 分段，便于 Client 按回合回放
3. **Action 集成** - 所有游戏效果通过 Action 系统，自动生成日志
4. **类型安全** - 使用 util.Metadata 存储元数据，支持类型安全访问
5. **snake_case 命名** - ActionType 使用 snake_case，符合 JSON 常规命名习惯

## Metadata 契约

**重要**：`LogEntry.Metadata` 字段使用遵循契约文档定义。

详见：[doc/metadata/logentry.md](../../doc/metadata/logentry.md) - LogEntry.Metadata 契约（客户端可见字段）

新增 ActionType 的 Metadata 字段时：
1. 在契约文档更新表格
2. 同步更新 TypeScript 类型定义
3. 更新 `internal/net/builder.go` 的 `buildAction()` 方法

## 测试

```bash
go test ./pkg/gamelog/... -v
```