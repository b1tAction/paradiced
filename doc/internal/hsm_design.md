# HSM 分层状态机设计文档

## 概述

本文档描述《命运骰子》游戏的分层状态机（HSM - Hierarchical State Machine）架构设计。该设计替代原有的 `TurnFlow` 流水线架构，解决双轨制控制问题，实现单一数据源（SSOT）。

## 设计目标

1. **单一数据源**：状态机的 `CurrentState` 是游戏状态的唯一真理
2. **断线重连支持**：只需保存 `GlobalStateID` 和 `PlayerStateID` 即可恢复
3. **非线性跳跃能力**：支持跳过阶段、打断、强制进入特殊状态
4. **Phase 与 State 完美契合**：Phase 作为事件广播时机嵌入 State

## 实现进度

### Phase 1: 核心框架 ✅ (已完成)

| 组件 | 文件 | 状态 | 说明 |
|------|------|------|------|
| StateID | state_id.go | ✅ | 三层枚举定义 |
| State接口 | state.go | ✅ | Enter/Update/Exit生命周期 |
| StateContext | state.go | ✅ | 状态上下文数据传递 |
| Adapter接口 | adapter.go | ✅ | EventBus/MapEngine/Game适配器 |
| StateStack | state_stack.go | ✅ | 中断入栈/出栈机制 |
| HSM主结构 | hsm.go | ✅ | 状态注册、转移、生命周期管理 |
| 全局状态 | global_states.go | ✅ | 6个Layer 1状态实现 |
| 单元测试 | *_test.go | ✅ | 55+测试用例 |

### Phase 2: 回合层状态 ✅ (已完成)

| 组件 | 文件 | 状态 | 说明 |
|------|------|------|------|
| TurnUpkeepState | turn_states.go | ✅ | PhaseBeforeTurn触发、SkipTurn/IsDead检查 |
| MainActionState | turn_states.go | ✅ | 等待道具/骰子选择、超时处理 |
| TurnMovingState | turn_states.go | ✅ | 路径计算、Fragile/Fog处理 |
| TurnLandedState | turn_states.go | ✅ | PhaseOnLand触发、CellType检查 |
| TurnEventState | turn_states.go | ✅ | PhasePreEvent触发、事件抽取 |
| TurnEndState | turn_states.go | ✅ | PhaseAfterTurn触发、TickBuffs、阵营充能 |

### Phase 3: 中断层状态 ✅ (已完成)

| 组件 | 文件 | 状态 | 说明 |
|------|------|------|------|
| WaitDecisionState | interrupt_states.go | ✅ | 决策等待、超时处理、用户选择执行 |

## 文件结构

```
internal/engine/hsm/
├── state.go           # State接口和StateContext定义
├── state_id.go        # StateID三层枚举
├── adapter.go         # EventBusAdapter、MapEngineAdapter、GameWrapper
├── hsm.go             # HSM主结构、OnRollDice/OnUseItem处理、HSMSnapshot
├── state_stack.go     # StateStack入栈出栈
├── global_states.go   # Layer 1全局状态实现 (已完成)
├── turn_states.go     # Layer 2回合状态实现 (已完成)
├── interrupt_states.go # Layer 3中断状态实现 (已完成)
├── README.md          # HSM包文档
├── global_states_test.go # Layer 1测试
├── turn_states_test.go   # Layer 2测试 (已完成)
├── interrupt_states_test.go # Layer 3测试 (已完成)
├── hsm_test.go        # HSM主结构测试
├── state_id_test.go   # StateID测试
├── state_stack_test.go # StateStack测试
└── state_test.go      # StateContext测试
```

### 🌐 第一层：全局对局层 (GlobalGameState)

管理所有玩家共同参与的"大轮次（Round）"生命周期。

```
┌─────────────────────────────────────────────────────────────────┐
│                      GlobalGameState                             │
├─────────────────────────────────────────────────────────────────┤
│  State_Match_Init ──► State_Round_MiniGame ──► State_Round_Prep │
│                              │                     │            │
│                              ▼                     ▼            │
│                       [等待小游戏结果]      State_Turn_Loop     │
│                                                    │            │
│                              State_Boss_Battle ◄──┼──► [循环]   │
│                              │                     │            │
│                              ▼                     ▼            │
│                       State_Game_Over         [下一玩家]        │
└─────────────────────────────────────────────────────────────────┘
```

#### 状态定义 (已实现)

