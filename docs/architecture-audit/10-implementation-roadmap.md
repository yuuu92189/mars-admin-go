# 10 Implementation Roadmap

## 1. Guiding Rules

- 先契约，后页面。
- 先审计，后重构。
- 先真实 API，后视觉联调。
- 不把旧 envelope、旧权限码和旧 Java 结构带进 React/Go 新领域。
- 不用静态 mock 页面代替真实业务。
- 不以减少代码量为理由牺牲视觉还原。
- 不执行未批准的破坏性数据库操作。

## 2. Phase Status

| 阶段 | 状态 | 交付 |
| --- | --- | --- |
| Phase 0 审计 | 已完成 | 本目录 11 份审计文档、Go 既有专题报告 |
| Phase 1 契约基线 | 已完成已确认技术范围；P1-07/P1-08 待产品决策 | OpenAPI/权限/ID/分页/错误统一、Go 治理、TypeScript 生成 |
| Phase 2 前端基础 | 未开始 | Next.js、Tailwind、CSS 令牌、UI 原语 |
| Phase 3 认证与外壳 | 未开始 | 登录、Cookie/CSRF、动态菜单、Admin layout |
| Phase 4 核心系统页 | 未开始 | 工作台、用户、角色、菜单、部门、岗位、字典、配置、通知 |
| Phase 5 监控/任务/代码生成 | 未开始 | 日志、任务、生成器 |
| Phase 6 漏洞业务 | 未开始 | 资产、漏洞、修复工单、统计 |
| Phase 7 实时/工作流/文件 | 未开始 | 消息、通知推送、WebSocket、工作流、文件 |
| Phase 8 验收 | 未开始 | E2E、截图、性能、安全、可访问性 |
| Phase 9 部署/迁移 | 未开始 | 生产部署、数据切换、Provider 认证 |

## 3. Phase 1: Contract Baseline

### Phase 1 Actual Status

| Task | Status | Evidence |
| --- | --- | --- |
| P1-01 Metrics security | 已实现、已测试、已验证 | OpenAPI + integration test |
| P1-02 Login contract | 已实现、已测试 | canonical DTO + legacy adapter tests |
| P1-03 Permission catalog | 已实现、已测试 | permission catalog + route/seed tests |
| P1-04 UUID/pagination/error | 已实现、已测试 | HTTP boundary tests |
| P1-05 OpenAPI schema | 已实现关键契约；所有需要响应体的 2xx 有 content | 100 paths / 135 operations |
| P1-06 Group management | 已实现、已测试 | OpenAPI + contract test |
| P1-07 Vulnerability API | 待产品决策 | no domain module |
| P1-08 AI assistant API | 待产品决策 | no provider decision |
| Cross-module sqlc | 已完成已知两项 | dependency test |
| Workflow Data Scope | 已实现、已测试 | PostgreSQL integration test |
| SQLSTATE | 已实现、已测试 | HTTP error tests |
| Repository interfaces | Access/Workflow 已拆分；Identity/Notification 待后续 | application interfaces |
| Legacy stubs | 已处理已确认项 | 501 Problem or real statistics |
| TypeScript generation | 已实现、已测试 | `frontend-api` + 0 `any` |

### Tasks

| ID | 任务 | 文件范围 | 依赖 | 验收 |
| --- | --- | --- | --- | --- |
| P1-01 | 修正 `/metrics` OpenAPI 与实现冲突 | `api/openapi/v1/openapi.yaml`, `internal/bootstrap/app.go`, integration tests | 监控认证决策 | 匿名 401/受权 200 与文档一致 |
| P1-02 | 统一登录字段和响应 | `api/openapi`, identity HTTP adapter, generated TS | 验证码/过期时间产品决策 | 登录、验证码、expiresAt、错误分支通过 |
| P1-03 | 统一权限码迁移表 | `db/migrations/000007`, Access, OpenAPI, frontend mapping | 角色/菜单设计 | 页面按钮与 Go Handler 同码 |
| P1-04 | 统一 UUID/分页/错误 | OpenAPI schemas, generated types, compat adapter | 前端迁移方案 | 无 `Number(uuid)`、分页契约一致 |
| P1-05 | 补齐页面实际消费的 OpenAPI schema | OpenAPI + generated TS | 页面清单 | 无 `additionalProperties: true` 的业务主模型 |
| P1-06 | 补齐群组管理 operation | OpenAPI + messaging handler tests | 消息权限模型 | 路由与文档逐条一致 |
| P1-07 | 设计漏洞领域 API/schema | OpenAPI + migration plan + module design | 业务确认 | 资产/漏洞/工单/统计契约可生成 TS |
| P1-08 | 设计 AI assistant API（如保留） | OpenAPI + provider contract | 模型/知识库供应商 | 未配置返回 typed error |

### Gate

- OpenAPI validation、operationId uniqueness、schema completeness。
- 前端从 OpenAPI 生成 TypeScript 不出现 `any` 泛滥。
- 关键页面 API 映射无“目标不存在”项。

## 4. Phase 2: Frontend Foundation

### Tasks

