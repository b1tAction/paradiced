# internal/engine/hsm - Hierarchical State Machine

分层状态机（Hierarchical State Machine）实现，用于管理游戏流程的三层状态结构。

## 概述

HSM 包提供三层架构的状态机：
- **Layer 1 (Global States)**: 全局状态，控制游戏大阶段
- **Layer 2 (Turn States)**: 回合状态，控制玩家回合流程
- **Layer 3 (Interrupt States)**: 中断状态，处理需要用户确认的决策

## 三层架构

```
┌─────────────────────────────────────────────────────────┐
│ Layer 1: Global States                                  │
│ MatchInit → RoundMiniGame → RoundPrep → TurnLoop        │
│                                    ↓                     │
│                               GameOver                   │
└─────────────────────────────────────────────────────────┘
              │
              ↓
┌─────────────────────────────────────────────────────────┐
│ Layer 2: Turn States (在 TurnLoop 内循环执行)            │
│ TurnUpkeep → MainAction → TurnMoving → TurnLanded       │
│                                            ↓             │
│                                         TurnDraw         │
│                                            ↓             │
│                                     TurnBossBattle       │
│                                            ↓             │
│                                         TurnEnd          │
└─────────────────────────────────────────────────────────┘
              │
              ↓ (需要用户决策时)
┌─────────────────────────────────────────────────────────┐
│ Layer 3: Interrupt States                               │
│ WaitDecision (暂停当前状态，等待用户输入)                  │
└─────────────────────────────────────────────────────────┘
```

## 核心文件

| 文件 | 描述 |
|------|------|
| `state.go` | State 接口定义、StateContext 上下文、Metadata key 常量 |
| `state_id.go` | StateID 枚举定义和辅助方法 |
| `state_stack.go` | StateStack 实现（入栈/出栈机制） |
| `hsm.go` | HSM 主结构、状态转移逻辑、Snapshot 机制 |
| `adapter.go` | EventBusAdapter、MapEngineAdapter、GameWrapper |
| `global_states.go` | Layer 1 全局状态实现 |
| `turn_states.go` | Layer 2 回合状态实现（含 TurnDrawState） |
| `interrupt_states.go` | Layer 3 中断状态实现 |

## State 接口

```go
type State interface {
    ID() StateID
    Enter(ctx *StateContext)
    Update(ctx *StateContext) StateID
    Exit(ctx *StateContext)
    CanTransitionTo(target StateID) bool
}
```

## StateContext

StateContext 提供状态执行的上下文数据，包含：
- **Metadata**: 嵌入的元数据存储（类型安全的键值对）
- **HSM**: HSM 实例引用（单一真相来源）
- **Player**: 当前玩家引用
- **Phase/PhaseData**: 阶段触发数据
- **Decision/Decisions**: 待处理决策
- **Stack**: 状态栈引用

通过 HSM 可以访问 Game、Bus、MapEngine：
- `ctx.GetGame()` → 通过 HSM 获取游戏数据
- `ctx.GetBus()` → 通过 HSM 获取 EventBus
- `ctx.GetMapEngine()` → 通过 HSM 获取地图引擎

```go
// 使用 WithHSM 模式（单一真相来源）
hsmInst := NewHSM(game)
hsmInst.SetMapEngine(mapEngine)
ctx := NewStateContext().
    WithHSM(hsmInst).
    WithPlayer(player)

// 使用 Metadata 方法
ctx.SetInt(KeyDiceSteps, 5)
ctx.SetBool(KeySkipTurn, true)
ctx.GetDiceSteps() // 返回 5
```

## Layer 1: Global States

| State | 描述 |
|-------|------|
| MatchInit | 初始化游戏：生成地图、分配阵营 |
| RoundMiniGame | 小游戏阶段：所有玩家参与竞争（Boss不参与） |
| RoundPrep | 回合准备：根据排名分配骰子类型 |
| TurnLoop | 回合循环：管理玩家回合轮转（Boss排在末尾） |
| GameOver | 游戏结束：结算和排名 |

## Layer 2: Turn States

