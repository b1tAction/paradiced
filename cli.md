# Go CLI 开发计划（基于 nakama-go）

本文档定义一个用于“快速验证 ParaDiced 可玩性”的 Go CLI 开发计划。目标是优先交付可自动化回归的 CLI，再按需扩展为 TUI。

## 1. 背景与目标

当前后端已经具备以下能力：

1. Nakama + CockroachDB 可正常启动。
2. Go 插件 `paradiced-server.so` 可被加载。
3. Match 创建函数 `paradiced_match` 已注册。

CLI 的核心目标：

1. 最短路径验证“后端可玩”而不依赖前端。
2. 支持 2-4 玩家自动对局，用于回归和稳定性检查。
3. 输出结构化日志，便于定位协议与状态机问题。

## 2. 范围定义

本期范围（MVP）：

1. 使用 `github.com/heroiclabs/nakama-go/v3` 实现客户端连接。
2. 支持批量创建/登录测试玩家。
3. 支持创建与加入 `paradiced_match`。
4. 支持接收并解析关键服务端消息（OpCode 1-8）。
5. 支持自动动作策略（优先 `roll_dice`，可选 `user_choice`）。
6. 支持对局结束报告（成功率、耗时、错误统计）。

本期不做：

1. 完整图形化界面。
2. 复杂 AI 策略（仅做规则驱动脚本）。
3. 全部业务规则重演（以可玩性验证为主）。

## 3. 技术方案

### 3.1 依赖

1. `nakama-go/v3`：认证、RPC、Socket、Match 通信。
2. `cobra`：CLI 命令组织。
3. `viper`：配置加载（可选，用于 yaml/env）。
4. `zap` 或 `log/slog`：结构化日志。
5. `testify`：单测断言。

### 3.2 建议目录

```text
cmd/
  pdcli/
    main.go
internal/cli/
  app/
    app.go
    config.go
  command/
    root.go
    auth.go
    match.go
    playtest.go
  nakama/
    client.go
    socket.go
    codec.go
  scenario/
    autoplay.go
    strategy.go
  report/
    report.go
    summary.go
  model/
    message.go
    event.go
```

### 3.3 配置模型

建议支持三层配置，优先级从高到低：

1. CLI flags。
2. 环境变量。
3. 配置文件（例如 `cli.config.yml`）。

核心配置项：

1. `server.http`（默认 `http://127.0.0.1:7350`）
2. `server.ws`（默认 `ws://127.0.0.1:7350/ws`）
3. `server.key`（默认 `defaultkey`，与 Nakama 配置一致）
4. `match.name`（默认 `paradiced_match`）
5. `players.count`（默认 `4`）
6. `play.max_turns`（默认 `50`）
7. `timeout.connect_sec`（默认 `10`）
8. `timeout.match_sec`（默认 `180`）

## 4. 命令设计

建议首批命令：

1. `pdcli auth bootstrap`
2. `pdcli match create`
3. `pdcli match join`
4. `pdcli playtest run`
5. `pdcli playtest soak`
6. `pdcli inspect opcodes`

建议参数（示例）：

```bash
pdcli playtest run \
  --players 4 \
  --match-name paradiced_match \
  --max-turns 50 \
  --timeout 180 \
  --output ./artifacts/playtest.json
```

## 5. 协议与事件映射

依据当前后端协议，CLI 需覆盖下列 OpCode：

1. 服务端 -> 客户端：
2. `1` StateSync
3. `2` TurnSync
4. `3` DecisionRequest
5. `4` Available
6. `5` MiniGameStart
7. `6` MiniGameResult
8. `7` GameOver
9. `8` FullSync

1. 客户端 -> 服务端：
2. `100` RollDice
3. `101` UseItem
4. `102` UseSkill
5. `103` UserChoice
6. `104` MiniGameResultSubmit

自动策略默认规则：

1. 收到 `Available` 时优先发送 `RollDice`。
2. 收到 `DecisionRequest` 时按预设策略发送 `UserChoice`（默认选 0）。
3. 若收到 `GameOver`，立即结算并退出。
4. 若超时或回合上限到达，记录失败并结束。

