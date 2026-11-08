# 08 API Contract Mapping

## 1. Contract Layers

当前存在三套相关契约：

| 层 | 路径/来源 | 形状 | 状态 |
| --- | --- | --- | --- |
| 参考前端 | `scaffold-vue/src/api/*` | `/api/*`、`{code,message,data}`、数字 ID、`{rows,pageNum,pageSize,total}` | 当前页面实际调用 |
| 参考 Spring Boot | `scaffold-system` | 参考前端对应的后端 | 仅作为参考，不进入目标运行时 |
| Go 目标后端 | `api/openapi/v1/openapi.yaml` + `/api/v1/*` | 直接 JSON、RFC 9457、UUID、`{items,page,pageSize,total}` | 已实现，但 schema/权限元数据不完整 |

目标不是让 Vue 页面继续依赖旧 envelope，而是：

1. 先定义统一 OpenAPI。
2. 生成 TypeScript 类型/client。
3. React 页面只调用新 API。
4. 旧 `/api/*` 仅在迁移期服务旧客户端。

## 1.1 Phase 1 Contract Update

Current canonical OpenAPI:

- 99 paths / 134 operations.
- Every 2xx response has a content schema.
- 110 operations declare `x-required-permission`.
- 12 authenticated-only operations declare `x-auth-required`.
- `/metrics` inherits bearer/cookie security and requires `audit:read`.
- Canonical login uses `captchaId`, `captchaCode` and `expiresAt`; legacy aliases are adapter-only.
- Canonical pagination is `items/total/page/pageSize`.
- Group member removal, role update, mute and owner transfer are now documented.
- Generated TypeScript types live in `frontend-api/src/schema.ts` and contain 0 `any` tokens.

The canonical response shape is now stable enough for React page work, but product-scoped pages such as vulnerability and payment remain absent.

## 2. Page-to-API Matrix

| 页面 | 业务能力 | Go 模块 | 当前 Vue API | 目标 Go API | 状态/缺口 |
| --- | --- | --- | --- | --- | --- |
| 登录 | 验证码、密码登录、当前用户、动态路由 | Identity + Access | `/auth/captcha`, `/auth/login`, `/auth/me`, `/auth/routes` | `/api/v1/auth/captcha`, `/login`, `/me`, Access menus | 契约已固化；旧 Vue 仍需适配新字段/Cookie |
| 工作台 | 统计、趋势、操作泳道、安全态势 | Monitoring + Audit + Vulnerability | `/dashboard/summary`, `/monitor/oper-logs/timeline`, `/vuln/statistics/*` | `/api/v1/admin/monitoring/dashboard`, `/api/v1/admin/audit/*`; vulnerability 新增 | Dashboard schema 和漏洞统计未完成 |
| 用户 | 用户 CRUD、状态、密码、角色/部门/岗位选项、导出 | Identity + Organization + Access | `/system/users*`, `/system/permissions/graph` | `/api/v1/admin/users*` | UUID/数字 ID、分页、权限码、选项字段差异 |
| 角色 | 角色 CRUD、菜单授权、数据范围、权限矩阵 | Access | `/system/roles*`, `/system/menus/tree` | `/api/v1/admin/access/roles*`, `/menus` | 旧 `system:role:*` 与新 `role:*`；矩阵批量写入需契约 |
| 菜单 | 菜单树、CRUD、显示状态、权限图 | Access | `/system/menus*` | `/api/v1/admin/access/menus*` | 旧组件标识/路径与新菜单字段需映射 |
| 部门 | 部门树、CRUD、组织画布、成员预览 | Organization | `/system/depts*`, `/system/users?deptId=` | `/api/v1/admin/organization/departments*` | 部门字段、ID、Data Scope 响应需统一 |
| 岗位 | 岗位 CRUD、成员、阶梯视图 | Organization | `/system/posts*`, `/system/users?postId=` | `/api/v1/admin/organization/positions*` | 旧 `postCode/postName` 与新 `code/name` 需映射 |
| 字典 | 类型/数据 CRUD、选项缓存 | System | `/system/dict/*` | `/api/v1/admin/dictionaries*` | 旧 `dictType/dictName` 与新 code/name；响应 schema 缺失 |
| 系统配置 | 参数分组、批量更新、AI 测试、知识库 | System + Integrations | `/system/configs*`, `/vuln/ai/*` | `/api/v1/admin/settings*`; AI 端点新增 | 旧参数列表与 setting group 结构不同 |
| 通知 | 我的通知、发送记录、已读、发送、删除 | Notification | `/notices/*` | `/api/v1/notices*` | 旧分页/收件人字段与 Go 响应需映射；OpenAPI schema 缺失 |
| 定时任务 | 任务 CRUD、暂停/恢复、立即执行、cron、日志 | Scheduler | `/monitor/jobs*`, `/monitor/job-logs`, `/monitor/cron-check` | `/api/v1/admin/scheduler/jobs*` | handler 命名/状态字段、cron 校验契约需统一 |
| 操作日志 | 列表、统计、泳道、导出、删除/清空 | Audit + Monitoring | `/monitor/oper-logs*` | `/api/v1/admin/audit/events`, `/logins`; 统计端点需补齐 | 新 audit 只有 events/logins，缺少旧 stats/timeline 等价 schema |
| 登录日志 | 列表、统计、热力、异常、导出、删除/清空 | Audit | `/monitor/login-logs*` | `/api/v1/admin/audit/logins`; 统计/热力端点需补齐 | 当前 OpenAPI 仅 login page，不覆盖 heatmap/stats |
| 代码生成 | 表导入、字段、预览、生成、ZIP | Codegen | `/tool/generator/*` | `/api/v1/admin/codegen/*` | 旧生成器模型（Java/Vue）与新 Go/OpenAPI 生成模型不同 |
| 资产 | 资产 CRUD、详情、导出、关联漏洞 | 目标 Vulnerability | `/vuln/assets*` | **Go 当前无等价模块** | P1 功能缺口 |
| 漏洞 | 漏洞 CRUD、详情、关联工单、导出 | 目标 Vulnerability | `/vuln/vulnerabilities*` | **Go 当前无等价模块** | P1 功能缺口 |
| 修复工单 | 工单 CRUD、接单、修复、复测、关闭 | 目标 Vulnerability | `/vuln/repair-tasks*` | **Go 当前无等价模块** | P1 功能缺口 |
| 无权限 | 空状态 | 前端路由 | 无 | 无 | 前端迁移 |
| 未找到 | 404 | 前端路由 | 无 | 无 | 前端迁移 |

