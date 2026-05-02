# LogEntry.Metadata 契约

`gamelog.LogEntry.Metadata` 存储Action效果详情，通过 `TurnSync.Entries` 发送给客户端渲染。

**位置**：`pkg/gamelog/entry.go`

**可见性**：客户端可见（通过 `pkg/net.TurnSync` 发送）

---

## 客户端使用说明

客户端根据 `action_type` 判断应解析哪些 metadata 字段：

```typescript
// 客户端渲染逻辑
for (const entry of turnSync.entries) {
    switch (entry.action_type) {
        case "damage":
            const hpChange = entry.metadata?.hp_change || 0;
            if (entry.metadata?.blocked_by) {
                // 显示阻挡来源：entry.metadata.blocked_by
            }
            if (entry.metadata?.piercing) {
                // 显示穿透效果图标
            }
            playDamageAnimation(entry.target, hpChange);
            break;
        case "heal":
            const hpChange = entry.metadata?.hp_change || 0;
            playHealAnimation(entry.target, hpChange);
            break;
        case "modify_lp":
            const lpChange = entry.metadata?.lp_change || 0;
            playLPChangeAnimation(entry.target, lpChange);
            break;
        case "move":
            const path = entry.metadata?.path || [];
            playMoveAnimation(entry.target, path);
            break;
        // ... 其他类型
    }
}
```

---

## 字段契约表

### 通用字段（多类型共用）

| 字段 | 类型 | 使用类型 | 用途 | 示例值 |
|------|------|----------|------|--------|
| `source` | string | 多类型 | 效果来源标识 | `"buff_divine"`, `"fragile_cell"` |

### damage 类型

| 字段 | 类型 | 必填 | 用途 | 客户端渲染 |
|------|------|------|------|-----------|
| `hp_change` | int | 是 | HP变化值（负数） | 显示伤害数值动画 |
| `blocked_by` | string | 否 | 阻挡来源Buff名称 | 显示"被XX阻挡"提示 |
| `piercing` | bool | 否 | 是否穿透防御 | 显示穿透效果图标 |

### heal 类型

| 字段 | 类型 | 必填 | 用途 | 客户端渲染 |
|------|------|------|------|-----------|
| `hp_change` | int | 是 | HP变化值（正数） | 显示恢复数值动画 |

### modify_lp 类型

| 字段 | 类型 | 必填 | 用途 | 客户端渲染 |
|------|------|------|------|-----------|
| `lp_change` | int | 是 | LP变化值 | 显示LP变化动画 |

### move 类型

| 字段 | 类型 | 必填 | 用途 | 客户端渲染 |
|------|------|------|------|-----------|
| `steps` | int | 是 | 移动步数 | 显示步数 |
| `start_pos` | int | 是 | 移动起点位置 | 内部计算 |
| `end_pos` | int | 是 | 移动终点位置 | 内部计算 |
| `path` | []int | 是 | 移动路径（格子索引列表） | 播放移动动画 |

**注意**：`path` 从 JSON 反序列化后可能是 `[]float64` 或 `[]interface{}`，需要转换为 `[]int`。

### add_buff 类型

| 字段 | 类型 | 必填 | 用途 | 客户端渲染 |
|------|------|------|------|-----------|
| `buff_type` | string | 是 | Buff类型标识 | 显示Buff图标 |
| `duration` | int | 是 | 持续回合数 | 显示持续时间（-1表示永久） |

### remove_buff 类型

| 字段 | 类型 | 必填 | 用途 | 客户端渲染 |
|------|------|------|------|-----------|
| `buff_type` | string | 是 | Buff类型标识 | 显示移除Buff动画 |

### draw_event 类型

| 字段 | 类型 | 必填 | 用途 | 客户端渲染 |
|------|------|------|------|-----------|
| `event_type` | string | 是 | 事件类型标识（客户端查找本地定义表） | 事件卡片类型 |
| `desc` | string | 否 | 事件描述文本 | 显示事件内容描述 |

### draw_item 类型

| 字段 | 类型 | 必填 | 用途 | 客户端渲染 |
|------|------|------|------|-----------|
| `item_type` | string | 是 | 道具类型标识 | 显示道具获取动画 |
| `desc` | string | 否 | 道具描述文本 | 显示道具内容描述 |

