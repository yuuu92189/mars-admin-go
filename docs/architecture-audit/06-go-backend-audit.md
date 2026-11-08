# 06 Go Backend Audit

## 1. Current Structure

`mars-admin-go` 是按业务模块组织的 modular monolith：

```text
cmd/*
  -> internal/bootstrap
      -> modules/*/application
          -> modules/*/domain
          -> modules/*/adapter/{http,postgres,river,local,s3}
      -> internal/platform/*
```

启动装配集中在 `internal/bootstrap/app.go`；API、Worker、Migrate、Legacy Import、Outbox 各自有入口。域包原则上不依赖 Chi、pgx、Redis 或 HTTP。

## 2. Module Inventory

| 模块 | 职责 | 关键文件 |
| --- | --- | --- |
| Identity | 用户、Argon2id、Session、Refresh、验证码、SMS、TOTP MFA、社交登录 | `internal/modules/identity` |
| Access | 角色、权限、菜单、超级管理员、数据范围 | `internal/modules/access` |
| Organization | 部门树、岗位、成员关系 | `internal/modules/organization` |
| System | 字典、配置、敏感配置加密、修订 | `internal/modules/system` |
| Audit | 登录事件、操作事件 | `internal/modules/audit` |
| Monitoring | Go runtime、数据库池、Redis、运行快照 | `internal/modules/monitoring` |
| Storage | 本地/S3 兼容存储、上传、私有下载 | `internal/modules/storage` |
| Messaging | 私聊、群聊、已读、幂等、WebSocket、Redis fan-out | `internal/modules/messaging` |
| Notification | 公告、收件人、已读、发送日志 | `internal/modules/notification` |
| Scheduler | River 任务定义、cron、运行记录 | `internal/modules/scheduler` |
| Workflow | JSONB 定义、状态机、审批、事件历史 | `internal/modules/workflow` |
| Codegen | PostgreSQL introspection、导入、预览、ZIP | `internal/modules/codegen` |
| Integrations | SMS、支付、推送、邮件、社交 Provider 契约 | `internal/modules/integrations` |
| Outbox | 事务内事件、River relay、消费补偿 | `internal/modules/outbox` |

## 3. API and Contract

### Scale

- OpenAPI 3.1 文件：`api/openapi/v1/openapi.yaml`。
- Phase 1 后为 100 paths，135 operations；所有需要响应体的 2xx response 都有 content schema。
- `api/openapi_test.go` 验证 OpenAPI 文档可加载、operationId 唯一。
- 规范 API 前缀：`/api/v1/*`。
- 旧 API：`/api/*`，由 `internal/platform/compat` 包装。

### Confirmed Contract Issues (First-Round Snapshot)

The following entries record the first-round audit findings. Phase 1 remediation is recorded in section 9; the current contract statistics are in section 3.

#### GO-API-001 `/metrics` 安全声明与实际实现冲突

- OpenAPI：`api/openapi/v1/openapi.yaml` 中 `/metrics` 为 `security: []`，并声明 200 文本响应。
- 实际：`internal/bootstrap/app.go:243` 使用 `RequirePermission("audit:read")`。
- 集成测试：`test/integration/api_test.go:121-141` 明确断言未认证 401、管理员 token 200。
- 影响：文档读者会误以为 Prometheus 可匿名抓取；实际部署需要认证。
- 处理：将 OpenAPI 改为受保护，或实现独立的只读监控认证方案；不能同时保留两种相反声明。

#### GO-API-002 登录字段和响应形状漂移

- `scaffold-vue/src/api/auth.ts` 的 `LoginRequest` 使用 `captcha`、`captchaUuid`。
- Go `identitydomain.LoginInput` 使用 `uuid`、`code` JSON 标签。
- `internal/modules/identity/adapter/http/handler.go` 的 `login` 直接解码 `LoginInput`，未知字段会被忽略。
- 前端期望 `LoginResponse.expiresIn`；Go `LoginResult` 返回 `expiresAt`、`token`、`user`、`access` 等。
- 影响：验证码开启时兼容登录可能失败；`expiresIn` 缺失会让前端过期时间计算异常。
- 处理：在兼容 HTTP 层做显式字段映射，或在 OpenAPI 中定义统一登录契约并更新前端；不能靠字段同名猜测。

#### GO-API-003 分页和 ID 类型不一致

