# LogEntry.Metadata 契约

`gamelog.LogEntry.Metadata` 存储Action效果详情，通过 `TurnSync.Actions` 发送给客户端渲染。

**位置**：`pkg/gamelog/entry.go`

**可见性**：客户端可见（通过 `pkg/net.TurnSync` 发送）

---

## 客户端使用说明

客户端根据 `action.type` 判断应解析哪些 metadata 字段：

```typescript
// 客户端渲染逻辑
for (const action of turnSync.actions) {
    switch (action.type) {
        case "damage":
            if (action.blocked_by) {
                // 显示阻挡来源：action.blocked_by
            }
            if (action.piercing) {
                // 显示穿透效果图标
            }
            break;
        case "move":
            // action.path 包含完整移动路径
            playMoveAnimation(action.target, action.path);
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
| `source` | string | 多类型 | 效果来源标识 | `"Buff_Divine"`, `"Cell_Fragile"` |

### damage 类型

| 字段 | 类型 | 必填 | 用途 | 客户端渲染 |
|------|------|------|------|-----------|
| `delta` | int | 是 | HP变化值（负数，存储在LogEntry.Delta） | 显示伤害数值动画 |
| `blocked_by` | string | 否 | 阻挡来源Buff名称 | 显示"被XX阻挡"提示 |
| `piercing` | bool | 否 | 是否穿透防御 | 显示穿透效果图标 |

### heal 类型

| 字段 | 类型 | 必填 | 用途 | 客户端渲染 |
|------|------|------|------|-----------|
| `delta` | int | 是 | HP变化值（正数，存储在LogEntry.Delta） | 显示恢复数值动画 |

### modify_lp 类型

| 字段 | 类型 | 必填 | 用途 | 客户端渲染 |
|------|------|------|------|-----------|
| `delta` | int | 是 | LP变化值（存储在LogEntry.Delta） | 显示LP变化动画 |

### move 类型

| 字段 | 类型 | 必填 | 用途 | 客户端渲染 |
|------|------|------|------|-----------|
| `start_pos` | int | 是 | 移动起点位置 | 内部计算 |
| `end_pos` | int | 是 | 移动终点位置 | 内部计算 |
| `path` | []int | 是 | 移动路径（格子索引列表） | 播放移动动画 |

**注意**：`path` 从 JSON 反序列化后可能是 `[]float64` 或 `[]interface{}`，需要转换为 `[]int`。

### add_buff 类型

| 字段 | 类型 | 必填 | 用途 | 客户端渲染 |
|------|------|------|------|-----------|
| `buff_type` | string | 是 | Buff类型标识 | 显示Buff图标 |
| `duration` | int | 是 | 持续回合数（存储在LogEntry.Delta） | 显示持续时间（-1表示永久） |

### remove_buff 类型

| 字段 | 类型 | 必填 | 用途 | 客户端渲染 |
|------|------|------|------|-----------|
| `buff_type` | string | 是 | Buff类型标识 | 显示移除Buff动画 |

### draw_event 类型

| 字段 | 类型 | 必填 | 用途 | 客户端渲染 |
|------|------|------|------|-----------|
| `event_type` | string | 是 | 事件类型标识 | 事件卡片类型 |
| `event_name` | string | 是 | 事件中文名（显示） | 事件卡片标题 |

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
| `delta` | int | 是 | 坠落伤害（存储在LogEntry.Delta） | 显示坠落伤害 |

### respawn 类型

| 字段 | 类型 | 必填 | 用途 | 客户端渲染 |
|------|------|------|------|-----------|
| `checkpoint_pos` | int | 是 | 重生检查点位置 | 重生动画位置 |

### dice_roll 类型

| 字段 | 类型 | 必填 | 用途 | 客户端渲染 |
|------|------|------|------|-----------|
| `dice_type` | string | 是 | 骰子类型 | 显示骰子类型 |
| `dice_steps` | int | 是 | 骰子结果 | 显示骰子数值 |

### state 类型

| 字段 | 类型 | 必填 | 用途 | 客户端渲染 |
|------|------|------|------|-----------|
| `from` | string | 是 | 状态转换起点 | 内部日志 |
| `to` | string | 是 | 状态转换终点 | 内部日志 |

---

## TypeScript 类型定义

```typescript
interface Action {
    type: string;
    target: string;
    source: string;
    
    // damage/heal/modify_lp (stored in delta field)
    delta?: number;
    
    // damage
    blocked_by?: string;
    piercing?: boolean;
    
    // move
    start_pos?: number;
    end_pos?: number;
    path?: number[];
    
    // add_buff/remove_buff
    buff_type?: string;
    
    // draw_event
    event_type?: string;
    event_name?: string;
    
    // teleport
    from_pos?: number;
    to_pos?: number;
    
    // fell_down/respawn
    position?: number;
    checkpoint_pos?: number;
    
    // steal_buff
    stolen_by?: string;
    
    // state
    from_state?: string;
    to_state?: string;
}
```

---

## Builder 映射

`internal/net/builder.go` 的 `buildAction()` 方法将 LogEntry.Metadata 字段映射到 `pkg/net.Action` 展平字段：

```go
func (b *Builder) buildAction(entry gamelog.LogEntry) pkgnet.Action {
    action := pkgnet.Action{
        Type:   string(entry.ActionType),
        Target: entry.Target,
        Source: entry.Source,
    }
    
    // 从 Metadata 提取并展平到 Action 字段
    switch entry.ActionType {
    case "move":
        action.Path = metadataGetIntSlice(meta, "path")
        action.StartPos = metadataGetInt(meta, "start_pos")
        action.EndPos = metadataGetInt(meta, "end_pos")
    case "add_buff":
        action.BuffType = metadataGetString(meta, "buff_type")
    // ... 其他类型
    }
}
```

---

## 相关文档

- [pkg/gamelog/README.md](../../pkg/gamelog/README.md) - GameLog系统
- [doc/internal/net_protocol.md](../internal/net_protocol.md) - 网络协议设计
- [pkg/net/README.md](../../pkg/net/README.md) - pkg/net协议层