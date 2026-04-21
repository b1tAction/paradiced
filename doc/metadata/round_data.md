# Game.RoundData 契约（回合级持久数据）

`Game.RoundData` 存储回合级持久数据，在回合结束时清理。

**位置**：`internal/engine/game.go`

**可见性**：内部（仅后端使用）

---

## 概述

RoundData 用于存储需要跨 tick 持久但不需要放入核心字段的数据。数据在每回合开始时（RoundMiniGameState.Enter）清空。

**数据生命周期**：
- 回合开始时清空
- 在 RoundMiniGame → RoundPrep → TurnLoop 期间持久
- 下一回合 RoundMiniGame.Enter 时清空

---

## 字段定义

| 键 | 类型 | 来源 | 用途 |
|----|------|------|------|
| `mini_game_ranks` | `map[string]int` | RoundMiniGame | 玩家小游戏排名（playerID → rank） |
| `dice_types` | `map[string]int` | RoundPrep | 玩家骰子类型（playerID → DiceType 值） |

---

## 常量定义

常量定义在 `internal/engine/round_data.go`，不暴露到 `pkg/constants`：

```go
const (
    KeyMiniGameRanks = "mini_game_ranks"
    KeyDiceTypes     = "dice_types"
)
```

---

## 辅助方法

`internal/engine/round_data.go` 提供辅助方法：

```go
// 获取小游戏排名 map
func GetMiniGameRanks(roundData *util.Metadata) map[string]int

// 设置小游戏排名 map
func SetMiniGameRanks(roundData *util.Metadata, ranks map[string]int)

// 获取骰子类型 map
func GetDiceTypes(roundData *util.Metadata) map[string]int

// 设置骰子类型 map
func SetDiceTypes(roundData *util.Metadata, types map[string]int)
```

---

## StateContext 便捷方法

StateContext 提供便捷方法访问 RoundData：

```go
// 设置玩家小游戏排名
func (ctx *StateContext) SetMiniGameRank(playerID string, rank int)

// 获取玩家小游戏排名（返回 0 表示未设置）
func (ctx *StateContext) GetMiniGameRank(playerID string) int

// 设置玩家骰子类型
func (ctx *StateContext) SetDiceType(playerID string, diceType rng.DiceType)

// 获取玩家骰子类型（返回 DiceTypeWood 表示未设置）
func (ctx *StateContext) GetDiceType(playerID string) rng.DiceType
```

---

## 使用示例

### RoundMiniGame 设置排名

```go
func (s *RoundMiniGameState) OnMiniGameResult(ctx *StateContext, playerID string, rank int) {
    ctx.SetMiniGameRank(playerID, rank)
}
```

### RoundPrep 读取排名、分配骰子

```go
func (s *RoundPrepState) Enter(ctx *StateContext) {
    for _, player := range players {
        rank := ctx.GetMiniGameRank(player.ID.UUID())
        diceType := rng.RankToDiceType(rank)
        ctx.SetDiceType(player.ID.UUID(), diceType)
    }
}
```

### 清空 RoundData

```go
func (s *RoundMiniGameState) Enter(ctx *StateContext) {
    game := ctx.GetGame()
    if game != nil && game.RoundData != nil {
        game.RoundData.Clear()
    }
}
```

---

## 数据分层设计

| 层级 | 存储位置 | 生命周期 | 适用数据 |
|-----|---------|---------|---------|
| **瞬时层** | StateContext.Metadata | 单 tick | Enter→Update→Exit 通信 |
| **回合层** | Game.RoundData | 一回合 | 跨状态但回合结束时清理 |
| **核心层** | Game/HSM 字段 | 整个游戏 | 类型安全的核心数据 |

---

## 扩展指南

新增回合级数据时：

1. 在 `internal/engine/round_data.go` 添加常量定义
2. 如需要，添加辅助方法
3. 在 StateContext 添加便捷方法（如常用）
4. 更新此契约文档

---

## 相关文档

- [doc/metadata/hsm_context.md](hsm_context.md) - StateContext.Metadata 契约
- [internal/engine/game.go](../../internal/engine/game.go) - Game 结构体
- [internal/engine/round_data.go](../../internal/engine/round_data.go) - 常量定义