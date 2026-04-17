# Protocol 包

Protocol包提供公共接口层，用于解决Go语言中循环依赖问题。

## 设计背景

Go语言禁止循环导入，但游戏架构中常见依赖关系如下：

```
internal/core (定义Player)
    ↑
internal/core/buff (需要Player方法)
    ↑
internal/engine/action (需要Game方法)
    ↑
internal/engine (定义Game)
```

这导致buff包无法导入core包获取Player，action包无法导入engine包获取Game。

## 解决方案

创建pkg/protocol作为公共接口层：

```
pkg/protocol (接口层，无内部包依赖)
    ↓
pkg/event, pkg/action, pkg/util (基础层)
    ↓
internal/core/buff, internal/core/event, internal/core/item
    ↓
internal/core (实现protocol.Player)
    ↓
internal/engine/action (使用protocol.Game)
    ↓
internal/engine (实现protocol.Game)
```

## 包分工

根据架构设计：

- **pkg/constants**: 存放 const 变量数据（BuffType, EventType, Faction, CellType 等）
- **pkg/protocol**: 存放 interface 定义（Player, Game, MapEngine, Broadcast 等）

## 接口分层

遵循Go的接口隔离原则，每个接口只包含必要的方法：

### Player接口

```go
// 只读接口
type PlayerReader interface {
    GetUserID() string
    GetHP() int
    GetLP() int
    GetPosition() int
    GetFaction() constants.Faction
    IsAlive() bool
    CanAct() bool
}

// 写入接口
type PlayerWriter interface {
    ModifyLP(amount int)
    Heal(amount int) error
    ApplyDamage(amount int) error
    Move(newPosition int, maxLength int) error
    Respawn(respawnPos int) error
}

// 组合接口（完整）
type Player interface {
    PlayerReader
    PlayerWriter
    GetFireCounter() int
    SetFireCounter(count int)
    IncrementFireCounter() int
    GetChargeCount() int
    SetChargeCount(count int)
    IncrementChargeCount() int
}

// 最小接口（Buff handler专用）
type PlayerLite interface {
    ModifyLP(amount int)
    GetFireCounter() int
    SetFireCounter(count int)
    IncrementFireCounter() int
}
```

### Game接口

```go
type Game interface {
    GetCurrentPlayer() interface{}
    GetPlayer(playerID id.PlayerID) interface{}
    GetPlayers() []interface{}
    GetGameLog() *gamelog.GameLog
}
```

### MapEngine接口

```go
type PathResult interface {
    GetTargetIndex() int
    GetPath() []int
}

type Cell interface {
    GetPosition() int
    GetType() constants.CellType
    IsFogActive() bool
}

type MapEngine interface {
    CalculatePath(startPos int, steps int) (PathResult, error)
    GetLength() int
    GetCell(pos int) (Cell, error)
    GetLastCheckpoint(pos int) int
    SetCellType(pos int, cellType constants.CellType) error
    ActivateFog(pos int) error
    IsFogActivated(pos int) bool
}
```

### Broadcast接口

```go
type Broadcast interface {
    BroadcastStateSync(state interface{}) error
    BroadcastTurnSync(turn interface{}) error
    SendDecision(playerID string, decision interface{}) error
    SendAvailable(playerID string, available interface{}) error
    BroadcastMiniGameStart(start interface{}) error
    BroadcastMiniGameResult(result interface{}) error
    BroadcastGameOver(over interface{}) error
    SendFullSync(playerID string, state, turn interface{}) error
}
```

### Dispatcher接口

```go
type Dispatcher interface {
    BroadcastMessage(opCode int64, data []byte) error
    SendMessage(playerID string, opCode int64, data []byte) error
}
```

## 使用方式

### Buff Handler使用PlayerLite

```go
// internal/core/buff/init.go
func handleZhuQueFire(phase event.Phase, ctx *event.Context) action.Action {
    player, ok := ctx.Player.(protocol.PlayerLite)
    if !ok {
        return nil
    }
    newCount := player.IncrementFireCounter()
    if newCount >= 4 {
        player.ModifyLP(1)
        player.SetFireCounter(0)
    }
    return nil
}
```

### Action类型使用Player接口

```go
// internal/engine/action/types.go
type DamageAction struct {
    TargetPlayer protocol.Player  // 使用接口
    SourceID     string
    Amount       int
}
```

### ActionContext使用Game接口

```go
// internal/engine/action/context.go
type ActionContext struct {
    Game        protocol.Game       // 使用接口
    EventBus    *event.EventBus
    MapEngine   protocol.MapEngine  // 使用接口
}
```

## Metadata 契约

**重要**: 项目中多个类型嵌入 `util.Metadata`，所有字段使用必须遵循契约文档。

详见 [doc/metadata/README.md](../../doc/metadata/README.md)。

## 命名规范

- 接口不带Interface后缀（遵循Go惯例）
- Reader/Writer分离表示读写能力
- Lite表示最小接口（仅包含必要方法）

## 相关文档

- [pkg/constants/README.md](../constants/README.md) - 常量类型定义
- [doc/internal/metadata.md](../../doc/internal/metadata.md) - Metadata工具类使用说明