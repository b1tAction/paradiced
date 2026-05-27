# 成就与积分系统

游戏结算从"Boss击败=胜利"改为纯积分排名制。每局游戏结束后，所有玩家按总积分降序排列，排名第1者为冠军。积分来源分为4个类别，配比约为：

| 类别 | 配比 | 说明 |
|------|------|------|
| 小游戏 (`mini_game`) | ~40% | 每轮小游戏排名得分 |
| Boss战斗 (`boss`) | ~35% | Boss伤害、暴击、K头得分 |
| 道具 (`item`) | ~12.5% | 获得道具、使用道具、骰子升级得分 |
| 成就 (`achievement`) | ~12.5% | 成就达成时额外加分 |

---

## 一、成就定义

每个成就有唯一 `AchievementType`（snake_case 字符串），以及静态定义 `AchievementDefinition`（名称、描述、加分值）。成就每局每人只能获得一次。

### 成就一览表

| AchievementType | 名称 | 描述 | 加分 | 触发机制 |
|----------------|------|------|------|----------|
| `triple_one` | 三连一 | 连续3次掷骰结果为1 | 5 pts | EventBus PhasePreAction |
| `triple_six` | 三连六 | 连续3次掷骰结果为6 | 5 pts | EventBus PhasePreAction |
| `boss_kill_shot` | K头 | 对Boss造成致命一击 | 5 pts | EventBus PhasePreAction |
| `boss_damage_ten` | 勇者之路 | 对Boss累积伤害达到10点 | 5 pts | EventBus PhasePreAction |
| `item_collector` | 道具收藏家 | 同时持有3个或更多道具 | 5 pts | EventBus PhasePreAction |
| `survivor` | 生存大师 | 游戏结束时从未死亡 | 8 pts | HSM-direct (GameOverState) |
| `luck_master` | 幸运之星 | 游戏结束时LP达到最大值 | 8 pts | HSM-direct (GameOverState) |
| `first_to_boss` | 先行者 | 第一个到达Boss格的玩家 | 5 pts | HSM-direct (TurnBossBattleState) |
| `mini_game_winner_three` | 小游戏之王 | 小游戏获得第1名累计达到3次 | 8 pts | HSM-direct (RoundMiniGameState.Exit) |

> **加分说明**：成就加分值直接计入 `achievement` 类别积分。例如获得 `survivor` 成就时，玩家 achievement 类别积分 +8。

---

## 二、触发机制分类

成就按触发机制分为两类：**EventBus PhasePreAction** 和 **HSM-direct**。

### 2.1 EventBus PhasePreAction 触发

EventBus 成就在 **每次 Action 执行前** 自动检测。`ActionContext.ExecuteAction()` 在 Step 0 向所有相关玩家发布 `PhasePreAction`，成就 handler 通过类型断言 `current_action` 来判断当前 Action 类型并提取字段。

#### 订阅时机

`WaitingForHostState.Exit()` 调用 `game.InitializePlayerAchievements(player)` 为每个非Boss玩家订阅 EventBus handler。Boss玩家不订阅（没有成就系统）。

#### 订阅格式

```go
// sourceID: "achievement_" + AchievementType (e.g., "achievement_triple_one")
// decision:  AutoDecision（无需用户确认）
// priority:  -10（低于核心 handler，核心 handler 如 DeathMark/Dominance 先执行）
g.Bus.Subscribe(PhasePreAction, player.ID, sourceID, "achievement", decision)
```

#### Handler 通用流程

1. 检查 `ctx.Player`（不为 nil）
2. 检查玩家是否已拥有该成就（`HasAchievement` → 已拥有则跳过）
3. 从 `ctx.Get("current_action")` 获取当前 Action 并类型断言
4. 断言失败 → 当前 Action 不是目标类型，直接跳过
5. 断言成功 → 提取 Action 字段进行条件判断
6. 条件满足 → 调用 `grantAchievementAndScore()` 授予成就 + 加分

