# 07 Legacy Compatibility Removal

> Historical compatibility plan. Raw Authorization and WebSocket query-token acceptance was
> removed in the Phase 4 follow-up; legacy clients must now use Bearer or session cookies.

## 1. Boundary

兼容层分为两类，不能混为一谈：

1. **在线兼容 API**：`internal/platform/compat` + `internal/bootstrap/app.go` 中的 `/api/*` 路由。
2. **离线数据迁移工具**：`internal/migration/legacy` 和 `cmd/legacy-import`。

在线兼容 API 可以在前端迁移完成后删除；离线迁移工具在没有完成数据审计和回滚方案前不能删除。

## 2. Online Compatibility Inventory

| 兼容入口 | 当前作用 | 新替代 | 删除前置条件 |
| --- | --- | --- | --- |
| `/api/auth/*`、`/api/app/auth/*`、`/api/web/auth/*` | 旧登录、验证码、资料、改密 | `/api/v1/auth/*` + Cookie/CSRF | 所有浏览器和 UniApp 客户端迁移 |
| `/api/sys/user/*` | 旧用户分页、CRUD、在线用户 | `/api/v1/admin/users` | 前端用户页与新权限码联调 |
| `/api/sys/role/*` | 旧角色 CRUD | `/api/v1/admin/access/roles` | 角色页与权限矩阵迁移 |
| `/api/sys/menu/*` | 旧菜单树/CRUD | `/api/v1/admin/access/menus` | 动态菜单与新权限码对齐 |
| `/api/sys/dept/*`、`/api/sys/post/*` | 旧组织/岗位 | `/api/v1/admin/organization/*` | 组织画布/岗位阶梯迁移 |
| `/api/sys/dict/*` | 旧字典类型/数据 | `/api/v1/admin/dictionaries` | 字典页与缓存失效验证 |
| `/api/sys/config-group/*`、`/api/sys/file-config/*` | 旧参数/文件配置 | `/api/v1/admin/settings/*` | 配置页与敏感设置加密验证 |
| `/api/sys/file/*`、`/api/sys/file-group/*` | 旧文件/文件组 | `/api/v1/files` | 上传下载、私有访问、前端文件页迁移 |
| `/api/sys/chat/*`、`/api/chat/group/*` | 旧私聊/群聊 | `/api/v1/messages`、`/api/v1/groups` | 新客户端使用 Cookie/新消息协议 |
| `/api/sys/notice/*` | 旧公告/通知 | `/api/v1/notices` | 通知中心与分页/read-all 迁移 |
| `/api/monitor/*` | 旧日志、服务监控、缓存、在线用户 | `/api/v1/admin/audit/*`、`/api/v1/admin/monitoring/*` | 监控页面和 Prometheus 认证方案确认 |
| `/api/workflow/*` | 旧 Warm-Flow 流程 API | `/api/v1/workflows` | 工作流前端与 JSONB 定义转换 |
| `/api/tool/gen/*` | 旧代码生成 | `/api/v1/admin/codegen/*` | 代码生成页面与新表结构迁移 |
| `/ws/message?token=...` | 已拒绝；旧客户端必须迁移 | `/ws/message` + Cookie/Bearer/Origin 校验 | 已完成后端关闭 |
| `{code,message,data}` envelope | 旧 HTTP 包装 | 直接 JSON + RFC 9457 Problem | 旧 Vue/UniApp 客户端全部迁移 |
| HTTP 200 包装应用错误 | 旧业务错误兼容 | 正确 HTTP 状态码 | 旧客户端错误处理迁移 |

## 3. Envelope Behavior

`internal/platform/compat/envelope.go` 的职责：

- 仅拦截 `/api/*` legacy 路由。
- JSON 响应包装为 `{code,message,data}`。
- 非 JSON 响应直接流式透传，避免文件下载全量缓冲。
- legacy 错误可能映射成 HTTP 200 包装错误；新 `/api/v1` 必须使用 RFC 9457 和正确状态码。

迁移规则：

- 新领域/Application 层禁止引用 legacy envelope 类型。
- 兼容映射只放在 HTTP adapter。
- 每个兼容路由必须有对应 contract test；删除前先让测试失败，证明没有调用者。

## 4. Legacy Runtime Compatibility Risks

### High

- `scaffold-vue` 当前仍使用 `/api` envelope 和 Bearer token 存储。
- `scaffold-vue` 登录页把明文密码写入 `localStorage`（`views/Login.vue`），必须在 React 版本删除。
- 旧客户端必须迁移到 Bearer 或 Cookie；WebSocket query token 已被拒绝。
- 旧权限码与新 Go 权限码不一致，不能只删除兼容路由。

### Medium

- 旧分页形状 `{rows,pageNum,pageSize,total}` 与新 `{items,page,pageSize,total}` 并存。
- 旧数字 ID 与新 UUID 字符串并存。
- 旧 Vue/UniApp 期望 `msg`/`message` 两种错误字段。
- 旧监控页面把 Go runtime 映射成 JVM 字段，容易误导运维。

### Low / Controlled

- legacy import 是一次性离线工具，不暴露在线业务 API。
- `cmd/outbox` 是受控运维 CLI，不应作为 HTTP API 暴露。

## 5. Stubs and Placeholders

以下兼容行为不能继续假装成完整业务：

| 位置/行为 | 当前事实 | 处理 |
| --- | --- | --- |
| `legacySchedulerStatistics` | 返回固定全零统计 | 实现真实统计或明确 not-configured |
| `legacyCleanLogs` | 空操作 204 | 实现清理或返回明确未支持 |
| `legacyNoFiles` | 返回空结果 | 实现真实文件预览/删除语义 |
| `legacyFileConfigTest` | 存储连通性测试未配置 | 返回 typed not-configured，不伪造成功 |
| `legacyRefreshSettings` | 刷新行为需与真实 settings 对齐 | 实现或删除兼容入口 |
| `legacyEmptySMSLogs` | 固定空 SMS 日志 | 接入真实发送日志 |
| 旧监控 JVM 字段 | Go runtime 映射为 JVM 形状 | 仅兼容期保留，新监控使用 Go 字段 |

## 6. Offline Import Boundary

`internal/migration/legacy` 的职责：

- 读取旧 MySQL dump/表数据。
- 映射旧表、ID、权限、部门、消息和工作流。
- 输出行数、关联、失败和幂等报告。
- 不进入在线 API、Worker 和普通 Handler。

当前已有隔离数据库导入证据，但生产切换尚未批准。生产切换至少需要：

1. 数据所有者批准字段映射和排除项。
2. 旧数据备份与恢复演练。
3. 密钥/密码/API Secret 轮换。
4. 迁移前后行数、外键、权限、金额/状态校验。
5. 回滚或双写窗口方案。

## 7. Removal Sequence

1. 建立每个兼容路由的调用者清单和 contract test。
2. React/UniApp 客户端迁移到 `/api/v1` 和 Cookie/CSRF。
3. 把旧权限码映射到新权限码并验证 403 分支。
4. 修复 ID/分页/错误字段差异。
5. query-token WebSocket 已关闭；legacy envelope 仍待旧客户端迁移后关闭。
6. 删除在线兼容 handler 和 route mounts。
7. 保留离线 importer，直到生产数据切换完成并归档。

在步骤 1-4 未完成前，直接删除兼容层会破坏现有客户端，不能以“新架构更干净”为理由执行。
