# 01 Project Inventory

## 1. Workspace Inventory

当前工作区：

```text
C:\Users\admin\Desktop\admin\
├── mars-admin-go/          # 目标 Go 后端，已独立 Git 仓库
├── mars-admin-master/      # 旧 Java/MySQL/Vue/UniApp 参考源码
└── scaffold-master-main/   # 前端参考快照，含 scaffold-vue 与 scaffold-system
```

`mars-admin-master` 和 `scaffold-master-main` 没有 `.git`，不能把它们当作可比较分支。前端参考的准确基线使用 GitHub `KernelC1130/scaffold-master` 提交 `e1488bf`；Go 后端基线使用当前仓库提交 `8c5ff1d`。

## 2. Frontend Reference

### Structure

| 路径 | 职责 | 证据 |
| --- | --- | --- |
| `scaffold-vue/src/views` | 19 个页面组件 | 递归文件清点 |
| `scaffold-vue/src/components/ui` | 27 个基础 UI 组件 | 目录清点 |
| `scaffold-vue/src/components/business` | 24 个业务组件 | 目录清点 |
| `scaffold-vue/src/components/layout` | 7 个外壳组件 | 目录清点 |
| `scaffold-vue/src/components/charts` | 6 个自绘图表/图谱组件 | 目录清点 |
| `scaffold-vue/src/components/ai` | 1 个全局 AI 助手 | `AiAssistant.vue` |
| `scaffold-vue/src/components` 根 | `Icon.vue`、`NeuralNet.vue` | 目录清点 |
| `scaffold-vue/src/styles` | 七层 CSS：tokens、base、materials、components、instruments、layout、pages | `main.ts` 引入顺序 |
| `scaffold-vue/src/api` | 统一 HTTP、认证、会话、系统、门户、漏洞接口 | 7 个 TypeScript 文件 |

### Runtime and Build

| 项目 | 事实 | 证据 |
| --- | --- | --- |
| Framework | Vue 3.5.13 | `scaffold-vue/package.json` |
| Build | Vite 6.4.3 resolved | `npm ci` 与 build 输出 |
| Router | Vue Router 4.5 | `package.json`、`router/index.ts` |
| Animation | GSAP 3.15 + GSAP Flip | `package.json`、`composables/motion.ts` |
| Styling | 原生 CSS、CSS Variables、自研组件 | `src/styles`、组件模板 |
| Tailwind | 未安装 | `package.json` 与源码引用检查 |
| Third-party UI kit | 未使用 Ant Design、Element Plus、Naive UI、MUI、shadcn、Radix 等完整 UI 库 | `package.json` |
| State | Vue `ref`/`reactive`/composables；无 Pinia | `src/composables` |
| HTTP | `fetch`，`/api` base URL，`{code,message,data}` envelope | `src/api/http.ts` |
| Session | `Authorization: Bearer <token>`；token 存 `localStorage` 或 `sessionStorage` | `src/api/http.ts`、`session.ts` |
| Tests | API/utils Node tests；无组件测试、无 Playwright 截图测试 | `src/api/__tests__`、`src/utils/__tests__` |
| Build evidence | `npm run test:unit`：22 passed；`npm run build`：通过 | 本次审计命令 |

### Current Backend Coupling

`scaffold-vue/README.md` 明确写的是 Spring Boot 后端。参考仓库中 `scaffold-system` 是实际提供 `/api` 契约的后端；它包含 Spring Boot、MyBatis、MySQL、Redis、JWT 等技术，不应成为新 Go 平台的运行时依赖。

## 3. Go Backend

### Structure and Scale