#### 各 EventBus Handler 详解

**triple_one / triple_six** — 骰子三连检测

| 项 | 值 |
|----|----|
| 类型断言 | `actionRef.(*RollDiceAction)` |
| 检测字段 | `rollAction.Steps` |
| 检测逻辑 | `player.PushDiceResult(Steps)` 记录最近3次骰子结果，检查3次连续等于目标值 |
| 触发时机 | 每次掷骰（RollDiceAction）执行前 |

**boss_kill_shot** — K头检测（HP预测法）

| 项 | 值 |
|----|----|
| 类型断言 | `actionRef.(*BossDamageAction)` |
| 玩家过滤 | `ctx.Player.ID.IsBoss() == false`（只处理 SourcePlayer，跳过 Boss） |
| 检测逻辑 | `game.GetBossPlayer().HP <= bossAction.Damage` |
| 预测原理 | PhasePreAction 在 BossDamageAction.Execute **之前**执行，此时 Boss HP 尚未被扣减。通过比较当前 Boss HP 与即将施加的伤害，预测是否致命 |
| 附加分数 | `ScoreBossKillShot` (15 pts) 计入 `boss` 类别 |
| 触发时机 | 每次玩家攻击 Boss（BossDamageAction）执行前 |

> **BossDamageAction.ActorPlayers()** 返回 `[Boss, SourcePlayer]`，PhasePreAction 对这两个玩家分别发布。handler 必须过滤掉 Boss 玩家，只处理 SourcePlayer。

**boss_damage_ten** — Boss累积伤害追踪 + Boss分数

| 项 | 值 |
|----|----|
| 类型断言 | `actionRef.(*BossDamageAction)` |
| 玩家过滤 | `ctx.Player.ID.IsBoss() == false`（只处理 SourcePlayer） |
| 追踪逻辑 | `player.AddBossDamageDealt(damage)` 累积伤害计数 |
| 附加分数 | `damage × ScoreBossDamagePerPt` (每点伤害1分) 计入 `boss` 类别 |
| 暴击加分 | `isCrit → ScoreBossCritBonus` (2分) 计入 `boss` 类别 |
| 成就检测 | `player.GetBossDamageDealt() >= 10` |
| 触发时机 | 每次玩家攻击 Boss（BossDamageAction）执行前 |

> **注意**：boss_damage_ten handler 同时负责 Boss 伤害积分和暴击积分的添加。HSM `enterPlayerBranch` 中已移除这些重复代码。

**item_collector** — 道具收藏检测

| 项 | 值 |
|----|----|
| 类型断言 | `actionRef.(*AddItemAction)` |
| 检测逻辑 | `len(player.Inventory) >= 2`（PhasePreAction 时道具尚未添加，>=2 表示添加后将有3+） |
| 附加分数 | `ScoreItemAcquired` (3 pts) 计入 `item` 类别 |
| 触发时机 | 每次获得道具（AddItemAction）执行前 |

> **注意**：item_collector handler 在每次 AddItemAction 时都会添加 `ScoreItemAcquired` (3分)，不仅仅是达成成就时。

#### 授予后自动取消订阅

`game.GrantAchievementToPlayer()` 在授予成就后调用 `g.Bus.UnsubscribeBySource("achievement_" + achievementType)` 取消该成就的 EventBus handler，防止重复触发。

### 2.2 HSM-direct 触发

HSM-direct 成就在特定状态转换时直接检测，不通过 EventBus。

#### survivor — GameOverState.Enter

游戏结束时（`GameOverState.Enter()`），遍历所有非Boss玩家：
- 条件：`player.GetDeathCount() == 0`
- `DeathAction.Execute()` 调用 `player.IncrementDeathCount()`，确保死亡次数正确计数
- 未死亡 → 授予 `survivor` + 8 pts（achievement 类别）

#### luck_master — GameOverState.Enter

