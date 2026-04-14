# internal/core/buff - Buff System

Buff 系统，支持多阶段触发和自定义处理器。

## 概述

buff 包提供 Buff 类型、定义和注册表。包级 init() 自动初始化所有 Buff 定义。

## Direct Import

```go
import "github.com/b1tAction/fated/internal/core/buff"

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
    ID              string       // UUID v7, auto-generated
    Duration        int
    Charge          int
    SubscriptionIDs []string     // EventBus 订阅ID列表
}

// NewBuff auto-generates UUID v7 ID
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
| Lost | MildBad | PreMove | 反向移动（在移动前拦截） |
| Corrupt | Bad | AfterTurn | HP-1/2回合 |
| Poison | VeryBad | BeforeTurn | 恶性事件 |
| Hidden | Neutral | PreDamage | 免疫（伤害前拦截） |
| Divine | VeryGood | BeforeTurn | LP+1/回合 |
| Rain | Good | AfterTurn | HP+1/2回合 |
| Exorcism | MildGood | PreEvent | 免疫毒瘴（事件前拦截） |
| Fire | Good | BeforeTurn | 朱雀被动 |

## EffectHandler 签名

Buff Handler 使用统一的 `EffectHandler` 签名，返回 Action：

```go
type EffectHandler func(phase event.Phase, ctx *event.Context) action.Action

// 示例：离火 Buff Handler
func handleZhuQueFire(phase event.Phase, ctx *event.Context) action.Action {
    // 使用 protocol.PlayerLite 最小接口
    player, ok := ctx.Player.(protocol.PlayerLite)
    if !ok {
        return nil
    }
    
    if phase != event.PhaseBeforeTurn {
        return nil
    }
    
    newCount := player.IncrementFireCounter()
    if newCount >= 4 {
        player.ModifyLP(1)
        player.SetFireCounter(0)
    }
    return nil // 直接修改，不返回 Action
}
```

## 与 protocol 包的关系

Buff Handler 使用 `protocol.PlayerLite` 接口而非本地定义的 Player 接口：

```go
// protocol.PlayerLite 最小接口
type PlayerLite interface {
    ModifyLP(amount int)
    GetFireCounter() int
    SetFireCounter(count int)
    IncrementFireCounter() int
}
```

## 测试

```bash
go test ./internal/core/buff/...
```