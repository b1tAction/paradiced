# internal/core/event - Event System

事件系统，包含随机事件的定义和注册表。

## 概述

event 包提供 EventType、定义和注册表。包级 init() 自动初始化所有事件定义和处理器配置。

## Direct Import

```go
import "github.com/b1tAction/paradiced/internal/core/event"

// 自动初始化，可直接使用
event.GetEventDefinition(event.EventTypeHerb)
event.GetEventHandlerConfig(event.EventTypeThunder)
event.GetEventEvaluation(event.EventTypeThunder)
```

## 数据类型

### EventType 枚举

EventType 使用 `constants.EventType` 字符串类型：

```go
const (
    // 良性事件
    EventTypeHerb        EventType = "herb"         // 采集草药：HP+1
    EventTypeMilkTea     EventType = "milk_tea"     // 捡到奶茶：LP+1
    EventTypeRelic       EventType = "relic"        // 圣遗物：道具抽奖
    EventTypeDivineBless EventType = "divine_bless" // 天使眷顾：神眷Buff

    // 中性事件
    EventTypeExchange    EventType = "exchange"     // 交换：位置互换
    EventTypeHiddenBuff  EventType = "hidden_buff"  // 麻了：隐匿Buff
    EventTypeTasteTest   EventType = "taste_test"   // 尝一口：随机Buff

    // 恶性事件
    EventTypeMosquito    EventType = "mosquito"     // 蚊虫叮咬：HP-1
    EventTypeGhostHit    EventType = "ghost_hit"    // 孤魂野鬼：HP-1
    EventTypeDogPoop     EventType = "dog_poop"     // 狗屎：LP-1
    EventTypeThief       EventType = "thief"        // 盗贼：丢失道具
    EventTypeCurseBuddha EventType = "curse_buddha" // 野佛：诅咒Buff
    EventTypeLostWay     EventType = "lost_way"     // 迷途：迷途Buff
    EventTypeThunder     EventType = "thunder"      // 雷劫：HP归零
)
```

### EventDefinition (静态元数据)

EventDefinition 只包含静态元数据，效果逻辑由 Handler 管理：

```go
type EventDefinition struct {
    Type        constants.EventType   `json:"type"`
    Eval        constants.Evaluation  `json:"evaluation"`
    EnglishName string                `json:"english_name"`  // String() 返回此值
    Name        string                `json:"name"`          // 中文显示名
    Desc        string                `json:"desc"`
}
```

移除字段：`HPChange`, `LPChange`, `BuffType`, `SpecialEffect`

### EventHandlerConfig (执行配置)

EventHandlerConfig 包含效果处理函数：

```go
type EventHandlerConfig struct {
    Handler EffectHandler  `json:"-"`  // 效果处理函数
}
```

说明：Event 不需要 Phases/Priority（Events 在 OnLand/PhaseOnEvent 时统一触发）

## 注册表 API

```go
// 获取定义（静态元数据）
event.GetEventDefinition(et) *EventDefinition

// 获取执行配置（含Handler）
event.GetEventHandlerConfig(et) *EventHandlerConfig

// 获取评估分数
event.GetEventEvaluation(et) constants.Evaluation

// 获取分类列表
event.GetEventTypesByCategory("Bad") []constants.EventType
event.GetAllEventTypes() []constants.EventType

// 按评估范围查询
event.GlobalEventRegistry.GetEventTypesByEvaluationRange(min, max) []constants.EventType
```

## 事件列表

| Event | 评分 | 效果 |
|-------|------|------|
| Herb | MildGood | HP+1 |
| MilkTea | Good | LP+1 |
| Relic | VeryGood | 道具抽奖 |
| DivineBless | Excellent | 神眷Buff |
| Exchange | Neutral | 位置互换 |
| HiddenBuff | Good | 隐匿Buff |
| TasteTest | Mixed | 随机Buff |
| Mosquito | MildBad | HP-1 |
| GhostHit | MildBad | HP-1 |
| DogPoop | MildBad | LP-1 |
| Thief | Bad | 丢失道具 |
| CurseBuddha | Bad | 诅咒Buff |
| LostWay | MildBad | 迷途Buff |
| Thunder | VeryBad | HP归零 |

## EffectHandler 签名

Event Handler 使用统一的 `EffectHandler` 签名：

```go
type EffectHandler func(phase constants.Phase, ctx *event.Context)

// 示例：草药事件 Handler
func handleHerbEvent(phase constants.Phase, ctx *event.Context) {
    // 信号HP+1意图（engine层通过Action执行）
    ctx.SetInt("hp_change", 1)
}
```

### Handler 辅助函数

```go
// 创建HP修改Handler
createModifyHPHandler(amount int) EffectHandler

// 创建LP修改Handler  
createModifyLPHandler(amount int) EffectHandler

// 创建给予Buff Handler
createGiveBuffHandler(buffType constants.BuffType, duration int) EffectHandler
```

## 测试

```bash
go test ./internal/core/event/...
```