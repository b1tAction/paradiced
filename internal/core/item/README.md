# internal/core/item - Item System

道具系统，包含道具的定义和注册表。

## 概述

item 包提供 ItemType、道具实例、定义和注册表。包级 init() 自动初始化所有道具定义。

## Direct Import

```go
import "github.com/b1tAction/fated/internal/core/item"

// 自动初始化，可直接使用
item.GetItemDefinition(item.ItemTypeAnyDoor)
item.GetItemEvaluation(item.ItemTypeDiceUpgrade)
```

## 数据类型

### ItemType 枚举

```go
type ItemType int

const (
    ItemTypeReverseClock // 反方向的钟：迷途Buff
    ItemTypeAnyDoor      // 任意门：传送
    ItemTypeDiceSwap     // 骰子交换
    ItemTypeDiceUpgrade  // 骰子升级卡
)
```

### ItemDefinition

```go
type ItemDefinition struct {
    Type          ItemType
    Eval          types.Evaluation
    EnglishName   string
    Name          string        // 中文显示名
    Desc          string
    TargetSelf    bool          // 可对自己使用
    TargetOther   bool          // 可对他人使用
    BuffType      buff.BuffType // 赋予的Buff
    Range         int           // 有效范围
    SpecialEffect types.SpecialEffect
    Phase         event.Phase   // 可使用时机
    Priority      int
    NeedConfirm   bool          // 默认true
}
```

### Item 实例

```go
type Item struct {
    Type           ItemType
    ID             string       // UUID v7, auto-generated
    Usable         bool
    TargetID       string
    SubscriptionID string       // EventBus 订阅ID
}

// NewItem auto-generates UUID v7 ID
item.NewItem(item.ItemTypeAnyDoor)

// NewItemWithID creates item with specific ID (for testing)
item.NewItemWithID(item.ItemTypeAnyDoor, "test-item-001")
```

## 注册表 API

```go
// 获取定义
item.GetItemDefinition(it) *ItemDefinition

// 获取评估分数
item.GetItemEvaluation(it) types.Evaluation

// 获取分类列表
item.GetItemTypesByCategory("Good") []ItemType
item.GetAllItemTypes() []ItemType
```

## 道具列表

| Item | 评分 | 目标 | 阶段 | 效果 |
|------|------|------|------|------|
| ReverseClock | Good | Other | AnyTime | 迷途Buff |
| AnyDoor | Neutral | Other | OnLand | 传送(30格内) |
| DiceSwap | Neutral | Other | AnyTime | 骰子等级交换 |
| DiceUpgrade | Good | Self | BeforeTurn | 骰子升级 |

## 测试

```bash
go test ./internal/core/item/...
```