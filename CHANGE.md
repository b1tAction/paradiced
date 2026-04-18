# CHANGE

## 2026-04-19 14:00:00 CST

### 已完成任务

本次迭代完成了 CLI 与协议层的多项关键修复和功能增强：

#### 1. 修复小游戏排名提交逻辑

- **文件**: `internal/cli/scenario/autoplay.go:428-430`
- **问题**: 原先使用 `rand.Intn(maxRank)` 导致所有玩家可能提交相同排名（如 [2,2,1,2]）
- **修复**: 使用 `rand.Perm(maxRank)` 生成唯一排列，每个玩家根据索引获得唯一排名
- **验证**: `pdcli playtest run --players 4` 小游戏阶段正常

#### 2. 实现 ActionRejected 协议层发送逻辑

- **协议定义**:
  - `pkg/net/opcode.go`: 新增 `OpActionRejected` (OpCode 9)
  - `pkg/net/sync.go`: 新增 `ActionRejected` 结构体（OpCode/Reason/Message）
- **Nakama 实现**:
  - `internal/nakama/broadcast.go`: 新增 `SendActionRejected` 方法
  - `internal/nakama/handler.go`: 新增 `sendActionRejected` 辅助方法
  - `internal/nakama/message.go`: 在 `handleRollDice`、`handleUseItem`、`handleUseSkill` 中添加拒绝反馈
- **拒绝场景**:
  - 玩家不存在：`player_not_found`
  - 非当前回合玩家：`not_current_player`
  - 状态不允许：`invalid_state`

#### 3. CLI 接入 ActionRejected 处理

- **文件**:
  - `internal/cli/model/message.go`: 新增 `ActionRejected` 模型
  - `internal/cli/scenario/autoplay.go:469-480`: 新增 `handleActionRejected` 处理器
- **效果**: CLI 收到拒绝消息后记录警告日志，避免"已发送但无响应"的黑盒体验

#### 4. MainAction 超时配置化

- **文件**:
  - `internal/engine/hsm/hsm.go`: 新增 `HSMConfig` 结构体和 `DefaultHSMConfig` 函数
  - `internal/engine/hsm/turn_states.go`: `NewMainActionState` 接收 timeout 参数
- **配置**:
  - 默认超时：45 秒
  - 开发环境（`PD_DEV=true`）：10 秒
- **效果**: 开发环境下每回合等待时间大幅缩短，改善调试体验

#### 5. 消除 Metadata 转换层

- **文件**:
  - `internal/nakama/presence.go`: `HandlePresenceJoin` 和 `parseFactionFromMetadata` 改为接收 `*util.Metadata`
  - `internal/nakama/adapter.go`: 移除 `metadataToStringMap` 函数，简化调用
- **效果**: 统一元数据管理方式，减少中间转换层

#### 6. 文档补充

- **新增**:
  - `internal/cli/README.md`: CLI 工具完整文档（命令设计、协议覆盖、自动策略）
- **更新**:
  - `internal/nakama/README.md`: 添加 Server→Client OpCode 路由表、ActionRejected 使用示例、日志调试章节
  - `CLAUDE.md`: 添加 ActionRejected OpCode 说明、CLI 工具文档

### 已提交 Commits

1. `docs: add CLI and update package READMEs` - 文档补充
2. `feat(pkg/net): add ActionRejected message type` - 协议定义
3. `fix(cli): 修复小游戏排名提交逻辑` - 排名修复
4. `feat(nakama): 实现 ActionRejected 发送逻辑` - Nakama 实现
5. `feat(cli): 接入 ActionRejected 处理` - CLI 接入
6. `feat(hsm): MainAction 超时配置化` - 超时配置
7. `refactor(nakama): 消除 Metadata 转换层` - Metadata 重构

### 后续待完善任务

#### Phase 3: 稳定性验证（优先级 P0）

1. **2 玩家 20 轮浸泡测试**
   ```bash
   pdcli playtest soak --players 2 --rounds 20
   ```
   **目标**: 成功率 ≥ 95%

2. **4 玩家 10 轮浸泡测试**
   ```bash
   pdcli playtest soak --players 4 --rounds 10
   ```
   **目标**: 成功率 ≥ 90%

3. **断线重连测试**
   - 手动验证或 CLI 模拟断线
   - 验证 FullSync 正常发送

#### Phase 4: 补充测试和文档（优先级 P1）

1. **ActionRejected 单元测试**
   - 文件：`internal/nakama/message_test.go`
   - 测试：验证非当前玩家掷骰子被拒绝

2. **CLI 模型测试**
   - 文件：`internal/cli/model/message_test.go`
   - 测试：ActionRejected JSON 解析

3. **更新 doc/internal/nakama.md**
   - 添加 ActionRejected 使用示例
   - 添加日志调试章节

### 验证计划

```bash
# 单元测试
GOMODCACHE=/app/.gomodcache go test ./pkg/net/... ./internal/nakama/... ./internal/cli/... -v

# 2 玩家快速测试
pdcli playtest run --players 2 --max-turns 20

# 4 玩家完整测试
pdcli playtest run --players 4 --max-turns 50

# 20 轮浸泡测试
pdcli playtest soak --players 2 --rounds 20
```