| State | 描述 | Phase 触发 |
|-------|------|-----------|
| TurnUpkeep | 回合准备：检查死亡/跳过、触发 BeforeTurn（Boss跳过）、毒瘴恶性事件抽取 | PhaseBeforeTurn |
| MainAction | 主要行动：等待骰子/道具/技能输入 | - |
| TurnMoving | 移动处理：HSM 预扫描路径、迷途 Steps 修改、CheckPoint 拆分、FellDown | PhasePreMove (HSM 发布) |
| TurnCheckpoint | CheckPoint 结算：DrawItemAction（宝箱道具） | - |
| TurnLanded | 落地处理：CellType 行为矩阵、触发 OnLand、捕获 DrawType 配置 | PhaseOnLand |
| TurnDraw | 概率抽取：根据 cell 的 DrawType 和 prob 配置进行事件/道具抽取 | PhasePreEvent (DrawEventAction) |
| TurnBossBattle | Boss 战斗：玩家攻击Boss/Boss反击（Boss格玩家或Boss回合） | - |
| TurnEnd | 回合结束：触发 AfterTurn、TickBuffs（Boss只TickBuffs不触发AfterTurn） | PhaseAfterTurn |

### TurnDrawState 详解

`TurnDrawState` 是统一的概率抽取状态，用于处理地图格子的概率抽取配置：

**状态转移**：
```
TurnLanded (检测到 DrawType != none)
    ↓
TurnDraw (执行概率抽取)
    ↓
TurnEnd
```

**执行逻辑**：
1. 从 TurnLandedState 接收 DrawType、ProbGood、ProbNeutral、ProbBad 配置
2. 创建 ActionContext 并设置概率配置
3. 根据 DrawType 执行对应的抽取动作：
   - `DrawTypeEvent`: 执行 `DrawEventAction` → 触发 `PhasePreEvent` → 调用事件 handler
   - `DrawTypeItem`: 执行 `DrawItemAction` → 获得道具

**概率抽取机制**：
- 使用 `rng.DrawEngine.DrawWithProb` 方法
- 按照 probGood/probNeutral/probBad 的权重选择池
- 如果概率总和 < 1.0，剩余概率从全部 items 中抽取（不进行池过滤）

### TurnBossBattleState 详解

`TurnBossBattleState` 是 Boss 战斗状态，根据当前玩家身份分两个分支：

**状态转移**：
```
MainAction (玩家在Boss格掷骰)
    ↓
TurnBossBattle (玩家攻击Boss)
    ↓
TurnEnd

TurnUpkeep (Boss回合)
    ↓
TurnBossBattle (Boss反击)
    ↓
TurnEnd
```

**玩家攻击Boss分支**：
- 骰子点数 = 基础伤害值
- 暴击伤害 = 基础伤害 × 2，暴击概率与骰子品质相关
- Boss被击败 → 设置 BossDefeated 标记 → TurnEnd → TurnLoop → GameOver

**Boss反击分支**：
- Boss回合排在所有玩家回合之后
- 如果没有玩家在Boss格，Boss回合空转
- Boss攻击类型基于Boss格存活玩家的平均LP
- Boss攻击（普通1点/暴击2点）可被隐匿Buff拦截（PhasePreDamage）
- Boss技能从技能池随机抽取（等权重）

**Boss击败后流程**：
- `KeyBossDefeated` 同时存于 StateContext 和 Game.RoundData（跨tick持久化）
- TurnLoop.Update() 检查 BossDefeated → 转移到 GameOver
- Boss被击败后跳过 AfterTurn 效果

## Layer 3: Interrupt States

| State | 描述 |
|-------|------|
| WaitDecision | 等待决策：暂停当前状态，等待用户选择 |

## decisionStateResetter 接口

决策状态（TurnUpkeep、TurnLanded）缓存 pending decisions，在决策被解决后需要清理。`decisionStateResetter` 接口确保缓存被正确清空：

```go
type decisionStateResetter interface {
    ResetPendingDecisions()
}
```

- `TurnUpkeepState.ResetPendingDecisions()` - 清空缓存的 decisions 列表
- `TurnLandedState.ResetPendingDecisions()` - 清空缓存的 decisions 列表
- `HSM.OnUserChoice()` 在决策解决后调用 `clearResolvedDecisionFromState()`，同时清理 `StateContext.Decisions` 和状态内部的缓存

## Phase 触发机制

设计原则：**谁产生时机，谁发布 Phase**

