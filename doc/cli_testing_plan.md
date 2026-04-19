# CLI 自动化测试计划

## 1. 概述

### 1.1 测试目标

CLI 自动化测试旨在验证 Paradiced 后端对局系统的**可玩性**、**稳定性**和**错误处理正确性**。通过自动化 bot 玩家模拟真实游戏流程，实现：

1. **功能验证**：验证 HSM 状态机、Action 系统、协议层正常工作
2. **错误处理验证**：验证错误从 Core → Action → HSM → Nakama 的正确传播和 ErrorCode 映射
3. **稳定性验证**：通过 soak 测试发现内存泄漏、goroutine 泄漏、偶发 panic
4. **回归测试**：每次代码变更后自动验证核心功能未被破坏

### 1.2 测试范围

| 测试类型 | 覆盖内容 | 优先级 |
|---------|---------|-------|
| 基础对局流程 | 创建/加入/开始/结束完整一局 | P0 |
| 回合主流程 | Upkeep → MainAction → Moving → Landed → Event → End | P0 |
| 决策中断 | DecisionRequest → UserChoice → 恢复流程 | P0 |
| 小游戏流程 | MiniGameStart → 排名提交 → MiniGameResult | P1 |
| 错误处理 | ActionRejected + ErrorCode 正确返回 | P0 |
| 断线重连 | FullSync 状态对齐 | P2 |
| 道具使用 | UseItem → 效果执行 | P2 |
| 阵营技能 | UseSkill → 充能/触发 | P2 |

### 1.3 测试环境要求

```bash
# 启动 Nakama + CockroachDB
make docker-up

# 构建插件并重启
make rebuild

# 确认服务健康
docker compose ps
```

---

## 2. CLI 测试架构

### 2.1 组件结构

```
cmd/pdcli (CLI 入口)
    │
    ├── internal/cli/command (Cobra 命令)
    │   ├── playtest run   - 单次对局测试
    │   └── playtest soak  - 多轮压力测试
    │
    ├── internal/cli/nakama (Nakama 客户端封装)
    │   ├── Client         - Nakama HTTP/WS 客户端
    │   ├── StandaloneClient - 独立 WS 客户端
    │   └── Logger         - 结构化日志
    │
    ├── internal/cli/scenario (测试场景)
    │   ├── AutoPlayPlayer - 自动化 bot 玩家
    │   └── RunAutoPlay    - 场景执行器
    │
    └── internal/cli/model (协议模型)
        ├── StateSync      - 全量状态同步
        ├── TurnSync       - 回合日志同步
        ├── Decision       - 决策请求
        ├── Available      - 可用行动
        ├── MiniGameStart  - 小游戏开始
        ├── MiniGameResult - 小游戏结果
        ├── GameOver       - 游戏结束
        └── ActionRejected - 行动拒绝
```

### 2.2 AutoPlayPlayer 策略

当前 AutoPlayPlayer 采用简化策略：

```go
// 收到 Available → 自动掷骰子
case *protocol.StateAvailable:
    client.SendRollDice()

// 收到 DecisionRequest → 自动选择第一个选项
case *protocol.DecisionRequest:
    client.SendUserChoice(0)

// 收到 MiniGameStart → 随机排名
case *protocol.MiniGameStart:
    rank := rand.Intn(len(msg.Players)) + 1
    client.SendMiniGameResult(rank)
```

---

## 3. 测试场景设计

### 3.1 P0: 基础对局流程测试 (TestBasicGameFlow)

**测试目标**：验证 2-4 人可完整完成一局游戏

**测试步骤**：
1. 启动 2/3/4 个 bot 玩家
2. 通过 Nakama Matchmaker 匹配并创建房间
3. 等待 MatchInit 完成
4. 运行至 GameOver 或达到最大回合数 (50)
5. 验证至少完成 5 个回合

**验证点**：
- [ ] MatchInit 成功，无 ctx.Error
- [ ] 每个回合正常推进 (TurnSync 条目数 > 0)
- [ ] 至少 5 个不同玩家获得过回合 (TurnSync.Player 变化)
- [ ] GameOver 正常广播或达到 maxTurns 后正常结束

**执行命令**：
```bash
GOMODCACHE=${PWD}/.gomodcache go run cmd/pdcli/main.go playtest run --players 4 --max-turns 50
```

**预期结果**：
- 成功率 100%
- 耗时 < 60 秒
- 无 Error 级别日志

---

### 3.2 P0: 错误处理传播测试 (TestErrorPropagation)

**测试目标**：验证错误从 Core → Action → HSM → Nakama 的正确传播和 ErrorCode 映射

**测试场景 3.2.1: 伤害验证错误**
```go
// 触发条件：DamageAction 尝试造成负数伤害
// 预期：ValidationError → ErrInvalidParameter (1001)
// 验证：ActionRejected.Reason 包含 "damage_amount"
```

