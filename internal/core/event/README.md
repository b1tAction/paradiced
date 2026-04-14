# internal/core/event - Event System

事件系统，包含随机事件的定义和注册表。

## 概述

event 包提供 EventType、定义和注册表。包级 init() 自动初始化所有事件定义。

## Direct Import

```go
import "github.com/b1tAction/paradiced/internal/core/event"

// 自动初始化，可直接使用
event.GetEventDefinition(event.EventTypeHerb)
event.GetEventEvaluation(event.EventTypeThunder)
```

## 数据类型

### EventType 枚举

```go
type EventType int

const (
    // 良性事件
    EventTypeHerb        // 采集草药：HP+1
    EventTypeMilkTea     // 捡到奶茶：LP+1
    EventTypeRelic       // 圣遗物：道具抽奖
    EventTypeDivineBless // 天使眷顾：神眷Buff

    // 中性事件
    EventTypeExchange    // 交换：位置互换
    EventTypeHiddenBuff  // 麻了：隐匿Buff
    EventTypeTasteTest   // 尝一口：随机Buff

    // 恶性事件
    EventTypeMosquito    // 蚊虫叮咬：HP-1
    EventTypeGhostHit    // 孤魂野鬼：HP-1
    EventTypeDogPoop     // 狗屎：LP-1
    EventTypeThief       // 盗贼：丢失道具
    EventTypeCurseBuddha // 野佛：诅咒Buff
    EventTypeLostWay     // 迷途：迷途Buff
    EventTypeThunder     // 雷劫：HP归零
)
```

### EventDefinition

```go
type EventDefinition struct {
    Type          EventType
    Eval          types.Evaluation
    EnglishName   string
    Name          string        // 中文显示名
    Desc          string
    HPChange      int
    LPChange      int
    BuffType      buff.BuffType // 赋予的Buff
    SpecialEffect types.SpecialEffect
}
```

## 注册表 API

```go
// 获取定义
event.GetEventDefinition(et) *EventDefinition

// 获取评估分数
event.GetEventEvaluation(et) types.Evaluation

// 获取分类列表
event.GetEventTypesByCategory("Bad") []EventType
event.GetAllEventTypes() []EventType

// 按评估范围查询
event.GlobalEventRegistry.GetEventTypesByEvaluationRange(min, max) []EventType
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

## 测试

```bash
go test ./internal/core/event/...
```