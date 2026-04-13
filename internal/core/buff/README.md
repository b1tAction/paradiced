# internal/core/buff - Buff System

Buff 系统，支持多阶段触发和自定义处理器。

## 概述

buff 包提供 Buff 类型、定义和注册表。包级 init() 自动初始化所有 Buff 定义。

## Direct Import

```go
import "github.com/b1tAction/Fated/internal/core/buff"

// 自动初始化，可直接使用
buff.GetBuffDefinition(buff.BuffTypeFire)
buff.BuffTypeDivine.IsPositive()
```

## 数据类型

### BuffType 枚举

```go
type BuffType int

const (
    BuffTypeCurse    // 诅咒：LP-1/回合
    BuffTypeLost     // 迷途：反向移动
    BuffTypeCorrupt  // 腐化：HP-1/2回合
    BuffTypePoison   // 毒瘴：恶性事件/回合
    BuffTypeHidden   // 隐匿：免疫
    BuffTypeDivine   // 神眷：LP+1/回合
    BuffTypeRain     // 甘霖：HP+1/2回合
    BuffTypeExorcism // 辟邪：免疫毒瘴
    BuffTypeFire     // 离火：朱雀被动
)
```

### BuffDefinition

```go
type BuffDefinition struct {
    Type          BuffType
    Eval          types.Evaluation
    EnglishName   string        // String() 返回此值
    Name          string        // 中文显示名
    Desc          string
    Duration      int           // -1 表示永久
    HPPerTurn     int
    LPPerTurn     int
    SpecialEffect types.SpecialEffect
    Phases        []event.Phase // 多阶段支持
    Priority      int
    NeedConfirm   bool
}
```

### Buff 实例

```go
type Buff struct {
    Type            BuffType
    ID              string
    Duration        int
    Charge          int
    SubscriptionIDs []string  // EventBus 订阅ID列表
}

buff.NewBuff(buff.BuffTypeCurse, 3)
```

## 注册表 API

```go
// 获取定义
buff.GetBuffDefinition(bt) *BuffDefinition

// 获取评估分数
buff.GetBuffEvaluation(bt) types.Evaluation

// 获取自定义处理器
buff.GetBuffHandler(bt) BuffHandler

// 获取分类列表
buff.GetBuffTypesByCategory("Good") []BuffType
buff.GetAllBuffTypes() []BuffType
```

## Buff 列表

| Buff | 评分 | 阶段 | 效果 |
|------|------|------|------|
| Curse | Bad | BeforeTurn | LP-1/回合 |
| Lost | MildBad | OnMove | 反向移动 |
| Corrupt | Bad | AfterTurn | HP-1/2回合 |
| Poison | VeryBad | BeforeTurn | 恶性事件 |
| Hidden | Neutral | PreDamage | 免疫 |
| Divine | VeryGood | BeforeTurn | LP+1/回合 |
| Rain | Good | AfterTurn | HP+1/2回合 |
| Exorcism | MildGood | PreEvent | 免疫毒瘴 |
| Fire | Good | BeforeTurn | 朱雀被动 |

## 测试

```bash
go test ./internal/core/buff/...
```