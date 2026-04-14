# pkg/constants - Unified Enum Types

统一枚举类型定义包，所有用于 JSON 通信的枚举类型集中于此。

## 设计原则

1. **类型定义**：使用 `string` 类型，而非 `int` + `String()` 方法
2. **命名规范**：常量名 PascalCase（Go 规范），值 snake_case（通信规范）
3. **类型安全**：每个枚举是独立类型，不会混淆
4. **JSON 友好**：直接序列化为 snake_case，无需转换

## 类型定义示例

```go
// Phase - 触发时机
type Phase string

const (
    PhaseBeforeTurn    Phase = "before_turn"
    PhaseOnLand        Phase = "on_land"
    PhasePreDamage     Phase = "pre_damage"
    // ...
)

// BuffType - Buff 类型
type BuffType string

const (
    BuffTypeCurse   BuffType = "curse"
    BuffTypeDivine  BuffType = "divine"
    // ...
)

// ActionType - Action 类型（保持 pkg/action 定义）
type ActionType string

const (
    ActionDamage  ActionType = "damage"
    ActionHeal    ActionType = "heal"
    // ...
)
```

## 包结构

```
pkg/constants/
├── phase.go      # Phase 枚举
├── buff.go       # BuffType 枚举
├── event.go      # EventType 枚举
├── item.go       # ItemType 枚举
├── effect.go     # SpecialEffect 枚举
├── faction.go    # Faction 枚举
├── cell.go       # CellType 枚举
├── state.go      # StateID 枚举
├── entry.go      # EntryType 枚举
├── source.go     # ActionSource 枚举（新增）
└── README.md     # 本文档
```

## 迁移计划

| 类型 | 原位置 | 新位置 | 状态 |
|------|--------|--------|------|
| ActionType | pkg/action | pkg/constants (alias) | ✓ 已完成 |
| Phase | pkg/event | pkg/constants | 待迁移 |
| BuffType | internal/core/buff | pkg/constants | 待迁移 |
| EventType | internal/core/event | pkg/constants | 待迁移 |
| ItemType | internal/core/item | pkg/constants | 待迁移 |
| SpecialEffect | internal/core/types | pkg/constants | 待迁移 |
| Faction | internal/core | pkg/constants | 待迁移 |
| CellType | internal/gamemap | pkg/constants | 待迁移 |
| StateID | internal/engine/hsm | pkg/constants | 待迁移 |
| EntryType | pkg/gamelog | pkg/constants | 待迁移 |
| ActionSource | - | pkg/constants (新增) | 待添加 |

## 使用方式

```go
import "github.com/b1tAction/Fated/pkg/constants"

// 使用强类型常量
phase := constants.PhaseBeforeTurn
buffType := constants.BuffTypeDivine

// 类型安全
func RegisterBuff(type constants.BuffType) {
    // 只接受 BuffType，不接受其他 string
}
```