| Phase | 发布者 | 触发位置 |
|-------|--------|----------|
| BeforeTurn | HSM | TurnUpkeep.Enter() |
| OnLand | HSM | TurnLanded.Enter() |
| AfterTurn | HSM | TurnEnd.Enter() |
| PreMove | HSM | TurnMoving.Enter() |
| PreEvent | Action | DrawEventAction.Execute() |
| PreDamage | Action | DamageAction / BossDamageAction / BossAttackAction |
| ItemUsed | Game | Game.UseItem() / MainAction.OnUseItem() |

## 毒瘴(Poison)恶性事件处理

TurnUpkeepState 在 BeforeTurn phase 触发后检查 `draw_bad_event` flag：

```go
// Step 5 in TurnUpkeepState.Enter()
if triggerCtx.GetBoolOrDefault("draw_bad_event", false) {
    drawAction := engineaction.NewDrawEventAction(player, "Poison_BadEvent")
    actionCtx.SetCellDraw(0, 0, 1.0)  // 100% bad probability
    actionCtx.ExecuteAction(drawAction)
    if drawAction.DrawnType.IsValid() {
        runEventEffect(drawAction.DrawnType, player, actionCtx)
    }
}
```

## OnUseItem 道具消耗流程

MainActionState.OnUseItem 处理道具使用的完整流程：

1. **Publish PhaseItemUsed** - 触发道具 Handler 执行
2. **Execute decisions** - 立即执行 Handler 产生的 decisions
3. **Run derived actions** - 执行 Handler 推送的 DerivedAction（如 DiceUpgradeAction）
4. **Apply dice upgrade** - 读取 ActionContext metadata 更新 DiceManager
5. **Consume item** - 追加 RemoveItemAction 消耗道具

## 与 Action 系统集成

Turn States 通过 ActionContext 执行 Action：

```
TurnUpkeep → ExecuteAction(ModifyLPAction) → PreTrigger → Execute → PostTrigger → EventLog
TurnMoving → HSM publishes PhasePreMove → CalculatePath → ExecuteAction(MoveAction) → Execute → EventLog
TurnCheckpoint → ExecuteAction(DrawItemAction) → Execute → EventLog
TurnLanded → 捕获 cell 配置 → 转移到 TurnDraw
TurnDraw → ExecuteAction(DrawEventAction/DrawItemAction) → Execute → EventLog
TurnBossBattle → ExecuteAction(BossDamageAction/BossAttackAction/BossSkillAction) → Execute → EventLog
```

## 使用示例

```go
// 创建 HSM
game := engine.NewGame("match1", 0)
hsm := NewHSM(game)

// 注册状态
RegisterGlobalStates(hsm)
RegisterTurnStates(hsm)
RegisterInterruptStates(hsm)

// 启动 HSM
ctx := NewStateContext().WithHSM(hsm)
hsm.Start(StateMatchInit, ctx)

// 更新循环
for hsm.IsRunning() {
    hsm.Update(ctx)
}

// 处理用户输入
hsm.OnRollDice(6, ctx)     // 骰子输入
hsm.OnUserChoice(1, ctx)   // 决策选择

// 创建 Snapshot（持久化）
snapshot := hsm.CreateSnapshot()

// 从 Snapshot 恢复
hsm.RestoreFromSnapshot(snapshot)
```

## Adapter 系统

HSM 使用 Adapter 模式隔离外部包依赖：

- **EventBusAdapter**: 隔离 pkg/event 包
- **MapEngineAdapter**: 隔离 internal/gamemap 包
- **GameWrapper**: 实现 protocol.Game 接口
- **ProtocolMapEngineWrapper**: 实现 protocol.MapEngine 接口

```go
// 使用 Wrapper 创建 Adapter
busAdapter := NewEventBusWrapper(game.Bus)
mapAdapter := NewMapEngineWrapper(mapEngine)
gameWrapper := NewGameWrapper(game)
```

## Metadata 契约

**重要**：`StateContext.Metadata` 字段使用遵循契约文档定义。

详见：[doc/metadata/hsm_context.md](../../../doc/metadata/hsm_context.md) - StateContext.Metadata 契约（状态机通信字段）

所有 Metadata key 使用预定义常量（KeyXxx），确保类型安全和命名统一。

新增状态标记时：
1. 在 `state.go` 添加常量定义（KeyXxx）
2. 如常用，添加便捷方法（WithXxx/GetXxx/SetXxx/IsXxx）
3. 在契约文档更新表格

## 测试

```bash
go test ./internal/engine/hsm/... -v
```
