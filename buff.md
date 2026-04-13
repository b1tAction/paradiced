## 状态：已重构解决（2026-04）

上述关于 Buff/Event/Item 散弹式修改的问题已通过 **统一注册表模式（Unified Registry Pattern）** 完成重构。

### 重构后的架构

新增以下文件实现单一数据源：

| 文件 | 说明 |
|------|------|
| `internal/core/registry.go` | 统一注册表结构和方法 |
| `internal/core/special_effect.go` | SpecialEffect 枚举替代字符串标记 |
| `internal/core/buff_init.go` | Buff 注册初始化（init() 时执行） |
| `internal/core/event_init.go` | Event 注册初始化 |
| `internal/core/item_init.go` | Item 注册初始化 |

### 新增 Buff/Event/Item 的流程（重构后）

只需 **2 处修改**：
1. 在枚举定义中添加新常量（`const BuffTypeNewBuff`）
2. 在对应 `_init.go` 文件中添加一行注册

```go
// buff_init.go
GlobalRegistry.RegisterBuff(&BuffDefinition{
    Type:        BuffTypeNewBuff,
    Eval:        EvaluationGood,
    EnglishName: "NewBuff",      // 用于 String()
    Name:        "新Buff",       // 中文显示名
    Desc:        "描述",
    Duration:    3,
    HPPerTurn:   1,
    Phases:      []event.Phase{event.PhaseAfterTurn},
    Priority:    50,
}, nil)  // 可选：传入自定义 Handler
```

### 核心改进

1. **单一数据源**：所有定义在 `_init.go` 文件中统一注册
2. **自动推导**：`String()`、`Evaluation` 映射、分类列表由注册表自动生成
3. **逻辑内聚**：Handler 在注册时传入，数据和逻辑统一管理
4. **类型安全**：`SpecialEffect` 枚举替代字符串标记

---

## 原始问题分析（已解决）

你的直觉非常敏锐。从目前的代码来看，这是一个具有很高水准的派对游戏服务端架构，但在具体的流程控制和数据定义上，确实存在结构性的代码冗余。

### 1. 整体架构评价（优秀的骨架设计）

整个项目采用了**领域驱动设计（DDD）**和**权威服务器（Authoritative Server）**模式，将状态维护和计算完全收拢在服务端，这对于搭配 Godot 等客户端引擎进行纯渲染交互来说，是非常契合的联机游戏架构，能有效杜绝客户端作弊。

项目中几个非常亮眼的设计：
* **EventBus 与 Hook 机制**：将游戏内繁杂的“四神兽阵营被动”和“道具Buff联动”从核心主循环中彻底剥离出来。通过 `PhaseOnBuffApplied` 等生命周期事件，实现了极高的解耦。
* **状态同步与快照系统**：`StateMachine` 配合 `FlowSnapshot`，严密地处理了异步的“等待玩家决策”机制以及断线重连问题。
* **元数据容器（Metadata）**：利用 Go 的匿名嵌套（`*util.Metadata`），让 `Context` 和 `Player` 等结构体获得了类型安全的动态属性扩展能力，极其优雅且利于 JSON 序列化传输。

### 2. TurnFlow 中的“大量重复”解析与优化

正如你所感受到的，`internal/engine/turn_flow.go` 中的 `TurnFlow` 回合流转系统存在明显的模板代码（Boilerplate）重复。

**重复表现：**
多个阶段（如 `executeBeforeTurn`、`executePreEvent`、`executeAfterTurn`、`executeOnLand`）的内部逻辑高度同质化。它们本质上都在重复做同一套流程：
1. 构造 `StepResult` 结构。
2. 调用 `tf.StateMachine.TriggerPhase` 触发特定时机。
3. 检查是否产生了 `pendingDecisions`，如果有则暂停状态机等待用户确认，没有则直接返回 `Success = true`。

**重构方案：**
你可以提取一个通用的“标准阶段处理”函数，将重复的样板代码抹平：

```go
// 提取出的通用阶段触发器
func (tf *TurnFlow) triggerStandardPhase(step TurnStep, phase event.Phase, player *core.Player) *StepResult {
    result := &StepResult{Step: step}
    pendingDecisions := tf.StateMachine.TriggerPhase(phase, player)
    
    if len(pendingDecisions) > 0 {
        result.Decisions = pendingDecisions
        result.Success = false // 拦截并等待用户输入
    } else {
        result.Success = true
        result.PlayerUpdated = true
    }
    return result
}
```

有了这个底层方法后，你原来的多个函数就可以缩减为一行：

```go
func (tf *TurnFlow) executeBeforeTurn(player *core.Player) *StepResult {
    return tf.triggerStandardPhase(StepBeforeTurn, event.PhaseBeforeTurn, player)
}

func (tf *TurnFlow) executePreEvent(player *core.Player) *StepResult {
    return tf.triggerStandardPhase(StepPreEvent, event.PhasePreEvent, player)
}
```
这样不仅消除了大量的重复，未来如果要给所有 Phase 增加统一的日志记录或超时打点，也只需改动这一个通用方法。

### 3. 另一处隐藏的重复：数据字典（Registry）的散弹式修改

除了 `TurnFlow`，在 `internal/core` 目录下（如 `buff.go`, `event.go`, `item.go`），由于 Go 语言缺乏高级枚举特性，也产生了明显的冗余。

当你想要新增一个 `Buff` 时，当前的设计强迫你必须在多处进行修改：
1. 在 `const` 中增加 `BuffType` 枚举。
2. 修改 `String()` 里的 map 映射。
3. 修改 `GetEvaluation()` 里的 map 映射。
4. 修改 `GetBuffDefinition()` 里的 map 结构。
5. 修改 `NewBuffRegistry()` 中的分类切片。

**重构方案：**
将硬编码的多个独立 map 收敛为单一的**数据驱动（Data-Driven）**设计。你可以定义一个全局的 `BuffDefinitionsMap`，然后在包的 `init()` 函数中，动态遍历这个单一的数据源，自动解析生成对应的 `String` 映射和基于 `Evaluation` 的良性/恶性分类。长远来看，这部分配置甚至可以完全从 Go 代码中抽离，变成 JSON 或 Excel 表格供策划配置，服务端启动时一键加载入内存。

**总结：**
这套架构的骨架（状态管线、事件总线、动态属性）设计得极具前瞻性，能够从容应对桌游复杂的技能联动。当前的重复代码多属于微观实现上的“体力活”，只需通过提炼通用函数和引入数据驱动配置，就能让项目变得极其清爽易维护。


