# TurnFlow 回合流转系统实现文档

## 概述

TurnFlow 是《命运骰子》游戏的回合流转控制器，负责编排完整的玩家回合流程，包括步骤执行、用户决策处理、中断恢复和状态同步。

## 设计目标

1. **完整回合流程**：统一管理从回合开始到结束的所有步骤
2. **用户决策支持**：等待用户输入，支持超时自动执行
3. **中断恢复**：保存/恢复快照，支持断线重连
4. **状态同步**：完整同步和增量同步，优化网络传输
5. **特殊情况处理**：跳过回合、死亡复活、Boss 战触发

## 文件结构

```
internal/engine/
├── turn_flow.go      # TurnFlow 控制器和 TurnStep 定义
├── snapshot.go       # FlowSnapshot、PlayerSnapshot、SnapshotManager
├── state_sync.go     # StateSync 状态同步器
├── turn_flow_test.go # 单元测试
```

## TurnStep 步骤枚举

```go
type TurnStep int

const (
    StepInit TurnStep = iota      // 初始化
    StepUpcheck                   // 检查冻结/眩晕，决定跳过回合
    StepBeforeTurn                // 触发 BeforeTurn Phase
    StepMainAction                // 用户选择道具/技能
    StepOnMove                    // 骰子投掷，路径计算
    StepOnLand                    // 落地事件
    StepPreEvent                  // 事件免疫检查
    StepAfterTurn                 // Tick Buff，回合后效果
    StepComplete                  // 回合完成
)
```

| 步骤 | 说明 | 可能产生决策 |
|------|------|-------------|
| Init | 初始化状态 | ❌ |
| Upcheck | 检查 SkipTurn、IsDead | ❌ |
| BeforeTurn | 触发神眷/诅咒等 | ✓ |
| MainAction | 道具选择 | ✓ |
| OnMove | 骰子投掷、移动 | ❌ |
| OnLand | 落地事件、任意门 | ✓ |
| PreEvent | 辟邪免疫检查 | ❌ |
| AfterTurn | TickDuration、复活 | ❌ |
| Complete | 回合结束，NextTurn | ❌ |

## TurnFlow 结构

```go
type TurnFlow struct {
    Game          *Game                // 游戏实例
    StateMachine  *StateMachine        // 状态机
    MapEngine     *gamemap.MapEngine   // 地图引擎
    CurrentStep   TurnStep             // 当前步骤
    CurrentPlayer *core.Player         // 当前玩家
    Interrupted   bool                 // 是否被中断
    SavedSnapshot *FlowSnapshot        // 保存的快照
    Decisions     []*event.Decision    // 待处理决策
    DiceSteps     int                  // 骰子步数
}
```

## 回合执行流程

### ExecuteTurn 方法

```go
func (tf *TurnFlow) ExecuteTurn(player *core.Player) ([]*event.Decision, error) {
    // 1. 设置当前玩家和初始步骤
    tf.CurrentPlayer = player
    tf.CurrentStep = StepInit

    // 2. 循环执行步骤
    for tf.CurrentStep < StepComplete {
        result := tf.ExecuteStep(tf.CurrentStep, player)

        if result.Error != nil {
            return nil, result.Error
        }

        if len(result.Decisions) > 0 {
            // 需要用户输入，暂停并返回决策
            tf.Decisions = result.Decisions
            return result.Decisions, nil
        }

        tf.CurrentStep++
    }

    // 3. 回合完成，进入下一玩家
    tf.Game.NextTurn()
    return nil, nil
}
```

### 步骤执行详情

#### StepUpcheck - 上检查

```go
func (tf *TurnFlow) executeUpcheck(player *core.Player) *StepResult {
    // 1. 检查死亡状态
    if player.IsDead {
        checkpoint := tf.MapEngine.GetLastCheckpoint(player.Position)
        player.Respawn(checkpoint)
        result.PlayerUpdated = true
    }

    // 2. 检查 SkipTurn（冻结/眩晕）
    if player.SkipTurn {
        player.SkipTurn = false
        tf.CurrentStep = StepAfterTurn - 1 // 跳转到 AfterTurn
    }

    // 3. 检查是否可行动
    if !player.CanAct() {
        tf.CurrentStep = StepAfterTurn - 1
    }

    return result
}
```

#### StepBeforeTurn - 回合前

```go
func (tf *TurnFlow) executeBeforeTurn(player *core.Player) *StepResult {
    // 触发 BeforeTurn Phase
    pendingDecisions := tf.StateMachine.TriggerPhase(event.PhaseBeforeTurn, player)
    if len(pendingDecisions) > 0 {
        result.Decisions = pendingDecisions
        result.Success = false // 等待用户输入
    }
    return result
}
```

