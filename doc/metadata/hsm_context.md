# StateContext.Metadata 契约（HSM状态机通信）

`hsm.StateContext.Metadata` 用于状态之间传递**瞬时数据**，生命周期为单 tick。

**位置**：`internal/engine/hsm/state.go`

**可见性**：内部（仅后端使用）

**生命周期**：每次 MatchLoop 创建新的 StateContext，Metadata 随之清空。

---

## 数据分层

| 数据类型 | 存储位置 | 生命周期 | 适用场景 |
|---------|---------|---------|---------|
| **瞬时数据** | StateContext.Metadata | 单 tick | Enter→Update→Exit 通信 |
| **回合级数据** | Game.RoundData | 一回合 | 跨 tick 持久（见 `round_data.md`） |

**重要**：需要跨 tick 持久的数据应使用 `Game.RoundData`，不要存储在 StateContext.Metadata。

---

## 概述

HSM状态使用预定义常量（KeyXxx）存储和读取状态标记，确保类型安全和命名统一：

```go
// internal/engine/hsm/state.go
const (
    KeySkipTurn     = "skip_turn"           // 跳过回合
    KeyFellDown     = "fell_down"           // 落坑标记
    KeyReachedEnd   = "reached_end"         // 到达终点
    KeyDiceSteps    = "dice_steps"          // 骰子结果
    KeyTargetPos    = "target_pos"          // 目标位置
    // ...
)

// 状态中使用
func (ctx *StateContext) WithDiceSteps(steps int) *StateContext {
    ctx.SetInt(KeyDiceSteps, steps)
    return ctx
}

func (ctx *StateContext) GetDiceSteps() int {
    return ctx.GetIntOrDefault(KeyDiceSteps, 0)
}
```

---

## 常量定义表

### 回合流程标记

| 常量 | 值 | 类型 | 来源状态 | 用途 | 目标状态 |
|------|-----|------|----------|------|----------|
| `KeySkipTurn` | `skip_turn` | bool | TurnUpkeep | 跳过剩余回合 | TurnEnd |
| `KeyFellDown` | `fell_down` | bool | TurnMoving | 落坑标记 | TurnLanded |
| `KeyReachedEnd` | `reached_end` | bool | TurnMoving | 到达Boss格子 | BossBattle |
| `KeyDiceSteps` | `dice_steps` | int | MainAction/TurnMoving | 骰子结果 | MoveAction |
| `KeyTargetPos` | `target_pos` | int | TurnMoving | 目标位置 | MoveAction |

### 小游戏相关

| 常量 | 值 | 类型 | 用途 |
|------|-----|------|------|
| `KeyMiniGameRank` | `mini_game_rank` | int | 前缀，实际为 `result_{playerID}` |
| `KeyDiceType` | `dice_type` | int | 前缀，实际为 `dice_{playerID}` |

### Boss/游戏结束

| 常量 | 值 | 类型 | 来源状态 | 用途 |
|------|-----|------|----------|------|
| `KeyBossTrigger` | `boss_trigger_player` | string | TurnLanded | 触发Boss战的玩家ID |
| `KeyWinner` | `winner_id` | string | GameOver | 胜利玩家ID |

### 状态流标记

| 常量 | 值 | 类型 | 来源状态 | 用途 |
|------|-----|------|----------|------|
| `KeyInitialized` | `initialized` | bool | MatchInit | 游戏初始化完成 |
| `KeyMiniGameStarted` | `mini_game_started` | bool | RoundMiniGame | 小游戏阶段开始 |
| `KeyWaitingForResults` | `waiting_for_results` | bool | RoundMiniGame | 等待小游戏结果 |
| `KeyTurnLoopActive` | `turn_loop_active` | bool | TurnLoop | 回合循环活跃 |
| `KeyBossBattleActive` | `boss_battle_active` | bool | BossBattle | Boss战活跃 |
| `KeyGameOver` | `game_over` | bool | GameOver | 游戏结束 |

---

## 动态键（拼接）

| 键格式 | 类型 | 来源 | 用途 |
|--------|------|------|------|
| `result_{playerID}` | int | RoundMiniGame | 玩家小游戏排名（1-4） |
| `dice_{playerID}` | int | RoundPrep | 玩家骰子类型（rng.DiceType） |
| `decision_choice` | int | WaitDecision | 用户选择的选项索引 |
| `decision_processed` | bool | WaitDecision | 决策已处理标记 |

---

## 便捷方法

StateContext 提供便捷方法访问常用字段：

```go
// 回合标记
func (ctx *StateContext) IsSkipTurn() bool
func (ctx *StateContext) SetSkipTurn(skip bool)
func (ctx *StateContext) IsFellDown() bool
func (ctx *StateContext) SetFellDown(fell bool)
func (ctx *StateContext) HasReachedEnd() bool
func (ctx *StateContext) SetReachedEnd(reached bool)

// 移动数据
func (ctx *StateContext) WithDiceSteps(steps int) *StateContext
func (ctx *StateContext) GetDiceSteps() int
func (ctx *StateContext) WithTargetPos(pos int) *StateContext
func (ctx *StateContext) GetTargetPos() int

// 小游戏结果
func (ctx *StateContext) SetMiniGameRank(playerID string, rank int)
func (ctx *StateContext) GetMiniGameRank(playerID string) int
func (ctx *StateContext) SetDiceType(playerID string, diceType rng.DiceType)
func (ctx *StateContext) GetDiceType(playerID string) rng.DiceType
```

---

## 状态间通信示例

### TurnMoving → MoveAction

```go
// TurnMoving 设置骰子结果
ctx.SetInt(KeyDiceSteps, diceSteps)
ctx.SetInt(KeyTargetPos, targetPosition)

// MoveAction 读取数据
steps := ctx.GetIntOrDefault(KeyDiceSteps, 0)
targetPos := ctx.GetIntOrDefault(KeyTargetPos, 0)
```

### RoundMiniGame → RoundPrep

```go
// RoundMiniGame 设置排名
ctx.SetMiniGameRank("player-001", 1)  // 排名第1
ctx.SetMiniGameRank("player-002", 2)  // 排名第2

// RoundPrep 分配骰子类型
rank := ctx.GetMiniGameRank("player-001")
diceType := rankToDiceType(rank)
ctx.SetDiceType("player-001", diceType)
```

### TurnLanded → BossBattle

```go
// TurnLanded 检查到达Boss
if player.Position == mapEngine.GetLength()-1 {
    ctx.SetReachedEnd(true)
    ctx.SetString(KeyBossTrigger, player.ID.UUID())
}

// BossBattle 获取触发者
triggerID := ctx.GetStringOrDefault(KeyBossTrigger, "")
```

---

## 扩展说明

新增状态标记时：
1. 在 `state.go` 添加常量定义（KeyXxx）
2. 如常用，添加便捷方法（WithXxx/GetXxx/SetXxx/IsXxx）
3. 更新此契约文档
4. 确保状态转换正确使用标记

---

## 相关文档

- [internal/engine/hsm/README.md](../../internal/engine/hsm/README.md) - HSM设计文档
- [doc/internal/hsm_design.md](../internal/hsm_design.md) - HSM完整设计
- [internal/engine/hsm/state.go](../../internal/engine/hsm/state.go) - StateContext实现