游戏结束时，遍历所有非Boss玩家：
- 条件：`player.LP == player.MaxLP`
- LP达到最大值 → 授予 `luck_master` + 8 pts（achievement 类别）

#### first_to_boss — TurnBossBattleState.Enter

非Boss玩家进入 Boss 战斗时：
- 检查 `game.Metadata.GetBoolOrDefault("first_to_boss_set", false)`
- 标记未设置 → 设为 true，授予 `first_to_boss` + 5 pts
- 标记存储在 `game.Metadata`（游戏级持久化，**不会每轮清空**），而非 `game.RoundData`（每轮清空）

> **Bug 修复记录**：原实现将 `first_to_boss` 标记存储在 `game.RoundData` 中，`RoundMiniGameState.Enter()` 每轮调用 `RoundData.Clear()` 导致标记丢失。跨轮后第二个到达 Boss 格的玩家会再次触发成就。现已修复为使用 `game.Metadata`。

#### mini_game_winner_three — RoundMiniGameState.Exit

每轮小游戏结束时：
- 遍历所有非Boss玩家，rank=1 的玩家 `player.IncrementRoundsWon()`
- 检查 `player.GetRoundsWon() >= 3`
- 累计获得3次第1名 → 授予 `mini_game_winner_three` + 8 pts

---

## 三、积分体系

### 3.1 积分类别

| ScoreCategory | 值 | 说明 |
|---------------|------|------|
| `mini_game` | `"mini_game"` | 小游戏排名得分 |
| `boss` | `"boss"` | Boss战斗得分（伤害+暴击+K头） |
| `item` | `"item"` | 道具得分（获得+使用+骰子升级） |
| `achievement` | `"achievement"` | 成就加分 |

### 3.2 积分点值

**小游戏排名得分**（每轮）

| 排名 | 2人局 | 3人局 | 4人局 |
|------|-------|-------|-------|
| 第1名 | 10 pts | 10 pts | 10 pts |
| 第2名 | 7 pts | 7 pts | 7 pts |
| 第3名 | — | 4 pts | 4 pts |
| 第4名 | — | — | 1 pts |

> 公式：`10 - (rank - 1) × 3`，最低1分。`MiniGameRankToScore(rank, totalPlayers)` 实现。

**Boss战斗得分**

| 得分项 | 常量 | 值 | 说明 |
|--------|------|----|------|
| Boss伤害 | `ScoreBossDamagePerPt` | 1 pt/damage | 每点伤害1分 |
| Boss暴击 | `ScoreBossCritBonus` | 2 pts | 暴击额外加分 |
| K头 | `ScoreBossKillShot` | 15 pts | 击败Boss加分 |

**道具得分**

| 得分项 | 常量 | 值 | 说明 |
|--------|------|----|------|
| 获得道具 | `ScoreItemAcquired` | 3 pts | AddItemAction 时自动添加 |
| 使用道具 | `ScoreItemUsed` | 2 pts | RemoveItemAction 时添加 |
| 骰子升级 | `ScoreDiceUpgrade` | 5 pts | DiceUpgradeAction 时添加 |

**成就加分**

成就加分值即 `AchievementDefinition.Points`：
- 5 pts 类：triple_one, triple_six, boss_kill_shot, boss_damage_ten, item_collector, first_to_boss, luck_master
- 8 pts 类：survivor, mini_game_winner_three

### 3.3 积分添加时机

