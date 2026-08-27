# AppAgent XHS 设备集群扩展规格

状态：Draft

版本：v1

## 0. 已确认架构决策

| 决策 | v1选择 |
| --- | --- |
| Tenant | 复用r1rpc group作为租户/资源边界，API Key作为调用身份，User用于后台管理 |
| XHS API | `/api/v1/xhs/*`，破坏性变更才升级主版本 |
| Opaque token | 随机opaque ID + 服务端状态，数据库/日志只使用token hash |
| 短期状态 | 单节点进程内有界缓存；暂不引入Redis或持久化native handle |
| 设备凭据 | 每设备凭据，绑定SDK/插件/协议版本；group key兼容两个发布版本 |
| accessToken | 可按scope返回普通AI响应，不进入日志、指标或长期审计 |
| device_busy | HTTP 503并返回Retry-After；调用方配额超限才使用429 |
| MCP图片工具 | 上传+搜索组合工具为主，保留高级两步工具复用同一图片 |
| 指定deviceId | 保留，仅admin或`device:select` scope，不进入普通MCP schema |
| 大数据传输 | v1图片HTTP入口后以受控WS base64下发，大响应复用gzip；后续增加设备HTTP临时对象下载 |
| 队列 | 单节点内存lease层、Hub/WebSocket接线和集中reaper已完成；后续实现Redis backend/Stream |

这些选择已经确认，不再列为Open Questions。多节点部署必须同时迁移Hub、binding和queue，不能只迁移
其中一个内存组件。
适用项目：`r1rpc`、`AppAgentKit`、`xhs-mcp-server`

## 1. 目标

本规格在 r1rpc 现有设备协议、Hub、分组、鉴权、审计和监控能力上扩展 AppAgent/XHS 能力，
不创建第二套不兼容的设备协议。

目标：

- 多台安装 AppAgent 插件的 iOS 设备接入同一个 r1rpc 集群。
- 服务端按 action、能力、健康状态和并发容量选择设备。
- 文本搜索、图片上传、整图搜索和 bbox 搜索通过统一服务 API 提供给 AI/MCP。
- 上传句柄和分页游标保持服务端 opaque，并强制绑定原设备。
- 统一成功响应、错误响应、审计脱敏和可观测指标。

非目标：

- 不向 AI 暴露任意 HTTP、任意 Selector、Cookie、签名或登录凭据。
- 不在图片原生分页入口验证完成前公开图片 cursor。
- 不改变 AppAgent Router 和现有业务命令的输入输出语义。
- v1 不解决 r1rpc 多节点 Hub 状态共享；仍按当前单服务节点部署。

## 2. 术语

| 术语 | 含义 |
| --- | --- |
| Device | 运行注入 App 和 AppAgentKit 的 iOS 设备进程 |
| Group | r1rpc 现有设备池和调用鉴权边界 |
| Action | r1rpc Job 的 `action`，映射 AppAgent command |
| Capability | 设备支持的 action、schema、feature 和并发声明 |
| Affinity | 后续调用必须回到产生状态的同一设备 |
| Native handle | 仅存在于 App进程内的上传或分页状态标识 |
| Opaque token | 服务端签发、客户端不可解析和不可篡改的 handle/cursor |
| Active | 设备或 action允许的最大在途执行数 |

## 3. r1rpc 协议基线

### 3.1 设备登录

继续使用现有接口：

```http
POST /api/client/login
Content-Type: application/json
```

```json
{
  "deviceKey": "<group-device-key>",
  "clientId": "ios-device-01",
  "group": "xhs-prod",
  "platform": "ios-appagent",
  "maxInFlight": 1,
  "actions": [
    "content.search_notes",
    "content.search",
    "media.upload_image",
    "content.search_by_image",
    "network.request"
  ],
  "extra": {}
}
```

登录成功仍返回现有 JWT 和 `wsUrl`。v1 扩展字段全部可选；旧设备只上报 `actions` 时继续兼容。

### 3.2 WebSocket

设备连接现有：

```text
GET /api/client/ws?token=<device-jwt>
```

服务端 welcome：

```json
{
  "type": "welcome",
  "clientId": "ios-device-01",
  "group": "xhs-prod",
  "serverId": "r1rpc-node-1",
  "time": "2026-08-27T05:00:00+08:00",
  "maxInFlight": 1
}
```

服务端任务：

