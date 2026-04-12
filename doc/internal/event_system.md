# Event & Buff 属性评分系统实现文档

## 概述

Evaluation 属性评分系统是《命运骰子》游戏的事件分类框架，为所有事件、Buff 和道具提供统一的评分系统（0~100），量化"有多好/有多坏"，便于 RNG 抽卡和游戏逻辑处理。

## 文件结构

```
internal/game/
├── faction.go   # 阵营定义 + 被动技能描述
├── buff.go      # Evaluation 系统 + Buff 系统（类型 + 定义 + 注册表）
├── event.go     # Event 系统（类型 + 定义 + 注册表）
├── item.go      # Item 系统（类型 + 定义 + 注册表）
├── buff_test.go # Buff 和 Evaluation 单元测试
├── item_test.go # Item 单元测试
└── event_test.go # Event 单元测试
```

每个文件专注于一个领域，包含：
- 类型枚举定义
- 实例结构体
- 静态定义（Definition）
- 注册表（Registry）

## Evaluation 评分系统

### 评分范围

```go
type Evaluation int  // 0~100

const (
    // 评分范围常量
    EvaluationMin    Evaluation = 0   // 最低评分
    EvaluationMax    Evaluation = 100 // 最高评分

    // 分类边界
    EvaluationBadThreshold     Evaluation = 40  // 恶性上限（≤40）
    EvaluationNeutralThreshold Evaluation = 65  // 中性上限（≤65）
    // Evaluation > 65 为良性
)
```

| 评分范围 | 类别 | 说明 |
|---------|------|------|
| 0~40 | Bad（恶性） | 负面效果 |
| 41~65 | Neutral（中性） | 混合/随机效果 |
| 66~100 | Good（良性） | 正面效果 |

### 预定义评分常量

```go
const (
    // 恶性评分（0~40）
    EvaluationVeryBad   Evaluation = 10  // 极恶（如雷劫）
    EvaluationBad       Evaluation = 25  // 较恶（如诅咒）
    EvaluationMildBad   Evaluation = 35  // 轻恶（如蚊虫叮咬）

    // 中性评分（41~65）
    EvaluationNeutral   Evaluation = 50  // 标准中性（如交换）
    EvaluationMixed     Evaluation = 55  // 混合效果（如尝一口）

    // 良性评分（66~100）
    EvaluationMildGood  Evaluation = 70  // 轻良（如草药）
    EvaluationGood      Evaluation = 80  // 较良（如奶茶）
    EvaluationVeryGood  Evaluation = 90  // 极良（如圣遗物）
    EvaluationExcellent Evaluation = 100 // 最佳（如神眷）
)
```

### Evaluation 方法

```go
func (e Evaluation) IsValid() bool       // 检查评分是否在 0~100 范围内
func (e Evaluation) GetCategory() string // 返回 "Bad"/"Neutral"/"Good"
func (e Evaluation) IsGood() bool        // > 65
func (e Evaluation) IsNeutral() bool     // > 40 && ≤ 65
func (e Evaluation) IsBad() bool         // ≤ 40
func (e Evaluation) Compare(other Evaluation) int // 比较评分
```

### 旧版兼容

```go
// 旧版 EventAttribute 保留用于向后兼容
type EventAttribute int

const (
    AttributeGood     EventAttribute = iota
    AttributeNeutral
    AttributeBad
)

// 转换方法
func (ea EventAttribute) ToEvaluation() Evaluation
```

## 事件评分对照表

### EventType (事件类型枚举)

**良性事件 (Evaluation > 65)**：
| 类型 | 名称 | 效果 | 评分 |
|------|------|------|------|
| Herb | 采集到草药 | HP+1 | 70 (轻良) |
| MilkTea | 捡到奶茶 | LP+1 | 80 (较良) |
| Relic | 捡到勇士的圣遗物 | 道具抽奖 | 90 (极良) |
| DivineBless | 受到天使眷顾 | 神眷Buff | 100 (最佳) |
| HiddenBuff | 麻了 | 隐匿Buff | 80 (较良) |

**中性事件 (Evaluation 41~65)**：
| 类型 | 名称 | 效果 | 评分 |
|------|------|------|------|
| Exchange | 交换 | 与随机玩家交换位置 | 50 (中性) |
| TasteTest | 这是什么？尝一口 | 腐化/甘霖Buff（随机） | 55 (混合) |

**恶性事件 (Evaluation ≤ 40)**：
| 类型 | 名称 | 效果 | 评分 |
|------|------|------|------|
| Mosquito | 被蚊虫叮咬 | HP-1 | 35 (轻恶) |
| GhostHit | 偶遇孤魂野鬼 | HP-1 | 35 (轻恶) |
| DogPoop | 踩到了狗屎 | LP-1 | 35 (轻恶) |
| LostWay | 迷途 | 迷途Buff | 35 (轻恶) |
| Thief | 啊？！贼 | 随机丢失道具 | 25 (较恶) |
| CurseBuddha | 虔诚拜三拜 | 诅咒Buff | 25 (较恶) |
| Thunder | 雷劫 | HP归零 | 10 (极恶) |

