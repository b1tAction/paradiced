# Paradiced 项目 Wrapper 和 Adapter 简化结果

## 已删除内容

| 内容 | 文件 | 原因 |
|------|------|------|
| `EventBusAdapter` | hsm/adapter.go | HSM 直接使用 `*event.EventBus` |
| `EventBusWrapper` | hsm/adapter.go | 冗余包装 |
| `GameWrapper` | hsm/adapter.go | ActionContext 使用 protocol.Game 接口 |
| `MapEngineAdapter` | hsm/adapter.go | HSM 直接使用 `*gamemap.MapEngine` |
| `MapEngineWrapper` | hsm/adapter.go | 冗余包装 |
| `PathResultWrapper` | hsm/adapter.go | 冗余包装 |
| `CellWrapper` | hsm/adapter.go | 冗余包装 |
| `ProtocolMapEngineWrapper` | hsm/adapter.go | 冗余包装（三层包装最外层） |
| `hsm.BroadcastAdapter` | hsm/state.go | 与 pkg/net.BroadcastAdapter 重复 |
| `adapter.go` 整个文件 | hsm/adapter.go | 无需保留 |
| `createBuffAction` | handlers.go | 内联到 game.go SubscribeBuff |
| `createItemAction` | handlers.go | 内联到 game.go SubscribeItem |
| `NewHealAction` 等7个包装函数 | handlers.go | 死代码，完全未使用 |
| `handlers.go` 整个文件 | handlers.go | 无需保留，逻辑已内联 |

## 保留内容

| 内容 | 文件 | 原因 |
|------|------|------|
| `pkg/protocol/game.go` | Game 接口 | 避免 action/engine 循环依赖 |
| `pkg/protocol/map.go` | MapEngine/PathResult/Cell 接口 | 供测试 Mock 使用 |
| `pkg/net/broadcast.go` | BroadcastAdapter + MockBroadcastAdapter | 类型安全的广播接口 |
| `internal/nakama/dispatcher.go` | DispatcherAdapter | 隔离 Nakama SDK |
| `internal/nakama/dispatcher_mock.go` | MockDispatcherAdapter | 测试 Mock |

## 最终架构

```
pkg/protocol/
├── game.go      # Game 接口（避免循环依赖）
├── map.go       # MapEngine/PathResult/Cell 接口（测试 Mock）

pkg/net/
├── broadcast.go # BroadcastAdapter（唯一广播接口）

internal/engine/
├── game.go      # Game 实现，添加 GetCurrentPlayerInterface() 等方法实现 protocol.Game
├── action/
│   ├── context.go # Game 使用 protocol.Game，MapEngine 使用 *gamemap.MapEngine
├── hsm/
│   ├── hsm.go     # bus: *event.EventBus, mapEngine: *gamemap.MapEngine
│   ├── state.go   # Broadcast 使用 pkg/net.BroadcastAdapter
```

## 依赖方向

```
pkg/protocol (接口定义，零 internal 依赖)
    ↓
pkg/net (BroadcastAdapter)
    ↓
internal/core, internal/event, internal/gamemap (具体类型)
    ↓
internal/engine/hsm (直接使用具体类型)
    ↓
internal/engine/action (Game 使用 protocol.Game 接口避免循环)
    ↓
internal/nakama (实现 pkg/net.BroadcastAdapter)
```

## 关键设计决策

1. **action 包不能导入 engine 包**：因为 action 是 engine 的子包，导入父包会造成循环依赖
2. **protocol.Game 接口使用 interface{} 返回类型**：避免 pkg 层依赖 internal 层
3. **engine.Game 实现接口方法名加 Interface 后缀**：GetCurrentPlayerInterface() 返回 interface{} 满足接口要求
4. **HSM 直接使用具体类型**：bus 和 mapEngine 不需要接口包装