- `scaffold-vue` 使用数字 ID 和 `{pageNum,pageSize,total,rows}`。
- Go 原生使用 UUID 字符串和 `{items,total,page,pageSize}`。
- 旧 `/api/sys/user/page` 等兼容路由会做部分映射；`/api/v1` 原生路由不保证与 Vue 类型兼容。
- 影响：不能直接把 `scaffold-vue` 的 `system.ts` 指向 `/api/v1`。
- 处理：生成 TypeScript 类型后逐页适配；UUID 在前端统一按字符串处理，禁止转 `number`。

#### GO-API-004 OpenAPI 响应 schema 不完整（第一轮）

大量 operation 只有 `description`，没有 `content/schema`，例如字典、审计、部分消息、通知、调度、工作流和监控。`CurrentUser.menus` 仍使用 `additionalProperties: true`。

影响：oapi-codegen 无法生成稳定 TypeScript 类型；前端会继续维护手写类型，违反“OpenAPI 为唯一契约来源”。

处理：Phase 1 已补齐当前 canonical operation 的响应 schema；动态 map 的剩余边界和后续产品模块见 section 9。

#### GO-API-005 群组管理路由未完整进入 OpenAPI

实际 handler 有：

- `DELETE /api/v1/groups/{groupId}/members/{memberId}`
- `POST /api/v1/groups/{groupId}/members/{memberId}/role`
- `POST /api/v1/groups/{groupId}/members/{memberId}/mute`
- `POST /api/v1/groups/{groupId}/transfer/{newOwnerId}`

OpenAPI 当前主要覆盖 list/add members、messages、quit、dissolve 等，未完整覆盖这些管理动作。

#### GO-API-006 权限码与前端旧权限码不一致

Go 种子权限见 `db/migrations/000007_seed_authorization.sql`，例如 `user:read`、`role:read`、`department:read`、`dictionary:read`、`settings:read`、`audit:read`、`message:read`、`notice:read`、`scheduler:read`、`workflow:read`、`codegen:read`。

`scaffold-vue` 页面使用 `system:user:*`、`monitor:operlog:*`、`tool:gen:*`、`vuln:*` 等旧码。两者不能直接共用授权菜单。

处理：建立明确的权限码迁移表，或让兼容层做只读映射；不要同时接受两套码导致权限膨胀。

## 4. Authentication and Authorization

### Implemented

- Argon2id 密码哈希，支持 bcrypt 迁移。
- Access/Refresh Token 只存 SHA-256 hash。
- Refresh CAS、generation、history、grace period 和重放撤销。
- Cookie 默认 HttpOnly；生产要求 Secure。
- CSRF：Cookie Session 的 unsafe 请求必须带 `X-CSRF-Token`。
- 登录失败数据库计数、Redis 限流、账号锁定。
- TOTP MFA，密钥 AES-256-GCM。
- `RequirePermission` / `RequireAnyPermission` 路由中间件。
- 用户列表和部门树支持 Data Scope；部门查询下推到 PostgreSQL。

### First-Round Gaps

- 第一轮审计发现 Workflow 读接口缺少实例级 Data Scope；Phase 1 已在实例、详情和事件读取中修复，见 section 9。
- `internal/modules/integrations/http.go` 的 provider status 路由只使用认证，没有 `settings:read`；是否允许普通登录用户查看 Provider 状态需明确。
- 兼容层仍接受 legacy raw Authorization；这是迁移期边界，不是目标浏览器认证。
- `scaffold-vue` 仍把 token 和明文密码放在浏览器存储中；Go 后端无法修复该前端漏洞。

## 5. Database and Migrations

### Current

- PostgreSQL 是唯一业务主数据库。
- 23 个 Goose 迁移，包含 identity、access、organization、system/audit、storage/messaging、scheduler/workflow/codegen、legacy import、MFA、social safety、soft-delete uniqueness、query indexes、transactional outbox。
- UUIDv7 主键；时间使用 `timestamptz`。
- 软删除使用部分唯一索引，允许删除后复用用户名/编码。
- Outbox 使用 PostgreSQL 事务和 River `InsertTx`。
- 迁移和数据库访问不依赖 GORM/Ent。

### Gaps

- 支付只有 Provider 接口和回调验签；没有订单状态账本、金额/订单号绑定、回调幂等和重放窗口。
- Workflow Data Scope 尚未落实到实例读取。
- 历史数据迁移已在隔离数据库执行，不代表生产切换已批准。
- 旧 SQL dump 含敏感/疑似真实数据，不能提交或用于生产种子。

## 6. Error, Observability, and Operations

### Implemented