### teleport 类型

| 字段 | 类型 | 必填 | 用途 | 客户端渲染 |
|------|------|------|------|-----------|
| `from_pos` | int | 是 | 传送起点位置 | 传送动画起点 |
| `to_pos` | int | 是 | 传送终点位置 | 传送动画终点 |

### steal_buff 类型（白虎劫运）

| 字段 | 类型 | 必填 | 用途 | 客户端渲染 |
|------|------|------|------|-----------|
| `stolen_by` | string | 是 | 偷取者玩家ID | 显示偷取来源 |
| `buff_type` | string | 是 | 被偷的Buff类型 | 显示被偷Buff |

### fell_down 类型

| 字段 | 类型 | 必填 | 用途 | 客户端渲染 |
|------|------|------|------|-----------|
| `position` | int | 是 | 落坑位置 | 落坑动画位置 |

> 注：坠落伤害由衍生的 `damage` 类型 LogEntry 承担（包含 `hp_change`），fell_down 类型仅记录语义信号。

### respawn 类型

| 字段 | 类型 | 必填 | 用途 | 客户端渲染 |
|------|------|------|------|-----------|
| `checkpoint_pos` | int | 是 | 重生检查点位置 | 重生动画位置 |

### boss_damage 类型（玩家攻击Boss）

| 字段 | 类型 | 必填 | 用途 | 客户端渲染 |
|------|------|------|------|-----------|
| `damage` | int | 是 | 对Boss造成的伤害值 | 显示伤害数值动画 |
| `is_crit` | bool | 是 | 是否暴击 | 暴击特效标识 |
| `boss_remaining_hp` | int | 是 | Boss剩余HP | Boss血条更新 |

### boss_attack 类型（Boss物理攻击玩家）

| 字段 | 类型 | 必填 | 用途 | 客户端渲染 |
|------|------|------|------|-----------|
| `attack_type` | string | 是 | 攻击类型（"normal"/"crit"） | 攻击类型动画 |
| `target` | string | 是 | 目标玩家ID | 目标标识 |

> 注1：Boss攻击伤害由衍生的 `damage` 类型 LogEntry 承担（包含 `hp_change`），boss_attack 类型仅记录语义信号。
> 注2：BossAttackAction 仅用于 Boss 普通攻击/暴击。技能效果（Thunder等）通过 `boss_skill` + 衍生 `damage` LogEntry 表达，不产生 `boss_attack` LogEntry。
> 注3：Thorns反刺buff的反射伤害不产生 boss_attack LogEntry，直接产生 `damage` 类型 LogEntry（source=`buff_thorns`）。

### boss_skill 类型（Boss使用技能）

| 字段 | 类型 | 必填 | 用途 | 客户端渲染 |
|------|------|------|------|-----------|
| `skill_type` | string | 是 | 技能类型（"thunder"/"curse"/"lost"/"rest"） | 技能动画类型 |
| `targets` | string | 是 | 目标玩家ID列表（逗号分隔） | 目标标识 |

### draw_buff 类型

| 字段 | 类型 | 必填 | 用途 | 客户端渲染 |
|------|------|------|------|-----------|
| `buff_type` | string | 是（抽取成功时） | Buff类型标识 | 显示Buff获取动画 |
| `desc` | string | 否 | Buff描述文本 | 显示Buff内容描述 |

### dice_upgrade 类型

| 字段 | 类型 | 必填 | 用途 | 客户端渲染 |
|------|------|------|------|-----------|
| `from_dice` | string | 是 | 原始骰子类型 | 显示升级起点 |
| `to_dice` | string | 是 | 升级后骰子类型 | 显示升级终点 |

### add_item 类型

| 字段 | 类型 | 必填 | 用途 | 客户端渲染 |
|------|------|------|------|-----------|
| `item_type` | string | 是 | 道具类型标识 | 显示道具获得动画 |

### remove_item 类型

| 字段 | 类型 | 必填 | 用途 | 客户端渲染 |
|------|------|------|------|-----------|
| `item_type` | string | 是 | 道具类型标识 | 显示道具移除动画 |

### death 类型

