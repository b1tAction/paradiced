# HSM 分层状态机设计文档

## 概述

本文档描述《派乐代》游戏的分层状态机（HSM - Hierarchical State Machine）架构设计。该设计替代原有的 `TurnFlow` 流水线架构，解决双轨制控制问题，实现单一数据源（SSOT）。

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
| 全局状态 | global_states.go | ✅ | 5个Layer 1状态实现（BossBattle已移至Turn层） |
| 单元测试 | *_test.go | ✅ | 55+测试用例 |

### Phase 2: 回合层状态 ✅ (已完成)

| 组件 | 文件 | 状态 | 说明 |
|------|------|------|------|
| TurnUpkeepState | turn_states.go | ✅ | PhaseBeforeTurn触发、SkipTurn/IsDead检查 |
| MainActionState | turn_states.go | ✅ | 等待道具/骰子选择、超时处理 |
| TurnMovingState | turn_states.go | ✅ | HSM预扫描路径、迷途修改Steps、CheckPoint拆分、FellDown处理 |
| TurnCheckpointState | turn_states.go | ✅ | DrawItemAction（宝箱道具） |
| TurnLandedState | turn_states.go | ✅ | CellType行为矩阵、PhaseOnLand触发 |
| TurnDrawState | turn_states.go | ✅ | 概率抽取（DrawEvent/DrawItem） |
| TurnBossBattleState | turn_states.go | ✅ | Boss战斗（玩家攻击/Boss反击）、Boss击败检测 |
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
├── adapter.go         # EventBusAdapter、MapEngineAdapter、GameWrapper（含GetGameLog()）
├── hsm.go             # HSM主结构、OnRollDice/OnUseItem处理、HSMSnapshot、状态转换日志记录
├── state_stack.go     # StateStack入栈出栈
├── global_states.go   # Layer 1全局状态实现 (已完成)
├── turn_states.go     # Layer 2回合状态实现 (已完成，含GameLog集成)
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
│                                                    ▼            │
│                       State_Game_Over         [下一玩家]        │
└─────────────────────────────────────────────────────────────────┘
```

#### 状态定义 (已实现)

| StateID | 结构体名称 | 行为 | Phase触发 | 转移条件 |
|---------|------------|------|-----------|----------|
| `StateMatchInit` (100) | `MatchInitState` | 初始化标记、设置metadata | - | Update自动返回StateWaitingForHost |
| `StateWaitingForHost` (101) | `WaitingForHostState` | 等待房主手动启动游戏 | - | 收到StartGame后返回StateRoundMiniGame |
| `StateRoundMiniGame` (102) | `RoundMiniGameState` | 等待小游戏排名 | - | 收到所有排名后返回StateRoundPrep |
| `StateRoundPrep` (103) | `RoundPrepState` | 根据排名分配骰子、增加Round计数 | - | Update自动返回StateTurnLoop |
| `StateTurnLoop` (104) | `TurnLoopState` | 管理回合队列、检查Boss击败 | - | StartPlayerTurn→StateTurnUpkeep；BossDefeated→StateGameOver |
| `StateRoundEndWait` (105) | `RoundEndWaitState` | 等待所有客户端信号RoundReady | - | 收到所有后返回StateRoundMiniGame |
| `StateGameOver` (106) | `GameOverState` | 终态，记录winner | - | 无转移（终态） |

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
│                                       S_TURN_DRAW               │
│                                            │                     │
│                                            ▼                     │
│                                  S_TURN_BOSS_BATTLE (Boss格)     │
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
| `S_TURN_UPKEEP` | 回合准备 | 检查SkipTurn、IsDead（Boss跳过BeforeTurn） | `PhaseBeforeTurn` | 可行动→MainAction；Boss→TurnBossBattle；不可→TurnEnd |
| `S_MAIN_ACTION` | 主行动 | 等待道具/骰子选择（Boss格玩家→TurnBossBattle） | `PhaseItemUsed`道具可用 | 收到RollDice→TurnMoving/TurnBossBattle |
| `S_TURN_MOVING` | 移动结算 | HSM预扫描路径、迷途处理、CheckPoint拆分 | HSM发布`PhasePreMove` | 正常→TurnLanded；坠落→TurnEnd；CheckPoint→TurnCheckpoint |
| `S_TURN_CHECKPOINT` | CheckPoint结算 | DrawItemAction（宝箱道具） | - | →TurnMoving(remaining steps) |
| `S_TURN_LANDED` | 落地结算 | CellType行为矩阵触发 | `PhaseOnLand` | DrawType→TurnDraw；Boss→TurnBossBattle；Normal→TurnEnd |
| `S_TURN_DRAW` | 概率抽取 | DrawEventAction/DrawItemAction | `PhasePreEvent`(DrawEvent) | →TurnEnd |
| `S_TURN_BOSS_BATTLE` | Boss战斗 | 玩家攻击Boss/Boss反击 | - | →TurnEnd（Boss击败→GameOver） |
| `S_TURN_END` | 回合收尾 | TickBuff、死亡检查（Boss跳过AfterTurn） | `PhaseAfterTurn` | 完成后返回父状态 |

#### Phase 与发布者映射

**设计原则：谁产生时机，谁发布Phase**

```go
// ========== HSM发布的Phase（状态时机） ==========
// 这些Phase在State.Enter()方法中发布
PhaseBeforeTurn    → S_TURN_UPKEEP.Enter()    // 回合开始前
PhaseOnLand        → S_TURN_LANDED.Enter()    // 落地后
PhaseAfterTurn     → S_TURN_END.Enter()       // 回合结束后
PhasePreMove       → S_TURN_MOVING.Enter()    // 移动前（迷途handler修改Steps）

