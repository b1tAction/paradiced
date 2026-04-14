# internal/core/buff - Buff System

Buff 系统，支持多阶段触发和自定义处理器。

## 概述

buff 包提供 Buff 类型、定义和注册表。包级 init() 自动初始化所有 Buff 定义和处理器配置。

## Direct Import

```go
import "github.com/b1tAction/paradiced/internal/core/buff"

// 自动初始化，可直接使用
buff.GetBuffDefinition(buff.BuffTypeFire)
buff.GetBuffHandlerConfig(buff.BuffTypeFire)
buff.BuffTypeDivine.IsPositive()
```

## 数据类型

### BuffType 枚举

BuffType 使用 `constants.BuffType` 字符串类型：

```go
const (
    BuffTypeCurse    BuffType = "curse"    // 诅咒：LP-1/回合
    BuffTypeLost     BuffType = "lost"     // 迷途：反向移动
    BuffTypeCorrupt  BuffType = "corrupt"  // 腐化：HP-1/2回合
    BuffTypePoison   BuffType = "poison"   // 毒瘴：恶性事件/回合
    BuffTypeHidden   BuffType = "hidden"   // 隐匿：免疫
    BuffTypeDivine   BuffType = "divine"   // 神眷：LP+1/回合
    BuffTypeRain     BuffType = "rain"     // 甘霖：HP+1/2回合
    BuffTypeExorcism BuffType = "exorcism" // 辟邪：免疫毒瘴
    BuffTypeFire     BuffType = "fire"     // 离火：朱雀被动
)
```

### BuffDefinition (静态元数据)

BuffDefinition 只包含静态元数据，效果逻辑由 Handler 管理：

```go
type BuffDefinition struct {
    Type        constants.BuffType   `json:"type"`
    Eval        constants.Evaluation `json:"evaluation"`
    EnglishName string               `json:"english_name"`  // String() 返回此值
    Name        string               `json:"name"`          // 中文显示名
    Desc        string               `json:"desc"`
    Duration    int                  `json:"duration"`      // -1 表示永久
}
```

移除字段：`HPPerTurn`, `LPPerTurn`, `SpecialEffect`, `Phases`, `Priority`, `NeedConfirm`

### BuffHandlerConfig (执行配置)

BuffHandlerConfig 包含执行配置和效果处理函数：

```go
type BuffHandlerConfig struct {
    Phases      []constants.Phase    `json:"phases"`        // 触发Phase列表
    Priority    int                  `json:"priority"`      // 执行优先级
    NeedConfirm bool                 `json:"need_confirm"`  // 是否需要用户确认
    Handler     EffectHandler        `json:"-"`             // 效果处理函数
}

func (c *BuffHandlerConfig) GetPhases() []constants.Phase
func (c *BuffHandlerConfig) HasPhase(phase constants.Phase) bool
```

### Buff 实例

```go
type Buff struct {
    Type            constants.BuffType
    ID              id.BuffID             // UUID v7, auto-generated
    Duration        int
    Charge          int
    SubscriptionIDs []id.SubscriptionID   // EventBus 订阅ID列表
}

// NewBuff auto-generates UUID v7 ID
buff.NewBuff(constants.BuffTypeCurse, 3)
```

## 注册表 API

```go
// 获取定义（静态元数据）
buff.GetBuffDefinition(bt) *BuffDefinition

// 获取执行配置（含Handler）
buff.GetBuffHandlerConfig(bt) *BuffHandlerConfig

// 检查是否有Handler
buff.HasBuffHandler(bt) bool

// 获取评估分数
buff.GetBuffEvaluation(bt) constants.Evaluation

// 获取触发阶段列表
buff.GetBuffPhases(bt) []constants.Phase

// 获取分类列表
buff.GetBuffTypesByCategory("Good") []constants.BuffType
buff.GetAllBuffTypes() []constants.BuffType
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

Buff Handler 使用统一的 `EffectHandler` 签名：

```go
type EffectHandler func(phase constants.Phase, ctx *event.Context)

// 示例：离火 Buff Handler
func handleZhuQueFire(phase constants.Phase, ctx *event.Context) {
    // 使用 protocol.PlayerLite 最小接口
    player, ok := ctx.Player.(protocol.PlayerLite)
    if !ok {
        return
    }
    
    if phase != constants.PhaseBeforeTurn {
        return
    }
    
    newCount := player.IncrementFireCounter()
    if newCount >= 4 {
        player.ModifyLP(1)
        player.SetFireCounter(0)
    }
}
```

### Handler 辅助函数

```go
// 创建HP修改Handler
createModifyHPHandler(amount int) EffectHandler

// 创建LP修改Handler
createModifyLPHandler(amount int) EffectHandler

// 创建每N回合执行的Handler
createEveryNTurnsHandler(everyN int, innerHandler EffectHandler) EffectHandler
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