# 图片识别任务平台架构设计

> 状态：设计阶段
> 版本：v0.1
> 日期：2026-08-27
> 适用项目：r1rpc

## 1. 背景与目标

r1rpc 当前已经具备设备登录、WebSocket 通信、Action 能力声明、设备并发限制、Lease 队列、
`media.upload_image` 和 `content.search_by_image` 等基础能力。本阶段在此基础上建设通用图片识别任务平台，
使管理后台、Telegram Bot 和第三方 API 使用同一套业务服务，而不是分别实现上传、调度和识图流程。

首期目标：

1. 支持单张和最多 100 张图片创建识图任务。
2. 批量任务拆分为独立子任务，并分配到具备能力的在线设备。
3. 保证设备上传和图片搜索使用同一设备及有效会话。
4. 逐字节保存设备返回的原始 JSON，并备份到对象存储。
5. 下载首页候选图片，在本地处理后上传对象存储并建立数据库档案。
6. 使用 SHA-256、pHash、dHash、aHash 完成首期去重和相似度排序。
7. 历史结果满足算法版本和置信度要求时直接命中缓存。
8. 为后续图片向量、人脸特征和人体特征预留可演进的数据边界。
9. 提供后台批量任务页面、单图识图 Action 和稳定的第三方业务 API。

首期不实现人脸识别、人体 ReID、CLIP/DINO 推理、图片搜索分页和专用向量数据库。

## 2. 技术决策

### 2.1 采用 GoFrame

项目采用 GoFrame v2，当前本机 CLI 版本为 `v2.10.3`。GoFrame 负责：

- HTTP Server、路由、中间件和参数校验；
- 配置加载和环境变量覆盖；
- `glog` 结构化日志和 Context Trace ID；
- `gerror` 错误堆栈；
- `gdb` ORM、事务和 SQL 日志；
- `gf gen dao` 生成 DAO、DO、Entity；
- OpenAPI 接口描述；
- 服务生命周期和优雅退出的基础管理。

GoFrame 不负责替代：

- Redis 持久化业务任务队列；
- 现有设备 Hub、Lease 和会话调度；
- 云对象存储；
- 图片指纹和视觉模型；
- PostgreSQL/pgvector 或专用向量数据库。

### 2.2 数据库阶段决策

GoFrame 与数据库选型是两个不同层次。GoFrame ORM 同时支持 MySQL 和 PostgreSQL，但不会让 MySQL
获得原生向量索引能力。

当前阶段继续使用 MySQL，原因是：

- 现有控制面表和 SQL 已在 MySQL 上运行；
- 首期只需要 SHA-256 和感知哈希，不依赖向量近邻索引；
- 先完成业务闭环比立即迁移数据库风险更低；
- GoFrame DAO 可以减少当前手写 SQL，且保留未来切换数据库的空间。

满足以下任一条件时启动 PostgreSQL + pgvector 迁移评估：

- 正式接入 CLIP、DINO、人脸或人体 Embedding；
- 向量数量接近或超过 100 万；
- 需要 Top-K 向量检索及 HNSW/IVFFlat 索引；
- MySQL 应用层候选过滤无法满足延迟和召回率要求。

届时优先评估全库迁移到 PostgreSQL，避免长期双数据库。达到数千万向量或高并发瓶颈后，再评估
Qdrant、Milvus 等专用向量库。

### 2.3 任务队列

队列分为两层：

```text
Asynq + Redis：业务任务编排、重试、延迟执行、优先级
现有 Hub + Lease：设备选择、WebSocket 下发、执行租约、结果回收
```

Asynq Task Payload 只保存 `jobId` 和阶段，不保存图片、base64、原始 JSON 或候选列表。Worker 必须从
数据库重新读取权威状态，以保证任务可重试和幂等。

### 2.4 对象存储

采用薄 `ObjectStore` 接口，基础实现优先使用 Go CDK Blob。开发环境使用本地文件 backend，生产环境
使用私有 S3 兼容对象存储。所有图片先在本地工作目录解码、校验和计算特征，完成后上传云端备份。

数据库只保存对象 Key、内容哈希、大小、类型和状态，不保存图片二进制或 base64。

## 3. 总体架构

```text
┌───────────────────────────────────────────────┐
│ 接入层                                        │
│ 管理后台 │ 业务 HTTP API │ Telegram │ Webhook │
└──────────────────────┬────────────────────────┘
                       │ 调用统一 Service
┌──────────────────────▼────────────────────────┐
│ GoFrame Controller                            │
│ 参数绑定、校验、鉴权、响应转换，不编排工作流   │
└──────────────────────┬────────────────────────┘
                       │ Input / Output
┌──────────────────────▼────────────────────────┐
│ Service                                       │
│ Job │ Batch │ Asset │ Search │ Analysis       │
│ 状态机、幂等、缓存、设备亲和、事务边界          │
└───────────────┬────────────────┬──────────────┘
                │                │
       ┌────────▼────────┐ ┌────▼────────────────┐
       │ GoFrame DAO/ORM │ │ Infrastructure Port │
       │ MySQL（首期）    │ │ Queue/Blob/Device   │
       └─────────────────┘ └─────────────────────┘
```