// ========== Action发布的Phase（动作时机） ==========
// 这些Phase由ActionContext.ExecuteAction()发布
PhasePreDamage       → DamageAction/BossDamageAction.PreTriggerPhase()    // 伤害应用前
PhasePreEvent        → DrawEventAction.PreTriggerPhase()                  // 事件触发前
PhasePreRespawn      → RespawnAction.PreTriggerPhase()                    // 重生前
PhasePreBuffApplied  → AddBuffAction.PreTriggerPhase()                   // Buff添加前
PhasePostBuffApplied → AddBuffAction.PostTriggerPhase()                  // Buff添加后（入场效果）
PhasePreBuffRemoved  → RemoveBuffAction.PreTriggerPhase()               // Buff移除前（亡语）
PhasePostBuffRemoved → RemoveBuffAction.PostTriggerPhase()              // Buff移除后
PhasePreAction       → ActionContext.ExecuteAction()                     // 任何Action前（死亡标记）
PhasePreDiceRoll     → RollDiceAction.PreTriggerPhase()                 // 骰子结果前（可拦截）

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
    *util.Metadata                    // 元数据存储（嵌入）
    HSM           *HSM                // HSM引用
    Player        *core.Player        // 当前玩家（回合层使用）
    Builder       pkgnet.Builder      // 协议构建器
    Broadcast     pkgnet.BroadcastAdapter // 广播适配器
    Decisions     []*event.Decision   // 待处理决策列表
    StartTime     time.Time           // 状态进入时间
    RoundData     *util.Metadata      // 回合周期性数据（charge_counter/fire_counter等）
}
```

### HSM 主结构

```go
// HSM 分层状态机
type HSM struct {
    // 全局层
    globalState    State
    globalStateID  StateID

    // 回合层（子状态机）
    turnState      State
    turnStateID    StateID

    // 中断栈
    interruptStack *StateStack

    // 状态注册表
    states         map[StateID]State
    factory        StateFactory

    // 游戏引用（通过getter访问）
    game           *engine.Game
    bus            *event.EventBus

    // Builder和Broadcast
    builder        pkgnet.Builder
    broadcast      pkgnet.BroadcastAdapter

    // 运行状态
    running        bool
    paused         bool
    round          int
    turn           int
}

