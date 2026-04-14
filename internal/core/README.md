# internal/core - Core Data Structures

核心数据结构包，提供游戏实体的统一入口。

## 概述

core 包采用 **Direct Import** 模式，重导出所有子包类型。导入 core 包自动初始化所有子包。

## 使用方式

```go
// 完整游戏逻辑（自动初始化所有子包）
import "github.com/b1tAction/fated/internal/core"

// 直接使用重导出的类型
core.BuffTypeDivine
core.EventTypeHerb
core.ItemTypeAnyDoor

// 使用重导出的函数
core.GetBuffDefinition(core.BuffTypeFire)
core.GetEventEvaluation(core.EventTypeThunder)

// 使用向后兼容的 CombinedRegistry
core.GlobalRegistry.GetBuffTypesByEvaluationRange(0, 40)
```

## 子包结构

| 子包 | 描述 | 依赖 |
|------|------|------|
| types/ | 共享基础类型 | 无 |
| buff/ | Buff 系统 | types, pkg/protocol |
| event/ | Event 系统 | buff, types |
| item/ | Item 系统 | buff, types |

## 核心文件

| 文件 | 描述 |
|------|------|
| init.go | 统一入口，重导出所有类型 |
| player.go | Player 结构体（实现 protocol.Player） |
| faction.go | Faction 阵营定义（类型别名到 protocol.Faction） |

## Player 结构

Player 实现 `pkg/protocol.Player` 接口，提供完整的读写能力：

```go
type Player struct {
    UserID      string
    Faction     Faction
    Position    int
    HP          int        // 默认最大6
    LP          int        // 范围0~8
    Inventory   []*item.Item
    ActiveBuffs []*buff.Buff
    IsDead      bool
    SkipTurn    bool
    *util.Metadata
}

// 创建玩家
player := core.NewPlayer(core.PlayerConfig{
    UserID:  "player-001",
    Faction: core.FactionZhuQue,
})

// Getter 方法（实现 protocol.PlayerReader）
player.GetUserID()
player.GetHP()
player.GetLP()
player.GetPosition()
player.GetFaction()
player.IsAlive()
player.CanAct()

// Writer 方法（实现 protocol.PlayerWriter）
player.ModifyLP(amount)
player.Heal(amount)
player.ApplyDamage(amount)
player.Move(newPos, maxLength)
player.Respawn(respawnPos)

// Metadata 方法
player.GetFireCounter()
player.SetFireCounter(count)
player.IncrementFireCounter()
```

## Faction 阵营

Faction 类型定义在 `pkg/protocol/player.go`，core 包使用类型别名：

```go
// pkg/protocol/player.go 定义
type Faction int
const (
    FactionQingLong Faction = iota
    FactionZhuQue
    FactionBaiHu
    FactionXuanWu
)

// internal/core/faction.go 类型别名
type Faction = protocol.Faction
```

| 阵营 | 技能 | 描述 |
|------|------|------|
| QingLong | 行迹 | 每5回合充能，无视负面地形 |
| ZhuQue | 离火 | 每4回合LP+1 |
| BaiHu | 劫运 | 反超偷Buff |
| XuanWu | 鎮厄 | 每5回合充能，抵消恶性事件 |

## 与 protocol 包的关系

- Player 实现 `protocol.Player` 接口
- Faction 类型定义在 protocol 包
- Buff Handler 使用 `protocol.PlayerLite` 最小接口

## 测试

```bash
go test ./internal/core/...
```

## 相关文档

- [types/README.md](types/README.md)
- [buff/README.md](buff/README.md)
- [event/README.md](event/README.md)
- [item/README.md](item/README.md)
- [pkg/protocol/README.md](../../pkg/protocol/README.md)