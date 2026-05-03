# YAML 定义系统

## 概述

YAML 定义系统是游戏内容的唯一数据源（Single Source of Truth），所有 Event/Buff/Item 的静态元数据集中定义在 `pkg/resource/paradiced.yml`，通过 `go:embed` 在启动时加载。

**设计原则**：
- 定义数据（名称、描述、评分等）存储在 YAML，业务逻辑（Handler 处理、触发时机等）保留在 Go 代码
- Definition 与 Instance 分离：Definition 是静态配置，Instance 是运行时动态状态
- 前端不再硬编码中文名称/描述，从后端推送的 DefinitionsConfig 中获取

## 文件结构

```
pkg/resource/
├── paradiced.yml       # 游戏内容定义（Event/Buff/Item 元数据）
├── resource.go         # 加载逻辑：LoadDefinitionsFromYAML、parseEvaluation
├── default.json        # 地图配置（JSON 格式）
└── resource_test.go    # 测试

pkg/constants/
├── buff.go             # BuffDefinition 结构体 + BuffType.IsFaction() / IsDraw()
├── event.go            # EventDefinition 结构体
├── item.go             # ItemDefinition 结构体

internal/net/builder.go # BuildDefinitionsConfig() → 协议推送
pkg/net/sync.go         # DefinitionsConfig / BuffDefinitionConfig 协议类型
```

## YAML 定义格式

### 事件定义

```yaml
events:
  herb:
    evaluation: mild_good    # 命名常量或数字（0-100）
    english_name: Herb       # 英文标识名
    name: 采集到草药          # 中文显示名
    desc: 在路边发现了草药，恢复了体力  # 中文描述
```

### Buff 定义

```yaml
buffs:
  divine:
    evaluation: very_good
    english_name: Divine
    name: 神眷
    desc: 接下来3回合保持LP+1状态
    duration: 3               # 回合数，-1表示永久
```

### 道具定义

```yaml
items:
  any_door:
    evaluation: neutral
    english_name: AnyDoor
    name: 任意门
    desc: 前往指定玩家身边
```

## 评分系统（Evaluation）

YAML 中的 `evaluation` 字段支持两种格式：

| 命名常量 | 数值 | 分类 |
|----------|------|------|
| `very_bad` | 10 | Bad (≤40) |
| `bad` | 25 | Bad (≤40) |
| `mild_bad` | 35 | Bad (≤40) |
| `neutral` | 50 | Neutral (41-65) |
| `mixed` | 55 | Neutral (41-65) |
| `mild_good` | 70 | Good (>65) |
| `good` | 80 | Good (>65) |
| `very_good` | 90 | Good (>65) |
| `excellent` | 100 | Good (>65) |

也可以直接使用 0-100 的数字字符串，例如 `evaluation: "65"`。

`parseEvaluation()` 函数负责将 YAML 字符串转换为 `constants.Evaluation`。

## 类型安全（Typed Map Keys）

`DefinitionSet` 使用类型安全的枚举 map key：

```go
type DefinitionSet struct {
    Events map[constants.EventType]*constants.EventDefinition
    Buffs  map[constants.BuffType]*constants.BuffDefinition
    Items  map[constants.ItemType]*constants.ItemDefinition
}
```

`LoadDefinitionsFromYAML` 对每个 YAML key 进行验证：
- 通过 `ParseEventType/ParseBuffType/ParseItemType` 将字符串 key 转换为枚举类型
- 如果 key 对应的枚举类型不存在（`EventTypeNone`/`BuffTypeNone`/`ItemTypeNone`），返回错误
- 这确保 YAML 中不会出现未注册的类型 key

## Buff 分类体系

BuffType 有多层分类方法，用于不同的游戏逻辑场景：

| 方法 | 含义 | 适用的 Buff |
|------|------|------------|
| `IsPositive()` | 正面效果 | Divine, Rain, Exorcism, Fire, Hidden |
| `IsNegative()` | 负面效果 | Curse, Lost, Corrupt, Poison |
| `IsBoss()` | Boss 相关 | Thorns（Boss自身）、DeathMark |
| `IsHidden()` | 隐藏行动 | DeathMark |
| `IsFaction()` | 阵营被动 | Fire（朱雀离火） |
| `IsDraw()` | 可抽签 | !IsBoss && !IsHidden && !IsFaction |

`IsDraw()` 决定哪些 Buff 可以进入随机抽签池（DrawBuffAction）。Boss buff、Hidden buff 和 Faction buff 不参与抽签。

## 定义推送（DefinitionsConfig）

后端通过 `BuildDefinitionsConfig()` 将 YAML 定义转换为协议格式，随 `OpStartGameAck` 推送到前端：

```go
func BuildDefinitionsConfig() pkgnet.DefinitionsConfig {
    // 遍历 GlobalDefinitionSet，将每个 Definition 转换为协议 Config
    // 包括计算字段：Category（从 Evaluation 推导）、IsPositive/IsNegative/IsBoss/IsHidden/IsFaction/IsDraw
}
```

前端存储在 Zustand `useGameStore.getState().definitions`，用于：
- 渲染 Event/Buff/Item 名称（替代硬编码中文）
- 渲染 Buff 描述（替代 `BUFF_EFFECTS` map）
- 显示 Buff 分类信息

## 初始化流程

```go
// pkg/resource/resource.go
var GlobalDefinitionSet *DefinitionSet

func init() {
    set, err := LoadDefinitions()
    if err != nil {
        panic(fmt.Sprintf("failed to load definitions: %v", err))
    }
    GlobalDefinitionSet = set
}
```

各 Registry（EventRegistry/BuffRegistry/ItemRegistry）在 `init()` 中读取 `GlobalDefinitionSet` 注册 Definition 和 HandlerConfig。

## 新增定义步骤

1. 在 `pkg/constants/` 中添加新的枚举常量（EventType/BuffType/ItemType）和 `Parse*` 函数
2. 在 `pkg/resource/paradiced.yml` 中添加对应定义
3. 在 `internal/engine/*_registry.go` 中注册 HandlerConfig（业务逻辑）
4. 运行测试验证：
   - `go test ./pkg/resource/...` 验证 YAML 加载
   - `go test ./internal/net/...` 验证 DefinitionsConfig 推送
   - `go test ./internal/engine/...` 验证 Handler 行为
5. 前端：definitions 自动包含新定义，无需额外修改