| 得分事件 | 添加时机 | 添加位置 | 类别 |
|----------|----------|----------|------|
| 小游戏排名得分 | RoundMiniGameState.Exit | global_states.go | mini_game |
| Boss伤害得分 | EventBus PhasePreAction (boss_damage_ten handler) | achievement_registry.go | boss |
| Boss暴击得分 | EventBus PhasePreAction (boss_damage_ten handler) | achievement_registry.go | boss |
| K头得分 | EventBus PhasePreAction (boss_kill_shot handler) | achievement_registry.go | boss |
| 获得道具得分 | EventBus PhasePreAction (item_collector handler) | achievement_registry.go | item |
| 使用道具得分 | MainActionState.OnUseItem (RemoveItemAction后) | turn_states.go | item |
| 骰子升级得分 | MainActionState.OnUseItem (DiceUpgradeAction后) | turn_states.go | item |
| 成就加分 | grantAchievementAndScore() / HSM-direct | achievement_registry.go / global_states.go / turn_states.go | achievement |

### 3.4 积分记录机制

所有积分通过 `game.AddScoreToPlayer(player, category, points, reason, round)` 添加。该方法同时：
1. `player.AddScore(category, points)` — 累加到 `total_score` 和类别子键
2. `player.AppendScoreReason(ScoreReason)` — 记录积分原因供 GameOver 协议渲染

`ScoreReason` 结构：

```go
type ScoreReason struct {
    Category string `json:"category"` // "mini_game", "boss", "item", "achievement"
    Reason   string `json:"reason"`   // "小游戏第1名", "Boss伤害", "K头", "获得道具", etc.
    Points   int    `json:"points"`   // 本次得分点数
    Round    int    `json:"round"`    // 轮次（0表示非轮次相关）
}
```

---

## 四、前端协议：GameOver 数据结构

游戏结束时，服务器广播 `GameOver` 消息（OpCode = `OpGameOver = 7`）。包含两个数组：`Rankings` 和 `Stats`。

### 4.1 GameOver 结构

```json
{
  "rankings": [
    {
      "player_id": "xxx-xxx-xxx",
      "display_name": "Alice",
      "rank": 1,
      "total_score": 58,
      "mini_game_score": 27,
      "boss_score": 18,
      "item_score": 8,
      "achievement_score": 5,
      "achievements": ["survivor", "boss_kill_shot"],
      "score_reasons": [
        {"category": "mini_game", "reason": "小游戏第1名", "points": 10, "round": 1},
        {"category": "boss", "reason": "Boss伤害", "points": 5, "round": 2},
        {"category": "achievement", "reason": "K头", "points": 5, "round": 0}
      ]
    },
    ...
  ],
  "stats": [
    {
      "player_id": "xxx-xxx-xxx",
      "display_name": "Alice",
      "rounds_won": 2,
      "events_drawn": 3,
      "items_used": 2,
      "boss_damage_dealt": 8,
      "achievements": ["survivor", "boss_kill_shot"],
      "total_score": 58,
      "score_breakdown": {
        "mini_game": 27,
        "boss": 18,
        "item": 8,
        "achievement": 5
      }
    },
    ...
  ]
}
```

### 4.2 Rankings vs Stats

| 字段 | Rankings | Stats | 说明 |
|------|----------|-------|------|
| 包含范围 | 仅非Boss玩家 | 所有玩家（含Boss） | Rankings 用于排名展示，Stats 用于全玩家统计 |
| 排序方式 | 总积分降序 | 不排序（按 Players 列表原序） | Rankings[0] = 冠军 |
| 积分明细 | ✅ 4个类别子分数 | ✅ score_breakdown map | Rankings 用扁平字段，Stats 用 map |
| 成就列表 | ✅ | ✅ | 类型字符串数组 |
| ScoreReasons | ✅ | ❌ | Rankings 提供逐条得分明细供UI渲染 |
| Boss伤害 | ❌ | ✅ boss_damage_dealt | Stats 额外提供原始Boss伤害数值 |

### 4.3 PlayerRanking 字段详解

