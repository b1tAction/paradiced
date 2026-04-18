# Package internal/cli - 命令行测试工具

本包提供 Paradiced 游戏后端的命令行测试工具，用于快速验证游戏可玩性和进行回归测试。

## 设计目标

1. **快速验证**：最短路径验证后端可玩性，不依赖前端
2. **自动化回归**：支持 2-4 玩家自动对局，用于稳定性检查
3. **结构化日志**：输出结构化日志，便于定位协议与状态机问题
4. **报告生成**：生成 JSON 测试报告，包含成功率、耗时、错误统计

## 目录结构

```
internal/cli/
├── README.md           # 本文件
├── command/
│   ├── root.go         # 根命令定义
│   ├── playtest.go     # playtest 命令（run/soak）
│   └── ...
├── model/
│   └── message.go      # CLI 协议消息类型定义
├── nakama/
│   ├── client.go       # Nakama HTTP 客户端
│   └── socket.go       # Nakama WebSocket 客户端
├── scenario/
│   └── autoplay.go     # 自动化对局场景
└── report/
    └── summary.go      # 测试报告生成（待实现）
```

## 命令设计

### pdcli playtest run

运行单次自动化对局：

```bash
pdcli playtest run \
  --players 4 \
  --match-name paradiced_match \
  --max-turns 50 \
  --timeout 180 \
  --output ./artifacts/playtest.json
```

**参数说明**：

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `--players` | 4 | 玩家数量 (1-4) |
| `--match-name` | paradiced_match | 匹配名称 |
| `--max-turns` | 50 | 最大回合数 |
| `--timeout` | 180 | 超时时间 (秒) |
| `--output` | - | 输出 JSON 报告路径 |
| `--server-http` | http://127.0.0.1:7350 | Nakama HTTP 地址 |
| `--server-ws` | ws://127.0.0.1:7350/ws | Nakama WS 地址 |
| `--server-key` | defaultkey | Nakama 服务器密钥 |
| `--verbose` | false | 详细日志输出 |

### pdcli playtest soak

运行多次浸泡测试，验证稳定性：

```bash
pdcli playtest soak \
  --players 2 \
  --rounds 20 \
  --verbose
```

**参数说明**：

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `--players` | 2 | 玩家数量 (1-4) |
| `--rounds` | 20 | 测试轮数 |
| `--server-http` | http://127.0.0.1:7350 | Nakama HTTP 地址 |
| `--server-ws` | ws://127.0.0.1:7350/ws | Nakama WS 地址 |
| `--server-key` | defaultkey | Nakama 服务器密钥 |
| `--verbose` | false | 详细日志输出 |

## 自动策略

CLI 内置默认自动策略，用于快速验证游戏流程：

| 触发条件 | 响应动作 | 说明 |
|----------|----------|------|
| 收到 `Available` | 发送 `RollDice` | 优先掷骰子 |
| 收到 `DecisionRequest` | 发送 `UserChoice{Choice: 0}` | 默认选择第一个选项 |
| 收到 `MiniGameStart` | 发送 `MiniGameResultSubmit{Rank: 随机}` | 临时策略：随机排名 |
| 收到 `GameOver` | 退出对局 | 对局结束 |

**注意**：小游戏排名策略需要改进，当前随机策略可能导致排名不唯一。

## 协议覆盖

CLI 需要覆盖以下服务端 OpCode（1-99）：

| OpCode | 名称 | 数据类型 | 处理状态 |
|--------|------|----------|----------|
| 1 | `OpStateSync` | `StateSync` | ✅ 已实现 |
| 2 | `OpTurnSync` | `TurnSync` | ✅ 已实现 |
| 3 | `OpDecisionRequest` | `Decision` | ✅ 已实现 |
| 4 | `OpAvailable` | `Available` | ✅ 已实现 |
| 5 | `OpMiniGameStart` | `MiniGameStart` | ✅ 已实现 |
| 6 | `OpMiniGameResult` | `MiniGameResult` | ⏳ 待实现 |
| 7 | `OpGameOver` | `GameOver` | ✅ 已实现 |
| 8 | `OpFullSync` | `FullSync` | ⏳ 待实现 |

客户端发送 OpCode（100+）：

| OpCode | 名称 | 数据类型 | 处理状态 |
|--------|------|----------|----------|
| 100 | `OpRollDice` | `RollDice` | ✅ 已实现 |
| 101 | `OpUseItem` | `UseItem` | ⏳ 待实现 |
| 102 | `OpUseSkill` | `UseSkill` | ⏳ 待实现 |
| 103 | `OpUserChoice` | `UserChoice` | ✅ 已实现 |
| 104 | `OpMiniGameResultSubmit` | `MiniGameResultSubmit` | ✅ 已实现 |

## 使用示例

### 单次对局测试

```bash
# 2 玩家快速测试
pdcli playtest run --players 2 --max-turns 20 --verbose

# 4 玩家完整测试，输出 JSON 报告
pdcli playtest run --players 4 --output ./report.json
```

### 浸泡测试

```bash
# 20 轮 2 玩家稳定性测试
pdcli playtest soak --players 2 --rounds 20

# 预期输出：成功率 >= 90%
```

### 查看测试报告

```bash
# JSON 报告内容示例
cat ./report.json | jq

# {
#   "success": true,
#   "duration": 120500000000,  // 纳秒
#   "messages_received": 156,
#   "turns_completed": 8,
#   "global_state": "game_over"
# }
```

## 输出示例

