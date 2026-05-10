# Paradiced 服务器启动指南

本文档描述 Paradiced 后端在本地开发和生产环境中的启动、端口和部署边界。

## 前置条件

- Docker 和 Docker Compose V2
- Go 1.22+（用于本地测试和 CLI 构建）
- CGO 支持（Go 插件模式需要）

## 本地快速启动

```bash
# 1. 构建 Nakama 插件
make build-plugin

# 2. 启动服务器（包含 CockroachDB）
make docker-up

# 3. 查看服务状态
make status

# 4. 查看 Nakama 日志
make docker-logs
```

`make rebuild` 会重新构建插件并重启 `nakama` 容器，适合代码变更后的本地迭代。

## 端口说明

### 本地开发

| 入口 | 地址 | 说明 |
|---|---|---|
| Nakama HTTP API | `http://127.0.0.1:17350/v2/...` | 宿主机 loopback，仅本机访问 |
| Nakama WebSocket | `ws://127.0.0.1:17350/ws` | 宿主机 loopback，仅本机访问 |
| CockroachDB SQL | `localhost:26257` | 本地数据库调试 |
| CockroachDB Admin UI | `http://localhost:8080` | 本地数据库调试 |

Nakama gRPC `7349` 和 Console `7351` 只在容器内部暴露，不发布为宿主机端口。

### 生产环境

生产环境的公网入口只经过宿主机 nginx，并统一放在 Paradice 游戏命名空间下：

| 用途 | 公网地址 | nginx upstream |
|---|---|---|
| 游戏前端 | `https://bitaction.cn/game/paradice/` | `/var/www/paraweb/current` |
| HTTP API | `https://bitaction.cn/game/paradice/api/v2/...` | `127.0.0.1:17350`，转发为 Nakama 原生 `/v2/...` |
| WebSocket | `wss://bitaction.cn/game/paradice/api/ws` | `127.0.0.1:17350`，转发为 Nakama 原生 `/ws` |

`https://bitaction.cn/game/` 直接 `308` 到 `https://bitaction.cn/game/paradice/`。旧 `/v2/...` 与 `/ws` 不保留生产兼容 proxy；切换后不再作为 Paradice 入口。生产不公网暴露 `7349`、`7350`、`7351`。如果需要排查 Nakama upstream，应在服务器本机访问 `127.0.0.1:17350`，不要临时开放公网业务端口作为常规排障方式。

## Web 控制台

Nakama Console 不作为生产公网入口。开发环境如需访问 Console，应通过容器内部网络、临时受控维护入口或 SSH tunnel，并先更改默认弱密码。

CockroachDB Admin UI 可在本地通过 `http://localhost:8080` 查看数据库状态；生产公网访问由云安全组阻断。

## 架构说明

```text
Internet
  -> https://bitaction.cn/game/                      -> host nginx -> 308 /game/paradice/
  -> https://bitaction.cn/game/paradice/             -> host nginx -> /var/www/paraweb/current
  -> https://bitaction.cn/game/paradice/api/v2/...   -> host nginx -> 127.0.0.1:17350 -> Nakama /v2/... on :7350
  -> wss://bitaction.cn/game/paradice/api/ws         -> host nginx -> 127.0.0.1:17350 -> Nakama /ws on :7350

Docker internal network
  cron-cleanup -> nakama:7350
  nakama       -> cockroachdb:26257
```

`docker-compose.yml` 使用 `127.0.0.1:17350:7350` 作为宿主机 nginx upstream。`cron-cleanup` 继续通过 Compose 网络内的 `http://nakama:7350/...` 调用 Nakama，不依赖宿主机端口发布。

## 生产部署流程

后端生产发布由 `Deploy Production` GitHub Actions workflow 触发：

- 开发分支 PR 合并到 `master`，或人工执行 `workflow_dispatch`。
- Actions 运行 `go mod verify`、`go test ./...` 和构建检查。
- Actions 通过 `git archive` 生成不含 `.git` 的 source archive。
- Archive 上传到 `/opt/paradiced/incoming/<sha>.tar.gz`。
- 服务器通过受控 sudo wrapper `/usr/local/sbin/paradiced-deploy-archive` 调用 root-owned 固定部署实现 `/usr/local/lib/paradiced/deploy-archive.sh`，校验 archive 后同步到固定实际目录 `/opt/paradiced/current`。
- 激活流程固定为 pluginbuilder 构建 `paradiced-server.so` 后执行 `COMPOSE_PROJECT_NAME=paradiced docker compose up -d --no-deps --force-recreate nakama cron-cleanup`，确保首次迁移也会更新 Docker bind mount 与端口发布；不执行 archive 提供的 `Makefile`，也不授予 deploy 用户通用 root shell 或 Docker socket 权限。

`/opt/paradiced/current` 必须是固定实际目录，不使用 symlink 切换 release，避免 Docker Compose 相对 bind mount 指向旧 release。

## Makefile 命令

| 命令 | 说明 |
|---|---|
| `make build-plugin` | 构建 Nakama 插件（`.so` 文件） |
| `make build-dev` | 开发模式验证编译 |
| `make test` | 运行测试 |
| `make docker-up` | 启动 Docker 服务 |
| `make docker-down` | 停止 Docker 服务 |
| `make docker-clean` | 停止并删除数据，仅限明确批准后使用 |
| `make docker-logs` | 查看 Nakama 日志 |
| `make dev` | 构建插件并启动服务器 |
| `make dev-logs` | 启动并查看日志 |
| `make rebuild` | 构建插件并重启 Nakama |

## 客户端连接

生产客户端应使用单地址 endpoint：

```text
https://bitaction.cn/game/paradice/api
```

解析后对应：

```text
HTTP API: https://bitaction.cn/game/paradice/api/v2/...
WebSocket: wss://bitaction.cn/game/paradice/api/ws
```

本地客户端可使用单地址输入 `127.0.0.1:17350`，解析后对应：

```text
HTTP API: http://127.0.0.1:17350/v2/...
WebSocket: ws://127.0.0.1:17350/ws
```

## 消息 OpCode

| OpCode | 客户端→服务器 | 服务器→客户端 |
|---|---|---|
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

## 数据持久化与回滚

停止服务但保留数据：

```bash
make docker-down
```

完全清理数据：

```bash
make docker-clean
```

`make docker-clean` 会删除数据，生产环境禁止在未确认备份和维护窗口前执行。部署失败时应优先由固定部署实现回滚上一版 source snapshot，再重新执行固定 pluginbuilder / `docker compose up -d --no-deps --force-recreate nakama cron-cleanup` 激活流程，不要恢复旧的公网 `7349` / `7351` 暴露。

## 相关文档

- [doc/internal/nakama.md](internal/nakama.md) - Nakama 集成设计
- [pkg/net/README.md](../pkg/net/README.md) - 协议层接口
- [Nakama 官方文档](https://heroiclabs.com/docs/)