| 字段 | 类型 | 说明 |
|------|------|------|
| `player_id` | string | 玩家UUID（`core.Player.ID.UUID()`） |
| `display_name` | string | 用户昵称，由 NakamaBroadcastAdapter 注入，回退到 UUID |
| `rank` | int | 排名位置（1 = 冠军），按 total_score 降序赋值 |
| `total_score` | int | 4个类别积分总和 |
| `mini_game_score` | int | 小游戏类别累计积分 |
| `boss_score` | int | Boss类别累计积分 |
| `item_score` | int | 道具类别累计积分 |
| `achievement_score` | int | 成就类别累计积分 |
| `achievements` | string[] | 已达成成就类型列表（如 `["survivor", "boss_kill_shot"]`） |
| `score_reasons` | ScoreReason[] | 逐条积分明细（类别+原因+点数+轮次） |

### 4.4 PlayerStats 字段详解

| 字段 | 类型 | 说明 |
|------|------|------|
| `player_id` | string | 玩家UUID |
| `display_name` | string | 用户昵称 |
| `rounds_won` | int | 小游戏获得第1名的轮数 |
| `events_drawn` | int | 触发的随机事件数 |
| `items_used` | int | 消耗的道具数 |
| `boss_damage_dealt` | int | 对Boss的原始累积伤害数值（不含乘分） |
| `achievements` | string[] | 已达成成就类型列表 |
| `total_score` | int | 总积分 |
| `score_breakdown` | map<string,int> | 类别→积分映射（`{"mini_game": 27, "boss": 18, ...}`） |

### 4.5 排名规则

1. 按 `total_score` 降序排列
2. 相同总分时，按 `player_id` 字符序排列（确定性排序）
3. 排名位置从 1 开始赋值（`rankings[i].Rank = i + 1`）

### 4.6 DisplayName 注入

`NakamaBroadcastAdapter.BroadcastGameOver()` 自动注入 `DisplayName`：
- Rankings：遍历 `over.Rankings[i]`，从 `handler.players` 中查找匹配 UUID 的玩家，注入 `display_name`
- Stats：同上
- 注入不会覆盖 UUID，仅在 `display_name` 为空时填充

---

## 五、内部数据存储

### 5.1 Player 积分存储（Metadata 键）

| 键 | 类型 | 说明 |
|----|------|------|
| `total_score` | int | 总积分 |
| `score_mini_game` | int | 小游戏类别积分 |
| `score_boss` | int | Boss类别积分 |
| `score_item` | int | 道具类别积分 |
| `score_achievement` | int | 成就类别积分 |
| `boss_damage_dealt` | int | Boss累积伤害追踪 |
| `achievements` | []string | 已达成成就类型列表 |
| `last_dice_results` | []int | 最近3次骰子结果（用于三连检测） |
| `death_count` | int | 死亡次数（用于 survivor 检测） |
| `score_reasons` | []ScoreReason | 逐条积分明细 |
| `rounds_won` | int | 小游戏第1名累计轮数 |

### 5.2 Game 级存储

| 键 | 存储位置 | 说明 |
|----|----------|------|
| `first_to_boss_set` | `game.Metadata` | 首个到达Boss格标记（**游戏级持久化**，不随每轮清空） |
| `first_to_boss_player` | `game.Metadata` | 首个到达Boss格的玩家ID |
| `boss_defeated` | `game.RoundData` | Boss被击败标记（轮级数据） |
| `boss_defeated_by` | `game.RoundData` |击败Boss的玩家ID（轮级数据） |

> **关键区别**：`game.RoundData` 在每轮 `RoundMiniGameState.Enter()` 时调用 `Clear()` 清空；`game.Metadata` 跨轮持久化。

---

## 六、代码架构

### 6.1 文件位置