```json
{
  "type": "job",
  "job": {
    "requestId": "req-id",
    "group": "xhs-prod",
    "action": "content.search_notes",
    "clientId": "ios-device-01",
    "payload": {"query": "咖啡", "limit": 20},
    "createdAt": "2026-08-27T05:00:00+08:00",
    "deadlineAt": "2026-08-27T05:00:25+08:00"
  }
}
```

设备结果：

```json
{
  "type": "result",
  "result": {
    "requestId": "req-id",
    "status": "success",
    "httpCode": 200,
    "payload": {"items": [], "hasMore": false, "nextCursor": null},
    "error": "",
    "latencyMs": 325
  }
}
```

保留现有 `heartbeat/heartbeatAck`、WebSocket ping/pong 和 `probe/probeAck`。AppAgent iOS客户端必须
兼容未知 message type，忽略而不是断开。

### 3.3 兼容规则

- 现有 `Job/JobResult` 字段不改名、不改变含义。
- 扩展字段必须可选；旧客户端和旧服务端应能忽略未知字段。
- `requestId`是结果关联和幂等判定键；设备不得修改。
- 已超时结果按 r1rpc 现有 late-result 规则丢弃，不重新交付调用方。
- 同一 `clientId`的新连接替换旧连接，设备必须保证 clientId稳定且不跨设备复用。

## 4. Capability 扩展

### 4.1 登录声明

在保留 `actions: string[]` 的基础上，新增可选 `capabilities`：

```json
{
  "capabilities": {
    "protocolVersion": 1,
    "pluginVersion": "0.1.0",
    "appVersion": "9.44",
    "appBuild": "9440821",
    "platform": "ios-appagent",
    "actions": [
      {
        "name": "content.search_notes",
        "schemaVersion": 1,
        "maxInFlight": 1,
        "features": ["pagination"]
      },
      {
        "name": "media.upload_image",
        "schemaVersion": 1,
        "maxInFlight": 1,
        "features": ["jpeg", "png", "heic"]
      },
      {
        "name": "content.search_by_image",
        "schemaVersion": 1,
        "maxInFlight": 1,
        "features": ["full_image", "bbox"]
      }
    ]
  }
}
```

`actions`旧字段仍是最低兼容集合。存在 `capabilities.actions` 时，服务端以结构化声明为准，并验证
它与旧 `actions` 不冲突。

### 4.2 正式 action

| Action | 状态 | AI公开 | 说明 |
| --- | --- | --- | --- |
| `content.search_notes` | 正式 | 是 | 文本搜索 |
| `content.search` | 兼容 | 否 | protocol v1兼容别名 |
| `media.upload_image` | 正式 | 通过组合工具间接使用 | 返回设备本地短期 handle |
| `content.search_by_image` | 正式首页 | 是 | 整图或显式 bbox 首页 |
| `network.request` | 受控 | 否 | 只允许服务端白名单业务，不向普通 AI开放 |
| `diagnostics.*` | 管理域 | 否 | 仅管理员、测试组和调试设备 |

图片分页 capability 在原生分页触发入口、上下文复用和真机回归完成前不得包含 `pagination`。

## 5. iOS R1RPCDeviceTransport

### 5.1 边界

新增 `R1RPCDeviceTransport`，实现 r1rpc 登录与 WebSocket协议，并适配现有：

```text
AgentCommandRouter
AgentRequestEnvelope
AgentResponseEnvelope
```

Router、`content.search_notes`、`media.upload_image`、`content.search_by_image`和现有 Objective-C
业务服务不修改。

映射规则：

```text
r1rpc job.requestId       -> AgentRequestEnvelope.id
r1rpc job.action          -> AgentRequestEnvelope.command
r1rpc job.payload         -> AgentRequestEnvelope.payload
deadlineAt - now          -> timeoutMilliseconds
AgentResponse.result      -> JobResult.payload/status=success/httpCode=200
AgentResponse.error       -> JobResult.status=error/error/httpCode
```

为保留 AppAgent 的机器可读错误，`JobResult`新增可选 `errorInfo`；旧客户端只发送字符串 `error`时
继续兼容：

```json
{
  "error": "Device execution failed",
  "errorInfo": {
    "code": "device_error",
    "message": "Device execution failed",
    "retryable": true,
    "details": {}
  }
}
```

