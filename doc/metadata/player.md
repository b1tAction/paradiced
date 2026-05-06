# Player.Metadata 契约

`core.Player.Metadata` 存储玩家动态属性，不同阵营有不同字段。

**位置**：`internal/core/player.go`

**可见性**：客户端可见（通过 `pkg/net.StateSync.Players` 发送）

---

## 概述

Player.Metadata 主要存储阵营特定属性，通过便捷方法访问：

```go
// 便捷方法定义在 internal/core/player.go
func (p *Player) GetChargeCount() int {
    return p.GetIntOrDefault("charge_count", 0)
}

func (p *Player) SetChargeCount(count int) {
    p.SetInt("charge_count", count)
}
```

---

## 字段契约表

| 字段 | 类型 | 所属阵营 | 用途 | 便捷方法 |
|------|------|----------|------|----------|
| `display_name` | string | 通用 | 用户显示名称，fallback 到 userID | 无（直接通过 Metadata 访问） |
| `charge_count` | int | 青龙/白虎/玄武 | 阵营技能充能数 | `GetChargeCount/SetChargeCount/IncrementChargeCount` |
| `charge_turn_counter` | int | 青龙/白虎/玄武 | 充能回合计数器（每2回合+1充能） | `GetChargeTurnCounter/SetChargeTurnCounter/IncrementChargeTurnCounter` |
| `fire_counter` | int | 朱雀 | 离火计数器（每3回合LP+1） | `GetFireCounter/SetFireCounter/IncrementFireCounter` |
| `events_drawn` | int | 通用 | 累计抽取Event次数 | `GetEventsDrawn/IncrementEventsDrawn` |
| `items_used` | int | 通用 | 累计消耗Item次数 | `GetItemsUsed/IncrementItemsUsed` |
| `rounds_won` | int | 通用 | 累计MiniGame排名第1次数 | `GetRoundsWon/IncrementRoundsWon` |

### display_name 说明

**通用字段**，所有阵营玩家均拥有：
- 由客户端通过 MatchJoin metadata 传入
- 存储在 `Player.Metadata` 中
- 通过 `BroadcastStateSync` 注入到 `pkg/net.Player.DisplayName` 字段
- 通过 `WaitingSync` 注入到 `pkg/net.WaitingPlayer.DisplayName` 字段
- fallback：未传入时默认为 Nakama UserID（UUID）

### 阵营用途说明

**青龙（威势）**：
- `charge_turn_counter` 每回合增加1，达到2时 `charge_count` 增加1并重置
- 充能达到1时可使用威势技能，增益效果翻倍

**白虎（劫运）**：
- `charge_turn_counter` 每回合增加1，达到2时 `charge_count` 增加1并重置
- 充能达到1时可使用劫运技能，指定目标玩家，使其增益改向自身

**玄武（镇厄）**：
- `charge_turn_counter` 每回合增加1，达到2时 `charge_count` 增加1并重置
- 充能达到1时可使用镇厄技能，1回合免疫恶性事件和负面Buff

**朱雀（离火）**：
- `fire_counter` 每回合增加1
- 达到3时触发LP+1，并重置为0
- 最大LP为8，超过则不增加

---

## Builder 映射

`internal/net/builder.go` 的 `BuildPlayer()` 方法将 Metadata 字段映射到 `pkg/net.Player` 展平字段：

```go
func (b *Builder) BuildPlayer(p *core.Player) pkgnet.Player {
    return pkgnet.Player{
        UserID:      p.ID.UUID(),
        Faction:     p.Faction.SnakeCase(),
        Position:    p.Position,
        HP:          p.HP,
        LP:          p.LP,
        Buffs:       b.BuildBuffs(p.ActiveBuffs),
        Items:       b.BuildItems(p.Inventory),
        Charge:      p.GetChargeCount(),     // 从 Metadata 提取
        FireCounter: p.GetFireCounter(),     // 从 Metadata 提取
        IsDead:      p.IsDead,
        SkipTurn:    p.SkipTurn,
    }
}
```

客户端收到 `StateSync.Players` 后可直接使用 `charge` 和 `fire_counter` 字段显示充能状态。

---

## 客户端渲染

```typescript
interface Player {
    user_id: string;
    faction: string;
    position: number;
    hp: number;
    lp: number;
    buffs: Buff[];
    items: Item[];
    charge: number;      // 青龙/白虎/玄武充能数
    fire_counter: number; // 朱雀火计数
    is_dead: boolean;
    skip_turn: boolean;
}

// 渲染逻辑
function renderPlayerStatus(player: Player) {
    if (player.faction === "qing_long" || player.faction === "xuan_wu" || player.faction === "bai_hu") {
        // 显示充能状态
        if (player.charge >= 1) {
            showSkillReadyIcon();
        }
        showChargeCount(player.charge);
    }

    if (player.faction === "zhu_que") {
        // 显示火计数进度
        showFireProgress(player.fire_counter, 3);
    }
}
```

---

## 扩展说明

白虎劫运相关计数器已实现（charge_count + charge_turn_counter）。

未来可能添加：
- 通用临时标记（如"已访问检查点"）

新增属性时：
1. 在 `internal/core/player.go` 添加便捷方法
2. 更新此契约文档
3. 更新 `BuildPlayer()` 映射
4. 更新客户端 TypeScript 类型

---

## 相关文档

- [internal/core/README.md](../../internal/core/README.md) - Player结构定义
- [pkg/protocol/player.go](../../pkg/protocol/player.go) - Player接口
- [doc/internal/core.md](../internal/core.md) - Core包文档