| StateID | 结构体名称 | 行为 | Phase触发 | 转移条件 |
|---------|------------|------|-----------|----------|
| `StateMatchInit` (100) | `MatchInitState` | 初始化标记、设置metadata | - | Update自动返回StateRoundMiniGame |
| `StateRoundMiniGame` (101) | `RoundMiniGameState` | 等待小游戏排名 | - | 收到所有排名后返回StateRoundPrep |
| `StateRoundPrep` (102) | `RoundPrepState` | 根据排名分配骰子、增加Round计数 | - | Update自动返回StateTurnLoop |
| `StateTurnLoop` (103) | `TurnLoopState` | 管理回合队列、检查Boss触发 | - | StartPlayerTurn→StateTurnUpkeep；reachedEnd→StateBossBattle |
| `StateBossBattle` (104) | `BossBattleState` | Boss战触发玩家记录 | - | OnBossDefeated→StateGameOver |
| `StateGameOver` (105) | `GameOverState` | 终态，记录winner | - | 无转移（终态） |

### 🔄 第二层：玩家回合子层 (PlayerTurnState)

当全局处于 `State_Turn_Loop` 时，当前行动玩家进入此子状态机。

```
┌─────────────────────────────────────────────────────────────────┐
│                      PlayerTurnState                             │
│              (父状态: State_Turn_Loop)                            │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  S_TURN_UPKEEP ──► S_MAIN_ACTION ──► S_TURN_MOVING              │
│       │                  │               │                       │
│       ▼                  ▼               ▼                       │
│  [SkipTurn检查]     [等待道具选择]    [路径计算]                  │
│       │                  │               │                       │
│       │                  │               ├─► Fragile坠落?        │
│       │                  │               │     └─► S_TURN_END    │
│       │                  │               │                       │
│       └──────────────────┴───────────────► S_TURN_LANDED        │
│                                            │                     │
│                                            ▼                     │
│                                       S_TURN_EVENT               │
│                                            │                     │
│                                            ▼                     │
│                                       S_TURN_END                 │
│                                            │                     │
│                                            ▼                     │
│                                       [NextTurn / 父状态循环]    │
└─────────────────────────────────────────────────────────────────┘
```

#### 状态定义

| StateID | 状态名称 | 行为 | Phase触发 | 转移条件 |
|---------|----------|------|-----------|----------|
| `S_TURN_UPKEEP` | 回合准备 | 检查SkipTurn、IsDead | `PhaseBeforeTurn` | 可行动→MainAction；不可→TurnEnd |
| `S_MAIN_ACTION` | 主行动 | 等待道具/骰子选择 | `PhaseItemUsed`道具可用 | 收到RollDice→TurnMoving |
| `S_TURN_MOVING` | 移动结算 | 路径计算、Fragile/Fog处理 | Action发布`PhasePreMove` | 正常→TurnLanded；坠落→TurnEnd |
| `S_TURN_LANDED` | 落地结算 | 触发落地事件 | `PhaseOnLand` | 自动进入TurnEvent |
| `S_TURN_EVENT` | 事件结算 | 抽取事件、执行效果 | Action发布`PhasePreEvent`/`PhasePreDamage` | 完成后→TurnEnd |
| `S_TURN_END` | 回合收尾 | TickBuff、死亡检查 | `PhaseAfterTurn` | 完成后返回父状态 |

#### Phase 与发布者映射

**设计原则：谁产生时机，谁发布Phase**

```go
// ========== HSM发布的Phase（状态时机） ==========
// 这些Phase在State.Enter()方法中发布
PhaseBeforeTurn    → S_TURN_UPKEEP.Enter()    // 回合开始前
PhaseOnLand        → S_TURN_LANDED.Enter()    // 落地后
PhaseAfterTurn     → S_TURN_END.Enter()       // 回合结束后

// ========== Action发布的Phase（动作时机） ==========
// 这些Phase由ActionContext.ExecuteAction()发布
PhasePreDamage     → DamageAction.PreTriggerPhase()    // 伤害应用前
PhasePreEvent      → DrawEventAction.PreTriggerPhase() // 事件触发前
PhasePreMove       → MoveAction.PreTriggerPhase()      // 移动前
PhaseOnBuffApplied → AddBuffAction.PostTriggerPhase()  // Buff添加后
PhaseOnBuffRemoved → RemoveBuffAction.PreTriggerPhase() // Buff移除前

// ========== 特殊Phase ==========
PhaseAnyTime   → 全局可用，由客户端主动触发
PhaseItemUsed  → game.UseItem()发布，道具使用时
```

