# pkg/constants - Unified Enum Types

统一枚举类型定义包，所有用于 JSON 通信的枚举类型集中于此。

## 设计原则

1. **类型定义**：使用 `string` 类型（`Evaluation` 除外，使用 `int`）
2. **命名规范**：常量名 PascalCase（Go 规范），值 snake_case（通信规范）
3. **类型安全**：每个枚举是独立类型，不会混淆
4. **JSON 友好**：直接序列化为 snake_case，无需转换
5. **零依赖**：包不依赖任何其他包，可被任何包安全导入

## 包结构

```
pkg/constants/
├── phase.go        # Phase - 触发时机
├── buff.go         # BuffType - Buff 类型
├── event.go        # EventType - 事件类型
├── item.go         # ItemType - 道具类型
├── effect.go       # SpecialEffect - 特殊效果
├── faction.go      # Faction - 四神兽阵营
├── cell.go         # CellType - 地图格子类型
├── state.go        # StateID - HSM 状态标识
├── entry.go        # EntryType - GameLog 条目类型
├── source.go       # ActionSource - Action 来源标识
├── evaluation.go   # Evaluation - 评分系统（int 类型）
├── constants_test.go # 单元测试
└── README.md       # 本文档
```

## 类型列表

### BuffType - Buff 类型标识

提供 `IsValid()`、`IsPositive()`、`IsNegative()` 方法。

| 常量 | 值 | 分类 | 说明 |
|------|-----|------|------|
| `BuffTypeDivine` | `divine` | Positive | 神眷：LP+1/回合 |
| `BuffTypeRain` | `rain` | Positive | 甘霖：HP+1/2回合 |
| `BuffTypeExorcism` | `exorcism` | Positive | 辟邪：免疫毒瘴 |
| `BuffTypeHidden` | `hidden` | Neutral | 隐匿：免疫伤害/事件 |
| `BuffTypeFire` | `fire` | Positive | 离火：朱雀被动 |
| `BuffTypeCurse` | `curse` | Negative | 诅咒：LP-1/回合 |
| `BuffTypeLost` | `lost` | Negative | 迷途：反向移动 |
| `BuffTypeCorrupt` | `corrupt` | Negative | 腐化：HP-1/2回合 |
| `BuffTypePoison` | `poison` | Negative | 毒瘴：每回合恶性事件 |

### EventType - Event 类型标识

提供 `IsValid()` 方法。

| 分类 | 常量 | 值 | 说明 |
|------|------|-----|------|
| Good | `EventTypeHerb` | `herb` | 草药：HP+1 |
| Good | `EventTypeMilkTea` | `milk_tea` | 奶茶：LP+1 |
| Good | `EventTypeRelic` | `relic` | 圣遗物：抽道具 |
| Good | `EventTypeDivineBless` | `divine_bless` | 天使眷顾：神眷Buff |
| Neutral | `EventTypeExchange` | `exchange` | 交换：换位 |
| Neutral | `EventTypeHiddenBuff` | `hidden_buff` | 隐匿Buff |
| Neutral | `EventTypeTasteTest` | `taste_test` | 嘗一口：随机Buff |
| Bad | `EventTypeMosquito` | `mosquito` | 蚊虫：HP-1 |
| Bad | `EventTypeGhostHit` | `ghost_hit` | 野鬼：HP-1 |
| Bad | `EventTypeDogPoop` | `dog_poop` | 狗屎：LP-1 |
| Bad | `EventTypeThief` | `thief` | 盗贼：失道具 |
| Bad | `EventTypeCurseBuddha` | `curse_buddha` | 野佛：诅咒Buff |
| Bad | `EventTypeLostWay` | `lost_way` | 迷途：迷途Buff |
| Bad | `EventTypeThunder` | `thunder` | 雷劫：HP=0 |

### ItemType - Item 类型标识

提供 `IsValid()` 方法。

| 常量 | 值 | 说明 |
|------|-----|------|
| `ItemTypeReverseClock` | `reverse_clock` | 反方向的钟：给目标迷途Buff |
| `ItemTypeAnyDoor` | `any_door` | 任意门：传送 |
| `ItemTypeDiceSwap` | `dice_swap` | 骰子交换：交换骰子 |
| `ItemTypeDiceUpgrade` | `dice_upgrade` | 骰子升级：升级骰子 |

