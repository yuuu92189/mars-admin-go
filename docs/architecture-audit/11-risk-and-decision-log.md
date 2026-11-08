# 11 Risk and Decision Log

## 1. Severity

| 级别 | 含义 |
| --- | --- |
| P0 | 阻断目标完成或存在可直接利用的高危安全问题 |
| P1 | 会破坏核心安全/业务/契约，进入实现前必须处理 |
| P2 | 重要工程质量或可维护性风险，可分批处理 |
| P3 | 优化项，不阻断核心交付 |

## 1.1 Phase 1 Risk Update

### Resolved or Reduced

- `RISK-002`：Metrics 安全声明、登录 canonical 字段和绝对过期时间已统一；legacy 字段只在 adapter 映射。
- `RISK-004`：Workflow 实例、详情和事件读取已执行实例级 Data Scope，并有真实 PostgreSQL 集成测试。
- `RISK-008`：Identity 和 Notification 的已知跨模块 sqlc 引用已移除。
- `RISK-009`：Access 和 Workflow Repository 已按用例拆分；Identity/Notification 仍未拆分，保留为后续债务。
- `RISK-011`：已确认的 Legacy Stub 已改为真实统计或明确 501 Problem。
- `RISK-007`：OpenAPI 当前所有 2xx response 都有 content schema；剩余动态 map 均有明确业务原因。

### Still Open

- `RISK-001`：React 前端尚未实现，因此旧的明文密码和浏览器 Token 问题仍存在于参考 Vue。
- `RISK-003`：漏洞领域未实现，等待产品确认。
- `RISK-005`：支付订单账本未实现，等待产品确认。
- `RISK-006`：外部 Provider 仍不能宣称 exactly-once。
- `RISK-010`：Monitoring 跨模块 read model 仍是当前架构的已知边界。
- `RISK-012`：Identity/Notification application、adapter 和部分错误分支覆盖率仍需补强。
- `RISK-013`：React 组件测试、Playwright 和视觉回归尚未开始。
- `RISK-014`：生产数据切换仍未批准。

Phase 1 的 OpenAPI 和 TypeScript 基线已可支撑 Phase 2 前端工程；未确认的产品模块不应被前端静态页面替代。

## 2. Confirmed Risks

### P1

#### RISK-001 明文密码和长期 Token 存在浏览器存储

- 位置：`scaffold-vue/src/views/Login.vue` 的 `scaffold.login-hint`；`scaffold-vue/src/api/session.ts` 的 `scaffold.token`。
- 事实：勾选记住登录后把账号和密码写入 `localStorage`；Bearer token 写入 `localStorage`/`sessionStorage`。
- 影响：XSS、共享设备和浏览器扩展可读取凭据。
- 处理：React 版本不保存密码；浏览器认证改为 HttpOnly Cookie + CSRF；仅保留非敏感 UI 偏好。

#### RISK-002 OpenAPI 与实际安全/登录契约漂移

- `/metrics` 文档声明公开，实际 `audit:read`。
- 登录字段 `captcha/captchaUuid` 与 Go `uuid/code` 不一致。
- 登录响应 `expiresIn` 与 Go `expiresAt` 不一致。
- 处理：Phase 1 先统一契约和兼容映射，再生成前端类型。

#### RISK-003 `scaffold-vue` 的安全业务没有 Go 等价模块

- 资产、漏洞、修复工单、统计和 AI 端点存在于 Vue 参考，Go 当前没有完整领域。
- 影响：不能实现真实漏洞页面；不能用 mock 代替。
- 处理：新增 `vulnerability` 模块或书面确认排除该业务。

#### RISK-004 Workflow 缺少实例级 Data Scope

- 位置：`internal/modules/workflow` 读接口；见 `docs/go-standard-audit/security.md`。
- 影响：有 `workflow:read` 的用户可能读取无权实例。
- 处理：按业务归属/部门/发起人接入 SQL Data Scope，增加越权测试。

#### RISK-005 支付回调没有订单账本和重放保护

- 现状：Provider 验签存在，但没有订单状态、金额/订单号绑定、回调幂等和 replay window。
- 影响：不能证明已结算，不能安全重放/重复回调。
- 处理：如保留支付，新增 payment/order 表、回调账本、金额校验、幂等键和测试。

#### RISK-006 外部 Provider 不是严格 exactly-once

- SMS、邮件、Push、支付、社交登录依赖真实凭证。
- Outbox/River 提供至少一次投递，不能替代 Provider 原生幂等。
- 处理：保持“至少一次”声明；Provider 成功但本地超时可能重复发送。

#### RISK-007 OpenAPI 响应 schema 不完整

- 多个 operation 只有 description，没有 response content/schema。
- `additionalProperties: true` 仍用于菜单等业务模型。
- 处理：按页面实际消费字段补齐 schema，并生成 TS client。

### P2

#### RISK-008 跨模块 sqlc 依赖

- Identity PostgreSQL adapter 依赖 Access sqlc。
- Notification adapter 依赖 Messaging sqlc。
- 处理：使用方定义最小 reader port；拆 notification-owned queries。