### EventDefinition (事件定义)

```go
type EventDefinition struct {
    Type       EventType   // 事件类型
    Eval       Evaluation  // 属性评分
    Name       string      // 事件名称（中文）
    Desc       string      // 事件描述
    HPChange   int         // HP变化值
    LPChange   int         // LP变化值
    BuffType   BuffType    // 获得的Buff类型
    ItemAction string      // 道具行为（gain/lose/draw）
}
```

## Buff 评分对照表

**良性 Buff (Evaluation > 65)**：
| 类型 | 名称 | 效果 | 持续 | 评分 |
|------|------|------|------|------|
| Divine | 神眷 | 每回合LP+1 | 3 | 90 (极良) |
| Hidden | 隐匿 | 免疫任意事件/BUFF/道具 | 3 | 100 (最佳) |
| Rain | 甘霖 | 每2回合HP+1 | 4 | 80 (较良) |
| Exorcism | 辟邪 | 无视毒瘴buff | 5 | 70 (轻良) |
| Fire | 离火 | 每4回合LP+1 | 永久 | 80 (较良) |

**恶性 Buff (Evaluation ≤ 40)**：
| 类型 | 名称 | 效果 | 持续 | 评分 |
|------|------|------|------|------|
| Curse | 诅咒 | 每回合LP-1 | 3 | 25 (较恶) |
| Lost | 迷途 | 反方向移动 | 1 | 35 (轻恶) |
| Corrupt | 腐化 | 每2回合HP-1 | 4 | 25 (较恶) |
| Poison | 毒瘴 | 每回合恶性随机事件 | 3 | 10 (极恶) |

### BuffDefinition (Buff定义)

```go
type BuffDefinition struct {
    Type      BuffType   // Buff类型
    Eval      Evaluation // 属性评分
    Name      string     // Buff名称（中文）
    Desc      string     // Buff描述
    Duration  int        // 持续回合数（-1表示永久）
    HPPerTurn int        // 每回合HP变化
    LPPerTurn int        // 每回合LP变化
    Special   string     // 特殊效果描述
}
```

## 道具评分对照表

**良性道具 (Evaluation > 65)**：
| 类型 | 名称 | 效果 | 评分 |
|------|------|------|------|
| DiceUpgrade | 骰子升级卡 | 骰子升级 | 80 (较良) |

**中性道具 (Evaluation 41~65)**：
| 类型 | 名称 | 效果 | 评分 |
|------|------|------|------|
| AnyDoor | 任意门 | 去到30格内指定玩家身边 | 50 (中性) |
| DiceSwap | 骰子交换 | 与指定玩家交换骰子等级 | 50 (中性) |

**恶性道具 (Evaluation ≤ 40)**：
| 类型 | 名称 | 效果 | 评分 |
|------|------|------|------|
| ReverseClock | 反方向的钟 | 给予指定玩家迷途Buff | 25 (较恶) |

### ItemDefinition (道具定义)

```go
type ItemDefinition struct {
    Type        ItemType   // 道具类型
    Eval        Evaluation // 属性评分
    Name        string     // 道具名称（中文）
    Desc        string     // 道具描述
    TargetSelf  bool       // 是否对自己使用
    TargetOther bool       // 是否对他人使用
    BuffType    BuffType   // 赋予的Buff类型
    Range       int        // 有效范围
}
```

## 核心功能

### 1. 获取事件评分

```go
event := EventTypeHerb
eval := event.GetEvaluation()  // 返回 70

if eval.IsGood() {
    fmt.Println("这是良性事件")
}

def := event.GetEventDefinition()
fmt.Println(def.Name)    // "采集到草药"
fmt.Println(def.HPChange) // 1
```

### 2. 按 Evaluation 范围筛选

```go
registry := NewEventRegistry()

// 获取评分在 90~100 的事件（极良）
excellentEvents := registry.GetEventsByEvaluationRange(90, 100)
// 返回: [EventTypeRelic, EventTypeDivineBless]

// 获取评分在 30~40 的事件（轻恶）
mildBadEvents := registry.GetEventsByEvaluationRange(30, 40)
// 返回: [EventTypeMosquito, EventTypeGhostHit, EventTypeDogPoop, EventTypeLostWay]

// 获取评分在 0~15 的事件（极恶）
veryBadEvents := registry.GetEventsByEvaluationRange(0, 15)
// 返回: [EventTypeThunder]
```

### 3. 按类别获取

```go
registry := NewEventRegistry()

good := registry.GetEventsByCategory("Good")
// 返回: [Herb, MilkTea, Relic, DivineBless, HiddenBuff] (5个)

neutral := registry.GetEventsByCategory("Neutral")
// 返回: [Exchange, TasteTest] (2个)

bad := registry.GetEventsByCategory("Bad")
// 返回: [Mosquito, GhostHit, DogPoop, Thief, CurseBuddha, LostWay, Thunder] (7个)
```

### 4. Buff 注册表

