# 09 Target Architecture

> Historical target design. The current Phase 4 scope excludes Asset Management, Vulnerability
> Management and Remediation Work Orders. The `security/*` paths below are not target routes.

## 1. Product Shape

目标是一个同源部署的管理平台：

```text
Browser
  |
  +-- Next.js App Router (React UI)
  |
  +-- /api/v1/* -> Go API
  |
  +-- /ws/*     -> Go WebSocket
```

前端以 `scaffold-vue` 的视觉、布局、组件和动效为唯一还原基准；后端以现有 `mars-admin-go` 为业务基础，不再为旧 Java 运行时保留新领域模型。

## 2. Frontend Stack

| 领域 | 选型 |
| --- | --- |
| Framework | Next.js App Router + React |
| Language | TypeScript strict |
| Styling | Tailwind CSS + CSS Variables + preserved custom CSS |
| Animation | GSAP 按需 + CSS Animation/Transition；WAAPI 仅在明确必要时 |
| Icons | 现有图标资源或统一图标库 |
| Unit | Vitest |
| E2E | Playwright |
| Package manager | pnpm |
| API types | OpenAPI 生成，不手写漂移类型 |

版本原则：使用当前稳定且相互兼容的 Next.js/React/Tailwind 版本，在 `package.json` 和 lockfile 中固定；升级必须通过构建、E2E 和视觉回归。不得为了“新”而引入第二个 UI 框架。

## 3. Frontend Directory

```text
frontend/
├── src/
│   ├── app/
│   │   ├── (auth)/login/page.tsx
│   │   ├── (admin)/layout.tsx
│   │   ├── (admin)/dashboard/page.tsx
│   │   ├── (admin)/system/users/page.tsx
│   │   ├── (admin)/system/roles/page.tsx
│   │   ├── (admin)/system/menus/page.tsx
│   │   ├── (admin)/system/departments/page.tsx
│   │   ├── (admin)/system/positions/page.tsx
│   │   ├── (admin)/system/dictionaries/page.tsx
│   │   ├── (admin)/system/settings/page.tsx
│   │   ├── (admin)/system/notices/page.tsx
│   │   ├── (admin)/monitor/jobs/page.tsx
│   │   ├── (admin)/monitor/operation-logs/page.tsx
│   │   ├── (admin)/monitor/login-logs/page.tsx
│   │   ├── (admin)/codegen/page.tsx
│   │   ├── forbidden/page.tsx
│   │   ├── not-found.tsx
│   │   ├── layout.tsx
│   │   └── globals.css
│   ├── components/
│   │   ├── ui/
│   │   ├── business/
│   │   ├── layout/
│   │   ├── charts/
│   │   └── visual/
│   ├── features/
│   │   ├── auth/
│   │   ├── users/
│   │   ├── access/
│   │   ├── organization/
│   │   ├── dictionaries/
│   │   ├── settings/
│   │   ├── notices/
│   │   ├── monitoring/
│   │   ├── codegen/
│   ├── lib/
│   │   ├── api/
│   │   ├── auth/
│   │   ├── permissions/
│   │   └── motion/
│   ├── styles/
│   └── types/
├── public/
├── tests/
├── package.json
├── next.config.ts
└── tsconfig.json
```

目录可以由真实业务调整，但不能把所有页面写成一个大 Client Component，也不能使用第三方完整 UI 库替换自研组件。

## 4. Server/Client Boundaries

### Server Components

- 根布局、元数据、字体。
- 非交互页面骨架。
- 读取 Cookie 后的服务端权限判断和菜单注入（只做展示初判，真正授权仍在 Go）。
- OpenAPI 生成类型的服务端请求。

### Client Components

- 表格、筛选、分页、CRUD 表单。
- GSAP、Canvas、WebGL、ResizeObserver、Pointer Events。
- 主题/密度/localStorage。
- Drawer/Modal/Select/Toast 的焦点栈。
- 图谱、看板、组织画布、代码预览。

禁止：

