# GameOver 动画与结算页前端设计

## 一、概述

当前 GameOver 页面在收到 `OpGameOver (7)` 消息后直接渲染完整的排名与统计数据，缺乏仪式感与戏剧性。本设计提出一套**逐名揭示 + 分数快速滚动 + 全局结算页**的三阶段动画流程，增强游戏结束的观赏体验，并补充数据统计维度（成就数、死亡次数等）。

---

## 二、动画流程总览

整体动画分为 **三个阶段**：

| 阶段 | 名称 | 时长（建议） | 说明 |
|------|------|-------------|------|
| Phase 1 | 逐名揭示（Rank Reveal） | ~12-16s | 从第4名到第1名依次揭示，每人展示分数从0涨到总分的快速滚动动画 |
| Phase 2 | 冠军特写（Champion Spotlight） | ~3-4s | 冠军专属高光动画 |
| Phase 3 | 全局结算页（Full GameOver Page） | 常驻 | 展示完整排名 + 全部统计数据 |

---

## 三、Phase 1：逐名揭示（Rank Reveal）

### 3.1 揭示顺序

按照 `rankings` 数组从末尾到开头依次揭示，即：

- 第4名（`rankings[3]`） → 第3名（`rankings[2]`） → 第2名（`rankings[1]`） → 第1名（`rankings[0]`）

> 注意：2人局时只有第2名和第1名，3人局时有第3、第2、第1名。动画应根据 `rankings.length` 动态调整揭示人数。

### 3.2 单人揭示流程

每位玩家的揭示包含以下步骤：

1. **入场动画**（~0.8s）
   - 玩家头像/阵营图标 + DisplayName 从屏幕侧方滑入
   - 排名数字淡入（大字号，如 "4th" / "第4名"）
   - 阵营色彩主题（青龙=青色、朱雀=红色、白虎=白色、玄武=深蓝/紫色）

2. **分数滚动动画**（~2-3s）
   - 中心位置的大数字从 `0` 快速滚动到 `total_score`
   - **核心设计**：按照 `score_reasons` 数组逐条累加，每条 reason 对应一段滚动增量
   - 滚动过程中，在数字旁动态浮现每条 score_reason 的摘要标签（如 "小游戏第1名 +10"），短暂停留后淡出
   - 滚动完成后数字停在 `total_score`，伴随一个缩放/弹跳效果

3. **成就徽章展示**（~1s，仅在 achievements.length > 0 时）
   - 从 `achievements` 数组依次弹出成就徽章图标
   - 每个徽章使用 `DefinitionsConfig` 中对应 `AchievementDefinition`（若协议中有成就定义）或客户端本地缓存的成就元数据（name, desc）渲染
   - 徽章弹出时有轻微旋转+缩放动画

4. **退场/过渡**（~0.5s）
   - 玩家卡片整体缩小到侧边栏位置，留出中间舞台给下一个玩家
   - 或者：所有已揭示玩家保持在屏幕侧边排列，新玩家占据中央

### 3.3 分数滚动的实现细节

分数滚动是本动画的核心视觉。具体实现建议：

```
score_reasons = [
  { category: "mini_game", reason: "小游戏第1名", points: 10, round: 1 },
  { category: "boss",      reason: "Boss伤害",    points: 5,  round: 0  },
  { category: "boss",      reason: "Boss暴击",    points: 2,  round: 0  },
  { category: "boss",      reason: "K头",         points: 15, round: 0  },
  { category: "item",      reason: "获得道具",    points: 3,  round: 0  },
  { category: "achievement", reason: "生存大师",  points: 8,  round: 0  },
]

// 滚动过程：
// 0 → 10 (浮现 "小游戏第1名 +10")
// 10 → 15 (浮现 "Boss伤害 +5")
// 15 → 17 (浮现 "Boss暴击 +2")
// 17 → 32 (浮现 "K头 +15")
// 32 → 35 (浮现 "获得道具 +3")
// 35 → 43 (浮现 "生存大师 +8")
// 最终停在 43 → 弹跳效果
```

#### 类别色彩映射

每个 `ScoreCategory` 对应一种色彩，在浮动标签和进度条中使用：