**测试场景 3.2.2: 玩家不存在错误**
```go
// 触发条件：非当前回合玩家尝试行动
// 预期：InternalError → ErrPlayerNotFound (2001)
// 验证：ActionRejected.OpCode 对应请求的 OpCode
```

**测试场景 3.2.3: 状态执行错误**
```go
// 触发条件：HSM 状态 Enter/Update 返回错误
// 预期：HSMError → ErrInvalidState (1002) 或 ErrInternal (3001)
// 验证：日志包含状态名和错误详情
```

**验证点**：
- [ ] ActionRejected 消息正确解析
- [ ] ErrorCode 与错误类型映射正确
- [ ] 错误日志包含足够上下文（状态名、玩家 ID、错误原因）

**执行命令**：
```bash
# 需要注入错误场景（通过测试 Hook 或修改代码触发）
GOMODCACHE=${PWD}/.gomodcache go test ./internal/nakama/... -run TestErrorCodeForError -v
```

---

### 3.3 P0: 决策中断测试 (TestDecisionInterrupt)

**测试目标**：验证 WaitDecisionState 正确推入状态栈，等待玩家选择后恢复

**测试步骤**：
1. 触发需要决策的事件（如某些特殊格子事件）
2. 验证收到 DecisionRequest 消息
3. 验证 HSM 状态栈推入 WaitDecision
4. 发送 UserChoice
5. 验证 WaitDecision 弹出，流程恢复

**验证点**：
- [ ] DecisionRequest.ID 有效（非空 UUID）
- [ ] DecisionRequest.Options 数量 > 0
- [ ] UserChoice 后流程继续（收到下一个 TurnSync）
- [ ] 超时后自动选择默认选项

**执行命令**：
```bash
GOMODCACHE=${PWD}/.gomodcache go run cmd/pdcli/main.go playtest run --players 2 --verbose
```

---

### 3.4 P1: 小游戏流程测试 (TestMiniGameFlow)

**测试目标**：验证小游戏开始 → 排名提交 → 结果广播完整流程

**测试步骤**：
1. 等待 MiniGameStart 消息
2. 验证游戏类型和玩家列表
3. 发送 MiniGameResultSubmit
4. 等待 MiniGameResult 广播
5. 验证排名无重复

**验证点**：
- [ ] MiniGameStart.Players 数量与房间玩家数一致
- [ ] MiniGameResult.Rankings 数量与玩家数一致
- [ ] 排名 1-N 无重复（N=玩家数）
- [ ] 小游戏后回合正常继续

**当前问题**：
- CLI 未完整处理 MiniGameResult 消息
- 需要修复 autoplay.go 的 MiniGameStart 处理逻辑

**修复计划**：
```go
// internal/cli/scenario/autoplay.go
func (p *AutoPlayPlayer) handleMiniGameStart(client nakama.IClient, msg *protocol.MiniGameStart) {
    // 当前问题：所有玩家使用相同随机数，导致排名重复
    // 修复：使用玩家 ID 哈希生成唯一排名
    playerHash := hash(p.UserID)
    rank := (playerHash % len(msg.Players)) + 1
    client.SendMiniGameResult(rank)
}
```

---

### 3.5 P1: 阵营技能测试 (TestFactionSkill)

**测试目标**：验证青龙/朱雀/白虎/玄武技能正确触发

**测试场景**：
| 阵营 | 技能 | 触发条件 | 验证点 |
|------|------|---------|--------|
| 青龙 | 行迹 | 每 5 回合 | 充能计数 +1，下一回合无视地形 |
| 朱雀 | 离火 | 每 4 回合 | LP+1 (max 8) |
| 白虎 | 劫运 | 超越他人时 | 偷取随机 Buff |
| 玄武 | 镇厄 | 每 5 回合 | 充能计数 +1，抵消一个坏事件 |

**验证点**：
- [ ] StateSync.Player.Charge 正确更新
- [ ] StateSync.Player.LP 变化符合预期
- [ ] Buff 变化正确记录到 TurnSync.Entries

**执行命令**：
```bash
# 需要长回合测试验证技能触发
GOMODCACHE=${PWD}/.gomodcache go run cmd/pdcli/main.go playtest run --players 2 --max-turns 100 --verbose
```

---

### 3.6 P1: 道具使用测试 (TestItemUsage)

**测试目标**：验证道具抽取 → 使用 → 效果执行完整流程

**测试步骤**：
1. 等待 Available 消息（包含 Items 列表）
2. 发送 UseItem 请求
3. 验证 ActionRejected 或 TurnSync 包含道具效果