#### StepOnMove - 移动

```go
func (tf *TurnFlow) executeOnMove(player *core.Player) *StepResult {
    // 1. 触发 OnMove Phase（迷途反向）
    tf.StateMachine.TriggerPhase(event.PhaseOnMove, player)

    // 2. 检查迷途 Buff
    if player.HasBuff(core.BuffTypeLost) {
        // 反向移动
        targetPos := player.Position - tf.DiceSteps
        tf.DiceSteps = player.Position - targetPos
    }

    // 3. 计算路径
    pathResult, err := tf.MapEngine.CalculatePath(player.Position, tf.DiceSteps)
    player.Position = pathResult.TargetIndex

    // 4. 处理坠落
    if pathResult.FellDown {
        player.ApplyDamage(1)
    }

    return result
}
```

#### StepAfterTurn - 回合后

```go
func (tf *TurnFlow) executeAfterTurn(player *core.Player) *StepResult {
    // 1. 触发 AfterTurn Phase
    tf.StateMachine.TriggerPhase(event.PhaseAfterTurn, player)

    // 2. Tick Buff 持续时间
    expiredBuffs := player.TickBuffs()
    for _, buff := range expiredBuffs {
        tf.Game.UnsubscribeBuff(buff)
    }

    // 3. 检查死亡
    if player.IsDead {
        checkpoint := tf.MapEngine.GetLastCheckpoint(player.Position)
        player.Respawn(checkpoint)
    }

    return result
}
```

## FlowSnapshot 快照系统

### 快照结构

```go
type FlowSnapshot struct {
    GameID           string              // 游戏ID
    Round            int                 // 当前回合
    Turn             int                 // 当前轮次
    CurrentStep      TurnStep            // 当前步骤
    PlayerID         string              // 当前玩家ID
    WaitingDecisions []*DecisionSnapshot // 待处理决策
    PlayerSnapshots  []*PlayerSnapshot   // 玩家状态快照
    Timestamp        time.Time           // 时间戳
}

type PlayerSnapshot struct {
    UserID      string                   // 玩家ID
    Faction     core.Faction             // 阵营
    Position    int                      // 位置
    HP          int                      // 血量
    LP          int                      // 运势
    IsDead      bool                     // 是否死亡
    SkipTurn    bool                     // 是否跳过回合
    Inventory   []*ItemSnapshot          // 道具
    ActiveBuffs []*BuffSnapshot          // Buff
    Metadata    map[string]interface{}   // 元数据
}
```

### 快照管理

```go
// 创建快照
snapshot := tf.CreateSnapshot()

// 序列化
data, err := snapshot.ToJSON()

// 反序列化
restored := &FlowSnapshot{}
restored.FromJSON(data)

// SnapshotManager 管理
sm := NewSnapshotManager()
sm.Save(snapshot)            // 保存
loaded, _ := sm.Load(gameID) // 加载
sm.Delete(gameID)            // 删除
sm.HasSnapshot(gameID)       // 检查是否存在
```

### 中断恢复

```go
// 中断（保存快照）
func (tf *TurnFlow) Interrupt() error {
    if tf.IsWaiting() {
        tf.Interrupted = true
        tf.SavedSnapshot = tf.CreateSnapshot()
        return nil
    }
    return errors.New("cannot interrupt while not waiting")
}

// 恢复
func (tf *TurnFlow) ResumeFromInterrupt(snapshot *FlowSnapshot) error {
    // 1. 恢复游戏状态
    tf.Game.State.Round = snapshot.Round
    tf.Game.State.Turn = snapshot.Turn

    // 2. 恢复当前步骤和玩家
    tf.CurrentStep = snapshot.CurrentStep
    tf.CurrentPlayer = tf.Game.GetPlayer(snapshot.PlayerID)

    // 3. 恢复待处理决策
    tf.Decisions = restoreDecisions(snapshot.WaitingDecisions)

    tf.Interrupted = false
    return nil
}
```

## StateSync 状态同步

### 同步类型

```go
type SyncType int

const (
    SyncTypeFull     SyncType = iota // 完整状态同步
    SyncTypeDelta                    // 增量同步（变化部分）
    SyncTypeEvent                    // 事件通知
    SyncTypeDecision                 // 决策请求
)
```

### 同步消息结构

```go
type SyncMessage struct {
    Type      SyncType   // 同步类型
    Data      []byte     // JSON 编码数据
    Timestamp time.Time  // 时间戳
    GameID    string     // 游戏ID
    TargetID  string     // 目标玩家（空表示广播）
}
```

