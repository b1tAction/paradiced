# ParaDiced 游戏后端

本项目后端由以下组件组成：
- CockroachDB（游戏数据存储）
- Nakama（网关与实时服务）
- Paradiced Go 插件（权威 Match 逻辑，Match 名称：paradiced_match）

## 前置要求

- Linux / macOS
- Docker（建议 24+）
- Docker Compose V2（命令是 docker compose）

说明：

- 不需要本机安装特定 Go 版本来构建插件。
- 插件构建已通过 Makefile 固定使用 heroiclabs/nakama-pluginbuilder:3.22.0。

## 启动流程

1) 构建 Paradiced 插件
   ```shell
    make build-plugin
   ```

2) 启动数据库和 Nakama
    ```shell
    docker compose up --build -d
    ```

3) 查看服务状态
    ```shell
    docker compose ps
    ```

4) 查看 Nakama 实时日志
    ```shell
    docker compose logs -f nakama
    ```

## 连接

- HTTP / WebSocket: http://localhost:7350
- gRPC: localhost:7349
- Nakama Console: http://localhost:7351
- CockroachDB SQL: localhost:26257
- CockroachDB Admin UI: http://localhost:8080

Nakama Console 默认账号（开发环境）：

- 用户名：admin
- 密码：password123

## 停止与清理

```shell
docker compose down
```