# r1rpc 开发进度

## 2026-08-27：图片识别平台架构与 GoFrame 决策

- 完成 `docs/image-recognition-architecture.md`，定义单图/批量识图、通用业务接口、任务队列、设备亲和、
  原始 JSON、对象存储、相似度、后台页面和实施阶段。
- 完成 `docs/development-standards.md`，明确 GoFrame 分层、DAO/DO/Entity 生成边界、文件不超过 500 行、
  中文注释、Trace、日志、安全、测试和提交规范。
- 核验本机已安装 `gf CLI v2.10.3`，但当前 `go.mod` 尚未引入 `github.com/gogf/gf/v2`；后续应按
  阶段正式接入，不能把 CLI 安装状态视为框架已集成。
- 决定使用 GoFrame 统一 HTTP、配置、日志、错误、校验、Trace 和 ORM/DAO 代码生成。
- 当前阶段继续使用 MySQL 完成感知哈希闭环；正式接入百万级视觉向量时，再评估迁移 PostgreSQL +
  pgvector。GoFrame 与数据库选型相互独立。
- 业务任务拟使用 Asynq + Redis，设备执行继续复用现有 Hub + Lease，不重复实现设备队列。
- 对象存储采用薄接口并优先评估 Go CDK Blob；图片本地处理后上传私有云存储，数据库保存对象档案。

### 下一阶段候选

1. 引入 GoFrame v2.10.x 和数据库驱动。
2. 建立 `manifest/config`、`hack/config.yaml` 及最小 `cmd/controller/service` 结构。
3. 先迁移配置、日志、Trace 和统一错误处理中间件，不改变设备协议。
4. 为新增图片领域设计首批 Schema，并通过 `gf gen dao` 生成 DAO/DO/Entity。
5. 完成单图和批量任务 Service 及状态机测试后，再接入后台页面和 Asynq。

## 2026-08-27：GoFrame 基础接入

- 引入 GoFrame v2.10.3 基础组件，保持与本机 gf CLI 版本一致。
- 新增适配现有 `net/http` 的 Trace 中间件，复用 `gctx/gtrace/glog`，不替换 WebSocket、静态资源和
  shutdown/drain 生命周期。
- 合法的 32 位十六进制 `X-Request-ID` 会注入 OpenTelemetry Trace Context；其他业务 requestId 继续
  独立保存，避免把任意字符串误当标准 Trace ID。
- 服务入口改用 GoFrame 日志，并删除包含管理员明文密码的启动日志。
- 新增无密钥 DAO 生成模板和 `make dao-image`；真实 DSN 只允许通过环境变量注入。
- 现有 `store.go/hub.go/http.go/app.go` 等历史文件超过 500 行，列为渐进拆分技术债；本阶段不混入
  高风险重构，新建人工维护文件严格遵守 500 行上限。
- 新增 `image_assets/image_batches/image_jobs` 三张首批真实表，并通过 `gf gen dao` 生成 DAO、DO、Entity；
  所有生成文件均保留 GoFrame 生成标记且未手工修改。
- 新增 GoFrame ORM 适配层，复用现有配置和 MySQL 实例；真实 `dbinit`、DAO 查询和事务写入通过。
- 新增与传输协议解耦的 `imagetask.Service`，支持单图和最多 100 张图片的批次创建、可选 externalId
  幂等、SHA-256 资产复用和事务回滚。
- 新增业务 API：`POST /api/v1/image-recognition/jobs`、`POST/GET /api/v1/image-recognition/batches`；
  新增服务端 Action `image.recognize`，创建异步业务任务而不直接下发设备。
- 新增后台“批量识图”页面，支持选择最多 100 张图片、逐张上传、创建批次和查看批次进度。
- 真实验收单图 API、批次幂等和 `image.recognize` 均返回 HTTP 202；数据库自动维护
  `created_at/updated_at`，空 externalId 使用 NULL，未出现空字符串唯一键冲突。

## 2026-09-19：图片组检索请求基础链路

- 新增 `image_search_requests`、`image_search_groups`、`image_search_items`，表达“请求包含多个组、组内任一图片命中即成功”的领域结构。
- 通过 `gf gen dao` 生成全部 DAO/DO/Entity；业务服务只使用生成 DAO、DO 和事务模型，不包含手写 SQL。
- API 定义放在 `api/image_search/v1`，通过 `gf gen ctrl` 生成接口与 Controller 骨架，并实现创建、详情、分页查询。
- 新增 `POST/GET /api/v1/image-search/requests` 管理接口；当前 `net/http` 仅保留薄适配层，后续切换 GoFrame HTTP 时无需改业务服务。
- 创建流程支持 `source + externalId` 幂等、图片资产复用、事务回滚、请求人记录和回调地址校验。
- 新增参数校验测试；`go test ./...`、`go vet ./...`、`go build ./...` 和 Controller 再生成检查均通过。

