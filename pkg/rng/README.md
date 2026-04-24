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
    PoolTypeGood    // Good 池 (Evaluation > 65)
    PoolTypeNeutral // Neutral 池 (Evaluation 41-65)
    PoolTypeBad     // Bad 池 (Evaluation <= 40)
)
```

### DrawEngine

```go
engine := rng.NewDrawEngine(rand.New(rand.NewSource(seed)))

// 从指定池抽取
eventType := engine.DrawEvent(rng.PoolTypeGood, player.LP)
buffType  := engine.DrawBuff(rng.PoolTypeBad, player.LP)
itemType  := engine.DrawItem(player.LP)  // 道具都是 Good
```

### DrawWithProb - 概率加权抽取

`DrawWithProb` 方法支持基于概率权重的抽取，用于地图格子的概率配置：

```go
// 从事件池抽取，根据三个概率权重选择池
result := engine.DrawWithProb(
    eventPool,           // []*EvaluatedItem
    0.6, 0.3, 0.1,      // probGood, probNeutral, probBad
    player.LP,          // 幸运值
)

// result: *CategoryDrawResult
//   - Category: PoolType (Good/Neutral/Bad)
//   - Item: *EvaluatedItem (抽中的项目)
//   - FromAllItems: bool (是否从全部 items 抽取)
```

**概率总和 < 1.0 的处理**：

当 `probGood + probNeutral + probBad < 1.0` 时，剩余概率表示从**全部 items**中抽取（不进行池过滤）：

```go
// 示例：30% Good, 30% Neutral, 30% Bad, 10% 从全部 items 抽取
result := engine.DrawWithProb(items, 0.3, 0.3, 0.3, lp)
// 有 10% 的概率直接从全部 items 抽取，不过滤池
```

### CategoryDrawResult

概率抽取的结果结构：

```go
type CategoryDrawResult struct {
    Category   PoolType       // 选中的池类型
    Item       *EvaluatedItem // 抽中的项目
    FromAllItems bool         // 是否从全部 items 抽取（当 total<1.0 时）
}
```

## LP 影响机制

| 池类型 | LP 效果 |
|--------|---------|
| Good | LP 越高 → 高 Evaluation 权重越大 → 抽到更好的 |
| Bad | LP 越高 → 高 Evaluation 权重越大 → 抽到不那么坏的 |
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
├── draw_engine.go       # DrawEngine 实现（含 DrawWithProb）
├── weighted_pool.go     # WeightedPool 实现
├── luck_modifier.go     # LuckModifier 实现
├── dice.go              # DiceType/DiceManager 骰子系统
├── boss.go              # Boss攻击计算（CalcBossAttackType、SelectBossTarget、CalcPlayerCrit）
├── draw_engine_test.go  # 单元测试
└── README.md            # 本文档
```

## 使用场景

| 场景 | 池类型 | 说明 |
|------|--------|------|
| 毒瘴 Buff 效果 | PoolTypeBad | 每回合触发恶性事件 |
| 捡到圣遗物 | DrawItem | 获得道具奖励 |
| 神眷 Buff 奖励 | PoolTypeGood | 获得良性效果 |
| 迷途/诅咒替换 | PoolTypeBad | 强制给予恶意 Buff |
| **地图格子抽取** | DrawWithProb | 根据格子配置的 probGood/Neutral/Bad 进行概率抽取 |
| **概率混合抽取** | DrawWithProb(total<1) | 部分概率从全部 items 抽取，部分从指定池抽取 |
| **Boss攻击类型决定** | CalcBossAttackType | 根据avgLP决定Boss普通/暴击/技能攻击 |
| **Boss目标选择** | SelectBossTarget | LP加权选择，LP越低越容易被攻击 |
| **玩家暴击计算** | CalcPlayerCrit | 根据骰子品质计算暴击概率 |

## Boss 攻击计算

### BossAttackResult

Boss攻击决策结果：

```go
type BossAttackResult struct {
    AttackType string // "normal", "crit", or "skill"
    SkillType  string // Skill type if AttackType is "skill"
    Target     string // Target player ID
    Damage     int    // Damage amount
}
```

### CalcBossCritSkillProb

Boss暴击/技能概率基于Boss格存活玩家的平均LP：

```
prob = 0.1 + 0.05 × (8 - avgLP)
```

| avgLP | 概率 |
|-------|------|
| 0 | 50% |
| 4 | 30% |
| 8 | 10% |

### CalcBossAttackType

当暴击/技能触发时，50%概率暴击，50%概率技能。技能从BossSkillPool等权重随机抽取。

### SelectBossTarget

Boss攻击目标使用LP加权选择：

```
weight = 1.0 + 0.3 × (8 - LP)
```

LP越低越容易被攻击。

### CalcPlayerCritRate / CalcPlayerCrit

玩家暴击概率基于骰子品质：

| DiceType | 暴击率 |
|----------|--------|
| Gold | 30% |
| Silver | 20% |
| Copper | 10% |
| Wood | 5% |

## 测试

```bash
go test ./pkg/rng/...
```
