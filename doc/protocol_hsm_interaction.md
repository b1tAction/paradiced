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
| OpRollDice | handleRollDice | hsm.OnRollDice | StateMainAction |
| OpUseItem | handleUseItem | hsm.OnUseItem | StateMainAction |
| OpMiniGameResult | handleMiniGameResult | hsm.OnMiniGameResult | StateRoundMiniGame |
| OpUserChoice | handleUserChoice | hsm.OnUserChoice | StateWaitDecision |

```go
// handleRollDice 示例
func (h *NakamaMatchHandler) handleRollDice(sender string) error {
    player := h.GetPlayer(sender)
    
    // 检查状态
    if h.hsm.GetCurrentStateID() != hsm.StateMainAction {
        return nil
    }
    
    // 掷骰子
    steps := h.diceMgr.RollSpecialDice(sender)
    
    // 创建完整的 ctx
    builder := net.NewBuilder(h.hsm)
    ctx := hsm.NewStateContext()
        .WithHSM(h.hsm)
        .WithPlayer(player)
        .WithBroadcast(NewNakamaBroadcastAdapter(h))
        .WithBuilder(builder)  // ⭐
    
    // 调用 HSM
    return h.hsm.OnRollDice(steps, ctx)
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
TurnEndState.Enter(ctx)
    │
    ├─→ PhaseAfterTurn 触发
    ├─→ Tick Buffs
    ├─→ Faction Charging
    │
    └─→ broadcastTurnSync(ctx)
            │
            └─→ ctx.Builder.BuildTurnSync()  ✅
                    │
                    └─→ ctx.Broadcast.BroadcastTurnSync()

    └─→ broadcastStateSync(ctx)
            │
            └─→ ctx.Builder.BuildStateSync()  ✅
                    │
                    └─→ ctx.Broadcast.BroadcastStateSync()
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
5. **TurnCheckpoint 不单独广播** - DrawItem LogEntry 记录到 GameLog，回合结束时统一广播 TurnSync

## 协议消息对应表

| 阶段 | State.Enter | Builder 方法 | Broadcast 方法 | OpCode |
|------|-------------|--------------|----------------|--------|
| MatchInit | MatchInitState.Enter | BuildStateSync | BroadcastStateSync | OpStateSync |
| MiniGame | RoundMiniGameState.Enter | BuildStateSync | BroadcastMiniGameStart | OpMiniGameStart |
| TurnUpkeep | TurnUpkeepState.Enter | BuildStateSync | BroadcastStateSync | OpStateSync |
| MainAction | MainActionState.Enter | BuildAvailable | SendAvailable | OpAvailable |
| TurnMoving | TurnMovingState.Enter (PhasePreMove) | - | - | - (纯移动，不单独广播) |
| TurnCheckpoint | TurnCheckpointState.Enter | - | - | - (DrawItem，不单独广播) |
| TurnLanded | TurnLandedState.Enter (PhaseOnLand) | - | - | - (落地，不单独广播) |
| TurnEnd | TurnEndState.Enter | BuildTurnSync + BuildStateSync | BroadcastTurnSync + BroadcastStateSync | OpTurnSync + OpStateSync |
| GameOver | GameOverState.Enter | BuildStateSync | BroadcastGameOver | OpGameOver |