- `AgentCommandErrorPayload.code/message/retryable`原样进入 `errorInfo`，服务端再映射领域错误码和HTTP。
- `error`只保留脱敏兼容文本，不允许把结构化错误JSON塞进字符串后再解析。
- 旧设备只有 `error`时映射为 `device_error`，retryable由action策略决定。
- 协议版本、未知action和payload解码错误不能统一压成HTTP 500。

### 5.2 状态机

```text
idle
 -> loggingIn
 -> connecting
 -> online
 -> reconnectBackoff
 -> loggingIn
 -> stopped
```

- 登录失败按指数退避并增加随机抖动，认证失败使用较长退避。
- JWT过期或 WS收到认证错误时重新登录，不持久化服务端 JWT到日志。
- 网络恢复后自动重连；同一进程只允许一个活动 transport。
- welcome 后才视为 online。
- job执行使用 Swift Task；完成、失败或 deadline 到达时只回一次 result。
- v1 不承诺远程强制取消已经进入私有 SDK的原生调用；超时后本地清理，迟到结果不再回传或由服务端忽略。
- heartbeat、probe回包与 job结果写入必须串行化，避免 WebSocket帧并发写。
- 消息上限与 r1rpc 当前 4 MiB保持一致；图片 base64可能超过该值，因此生产应支持受控 HTTP上传
  或把上限配置化。不能无限提高 WebSocket上限。
- 设备可使用 r1rpc 已支持的 `gzip+base64+json`响应压缩；请求压缩需另行版本化。

## 6. Capability-aware 调度

当前 `Hub.pickSession`已按旧版 `actions`声明过滤并校验preferred client的group/action。结构化
capability中的schema/features尚未接入，仍是后续工作。

候选设备必须同时满足：

```text
session.Group == requested group
session supports requested action
schemaVersion满足请求
required features是设备features的子集
session在线且probe健康（支持probe的设备）
设备级和action级并发未满
队列未满
若指定affinity device，则clientId完全匹配
```

preferred client必须校验：

- client存在且在线；
- client属于请求 group；
- client支持 action/schema/features；
- affinity token中的 group、action scope与请求一致。

调度错误：

| 场景 | 错误码 |
| --- | --- |
| 组内没有在线设备 | `no_capable_device` |
| 有在线设备但都不支持 action/feature | `no_capable_device` |
| 指定设备离线 | `device_offline` |
| 指定设备不支持 action | `no_capable_device` |
| 所有候选均满载 | `device_busy` |
| affinity设备已不可恢复 | `affinity_lost` |

## 7. 并发策略

### 7.1 默认值

- 新 AppAgent设备默认 `maxInFlight=1`。
- 管理员可以修改设备 Active上限。
- action声明可指定更小的 `maxInFlight`；有效值为设备上限、action上限和服务端策略的最小值。
- 不允许设备通过重新登录把服务端管理员设置的硬上限调高。

### 7.2 action并发

建议初始策略：

| Action | 默认并发 |
| --- | ---: |
| `content.search_notes` | 1 |
| `media.upload_image` | 1 |
| `content.search_by_image` | 1 |
| `network.request` | 1 |

图片搜索插件当前只允许一个原生搜索，因此即使设备 Active提高，`content.search_by_image`仍必须为1。

Hub需要维护：

```text
deviceInFlight
actionInFlight[clientId][action]
queuedByAction
```

当前单节点Hub已维护设备和action在途计数，并以
`requestID/clientId/action/queueID/leaseID/session incarnation`绑定执行租约；generation仅用于进程内排序。旧版actions未提供结构化
上限时使用配置的安全默认值（默认1），显式测试接口可提供action上限且最终值不会超过设备上限；管理员
持久化策略和结构化capability接线尚未实现。

结果成功、业务失败和设备错误均释放槽位。服务端调用超时后将请求标为 expired；如果设备仍在执行，
槽位可在迟到结果到达或执行租约到期时释放。连接断开会先把仍有效的delivery重排，供相同clientId的
replacement incarnation重试，而不是Ack丢弃。执行租约到期时间取
`max(deadlineAt, acquireAt + executionGrace)`；到期后强制回收槽位并进入终态。释放状态机必须幂等，防止强制回收后
收到迟到结果再次减计数。当前 completed记录的10分钟清理不能作为槽位回收机制；设备持续在线但
永不回包时也必须恢复容量，并记录租约超时和强制回收指标。

