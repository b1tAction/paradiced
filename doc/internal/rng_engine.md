# RNG Engine (随机数引擎) 实现文档

## 概述

RNG Engine 是《命运骰子》游戏的随机抽卡核心模块，负责带权重随机抽取、幸运值调节和概率配置管理。

## 数据结构

### WeightedItem (带权重抽卡项)

```go
type WeightedItem struct {
    ID     string      // 唯一标识
    Type   string      // 类型分类
    Weight int         // 权重值
    Data   interface{} // 附加数据
}
```

### WeightedPool (带权重抽卡池)

```go
type WeightedPool struct {
    Items       []WeightedItem // 抽卡项列表
    TotalWeight int            // 总权重
}
```

## 核心功能

### 1. 基础抽卡

```go
pool := NewWeightedPool()
pool.AddItem("item1", "typeA", 10, nil)  // 权重10
pool.AddItem("item2", "typeB", 30, nil)  // 权重30
pool.AddItem("item3", "typeC", 60, nil)  // 权重60

// 单次抽卡
item, err := pool.Draw()

// 按类型抽卡
item, err := pool.DrawWithType("typeA")

// 多次抽卡（不重复）
items, err := pool.DrawMultiple(3)
```

**权重概率计算**：
- item1: 10/100 = 10%
- item2: 30/100 = 30%
- item3: 60/100 = 60%

### 2. 幸运值调节系统

```go
modifier := NewLuckModifier(50, 0.1)  // 基础权重50，影响因子0.1

// luck=5，良性事件：权重增加
weight := modifier.CalculateWeight(5, true)  // 50 + 25 = 75

// luck=5，恶性事件：权重减少
weight := modifier.CalculateWeight(5, false) // 50 - 25 = 25
```

**幸运值影响规则**：
| 幸运值 | 良性事件权重 | 恶性事件权重 |
|--------|-------------|-------------|
| 0 | 不变 | 不变 |
| 5 | +50% | -50% |
| 8 | +80% | -80% |

### 3. 事件抽卡池 (EventPool)

```go
eventPool := NewEventPool()
eventPool.AddGoodEvent("hp_plus", 10, map[string]int{"hp": 1})
eventPool.AddBadEvent("hp_minus", 10, map[string]int{"hp": -1})
eventPool.AddNeutralEvent("exchange", 10, nil)

// 根据幸运值抽取事件
item, eventType, err := eventPool.DrawEvent(luck, rng)
```

**幸运值对事件类型的影响**：
| 幸运值 | 良性概率 | 恶性概率 | 中性概率 |
|--------|---------|---------|---------|
| 0 | 30% | 30% | 40% |
| 5 | 55% | 5% | 40% |
| 8 | 70% | 10% | 40% |

### 4. 道具抽卡池 (ItemPool)

```go
itemPool := NewItemPool()
itemPool.AddCommonItem("clock", 10, nil)
itemPool.AddRareItem("door", 10, nil)
itemPool.AddEpicItem("dice_upgrade", 10, nil)

// 根据幸运值抽取道具
item, rarity, err := itemPool.DrawItem(luck, rng)
```

**幸运值对稀有度的影响**：
| 幸运值 | 普通 | 稀有 | 史诗 |
|--------|-----|-----|-----|
| 0 | 70% | 20% | 10% |
| 5 | 30% | 30% | 20% |
| 8 | 18% | 34% | 26% |

### 5. 概率配置系统

```go
config := NewProbabilityConfig("event_config", "事件概率")
config.SetProbability("hp_plus", 25.0)
config.SetProbability("hp_minus", 25.0)
config.SetProbability("lp_plus", 25.0)
config.SetProbability("lp_minus", 25.0)

// 转换为权重池
pool, err := config.ToWeightedPool()
```

### 6. 抽卡统计

```go
stats := NewDrawStatistics()

// 记录抽卡结果
stats.Record(item)

// 获取概率
prob := stats.GetProbability("item1")

// 获取热门项
topItems := stats.GetTopItems(3)
```

## 与 PlayerSystem 集成

```go
// 玩家幸运值影响抽卡结果
player := NewPlayer(PlayerConfig{UserID: "test", MaxLP: 5})
player.ModifyLP(3)  // 幸运值 = 8

// 创建事件池
eventPool := NewEventPool()
eventPool.AddGoodEvent("hp_plus", 50, nil)
eventPool.AddBadEvent("hp_minus", 50, nil)
eventPool.AddNeutralEvent("exchange", 50, nil)

// 高幸运值玩家抽卡，更倾向良性事件
rng := rand.New(rand.NewSource(time.Now().UnixNano()))
item, eventType, _ := eventPool.DrawEvent(player.LP, rng)
```

## 测试覆盖

测试文件：`pkg/rng/weighted_random_test.go`

| 测试类 | 覆盖内容 |
|--------|----------|
| WeightedPoolTest | 创建、添加/移除项、获取项 |
| DrawTest | 单次抽卡、空池错误处理 |
| DrawDistributionTest | 10000次抽卡概率分布验证 |
| DrawWithTypeTest | 按类型抽卡 |
| DrawMultipleTest | 多次不重复抽卡 |
| LuckModifierTest | 权重计算、范围限制 |
| LuckAdjustedPoolTest | 幸运值调节抽卡 |
| ProbabilityConfigTest | 概率配置、验证、转换 |
| DrawStatisticsTest | 统计记录、概率计算、热门项 |
| EventPoolTest | 事件池创建、添加、抽卡 |
| ItemPoolTest | 道具池创建、添加、抽卡 |

## 预定义事件配置

根据 `doc/background.md` 中的事件设计：

### 良性事件 (Good Events)
| ID | 名称 | 效果 |
|----|------|------|
| hp_plus | 采集到草药 | HP+1 |
| lp_plus | 捡到奶茶 | LP+1 |
| item_draw | 捡到圣遗物 | 道具抽奖 |
| divine_buff | 天使眷顾 | 神眷Buff |

### 恶性事件 (Bad Events)
| ID | 名称 | 效果 |
|----|------|------|
| hp_minus | 被蚊虫叮咬 | HP-1 |
| lp_minus | 踩到狗屎 | LP-1 |
| item_lost | 遭贼 | 丢失道具 |
| curse_buff | 拜野佛 | 诅咒Buff |

### 中性事件 (Neutral Events)
| ID | 名称 | 效果 |
|----|------|------|
| exchange | 交换 | 与随机玩家交换位置 |
| hidden_buff | 麻了 | 隐匿Buff |
| random_buff | 这是什么 | 腐化/甘霖Buff |

## 后续扩展

1. **骰子类型系统**：金骰子/银骰子/木骰子的概率配置
2. **Boss战抽卡**：击败Boss的装备抽奖
3. **彩蛋系统**：连续3次1/6触发特殊抽奖

## 文件结构

```
pkg/rng/
├── weighted_random.go      # 核心实现 (~750行)
└── weighted_random_test.go # 单元测试 (~700行)
```