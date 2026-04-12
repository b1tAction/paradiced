# pkg/event - Event Bus System

事件总线系统，为 Buff/道具提供统一的触发机制框架。

## 功能

- **Phase 系统**: 定义 6 种触发时机
  - `BeforeTurn`: 回合开始前
  - `OnMove`: 移动时
  - `OnLand`: 落地后
  - `PreEvent`: 事件触发前
  - `PreDamage`: 受伤前
  - `AfterTurn`: 回合结束后
  - `AnyTime`: 任何时候可用（主动触发）

- **EventBus**: 管理 Buff/道具的订阅和触发
  - Subscribe/Unsubscribe 订阅管理
  - Publish 发布 Phase 事件
  - Priority 优先级排序

- **Decision**: 用户决策机制
  - NeedConfirm 区分自动执行/用户确认
  - Option 选项列表
  - Action 执行动作

- **Context**: 执行上下文
  - Player/GameState/Data 数据传递

## 文件结构

```
pkg/event/
├── phase.go      # Phase 枚举定义
├── bus.go        # EventBus 结构和方法
├── decision.go   # Decision 和 Option 结构
├── context.go    # Context 结构
└── bus_test.go   # 单元测试
```

## 使用示例

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

## 测试覆盖率

91.9% statements