### ⏸️ 第三层：中断与决策层 (InterruptState)

处理用户决策等待，采用状态入栈/出栈机制。

```
┌─────────────────────────────────────────────────────────────────┐
│                      InterruptState                              │
│              (可叠加在任意第二层状态之上)                          │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  [任意State] ──► S_WAIT_DECISION ──► [恢复原State]               │
│       │                  │                                       │
│       │                  │                                       │
│       ▼                  ▼                                       │
│   Push(State)         等待用户选择                                │
│       │                  │                                       │
│       │                  ├─► OnUserChoice → Pop()                │
│       │                  │                                       │
│       │                  ├─► Timeout → Execute(Default) → Pop()  │
│       │                  │                                       │
│       └──────────────────┴─► 恢复原State继续执行                  │
└─────────────────────────────────────────────────────────────────┘
```

#### 决策类型

| DecisionType | 触发场景 | 超时 | 默认选项 |
|--------------|----------|------|----------|
| `DICE_UPGRADE` | 骰子升级卡使用确认 | 30s | 跳过 |
| `ANY_DOOR` | 任意门目标选择 | 30s | 取消 |
| `ITEM_USE` | 道具使用确认 | 20s | 跳过 |
| `FACTION_SKILL` | 阵营技能激活 | 15s | 跳过 |
| `EVENT_CHOICE` | 事件分支选择 | 25s | 随机 |

## 核心数据结构

### State 接口定义

```go
// State 状态接口
type State interface {
    ID() StateID
    Enter(ctx *StateContext)      // 进入状态
    Update(ctx *StateContext)      // 状态更新（每帧/每事件）
    Exit(ctx *StateContext)       // 退出状态
    CanTransitionTo(target StateID) bool // 转移检查
}

// StateID 状态标识
type StateID int

// StateContext 状态上下文
type StateContext struct {
    Game        *Game
    Player      *core.Player      // 当前玩家（回合层使用）
    Phase       event.Phase       // 当前Phase
    Data        interface{}       // 附加数据（如骰子点数、伤害值）
    Stack       *StateStack       // 状态栈（中断层使用）
    Decision    *event.Decision   // 待处理决策
    Timeout     time.Duration     // 超时时间
}
```

### HSM 主结构

```go
// HSM 分层状态机
type HSM struct {
    // 全局层
    GlobalState    State
    GlobalStateID  StateID

    // 回合层（子状态机）
    TurnState      State
    TurnStateID    StateID
    TurnPlayer     *core.Player

    // 中断栈
    InterruptStack *StateStack

    // 状态注册表
    States         map[StateID]State

    // 游戏引用
    Game           *Game
}

// StateStack 状态栈
type StateStack struct {
    Stack        []State
    ContextStack []*StateContext
}
```

## 状态流转详解

### 第一层流转

```
S_MATCH_INIT (对局初始化)
├── 行为
│   ├── MapEngine.GenerateMap(length)
│   ├── 分配玩家阵营 (青龙/朱雀/白虎/玄武)
│   ├── 朱雀玩家添加 Fire Buff (永久)
│   └── 初始化 EventBus
├── 转移
│   └── 自动 → S_ROUND_MINIGAME
│

S_ROUND_MINIGAME (小游戏)
├── 行为
│   ├── 下发 MiniGameStart 消息
│   ├── 等待所有玩家提交排名
│   └── 记录排名结果
├── 状态
│   ├── WaitingForMiniGameResults = true
│   ├── MiniGameResults map[playerID]int
├── 转移
│   ├── 收到全部结果 → S_ROUND_PREP
│   └── 超时(60s) → 自动排名 → S_ROUND_PREP
│

S_ROUND_PREP (轮次筹备)
├── 行为
│   ├── 根据排名分配骰子等级
│   │   ├── 第1名: 金骰子 (1-10)
│   │   ├── 第2名: 银骰子 (1-7)
│   │   ├── 第3名: 铜骰子 (1-5)
│   │   ├── 第4名: 木骰子 (1-3)
│   ├── 初始化回合队列 (排名逆序)
│   └── RoundCount++
├── 转移
│   └── 自动 → S_TURN_LOOP (进入第一个玩家回合)
│

S_TURN_LOOP (回合循环)
├── 行为
│   ├── 管理回合队列
│   ├── 启动当前玩家的子状态机
│   ├── 监控玩家到达终点
├── 子状态机
│   └── PlayerTurnState (第二层)
├── 转移
│   ├── 子状态机完成 → 下一玩家 (循环)
│   ├── 所有玩家完成 → S_ROUND_MINIGAME (下一轮)
│   ├── 任意玩家到达终点 → S_BOSS_BATTLE
│

S_BOSS_BATTLE (Boss战)
├── 行为
│   ├── 进入特殊战斗机制
│   ├── 触发终点玩家与Boss对战
│   └── 处理战斗结果
├── 转移
│   ├── Boss击败 → S_GAME_OVER (该玩家获胜)
│   ├── Boss未击败 → 返回 S_TURN_LOOP (继续回合)
│

S_GAME_OVER (对局结束)
├── 行为
│   ├── 广播获胜者
│   ├── 结算数据 (统计数据、成就等)
│   └── 清理资源
├── 状态
│   └── 终态，无转移
```

