# r1rpc Agent 开发交接文档

> 后续 Agent 开始前必须阅读本文、`docs/development-standards.md`、`docs/image-recognition-architecture.md`，然后检查 `git status`。更新时间：2026-08-27。

## 1. 项目目标

r1rpc 是面向 iOS 设备集群的 Action/RPC 调度服务。最终目标是建设通用图片识别任务平台：

```text
单图/批量上传 → 持久化任务 → Redis队列 → 设备上传/搜索
→ 保存原始响应 → 下载候选 → 本地特征分析 → 云归档 → 缓存或最佳匹配
```

后台、HTTP、Telegram Bot 和第三方接口必须共享同一套 Service，不能分别复制工作流。

## 2. 当前代码和提交

工作目录：`/Users/ley/Desktop/CodeProject/r1rpc`

重要提交：

```text
6edc717 r1rpc：完善图片上传与图片搜索自动串联
654cab7 阶段存档：完善设备集群与存储基础能力
```

当前工作区有未提交的 GoFrame、DAO、图片任务、批量页面和 Asynq 改动。不要覆盖或回滚。

不要提交：`data/`、`.spec-workflow/`、日志、响应样本、Token、Cookie、密钥、签名URL、Hopper工程和调试产物。

## 3. 已完成能力

### 3.1 设备和 RPC

- iOS 使用单一设备协议，自动生成并持久化纯小写 UUID。
- 服务端固定自动归入 `XHS` group。
- Hub 支持 capability/action 调度、设备亲和、并发限制、Lease、断线重排、迟到结果幂等和 session incarnation。
- `media.upload_image` 使用本地 `fileId`，调用前物化为设备既有 base64 图片协议。
- `content.search_by_image` 支持已有 `uploadHandle`，后台也支持自动先上传再搜索。
- `content.search` 和 `content.search_notes` 已完成原生文本搜索。

### 3.2 文件服务

代码位于 `internal/files/`，支持 JPEG、PNG、HEIC、HEIF magic bytes 校验，最大 12 MiB，临时文件原子 rename。

```text
POST   /api/files
GET    /api/files
GET    /api/files/{id}/content
DELETE /api/files/{id}
```

文件 ID 是纯 32 位 hex，与 `files.id CHAR(32)` 兼容。

### 3.3 GoFrame基础接入

已引入：

```text
github.com/gogf/gf/v2 v2.10.3
github.com/gogf/gf/contrib/drivers/mysql/v2 v2.10.3
gf CLI v2.10.3
```

已完成：`internal/observability` Trace middleware、`glog` 结构化日志、`internal/persistence/goframe.go` ORM注册、GoFrame DAO生成，以及不改变现有 WebSocket/embed/shutdown 的渐进适配。

当前仍是 `net/http.Server + GoFrame适配层`，不是完整 `ghttp.Server`，不要直接全量替换。

## 4. 图片任务与队列

已新增真实表：`image_assets`、`image_batches`、`image_jobs`。GoFrame生成目录为 `internal/dao/`、`internal/model/do/`、`internal/model/entity/`，生成文件禁止手工修改。

`internal/service/imagetask/` 支持单图、1～100张批量任务、`source + externalId` 幂等、SHA-256资产复用、事务和批次分页列表。

API：

```text
POST /api/v1/image-recognition/jobs
POST /api/v1/image-recognition/batches
GET  /api/v1/image-recognition/batches
```

`image.recognize` 是服务端业务Action，只创建任务，不直接下发设备。后台页面为 `/#/image-batches`。

已引入 Asynq v0.25.1，代码位于 `internal/taskqueue/`：

```text
Asynq + Redis：业务任务编排和重试
现有 Hub + Lease：设备选择、WebSocket下发和设备执行租约
```

Asynq payload 只有 `{"jobId":"..."}`。Worker流程为：读取任务 → 设备上传 → 保存实际clientId和uploadHandle → 同设备图片搜索 → 保存 `normalized_response_json` → completed。

设备RPC requestId必须使用 `jobId-u-attempt` 和 `jobId-s-attempt`，不能固定使用 `jobId-upload`，否则重试会撞 `rpc_requests.request_id` 唯一键。

## 5. 重要未完成边界

