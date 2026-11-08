# Architecture Audit

这是 Mars Admin 平台第一轮前后端架构审计的统一入口。审计结论以源码、GitHub 提交和实际命令结果为依据，不以 README 或目录名称代替验证。

## Audit Baseline

| 项目 | 基线 | 证据 |
| --- | --- | --- |
| 前端参考 | `KernelC1130/scaffold-master`，`main`，提交 `e1488bf85935ec19f1ebae89703d151f2feddd3f` | GitHub `git ls-remote`；本地 `scaffold-vue/src` 与精确克隆在规范化换行后逐文件 SHA-256 一致 |
| Go 后端 | `yuuu92189/mars-admin-go`，`main`，提交 `8c5ff1d3440fa565baf716a1e02e850be88b8b77` | `git rev-parse HEAD`，工作树干净，`origin/main` 同提交 |
| 前端技术栈 | Vue 3.5、Vite 6、TypeScript、Vue Router、GSAP 3.15，自有 UI 组件 | `scaffold-vue/package.json` |
| 后端技术栈 | Go 1.27.3、Chi 5.2、pgx 5.12、sqlc 1.30、Goose 3.29、PostgreSQL 18.6、Redis 8.10.1、River 0.48 | `mars-admin-go/go.mod`、`docs/status.md` |

## Documents

| 文档 | 内容 |
| --- | --- |
| [01-project-inventory.md](01-project-inventory.md) | 工作区、仓库、依赖、运行与构建盘点 |
| [02-vue-page-inventory.md](02-vue-page-inventory.md) | 全部 Vue 页面、路由、布局、交互、API、权限和动效 |
| [03-vue-component-inventory.md](03-vue-component-inventory.md) | 全部 Vue 组件、Props、事件、状态、样式和 React 映射 |
| [04-design-system-audit.md](04-design-system-audit.md) | 珍珠玻璃设计系统、令牌、主题、密度和响应式 |
| [05-animation-audit.md](05-animation-audit.md) | GSAP、CSS Animation、Transition、WAAPI 与迁移方案 |
| [06-go-backend-audit.md](06-go-backend-audit.md) | Go 模块、API、数据库、认证、权限、测试和工程问题 |
| [07-legacy-compatibility-removal.md](07-legacy-compatibility-removal.md) | 旧 Java/旧 Vue 兼容层、离线迁移和删除边界 |
| [08-api-contract-mapping.md](08-api-contract-mapping.md) | 页面到 Go API 的逐页映射与缺口 |
| [09-target-architecture.md](09-target-architecture.md) | React/Next.js 与 Go 的目标架构 |
| [10-implementation-roadmap.md](10-implementation-roadmap.md) | 分阶段任务、文件范围、依赖和验收门禁 |
| [11-risk-and-decision-log.md](11-risk-and-decision-log.md) | P0/P1/P2 风险、技术决策和待确认事项 |

## First-Round Status

- Vue 页面、组件、设计令牌、动效和 API 调用已完成源码盘点。
- Go 模块、迁移、OpenAPI、认证、权限、测试和部署已完成源码盘点。
- 目标架构与实施路线已形成。
- React/Next.js 页面尚未实现；本轮没有用静态假页面冒充迁移完成。
- `scaffold-vue` 当前连接的是参考仓库内的 Spring Boot `scaffold-system`，不是 Go 后端。Go 集成仍需独立适配或新 API 实现。
- 当前 Go 后端没有与 `scaffold-vue` 漏洞资产、漏洞和修复工单页面等价的完整领域模块。

## Evidence Commands

以下命令在本次审计中实际执行：

| 命令 | 结果 |
| --- | --- |
| `npm run test:unit`（`scaffold-vue`） | 22 passed，0 failed |
| `npm run build`（`scaffold-vue`） | `vue-tsc` 与 Vite production build 通过 |
| `go test ./...`（`mars-admin-go`） | 通过 |
| `go vet ./...`（`mars-admin-go`） | 通过 |
| `go test -tags=integration ./... -count=1` | 通过，`test/integration` 52.828s |
| OpenAPI validation test | `api/openapi_test.go` 通过；95 paths，130 operations，operationId 唯一 |

更早的 PostgreSQL/Redis/MySQL Testcontainers、Race、lint、漏洞扫描、Compose 和 Helm 记录见 `docs/go-standard-audit/verification.md` 与 `docs/audit-round-3/verification-report.md`。本轮没有把历史记录替代为当前运行态证明；本轮重新执行的命令以上表为准。

## Reading Rule

文档中的“已实现”表示当前代码路径存在；“已验证”表示本次或明确标注的历史证据可复核；“缺口”表示不能用于生产验收或目标功能尚未完成。外部 Provider、生产数据切换和前端视觉回归在没有真实凭证、生产批准或截图基线时不得标记完成。