依赖方向必须保持：

```text
Controller → Service → DAO / Port 实现
```

Telegram、后台和第三方 API 不得直接操作 DAO、Asynq 或设备 Hub。

## 4. GoFrame 工程目录

采用 GoFrame 推荐的三层结构，并保留现有设备协议模块。迁移后目标目录：

```text
api/
├── image_job/v1/
├── image_batch/v1/
└── action/v1/

internal/
├── cmd/
│   ├── server.go
│   └── worker.go
├── controller/
│   ├── image_job/
│   ├── image_batch/
│   └── action/
├── service/
│   ├── imagejob/
│   ├── imagebatch/
│   ├── imageasset/
│   ├── imagesearch/
│   ├── imageanalysis/
│   └── objectarchive/
├── model/
│   ├── do/                 # gf gen dao 生成，禁止手改
│   ├── entity/             # gf gen dao 生成，禁止手改
│   ├── input/              # Service 输入
│   └── output/             # Service 输出
├── dao/                    # gf gen dao 生成的基础 DAO
├── consts/
├── middleware/
├── taskqueue/
├── objectstore/
├── imaging/
├── devicegateway/
├── rpc/                    # 保留现有 Hub/Lease
└── transport/              # 保留设备 WebSocket 协议

manifest/config/
resource/migrations/
hack/config.yaml            # gf CLI 生成配置
web/src/features/image-task/
```

不为追求形式进行一次性目录搬迁。GoFrame 接入按配置日志、HTTP、ORM/DAO、业务模块顺序渐进实施，每个
阶段均保持系统可运行和可回滚。

## 5. 通用业务接口

Service 层使用内部 Input/Output，不接收 HTTP Req，也不返回 HTTP Res：

```go
type CreateJobInput struct {
    Source       string
    ExternalID   string
    FileID       string
    Priority     int
    ForceRefresh bool
    Callback     *CallbackInput
    RequestedBy  *RequesterInput
}

type CreateBatchInput struct {
    Source       string
    ExternalID   string
    FileIDs      []string
    Priority     int
    ForceRefresh bool
    Callback     *CallbackInput
    RequestedBy  *RequesterInput
}
```

接入来源统一枚举：

```text
admin / api / telegram / webhook / internal
```

外部来源通过 `source + external_id` 唯一约束实现幂等创建。

## 6. 对外接口

### 6.1 单图任务

```text
POST /api/v1/image-recognition/jobs
GET  /api/v1/image-recognition/jobs/{id}
POST /api/v1/image-recognition/jobs/{id}/cancel
POST /api/v1/image-recognition/jobs/{id}/retry
```

支持上传文件或传入已有 `fileId`，两种形式最终转换为同一个 `CreateJobInput`。

### 6.2 批量任务

```text
POST /api/v1/image-recognition/batches
GET  /api/v1/image-recognition/batches
GET  /api/v1/image-recognition/batches/{id}
GET  /api/v1/image-recognition/batches/{id}/jobs
POST /api/v1/image-recognition/batches/{id}/cancel
```

单次最多 100 张，限制必须同时存在于 API 校验、Service 校验和配置中。

### 6.3 单图识图 Action

新增服务端业务 Action：

```text
image.recognize
```

Payload：

```json
{
  "image": {"fileId": ""},
  "forceRefresh": false,
  "priority": 0
}
```

该 Action 创建持久化业务任务，不直接下发设备。设备 Action 保持：

```text
media.upload_image
content.search_by_image
```

## 7. 后台页面

新增：

```text
/#/image-jobs
/#/image-batches
/#/image-batches/{id}
```

批量创建页面支持拖拽或选择 1～100 张图片、逐文件上传进度、失败重试和创建批次。批次列表显示总数、
排队、执行中、完成、失败和缓存命中数量。详情页展示每个 Job 的原图、状态、阶段、设备、Trace ID、
候选数、最高相似度、最佳候选图及错误代码。

前端只负责上传和调用批次 API，不在浏览器中串联设备上传和图片搜索。

## 8. 状态机与幂等

Job 状态：

```text
created → queued → preparing → waiting_device
→ uploading_to_device → searching → response_received
→ downloading_candidates → analyzing → archiving → completed
```

旁路状态：

```text
cache_hit / retry_wait / failed / cancelled
```

关键幂等约束：

- `image_assets.sha256` 唯一；
- `source + external_id` 唯一；
- `job_id + request_id` 原始响应唯一；
- `job_id + rank` 候选唯一；
- `asset_id + algorithm + version` 特征唯一；
- Worker 每次执行前检查数据库阶段，不依赖 Redis 投递次数判断状态。

## 9. 设备亲和