组内调度采用 capability过滤后的轮询；同一 action队列应避免单个调用方长期占满设备。

## 8. Opaque handle、cursor与设备亲和

### 8.1 原则

- App原生 handle、`file_id/search_id/request_id`不得返回给 MCP或持久化到普通审计日志。
- 服务端对外 token必须不可猜测、不可篡改、有 TTL且单用途。
- token至少绑定 tenant/caller、group、clientId、kind、action和过期时间。
- token必须同时绑定设备进程会话代次 `sessionId`。相同clientId的App重启或连接替换后，原生内存
  handle已经失效，不能仅凭稳定clientId继续使用。
- 后续调用必须通过 `clientId`固定路由到原设备，禁止静默换设备。

逻辑 claims示例（不是对外格式承诺）：

```json
{
  "v": 1,
  "kind": "image_upload_handle",
  "tenant": "tenant-id",
  "group": "xhs-prod",
  "clientId": "ios-device-01",
  "sessionId": "process-incarnation-id",
  "action": "content.search_by_image",
  "nativeRef": "server-side-reference",
  "expiresAt": 1787776000,
  "nonce": "random"
}
```

`nativeRef`推荐指向服务端短期状态记录，避免把设备 native handle直接放入自包含 token。Token签名算法和
状态存储方式属于待确认项。

每个 App进程生成稳定的 process incarnation；服务端每次新进程登录建立新sessionId。进程重启、连接
替换或显式注销时批量失效该sessionId关联的handle/cursor。普通网络重连只有在设备证明进程代次未变时
才能继续使用原上下文，服务端不能自行猜测。

### 8.2 图片上传两步亲和

```text
AI上传图片
 -> 服务端选择设备A
 -> A执行media.upload_image并返回native handle
 -> 服务端保存短期映射并返回opaque uploadHandle
AI调用图片搜索
 -> 服务端解析uploadHandle
 -> 强制路由设备A
 -> payload内注入native handle
```

设备A离线时返回 `affinity_lost`或`device_offline`，不得把 handle发给设备B。

### 8.3 cursor亲和

- 文本 cursor如由设备内存生成，也必须绑定原设备。
- 图片 cursor必须绑定上传上下文、bbox、业务 ID和原设备。
- cursor过期返回 `cursor_expired`；handle过期返回 `handle_expired`。
- 图片分页未验收前，服务 API和 MCP不返回图片 `nextCursor`能力承诺。

## 9. XHS Service API

MCP Gateway只调用领域 API，不再持有设备 WebSocket。以下路径是建议草案，最终版本前缀待确认。

### 9.1 文本搜索

```http
POST /api/v1/xhs/notes/search
```

```json
{"query":"咖啡","limit":20,"cursor":null,"deviceId":null}
```

`deviceId`只允许具备管理/定向调用权限的调用方使用。普通 AI由服务端调度。

### 9.2 图片上传

```http
POST /api/v1/xhs/images/uploads
Content-Type: multipart/form-data
```

返回服务端 opaque handle：

```json
{
  "uploadHandle": "opaque",
  "width": 1228,
  "height": 768,
  "expiresInSeconds": 300
}
```

### 9.3 图片搜索

```http
POST /api/v1/xhs/images/search
```

默认不发送 bbox：

```json
{
  "uploadHandle": "opaque",
  "bboxEnabled": false,
  "limit": 30
}
```

显式 bbox：

```json
{
  "uploadHandle": "opaque",
  "bboxEnabled": true,
  "crop": {"x":0.1,"y":0.1,"width":0.8,"height":0.8},
  "limit": 30
}
```

- `bboxEnabled=false`时忽略 crop，设备请求不包含 bbox。
- `bboxEnabled=true`时必须校验有限数值、正面积和归一化边界。
- bbox Double文本格式由客户端插件按已验证原生格式兜底。

### 9.4 网络请求

`network.request`仅作为内部 action存在。服务端必须按 tenant/action配置 Host、path、method和 body大小
白名单，普通 MCP不提供“任意 URL请求”工具。

## 10. 标准数据模型

### 10.1 成功信封

```json
{
  "success": true,
  "requestId": "server-request-id",
  "data": {},
  "meta": {
    "deviceId": "ios-device-01",
    "latencyMs": 325
  }
}
```

普通 AI响应可省略或脱敏 `deviceId`；管理调用保留。

