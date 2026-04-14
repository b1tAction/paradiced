# pkg/rng - 随机抽奖引擎

随机抽奖核心模块，为《派乐代》游戏提供 Event/Buff/Item 的随机抽取功能。

## 设计原则

1. **池类型由调用者决定**：游戏逻辑明确指定从哪个池（Good/Neutral/Bad）抽取
2. **LP 仅影响池内权重**：高 LP 在 Bad 池抽到"不那么坏"的结果，在 Good 池抽到"更好"的结果
3. **唯一随机源**：每局游戏共享一个 `rand.Rand` 实例，确保可复现

## 核心结构

### PoolType

```go
type PoolType int
const (
    PoolTypeGood    // Good池 (Evaluation > 65)
    PoolTypeNeutral // Neutral池 (Evaluation 41-65)
    PoolTypeBad     // Bad池 (Evaluation <= 40)
)
```

### DrawEngine

```go
engine := rng.NewDrawEngine(rand.New(rand.NewSource(seed)))

// 从指定池抽取
eventType := engine.DrawEvent(rng.PoolTypeGood, player.LP)
buffType  := engine.DrawBuff(rng.PoolTypeBad, player.LP)
itemType  := engine.DrawItem(player.LP)  // 道具都是Good
```

## LP 影响机制

| 池类型 | LP 效果 |
|--------|---------|
| Good | LP越高 → 高Evaluation权重越大 → 抽到更好的 |
| Bad | LP越高 → 高Evaluation权重越大 → 抽到不那么坏的 |
| Neutral | LP 影响较小 |

**权重公式**：
```
weight = baseWeight × (1 + 0.1 × LP × baseWeight)
```

## 与 Game 集成

```go
// internal/engine/game.go
type Game struct {
    RNG  *rand.Rand      // 游戏唯一随机源
    Draw *rng.DrawEngine // 抽奖引擎
}

func NewGame(gameID string, seed int64) *Game

// 使用
badBuff := game.Draw.DrawBuff(rng.PoolTypeBad, player.LP)
```

## 文件结构

```
pkg/rng/
├── draw_engine.go      # DrawEngine 实现
├── draw_engine_test.go # 单元测试
└── README.md           # 本文档
```

## 使用场景

| 场景 | 池类型 | 说明 |
|------|--------|------|
| 毒瘴Buff效果 | PoolTypeBad | 每回合触发恶性事件 |
| 捡到圣遗物 | DrawItem | 获得道具奖励 |
| 神眷Buff奖励 | PoolTypeGood | 获得良性效果 |
| 迷途/诅咒替换 | PoolTypeBad | 强制给予恶意Buff |

## 测试

```bash
go test ./pkg/rng/...
```