| 文件 | 包 | 说明 |
|------|-----|------|
| `pkg/constants/achievement.go` | constants | AchievementType 枚举 + AchievementDefinition |
| `pkg/constants/score.go` | constants | ScoreCategory 枚举 + ScoreReason + 积分常量 |
| `internal/core/player.go` | core | Player 积分/成就追踪方法 |
| `internal/engine/achievement_registry.go` | engine | AchievementRegistry + AchievementHandlerConfig + handler factories |
| `internal/engine/score_registry.go` | engine | ScoreRegistry + 辅助方法 |
| `internal/engine/game.go` | engine | Game.AddScoreToPlayer / GrantAchievementToPlayer / InitializePlayerAchievements |
| `internal/engine/hsm/global_states.go` | hsm | GameOverState (survivor/luck_master) + RoundMiniGameState.Exit (mini_game_winner_three) |
| `internal/engine/hsm/turn_states.go` | hsm | TurnBossBattleState.Enter (first_to_boss) + MainActionState.OnUseItem (item scores) |
| `internal/engine/action/types.go` | action | DeathAction.Execute (IncrementDeathCount) |
| `pkg/net/sync.go` | net | GameOver + PlayerRanking + PlayerStats 协议结构 |

### 6.2 Registry 模式

AchievementRegistry 采用与 BuffRegistry 相同的模式：

```
AchievementRegistry
  ├── definitions: map[AchievementType]*AchievementDefinition   // 静态元数据
  └ configs:     map[AchievementType]*AchievementHandlerConfig  // EventBus 运行时配置
      ├── Phases:   []Phase          // 通常为 [PhasePreAction]
      ├── Priority: int              // -10（核心 handler 先执行）
      └ Handler:  AchievementHandler // 检测逻辑
```

- **Definitions**：可在启动时查询所有成就元数据（名称、描述、加分值）
- **Configs**：仅 EventBus 成效有 config；HSM-direct 成效的 config 为 nil
- **Lazy Init**：`GlobalAchievementRegistry` 通过 `EnsureAchievementRegistryInitialized()` 延迟初始化，避免与 `GlobalScoreRegistry` 的 init 循环

### 6.3 成就触发最终架构

| AchievementType | 触发机制 | 检测方式 | 触发时机 |
|----------------|----------|---------|---------|
| triple_one | EventBus PhasePreAction | 类型断言 RollDiceAction, PushDiceResult, 连续3次=1 | 任何Action触发时 |
| triple_six | EventBus PhasePreAction | 类型断言 RollDiceAction, PushDiceResult, 连续3次=6 | 任何Action触发时 |
| boss_kill_shot | EventBus PhasePreAction | 类型断言 BossDamageAction, ctx.Player≠Boss, bossPlayer.HP≤Damage | 任何Action触发时 |
| boss_damage_ten | EventBus PhasePreAction | 类型断言 BossDamageAction, ctx.Player≠Boss, AddBossDamageDealt+积分 | 任何Action触发时 |
| item_collector | EventBus PhasePreAction | 类型断言 AddItemAction, len(Inventory)>=2(预判3+) | 任何Action触发时 |
| survivor | HSM-direct (GameOverState) | player.GetDeathCount()==0 | 游戏结束时 |
| luck_master | HSM-direct (GameOverState) | player.LP==player.MaxLP | 游戏结束时 |
| first_to_boss | HSM-direct (TurnBossBattleState) | game.Metadata标记，第一个进入Boss战 | 进入Boss战时 |
| mini_game_winner_three | HSM-direct (RoundMiniGameState.Exit) | player.GetRoundsWon()>=3 | 小游戏结束时 |

---

## 七、完整数据流示例

### 7.1 4人局积分分布示例

假设4人局3轮游戏：

**玩家A**：2次小游戏第1名 + Boss伤害12点(含1次暴击) + K头 + 获得2道具使用1道具 + survivor + first_to_boss

```
mini_game:  10×2 + 7 = 27 pts (第1名×2轮, 第2名×1轮)
boss:       12×1 + 2(暴击) + 15(K头) = 29 pts
item:       3×2(获得) + 2×1(使用) = 8 pts
achievement: 8(survivor) + 5(first_to_boss) + 5(K头) = 18 pts
total:      82 pts → Rank 1 (冠军)
```

**玩家B**：1次小游戏第1名 + Boss伤害8点 + 获得3道具 + item_collector + boss_damage_ten