### 第二层流转

```
S_TURN_UPKEEP (回合准备)
├── Enter()
│   ├── 检查 IsDead → Respawn(checkpoint)
│   ├── 检查 SkipTurn → 清除标志，跳转 S_TURN_END
│   ├── 触发 PhaseBeforeTurn
│   │   ├── 神眷 Buff: LP+1
│   │   ├── 诅咒 Buff: LP-1
│   │   ├── 毒瘴 Buff: 标记触发恶性事件
│   │   ├── 离火 Buff: 每4回合 LP+1 (朱雀被动)
│   └── 处理决策队列 → Push S_WAIT_DECISION (如有)
├── Update()
│   ├── 检查决策完成 → 继续
│   └── 检查 SkipTurn 标志被设置 → S_TURN_END
├── 转移
│   ├── 正常 → S_MAIN_ACTION
│   ├── SkipTurn → S_TURN_END (跳过本回合)
│

S_MAIN_ACTION (主行动)
├── Enter()
│   ├── 构建可用道具列表
│   │   ├── PhaseAnyTime 道具 (反方向的钟等)
│   │   ├── PhaseBeforeTurn 道具 (骰子升级卡)
│   └── 构建阵营技能可用性
│       ├── 青龙行迹: ChargeCount >= 1 → 可激活
│       ├── 玄武镇厄: ChargeCount >= 1 → 可激活
├── 状态
│   ├── WaitingForAction = true
│   ├── AvailableActions []ActionOption
├── Update()
│   ├── 收到 UseItem(itemId) → 执行道具效果
│   ├── 收到 UseSkill() → 执行阵营技能
│   ├── 收到 RollDice() → 记录骰子点数，S_TURN_MOVING
├── 转移
│   ├── RollDice → S_TURN_MOVING
│   └── 超时(45s) → 自动 RollDice → S_TURN_MOVING
│

S_TURN_MOVING (移动结算)
├── Enter()
│   ├── 创建 MoveAction，通过 ActionContext.ExecuteAction()
│   │   ├── Action发布 PhasePreMove（PreTrigger阶段）
│   │   │   ├── 迷途 Buff订阅PhasePreMove，篡改Steps为负数
│   │   ├── 执行 Execute() → 调用 MapEngine.CalculatePath(position, diceSteps)
│   │   ├── 检查迷途 Buff → 修正移动方向（已通过Phase篡改）
├── 路径处理
│   ├── Fragile 处理
│   │   ├── 首次经过 → 标记已碎
│   │   ├── 落点在未碎 Fragile → FellDown=true, HP-1
│   │   └── 坠落 → 强制转移 S_TURN_END
│   ├── Fog 处理
│   │   ├── 首位经过 → 激活迷雾区域
│   │   │   ├── 区域内玩家获得 Poison Buff
│   │   │   └── 后续经过 → 投骰子获得 Exorcism Buff
│   ├── 反超处理
│   │   ├── 经过其他玩家 → 触发白虎被动 [劫运]
│   │   │   ├── 偷取对方随机 Buff
│   ├── 捷径处理
│   │   ├── 经过传送阵 → 向前传送
├── 转移
│   ├── 正常到达 → S_TURN_LANDED
│   ├── Fragile 坠落 → S_TURN_END (提前结束)
│   ├── 到达终点 → 父状态转移至 S_BOSS_BATTLE
│

S_TURN_LANDED (落地结算)
├── Enter()
│   ├── 触发 PhaseOnLand
│   │   ├── 任意门道具: 可选择传送到目标
│   │   ├── 检查点宝箱: 刷新道具
│   ├── 检查 CellType
│   │   ├── Checkpoint: 触发宝箱刷新
│   │   ├── Boss: 标记到达终点
├── 转移
│   └── 自动 → S_TURN_EVENT
│

S_TURN_EVENT (事件结算)
├── Enter()
│   ├── 触发 PhasePreEvent (事件免疫检查)
│   │   ├── 辟邪 Buff: 免疫 Poison 相关事件
│   │   ├── 玄武镇厄: 可取消一次恶性事件
│   ├── 抽取随机事件 (EventPool.Draw modified by LP)
│   │   ├── LP 影响权重
│   │   ├── Evaluation 分数决定良性/中性/恶性
│   ├── 执行事件效果
│   │   ├── HP/LP 变化
│   │   ├── Buff 添加/移除
│   │   ├── 道具获得/丢失
│   ├── 事件包含伤害 → 触发 PhasePreDamage
│   │   ├── 隐匿 Buff: 免疫伤害
│   ├── 处理决策 → Push S_WAIT_DECISION (如有选择分支)
├── 转移
│   └── 自动 → S_TURN_END
│

S_TURN_END (回合收尾)
├── Enter()
│   ├── 触发 PhaseAfterTurn
│   │   ├── 甘霖 Buff: 每2回合 HP+1
│   │   ├── 腐化 Buff: 每2回合 HP-1
│   ├── TickBuffs()
│   │   ├── Duration -= 1
│   │   ├── Duration == 0 → 移除 Buff
│   │   ├── 触发 PhaseOnBuffRemoved (移除时广播)
│   ├── 检查 IsDead
│   │   ├── 死亡 → Respawn(checkpoint)
│   ├── 阵营充能计数
│   │   ├── 青龙: IncrementChargeCount (每回合+1, max=1)
│   │   ├── 玄武: IncrementChargeCount (每回合+1, max=1)
│   ├── 朱雀 FireCounter 处理
│   │   ├── FireCounter += 1
│   │   ├── FireCounter == 4 → LP+1, FireCounter=0
├── 转移
│   └── 返回父状态 S_TURN_LOOP → NextTurn()
```