| Category | 色彩 | 用途 |
|----------|------|------|
| `mini_game` | 金黄色 (#FFD700) | 小游戏类标签、进度条段 |
| `boss` | 橙红色 (#FF4500) | Boss战斗类标签、进度条段 |
| `item` | 翠绿色 (#32CD32) | 道具类标签、进度条段 |
| `achievement` | 紫色 (#9370DB) | 成就类标签、进度条段 |

#### 滚动速度控制

- 每条 reason 的滚动时长 = `base_time_per_reason` × (1 + log₂(points / avg_points))
- 大额加分（如 K头 +15）滚动时间更长，视觉冲击更强
- 建议基础时间 ~300ms/reason，动态调整

#### 伴随进度条

在分数数字下方可渲染一条**分类进度条**（堆叠条形图），长度按各类别占比分配，在分数滚动过程中逐步填充：

```
|████████(mini_game)████████(boss)██(item)█(achievement)|
```

### 3.4 Phase 1 整体时间估算

| 玩家数 | 单人揭示时间 | 总揭示时间 | 冠军特写 | 总 Phase 1+2 |
|--------|------------|-----------|---------|-------------|
| 2人局 | ~4s × 2 = ~8s | ~8s | ~3s | ~11s |
| 3人局 | ~4s × 3 = ~12s | ~12s | ~3s | ~15s |
| 4人局 | ~4s × 4 = ~16s | ~16s | ~3s | ~19s |

---

## 四、Phase 2：冠军特写（Champion Spotlight）

第1名揭示完毕后，进入短暂的高光动画：

1. **冠军卡片放大居中**（~1s）
   - 当前揭示的冠军卡片从侧边位置放大回屏幕中央
   - 金色光晕/粒子效果围绕

2. **冠军标识**（~1.5s）
   - "🏆 冠军" / "第一名" 大字标题淡入
   - 阵营专属胜利特效（如青龙=龙形粒子、朱雀=火焰粒子等）

3. **短暂停留**（~1s）
   - 冠军分数再次弹跳确认
   - 成就徽章整体闪烁一次

4. **过渡到全局结算页**（~0.5s）
   - 整个画面渐暗，然后 Phase 3 页面从底部滑入

---

## 五、Phase 3：全局结算页（Full GameOver Page）

### 5.1 页面布局

全局结算页为常驻页面，用户可自由浏览，包含以下区块：

```
┌─────────────────────────────────────────────────────┐
│                 🏆 冠军：玩家A                        │  ← 顶部冠军横幅
│            阵族：青龙  总分：82                        │
├─────────────────────────────────────────────────────┤
│                                                     │
│  ┌─ 排名列表 ──────────────────────────────────┐   │  ← 左侧：排名面板
│  │ 1. 玩家A  青龙  82pts                       │   │
│  │    mini_game:27 boss:29 item:8 achievement:18│   │
│  │ 2. 玩家B  白虎  44pts                       │   │
│  │    mini_game:15 boss:8 item:11 achievement:10│   │
│  │ 3. 玩家C  玄武  30pts                       │   │
│  │ 4. 玩家D  朱雀  22pts                       │   │
│  └─────────────────────────────────────────────┘   │
│                                                     │
│  ┌─ 数据统计 ──────────────────────────────────┐   │  ← 右侧：统计面板
│  │  详细统计表（见 5.2）                        │   │
│  └─────────────────────────────────────────────┘   │
│                                                     │
│  ┌─ 成就一览 ──────────────────────────────────┐   │  ← 底部：成就展示
│  │  全局成就汇总（见 5.3）                      │   │
│  └─────────────────────────────────────────────┘   │
│                                                     │
│              [返回大厅]  [再来一局]                   │  ← 操作按钮
└─────────────────────────────────────────────────────┘
```

### 5.2 数据统计表（扩展版）

当前 `PlayerStats` 已包含 `rounds_won`、`events_drawn`、`items_used`、`boss_damage_dealt`、`achievements`、`total_score`、`score_breakdown`。建议前端在结算页中展示以下**增强统计**：

| 统计项 | 数据来源 | 说明 | 当前协议是否包含 |
|--------|---------|------|----------------|
| 总积分 | `total_score` | 已有 | ✅ |
| 积分分类占比 | `score_breakdown` | 饼图/柱状图渲染 | ✅ |
| 小游戏胜场 | `rounds_won` | 小游戏第1名次数 | ✅ |
| 事件触发数 | `events_drawn` | 随机事件触发次数 | ✅ |
| 道具使用数 | `items_used` | 消耗道具数 | ✅ |
| Boss伤害 | `boss_damage_dealt` | 原始伤害值 | ✅ |
| 成就列表 | `achievements` | 已达成成就类型数组 | ✅ |
| 成就数目 | `achievements.length` | 成就达成总数 | ✅（前端计算） |
| 死亡次数 | **新增字段** | 累计死亡次数 | ❌ 需后端扩展 |
| 最终位置 | `position`（StateSync.Players） | 游戏结束时地图位置 | ❌ 需后端扩展 |
| 最终HP/LP | `hp`、`lp`（StateSync.Players） | 游戏结束时状态 | ❌ 需后端扩展 |
| 总回合数 | 游戏级数据 | 游戏进行了多少轮 | ❌ 需后端扩展 |

#### 前端可直接渲染的增强统计

以下统计项前端可从现有协议数据直接计算，无需后端修改：

- **成就数目**：`achievements.length`
- **积分明细**：`score_reasons` 渲染为可展开的积分明细列表（每条 reason 展示 category + reason + points + round）
- **积分分类占比饼图**：`score_breakdown` 渲染为饼图
- **Boss伤害 vs 积分对比**：`boss_damage_dealt`（原始伤害）与 `boss_score`（积分）双列对比

#### 需要后端扩展的统计项

若要展示死亡次数、最终位置、最终HP/LP、总回合数，需要在 `PlayerStats` 或 `GameOver` 结构中新增字段。详见第六节。

### 5.3 成就展示区

底部或独立区块展示所有玩家的成就汇总：

- 每个成就一行，展示：成就图标 + 成就名称 + 描述 + 达成者（头像+名称）
- 使用客户端本地缓存的 `AchievementDefinition`（name, desc, points）渲染
- 成就定义来源：
  - 方案A：在 `StartGameAck.Definitions` 中增加 `achievements` 字段，后端在开局时发送成就定义（推荐）
  - 方案B：前端硬编码成就元数据（简单但不够灵活）

---

## 六、后端协议扩展建议

### 6.1 PlayerStats 新增字段

```go
// PlayerStats 新增字段建议
type PlayerStats struct {
    // ... 现有字段 ...

    // 新增：死亡次数
    DeathCount int `json:"death_count"`

    // 新增：最终HP（游戏结束时）
    FinalHP int `json:"final_hp"`

    // 新增：最终LP（游戏结束时）
    FinalLP int `json:"final_lp"`

    // 新增：最终地图位置
    FinalPosition int `json:"final_position"`

    // 新增：阵营（用于结算页渲染）
    Faction string `json:"faction"`
}
```

### 6.2 GameOver 新增字段

```go
// GameOver 顶层新增字段建议
type GameOver struct {
    Rankings []PlayerRanking `json:"rankings"`
    Stats    []PlayerStats   `json:"stats"`

    // 新增：总回合数
    TotalRounds int `json:"total_rounds"`

    // 新增：Boss是否被击败
    BossDefeated bool `json:"boss_defeated"`

    // 新增：击败Boss的玩家ID（空则Boss未被击败）
    BossDefeatedBy string `json:"boss_defeated_by,omitempty"`
}
```

### 6.3 StartGameAck 扩展成就定义

```go
// DefinitionsConfig 新增成就定义
type DefinitionsConfig struct {
    Events    map[string]EventDefinitionConfig    `json:"events"`
    Buffs     map[string]BuffDefinitionConfig     `json:"buffs"`
    Items     map[string]ItemDefinitionConfig     `json:"items"`
    MiniGames map[string]MiniGameDefinitionConfig `json:"mini_games"`

    // 新增：成就定义
    Achievements map[string]AchievementDefinitionConfig `json:"achievements"`
}

// AchievementDefinitionConfig 成就定义配置
type AchievementDefinitionConfig struct {
    Type        string `json:"type"`
    EnglishName string `json:"english_name"`
    Name        string `json:"name"`   // 中文显示名
    Desc        string `json:"desc"`   // 中文描述
    Points      int    `json:"points"` // 加分值
}
```

### 6.4 PlayerRanking 新增阵营字段

```go
// PlayerRanking 新增字段
type PlayerRanking struct {
    // ... 现有字段 ...

    // 新增：阵营（用于动画中的阵营色彩主题）
    Faction string `json:"faction"`
}
```

---

## 七、前端组件设计

### 7.1 新增组件清单

| 组件名 | 功能 | 说明 |
|--------|------|------|
| `GameOverAnimation` | Phase 1+2 动画控制器 | 管理逐名揭示 + 冠军特写的时序 |
| `RankRevealCard` | 单人揭示卡片 | 头像、阵营、排名、分数滚动、成就徽章 |
| `ScoreCounter` | 分数滚动数字 | 从0到total_score的动画数字，带浮动reason标签 |
| `CategoryProgressBar` | 分类进度条 | 堆叠条形图，按mini_game/boss/item/achievement占比 |
| `AchievementBadge` | 成就徽章 | 图标+名称的小徽章组件，弹出动画 |
| `ChampionSpotlight` | 冠军特写 | 金色光晕、阵营粒子、大字标题 |
| `GameOverFullPage` | Phase 3 结算页 | 排名列表、统计表、成就展示、操作按钮 |
| `ScoreBreakdownChart` | 积分占比饼图 | 4类积分的占比可视化 |
| `StatsTable` | 统计数据表 | 增强版统计表格（含成就数目、死亡次数等） |

### 7.2 动画状态管理

使用状态机管理动画流程：

```
States:
  IDLE          → 初始状态，收到 OpGameOver 消息后进入
  RANK_REVEAL   → 逐名揭示阶段，currentRevealIndex 从 rankings.length-1 递减到 0
  CHAMPION      → 冠军特写阶段
  FULL_PAGE     → 全局结算页阶段（常驻）

Transitions:
  IDLE → RANK_REVEAL      : 收到 GameOver 数据
  RANK_REVEAL → RANK_REVEAL : 当前玩家揭示完成，揭示下一个
  RANK_REVEAL → CHAMPION    : 所有玩家揭示完成（最后一个=冠军）
  CHAMPION → FULL_PAGE     : 冠军特写动画完成
```

### 7.3 跳过机制

- 用户可点击"跳过动画"按钮，直接进入 `FULL_PAGE` 状态
- 跳过按钮在 Phase 1 开始时即显示在角落，半透明避免遮挡主画面

---

## 八、分数滚动动画的算法伪代码

```typescript
interface ScoreReason {
  category: string;   // "mini_game" | "boss" | "item" | "achievement"
  reason: string;     // "小游戏第1名", "Boss伤害", etc.
  points: number;     // 加分值
  round: number;      // 轮次
}

interface AnimationStep {
  startScore: number;
  endScore: number;
  reason: ScoreReason;
  duration: number;   // ms
}

// 将 score_reasons 转换为动画步骤序列
function buildAnimationSteps(scoreReasons: ScoreReason[]): AnimationStep[] {
  const avgPoints = scoreReasons.reduce((s, r) => s + r.points, 0) / scoreReasons.length;
  const BASE_TIME = 300; // ms per reason

  let currentScore = 0;
  return scoreReasons.map((reason) => {
    const startScore = currentScore;
    const endScore = currentScore + reason.points;
    // Larger points → longer duration for visual emphasis
    const duration = BASE_TIME * (1 + Math.log2(Math.max(1, reason.points / avgPoints)));

    currentScore = endScore;
    return { startScore, endScore, reason, duration };
  });
}

// 每个步骤执行：
// 1. 数字从 startScore 滚动到 endScore（duration ms）
// 2. 浮动标签显示 reason.reason + "+N"（淡入→停留→淡出）
// 3. CategoryProgressBar 对应类别段追加 reason.points 长度
```

---

## 九、色彩与视觉设计建议

### 9.1 阵营主题色

| 阵营 | 主色 | 辅色 | 动画元素建议 |
|------|------|------|-------------|
| 青龙 (qing_long) | #00CED1 (青色) | #00796B (深青) | 龙形粒子、水波纹背景 |
| 朱雀 (zhu_que) | #E53935 (红色) | #B71C1C (深红) | 火焰粒子、热浪效果 |
| 白虎 (bai_hu) | #FFFFFF (白色) | #9E9E9E (灰) | 冰晶粒子、锐利线条 |
| 玄武 (xuan_wu) | #5C6BC0 (蓝紫) | #283593 (深蓝紫) | 盾形粒子、厚重感 |

### 9.2 排名位置视觉

| 排名 | 视觉强度 | 建议 |
|------|---------|------|
| 第1名 | 金色 (#FFD700)，大字号，粒子光环 | 冠军专属特效 |
| 第2名 | 银色 (#C0C0C0)，中字号 | 简洁但清晰 |
| 第3名 | 铜色 (#CD7F32)，常规字号 | 正常揭示 |
| 第4名 | 灰色 (#808080)，常规字号 | 淡淡揭示 |

### 9.3 背景

- Phase 1+2：深色背景（#1A1A2E），配合揭示玩家阵营色的环境光渐变
- Phase 3：稍亮背景（#2D2D44），数据可读性优先

---

## 十、前端数据依赖总结

### 10.1 现有协议数据（已可用）

| 数据 | 来源 | 动画用途 |
|------|------|---------|
| `rankings` | `GameOver.rankings` | 逐名揭示顺序、排名、总分 |
| `score_reasons` | `PlayerRanking.score_reasons` | 分数滚动逐条累加 |
| `achievements` | `PlayerRanking.achievements` | 成就徽章展示 |
| `mini_game_score` 等 | `PlayerRanking` 四个分类子分数 | 分类进度条渲染 |
| `stats` | `GameOver.stats` | Phase 3 统计表 |
| `score_breakdown` | `PlayerStats.score_breakdown` | 饼图渲染 |
| `boss_damage_dealt` | `PlayerStats.boss_damage_dealt` | Boss伤害统计 |

### 10.2 需新增的协议数据

| 数据 | 建议来源 | 用途 | 优先级 |
|------|---------|------|--------|
| `faction` | `PlayerRanking` + `PlayerStats` 新增字段 | 阵营色彩主题、阵营图标 | **P0 必须** |
| `achievements` 定义 | `StartGameAck.Definitions` 新增 | 成就名称/描述渲染 | **P0 必须** |
| `death_count` | `PlayerStats` 新增字段 | 统计表死亡次数列 | P1 推荐 |
| `final_hp` / `final_lp` | `PlayerStats` 新增字段 | 统计表最终状态 | P1 推荐 |
| `final_position` | `PlayerStats` 新增字段 | 统计表最终位置 | P2 可选 |
| `total_rounds` | `GameOver` 新增字段 | 统计表总轮次 | P2 可选 |
| `boss_defeated` / `boss_defeated_by` | `GameOver` 新增字段 | Boss击败信息展示 | P2 可选 |

### 10.3 前端本地计算数据

| 数据 | 计算方式 | 用途 |
|------|---------|------|
| 成就数目 | `achievements.length` | 统计表成就数列 |
| 积分明细列表 | 渲染 `score_reasons` 为可展开列表 | 积分详情 |
| 积分占比 | `score_breakdown` 饼图 | 分类占比可视化 |

---

## 十一、实现优先级与里程碑

### Milestone 1：基础动画（Phase 1 核心）

- 实现 `GameOverAnimation` 状态机
- 实现 `RankRevealCard` 入场 + 退场动画
- 实现 `ScoreCounter` 分数从0到total_score滚动（先实现简单版：一次性滚动到总分，不逐条）
- 实现跳过按钮

### Milestone 2：分数滚动增强

- 实现 `score_reasons` 逐条累加滚动
- 实现浮动 reason 标签淡入淡出
- 实现 `CategoryProgressBar` 堆叠进度条
- 阵营色彩主题映射

### Milestone 3：冠军特写 + 成就

- 实现 `ChampionSpotlight` 动画（Phase 2）
- 实现 `AchievementBadge` 弹出动画
- 成就定义协议扩展（`StartGameAck.Definitions.achievements`）

### Milestone 4：全局结算页

- 实现 `GameOverFullPage` 常驻页面
- 实现 `ScoreBreakdownChart` 饼图
- 实现 `StatsTable` 增强版统计表
- 后端协议扩展（`faction`, `death_count`, `final_hp/lp`, `total_rounds` 等）
- 成就展示区块

---

## 十二、注意事项

1. **动画性能**：分数滚动使用 `requestAnimationFrame` 驱动，避免 `setTimeout` 导致的帧率不稳定
2. **数据就绪时机**：`GameOver` 消息一次性包含所有数据，前端收到后即可启动动画，无需额外请求
3. **断线重连**：若玩家在动画期间断线重连，收到 `OpFullSync (8)` 后应直接跳到 `FULL_PAGE` 状态，不重播动画
4. **2人局适配**：揭示人数 = `rankings.length`，动画时长动态调整
5. **Boss玩家排除**：`rankings` 不包含 Boss 玩家，动画和结算页只展示非Boss排名；`stats` 包含 Boss 但结算页可将其放在独立区域或折叠
6. **成就定义来源**：推荐在 `StartGameAck` 中发送 `achievements` 定义，避免前端硬编码。若暂不修改后端，前端可先用本地 JSON 映射表作为过渡
7. **阵营数据缺失**：当前 `PlayerRanking` 和 `PlayerStats` 不包含 `faction` 字段。动画需要阵营色彩主题，这是 **P0 必须** 的扩展。过渡方案：前端可从 `OpFullSync/OpStateSync` 的 `players` 数组中按 `player_id` 匹配获取 `faction`，但这依赖客户端缓存的最后一次 StateSync 数据，不够可靠