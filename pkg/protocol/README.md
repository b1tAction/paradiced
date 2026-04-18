# Protocol 包

Protocol包提供公共接口层，用于解决Go语言中循环依赖问题。

## 设计背景

Go语言禁止循环导入，但游戏架构中常见依赖关系如下：

```
internal/engine/action (需要Game方法获取全局日志)
    ↑
internal/engine (定义Game)
```

这导致action包无法导入engine包获取Game，造成循环依赖。

## 解决方案

创建pkg/protocol作为公共接口层：

```
pkg/protocol (接口层，仅依赖pkg/gamelog, pkg/id, pkg/constants)
    ↓
internal/engine/action (使用protocol.Game)
    ↓
internal/engine (实现protocol.Game)
```

## 当前接口

### Game接口

用于ActionContext访问游戏全局状态，避免循环依赖：

```go
type Game interface {
    GetCurrentPlayer() interface{}
    GetPlayer(id id.PlayerID) interface{}
    GetPlayers() []interface{}
    GetGameLog() *gamelog.GameLog // 获取全局游戏日志
}
```

使用`interface{}`返回类型是因为：
- action包无法导入core包（会造成循环依赖）
- 调用方需要自行做类型断言转换为`*core.Player`

### MapEngine接口

用于HSM和Action包访问地图引擎：

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

## 使用方式

### ActionContext使用Game接口

```go
// internal/engine/action/context.go
type ActionContext struct {
    Game        protocol.Game      // 使用接口避免循环依赖
    EventBus    *event.EventBus
    MapEngine   protocol.MapEngine
}

// 使用时做类型断言
func (ctx *ActionContext) ExecuteAction(action ExecutableAction) error {
    currentPlayer, ok := ctx.Game.GetCurrentPlayer().(*core.Player)
    if !ok || currentPlayer == nil {
        currentPlayer = nil
    }
    // ...
}
```

### HSM使用GameWrapper适配器

```go
// internal/engine/hsm/adapter.go
type GameWrapper struct {
    game *engine.Game
}

func NewGameWrapper(game *engine.Game) protocol.Game {
    return &GameWrapper{game: game}
}

func (w *GameWrapper) GetCurrentPlayer() interface{} {
    return w.game.GetCurrentPlayer()
}
```

## 相关文档

- [pkg/net/broadcast.go](../net/broadcast.go) - BroadcastAdapter接口（类型安全的广播接口）
- [internal/engine/hsm/adapter.go](../../internal/engine/hsm/adapter.go) - 适配器实现