```go
registry := NewBuffRegistry()

// 按评分范围获取
goodBuffs := registry.GetBuffsByEvaluationRange(66, 100)
badBuffs := registry.GetBuffsByEvaluationRange(0, 40)

// 按类别获取
goodBuffs := registry.GetBuffsByCategory("Good")  // 5个
badBuffs := registry.GetBuffsByCategory("Bad")    // 4个
```

### 5. Item 注册表

```go
registry := NewItemRegistry()

// 按评分范围获取
goodItems := registry.GetItemsByEvaluationRange(66, 100)  // [DiceUpgrade]
neutralItems := registry.GetItemsByEvaluationRange(41, 65) // [AnyDoor, DiceSwap]
badItems := registry.GetItemsByEvaluationRange(0, 40)     // [ReverseClock]
```

## RNG 集成

### 幸运值对评分概率的影响

| 幸运值 | 良性概率(66~100) | 恶性概率(0~40) | 中性概率(41~65) |
|--------|-----------------|---------------|----------------|
| 0 | 30% | 30% | 40% |
| 5 | 55% | 5% | 40% |
| 8 | 70% | 10% | 40% |

计算公式：
- `goodProb = 30 + luck*5`
- `badProb = 30 - luck*5`
- `neutralProb = 40`（固定）

### 细分评分权重

可以根据具体评分值设置权重，实现更精细的控制：

```go
// 雷劫(10) 权重更低，神眷(100) 权重更高
weights := map[EventType]int{
    EventTypeThunder:      1,  // 极恶事件权重低
    EventTypeMosquito:     10, // 轻恶事件权重高
    EventTypeHerb:         10, // 轻良事件权重高
    EventTypeDivineBless:  5,  // 极良事件权重低
}
```

## 与其他模块协作

### PlayerSystem 集成

```go
player := NewPlayer(PlayerConfig{UserID: "test"})

// 检查 Buff 评分决定是否免疫
if buff.GetEvaluation().IsBad() && player.HasBuff(BuffTypeHidden) {
    // 隐匿状态下免疫负面 Buff
}

// 检查事件评分触发阵营被动
if event.GetEvaluation().IsBad() && player.Faction == FactionXuanWu {
    // 玄武可以抵消恶性事件
    player.TriggerFactionSkill(event)
}
```

### 游戏流程集成

```go
// State_Turn_Landed 落点事件结算
func (s *StateManager) processLandedEvent(player *Player) {
    event := s.drawEventBasedOnLuck(player.LP)

    // 玄武抵消恶性事件
    if event.GetEvaluation().IsBad() {
        ev := &GameEvent{Type: EventPreBadEvent}
        player.DispatchEvent(ev)
        if ev.IsCancel {
            return // 事件被抵消
        }
    }

    // 执行事件效果
    def := event.GetEventDefinition()
    player.HP += def.HPChange
    player.LP += def.LPChange
    if def.BuffType != BuffTypeNone {
        player.AddBuff(NewBuff(def.BuffType, def.Duration))
    }
}
```

## 测试覆盖

| 测试文件 | 覆盖内容 |
|---------|----------|
| buff_test.go | Evaluation 有效性、类别判断、比较方法、常量验证 |
| buff_test.go | Buff 类型名称、评分获取、定义获取 |
| buff_test.go | BuffRegistry 创建、按评分/类别筛选 |
| event_test.go | EventType 名称、评分获取、类别获取 |
| event_test.go | EventDefinition 一致性验证 |
| event_test.go | EventRegistry 创建、按评分/类别筛选 |
| item_test.go | ItemType 名称、评分获取、类别获取 |
| item_test.go | Item 实例创建、字段测试 |
| item_test.go | ItemDefinition 获取、评分一致性 |
| item_test.go | ItemRegistry 创建、按评分/类别筛选 |

## 后续扩展

1. **评分动态调整**：根据游戏阶段调整评分权重（后期恶性事件权重增加）
2. **评分阈值配置**：允许通过配置文件修改 Bad/Neutral 边界值
3. **评分影响链**：某些 Buff 可以临时改变事件评分（如"幸运"Buff）
4. **评分可视化**：在 UI 中显示评分颜色（红/黄/绿）

## 文件统计

```
internal/game/
├── faction.go      # 阵营定义（~100行）
├── buff.go         # Evaluation + Buff 系统（~360行）
├── event.go        # Event 系统（~290行）
├── item.go         # Item 系统（~200行）
├── buff_test.go    # Buff 和 Evaluation 测试（~300行）
├── item_test.go    # Item 测试（~330行）
├── event_test.go   # Event 测试（~330行）
└── ...
```

模块组成：
| 文件 | 类型枚举 | 实例结构 | Definition | Registry | GetEvaluation |
|------|---------|---------|-----------|----------|---------------|
| faction.go | Faction | - | - | - | - |
| buff.go | BuffType | Buff | BuffDefinition | BuffRegistry | ✓ |
| event.go | EventType | - | EventDefinition | EventRegistry | ✓ |
| item.go | ItemType | Item | ItemDefinition | ItemRegistry | ✓ |