### Phase - 触发时机

**设计原则：谁产生时机，谁发布 Phase**

提供 `IsValid()`、`NeedsSubscription()`、`IsHSMPublished()`、`IsActionPublished()` 方法。

| 发布者 | 常量 | 值 | 说明 |
|--------|------|-----|------|
| HSM | `PhaseBeforeTurn` | `before_turn` | 回合开始前 |
| HSM | `PhaseOnLand` | `on_land` | 落地后 |
| HSM | `PhaseAfterTurn` | `after_turn` | 回合结束后 |
| Action | `PhasePreDamage` | `pre_damage` | 伤害应用前 |
| Action | `PhasePreEvent` | `pre_event` | 事件触发前 |
| Action | `PhasePreMove` | `pre_move` | 移动前 |
| Action | `PhasePreRespawn` | `pre_respawn` | 重生前（可拦截） |
| Action | `PhaseOnBuffApplied` | `on_buff_applied` | Buff应用后 |
| Action | `PhaseOnBuffRemoved` | `on_buff_removed` | Buff移除前 |
| Special | `PhaseAnyTime` | `any_time` | 任意时刻（手动触发，不订阅） |
| Special | `PhaseItemUsed` | `item_used` | 道具主动使用 |

### Faction - 四神兽阵营

提供 `IsValid()`、`AllFactions()` 方法。

| 常量 | 值 | 技能名 | 说明 |
|------|-----|--------|------|
| `FactionQingLong` | `qing_long` | 行迹 | 每5回合蓄力，忽略负面地形 |
| `FactionZhuQue` | `zhu_que` | 离火 | 每4回合LP+1 |
| `FactionBaiHu` | `bai_hu` | 劫运 | 超越他人时偷Buff |
| `FactionXuanWu` | `xuan_wu` | 鎮厄 | 每5回合蓄力，取消恶性事件 |

### CellType - 地图格子类型

提供 `IsValid()`、`IsSpecial()` 方法。

| 常量 | 值 | 说明 |
|------|-----|------|
| `CellTypeNormal` | `normal` | 普通格子 |
| `CellTypeFragile` | `fragile` | 易碎格子（穿透） |
| `CellTypeFog` | `fog` | 迷雾格子（毒瘴区域） |
| `CellTypeCheckpoint` | `checkpoint` | 检查点（重生点） |
| `CellTypeBoss` | `boss` | Boss格子（终点） |

### StateID - HSM 状态标识

三层状态机结构，提供 `IsValid()`、`IsGlobalState()`、`IsTurnState()`、`IsInterruptState()`、`Layer()` 方法。

| 层级 | 常量 | 值 | 说明 |
|------|------|-----|------|
| Layer 1 | `StateMatchInit` | `match_init` | 对局初始化 |
| Layer 1 | `StateRoundMiniGame` | `round_mini_game` | 小游戏回合 |
| Layer 1 | `StateRoundPrep` | `round_prep` | 回合准备 |
| Layer 1 | `StateTurnLoop` | `turn_loop` | 行动循环 |
| Layer 1 | `StateBossBattle` | `boss_battle` | Boss战 |
| Layer 1 | `StateGameOver` | `game_over` | 游戏结束 |
| Layer 2 | `StateTurnUpkeep` | `turn_upkeep` | 回合维护 |
| Layer 2 | `StateMainAction` | `main_action` | 主要行动 |
| Layer 2 | `StateTurnMoving` | `turn_moving` | 移动中 |
| Layer 2 | `StateTurnLanded` | `turn_landed` | 落地处理 |
| Layer 2 | `StateTurnEvent` | `turn_event` | 事件触发 |
| Layer 2 | `StateTurnEnd` | `turn_end` | 回合结束 |
| Layer 3 | `StateWaitDecision` | `wait_decision` | 等待决策 |

### EntryType - GameLog 条目类型

提供 `IsValid()` 方法。