| 项目 | 数量/事实 | 证据 |
| --- | --- | --- |
| Go packages | 88 | `go list ./...` |
| Go files | 163 | 递归清点 |
| Goose migrations | 23，编号 `000001` 至 `000023` | `db/migrations` |
| SQL query files | 11 | `db/queries` |
| OpenAPI paths | 100（Phase 1 后） | `api/openapi/v1/openapi.yaml` |
| OpenAPI operations | 135（Phase 1 后） | `operationId` 清点 |
| Entrypoints | API、Worker、Migrate、Legacy Import、Outbox | `cmd/*` |
| Domain modules | Identity、Access、Organization、System、Audit、Monitoring、Storage、Messaging、Notification、Scheduler、Workflow、Codegen、Integrations、Outbox | `internal/modules` |
| Platform | Config、Database、Cache、Logging、Telemetry、HTTP、Security、Request Context、Outbox、Shutdown | `internal/platform` |

### Fixed Stack

| 领域 | 选型 | 证据 |
| --- | --- | --- |
| Go | `go 1.27.0`，`toolchain go1.27.3` | `go.mod` |
| HTTP | `net/http` + `go-chi/chi/v5` 5.2.5 | `go.mod` |
| Database | PostgreSQL 18.6、pgx 5.12.0 | `go.mod`、Compose |
| SQL generation | sqlc 1.30.0 | `go.mod`、`sqlc.yaml` |
| Migration | Goose 3.29.0 | `go.mod`、`internal/platform/database` |
| Cache/rate limit | Redis 8.10.1、go-redis 9.23.0 | `go.mod`、Compose |
| Background jobs | River 0.48.0 | `go.mod`、`internal/modules/*/adapter/river` |
| WebSocket | coder/websocket 1.8.15 | `go.mod`、`messaging/realtime` |
| API contract | OpenAPI 3.1 + oapi-codegen 2.8.0 | `api/openapi`、`api/oapi-codegen.yaml` |
| Observability | slog、Prometheus、OpenTelemetry | `internal/platform` |
| Deployment | Multi-stage Docker、Compose、Helm | `Dockerfile`、`compose.yaml`、`deploy/helm` |

## 4. Existing Audit Documents

Go 仓库已有可复核的专题文档：

- `docs/analysis/`：旧项目、功能、API、数据库、认证、集成、工作流、前端契约。
- `docs/database/`：旧库盘点、表映射、类型映射、关系映射和导入验证。
- `docs/go-standard-audit/`：Go 结构、依赖、数据库、并发、安全、测试和整改。
- `docs/audit-round-2/`、`docs/audit-round-3/`、`docs/audit-final/`：Refresh、Outbox、多 Worker、权限和运行验证。
- `docs/api/frontend-compatibility.md`：canonical React 契约、已删除兼容入口和离线导入边界。

本目录不是对上述证据的替代，而是把前端参考、Go 后端、契约缺口和目标架构放进同一套可执行路线。

## 5. Build and Run Inventory

### Frontend

```powershell
cd C:\Users\admin\Desktop\admin\scaffold-master-main\scaffold-vue
npm ci
npm run test:unit
npm run build
npm run dev
```

### Go API

```powershell
cd C:\Users\admin\Desktop\admin\mars-admin-go
go test ./...
go vet ./...
docker compose up --build
```

### Go Verification

```powershell
go test -race ./...
go test -tags=integration ./... -count=1
golangci-lint run
govulncheck ./...
go tool sqlc generate
go tool oapi-codegen -config api/oapi-codegen.yaml api/openapi/v1/openapi.yaml
```

## 6. Project Status

| 工作流 | 状态 |
| --- | --- |
| Vue 源码完整盘点 | 已完成 |
| Vue 组件/样式/动画盘点 | 已完成 |
| Go 模块/数据库/认证盘点 | 已完成 |
| OpenAPI 基础结构 | Phase 1 canonical 契约已补强；产品范围缺口仍保留 |
| React/Next.js 工程 | 未开始 |
| React 页面与真实 Go API 联调 | 未开始 |
| 视觉截图回归 | 未开始 |
| 生产数据切换 | 未批准、未执行 |
| 外部 Provider 认证 | 依赖真实凭证，未认证 |

结论：当前项目具备审计和分阶段实施基础，但不具备“Vue 到 React 产品级重构完成”的验收条件。
