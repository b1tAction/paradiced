# Paradiced 服务器启动指南

本文档描述如何启动 Paradiced 游戏服务器。

## 前置条件

- Docker 和 Docker Compose
- Go 1.22+ （用于构建插件）
- CGO 支持（Go 插件模式需要）

## 快速启动

```bash
# 1. 构建 Nakama 插件
make build-plugin

# 2. 启动服务器
make docker-up

# 3. 查看日志
make docker-logs
```

## 端口说明

| 端口 | 用途 |
|------|------|
| 7350 | WebSocket/API（客户端连接） |
| 7351 | HTTP API |
| 7349 | gRPC API |
| 7352 | Web 控制台 |

## Web 控制台

启动后访问 http://localhost:7352 登录管理控制台：
- 用户名: `admin`
- 密码: `password123`

## 架构说明

```
┌─────────────────────────────────────────────────────┐
│                    Docker                            │
│  ┌──────────────────┐    ┌────────────────────────┐ │
│  │   PostgreSQL     │    │     Nakama Server      │ │
│  │   (port 5432)    │───▶│   (port 7350-7352)     │ │
│  │                  │    │                        │ │
│  │   数据库存储      │    │   ┌──────────────────┐ │ │
│  │                  │    │   │  paradiced.so    │ │ │
│  │                  │    │   │  (Go 插件)       │ │ │
│  │                  │    │   └──────────────────┘ │ │
│  └──────────────────┘    └────────────────────────┘ │
└─────────────────────────────────────────────────────┘
```

## Makefile 命令

| 命令 | 说明 |
|------|------|
| `make build-plugin` | 构建 Nakama 插件（.so 文件） |
| `make build-dev` | 开发模式验证编译 |
| `make test` | 运行测试 |
| `make docker-up` | 启动 Docker 服务 |
| `make docker-down` | 停止 Docker 服务 |
| `make docker-clean` | 停止并删除数据 |
| `make docker-logs` | 查看 Nakama 日志 |
| `make dev` | 构建插件并启动服务器 |
| `make dev-logs` | 启动并查看日志 |

## 客户端连接

客户端使用 WebSocket 连接到 `ws://localhost:7350`：

```typescript
// 创建匹配
const match = await client.createMatch(token, "paradiced_match");

// 加入匹配
const result = await socket.joinMatch(match.match_id);

// 发送消息
socket.sendMatchState(match.match_id, OpCode.RollDice, {});

// 接收消息
socket.onMatchData = (matchData) => {
    // 处理服务器消息
};
```

## 消息 OpCode

| OpCode | 客户端→服务器 | 服务器→客户端 |
|--------|--------------|--------------|
| 1 | - | StateSync |
| 2 | - | TurnSync |
| 3 | - | Decision |
| 4 | - | Available |
| 5 | - | MiniGameStart |
| 6 | - | MiniGameResult |
| 7 | - | GameOver |
| 100 | RollDice | - |
| 101 | UseItem | - |
| 102 | UseSkill | - |
| 103 | UserChoice | - |
| 104 | MiniGameResultSubmit | - |

## 开发建议

### 调试模式

修改 `config.yml` 中的日志级别：

```yaml
logger:
  level: "debug"  # 启用详细日志
```

### 热重载

插件修改后需要重新构建并重启 Nakama：

```bash
make build-plugin
make docker-down
make docker-up
```

### 数据持久化

停止服务但保留数据：

```bash
make docker-down
```

完全清理数据：

```bash
make docker-clean
```

## 相关文档

- [doc/internal/nakama.md](../doc/internal/nakama.md) - Nakama 集成设计
- [pkg/net/README.md](../pkg/net/README.md) - 协议层接口
- [Nakama 官方文档](https://heroiclabs.com/docs/)