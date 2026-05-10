# ParaDiced 游戏后端

本项目后端由以下组件组成：

- CockroachDB（游戏数据存储）
- Nakama（HTTP API / WebSocket 网关）
- Paradiced Go 插件（权威 Match 逻辑，Match 名称：`paradiced_match`）

## 前置要求

- Linux / macOS，或 Windows 上的 Git Bash / WSL 风格 shell
- Docker（建议 24+）
- Docker Compose V2（命令是 `docker compose`）

说明：

- 不需要本机安装特定 Go 版本来构建 Nakama 插件。
- 插件构建已通过 `Makefile` 固定使用 `heroiclabs/nakama-pluginbuilder:3.22.0`。

## 本地启动流程

1. 构建 Paradiced 插件。

   ```shell
   make build-plugin
   ```

2. 启动数据库和 Nakama。

   ```shell
   docker compose up --build -d
   ```

3. 查看服务状态。

   ```shell
   docker compose ps
   ```

4. 查看 Nakama 实时日志。

   ```shell
   docker compose logs -f nakama
   ```

5. 查看自动落盘日志文件（宿主机）。

   ```shell
   tail -f ./logs/nakama.log
   ```

## 本地连接

`docker-compose.yml` 只把 Nakama HTTP / WebSocket 绑定到宿主机 loopback，供本机客户端或宿主机 nginx 访问：

| 入口 | 地址 | 说明 |
|---|---|---|
| Nakama HTTP API | `http://127.0.0.1:17350/v2/...` | 本机开发 / nginx upstream |
| Nakama WebSocket | `ws://127.0.0.1:17350/ws` | 本机开发 / nginx upstream |
| CockroachDB SQL | `localhost:26257` | 本地数据库调试 |
| CockroachDB Admin UI | `http://localhost:8080` | 本地数据库调试 |

Nakama gRPC `7349` 和 Console `7351` 不发布到宿主机端口。Console 默认账号 `admin` / `password123` 只适用于开发环境，不应作为生产公网入口。

## 生产入口

生产环境由宿主机 nginx 统一终止 HTTPS，并把 Paradice 放在 `/game/paradice/` 命名空间下：

| 用途 | 生产地址 | upstream |
|---|---|---|
| 游戏前端 | `https://bitaction.cn/game/paradice/` | `/var/www/paraweb/current` |
| HTTP API | `https://bitaction.cn/game/paradice/api/v2/...` | `127.0.0.1:17350`，转发为 Nakama 原生 `/v2/...` |
| WebSocket | `wss://bitaction.cn/game/paradice/api/ws` | `127.0.0.1:17350`，转发为 Nakama 原生 `/ws` |

`https://bitaction.cn/game/` 仅作为入口重定向，直接 `308` 到 `https://bitaction.cn/game/paradice/`。旧 `/v2/...` 与 `/ws` 不保留生产兼容 proxy；切换后不再作为 Paradice 入口。

生产不公网暴露 `7349`、`7350`、`7351`。后端发布由 GitHub Actions 上传 source archive 到 `/opt/paradiced/incoming/`，服务器通过受控 sudo wrapper `/usr/local/sbin/paradiced-deploy-archive` 调用 root-owned 固定部署实现 `/usr/local/lib/paradiced/deploy-archive.sh`，同步到固定实际目录 `/opt/paradiced/current`，再用固定 pluginbuilder / `docker compose up -d --no-deps --force-recreate nakama cron-cleanup` 流程激活，确保首次迁移也会更新 Docker bind mount 与端口发布。

## 停止与清理

```shell
docker compose down
```

不要在生产环境执行会删除数据卷的清理命令，除非已经完成备份并得到明确批准。