- 在 Server Component 访问 `window`、`document`、`localStorage`。
- 为了省事把整个 admin layout 标成 Client Component。
- 在浏览器 localStorage 保存长期 token 或明文密码。
- 用 Next.js Middleware 代替 Go 后端鉴权。

## 5. Auth and Session

目标浏览器流程：

1. `POST /api/v1/auth/login`。
2. Go 设置 HttpOnly `mars_session`、refresh cookie 和 CSRF cookie。
3. 浏览器不读取 access/refresh token。
4. unsafe 请求自动带 Cookie 和 `X-CSRF-Token`。
5. Go 校验 Session、CSRF、权限和数据范围。
6. 401 统一清理 UI 状态并跳登录；403 显示无权限。
7. 管理员强制下线后，下一次请求失败。

App/UniApp 可以使用受控 Bearer，但必须使用独立客户端策略和 Token 生命周期。

## 6. Permission and Data Scope

### RBAC

- 用户、角色、权限、菜单、按钮、API 权限统一。
- 菜单可见性不等于业务授权。
- 超级管理员是数据库属性，不是用户名约定。
- 权限变更后失效缓存；相关用户下一次请求或登录生效。

### Data Scope

至少支持：

- 全部。
- 本人。
- 本部门。
- 本部门及下级。
- 指定部门。
- 自定义范围。

Data Scope 必须下推到 SQL，不得在 Handler 取全量后过滤。Workflow、审计等新读接口也要显式接入，不能只给用户列表做。

## 7. Go Backend Target

保留现有模块化单体。资产、漏洞和修复工单不新增模块。若产品明确保留以下能力，再新增：

- `assistant`（如保留 AI 助手）：会话、知识库、模型 Provider、权限。
- `payment`/`billing`（如保留支付）：订单、支付尝试、回调账本、幂等键、金额校验。

模块内部继续采用 application/domain/adapter，不为简单 CRUD 机械造层。

### API Rules

- `/api/v1/*` 是唯一新契约。
- Request/Response 必须有完整 schema。
- 错误统一 RFC 9457。
- 分页统一 `page/pageSize/items/total`。
- ID 统一 UUID 字符串。
- 排序/过滤白名单。
- 权限在路由/服务端执行。
- 文件上传/下载独立 content type。
- WebSocket 消息协议单独文档化。

### Database Rules

- PostgreSQL 是唯一业务主库。
- Goose 迁移，新增不修改已发布历史迁移。
- sqlc 查询按模块拥有，跨模块只能通过公开端口/read model。
- 关键状态机使用事务和 CAS/行锁。
- 业务数据与 Outbox 在同一事务提交。
- 敏感配置只存密文。
- 支付回调需要订单状态账本和 replay protection。

## 8. Deployment

推荐同源：

```text
Nginx
  /            -> Next.js
  /api/v1/*    -> Go API
  /ws/*        -> Go WebSocket
```

部署要求：

- Docker multi-stage；非 Root；只读根文件系统。
- Compose 本地 PostgreSQL/Redis/API/Worker。
- Helm 作为生产 Kubernetes 选项，不是本地开发前置。
- 数据库迁移独立命令。
- 健康检查 `/health/live`、`/health/ready`。
- `/metrics` 采用独立只读监控认证，不能匿名暴露。
- 备份、恢复、密钥轮换、日志脱敏有文档。

## 9. Quality Gates

### Frontend

- `tsc --noEmit`。
- ESLint。
- Vitest。
- Playwright E2E。
- 截图回归。
- 真实 API contract test。
- 权限、Session 过期、CSRF、文件上传/下载。

### Backend

- `gofmt`、`go vet`、`go test`、race、integration。
- `golangci-lint`、`govulncheck`。
- sqlc/oapi-codegen 生成一致。
- 数据库迁移升级/回滚验证。
- 并发、事务、幂等、安全测试。

### Visual

- 浅色/深色。
- regular/compact。
- 桌面/平板/移动。
- reduced motion/reduced transparency。
- 登录、主布局、主要业务页、表格、表单、弹窗、抽屉。

任何未通过真实 API、截图和权限分支的功能不得标记完成。
