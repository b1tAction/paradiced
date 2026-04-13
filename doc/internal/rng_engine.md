# RNG Engine (随机数引擎) 实现文档

## 概述

RNG Engine 是《命运骰子》游戏的随机抽奖核心模块。每局游戏拥有唯一的 `rand.Rand` 实例，确保随机序列可复现。抽奖分为三个池（Good/Neutral/Bad），由游戏逻辑决定池类型，用户 LP 值影响池内具体抽取结果。

## 核心设计

### PoolType 池类型

池类型由游戏逻辑（事件、道具效果等）决定，而非 LP 自动选择：

```go
type PoolType int
const (
    PoolTypeGood    // Good池 (Evaluation > 65)
    PoolTypeNeutral // Neutral池 (Evaluation 41-65)
    PoolTypeBad     // Bad池 (Evaluation <= 40)
)
```

**使用场景示例**：
- 某事件需要给用户增加恶意 Buff → 调用者指定 `PoolTypeBad`
- 某事件需要给用户奖励良性事件 → 调用者指定 `PoolTypeGood`
- 某道具效果随机增益 → 调用者指定 `PoolTypeGood`

### LP 影响机制

LP（幸运值 0-8）仅影响池内权重，不决定池类型：

| 池类型 | LP=0 | LP=8 |
|--------|------|------|
| Good池 | 基础权重 | 高 Evaluation 权重更大（抽到更好的） |
| Neutral池 | 基础权重 | 影响较小 |
| Bad池 | 基础权重 | 高 Evaluation 权重更大（抽到不那么坏的） |

**权重公式**：
```
// Good池: LP越高，高Evaluation权重越大
weight = baseWeight × (1 + lpEvalFactor × LP × baseWeight)

// Bad池: LP越高，高Evaluation（不那么坏）权重越大
weight = baseWeight × (1 + lpEvalFactor × LP × baseWeight)
// 其中 baseWeight = Evaluation / 40（Bad阈值）
```

### DrawEngine 抽奖引擎

```go
type DrawEngine struct {
    rng *rand.Rand  // 游戏唯一随机源
}

// 从指定池抽取事件
func (e *DrawEngine) DrawEvent(poolType PoolType, lp int) event.EventType

// 从指定池抽取Buff
func (e *DrawEngine) DrawBuff(poolType PoolType, lp int) buff.BuffType

// 抽取道具（道具都是Good）
func (e *DrawEngine) DrawItem(lp int) item.ItemType
```

## 与 Game 集成

```go
type Game struct {
    ID      string
    Bus     *event.EventBus
    Players []*core.Player
    State   *GameState
    RNG     *rand.Rand      // 游戏唯一随机源
    Draw    *rng.DrawEngine // 抽奖引擎
}

// 创建游戏（seed=0时自动生成）
func NewGame(gameID string, seed int64) *Game

// 使用示例：某事件需要给玩家增加恶意Buff
badBuff := game.Draw.DrawBuff(rng.PoolTypeBad, player.LP)
// 高LP玩家在Bad池会抽到"不那么坏"的Buff
```

## 文件结构

```
pkg/rng/
├── draw_engine.go      # DrawEngine 核心实现
├── draw_engine_test.go # 单元测试
└── README.md           # 包说明文档
```

## Evaluation 分类

基于 `internal/core/types/evaluation.go`：

| 分类 | Evaluation范围 | 示例 |
|------|---------------|------|
| Good | > 65 | DivineBless(100), MilkTea(80), Herb(70) |
| Neutral | 41-65 | Exchange(50), HiddenBuff(55), TasteTest(55) |
| Bad | ≤ 40 | Thunder(10), CurseBuddha(25), Mosquito(35) |

## LP 影响示例

### Good池抽奖

| Event | Evaluation | LP=0权重 | LP=8权重 |
|-------|------------|---------|---------|
| DivineBless | 100 | 1.0 | 1.8 |
| MilkTea | 80 | 0.8 | 1.44 |
| Herb | 70 | 0.7 | 1.26 |

LP=8时，DivineBless（Evaluation=100）概率显著高于 Herb（Evaluation=70）。

### Bad池抽奖

| Event | Evaluation | LP=0权重 | LP=8权重 |
|-------|------------|---------|---------|
| Thunder | 10 | 0.25 | 0.45 |
| CurseBuddha | 25 | 0.625 | 1.05 |
| Mosquito | 35 | 0.875 | 1.58 |

LP=8时，Mosquito（Evaluation=35，仅损失 HP-1）概率高于 Thunder（Evaluation=10，HP归零）。

## 测试覆盖

| 测试类 | 覆盖内容 |
|--------|----------|
| PoolTypeTest | String/ToCategoryString |
| DrawEngineTest | 创建、GetRNG |
| WeightTest | Good/Bad/Neutral权重计算 |
| DrawEventTest | Good/Neutral/Bad池抽取验证 |
| DrawBuffTest | Good/Neutral/Bad池抽取验证 |
| DrawItemTest | 道具抽取、LP影响 |
| LPInfluenceTest | LP=0 vs LP=8 Evaluation对比 |
| EdgeCaseTest | LP边界、seed一致性 |

## 游戏场景示例

### 事件触发抽奖

```go
// 某事件"毒瘴Buff效果"：每回合触发一次恶性随机事件
// 需要从Bad池抽取事件
func handlePoisonBuff(game *Game, player *core.Player) {
    eventType := game.Draw.DrawEvent(rng.PoolTypeBad, player.LP)
    // 高LP玩家会抽到不那么坏的恶性事件
    applyEventEffect(player, eventType)
}
```

### 道具奖励抽奖

```go
// 某事件"捡到圣遗物"：获得一次道具抽奖
// 道具都是Good，LP影响抽到更好的道具
func handleRelicEvent(game *Game, player *core.Player) {
    itemType := game.Draw.DrawItem(player.LP)
    // 高LP玩家更可能抽到高Evaluation道具
    item := core.NewItem(itemType, core.GenerateItemID())
    player.AddItem(item)
}
```

### Buff 交换效果

```go
// 某道具效果"将敌人的良性Buff换成恶意Buff"
// 需要从Bad池抽取替换的Buff
func applyBuffSwap(game *Game, target *core.Player) {
    newBuffType := game.Draw.DrawBuff(rng.PoolTypeBad, target.LP)
    // 目标的高LP会使其获得不那么坏的恶意Buff
    newBuff := core.NewBuff(newBuffType, 3)
    game.ApplyBuffToPlayer(target, newBuff)
}
```