**验证点**：
- [ ] Available.Items 非空时可以使用道具
- [ ] UseItem.ItemID 有效
- [ ] 道具效果正确记录到 GameLog

**待实现**：
- 需要验证道具目标选择（target_id）

---

### 3.7 P2: 断线重连测试 (TestReconnection)

**测试目标**：验证玩家断线后重连可正确恢复状态

**测试步骤**：
1. 玩家 A 加入房间
2. 运行 5 回合后玩家 A 断开连接
3. 玩家 B 继续游戏 3 回合
4. 玩家 A 重新连接
5. 验证收到 FullSync 且状态正确

**验证点**：
- [ ] 重连后 StateSync 与断线前一致
- [ ] 重连后 TurnSync 包含断线期间的日志
- [ ] 重连玩家 ID 保持一致

**待实现**：
- CLI 需要支持模拟断线（关闭 WebSocket）
- 需要实现 FullSync 消息处理

---

### 3.8 P2: Soak 压力测试 (TestSoakStability)

**测试目标**：验证长时间运行稳定性

**测试步骤**：
1. 连续运行 20 局对局
2. 每局 2-4 人随机
3. 统计成功率和平均耗时

**验证点**：
- [ ] 成功率 >= 90%
- [ ] 无内存泄漏（内存增长 < 10%）
- [ ] 无 goroutine 泄漏
- [ ] 无 fatal/panic

**执行命令**：
```bash
GOMODCACHE=${PWD}/.gomodcache go run cmd/pdcli/main.go playtest soak --players 2 --rounds 20
```

---

## 4. 协议覆盖分析

### 4.1 当前协议覆盖状态

| OpCode | 方向 | 消息类型 | CLI 支持状态 |
|--------|------|---------|-------------|
| **Server → Client** | | | |
| OpStateSync(1) | S→C | StateSync | ✅ 已支持 |
| OpTurnSync(2) | S→C | TurnSync | ✅ 已支持 |
| OpDecisionRequest(3) | S→C | Decision | ✅ 已支持 |
| OpAvailable(4) | S→C | Available | ✅ 已支持（新增道具/技能使用策略） |
| OpMiniGameStart(5) | S→C | MiniGameStart | ✅ 已支持（排名策略已修复） |
| OpMiniGameResult(6) | S→C | MiniGameResult | ✅ 已支持 |
| OpGameOver(7) | S→C | GameOver | ✅ 已支持 |
| OpFullSync(8) | S→C | StateSync | ✅ 已支持（重连场景） |
| OpActionRejected(9) | S→C | ActionRejected | ✅ 已支持（记录拒绝统计） |
| **Client → Server** | | | |
| OpRollDice(100) | C→S | RollDice | ✅ 已支持 |
| OpUseItem(101) | C→S | UseItem | ✅ 已支持（自动使用第一个道具） |
| OpUseSkill(102) | C→S | UseSkill | ✅ 已支持（自动使用 faction 技能） |
| OpUserChoice(103) | C→S | UserChoice | ✅ 已支持 |
| OpMiniGameResultSubmit(104) | C→S | MiniGameResultSubmit | ✅ 已支持 |

## 5. 已完成的实施

### Phase 1: 修复现有问题 (P0)

#### 任务 1.1: 修复 MiniGameStart 排名策略 ✅
**文件**: `internal/cli/scenario/autoplay.go`
**改动**: 使用玩家索引生成唯一排名（索引 0→排名 1, 索引 1→排名 2），避免所有玩家使用相同随机种子导致排名重复

#### 任务 1.2: 添加 ActionRejected 处理 ✅
**文件**: `internal/cli/scenario/autoplay.go`
**改动**:
- 添加 `rejections` 字段记录拒绝次数
- 添加 `lastRejection` 字段记录最后一次拒绝详情
- 在 `handleActionRejected` 中更新统计

#### 任务 1.3: 增强 Result 结构 ✅
**文件**: `internal/cli/scenario/autoplay.go`
**改动**:
- 添加 `Rejections int` 字段
- 添加 `LastError string` 字段
- 更新 `runNakamaPlay` 聚合拒绝统计
- 更新 `printSummary` 显示拒绝信息

---

### Phase 2: 实现缺失协议 (P1)

#### 任务 2.1: 实现 OpMiniGameResult 处理 ✅
**文件**: `internal/cli/scenario/autoplay.go`
**改动**: 添加 `handleMiniGameResult` 方法，记录自己获得的排名

#### 任务 2.2: 实现 OpFullSync 处理 ✅
**文件**: `internal/cli/scenario/autoplay.go`
**改动**: 添加 `handleFullSync` 方法，处理重连场景的状态同步

#### 任务 2.3: 实现 OpUseItem 支持 ✅
**文件**: `internal/cli/scenario/autoplay.go`
**改动**: 修改 `handleAvailable`，当有道具时自动使用第一个道具