#### RISK-009 Repository 接口过大

- Identity、Access、Messaging 的 domain Repository 聚合大量无关方法。
- 处理：渐进按用例拆分，不一次性重写。

#### RISK-010 Monitoring 直接读多模块表

- 监控服务跨多个业务表读取统计。
- 处理：明确 read model 边界；长期使用 SQL view/独立 dashboard query。

#### RISK-011 Legacy stubs 和固定成功

- 调度统计、清日志、文件预览/删除、SMS 日志等存在空实现或固定值。
- 处理：实现真实能力或返回 typed not-configured；禁止假成功。

#### RISK-012 覆盖率不足

- 多个 application/adapter 包单测为 0%。
- 集成测试覆盖关键链路，但不能替代错误分支/并发边界单测。
- 处理：按风险补 Handler/Service/Repository 测试，不追求无意义覆盖率数字。

#### RISK-013 前端没有组件/E2E/截图回归

- 当前只有 API/utils Node tests。
- 处理：React 工程从一开始建立 Vitest + Playwright + screenshot baseline。

#### RISK-014 Legacy 数据迁移不能直接生产切换

- 隔离数据库导入成功不等于生产批准。
- 需要备份、字段映射批准、密钥轮换、行数/关联/金额校验、回滚窗口。

## 3. Design Decisions

### DEC-001 保持 Modular Monolith

- 决策：不拆微服务，不引入 Kafka/NATS/Elasticsearch。
- 理由：当前业务规模和部署复杂度不需要；River/PostgreSQL/Redis 足够。
- 例外：外部 Provider 和 Worker 仍有明确进程边界。

### DEC-002 保留自研组件和珍珠玻璃 CSS

- 决策：React 重建自研组件库；Tailwind 负责布局/响应式，复杂材质保留 CSS。
- 理由：原项目视觉资产是核心验收基准，不能用第三方 UI 库替换。

### DEC-003 GSAP + CSS 分层

- 决策：简单反馈用 CSS；多元素/共享元素/画布/复杂时间线用 GSAP；WAAPI 非必要不用。
- 理由：匹配原项目实际实现，避免把所有动画改成一种技术。

### DEC-004 浏览器认证改为 HttpOnly Cookie + CSRF

- 决策：React 不保存 access/refresh token；App 客户端保留受控 Bearer。
- 理由：消除 localStorage token 风险，符合目标提示词。

### DEC-005 OpenAPI 是唯一 API 契约

- 决策：新页面类型从 OpenAPI 生成；兼容层只在 HTTP adapter。
- 理由：防止 Vue/React/Go 三套类型漂移。

### DEC-006 UUID 字符串贯穿前端

- 决策：TypeScript 使用 string，不把 UUID 转 number。
- 理由：避免精度丢失和运行时错误。

### DEC-007 旧兼容层保留到迁移完成

- 决策：不立即删除 `/api/*`、envelope、query-token WebSocket。
- 理由：现有 Vue/UniApp 仍依赖；先建立调用者清单和契约测试。

### DEC-008 外部 Provider 不宣称 exactly-once

- 决策：统一声明至少一次；需要 Provider 原生幂等或业务账本才能升级。
- 理由：Outbox/River 只能保证本地投递可靠性。

## 4. Open Decisions

以下问题需要产品/数据所有者确认，不能由代码助手擅自决定：

1. 是否保留 `scaffold-vue` 的漏洞资产/漏洞/修复工单业务？
2. AI 助手是否进入目标产品？使用哪个模型/知识库 Provider？
3. 支付/订单是否在目标范围？如保留，订单状态和结算规则是什么？
4. 目标前端是独立仓库、monorepo，还是与 Go 后端同仓？
5. 旧 Vue/UniApp 的在线兼容窗口多长？哪些客户端必须支持？
6. 旧数据生产切换的时间、停机窗口、回滚责任人和审批人是谁？
7. 生产部署是否必须 Kubernetes/Helm，还是 Compose/Nginx 即可？
8. 是否保留工作流 Data Scope 之外的审批/会签/转交能力？

## 5. Do Not Modify Without Explicit Approval

- 已发布数据库迁移的历史内容。
- 生产数据库、生产配置、生产密钥。
- 旧数据导入结果和生产切换脚本。
- 现有用户/业务数据。
- 未确认归属的旧兼容路由。
- 外部 Provider 的凭据和回调地址。

## 6. Residual Verification Gaps

- 未执行完整 React 视觉回归，因为 React 页面尚未实现。
- 未执行真实 S3/MinIO/OSS、短信、邮件、支付、Push、社交 OAuth 认证。
- 未执行 UniApp 真机/微信工具链 E2E。
- 未执行生产数据切换。
- 未验证生产 Prometheus 独立只读认证。
- 未验证所有页面在真实 Go API 下的空/错误/403/409/超时分支。

这些缺口必须保留在最终交付报告中，不能以“已有接口”或“构建通过”替代。