### 10.2 Note DTO

```json
{
  "id": "note-id",
  "type": "normal",
  "title": "标题",
  "description": "摘要",
  "author": {
    "id": "author-id",
    "name": "昵称",
    "avatarURL": "https://..."
  },
  "coverURL": "https://...",
  "imageURLs": ["https://..."],
  "likeCount": 0,
  "commentCount": 0,
  "accessToken": "opaque-upstream-token"
}
```

搜索响应：

```json
{
  "items": [],
  "hasMore": false,
  "nextCursor": null
}
```

字段缺失使用 `null`或省略的最终策略必须在 OpenAPI/MCP schema中统一，不能 Gateway和设备各自决定。

### 10.3 错误信封

```json
{
  "success": false,
  "requestId": "server-request-id",
  "error": {
    "code": "device_busy",
    "message": "No capable device currently has execution capacity",
    "retryable": true,
    "details": {}
  }
}
```

设备原始错误可映射和上报，但 message必须脱敏，不得包含 URL query、Token、Cookie、签名、图片正文、
native handle或完整私有对象描述。

## 11. 统一错误码

| Code | HTTP | Retryable | MCP处理 |
| --- | ---: | ---: | --- |
| `validation_error` | 400 | 否 | `isError=true`，提示修正参数 |
| `unauthorized` | 401 | 否 | `isError=true` |
| `forbidden` | 403 | 否 | `isError=true` |
| `rate_limited` | 429 | 是 | `isError=true`，可返回retryAfter |
| `no_capable_device` | 503 | 是 | `isError=true` |
| `device_offline` | 503 | 是 | `isError=true` |
| `device_busy` | 429或503 | 是 | `isError=true` |
| `affinity_lost` | 409 | 否 | 重新开始上传/搜索 |
| `handle_expired` | 410 | 否 | 重新上传 |
| `cursor_expired` | 410 | 否 | 重新搜索首页 |
| `timeout` | 504 | 是 | `isError=true` |
| `device_error` | 502 | 视设备错误 | `isError=true` |
| `upstream_error` | 502 | 视上游错误 | `isError=true` |
| `protocol_error` | 502 | 否 | `isError=true`并告警 |
| `internal_error` | 500 | 是 | `isError=true`，隐藏内部细节 |

MCP structuredContent只用于成功结果。错误返回 `isError=true`和统一错误 JSON文本，不把所有失败压平为
一条不可机器识别的字符串。

## 12. MCP Gateway

目标调用链：

```text
AI/MCP Client
 -> xhs-mcp-server (stdio或远程MCP)
 -> XHS Service API
 -> r1rpc调度
 -> iOS AppAgent设备
```

Gateway不再监听或连接设备 WebSocket，不负责设备替换、心跳和 pending request。

正式工具：

- `xhs_search_notes`
- `xhs_upload_search_image`（高级两步调用）
- `xhs_search_by_image`（支持直接图片上传的组合模式或服务端opaque handle）

图片分页完成后再增加 cursor输入/输出承诺。管理工具如设备列表、能力和诊断应放入独立 MCP server或
独立权限域，不与普通内容搜索工具混用。

MCP schema必须覆盖 Note DTO的 `description/commentCount/imageURLs/author`，不得被 Zod默认剥离。

## 13. 安全、脱敏与审计

- 生产必须使用 HTTPS/WSS。
- 现有 group device key可作为兼容入口，但应演进为每设备凭据，可独立吊销和轮换。
- App源码不得包含默认 Token；配置从受控环境或安全配置下发。
- `media.upload_image`的 base64/image bytes不得写入 `rpc_requests`。
- opaque token、native handle、`file_id/search_id/request_id`不得写入普通日志。
- `accessToken`默认不进入长期审计；如业务必须返回，只保留在调用响应中。
- `network.request`的 header/body按 action白名单脱敏，禁止 Cookie、Authorization和签名材料落库。
- 审计保留 requestId、tenant、group、deviceId、action、状态、错误码、大小、耗时和时间，不保留敏感正文。
- API Key必须支持作用域、配额、轮换和撤销；常量时间比较作为实现要求。

## 14. 数据和配置扩展

建议数据变化：