```
mini_game:  10 + 4 + 1 = 15 pts
boss:       8×1 = 8 pts
item:       3×3(获得) + 2×1(使用) = 11 pts
achievement: 5(item_collector) + 5(boss_damage_ten) = 10 pts
total:      44 pts → Rank 2
```

### 7.2 GameOver JSON 示例

```json
{
  "rankings": [
    {
      "player_id": "a1b2c3d4",
      "display_name": "玩家A",
      "rank": 1,
      "total_score": 82,
      "mini_game_score": 27,
      "boss_score": 29,
      "item_score": 8,
      "achievement_score": 18,
      "achievements": ["survivor", "boss_kill_shot", "first_to_boss"],
      "score_reasons": [
        {"category": "mini_game", "reason": "小游戏第1名", "points": 10, "round": 1},
        {"category": "mini_game", "reason": "小游戏第1名", "points": 10, "round": 2},
        {"category": "mini_game", "reason": "小游戏第2名", "points": 7, "round": 3},
        {"category": "boss", "reason": "Boss伤害", "points": 5, "round": 0},
        {"category": "boss", "reason": "Boss暴击", "points": 2, "round": 0},
        {"category": "boss", "reason": "K头", "points": 15, "round": 0},
        {"category": "item", "reason": "获得道具", "points": 3, "round": 0},
        {"category": "item", "reason": "获得道具", "points": 3, "round": 0},
        {"category": "item", "reason": "使用道具", "points": 2, "round": 0},
        {"category": "achievement", "reason": "生存大师", "points": 8, "round": 0},
        {"category": "achievement", "reason": "K头", "points": 5, "round": 0},
        {"category": "achievement", "reason": "先行者", "points": 5, "round": 0}
      ]
    },
    {
      "player_id": "e5f6g7h8",
      "display_name": "玩家B",
      "rank": 2,
      "total_score": 44,
      ...
    }
  ],
  "stats": [
    {
      "player_id": "a1b2c3d4",
      "display_name": "玩家A",
      "rounds_won": 2,
      "events_drawn": 4,
      "items_used": 1,
      "boss_damage_dealt": 12,
      "achievements": ["survivor", "boss_kill_shot", "first_to_boss"],
      "total_score": 82,
      "score_breakdown": {"mini_game": 27, "boss": 29, "item": 8, "achievement": 18}
    },
    {
      "player_id": "beeeeeef-beef-beef-beef-beeeeeeeeeef",
      "display_name": "凶兽",
      "rounds_won": 0,
      "events_drawn": 0,
      "items_used": 0,
      "boss_damage_dealt": 0,
      "achievements": [],
      "total_score": 0,
      "score_breakdown": {"mini_game": 0, "boss": 0, "item": 0, "achievement": 0}
    },
    ...
  ]
}
```

> **注意**：Stats 包含 Boss 玩家（`boss_damage_dealt=0, total_score=0`），Rankings 不包含 Boss 玩家。

---

## 八、前端渲染建议

### 8.1 排名页面

- 使用 `rankings` 数组渲染排名列表
- `rankings[0]` = 冠军，突出显示
- `score_reasons` 可展开为积分明细面板（每个得分条目的原因+点数）
- `achievements` 渲染为成就徽章（使用 `AchievementDefinition` 的 `name` + `desc`）

### 8.2 统计页面

- 使用 `stats` 数组渲染详细统计
- `score_breakdown` 渲染为饼图/柱状图（4类别占比）
- `boss_damage_dealt` 为原始伤害数值（非积分），可用于展示实际Boss战斗贡献

### 8.3 成就达成实时通知

EventBus 成效在游戏过程中实时达成，但当前协议不包含实时成就通知推送。前端可通过 `StateSync` 中的玩家数据变化检测（成就列表变更）来渲染达成动画。后续可考虑增加专用 OpCode 推送成就达成通知。