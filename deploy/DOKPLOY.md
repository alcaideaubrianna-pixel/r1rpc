# Dokploy 部署

## 创建 Compose 服务

1. 在 Dokploy 创建 Project 和 Compose 服务，选择 Git Provider，并绑定本仓库及生产分支。
2. Compose Path 填写 `deploy/dokploy-compose.yml`。
3. 将 `.env.dokploy.example` 中的变量录入 Dokploy Environment，所有 `replace-with-*` 必须替换为随机强密码。
4. 在 `server` 服务上绑定域名，容器端口填写 `9876`，并启用 HTTPS。
5. 首次点击 Deploy。MySQL、Redis 和上传文件分别使用具名卷，重新构建应用不会丢失数据。

运行日志默认写入持久化卷 `/app/data/logs`，目录总量限制为 200 MiB。可在 Dokploy Environment 中使用 `LOG_DIR` 和 `LOG_MAX_DIR_MB` 覆盖；日志按服务启动时间、日期和分片号区分。

## Git 自动部署

在 Compose 服务的 Deployments 设置中启用 Auto Deploy。使用 GitHub/GitLab App 时由 Dokploy 自动管理 webhook；使用普通 Git 仓库时，按 Dokploy 页面提示把部署 webhook 添加到仓库。之后生产分支每次 push 都会重新执行 Docker 构建并滚动更新 `server`。

镜像构建会执行：

```text
npm ci && npm run build
go build ./cmd/server
```

因此前端和后端都会以当前提交重新构建，不依赖仓库里的旧前端产物。

## 上线检查

- 确认 `server`、`mysql`、`redis` 均为 healthy。
- 打开 `/healthz`，确认返回成功。
- 首次登录后立即修改管理员密码。
- 定期备份 `mysql_data` 和 `uploads_data`；Redis AOF 用于队列恢复，但不能替代 MySQL 备份。
- 排障时优先查看 `logs_data` 中的运行日志；容器 stdout 同时保留最近 5 个 20 MiB 分片。
- 不要把 MySQL、Redis 的端口映射到公网。