### 使用方法

```go
sync := engine.NewStateSync(game)

// 完整同步
msg, _ := sync.SyncFull()

// 增量同步（与上次同步对比）
msg, _ := sync.SyncDelta()

// 事件通知
msg, _ := sync.SyncEvent("buff_applied", playerID, map[string]interface{}{
    "buff_applied": "神眷",
    "lp_change":    1,
})

// 决策请求
msg, _ := sync.SyncDecision(decisionID, playerID, "是否使用？", []string{"使用", "跳过"}, 30)

// 广播所有玩家
msgs, _ := sync.SyncAllPlayers()

// 同步单个玩家
msg, _ := sync.SyncPlayer(playerID)
```

### 增量同步优化

增量同步通过 `SyncCheckpoint` 记录上次同步状态，仅发送变化部分：

```go
type SyncCheckpoint struct {
    Round           int                    // 回合
    Turn            int                    // 轮次
    PlayerPositions map[string]int         // 玩家位置
    PlayerHPs       map[string]int         // 玩家血量
    PlayerLPs       map[string]int         // 玩家运势
    PlayerBuffs     map[string][]string    // 玩家 Buff
    PlayerItems     map[string]int         // 玩家道具数量
}
```

如果没有任何变化，`SyncDelta()` 返回 `nil`。

## 用户决策处理

### OnUserChoice 方法

```go
func (tf *TurnFlow) OnUserChoice(choice int) error {
    if len(tf.Decisions) == 0 {
        return errors.New("no pending decisions")
    }

    // 执行用户选择
    current := tf.Decisions[0]
    ctx := event.NewContext(tf.CurrentPlayer)
    current.Execute(choice, ctx)

    // 移除已处理的决策
    tf.Decisions = tf.Decisions[1:]

    // 所有决策处理完毕，继续流程
    if len(tf.Decisions) == 0 {
        tf.CurrentStep++
    }

    return nil
}
```

### 按决策ID处理

```go
func (tf *TurnFlow) OnUserChoiceWithID(decisionID string, choice int) error {
    for i, decision := range tf.Decisions {
        if decision.ID == decisionID {
            ctx := event.NewContext(tf.CurrentPlayer)
            decision.Execute(choice, ctx)
            tf.Decisions = append(tf.Decisions[:i], tf.Decisions[i+1:]...)

            if len(tf.Decisions) == 0 {
                tf.CurrentStep++
            }
            return nil
        }
    }
    return errors.New("decision not found")
}
```

## 特殊情况处理

### 跳过回合

当玩家有 SkipTurn 标志（冻结/眩晕）时：

```go
if player.SkipTurn {
    player.SkipTurn = false  // 重置标志
    tf.CurrentStep = StepAfterTurn - 1  // 直接跳到 AfterTurn
}
```

### 死亡复活

```go
if player.IsDead {
    checkpoint := tf.MapEngine.GetLastCheckpoint(player.Position)
    player.Respawn(checkpoint)  // 重置 HP，设置位置
}
```

### 迷途反向移动

```go
if player.HasBuff(core.BuffTypeLost) {
    targetPos := player.Position - tf.DiceSteps
    if targetPos < 0 {
        targetPos = 0
    }
    tf.DiceSteps = player.Position - targetPos
}
```

## 测试覆盖

TurnFlow 系统包含完整的单元测试：

- `TestNewTurnFlow`: 创建验证
- `TestTurnStepString`: 步骤字符串转换
- `TestExecuteUpcheck`: 上检查（正常/跳过/死亡）
- `TestExecuteBeforeTurn`: 回合前触发
- `TestExecuteOnMove`: 移动计算（含迷途 Buff）
- `TestExecuteAfterTurn`: 回合后处理
- `TestOnUserChoice`: 用户决策处理
- `TestCreateSnapshot`: 快照创建
- `TestInterruptAndResume`: 中断恢复
- `TestFlowSnapshotToJSON/FromJSON`: 序列化
- `TestSnapshotManager`: 快照管理器
- `TestStateSync`: 状态同步

## 后续扩展

1. **骰子系统集成**：集成真实的骰子投掷机制
2. **Boss 战触发**：到达终点后触发 Boss 战斗
3. **道具使用流程**：完善 MainAction 步骤的道具选择
4. **事件执行**：落地事件的完整执行逻辑
5. **网络层集成**：与 Nakama 服务器集成
6. **客户端协议**：定义 SyncMessage 的传输协议