- `devices`：增加 capability schema/version摘要、plugin/app版本和管理员Active上限。
- 新表或结构化列 `device_action_capabilities`：`client_id/action/schema_version/features/max_in_flight`。
- 短期状态表或缓存 `opaque_bindings`：token hash、kind、tenant、group、client、encrypted/native ref、expiry。
- `rpc_requests`：增加规范化 `error_code`，请求/响应正文改为action级审计摘要。
- action策略：默认并发、超时、请求体上限、允许组、审计策略和feature要求。

配置建议：

```yaml
limits:
  client_max_in_flight: 1
  action_default_max_in_flight: 1
  queue_lease_duration_seconds: 10
  queue_lease_reaper_interval_seconds: 1
  execution_grace_seconds: 30
  image_upload_max_bytes: 12582912
  opaque_handle_ttl_seconds: 300
  cursor_ttl_seconds: 300
```

数据库迁移必须幂等并兼容已有数据；不能只修改 `schema.sql`而不补 existing schema migration。

## 15. 可观测性

至少提供：

- 在线设备数、健康设备数、按 action可用设备数。
- 调度等待时间、执行时间、端到端时间。
- action成功、业务错误、设备错误、超时和饱和率。
- 每设备和每action的 in-flight、queue depth、late result。
- affinity命中、过期、设备离线和丢失次数。
- opaque handle签发、消费、过期次数，不记录 token值。
- 协议/schema/feature不匹配次数。

日志以 `requestId/clientId/group/action/errorCode`关联，敏感字段按第13节处理。

### 15.1 可控队列与Redis迁移边界

队列采用稳定 `queueId/jobId`和实现无关接口。内存lease层已完成 ready/leased分离、容量、过期、
`Reserve/Renew/Ack/RequeueDelivery/Sweep`和有界trace；trace事件包含 `queued`、`dequeued`、`requeued`、
`expired`、`dropped`、`reserved`、`lease_renewed`、`acked`和`lease_expired`，只包含request/group/action/client和
时间、原因及不可操作的lease引用，不包含payload或原始LeaseID。

成功Ack以有界tombstone提供重复Ack幂等：每条最多保留5分钟，每队列最多1024条；达到最大年龄或
容量上限时按先到者淘汰，任一约束先触发即结束对应delivery的幂等保证，不承诺固定保留5分钟。

Redis后端实现时必须保持现有lease语义：

- 单节点内存实现按clientId跨connection incarnation复用QueueID；JobID跨重排稳定，每次投递使用独立LeaseID。
- Enqueue、Reserve、Renew、Ack和RequeueDelivery在Redis中以Lua原子执行。
- ready与leased分别存储，lease到期由reaper重排或终结。
- 交付语义是at-least-once，不承诺exactly-once；业务以JobID去重。
- Redis实现以Redis TIME为权威；lease到期判断、Ack/Requeue校验和tombstone清理不得信任应用节点或
  调用方传入的墙上时钟。
- Ack tombstone同样受最多5分钟和每队列1024条约束，按最大年龄到期或最旧条目容量淘汰中先到者执行。
- Redis故障时fail closed，不静默切回内存队列产生双写。
- trace迁移到Redis Stream，状态转换和事件写入保持原子顺序。

单节点生产链路现已完成Hub/WebSocket的`Reserve/Ack/RequeueDelivery/Sweep`接线：writer先Reserve再
原子申请设备/action执行槽；reservation临近到期时续租，取得槽后按独立execution expiry续租delivery。
结果Ack delivery并幂等释放，集中reaper扫描全部保留的client queue并同步回收过期execution lease。
连接替换和断线按不可碰撞的session incarnation隔离旧连接，仍有效delivery重排、ready任务保留；若任务已经写入
旧socket，重排trace以`after_send_at_least_once`明确记录重复交付风险，物理发送不可撤销。调用方可先收到503，
replacement仍可继续at-least-once消费。HTTP result/logout只接受带incarnation的connection token；bootstrap token
只用于首次WebSocket绑定。旧登录token仍可建立WebSocket，welcome返回connection token，旧Python客户端忽略新增
welcome字段也不影响WebSocket基本收发。空闲client queue仅在无ready、leased、execution lease和waiter并超过TTL后
回收；每队列trace和Ack tombstone均有容量上限。以上只在
单节点进程内保证，进程退出仍会丢失内存队列。

仍未完成Redis backend/Stream、多节点Hub/binding/queue共享、管理员action策略持久化和结构化capability
schema/features接线。因此当前完成状态只适用于单节点内存队列，不宣称多节点生产队列已完成。