```
$ pdcli playtest run --players 2 --verbose

INFO 开始对局测试 players=2 max_turns=50
INFO 所有玩家已连接，开始匹配...
INFO 玩家已添加到匹配器 player=1 ticket=ticket-xxx
INFO 玩家已添加到匹配器 player=2 ticket=ticket-yyy
INFO 匹配器匹配成功，返回 match_id match_id=match-abc123
INFO 所有玩家已加入匹配 match_id=match-abc123 players=2
INFO 收到状态同步 global=match_init turn= round=0 players=2
INFO 收到小游戏开始 game_type=dice_race players=[player-001,player-002]
INFO 已提交小游戏随机结果 rank=2
INFO 收到小游戏结果 rankings=[{player-001 1} {player-002 2}]
INFO 收到可用动作 items=0 can_use_skill=false dice_type=gold
INFO 自动策略：掷骰子
...
========== 对局结果 ==========
状态：成功
耗时：120.50 秒
消息数：156
回合数：8
==============================
INFO 对局成功完成
```

## 消息模型

定义在 `internal/cli/model/message.go`：

```go
// StateSync - 完整游戏状态同步
type StateSync struct {
    GlobalState string   `json:"global_state"`
    TurnState   string   `json:"turn_state"`
    TurnPlayer  string   `json:"turn_player"`
    Round       int      `json:"round"`
    Turn        int      `json:"turn"`
    Paused      bool     `json:"paused"`
    Players     []Player `json:"players"`
}

// TurnSync - 回合内效果列表
type TurnSync struct {
    Round   int                `json:"round"`
    Turn    int                `json:"turn"`
    Player  string             `json:"player"`
    Entries []gamelog.LogEntry `json:"entries"`
}

// Decision - 决策请求
type Decision struct {
    ID      string   `json:"id"`
    Prompt  string   `json:"prompt"`
    Context string   `json:"context"`
    Options []Option `json:"options"`
    Timeout int      `json:"timeout"`
    Default int      `json:"default"`
}

// Available - 可用操作列表
type Available struct {
    Items       []Item `json:"items"`
    CanUseSkill bool   `json:"can_use_skill"`
    DiceType    string `json:"dice_type"`
}
```

## 架构定位

```
┌─────────────────────────────────────────────────────────────┐
│                    pdcli (cmd/pdcli/main.go)                  │
│  - Cobra 命令组织                                            │
│  - 配置管理（flags/env）                                      │
├─────────────────────────────────────────────────────────────┤
│                    internal/cli/command                        │
│  - playtest run/soak 命令实现                                │
│  - 报告输出                                                  │
├─────────────────────────────────────────────────────────────┤
│                    internal/cli/scenario                       │
│  - RunAutoPlay - 自动化对局场景                              │
│  - AutoPlayPlayer - 自动玩家（监听 + 响应）                   │
├─────────────────────────────────────────────────────────────┤
│                    internal/cli/nakama                         │
│  - Client - HTTP 认证客户端                                   │
│  - SocketClient - WebSocket 消息收发                          │
├─────────────────────────────────────────────────────────────┤
│                    internal/cli/model                          │
│  - 消息类型定义（StateSync, TurnSync, Decision 等）           │
└─────────────────────────────────────────────────────────────┘
```

## 依赖

| 依赖 | 用途 |
|------|------|
| `github.com/heroiclabs/nakama-common` | Nakama 通用接口 |
| `github.com/ascii8/nakama-go` | Nakama Go SDK |
| `github.com/spf13/cobra` | CLI 命令组织 |
| `go.uber.org/zap` | 结构化日志 |

## 测试

```bash
# 运行 CLI 相关测试
go test ./internal/cli/... -v

# 运行协议模型测试
go test ./internal/cli/model/... -v
```

## 故障排查

### 匹配器超时

**现象**：`匹配器超时 (收到 0/4 响应)`

**原因**：
1. Nakama 服务器未启动
2. 匹配查询条件不匹配
3. 网络延迟过高

**解决方案**：
1. 检查 Nakama 服务：`docker ps | grep nakama`
2. 检查日志：`docker logs nakama`
3. 增加超时：`--timeout 30`

### 小游戏卡死

**现象**：对局在 `match_init` 或 `round_mini_game` 状态卡住

**原因**：小游戏排名提交逻辑有误，所有玩家提交相同排名

**解决方案**：修复 `autoplay.go` 中的排名分配逻辑（见 Issue #XX）

### 协议解析失败

**现象**：`解析 StateSync 失败：json: cannot unmarshal...`

**原因**：服务端协议字段变更

**解决方案**：
1. 检查服务端 `pkg/net/sync.go` 字段定义
2. 更新 CLI 模型 `internal/cli/model/message.go`
3. 运行测试验证：`go test ./internal/cli/model/... -v`

## 待办事项

1. ⏳ 实现 `OpMiniGameResult` 处理
2. ⏳ 实现 `OpFullSync` 处理（断线重连测试）
3. ⏳ 实现道具使用测试（`OpUseItem`）
4. ⏳ 实现阵营技能测试（`OpUseSkill`）
5. ⏳ 改进小游戏排名策略（确保唯一性）
6. ⏳ 添加 ActionRejected 处理

## 相关文档

- [doc/internal/net_protocol.md](../../doc/internal/net_protocol.md) - 协议层完整设计
- [pkg/net/README.md](../../pkg/net/README.md) - 协议层包文档
- [internal/nakama/README.md](../nakama/README.md) - Nakama 适配层
- [cli.md](../../cli.md) - CLI 开发计划
