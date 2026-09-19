# r1rpc 项目开发规范

> 版本：v0.1
> 日期：2026-08-27
> 技术基线：Go 1.26、GoFrame v2.10.x、React + TypeScript

## 1. 基本原则

1. 优先使用 GoFrame 和成熟开源库，不重复实现配置、日志、ORM、校验、链路、队列和对象存储基础设施。
2. 引入任何 API/SDK 前必须核验官方文档、版本、License、Go 版本和维护状态。
3. 业务接口与 HTTP、Telegram、队列中间件和数据库实现解耦。
4. 不为未来需求提前创建大量空接口；只抽象当前已有两个及以上实现或明确的系统边界。
5. 不回滚工作区中其他人的未提交改动，不提交运行数据、密钥、日志和调试样本。
6. 技术说明、关键注释和提交信息使用中文。

## 2. GoFrame 使用规范

### 2.1 组件优先级

新代码优先使用：

```text
配置：gcfg / g.Cfg
日志：glog / g.Log
错误：gerror
HTTP：ghttp
校验：gvalid 和 Req 标签
数据库：gdb + gf gen dao
Context/Trace：gctx + gtrace
JSON：gjson（需要动态 JSON 时）
```

GoFrame 不具备的能力再引入专用依赖，例如 Asynq、Go CDK Blob 和图片算法库。

### 2.2 代码生成

`gf gen dao` 生成的以下目录禁止手工修改：

```text
internal/model/do
internal/model/entity
internal/dao/internal
```

DAO 外层文件只有确实需要通用扩展时才修改。业务逻辑不能放在生成文件或 DAO 中。

生成配置统一保存在：

```text
hack/config.yaml
```

禁止把生产数据库密码写入生成配置；连接信息通过环境变量或本地未跟踪配置注入。

### 2.3 DO 使用

数据库写操作必须使用 `internal/model/do` 对象，不使用 `g.Map` 或
`map[string]interface{}` 拼接 Data：

```go
dao.ImageJobs.Ctx(ctx).Data(do.ImageJobs{
    Status: status,
}).Where(dao.ImageJobs.Columns().Id, id).Update()
```

未设置的 DO 字段保持 nil，由 ORM 忽略。需要显式写 NULL 时使用框架支持的 Raw 表达式。

### 2.4 时间与软删除

存在 `created_at`、`updated_at`、`deleted_at` 时使用 GoFrame 自动时间维护：

- 不手工写入 created_at/updated_at；
- 不手工追加 `deleted_at IS NULL`；
- 使用 `Delete()` 执行软删除；
- 审计和原始响应等不可软删除的数据表应在表设计中明确生命周期策略。

## 3. 分层职责

### API

- 只定义外部 Req/Res 和校验标签；
- 按版本组织，例如 `api/image_batch/v1`；
- 不暴露内部 Entity、DO 或数据库字段细节。

### Controller

- 参数接收、鉴权、输入转换和响应转换；
- 不直接访问 DAO；
- 不直接投递 Asynq；
- 不编排设备上传和搜索；
- 单个方法应保持轻量。

### Service

- 承载可复用业务逻辑、状态机、事务和幂等；
- 输入输出使用内部 Input/Output；
- 不依赖 HTTP Req/Res；
- 管理本业务模块 DAO，不随意访问其他模块内部实现；
- Telegram、后台和第三方 API 必须调用同一 Service。

### DAO

- 只负责通用数据访问；
- 不包含任务状态流转、缓存判定或设备选择规则；
- 优先使用生成 DAO 的链式能力，不重复包装简单 CRUD。

### Infrastructure

- 队列、对象存储、设备网关、下载器和图片算法实现放在独立包；
- Service 依赖窄接口；
- 中间件类型不能泄漏到业务模型。

## 4. 文件与包规范

1. 人工维护的 Go、TS、TSX 文件不得超过 500 行。
2. 自动生成文件不受 500 行限制，但必须带生成标记且禁止手改。
3. React 页面建议不超过 400 行；复杂功能拆到 `features/<domain>/components`。
4. 一个文件只承担一个主要职责；Handler、Service、Repository、Worker 按场景拆分。
5. 包名使用小写单词，不使用下划线；避免 `common`、`utils` 等无边界大包。
6. 禁止循环依赖；依赖方向从 Controller 到 Service，再到 DAO/Infrastructure。
7. 三个及以上相关变量声明使用 `var` 块统一组织。

建议增加 CI 检查脚本，对人工文件行数进行校验，并排除：

```text
internal/dao/internal
internal/model/do
internal/model/entity
internal/web/ui
```

## 5. 注释规范

- 关键状态迁移、并发控制、幂等、安全边界和兼容逻辑必须添加中文注释；
- 注释解释“为什么”，不重复代码字面含义；
- 导出符号遵守 Go Doc 格式；
- 不在注释中保留真实 Token、Cookie、签名 URL 或生产业务 ID；
- 临时 TODO 必须包含原因和清理条件。

## 6. 错误处理

1. 使用 `gerror.Wrap/WrapCode/NewCode` 保留错误栈和稳定错误码。
2. Service 返回领域错误，不返回 HTTP 状态码。
3. Controller 通过统一中间件映射错误码和 HTTP 状态。
4. 禁止只记录错误后返回 nil；必须明确处理、包装或向上传递。
5. 可重试错误和永久错误必须区分。
6. 对外错误不得包含数据库 DSN、对象存储凭证、SQL、文件绝对路径和内部堆栈。