#### 任务 2.4: 实现 OpUseSkill 支持 ✅
**文件**: `internal/cli/scenario/autoplay.go`
**改动**: 修改 `handleAvailable`，当技能可用时自动使用 faction 技能

---

## 6. 待实施的测试场景

### 6.1 错误注入测试 (P1)

**文件**: `internal/cli/command/playtest.go`
**改动**: 添加 `--inject-error` 标志，注入特定错误验证处理

### 6.2 断线重连测试 (P2)

**文件**: `internal/cli/command/playtest.go`
**改动**: 添加 `--test-reconnection` 标志，模拟断线和重连

### 6.3 道具目标选择测试 (P2)

**文件**: `internal/cli/scenario/autoplay.go`
**改动**: 改进 UseItem 策略，支持选择目标

---

## 7. 测试执行清单

### 7.1 日常开发测试

```bash
# 快速验证 (2 人 10 回合)
GOMODCACHE=${PWD}/.gomodcache go run cmd/pdcli/main.go playtest run --players 2 --max-turns 10

# 完整验证 (4 人 50 回合)
GOMODCACHE=${PWD}/.gomodcache go run cmd/pdcli/main.go playtest run --players 4 --max-turns 50 --output ./reports/test.json

#  verbose 模式查看详细日志
GOMODCACHE=${PWD}/.gomodcache go run cmd/pdcli/main.go playtest run --players 2 --verbose
```

### 7.2 发布前测试

```bash
# Soak 测试 (20 轮)
GOMODCACHE=${PWD}/.gomodcache go run cmd/pdcli/main.go playtest soak --players 2 --rounds 20

# 多人数测试
for i in 2 3 4; do
    GOMODCACHE=${PWD}/.gomodcache go run cmd/pdcli/main.go playtest run --players $i --max-turns 50
done
```

### 7.3 错误处理验证

```bash
# 运行错误码映射测试
GOMODCACHE=${PWD}/.gomodcache go test ./internal/nakama/... -run TestErrorCodeForError -v

# 运行 Action 错误测试
GOMODCACHE=${PWD}/.gomodcache go test ./internal/engine/action/... -run TestError -v
```

---

## 8. 测试报告格式

### 8.1 控制台输出示例

```
========== Match Results ==========
Status: Success
Duration: 45.23 seconds
Messages Received: 342
Turns Completed: 24
Rejections: 3
====================================
```

### 8.2 JSON 报告示例

```json
{
  "success": true,
  "duration_seconds": 45.23,
  "messages_received": 342,
  "turns_completed": 24,
  "players_count": 4,
  "max_turns": 50,
  "rejections": 3,
  "last_error": ""
}
```

---

## 9. 成功标准

| 测试类型 | 成功标准 | 优先级 |
|---------|---------|--------|
| 基础对局 | 2-4 人可完整完成一局 | P0 |
| 错误处理 | ErrorCode 映射正确率 100% | P0 |
| 决策中断 | DecisionRequest/Response 闭环成功 | P0 |
| 小游戏 | 排名无重复，流程正常 | P1 |
| Soak | 20 轮成功率 >= 90% | P1 |
| 道具使用 | UseItem 可正确执行 | P2 |
| 技能使用 | UseSkill 可正确触发 | P2 |
| 断线重连 | FullSync 状态一致 | P2 |

---

## 10. 已知问题和限制

### 10.1 当前限制

1. **道具目标选择**: UseItem 暂不支持选择目标（target_id 为空）
2. **断线重连测试**: 尚未实现自动断线和重连模拟
3. **错误注入测试**: 尚未实现错误注入功能

### 10.2 已完成功能

以下问题已在本计划实施中解决：

| 问题 | 状态 | 说明 |
|------|------|------|
| MiniGameStart 排名 | ✅ 已修复 | 使用玩家索引生成唯一排名 |
| ActionRejected 处理 | ✅ 已完成 | 记录拒绝次数和详情到 Result |
| OpMiniGameResult | ✅ 已实现 | 添加 handleMiniGameResult 方法 |
| OpFullSync | ✅ 已实现 | 添加 handleFullSync 方法 |
| OpUseItem | ✅ 已实现 | 自动使用第一个道具 |
| OpUseSkill | ✅ 已实现 | 自动使用 faction 技能 |

---

## 11. 参考资料

- [internal/cli/README.md](../internal/cli/README.md) - CLI 使用文档
- [pkg/net/README.md](../pkg/net/README.md) - 协议层文档
- [doc/technical_plan.md](./technical_plan.md) - 技术规划书
- [doc/protocol_hsm_interaction.md](./protocol_hsm_interaction.md) - 协议-HSM 交互流程
