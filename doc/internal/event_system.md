# Event & Buff 属性系统实现文档

## 概述

Event 属性系统是《命运骰子》游戏的事件分类框架，为所有事件和 Buff 提供统一的属性分类（良性/中性/恶性），便于 RNG 抽卡和游戏逻辑处理。

## 数据结构

### EventAttribute (事件属性枚举)

```go
type EventAttribute int

const (
    AttributeGood     EventAttribute = iota // 良性（正面效果）
    AttributeNeutral                        // 中性（混合/随机效果）
    AttributeBad                            // 恶性（负面效果）
)
```

| 属性 | 说明 | 颜色标识 |
|------|------|----------|
| Good | 良性事件/正面Buff | 🥰 |
| Neutral | 中性事件/混合效果 | 🫥 |
| Bad | 恶性事件/负面Buff | 🤮 |

### EventType (事件类型枚举)

**良性事件 (Good)**：
| 类型 | 名称 | 效果 |
|------|------|------|
| Herb | 采集到草药 | HP+1 |
| MilkTea | 捡到奶茶 | LP+1 |
| Relic | 捡到勇士的圣遗物 | 道具抽奖 |
| DivineBless | 受到天使眷顾 | 神眷Buff |

**中性事件 (Neutral)**：
| 类型 | 名称 | 效果 |
|------|------|------|
| Exchange | 交换 | 与随机玩家交换位置 |
| HiddenBuff | 麻了 | 隐匿Buff |
| TasteTest | 这是什么？尝一口 | 腐化/甘霖Buff（随机） |

**恶性事件 (Bad)**：
| 类型 | 名称 | 效果 |
|------|------|------|
| Mosquito | 被蚊虫叮咬 | HP-1 |
| GhostHit | 偶遇孤魂野鬼 | HP-1 |
| DogPoop | 踩到了狗屎 | LP-1 |
| Thief | 啊？！贼 | 随机丢失道具 |
| CurseBuddha | 虔诚拜三拜 | 诅咒Buff |
| LostWay | 迷途 | 迷途Buff |
| Thunder | 雷劫 | HP归零 |

### EventDefinition (事件定义)

```go
type EventDefinition struct {
    Type       EventType      // 事件类型
    Attribute  EventAttribute // 事件属性
    Name       string         // 事件名称（中文）
    Desc       string         // 事件描述
    HPChange   int            // HP变化值
    LPChange   int            // LP变化值
    BuffType   BuffType       // 获得的Buff类型
    ItemAction string         // 道具行为（gain/lose/draw）
}
```

### BuffDefinition (Buff定义)

```go
type BuffDefinition struct {
    Type       BuffType      // Buff类型
    Attribute  EventAttribute // Buff属性
    Name       string         // Buff名称（中文）
    Desc       string         // Buff描述
    Duration   int            // 持续回合数（-1表示永久）
    HPPerTurn  int            // 每回合HP变化
    LPPerTurn  int            // 每回合LP变化
    Special    string         // 特殊效果描述
}
```

**良性 Buff**：
| 类型 | 名称 | 效果 | 持续 |
|------|------|------|------|
| Divine | 神眷 | 每回合LP+1 | 3 |
| Hidden | 隐匿 | 免疫任意事件/BUFF/道具 | 3 |
| Rain | 甘霖 | 每2回合HP+1 | 4 |
| Exorcism | 辟邪 | 无视毒瘴buff | 5 |
| Fire | 离火 | 每4回合LP+1 | 永久 |

**恶性 Buff**：
| 类型 | 名称 | 效果 | 持续 |
|------|------|------|------|
| Curse | 诅咒 | 每回合LP-1 | 3 |
| Lost | 迷途 | 反方向移动 | 1 |
| Corrupt | 腐化 | 每2回合HP-1 | 4 |
| Poison | 毒瘴 | 每回合恶性随机事件 | 3 |

## 核心功能

### 1. 获取事件属性

```go
event := EventTypeHerb
attr := event.GetAttribute()  // 返回 AttributeGood

def := event.GetEventDefinition()
fmt.Println(def.Name)    // "采集到草药"
fmt.Println(def.HPChange) // 1
```

### 2. 获取 Buff 属性

```go
buff := BuffTypeCurse
attr := buff.GetBuffAttribute()  // 返回 AttributeBad

def := buff.GetBuffDefinition()
fmt.Println(def.Name)      // "诅咒"
fmt.Println(def.LPPerTurn) // -1
fmt.Println(def.Duration)  // 3
```

### 3. 事件注册表

```go
registry := NewEventRegistry()

// 按属性获取事件列表
goodEvents := registry.GetEventsByAttribute(AttributeGood)
badEvents := registry.GetEventsByAttribute(AttributeBad)

// 统计
len(registry.GoodEvents)   // 4
len(registry.NeutralEvents) // 3
len(registry.BadEvents)    // 7
```

### 4. Buff 注册表

```go
registry := NewBuffRegistry()

goodBuffs := registry.GetBuffsByAttribute(AttributeGood)
badBuffs := registry.GetBuffsByAttribute(AttributeBad)

len(registry.GoodBuffs) // 5
len(registry.BadBuffs) // 4
```

## RNG 集成

### AttributeBasedPool (属性分类抽卡池)

```go
pool := NewAttributeBasedPool()

// 添加事件到对应属性池
pool.AddGoodItem("herb", 10, nil)
pool.AddBadItem("mosquito", 10, nil)
pool.AddNeutralItem("exchange", 10, nil)

// 从指定属性池抽取
item, err := pool.DrawFromAttribute(AttributeGood)

// 根据幸运值抽取（自动选择属性）
item, attr, err := pool.DrawWithLuck(player.LP)
```

### 幸运值对属性概率的影响

| 幸运值 | 良性概率 | 恶性概率 | 中性概率 |
|--------|---------|---------|---------|
| 0 | 30% | 30% | 40% |
| 5 | 55% | 5% | 40% |
| 8 | 70% | 10% | 40% |

计算公式：
- `goodProb = 30 + luck*5`
- `badProb = 30 - luck*5`
- `neutralProb = 40`（固定）

## 与其他模块协作

### PlayerSystem 集成

```go
player := NewPlayer(PlayerConfig{UserID: "test"})

// 检查 Buff 属性决定是否免疫
if buff.GetBuffAttribute() == AttributeBad && player.HasBuff(BuffTypeHidden) {
    // 隐匿状态下免疫负面 Buff
}

// 检查事件属性触发阵营被动
if event.GetAttribute() == AttributeBad && player.Faction == FactionXuanWu {
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
    if event.GetAttribute() == AttributeBad {
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

测试文件：`internal/game/event_test.go`

| 测试类 | 覆盖内容 |
|--------|----------|
| EventAttributeTest | 属性名称、有效性、正面/负面判断 |
| EventTypeTest | 类型名称、属性获取、定义获取 |
| BuffAttributeTest | Buff属性获取 |
| BuffDefinitionTest | Buff定义获取（名称、持续时间、效果） |
| EventRegistryTest | 注册表创建、按属性获取 |
| BuffRegistryTest | 注册表创建、按属性获取 |

## 后续扩展

1. **事件执行器**：根据 EventDefinition 自动应用效果
2. **Buff 结算器**：回合结束时自动计算 HP/LP 变化
3. **动态权重**：根据游戏进度调整事件权重（后期恶性事件增多）

## 文件结构

```
internal/game/
├── event.go          # 事件属性系统实现 (~400行)
└── event_test.go     # 单元测试 (~300行)

pkg/rng/
└── weighted_random.go # AttributeBasedPool (追加 ~150行)
```