# 协议层与 HSM 层交互流程

## 完整交互链路

```
┌─────────────┐      ┌──────────────┐      ┌─────────┐      ┌──────────┐      ┌────────┐
│   Client    │ ──→  │ Nakama Msg   │ ──→  │   HSM   │ ──→  │  State   │ ──→  │ Builder│ ──→ Broadcast ──→ Client
│  (发送消息)  │      │   Handler    │      │ Method  │      │  Enter   │      │ Build  │
└─────────────┘      └──────────────┘      └─────────┘      └──────────┘      └────────┘
```

## 详细流程

### 1. 初始化流程 (MatchInit)

```
NakamaMatchHandler.MatchInit()
    │
    ├─→ NewBuilder(hsm)                // 创建 Builder
    ├─→ NewStateContext()
    │       .WithHSM(hsm)
    │       .WithBroadcast(adapter)
    │       .WithBuilder(builder)      // ⭐ Builder 设置
    │
    └─→ hsm.Start(StateMatchInit, ctx)
            │
            └─→ TransitionTo(StateMatchInit, ctx)
                    │
                    ├─→ MatchInitState.Enter(ctx)
                    │       │
                    │       ├─→ 地图生成
                    │       ├─→ InitializePlayerFactionBuffs()
                    │       │
                    │       └─→ ctx.Builder.BuildStateSync()  ✅
                    │               │
                    │               └─→ ctx.Broadcast.BroadcastStateSync()
                    │
                    └─→ MatchInitState.Update() → StateRoundMiniGame
                            │
                            └─→ TransitionTo(StateRoundMiniGame, ctx)  // 同一个 ctx
                                    │
                                    └─→ RoundMiniGameState.Enter(ctx)
                                            │
                                            └─→ ctx.Builder.BuildStateSync()  ✅
                                                    │
                                                    └─→ ctx.Broadcast.BroadcastMiniGameStart()
```

### 2. 客户端消息处理

| OpCode | Handler | HSM Method | 状态 |
|--------|---------|------------|------|
| OpRollDice (100) | handleRollDice | hsm.OnRollDice(ctx) | StateMainAction |
| OpUseItem (101) | handleUseItem | hsm.OnUseItem(ctx) | StateMainAction |
| OpUseSkill (102) | handleUseSkill | hsm.OnUseSkill(ctx) | StateMainAction |
| OpUserChoice (103) | handleUserChoice | hsm.OnUserChoice | StateWaitDecision |
| OpStartGame (105) | handleStartGame | hsm.Start | StateWaitingForHost |
| OpRoundReady (106) | handleRoundReady | hsm.OnRoundReady | StateRoundEndWait |
| OpMiniGameDataSubmit (107) | handleMiniGameDataSubmit | hsm.OnMiniGameDataSubmit | StateRoundMiniGame |

```go
// handleRollDice 示例 - Steps 由 RollDiceAction 内部计算
func (h *NakamaMatchHandler) handleRollDice(sender string) error {
    player := h.GetPlayer(sender)

    // 检查状态
    if h.hsm.GetTurnStateID() != hsm.StateMainAction {
        return nil
    }

    // 创建 StateContext（Steps 由 HSM 内部 RollDiceAction 计算）
    ctx := h.hsm.NewStateContext()
        .WithPlayer(player)

    // 调用 HSM（无 steps 参数）
    return h.hsm.OnRollDice(ctx)
}
```

### 3. 状态轮转 (MatchLoop)

```
MatchLoop() 每个 tick
    │
    ├─→ NewBuilder(h.hsm)              // 每次创建 Builder
    ├─→ NewStateContext()
    │       .WithHSM(h.hsm)
    │       .WithPlayer(currentPlayer)
    │       .WithBroadcast(adapter)
    │       .WithBuilder(builder)      // ⭐
    │
    ├─→ 处理 TurnEnd → OnTurnComplete + StartPlayerTurn
    │       │
    │       └─→ TransitionTo(StateTurnUpkeep, ctx)
    │               │
    │               └─→ TurnUpkeepState.Enter(ctx)
    │                       │
    │                       └─→ ctx.Builder.BuildStateSync()  ✅
    │                               └─→ ctx.Broadcast.BroadcastStateSync()
    │
    └─→ hsm.Update(ctx)
            │
            └─→ 状态 Update 返回 nextState
                    │
                    └─→ TransitionTo(nextState, ctx)
```

### 4. Turn State 广播