## 3. Shared API Gaps

### 3.1 IDs

- Vue 参考：`number` ID。
- Go：`uuid.UUID`，JSON 为字符串。
- 处理：React 类型统一 `string`；所有路由参数、表格 key、批量操作不得 `Number(id)`。

### 3.2 Pagination

```text
Vue reference: { pageNum, pageSize, total, rows }
Go native:     { page, pageSize, total, items }
```

处理：新 API 统一 `page/pageSize/items/total`；兼容层单独映射 `rows/pageNum`。

### 3.3 Permissions

| 领域 | Vue 旧码 | Go 新码 |
| --- | --- | --- |
| 用户 | `system:user:list/add/edit/remove/resetPwd` | `user:read/create/update/delete` |
| 角色 | `system:role:list/add/edit/remove` | `role:read/create/assign` |
| 部门/岗位 | `system:dept:*`, `system:post:*` | `department:read/manage` |
| 字典 | `system:dict:*` | `dictionary:read/manage` |
| 配置 | `system:config:*` | `settings:read/manage` |
| 日志/监控 | `monitor:*` | `audit:read` |
| 代码生成 | `tool:gen:list` | `codegen:read/manage` |
| 漏洞 | `vuln:*` | 尚未定义 |

处理：建立一次性迁移表，页面按钮和 Go Handler 必须引用同一权限码；不能让前端仅隐藏按钮。

### 3.4 Errors

- 新 API：RFC 9457 `application/problem+json`，正确 4xx/5xx。
- 旧 API：envelope，部分应用错误 HTTP 200。
- 处理：React 只处理新 API；错误码、字段错误和 requestId/traceId 显示保持一致。

### 3.5 Auth

- 新浏览器：HttpOnly `mars_session` + refresh cookie + CSRF。
- App/UniApp：受控 Bearer token。
- 旧浏览器：Bearer in storage + WebSocket query token，仅兼容期。
- 处理：React 禁止读取/写入长期 token；CSRF token 由登录响应/独立端点下发。

## 4. OpenAPI Completeness Checklist

每个新 API 必须包含：

- `operationId`。
- tags。
- 认证/权限声明。
- path/query/header 参数。
- requestBody schema。
- 成功响应 schema。
- 错误响应 schema。
- 分页/排序/过滤白名单。
- 文件上传/下载 content type。
- WebSocket 消息协议（在 OpenAPI 之外单独文档化）。

当前只有 operationId 和部分 schema，不能声称“OpenAPI 完整”。

## 5. Page Acceptance Status

| 状态 | 含义 |
| --- | --- |
| `inventory` | 已从 Vue 源码盘点 |
| `contract-gap` | 目标 Go API 或 schema 不完整 |
| `migrated` | React 页面实现且使用真实 API |
| `verified` | 通过 Playwright/E2E/截图/权限/错误分支 |

当前全部页面处于 `inventory` 或 `contract-gap`；没有任何页面达到 `migrated` 或 `verified`。