- 当前 `normalized_response_json` 不是逐字节原始网络JSON。必须先修改iOS协议，在网络response边界传输 `rawPayload/rawSize/rawSha256/encoding`，不能改名伪装。
- 尚未完成候选下载、`image_search_responses`、`image_candidates`、对象存储、pHash、相似度融合和缓存命中。
- 尚未完成Telegram Bot、第三方API、人脸、人体和Embedding。
- 批次取消、重试、死信和完整计数迁移仍需完善。

曾发生风控风险：Worker启动自动恢复历史任务，短时间调用真实小红书。后续真机必须并发1、默认关闭历史自动恢复、增加冷却间隔，队列测试优先使用Mock设备。

## 6. PostgreSQL切换计划

用户已明确要求移除MySQL，切换PostgreSQL，并为百万级图片向量预留pgvector。GoFrame是应用框架，不是数据库。

顺序：

1. 冻结真实Worker，默认并发改为1，清理历史测试任务。
2. Compose加入PostgreSQL和pgvector。
3. 创建 `resource/migrations/` 版本化迁移，不再扩展MySQL `schema.sql`。
4. 转换 `AUTO_INCREMENT/TINYINT/ENUM/DATETIME/LONGTEXT` 为PostgreSQL语义。
5. 引入 `github.com/gogf/gf/contrib/drivers/pgsql/v2`，删除MySQL driver。
6. 用PostgreSQL真实表重新执行 `gf gen dao`。
7. 编写MySQL→PostgreSQL一次性迁移并校验行数、字段哈希、NULL、时间和原始JSON字节。
8. 切换server、dbinit、Makefile和Compose，删除MySQL运行时依赖。
9. 稳定后再增加 `image_embeddings/face_embeddings/person_embeddings` 和HNSW索引。

原始响应目标字段：`raw_json TEXT`、`raw_sha256`、`raw_size_bytes`、`raw_object_key`。不要只保存JSONB。

## 7. 对象存储和算法

推荐Go CDK Blob。流程必须是：

```text
本地临时文件 → 解码/安全校验 → SHA和特征 → Blob上传
→ Writer.Close成功 → Attributes校验 → DB archived → 本地清理
```

fileblob使用 `CreateDir:true, NoTempDir:true` 避免跨文件系统rename。生产S3兼容服务必须实测TLS、path-style、metadata、条件写、HEAD、GET、删除和大对象。不要把ETag当内容MD5。

首期算法为 `SHA-256 + pHash + dHash + aHash + 多版本归一化`。后续用独立Vision Worker接入CLIP/DINO、人脸和人体Embedding。

## 8. GoFrame开发规范

完整规范：`docs/development-standards.md`；技能：`/Users/ley/.agents/skills/goframe-v2/SKILL.md`。

1. 业务逻辑放 `internal/service/`；Controller只负责绑定、校验、鉴权和响应转换。
2. DAO/DO/Entity由 `gf gen dao` 生成，生成目录禁止手改。
3. 数据库写操作必须使用DO，禁止用 `g.Map` 或普通map作为Data。
4. 使用 `gerror`、`glog`、`gctx/gtrace`；日志禁止图片、base64、Token、Cookie、签名URL、完整原始JSON和向量。
5. Context必须贯穿数据库、队列、设备、下载和对象存储调用。
6. 人工维护Go、TS、TSX文件不超过500行，关键逻辑添加中文注释。
7. 事务中不能执行设备RPC、图片下载、云存储或模型推理。
8. 队列只保存ID和阶段，Worker必须幂等、有限重试并区分错误类型。
9. 第三方API先查官方文档、版本和License，禁止臆测。

## 9. 下一Agent执行顺序

### 第一阶段：切换PostgreSQL

1. 冻结真实设备Worker，默认并发1，关闭历史任务自动恢复。
2. Compose加入PostgreSQL+pgvector健康检查。
3. 创建PostgreSQL版本化迁移，覆盖现有全部表。
4. 切换GoFrame PostgreSQL driver并重新生成DAO。
5. 编写一次性数据迁移并校验行数、NULL、时间和原始字节。
6. 删除MySQL配置、driver、Compose服务和专属bootstrap。
7. 回归管理员、设备登录、WebSocket、RPC、文件和任务创建。

### 第二阶段：队列安全

1. 完善CAS状态、批次聚合、取消、重试、死信和恢复。
2. 增加Mock DeviceInvoker，禁止用真机做重试测试。
3. 增加设备冷却、最小间隔和疑似风控停止策略。