```
TurnLandedState.Enter(ctx)
    │
    ├─→ PhaseOnLand 触发
    │
    └─→ broadcastStateSync(ctx)  // 落地效果后广播 StateSync（含增量 Entries）
            │
            └─→ ctx.Builder.BuildStateSync()  ✅
                    │
                    └─→ ctx.Broadcast.BroadcastStateSync()

TurnDrawState.Enter(ctx)
    │
    ├─→ DrawEvent/DrawItem 执行
    │
    └─→ broadcastStateSync(ctx)  // 抽取效果后广播 StateSync（含增量 Entries）
            │
            └─→ ctx.Builder.BuildStateSync()  ✅
                    │
                    └─→ ctx.Broadcast.BroadcastStateSync()

TurnEndState.Enter(ctx)
    │
    ├─→ PhaseAfterTurn 触发
    ├─→ Tick Buffs
    ├─→ Faction Charging
    │
    └─→ broadcastStateSync(ctx)  // 回合结束时广播 StateSync（含剩余增量 Entries）
            │                       // 必须在 EndTurn 之前，因为 EndTurn 会清除 GameLog.current
            └─→ ctx.Builder.BuildStateSync()  ✅
                    │
                    └─→ ctx.Broadcast.BroadcastStateSync()

    └─→ game.Log.EndTurn()  // 结束回合日志
```

## 关键点

### ✅ 已实现

1. **所有 Handler 方法创建完整 ctx** (含 Builder)
2. **MatchInit 创建完整 ctx**
3. **MatchLoop 每次创建完整 ctx**
4. **State.Enter() 使用 ctx.Builder**
5. **Auto-proceed 使用同一个 ctx** (Builder 保留)

### ⚠️ 注意事项

1. **ctx 必须外部传入** - HSM 内部不创建 Builder
2. **每次 MatchLoop 创建新 Builder** - 确保 HSM 状态是最新的
3. **TransitionTo 不创建 ctx** - 依赖外部传入的完整 ctx
4. **PhasePreMove 由 HSM 发布** - TurnMovingState.Enter() 中发布，迷途 handler 通过 StepsModifier 接口修改 Steps（非 MoveAction.PreTrigger）
5. **TurnLanded/TurnDraw 广播 StateSync** - 落地和抽取效果执行后广播 StateSync（含增量 Entries），客户端按 Entries 顺序播放动画
6. **TurnEnd 广播必须在 EndTurn 之前** - `game.Log.EndTurn()` 会将 `GameLog.current` 设为 nil，导致 `GetNewEntries()` 返回空

## 协议消息对应表

| 阶段 | State.Enter | Builder 方法 | Broadcast 方法 | OpCode |
|------|-------------|--------------|----------------|--------|
| MatchInit | MatchInitState.Enter | BuildStateSync | BroadcastStateSync | OpStateSync (1) |
| WaitingForHost | WaitingForHostState.Enter | BuildStateSync | BroadcastWaitingSync | OpWaitingSync (10) |
| MiniGame | RoundMiniGameState.Enter | BuildStateSync | BroadcastMiniGameStart | OpMiniGameStart (5) |
| RoundPrep | RoundPrepState.Enter | BuildStateSync | BroadcastStateSync | OpStateSync (1) |
| TurnUpkeep | TurnUpkeepState.Enter | BuildStateSync | BroadcastStateSync | OpStateSync (1) |
| MainAction | MainActionState.Enter | BuildAvailable | SendAvailable | OpAvailable (4) |
| TurnMoving | TurnMovingState.Enter (PhasePreMove) | - | - | - (纯移动，不单独广播) |
| TurnLanded | TurnLandedState.Enter (PhaseOnLand) | BuildStateSync | BroadcastStateSync | OpStateSync (1) |
| TurnDraw | TurnDrawState.Enter (DrawEvent/DrawItem) | BuildStateSync | BroadcastStateSync | OpStateSync (1) |
| TurnBossBattle | TurnBossBattleState.Enter | BuildStateSync | BroadcastStateSync | OpStateSync (1) |
| TurnEnd | TurnEndState.Enter | BuildStateSync | BroadcastStateSync | OpStateSync (1) |
| GameOver | GameOverState.Enter | BuildStateSync | BroadcastGameOver | OpGameOver (7) |
| ActionRejected | - | - | SendActionRejected | OpActionRejected (9) |
| StartGameAck | - | BuildFullSyncStateSync + BuildMapInfo | BroadcastStartGameAck | OpStartGameAck (11) |