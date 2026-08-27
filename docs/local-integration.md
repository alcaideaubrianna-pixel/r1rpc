# Docker 本地联调

## 启动

在仓库根目录执行：

本地开发推荐让 Docker 只运行依赖，Go 服务由 `air` 热重载：

```bash
make dev
```

该命令使用本机已有的 `mysql:8.4`、`redis:7-alpine` 镜像启动依赖，然后在宿主机启动 Go 服务。
修改 `.go`、配置或内嵌网页资源后，`air` 会自动重新编译并重启服务。首次使用若未安装 Air：

```bash
go install github.com/air-verse/air@latest
```

只启动环境依赖：

```bash
make dev-deps
```

服务默认监听 `http://127.0.0.1:9876`，健康检查为 `http://127.0.0.1:9876/healthz`。

图片搜索调试流程：先在“文件管理”上传图片，再到“RPC 调用”选择
`media.upload_image` 并选择图片。页面会把本地 `fileId` 写入 Payload，服务端下发设备前自动转换为
现有 base64 图片协议；调用记录中只保存 `fileId`，不保存 base64。上传目录默认为
`./data/uploads`，单张图片最大 12 MiB。

`content.search_by_image` 也支持直接选择图片。没有 `uploadHandle` 时，管理端会自动先调用
`media.upload_image`，严格读取设备返回的 `uploadHandle` 后再执行图片搜索；已有 handle 时则直接搜索。
上传和搜索阶段使用独立 Trace ID。图片搜索记录可保留 `image.fileId` 作为审计上下文，但该辅助字段
会在下发设备前移除，不改变设备既有协议。

需要把 Go 服务也运行在 Docker 中时：

```bash
make dev-full
```

默认会启动 r1rpc、MySQL 和 Redis，并创建持久化 volume。仅用于本地联调的默认后台账号是 `admin / 123456`。端口冲突时可覆盖宿主机端口，例如：

```bash
R1RPC_HTTP_PORT=19876 R1RPC_MYSQL_PORT=13306 R1RPC_REDIS_PORT=16379 make dev-full
```

镜像默认复用本机的 `mysql:8.4` 和 `redis:7-alpine`，也可显式覆盖：

```bash
R1RPC_MYSQL_IMAGE=mysql:8.0 R1RPC_REDIS_IMAGE=redis:8.8.1 make dev-deps
```

## 地址

- 管理页与 HTTP 基地址：`http://127.0.0.1:9876`
- 健康检查：`http://127.0.0.1:9876/healthz`
- 设备登录：`POST http://127.0.0.1:9876/api/client/login`
- WebSocket：登录响应中的 `wsUrl`，路径为 `ws://127.0.0.1:9876/api/client/ws?token=...`
- RPC 调用：`POST http://127.0.0.1:9876/rpc/{group}/{action}`

真机不能使用 `127.0.0.1` 访问电脑。请确保手机和电脑在同一局域网，并把地址中的主机名改成电脑的局域网 IP，例如 `http://192.168.1.20:9876` 和 `ws://192.168.1.20:9876/...`。macOS 常用 `ipconfig getifaddr en0` 查看 Wi-Fi 地址；Linux 可用 `hostname -I`。同时确认系统防火墙允许该端口入站。

## 检查与日志

热重载模式的 Go 日志直接显示在运行 `make dev` 的终端。依赖状态和日志：

```bash
make dev-status
curl -fsS http://127.0.0.1:9876/healthz
docker compose -f deploy/docker-compose.yml exec mysql sh -c 'mysqladmin ping -h 127.0.0.1 -uroot -p"$MYSQL_ROOT_PASSWORD"'
docker compose -f deploy/docker-compose.yml exec redis redis-cli ping
```

服务日志写到容器标准输出和标准错误，由 Docker 管理；按服务查看：

```bash
docker compose -f deploy/docker-compose.yml logs -f server
docker compose -f deploy/docker-compose.yml logs -f mysql redis
```

热重载服务用 `Ctrl-C` 停止。停止容器并保留 MySQL、Redis 数据：

```bash
make dev-stop
```

需要同时删除本地数据库和 Redis volume 时执行 `make dev-clean`。