| 字段 | 类型 | 必填 | 用途 | 客户端渲染 |
|------|------|------|------|-----------|
| `position` | int | 是 | 死亡发生位置 | 死亡动画位置 |
| `death_source` | string | 是 | 死亡来源标识 | 显示死亡原因（如"buff_corrupt"、"fragile_cell"） |

### dice_roll 类型

| 字段 | 类型 | 必填 | 用途 | 客户端渲染 |
|------|------|------|------|-----------|
| `dice_type` | string | 是 | 骰子类型 | 显示骰子类型 |
| `dice_steps` | int | 是 | 骰子结果 | 显示骰子数值 |

### state 类型（仅内部，不发送给客户端）

HSM 状态转换日志，仅供内部调试和 CLI playtest 使用。Builder 在 `filterClientEntries` 中过滤掉 `EntryTypeState` 类型，客户端不会收到 state entries。

| 字段 | 类型 | 必填 | 用途 | 备注 |
|------|------|------|------|------|
| `from` | string | 是 | 状态转换起点 | 仅内部日志 |
| `to` | string | 是 | 状态转换终点 | 仅内部日志 |

---

## TypeScript 类型定义

```typescript
interface LogEntry {
    timestamp: string;
    type: string;  // "action" | "mini_game" | "boss" | "decision"
    action_type?: string;
    target?: string;
    source?: string;
    metadata?: {
        // 语义明确的数值字段（替代旧的 delta）
        hp_change?: number;    // damage/heal: HP变化
        lp_change?: number;    // modify_lp: LP变化
        duration?: number;     // add_buff: Buff持续时间
        steps?: number;        // move: 移动步数

        // damage
        blocked_by?: string;
        piercing?: boolean;

        // move
        start_pos?: number;
        end_pos?: number;
        path?: number[];

        // add_buff/remove_buff/draw_buff
        buff_type?: string;

        // draw_event
        event_type?: string;
        desc?: string;  // draw_event/draw_item/draw_buff: 描述文本

        // dice_upgrade
        from_dice?: string;
        to_dice?: string;

        // add_item / remove_item / draw_item
        item_type?: string;

        // teleport
        from_pos?: number;
        to_pos?: number;

        // fell_down/respawn/death
        position?: number;
        checkpoint_pos?: number;
        death_source?: string;

        // steal_buff
        stolen_by?: string;

        // boss_damage
        damage?: number;
        is_crit?: boolean;
        boss_remaining_hp?: number;

        // boss_attack
        attack_type?: string;
        target?: string;

        // boss_skill
        skill_type?: string;
        targets?: string;

        // dice_roll
        dice_type?: string;    // dice type (gold/silver/copper/wood)
        dice_steps?: number;   // dice roll result steps
    };
}
```

---

## Go Action 实现

`internal/engine/action/types.go` 中各 Action 的 `LogEntry()` 方法设置语义明确的 Metadata 字段：

| Action | Metadata 字段 | 示例 |
|--------|--------------|------|
| RollDiceAction | `dice_type`, `dice_steps` | `dice_type: "gold", dice_steps: 6` |
| DamageAction | `hp_change: -amount` | `hp_change: -5` |
| HealAction | `hp_change: amount` | `hp_change: 3` |
| ModifyLPAction | `lp_change: amount` | `lp_change: 1` |
| MoveAction | `steps: steps`, `start_pos`, `end_pos`, `path` | `steps: 5` |
| AddBuffAction | `buff_type`, `duration: duration` | `duration: 3` |
| FellDownAction | `position` | `position: 15` |
| DeathAction | `position`, `death_source` | `death_source: "buff_corrupt"` |
| DrawBuffAction | `buff_type`（抽取成功时） | `buff_type: "divine"` |
| DiceUpgradeAction | `from_dice`, `to_dice` | `from_dice: "silver", to_dice: "gold"` |
| AddItemAction | `item_type` | `item_type: "reverse_clock"` |
| RemoveItemAction | `item_type` | `item_type: "reverse_clock"` |

---

## 相关文档

- [pkg/gamelog/README.md](../../pkg/gamelog/README.md) - GameLog系统
- [pkg/net/sync.go](../../pkg/net/sync.go) - TurnSync契约注释
- [internal/engine/action/README.md](../../internal/engine/action/README.md) - Action实现