### 第三层流转

```
S_WAIT_DECISION (等待决策)
├── Enter()
│   ├── 原状态入栈 (Push)
│   ├── 设置 Timeout
│   ├── 设置 DefaultOption
│   ├── 下发 DecisionSync 消息给客户端
├── 状态
│   ├── DecisionID
│   ├── Prompt
│   ├── Options
│   ├── TimeoutRemaining
│   ├── TimeoutStart
├── Update()
│   ├── 收到 OnUserChoice(choice) → 执行 Decision.Action
│   ├── Timeout → 执行 Decision.Default
│   ├── 检查结果 → 出栈 (Pop)
├── Exit()
│   ├── 出栈恢复原状态
│   ├── 继续原状态的 Update 流程
├── 转移
│   └── Pop → 恢复原状态
```

## 特殊场景处理

### Fragile 坠落

```
S_TURN_MOVING
├── 计算路径 → PathResult.FellDown == true
├── 应用伤害 → player.ApplyDamage(1)
├── 强制转移 → S_TURN_END (跳过 Landed/Event)
└── 状态标记 → player.Metadata.Set("fell_down", true)
```

### 迷雾区域

```
S_TURN_MOVING
├── 经过 Fog Cell
├── 检查迷雾状态
│   ├── 首位激活
│   │   ├── MapEngine.ActivateFog(position)
│   │   ├── 区域内玩家 → AddBuff(Poison)
│   ├── 后续经过
│   │   ├── 投骰子 → 成功则 AddBuff(Exorcism)
├── 离开迷雾 → AddBuff(Exorcism, 5回合)
```

### Boss 战触发

```
S_TURN_MOVING
├── PathResult.ReachedEnd == true
├── 父状态转移 → S_BOSS_BATTLE
├── 保存触发玩家 → BossContext.TriggerPlayer
└── Boss 战逻辑 → 特殊状态机
```

### 死亡复活

```
任意状态
├── player.HP <= 0
├── player.IsDead = true
├── S_TURN_END.Enter()
│   ├── checkpoint = MapEngine.GetLastCheckpoint(position)
│   ├── player.Respawn(checkpoint)
│   ├── HP 恢复默认值
│   ├── Position 设置为检查点
│   ├── IsDead = false
```

