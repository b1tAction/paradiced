# Buff.Metadata 契约

`core.Buff.Metadata` 用于Buff实例的跨回合状态存储，**不发送给客户端**。

**位置**：`internal/core/buff.go`

**可见性**：内部（仅后端使用）

---

## 概述

Buff.Metadata 嵌入 `util.Metadata`，用于存储需要跨回合持久化的Buff状态数据。
与 `event.Context.Metadata`（每回合新建）不同，Buff.Metadata 与 Buff 实例绑定，
在回合间持久保存。

```go
// internal/core/buff.go
type Buff struct {
    Type         constants.BuffType `json:"type"`
    ID           id.BuffID          `json:"id"`
    Duration     int                `json:"duration"`
    tickEligible bool               // 内部状态：由 TurnUpkeep 的 MarkAllBuffsTickEligible 标记
    *util.Metadata `json:"metadata"` // Per-buff状态存储
}
```

**tickEligible 机制**：
- `NewBuff` 默认 `tickEligible=false`
- `TurnUpkeepState.Enter()` 在回合开始时调用 `player.MarkAllBuffsTickEligible()`，将所有已有buff标记为eligible
- 回合中途添加的buff（如另一玩家使用道具定向添加）默认 `tickEligible=false`，不会被当回合的TurnEnd扣减，到下一回合的TurnUpkeep才被标记
- `TickDuration()` 仅在 `tickEligible=true` 时扣减Duration

**初始化**：`NewBuff` 和 `NewBuffWithID` 自动初始化 `Metadata: util.NewMetadata()`。

**客户端同步**：Builder 只映射 Type/Name/Duration，Metadata 不发送给客户端。

---

## 字段契约表

| 字段 | 类型 | 来源 | 用途 | 说明 |
|------|------|------|------|------|
| `buff_turn_counter` | int | everyNTurns handler | 甘霖/腐化每N回合触发计数器 | counter<N时不触发，达到N时触发并重置为0 |
| `rob_luck_source_player` | string | OnUseSkill (BaiHu) | 劫运Buff重定向目标 | 白虎玩家的UUID，用于将目标的增益Action重定向到白虎玩家 |

---

## everyNTurns 计数器逻辑

甘霖（Rain）和腐化（Corrupt）使用 `createEveryNTurnsHandler` 实现每N回合触发效果：

```go
func createEveryNTurnsHandler(everyN int, buffType constants.BuffType, innerHandler EffectHandler) EffectHandler {
    return func(phase constants.Phase, ctx *event.Context) error {
        buff := ctx.Player.GetBuff(buffType)
        counter := buff.GetIntOrDefault("buff_turn_counter", 0)
        counter++
        buff.SetInt("buff_turn_counter", counter)

        if counter >= everyN {
            innerHandler(phase, ctx) // 触发效果
            buff.SetInt("buff_turn_counter", 0) // 重置计数器
        }
        return nil
    }
}
```

**计数器生命周期**：
- 每次 PhaseAfterTurn 发布时，handler 从 Buff.Metadata 读取并递增计数器
- 计数器达到 N 时触发效果并重置为 0
- Buff 持续时间延长时（AddBuffAction duration extend），计数器不受影响
- Buff 过期移除时，计数器随 Buff 实例一起消失

**为什么使用 Buff.Metadata 而不是 event.Context.Metadata**：
- HSM 每回合发布 PhaseAfterTurn 时创建新的 `event.NewContext(player)`
- Context.Metadata 在回合间不持久化，计数器每回合从 0 开始
- Buff.Metadata 与 Buff 实例绑定，在回合间持久保存

---

## Duration Extend 时的 Metadata 行为

当 AddBuffAction 检测到玩家已有相同 BuffType 时，执行 duration 延长：
- 保留现有 Buff 实例（包括 Metadata 和计数器）
- 不创建新实例，计数器不会丢失

---

## 相关文档

- [doc/internal/core.md](../internal/core.md) - Core数据结构文档
- [doc/metadata/event_context.md](event_context.md) - EventBus Context.Metadata 契约
- [internal/core/buff.go](../../internal/core/buff.go) - Buff实现