// StateStack 状态栈
type StateStack struct {
    stack        []State
    contextStack []*StateContext
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
│   ├── Boss排在Players末尾（最后行动）
│   ├── 检查Boss是否被击败
├── 子状态机
│   └── PlayerTurnState (第二层)
├── 转移
│   ├── 子状态机完成 → 下一玩家 (循环)
│   ├── 所有玩家完成 → S_ROUND_MINIGAME (下一轮)
│   ├── Boss被击败 → S_GAME_OVER
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
│   ├── 开始回合日志分段
│   │   └── ctx.Game.Log.StartTurn(round, turn, player.UserID)
│   ├── 检查 IsDead → RespawnAction（通过Action系统）
│   │   └── actionCtx.ExecuteAction(NewRespawnAction(player, checkpoint, "DeathRespawn"))
│   ├── 检查 SkipTurn → 清除标志，跳转 S_TURN_END
│   ├── 触发 PhaseBeforeTurn
│   │   ├── 神眷 Buff: LP+1（通过ModifyLPAction）
│   │   ├── 诅咒 Buff: LP-1（通过ModifyLPAction）
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
│   ├── Steps = ctx.GetDiceSteps() (from dice or remaining steps from CheckPoint)
│   ├── HSM 发布 PhasePreMove（迷途 handler 修改 Steps 为负数）
│   │   ├── 迷途 Buff: Steps > 0 → Steps = -Steps（防 double-flip）
│   ├── 路径预扫描 MapEngine.CalculatePath(position, Steps)
│   │   ├── 迷途反向移动: Steps < 0, direction=-1
│   ├── FellDown 处理（HSM 层）
│   │   ├── MoveAction 到坠落位置
│   │   ├── FellDownAction → 衍生 PiercingDamageAction（坠落伤害）
│   │   └── 强制转移 S_TURN_END
│   ├── CheckPoint 拆分（仅 Steps > 0 正向移动）
│   │   ├── 经过 CheckPoint → 拆分为两个移动段
│   │   ├── MoveAction(seg1) → CheckPoint位置
│   │   ├── ctx.SetInt(KeyDiceSteps, remainingSteps)
│   │   ├── hasCheckpoint flag → S_TURN_CHECKPOINT
│   ├── 正常移动 → MoveAction → S_TURN_LANDED
│   │   ├── 自动记录到 GameLog
│   ├── Fog 处理
│   │   ├── 首位经过 → 激活迷雾区域
│   ├── 反向移动不触发 CheckPoint 奖励
├── 转移
│   ├── 正常到达 → S_TURN_LANDED
│   ├── CheckPoint 拆分 → S_TURN_CHECKPOINT
│   ├── Fragile 坠落 → S_TURN_END (提前结束)
│   ├── Boss格 → S_TURN_BOSS_BATTLE (玩家攻击Boss)


S_TURN_CHECKPOINT (CheckPoint结算)
├── Enter()
│   ├── Execute DrawItemAction (CheckpointTreasure)
│   │   ├── 从 ItemPool 随机抽取道具
│   │   ├── 加入玩家 Inventory
│   │   ├── 自动记录到 GameLog
│   ├── 不中途广播 TurnSync
├── Update()
│   └── → S_TURN_MOVING (remaining steps as dice_steps)


S_TURN_LANDED (落地结算)
├── Enter()
│   ├── 触发 PhaseOnLand
│   │   ├── 任意门道具: 可选择传送到目标
│   ├── CellType 行为矩阵
│   │   ├── CellTypeEvent: 触发绑定固定事件 (cell.EventID)
│   │   │   ├── DrawEventAction (bound event)
│   │   │   ├── runEventEffect (执行事件效果)
│   │   │   ├── skipEvent=true → S_TURN_END
│   │   ├── CellTypeCheckpoint: 已在TurnCheckpoint处理 → skipEvent=true → S_TURN_END
│   │   ├── CellTypeBoss: 已在TurnMoving处理 → skipEvent=true → S_TURN_END
│   │   ├── CellTypeNormal/Fog/Fragile: → S_TURN_EVENT (随机DrawEvent)
├── 转移
│   ├── skipEvent=true → S_TURN_END
│   ├── skipEvent=false → S_TURN_EVENT


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
│   │   ├── 甘霖 Buff: 每2回合 HP+1（通过HealAction）
│   │   ├── 腐化 Buff: 每2回合 HP-1（通过DamageAction）
│   ├── TickBuffs()
│   │   ├── Duration -= 1
│   │   ├── Duration == 0 → 移除 Buff（RemoveBuffAction）
│   │   ├── 触发 PhaseOnBuffRemoved (移除时广播)
│   ├── 检查 IsDead
│   │   ├── 死亡 → RespawnAction（自动记录到 GameLog）
│   ├── 阵营充能计数
│   │   ├── 青龙: IncrementChargeCount (每回合+1, max=1)
│   │   ├── 玄武: IncrementChargeCount (每回合+1, max=1)
│   ├── 朱雀 FireCounter 处理
│   │   ├── FireCounter += 1
│   │   ├── FireCounter == 4 → LP+1, FireCounter=0
│   ├── 结束回合日志分段
│   │   └── ctx.Game.Log.EndTurn()
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
├── FellDownAction（通过Action系统执行）
│   ├── actionCtx.ExecuteAction(NewFellDownAction(player, pos, 1, "FragileCell"))
│   ├── FellDownAction → 衍生 PiercingDamageAction → 扣HP
│   ├── GameLog 记录两条 Entry
│   │   ├── ActionType: "fell_down", Metadata: {"position": pos}
│   │   ├── ActionType: "damage", Metadata: {"hp_change": -1}
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

### Boss 战（TurnBossBattleState）

```
S_TURN_BOSS_BATTLE (Boss战斗)
├── 玩家攻击Boss分支
│   ├── 骰子点数 = 基础伤害值
│   ├── 暴击概率与骰子品质相关（Gold:30%, Silver:20%, Copper:10%, Wood:5%）
│   ├── 暴击伤害 = 基础伤害 × 2
│   ├── BossDamageAction → 对Boss HP造成伤害
│   ├── Boss被击败 → KeyBossDefeated=true → GameOver
├── Boss反击分支（Boss回合）
│   ├── Boss格玩家 → Boss攻击（normal/crit/skill）
│   ├── Boss格无玩家 → Boss回合空转
│   ├── BossAttackAction → 衍生 DamageAction（Boss对玩家伤害）
│   ├── BossSkillAction → 从技能池随机抽取
│   ├── 玩家死亡 → RespawnAction → 复活至检查点
├── 转移
│   └── → S_TURN_END
```

### 死亡复活

```
任意状态
├── player.HP <= 0
├── player.IsDead = true
├── S_TURN_END.Enter() 或 S_TURN_UPKEEP.Enter()
│   ├── checkpoint = MapEngine.GetLastCheckpoint(position)
│   ├── RespawnAction（通过Action系统）
│   │   ├── actionCtx.ExecuteAction(NewRespawnAction(player, checkpoint, "DeathRespawn"))
│   │   ├── 自动记录到 GameLog
│   │   │   ├── ActionType: "respawn"
│   │   │   ├── Target: player.UserID
│   │   │   ├── Metadata: {"checkpoint_pos": checkpoint}
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
    GameLogSnapshot  []byte        // GameLog JSON 快照（用于回放恢复）
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
├── 恢复 GameLog（从 JSON 快照）
│   ├── 解析 GameLogSnapshot
│   ├── 恢复 segments 和 current
├── 下发 StateSync(Full) 消息
│   ├── 包含完整 GameLog JSON
│   ├── Client 可播放历史动画
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

### GameLog 集成

```go
// HSM 状态转换日志记录
func (hsm *HSM) TransitionTo(targetID StateID, ctx *StateContext) error {
    fromID := hsm.GetCurrentStateID()
    
    // 记录状态转换到全局日志
    if hsm.game.Log != nil {
        hsm.game.Log.LogStateTransition(fromID.String(), targetID.String(), getPlayerID(hsm.turnPlayer))
    }
    
    // ... 状态转换逻辑 ...
}

// 回合开始时启动日志分段
func (s *StateTurnUpkeep) Enter(ctx *StateContext) {
    ctx.Game.Log.StartTurn(ctx.Game.State.Round, ctx.Game.State.Turn, ctx.Player.UserID)
    // ...
}

// 回合结束时关闭日志分段
func (s *StateTurnEnd) Enter(ctx *StateContext) {
    // ...
    ctx.Game.Log.EndTurn()
}
```

### Action 系统集成

```go
// 所有游戏效果通过 Action 系统执行
func (s *StateTurnUpkeep) Enter(ctx *StateContext) {
    // 使用 RespawnAction
    if ctx.Player.IsDead {
        checkpoint := ctx.MapEngine.GetLastCheckpoint(ctx.Player.Position)
        respawnAction := engineaction.NewRespawnAction(ctx.Player, checkpoint, "DeathRespawn")
        s.actionCtx.ExecuteAction(respawnAction)
    }
}

func (s *StateTurnMoving) Enter(ctx *StateContext) {
    // 使用 FellDownAction
    if s.fellDown {
        fellDownAction := engineaction.NewFellDownAction(ctx.Player, ctx.Player.Position, 1, "FragileCell")
        s.actionCtx.ExecuteAction(fellDownAction)
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

### Phase 4: 特殊状态 ✅ (已完成)

1. ✅ BossBattle 已移至 Turn 层（TurnBossBattleState）
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
- Boss 战（TurnBossBattleState）
- 断线重连

---

## GameLog 系统说明

### 设计原则

所有游戏效果通过 Action 系统执行，自动记录到全局 GameLog：

1. **单一日志源** - Game 持有唯一的 GameLog 实例
2. **分段存储** - 按 Round/Turn 分段，便于 Client 按回合回放
3. **Action 集成** - 所有游戏效果通过 Action 系统，自动生成日志
4. **类型安全元数据** - LogEntry.Metadata 使用 util.Metadata

### HSM 状态与 GameLog 交互

| HSM 状态 | GameLog 操作 | 说明 |
|---------|-------------|------|
| TurnUpkeep.Enter() | StartTurn() | 开始新回合日志分段 |
| TurnEnd.Enter() | EndTurn() | 结束回合日志分段 |
| HSM.TransitionTo() | LogStateTransition() | 记录状态转换 |

### Action 类型与日志记录

| Action 类型 | ActionType | 日志字段 |
|------------|-----------|---------|
| DamageAction | "damage" | Delta=-Amount, Metadata(blocked_by, piercing) |
| HealAction | "heal" | Delta=Amount |
| ModifyLPAction | "modify_lp" | Delta=Amount |
| MoveAction | "move" | Delta=Steps, Metadata(start_pos, end_pos, path) |
| AddBuffAction | "add_buff" | Metadata(buff_type, duration) |
| RemoveBuffAction | "remove_buff" | Metadata(buff_type) |
| RespawnAction | "respawn" | Metadata(checkpoint_pos) |
| FellDownAction | "fell_down" | Metadata(position) |
| TeleportAction | "teleport" | Metadata(target_pos) |
| StealBuffAction | "steal_buff" | Metadata(stolen_buff_type) |
| BossDamageAction | "boss_damage" | Metadata(damage, is_crit, boss_remaining_hp) |
| BossAttackAction | "boss_attack" | Metadata(attack_type, target) |
| BossSkillAction | "boss_skill" | Metadata(skill_type, targets) |

### Client 回放支持

Client 可通过 GameLog JSON 播放完整游戏动画：

```json
{
  "segments": [
    {
      "round": 1,
      "turn": 0,
      "player_id": "player1",
      "entries": [
        {"type": "state", "metadata": {"from": "MatchInit", "to": "TurnLoop"}},
        {"type": "action", "action_type": "modify_lp", "delta": 1, "source": "Buff_Divine"},
        {"type": "action", "action_type": "respawn", "target": "player1", "metadata": {"checkpoint_pos": 50}},
        {"type": "action", "action_type": "move", "delta": 5, "metadata": {"path": [0,1,2,3,4,5]}},
        {"type": "action", "action_type": "fell_down", "delta": -1, "metadata": {"position": 5}},
        {"type": "state", "metadata": {"from": "TurnMoving", "to": "TurnEnd"}}
      ]
    }
  ]
}
```

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