### 下一阶段候选

1. 为每个 `image_search_item` 创建或关联设备识图 Job，并按阶段进入 Asynq。
2. 实现组级“首个命中即成功”的原子状态更新，取消尚未执行的同组任务。
3. 保存候选结果与 pHash 评分明细，再接入后台任务列表和详情页。
4. 增加队列积压、阶段耗时、设备吞吐、匹配率和失败原因指标。

## 2026-09-19：图片组设备搜索队列编排

- 创建图片检索请求时，在同一数据库事务中为每个 `image_search_item` 创建并关联一个 `image_jobs` 设备任务。
- 复用现有 Asynq Enqueuer；数据库先落 `queued` 再投递 Redis，启动恢复和幂等重投以数据库状态为准。
- Worker 在上传、搜索、重试、失败和完成阶段同步更新 item、group、request 三级进度。
- 设备搜索完成使用 `search_completed`，与后续相似度判定的 `matched/not_matched` 明确分离。
- 重试次数耗尽时落失败状态，避免任务永久停留在 `retry_wait`。
- 详情 API 增加组级 queued/running/completed/failed 计数，为后台任务面板提供数据。
- 本地真实验证创建、Redis 入队、无设备重试、数据库关联和 externalId 幂等均符合预期。

## 2026-09-19：候选解析与感知哈希分析

- 新增 `image_search_responses`、`image_candidates`、`image_matches`，并通过 `gf gen dao` 生成 DAO/DO/Entity。
- 新增独立 Asynq `image:analyze:v1` 阶段；设备搜索完成后释放设备链路，再异步下载和分析候选。
- 解析 iOS 归一化候选的内容 ID、作者 ID、作者名、标题和封面 URL，保存可分页查询的候选记录。
- 使用 `goimagehash` 计算 pHash/dHash/aHash，以加权分数和最大 pHash 距离联合判定。
- 组级 `matchPolicy` 支持 `scoreThreshold` 和 `maxPHashDistance` 覆盖默认阈值。
- 实现 HTTPS、DNS/IP、重定向、大小、像素和解码限制，拒绝环回、私网、链路本地和超大图片。
- 组内首个匹配会原子确认最佳候选并取消待执行的同组任务；全部完成后汇总 request 的 matched/notMatched/failed 数量。
- 新增 `GET /api/v1/image-search/requests/{id}/candidates`，支持分页和 matched 过滤。
- 分析任务可在重启后恢复；候选全部下载失败会标记分析失败，不会错误判定为未匹配。

## 2026-09-19：图片搜索任务面板

- 新增后台“图片搜索”入口，支持一个请求内创建多个图片组，每组上传多张图片并关联业务用户 ID。
- 创建时可配置综合分数阈值与最大 pHash 距离，前端直接写入组级 `matchPolicy`。
- 新增任务列表与自动刷新，展示组数、命中、未命中、失败和当前状态。
- 新增任务详情页，按组展示原图、图片项状态、错误、候选封面、内容 ID、作者 ID、哈希距离和综合分数。
- 详情 API 增加 item/fileId/error 信息；候选评分不存在时不再伪装成距离 0。
- 生产前端资源已重新构建到 `internal/web/ui`；Go、TypeScript、Vite、vet 和 build 检查通过。

## 2026-09-19：XHS 候选下载诊断与重分析

- 从真实任务确认 XHS CDN 默认返回 `format/heif`，导致 Go 图片解码器报 `image: unknown format`。
- 下载前将 XHS HEIF/AVIF 派生参数转换为 JPEG，并显式声明 JPEG/PNG/WebP Accept 类型。
- 真实复用同一份已保存响应重新分析：20 条候选全部下载和计算哈希成功，失败数由 19 降为 0。
- 新增完整响应接口 `GET /api/v1/image-search/requests/{id}/responses`，面板可展开查看格式化 JSON。
- 候选面板展示真实下载/解码错误，不再只显示笼统的“候选图片下载失败”。
- 新增 `POST /api/v1/image-search/requests/{id}/retry-analysis`，无需再次调用设备即可复用保存的 JSON 重新分析。