- RFC 9457 Problem Details。
- Request ID、trace ID、结构化 slog、Prometheus metrics、OpenTelemetry。
- 健康检查区分 live/ready；ready 检查 PostgreSQL 和 Redis。
- Graceful shutdown、River worker、Outbox relay。
- CORS 白名单、Security Headers、BodyLimit、Timeout、RateLimit、CSRF。
- 旧 envelope 仅在 HTTP 兼容层。

### Gaps

- `/metrics` 契约与实现不一致。
- Audit metadata 数组中的 map 递归脱敏仍需加强；审计写入错误可观测性不足。
- 旧兼容 handler 仍有大量 `map[string]any`，迁移完成后应删除而不是继续扩展。
- 多个 application/adapter 包单测覆盖为 0%，关键错误分支主要依赖集成测试。

## 7. Known Architecture Debt

| ID | 问题 | 级别 | 证据/建议 |
| --- | --- | --- | --- |
| GO-ARCH-001 | Identity PostgreSQL adapter 直接依赖 Access sqlc | 已修复 | `db/queries/identity/users.sql` + identity-owned sqlc |
| GO-ARCH-002 | Notification adapter 直接依赖 Messaging sqlc | 已修复 | `db/queries/notification/notification.sql` + notification-owned sqlc |
| GO-ARCH-003 | Domain Repository 接口过大 | P2 | Identity/Access/Messaging；按用例拆分 reader/store |
| GO-ARCH-004 | Monitoring 直接读多模块表 | P2 | 标记为 read model；长期使用 view/独立 dashboard query |
| GO-ARCH-005 | Workflow 实例级 Data Scope 缺失 | 已修复 | 实例/详情/事件 SQL 使用统一 visibility predicate |
| GO-ARCH-006 | Legacy stubs 返回固定成功/空数据 | 已处理已确认项 | 调度统计读取真实 run；无能力接口返回 501 Problem |

## 8. Verification Status

本次重新执行：

- `go test ./...`：通过。
- `go vet ./...`：通过。
- OpenAPI 文档校验：通过，100 paths、135 operations、operationId 唯一。

历史且已有记录的可复核证据：

- `docs/go-standard-audit/verification.md`：Race、Testcontainers、lint、govulncheck、Compose、Helm。
- `docs/audit-round-3/verification-report.md`：Refresh/Outbox/多 Worker。
- `docs/audit-final/final-verification.md`：最终运行态。

结论：Go 后端已达到较强的工程基础，但不是“生产就绪”证明；支付、Provider 认证、多租户模型和覆盖率缺口仍是明确风险。

## 9. Phase 1 Remediation Update

### Resolved

- `GO-API-001`：`/metrics` 保持 `audit:read`，OpenAPI 继承安全方案并声明权限、401/403/500。
- `GO-API-002`：新登录 API 使用 `captchaId`/`captchaCode`、`expiresAt`；旧字段和 `expiresIn` 只存在于 legacy adapter。
- `GO-API-003`：新 API 分页固定为 `items/total/page/pageSize`；旧别名只在兼容边界保留。
- `GO-API-004`：OpenAPI 当前所有需要响应体的 2xx response 都有 content schema；动态 metadata/settings/variables 保留明确说明。
- `GO-API-005`：群组成员删除、角色、禁言和群主转让已进入 OpenAPI。
- `GO-ARCH-001`：Identity 不再直接导入 Access sqlc。
- `GO-ARCH-002`：Notification 不再直接导入 Messaging sqlc。
- `GO-ARCH-005`：Workflow 实例、详情和事件读取执行实例级 Data Scope。
- `GO-ARCH-006`：已确认的 Legacy Stub 已改为真实统计或显式 501 Problem。
- PostgreSQL SQLSTATE 在 `internal/platform/httpserver/json.go` 统一映射，包含唯一、外键、非空、检查、序列化、死锁和取消。
- Access 的 Role/Menu Repository 和 Workflow 的 Definition/Runtime Repository 已按使用方拆分。

### Remaining

- Identity 和 Notification 的 Repository 仍然偏大，未做无收益的全量重写。
- 漏洞资产、漏洞和修复工单没有 Go 领域模块，属于产品决策后的 Phase 6。
- 支付没有订单状态账本和回调 replay ledger；本轮没有虚构支付结算。
- 外部 Provider 仍然只保证至少一次投递和显式 not-configured。
- 生产数据切换、Provider 认证、真实对象存储测试和 React 视觉回归未执行。
- 当前数据库没有 `tenant_id` 或多租户模型；现有 Workflow Data Scope 是单租户范围内的隔离。进入多租户产品前必须新增租户模型、租户绑定和跨租户测试。