- 初始化 `frontend/` Next.js App Router、TypeScript strict、Tailwind、pnpm、Vitest、Playwright。
- 复制并分层整理七层 CSS；CSS Variables 作为唯一设计令牌。
- 迁移 `focus.ts`、`confirm`、`toast`、`list`、`dict`、`dept`、`appearance` 等 composable 逻辑为 React hooks/stores。
- 迁移 27 个基础 UI 原语；先保证 Button/Input/Select/Checkbox/Switch/Modal/Drawer/DataTable/Pagination/Field。
- 建立组件 Story/示例和键盘/焦点测试。

### Gate

- 基础组件视觉与原 Vue 一致。
- reduced motion、深色、compact、窄屏通过。
- 没有引入第三方完整 UI 库。

## 5. Phase 3: Auth and Shell

### Tasks

- 登录页一比一还原：品牌面板、验证码、表单、错误、成功动画。
- 删除明文密码持久化；只允许非敏感账号提示或使用浏览器密码管理器。
- Go Cookie/CSRF 接入；刷新/过期/强制下线。
- 动态菜单、权限上下文、命令面板、通知铃、外观/密码弹窗。
- Admin layout 的桌面/移动侧栏、顶栏、页面切换。

### Gate

- 登录、退出、Session 过期、401/403。
- 动态菜单和按钮权限。
- 登录页/外壳浅色深色截图。

## 6. Phase 4: Core System Pages

顺序：

1. 工作台。
2. 用户。
3. 角色。
4. 菜单。
5. 部门。
6. 岗位。
7. 字典。
8. 配置。
9. 通知。

每页必须包含：

- 查询、重置、分页。
- 新增/编辑/删除。
- 批量操作（如有）。
- 权限按钮。
- 加载/空/错误/无权限。
- 导出（如有）。
- 替代视图（如有）。
- 截图与 E2E。

## 7. Phase 5: Monitoring, Scheduler, Codegen

- 操作日志：统计、趋势/泳道、详情、导出、删除/清空。
- 登录日志：统计、趋势/热力、异常、导出。
- 定时任务：CRUD、暂停/恢复、立即执行、cron 向导、日志。
- 代码生成：导入表、字段映射、代码预览、ZIP。

Gate：真实数据分页、时间线/热力、任务执行记录和下载不靠 mock。

## 8. Phase 6: Vulnerability Domain

如果保留 `scaffold-vue` 的安全业务：

- 新增 Go `vulnerability` 模块和 PostgreSQL 表。
- 资产、漏洞、修复工单状态机。
- 统计 summary/风险/月度/TOP。
- 资产详情关联漏洞、漏洞详情关联工单、工单接单/修复/复测/关闭。
- 数据权限和操作审计。
- React 三页及工作台安全态势。

Gate：

- 状态机并发测试。
- 资产/漏洞/工单关联一致性。
- 风险等级自动计算与权限。
- 三页 E2E 和截图。

## 9. Phase 7: Realtime, Workflow, Files

- 私聊/群聊、消息补拉、已读、幂等。
- WebSocket Cookie/Origin 认证，禁用 query token。
- 通知发送、Provider 重试、发送日志。
- 文件上传/下载/预览/删除。
- Workflow 定义、实例、待办、已办、终止、审批、Data Scope。

Gate：

- 断线重连和历史补拉。
- Outbox 重试不重复业务记录。
- Provider 失败可观测。
- Workflow 并发审批/终止。

## 10. Phase 8: Acceptance

### Functional

- 全部页面真实 API。
- 登录到退出完整流程。
- 权限拒绝、Session 过期、CSRF。
- 文件上传下载。
- WebSocket 重连。
- 所有 CRUD/批量/导出。

### Visual

- 浅色/深色。
- regular/compact。
- 桌面/平板/移动。
- 登录、主布局、主要业务页、表格、表单、弹窗、抽屉。
- 截图回归，不接受“肉眼差不多”。

### Engineering

- TypeScript strict。
- ESLint、Vitest、Playwright。
- Go test/vet/race/integration/lint/vulncheck。
- OpenAPI/codegen 一致性。
- Docker/Compose/Helm smoke。

## 11. Phase 9: Deployment and Data Cutover

- 同源 Nginx/反向代理。
- 生产 Cookie Secure/SameSite/CSRF。
- Prometheus 独立只读账号。
- 数据备份、恢复、迁移报告批准。
- Provider 密钥轮换和真实认证。
- 生产灰度/回滚方案。

## 12. Priority and Dependencies

| 优先级 | 工作 |
| --- | --- |
| P0 | 目标架构/契约/安全基线；不完成不进入大规模页面实现 |
| P1 | 认证、权限、核心系统页、漏洞领域、真实 API 联调 |
| P2 | 工作流 Data Scope、跨模块 sqlc 依赖、覆盖率、legacy stub 清理 |
| P3 | 性能优化、动画细节、运维体验 |

## 13. Explicitly Not Allowed

- 在审计/契约未完成时生成大量静态 React 页面。
- 用 `any` 或手写重复 API 类型绕过 OpenAPI。
- 把长期 token/密码放 localStorage。
- 用第三方完整 UI 库替换自研组件。
- 用假数据冒充真实业务。
- 未备份/未验证执行数据库迁移。
- 宣称外部 Provider 严格 exactly-once。