## 断线重连设计

### 快照数据

```go
type HSMStateSnapshot struct {
    GlobalStateID    StateID       // 第一层状态
    TurnStateID      StateID       // 第二层状态（若在回合中）
    TurnPlayerID     string        // 当前回合玩家
    InterruptStack   []StateID     // 中断栈（第三层）
    CurrentDecision  *DecisionInfo // 当前决策（若有）
    GameState        *GameState    // 游戏数据
    PlayerSnapshots  []*PlayerSnapshot // 所有玩家状态
}

type DecisionInfo struct {
    DecisionID  string
    Prompt      string
    Options     []string
    Default     int
    Timeout     int // 剩余秒数
}
```

### 重连流程

```
客户端断线重连
├── 服务端加载 HSMStateSnapshot
├── 恢复 GlobalStateID
├── 若在回合中
│   ├── 恢复 TurnStateID
│   ├── 恢复 TurnPlayerID
├── 若有中断栈
│   ├── 恢复 InterruptStack
│   ├── 恢复当前 Decision
├── 下发 StateSync(Full) 消息
└── 客户端恢复 UI 状态
```

## 与现有系统集成

### EventBus 集成

```go
// HSM 在特定状态触发 Phase
func (s *StateTurnUpkeep) Enter(ctx *StateContext) {
    // PhaseBeforeTurn 触发
    decisions := ctx.Game.Bus.Publish(event.PhaseBeforeTurn, ctx.Player.UserID, ctx)
    if len(decisions) > 0 {
        // 入栈中断状态
        ctx.Stack.Push(s, ctx)
        ctx.HSM.TransitionTo(S_WAIT_DECISION, decisions)
    }
}
```

### Buff/Item 订阅

```go
// 状态变化时管理订阅
func (s *StateTurnEnd) Enter(ctx *StateContext) {
    // Tick Buff
    expired := ctx.Player.TickBuffs()
    for _, buff := range expired {
        ctx.Game.UnsubscribeBuff(buff) // 移除 EventBus 订阅
    }
}
```

### RNG 集成

```go
// 事件抽取使用 DrawEngine
func (s *StateTurnEvent) Enter(ctx *StateContext) {
    // 使用 LP 修正权重抽取事件
    eventDef := ctx.Game.DrawEngine.DrawEvent(ctx.Player.LP)
    // 执行事件效果
    s.executeEvent(eventDef, ctx)
}
```

## 实现优先级

### Phase 1: 核心框架

1. 定义 State 接口和 StateID 枚举
2. 实现 HSM 主结构和 StateStack
3. 实现第一层状态 (MatchInit, RoundMiniGame, RoundPrep, TurnLoop, GameOver)
4. 实现基础转移逻辑

### Phase 2: 回合层状态

1. 实现第二层状态 (TurnUpkeep, MainAction, TurnMoving, TurnLanded, TurnEvent, TurnEnd)
2. 集成 Phase 触发点
3. 实现路径计算和特殊地形处理

### Phase 3: 中断层

1. 实现 WaitDecision 状态
2. 实现状态入栈/出栈机制
3. 实现超时处理

### Phase 4: 特殊状态

1. 实现 BossBattle 状态
2. 实现断线重连快照
3. 完善状态同步消息

## 测试策略

### 单元测试

- 每个状态的 Enter/Update/Exit 测试
- 状态转移条件测试
- Phase 触发测试

### 集成测试

- 完整回合流程测试
- 多玩家回合循环测试
- 中断恢复测试

### 边缘场景测试

- Fragile 坠落打断流程
- 迷雾区域触发
- 死亡复活
- Boss 战触发
- 断线重连

## 文件结构规划

```
internal/engine/
├── hsm/
│   ├── state.go           # State 接口定义
│   ├── state_id.go        # StateID 枚举
│   ├── hsm.go             # HSM 主结构
│   ├── state_stack.go     # StateStack 实现
│   ├── context.go         # StateContext 定义
│   ├── global_states.go   # 第一层状态实现
│   ├── turn_states.go     # 第二层状态实现
│   ├── interrupt_states.go # 第三层状态实现
│   ├── transitions.go     # 转移规则表
│   ├── snapshot.go        # HSM 快照系统
│   └── hsm_test.go        # HSM 测试
```