| 常量 | 值 | 说明 |
|------|-----|------|
| `EntryTypeAction` | `action` | Action 执行记录 |
| `EntryTypeState` | `state` | HSM 状态转换记录 |
| `EntryTypeMiniGame` | `mini_game` | 小游戏结果记录 |
| `EntryTypeBoss` | `boss` | Boss 战记录 |
| `EntryTypeDecision` | `decision` | 用户决策记录 |

### ActionSource - Action 来源标识

提供 `IsValid()`、`IsBuff()`、`IsItem()`、`IsEvent()`、`IsFaction()`、`IsSystem()` 方法。

通过前缀判断来源类型：
- `buff_*` - Buff 来源
- `item_*` - Item 来源
- `event_*` - Event 来源
- `faction_*` - Faction 来源
- `system_*` + 特殊值 - System 来源

### SpecialEffect - 特殊效果类型

提供 `IsValid()`、`IsBuffEffect()`、`IsItemEffect()`、`IsEventEffect()` 方法。

用于标识 Buff/Item/Event 的特殊行为（非简单 HP/LP 变化）。

### Evaluation - 评分系统

使用 `int` 类型（非 string），便于数值比较。

提供 `IsValid()`、`GetCategory()`、`IsGood()`、`IsNeutral()`、`IsBad()`、`Compare()` 方法。

| 预定义常量 | 值 | 分类 |
|-----------|-----|------|
| `EvaluationVeryBad` | 10 | Bad (≤40) |
| `EvaluationBad` | 25 | Bad (≤40) |
| `EvaluationMildBad` | 35 | Bad (≤40) |
| `EvaluationNeutral` | 50 | Neutral (41-65) |
| `EvaluationMixed` | 55 | Neutral (41-65) |
| `EvaluationMildGood` | 70 | Good (>65) |
| `EvaluationGood` | 80 | Good (>65) |
| `EvaluationVeryGood` | 90 | Good (>65) |
| `EvaluationExcellent` | 100 | Good (>65) |

阈值常量：
- `EvaluationBadThreshold = 40`：恶性上限
- `EvaluationNeutralThreshold = 65`：中性上限
- `EvaluationMin = 0`、`EvaluationMax = 100`：范围边界

## 迁移状态

| 类型 | 原位置 | 新位置 | 状态 |
|------|--------|--------|------|
| Phase | pkg/event | pkg/constants | ✓ 已完成 |
| BuffType | internal/core/buff | pkg/constants | ✓ 已完成 |
| EventType | internal/core/event | pkg/constants | ✓ 已完成 |
| ItemType | internal/core/item | pkg/constants | ✓ 已完成 |
| SpecialEffect | internal/core/types | pkg/constants | ✓ 已完成 |
| Faction | internal/core | pkg/constants | ✓ 已完成 |
| CellType | internal/gamemap | pkg/constants | ✓ 已完成 |
| StateID | internal/engine/hsm | pkg/constants | ✓ 已完成 |
| EntryType | pkg/gamelog | pkg/constants (alias) | ✓ 已完成 |
| Evaluation | internal/core/types | pkg/constants | ✓ 已完成 |
| ActionSource | - | pkg/constants | ✓ 已添加 |

## Alias 使用

其他包通过 type alias 引用 constants 类型，保持 API 兼容性：

```go
// pkg/event/context.go
type Phase = constants.Phase

// pkg/gamelog/entry.go
type EntryType = constants.EntryType
const (
    EntryTypeAction = constants.EntryTypeAction
    // ...
)
```

## 使用示例

```go
import "github.com/b1tAction/Fated/pkg/constants"

// 检查 Buff 类型有效性
bt := constants.BuffTypeDivine
if bt.IsValid() && bt.IsPositive() {
    // 处理正向 Buff
}

// 检查 Phase 发布者
phase := constants.PhaseBeforeTurn
if phase.IsHSMPublished() {
    // HSM 状态触发
}

// 检查 Evaluation 分类
eval := constants.EvaluationGood
if eval.IsGood() {
    // 处理良性评分
}
category := eval.GetCategory() // "good"

// 获取所有阵营
factions := constants.AllFactions()
for _, f := range factions {
    fmt.Println(f) // qing_long, zhu_que, bai_hu, xuan_wu
}
```