图片任务错误至少包含：

```text
error_code / stage / retryable / message
```

## 7. Context、Trace 与日志

### Context

- 所有 I/O、数据库、队列和跨服务方法第一个参数必须是 `context.Context`；
- 禁止用 `context.Background()` 截断请求链路，后台任务应从任务元数据恢复 Trace；
- 超时和取消必须向设备调用、下载和对象存储传播。

### Trace

- 使用 OpenTelemetry 规范的 32 位十六进制 Trace ID；
- 兼容已有 `X-Request-ID`，合法值转换并注入 Trace Context；
- HTTP、业务任务、设备 RPC 和回调保持可关联；
- `job_id` 和 `request_id` 不是 Trace ID，必须分别存储。

### 日志

统一使用 `glog` 和 Context。结构化日志必须传入 `g.Map` 或结构体，使日志内容输出为 JSON，至少包含：

```text
event, trace_id, job_id, request_id, client_id,
stage, status, duration_ms, error_code
```

禁止记录：

- 图片/base64；
- Token、Cookie、密码和密钥；
- 签名 URL；
- 完整原始 JSON；
- 人脸、人体和图片向量；
- 未脱敏的 Telegram 用户信息。

## 8. 数据库规范

1. 表名和字段名使用 snake_case。
2. 主键、新增业务 ID 和外部幂等 ID 的语义必须明确。
3. 状态字段使用受控常量，禁止散落魔法字符串。
4. 金额不用浮点数；相似度分数允许 REAL/DOUBLE，但必须记录算法版本。
5. 原始 JSON 使用 TEXT/LONGTEXT 逐字节保存；需要查询时另存解析 JSON 字段。
6. 图片二进制、base64 和大型 Embedding 不写普通业务表。
7. 所有列表接口必须分页并有稳定排序。
8. 高频过滤字段和唯一约束必须在迁移中显式创建索引。
9. 事务应尽量短，不在事务中执行网络请求、图片下载和模型推理。
10. 数据库 Schema 变更后必须重新运行 `gf gen dao`，并检查生成 diff。

## 9. 队列与并发规范

- Redis/Asynq 只保存任务 ID 和阶段；
- 数据库是业务任务权威状态；
- Worker 必须幂等，可承受重复投递；
- 使用有限重试、指数退避和死信/归档状态；
- 不允许无限 goroutine；并发量由配置和设备 Action 上限共同约束；
- 设备上传和搜索必须保持设备与会话亲和；
- 租约过期、断线、迟到结果和取消都必须有测试。

## 10. 文件与对象存储规范

- 上传后先写临时文件，再原子 rename；
- 使用 magic bytes 和真实解码校验，不只信任扩展名和 Content-Type；
- 使用 SHA-256 内容寻址和去重；
- 本地处理完成后上传私有对象存储；
- 数据库保存对象 Key，不持久化临时签名 URL；
- DB 与对象存储使用状态机和对账任务保证最终一致；
- 删除采用引用计数/延迟删除，避免删除共享内容对象。

## 11. HTTP 与外部调用规范

- 请求体、上传数量、文件大小和超时必须有限制；
- API 使用版本路径；
- 创建接口支持幂等键；
- 统一响应和错误码，不由各 Handler 自行设计；
- 外部下载必须防 SSRF、DNS 重绑定和重定向绕过；
- 外部回调必须签名、限时、有限重试并记录结果；
- Telegram Adapter 只转换消息和调用 Service，不实现任务流程。

## 12. 安全规范

- 密钥只来自环境变量或安全配置源；
- 配置示例使用占位符；
- 对象存储默认私有；
- 管理端、第三方 API 和 Bot 权限相互隔离；
- 人脸和人体特征作为敏感数据，启用最小权限和生命周期策略；
- 禁止把 `.env`、`config.yaml`、`data/`、日志、Hopper 工程和调试响应提交到 Git。

## 13. 测试规范

每个新功能至少覆盖：

- Service 单元测试；
- DAO/数据库集成测试；
- Worker 重复投递和重试测试；
- 状态机非法迁移测试；
- HTTP 参数和错误映射测试；
- 文件边界、未知格式和路径穿越；
- SSRF、重定向、超大响应和图片炸弹边界；
- 设备断线、会话变化和迟到结果；
- 批量 1、100 和超限输入；
- `go test -race ./...`。

提交前最低检查：

```bash
go test ./...
go test -race ./...
go vet ./...
npm run build
git diff --check
```

## 14. Git 与文档规范

- 一个提交只包含一个清晰阶段；
- 提交信息使用中文并说明业务影响；
- 不提交 `.spec-workflow/`、`data/` 和运行日志；
- 重要架构决策同步到 `docs/`；
- 每完成一个阶段更新 `docs/progress.md`；
- 生成代码和 Schema 迁移应在同一提交中，避免版本不一致。

## 15. 第三方依赖准入

引入前记录：

```text
用途、官方仓库、版本、License、维护状态、替代方案、风险、升级策略
```

优先级：

```text
GoFrame 内置能力
→ Go 官方/云厂商官方 SDK
→ 活跃、成熟的开源库
→ 项目自研
```

框架已提供的能力禁止再引入功能重叠的库，除非有基准、兼容或安全证据证明必要。
