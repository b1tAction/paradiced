# internal/core/item - Item System

道具系统，包含道具的定义和注册表。

## 概述

item 包提供 ItemType、道具实例、定义和注册表。包级 init() 自动初始化所有道具定义和处理器配置。

## Direct Import

```go
import "github.com/b1tAction/paradiced/internal/core/item"

// 自动初始化，可直接使用
item.GetItemDefinition(item.ItemTypeAnyDoor)
item.GetItemHandlerConfig(item.ItemTypeDiceUpgrade)
item.GetItemEvaluation(item.ItemTypeDiceUpgrade)
```

## 数据类型

### ItemType 枚举

ItemType 使用 `constants.ItemType` 字符串类型：

```go
const (
    ItemTypeReverseClock ItemType = "reverse_clock" // 反方向的钟：迷途Buff
    ItemTypeAnyDoor      ItemType = "any_door"      // 任意门：传送
    ItemTypeDiceSwap     ItemType = "dice_swap"     // 骰子交换
    ItemTypeDiceUpgrade  ItemType = "dice_upgrade"  // 骰子升级卡
)
```

### ItemDefinition (静态元数据)

ItemDefinition 只包含静态元数据和目标配置，效果逻辑由 Handler 管理：

```go
type ItemDefinition struct {
    Type        constants.ItemType   `json:"type"`
    Eval        constants.Evaluation `json:"evaluation"`
    EnglishName string               `json:"english_name"`  // String() 返回此值
    Name        string               `json:"name"`          // 中文显示名
    Desc        string               `json:"desc"`
    TargetSelf  bool                 `json:"target_self"`   // 可对自己使用
    TargetOther bool                 `json:"target_other"`  // 可对他人使用
    Range       int                  `json:"range"`         // 有效范围
}
```

移除字段：`BuffType`, `SpecialEffect`, `Phase`, `Priority`, `NeedConfirm`

说明：`TargetSelf/TargetOther/Range` 保留，因为这是目标选择配置，属于使用方式而非效果逻辑。

### ItemHandlerConfig (执行配置)

ItemHandlerConfig 包含执行配置和效果处理函数：

```go
type ItemHandlerConfig struct {
    Phase       constants.Phase      `json:"phase"`         // 可使用时机
    Priority    int                  `json:"priority"`      // 执行优先级
    NeedConfirm bool                 `json:"need_confirm"`  // 是否需要用户确认
    Handler     EffectHandler        `json:"-"`             // 效果处理函数
}
```

### Item 实例

```go
type Item struct {
    Type           constants.ItemType
    ID             id.ItemID             // UUID v7, auto-generated
    Usable         bool
    TargetID       id.PlayerID
    SubscriptionID id.SubscriptionID     // EventBus 订阅ID
}

// NewItem auto-generates UUID v7 ID
item.NewItem(constants.ItemTypeAnyDoor)

// NewItemWithID creates item with specific ID (for testing)
item.NewItemWithID(constants.ItemTypeAnyDoor, testID)
```

## 注册表 API

```go
// 获取定义（静态元数据）
item.GetItemDefinition(it) *ItemDefinition

// 获取执行配置（含Handler）
item.GetItemHandlerConfig(it) *ItemHandlerConfig

// 获取评估分数
item.GetItemEvaluation(it) constants.Evaluation

// 获取分类列表
item.GetItemTypesByCategory("Good") []constants.ItemType
item.GetAllItemTypes() []constants.ItemType
```

## 道具列表

| Item | 评分 | 目标 | 阶段 | 效果 |
|------|------|------|------|------|
| ReverseClock | Good | Other | AnyTime | 迷途Buff |
| AnyDoor | Neutral | Other | OnLand | 传送(30格内) |
| DiceSwap | Neutral | Other | AnyTime | 骰子等级交换 |
| DiceUpgrade | Good | Self | BeforeTurn | 骰子升级 |

## EffectHandler 签名

Item Handler 使用统一的 `EffectHandler` 签名：

```go
type EffectHandler func(phase constants.Phase, ctx *event.Context)

// 示例：任意门 Handler
func handleAnyDoor(phase constants.Phase, ctx *event.Context) {
    // 信号传送意图（engine层通过Action执行）
    targetPos := calculateTeleportPosition(ctx)
    ctx.SetInt("teleport_position", targetPos)
}
```

## 测试

```bash
go test ./internal/core/item/...
```