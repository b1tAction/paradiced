# Package internal/net - 同步数据构建层

本包提供同步数据构建器，用于将内部游戏结构转换为 `pkg/net` 协议数据。

## 设计目标

1. **数据转换**：将 `core.Player`、`gamelog.LogEntry` 转换为协议数据结构
2. **测试支持**：提供测试辅助函数模拟游戏流程
3. **复用现有组件**：使用 `pkg/rng.DiceManager` 进行骰子计算

## 文件说明

| 文件 | 说明 |
|------|------|
| `builder.go` | 同步数据构建器，转换内部数据为协议数据 |
| `test/helper.go` | 测试辅助函数，模拟消息广播和游戏流程 |

**注意**：骰子计算使用 `pkg/rng.DiceManager`，不在本包实现。

## Builder 使用

Builder 负责将内部数据结构转换为 `pkg/net` 协议数据：

```go
import (
    internalnet "github.com/b1tAction/paradiced/internal/net"
    pkgnet "github.com/b1tAction/paradiced/pkg/net"
)

// 创建构建器
builder := internalnet.NewBuilder(hsm, game)

// 构建状态同步
stateSync := builder.BuildStateSync()

// 构建玩家快照
players := builder.BuildPlayers()

// 构建Action同步（从GameLog）
entry := gamelog.LogEntry{...}
actionSync := builder.BuildActionSync(entry)

// 构建可用操作列表
builder.SetDiceType(rng.DiceTypeGold) // 设置骰子类型（使用 pkg/rng.DiceType 枚举）
available := builder.BuildAvailable(player)
```

## Metadata 键提取

Builder 自动从 `core.Player.Metadata` 提取已知键转为 typed 字段：

| Metadata 键 | Player 字段 | 用途 |
|-------------|-------------|------|
| `charge_count` | `Charge` | 青龙行迹/玄武镇厄充能数 |
| `fire_counter` | `FireCounter` | 朱雀离火计数器 |

```go
// Player 结构中的 typed 字段
pkgnet.Player{
    Charge:      player.GetChargeCount(),     // 从 Metadata 提取
    FireCounter: player.GetFireCounter(),    // 从 Metadata 提取
    ...
}
```

## 骰子计算

使用 `pkg/rng.DiceManager` 进行骰子计算（权威服务器）：

```go
import "github.com/b1tAction/paradiced/pkg/rng"

// 创建骰子管理器（传入游戏 RNG）
diceMgr := rng.NewDiceManager(game.RNG)

// 根据排名分配骰子类型（使用 rng.DiceType 枚举）
diceMgr.AssignDice(playerID, rank) // rank 1-4

// 滚骰子（返回 1-6，使用加权分布）
steps := diceMgr.RollSpecialDice(playerID)

// 或滚两个骰子（正常 + 特殊）
normal, special, total := diceMgr.RollBothDice(playerID)
```

| DiceType | 范围 | 权重分布 | 来源 |
|----------|------|----------|------|
| DiceTypeGold | 1-6 | 5-6权重70% | 排名 1 |
| DiceTypeSilver | 1-6 | 5-6权重50% | 排名 2 |
| DiceTypeCopper | 1-6 | 5-6权重40% | 排名 3 |
| DiceTypeWood/Normal | 1-6 | 均匀分布 | 排名 4 |

**注意**：`internal/net` 不再包含独立的骰子计算器，统一使用 `pkg/rng`。

骰子类型转字符串用于协议：
```go
diceType := rng.DiceTypeGold
str := diceType.String() // "gold" - 可直接用于 Available.DiceType
```

## 测试辅助

TestHelper 提供测试工具：

```go
import "github.com/b1tAction/paradiced/internal/net/test"

// 创建测试辅助
helper := test.NewTestHelper(12345)

// 模拟投骰子（使用 pkg/rng.DiceType 枚举）
steps := helper.SimulateRollDice(rng.DiceTypeGold)

// 模拟状态同步
helper.SimulateStateTransition("turn_loop", "main_action")

// 模拟 Action 执行
helper.SimulateActionExecution("damage", "player_001", -1, "Buff_Curse")

// 获取广播消息
broadcasts := helper.GetBroadcasts()

// 构建完整同步数据
fullSync := helper.BuildStateSync()
```

## 与 Nakama Handler 集成

实现 `pkg/net.MatchHandler` 时使用 Builder：

```go
func (h *NakamaMatchHandler) HandleMessage(sender string, msg pkgnet.Message) error {
    switch msg.OpCode {
    case pkgnet.OpRollDice:
        // 1. 计算骰子结果
        diceType := h.getCurrentDiceType(sender)
        steps := h.diceCalc.Calculate(diceType)

        // 2. 调用 HSM
        ctx := hsm.NewStateContext().WithGame(h.game)
        h.hsm.OnRollDice(steps, ctx)

        // 3. 广播结果
        actionSync := h.builder.BuildActionSync(...)
        h.Broadcast(pkgnet.OpActionSync, actionSync)

        // 4. 广播新状态
        stateSync := h.builder.BuildStateSync()
        h.Broadcast(pkgnet.OpStateSync, stateSync)
    }
}
```

## 相关文档

- [pkg/net/README.md](../../pkg/net/README.md) - 协议层定义
- [internal/engine/hsm/README.md](../engine/hsm/README.md) - HSM 状态机
- [pkg/gamelog/README.md](../../pkg/gamelog/README.md) - 游戏日志