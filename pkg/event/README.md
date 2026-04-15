# pkg/event - 事件总线系统

事件总线系统，为 Buff/道具提供统一的触发机制框架。

## 功能

- **Phase 系统**: 定义 10 种触发时机
  - **HSM 发布（状态时机）**:
    - `BeforeTurn`: 回合开始前（TurnUpkeep.Enter）
    - `OnLand`: 落地后（TurnLanded.Enter）
    - `AfterTurn`: 回合结束后（TurnEnd.Enter）
  - **Action 发布（动作时机）**:
    - `PreDamage`: 伤害应用前（隐匿、护盾拦截）
    - `PreEvent`: 事件触发前（辟邪、玄武）
    - `PreMove`: 移动前（迷途反向）
    - `OnBuffApplied`: Buff 添加后（入场效果）
    - `OnBuffRemoved`: Buff 移除前（亡语）
  - **特殊 Phase**:
    - `AnyTime`: 任何时候可用（主动触发）
    - `ItemUsed`: 道具主动使用时触发

- **EventBus**: 管理 Buff/道具的订阅和触发
  - Subscribe/Unsubscribe 订阅管理
  - Publish 发布 Phase 事件
  - Priority 优先级排序

- **Decision**: 用户决策机制
  - NeedConfirm 区分自动执行/用户确认
  - Option 选项列表
  - Action 执行动作
  - Timeout 超时处理（自动执行默认选项）

- **Context**: 执行上下文
  - Player/GameState/Data 数据传递
  - Metadata 嵌入支持

## 文件结构

```
pkg/event/
├── phase.go      # Phase 枚举定义（9种触发时机）
├── bus.go        # EventBus 结构和方法
├── decision.go   # Decision 和 Option 结构（含 Timeout 支持）
├── context.go    # Context 结构
├── bus_test.go   # 单元测试
```

## 使用示例

### 基本用法

```go
// 创建 EventBus
bus := event.NewEventBus("game-001")

// 订阅 Phase
d := event.NewDecision("是否使用护盾？", []event.Option{
    {ID: "use", Label: "使用"},
    {ID: "skip", Label: "跳过"},
})
subID := bus.Subscribe(event.PhasePreDamage, "player-001", "shield-001", "item", d)

// 发布 Phase
ctx := event.NewContext(player).WithData(10) // damage = 10
decisions := bus.Publish(event.PhasePreDamage, "player-001", ctx)

// 处理用户选择
decisions[0].Execute(0, ctx) // 选择第一个选项
```

### 超时处理

```go
// 创建带超时的决策
d := event.NewDecision("请选择：", options).
    WithTimeout(30*time.Second, 0) // 30秒超时，默认选择第一个选项

// 检查是否超时
if d.IsTimedOut(startTime) {
    d.ExecuteTimeout(ctx) // 自动执行默认选项
}

// 或使用 ExecuteTimeout
handled := d.ExecuteTimeout(ctx) // 返回 true 表示已处理超时
```

### 自动执行决策

```go
// 创建自动执行的决策（NeedConfirm=false）
d := event.NewAutoDecision("自动效果", []event.Option{
    {ID: "apply", Label: "应用", Action: func(ctx *Context) {
        // 自动执行的效果
    }},
})
// ShouldAsk() 返回 false，直接执行而不等待用户
```

### Decision Builder

```go
d := event.NewDecisionBuilder("选择目标：").
    AddOption("target-1", "目标1", action1).
    AddOption("target-2", "目标2", action2).
    SetPriority(10).
    SetTimeout(60*time.Second, 0).
    SetSource("buff-001", "buff").
    Build()
```

## Decision 字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| ID | string | 决策唯一标识 |
| Prompt | string | 提示文本 |
| Options | []Option | 可选项列表 |
| Priority | int | 执行优先级（越高越先执行） |
| Timeout | Duration | 超时时间（可选） |
| Default | int | 超时时默认选项索引 |
| NeedConfirm | bool | 是否需要用户确认 |
| Condition | func() bool | 动态条件检查 |
| OnChoice | func(int, *Context) | 选择后回调 |
| SourceID | string | 来源 ID（Buff/Item） |
| SourceType | string | 来源类型（"buff"/"item"） |

## Phase 与发布者关系

**设计原则：谁产生时机，谁发布 Phase**

| Phase | 发布者 | 说明 | 需订阅 EventBus |
|-------|--------|------|----------------|
| BeforeTurn | HSM | 回合开始前 | ✓ |
| OnLand | HSM | 落地后 | ✓ |
| AfterTurn | HSM | 回合结束后 | ✓ |
| PreDamage | Action | 伤害应用前 | ✓ |
| PreEvent | Action | 事件触发前 | ✓ |
| PreMove | Action | 移动前 | ✓ |
| OnBuffApplied | Action | Buff 添加后 | ✓ |
| OnBuffRemoved | Action | Buff 移除前 | ✓ |
| AnyTime | - | 任何时候可用 | ❌（主动触发） |
| ItemUsed | Game | 道具使用时 | ✓ |

## 测试覆盖率

91.9% statements

## Metadata 契约

**重要**：`Context.Metadata` 字段使用遵循契约文档定义。

详见：[doc/metadata/event_context.md](../../doc/metadata/event_context.md) - Context.Metadata 契约（Handler通信字段）

新增 Handler 意图信号时：
1. 选择语义明确的字段名（如 `hp_change` 而非 `val`）
2. 在契约文档更新表格
3. 确保信号消费方正确解析