### 第三阶段：原始响应和云归档

1. 修改iOS协议传输真正raw bytes。
2. 新增响应、候选、特征和匹配表。
3. 引入Go CDK Blob、对象状态机、校验、对账和清理。

### 第四阶段：候选分析和外部接入

1. SSRF安全下载首页20张候选图。
2. 实现SHA/pHash/dHash/aHash、相似度和缓存。
3. 实现Telegram Bot和第三方批量API，统一复用Service。

## 10. 验证命令

```bash
gofmt -w <changed-go-files>
go test ./...
go test -race ./...
go vet ./...
npm run build --prefix web
docker compose -f deploy/docker-compose.yml config
git diff --check
```

当前本地服务：HTTP `127.0.0.1:9876`、MySQL `127.0.0.1:3306`（待迁移）、Redis `127.0.0.1:6379`。

当前不能声称已完成：PostgreSQL切换、云存储、逐字节原始JSON、候选下载、pHash、Embedding、Telegram。

## 附录A：图片任务实现明细

真实表：

```text
image_assets
image_batches
image_jobs
```

GoFrame生成目录：

```text
internal/dao/
internal/dao/internal/
internal/model/do/
internal/model/entity/
```

生成命令：

```bash
R1RPC_DAO_DSN='mysql:root:密码@tcp(127.0.0.1:3306)/r1rpc' make dao-image
```

生成文件禁止手工修改，数据库写操作必须使用生成的 DO。

`internal/service/imagetask/` 已支持：

- 单图任务；
- 1～100 张批量任务；
- `source + externalId` 幂等；
- SHA-256 资产复用；
- GoFrame事务；
- 批次分页列表；
- 空 externalId 使用 NULL。

API：

```text
POST /api/v1/image-recognition/jobs
POST /api/v1/image-recognition/batches
GET  /api/v1/image-recognition/batches
```

业务 Action：`image.recognize`。它创建任务，不直接下发设备。

后台页面：`/#/image-batches`，支持最多100张图片逐张上传和批次摘要查看。

## 附录B：Asynq与真实设备验证明细

已引入：`github.com/hibiken/asynq v0.25.1`。

代码位于 `internal/taskqueue/`：

```text
asynq.go       Producer，仅保存 jobId
runtime.go     启停、恢复和handler注册
worker.go      上传→搜索→任务状态更新
```

任务类型：

```text
image:process:v1
```

Redis payload只有：

```json
{"jobId":"..."}
```

当前流程：

```text
读取image_job/image_asset
→ CAS标记running/uploading
→ media.upload_image
→ 保存实际clientId和uploadHandle
→ 同一clientId执行content.search_by_image
→ 保存normalized_response_json
→ 标记completed
```

重试的设备RPC ID必须使用：

```text
jobId-u-attempt
jobId-s-attempt
```

不能固定使用 `jobId-upload`，否则会与 `rpc_requests.request_id` 唯一键冲突。

当前已真实验证：3个任务完成 `media.upload_image` 和 `content.search_by_image`，且上传、搜索使用同一个设备。

## 附录C：已知问题详细记录

### C.1 风控

曾因Worker启动时自动恢复历史 `created/queued/retry_wait` 任务，短时间调用真实小红书，可能触发平台风控。后续必须：

- 真机并发固定为1；
- 默认关闭自动恢复历史任务；
- 队列和重试优先使用Mock设备；
- 不用真实设备做批量回归；
- 增加设备冷却和最小调用间隔；
- 清理/标记旧测试任务后再启动Worker。

当前配置中的 `worker_concurrency` 仍为4，下一 Agent必须在真实设备消费前改成1。

### C.2 原始JSON

当前iOS流程虽然在网络边界捕获过response body，但返回给r1rpc的是归一化结果。数据库字段 `normalized_response_json` 不能改名伪装成原始JSON。

真正的逐字节原始JSON需要修改iOS协议，在网络response边界直接传输：

```text
rawPayload / rawSize / rawSha256 / encoding
```

服务端再同时保存原文和归一化结果。普通日志禁止输出原始body。

### C.3 尚未完成

```text
候选图片下载
image_search_responses表
image_candidates表
对象存储归档
pHash/dHash/aHash实际计算
相似度融合和缓存命中
批次取消/重试/完整聚合
Telegram Bot
第三方接口
人脸/人体/Embedding
PostgreSQL切换
```