## 6. 开发里程碑

### M1：基础连通（1-2 天）

交付项：

1. CLI 骨架和配置系统。
2. 与 Nakama 建立 HTTP 认证和 WS 连接。
3. 单玩家创建并加入 match。

完成标准：

1. 可以在本地连接后端并收到至少一条服务器消息。

### M2：多玩家冒烟（2-3 天）

交付项：

1. 支持 2-4 玩家并发会话。
2. 实现 `playtest run`。
3. 实现基础自动动作（roll_dice + user_choice）。

完成标准：

1. 连续执行 20 次 `playtest run`，成功率 >= 90%。

### M3：报告与回归（1-2 天）

交付项：

1. JSON 测试报告输出。
2. 失败样本日志归档。
3. 增加关键错误码分类（认证失败、join 失败、超时、协议错误）。

完成标准：

1. 每次 run 均可输出 machine-readable 报告。

### M4：稳定性与可维护性（1-2 天）

交付项：

1. 关键模块单测。
2. scenario 级集成测试。
3. CI 任务（可选）执行 `playtest run --players 2`。

完成标准：

1. 代码覆盖率目标 >= 60%（CLI 核心逻辑）。
2. 主要流程无 data race（`go test -race` 通过）。

## 7. 详细任务拆解

### 7.1 连接与会话层

1. 封装 `NakamaHTTPClient`：认证、账号创建、会话获取。
2. 封装 `NakamaSocketClient`：连接、JoinMatch、SendMatchData、事件监听。
3. 统一错误模型：网络错误、认证错误、协议错误。

### 7.2 协议层

1. 建立 OpCode 到消息结构的 decode 路由。
2. 对未知 OpCode 保留原始 payload 并记录。
3. 统一事件总线（内部 channel）给策略层消费。

### 7.3 策略层

1. `AutoPlayStrategy` 接口：`OnState`、`OnDecision`、`OnAvailable`。
2. 默认策略 `BaselineStrategy`：满足最小可玩性验证。
3. 可扩展策略：随机策略、固定脚本策略。

### 7.4 报告层

1. 统计维度：成功、失败、平均时长、平均消息数、错误类别。
2. 输出格式：控制台摘要 + JSON 文件。
3. 失败回放上下文：最后 N 条消息快照。

## 8. 测试计划

单元测试：

1. OpCode 编解码。
2. 策略决策函数。
3. 报告聚合器。

集成测试：

1. 1 玩家连接与事件接收。
2. 4 玩家 create/join/autoplay。
3. 超时与重试机制验证。

回归测试（建议每日）：

1. `playtest run --players 4 --max-turns 50`
2. `playtest soak --rounds 30 --players 2`

## 9. 风险与对策

1. 风险：协议字段变化导致 CLI 解析失败。
2. 对策：保留 raw payload，采用宽松反序列化并记录版本。

1. 风险：网络波动导致误判不可玩。
2. 对策：引入重试和连接恢复，报告中区分环境错误与逻辑错误。

1. 风险：自动策略覆盖不足。
2. 对策：先保证关键链路（创建/加入/回合同步/结束），后续迭代策略深度。

## 10. 验收标准（DoD）

功能验收：

1. 一条命令可以完成 4 玩家自动对局验证。
2. 可以稳定识别成功结束与失败原因。
3. 结果可导出 JSON 报告。

工程验收：

1. `go test ./...` 通过。
2. `go test -race ./...` 在核心包通过。
3. README 中新增 CLI 使用说明。

## 11. TUI 扩展计划（后续）

在 CLI 稳定后再扩展 TUI：

1. 复用同一连接层与协议层。
2. 仅替换展示与输入层（建议 Bubble Tea）。
3. TUI 第一版只做“观战 + 手动触发 roll_dice/user_choice”。

这样可以保持核心逻辑单一，减少双实现带来的维护成本。