## 16. 测试矩阵

### 协议与Transport

- login/welcome/job/result/heartbeat/probe正常流程。
- Token过期、断网重连、重复连接和未知消息。
- malformed JSON、超大消息、错误requestId和重复/迟到result。
- AppAgent Router映射成功、业务错误、deadline和一次性完成。

### 调度

- 混合能力设备只收到自己支持的 action。
- preferred client跨group、离线、不支持action均拒绝。
- 默认Active=1和管理员/action覆盖。
- 满载、队列满、超时、断线槽位释放和公平轮询。

### 亲和

- 上传和搜索始终在同一设备。
- token篡改、过期、跨tenant、跨group、跨action拒绝。
- affinity设备离线不切换设备。
- 文本cursor回原设备；图片cursor在开放后同样验证。

### 业务

- 文本空/非空、首页和分页。
- 图片上传格式和大小边界。
- 图片整图请求无bbox。
- bbox开启时数值、字段顺序、17位Double格式和URL编码正确。
- 图片空/非空响应即时完成。

### MCP

- schema不丢字段。
- 所有服务错误稳定映射为统一MCP error。
- Gateway无设备WebSocket依赖。
- 敏感字段不出现在日志和持久化明细。

## 17. 实施阶段

### Phase 1：协议兼容和iOS Transport

1. 先实现action级审计策略和脱敏，保证图片base64、native handle、accessToken和签名材料不落库。
2. 扩展登录 capability、process incarnation和结构化错误模型，旧字段兼容。
3. 实现 `R1RPCDeviceTransport`和协议映射。
4. 完成 login/job/result/heartbeat/probe/重连兼容测试。
5. 移除 App源码硬编码Token。

验收：多台设备可同时在线，现有 action可通过 r1rpc定向调用，Router和业务命令未改；启用任何图片
action前，测试必须证明base64和敏感业务字段不进入数据库或日志。

### Phase 2：调度和并发

1. 已完成：旧版action capability进入在线Session，Hub按action筛选。
2. 未完成：结构化schema/features筛选与持久化。
3. 已完成：preferred client的group/action校验。
4. 部分完成：默认Active=1、设备Active和内存action并发覆盖已完成；管理员持久化策略未完成。
5. 已完成：单节点reservation/execution租约分离、强制槽位回收和幂等释放。
6. 已完成：槽位、超时、迟到结果、永不回包回收、incarnation fence、queue回收和单进程断线重试测试。

### Phase 3：XHS领域API和亲和

1. 实现标准Note DTO和统一错误模型。
2. 实现图片上传opaque handle和同设备搜索。
3. 实现文本cursor亲和。
4. 在Phase 1脱敏基础上增加tenant配额和网络白名单。

### Phase 4：MCP迁移

1. Gateway改调XHS Service API。
2. 补全MCP schema和机器可读错误。
3. 保留本地单设备Gateway作为开发模式，不作为生产集群入口。

### Phase 5：图片分页

只有同时满足以下条件才公开：

- 找到并验证稳定的原生分页触发入口。
- 真机证明复用 `file_id/search_id/request_id/bbox/source`且只推进cursor。
- 首页返回可用服务端opaque cursor。
- 同设备亲和、TTL、设备离线、空页和cursor过期测试通过。
- MCP/API schema和错误文档完成。

## 18. 灰度和回滚

- 新Transport和旧直连Transport通过配置切换，不同时建立两条生产连接。
- capability-aware调度先在测试组启用；未知能力设备保持旧action集合但不得接收要求feature的请求。
- XHS Service API先支持指定测试设备，再开启组内调度。
- opaque handle启用后禁止客户端native handle直通；回滚时只允许清空短期状态并要求重新上传。
- 数据库扩展采用只加列/表的前向兼容迁移，回滚不删除数据。

## 19. 待人工确认

以下内容在编码前必须确认：

1. 每设备凭据SDK版本策略使用精确版本、最低版本区间还是服务端兼容规则表。
2. API Key多Key模型是否在v1直接增加独立表，还是先兼容现有group单Key字段。
3. `accessToken`默认scope是否授予内置可信MCP Key，第三方Key默认不授予。
4. Redis实现选用的客户端库、部署拓扑和故障时fail-closed策略细节。
5. 管理端队列trace的保留时长、查询权限和是否写入长期审计。