`uploadHandle` 只在生成它的设备进程和会话中有效。业务 Job 必须保存：

```text
assigned_client_id
assigned_session_incarnation
upload_request_id
search_request_id
upload_handle
```

上传完成后，图片搜索必须指定同一设备和会话。会话变化或 Handle 失效时，从设备上传阶段重新执行，
不能把设备 A 的 Handle 下发给设备 B。

## 10. 文件和对象存储

处理顺序：

```text
写本地临时文件 → magic bytes/大小/解码校验 → SHA-256
→ 计算基础特征 → 上传对象存储 → 校验对象 → DB 标记 available
```

内容寻址 Key：

```text
assets/original/{sha256前2位}/{sha256}.{ext}
assets/candidate/{sha256前2位}/{sha256}.{ext}
assets/thumb/{sha256前2位}/{sha256}.webp
responses/{yyyy}/{mm}/{jobId}/{requestId}.json
```

对象状态：

```text
local_pending / uploading / available / upload_failed / deleting / deleted
```

数据库和对象存储没有跨系统事务，因此使用状态机和对账任务，不伪造原子提交。

## 11. 原始 JSON

设备返回内容必须在任何归一化之前获取并逐字节保存：

- 数据库 `raw_json` 使用 TEXT/LONGTEXT；
- 保存 `raw_sha256` 和 `raw_size_bytes`；
- 同一原始内容以 `.json` 上传对象存储；
- 可额外保存解析后的 JSON 字段，但不能替代原文；
- 不在普通日志打印响应体。

如果现有设备协议只返回归一化 JSON，则需要先扩展设备响应，显式增加原始 body 字段或原始附件，不能
把重新序列化后的对象标记为“逐字节原始响应”。

## 12. 图片分析与缓存

首期流程：

```text
SHA-256 精确查重
→ pHash/dHash/aHash 历史候选过滤
→ 下载搜索首页候选
→ 候选特征计算
→ 多指标评分
→ 保存全部评分和最佳候选
```

推荐使用 `github.com/corona10/goimagehash`，所有算法记录版本号。pHash 不能可靠解决大幅裁剪、水印、
人体姿态和人脸身份问题，因此只用于首期粗筛，不作为最终通用视觉判断。

缓存直接返回必须同时满足：

- SHA-256 完全相同，或感知哈希距离低于配置阈值；
- 历史任务为 completed；
- 算法版本一致；
- 最终置信度达到阈值；
- 结果未过期且对象仍可用。

## 13. 数据表

首期核心表：

```text
image_assets
image_batches
image_jobs
image_job_events
image_search_responses
image_candidates
image_features
image_matches
image_callbacks
```

图片二进制不入库。原始 JSON 保留原文，候选条目可同时保存 `raw_item_json` 以便重新解析。

GoFrame `gf gen dao` 生成：

```text
internal/dao
internal/model/do
internal/model/entity
```

生成文件禁止手工修改。业务规则放在 Service，不写进生成 DAO。

## 14. Trace 与日志

统一使用 OpenTelemetry Trace ID，并把现有 `X-Request-ID` 兼容映射为 Trace ID。任务还需独立保存：

```text
batch_id / job_id / request_id / trace_id
```

使用 `glog` 和 Context 输出结构化日志，推荐字段：

```text
event, trace_id, batch_id, job_id, request_id, client_id,
stage, status, attempt, duration_ms, error_code
```

禁止记录图片 base64、Token、Cookie、签名 URL、完整原始 JSON和生物特征向量。

## 15. 安全与资源限制

- 图片最大 12 MiB，批量默认最多 100 张；
- 外部下载只允许 HTTPS 和受控域名；
- 每次重定向重新校验目标 IP，拒绝内网、环回和链路本地地址；
- 限制下载大小、超时、重定向次数、像素总量和解码内存；
- 对象存储默认私有，访问使用短期签名 URL；
- Telegram 和第三方 API 使用白名单、速率限制和幂等键；
- 人脸及人体 Embedding 按敏感数据处理，不输出到日志。

## 16. 实施阶段

1. **GoFrame 基础接入**：依赖、配置、日志、Trace、错误处理中间件，不改变业务语义。
2. **DAO 试点**：对新图片领域建表并运行 `gf gen dao`，不立即重写全部旧 Store。
3. **通用任务服务**：Job/Batch Service、状态机、幂等和业务 API。
4. **任务队列**：Asynq Worker 与现有设备 Hub 对接。
5. **对象存储**：本地处理、云端归档、数据库状态和对账任务。
6. **后台页面**：批量创建、列表、详情、取消和重试。
7. **图片分析**：候选下载、感知哈希、评分和缓存。
8. **外部接入**：Telegram Bot、第三方接口和回调。
9. **向量阶段**：基准测试后决定 PostgreSQL + pgvector 或专用向量库。

每个阶段完成测试和文档后再进入下一阶段，不一次性重写运行中的设备协议和队列。
