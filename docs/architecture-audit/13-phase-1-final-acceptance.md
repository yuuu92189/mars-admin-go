# Phase 1 Final Acceptance

> Historical Phase 1 acceptance record. Phase 4 later excluded Asset Management, Vulnerability
> Management and Remediation Work Orders from the project scope.

## 1. 验收摘要

| 项目 | 值 |
| --- | --- |
| Phase 1 基线 | `c423c3d83882cc5c6c20934a5367d91d1c274152` |
| 本次验收提交 | 本轮工作树提交；最终 Git 信息由交付回复记录 |
| 验收日期 | 2026-11-08 |
| 修改文件数量 | 19（17 个已跟踪文件修改、2 个新增文件） |
| 验收范围 | FA-01 OpenAPI Schema、FA-02 权限全链路、FA-03 Workflow Data Scope、FA-04 文档一致性 |
| 最终结论 | `ACCEPTED WITH CONDITIONS` |

本轮没有重新设计模块、替换数据库、重写认证或提前开发 React 页面。修改集中在契约自动化、权限路由对账、Workflow SQL 隔离、真实 PostgreSQL 测试和审计文档。

## 2. FA-01 OpenAPI Schema 验收

### 检查逻辑

`api/openapi_contract_test.go` 新增全量遍历器，覆盖每个 Path、HTTP Method、Operation、Response 和 Response Media Type。检查器对需要响应体的 2xx 响应要求：

- 声明 `content`。
- 至少存在一个实际 Media Type。
- Media Type 声明可解析的 `schema`。
- `$ref` 目标存在且能被 kin-openapi 解析。
- 组合 Schema 能通过 `Schema.Validate`。
- 失败信息包含 Method、Path、Operation ID、Status 和 Media Type。

204、205 和 HEAD 按 HTTP 无响应体语义跳过；33 个无响应体 2xx 响应被明确统计，而不是被静默忽略。

### 实际覆盖

OpenAPI 当前为：

- 100 paths。
- 135 operations。
- 170 responses。
- 135 个 2xx responses。
- 33 个无响应体 2xx responses。
- 102 个需要响应体的 2xx responses。
- 102 个 Media Type Schema。

实际检查结果：`0` 个成功响应 Schema 问题、`0` 个悬空引用问题。动态 map 仍保留在消息 metadata、setting values、workflow variables 等确属动态的领域，不把动态字段误报为静态主模型。

### 负向测试

新增测试覆盖：

- 成功响应缺少 `content`。
- Media Type 缺少 `schema`。
- `$ref` 指向不存在的组件。
- 合法 204。
- 合法 `oneOf` 组合 Schema。
- 多个 Media Type 中至少一个合法 Schema。
- 非成功响应不被错误纳入成功响应规则。

定向执行：

```text
go test ./api -run TestAllSuccessResponsesHaveValidSchemas -count=1 -v
PASS
```

## 3. FA-02 权限系统验收

### 权威来源

- 实际路由权限：canonical handler 的路由注册和中间件。
- 权限目录：`docs/api/permission-catalog.md`。
- 初始化权限：`db/migrations/000007_seed_authorization.sql`。
- OpenAPI 权限声明：`x-required-permission` 和 `x-auth-required`。

`internal/platform/httpserver/permission_routes_test.go` 使用 Go AST 扫描真实 canonical handler 注册，处理 `RequirePermission`、`RequireAnyPermission`、`Use`、`Group` 和 `With`，并从 OpenAPI 反向查找未注册 operation。

扫描范围是 `/api/v1/*` canonical 路由及明确列出的 `/metrics`、health、payment callback 运维端点。旧 `/api/*` 兼容路由不进入 canonical OpenAPI，仍由独立兼容测试覆盖。

### 对账结果

```text
total=135 public=12 authOnly=12 business=111
```

- 135 个实际 canonical operations 与 OpenAPI 双向匹配。
- 12 个公开路由没有 `security`、`x-required-permission` 或 `x-auth-required`。
- 12 个仅认证路由声明 `x-auth-required`。
- 111 个业务权限路由声明 `x-required-permission`。
- 所有路由使用的权限码都存在于权限目录和 Seed。
- Seed 与权限目录双向一致。
- `RequireAnyPermission` 的 OR 语义由独立 HTTP 测试验证。

OpenAPI 当前有 111 个 `x-required-permission` 和 12 个 `x-auth-required`，与实际路由统计一致。没有通过删除权限声明、放宽权限或跳过测试来消除差异。

## 4. FA-03 Workflow Data Scope 验收

### 实现

Workflow 实例列表、详情和事件读取使用统一可见性谓词，支持：

- `all`
- `self`
- `department`
- `department_tree`
- `custom`
- task assignee
- candidate role

`custom` 范围连接 `access_roles`，要求角色 `status = 'active'` 且 `deleted_at IS NULL`。Candidate role 可见性和待办/已办查询同样校验角色行处于启用且未删除状态，避免调用方传入过期角色码扩大可见范围。

### PostgreSQL 测试矩阵

`test/integration/identity_security_test.go` 使用真实 PostgreSQL 容器验证：

- `self`：发起人可读自己的实例，非发起人不可读。
- `department`：同部门可读，其他部门不可读。
- `department_tree`：当前部门、直属子部门、多级孙部门和兄弟部门边界。
- 部门树结构变化：移动孙部门后可见结果立即变化，恢复后边界恢复。
- `custom`：单部门、多部门、父部门、子部门、空范围和无关部门。
- 不存在的部门 ID 被外键约束拒绝。
- task assignee 可读对应实例。
- candidate role 的真实成员可读实例和待办。
- 非候选角色成员不可读。
- 多角色 actor 仍按 OR 语义工作。
- 角色停用后不再获得候选角色可见性。
- 角色软删除后不再获得候选角色可见性。
- 用户角色关系删除后立即失效。
- custom 角色停用、软删除和成员关系删除后立即失效。
- 列表结果与分页 `total` 使用同一可见性条件。

### Tenant 边界

当前数据库没有 `tenant_id`、tenant 表或租户绑定模型，因此本轮**没有伪造跨租户隔离测试**，也没有把部门隔离宣称为租户隔离。当前实现是单租户系统内的组织、角色和数据范围隔离。

如果目标产品要求多租户，必须在 Phase 2 前另行完成租户模型、租户绑定、迁移和跨租户测试；在此之前只能按单租户部署验收。

## 5. FA-04 文档一致性验收

已同步：

- `docs/architecture-audit/README.md`：索引、基线、统计和 Phase 1 最终结论。
- `docs/architecture-audit/01-project-inventory.md`：OpenAPI 数量。
- `docs/architecture-audit/06-go-backend-audit.md`：第一轮历史问题与 Phase 1 当前状态分离。
- `docs/architecture-audit/08-api-contract-mapping.md`：历史、修复、当前限制和后续行动分离。
- `docs/architecture-audit/10-implementation-roadmap.md`：P1-05 当前统计。
- `docs/architecture-audit/11-risk-and-decision-log.md`：已解决风险、单租户限制和待决策项。
- `docs/architecture-audit/12-phase-1-implementation-report.md`：Phase 1 统计和验证记录。
- `docs/api/permission-catalog.md`：自动化权限对账说明。
- `docs/status.md`：2xx 响应体语义。

当前文档不再使用旧的 `99 paths / 134 operations` 作为现状，也不再声称所有无响应体的 2xx 必须有 Schema。

## 6. 自动化测试报告

| 检查项 | 执行命令 | 结果 | 备注 |
| --- | --- | --- | --- |
| Go 单元/包测试 | `go test ./... -count=1` | PASS | 全包通过 |
| OpenAPI 全量 Schema | `go test ./api -run TestAllSuccessResponsesHaveValidSchemas -count=1 -v` | PASS | 135 operations、170 responses、102 个需响应体成功响应 |
| 权限路由对账 | `go test ./internal/platform/httpserver -run TestCanonicalRoutePermissionsMatchOpenAPI -count=1 -v` | PASS | 135/12/12/111 |
| `RequireAnyPermission` OR | `go test ./internal/platform/httpserver -run TestRequireAnyPermissionPreservesORSemantics -count=1 -v` | PASS | 允许任一权限，拒绝无权限 |
| Go vet | `go vet ./...` | PASS | 无输出 |
| PostgreSQL/Redis/MySQL 集成 | `go test -tags=integration ./... -count=1` | PASS | 最终全量运行 42.701s；此前一次失败由旧测试使用虚构角色码导致，已改为真实角色成员关系 |
| Workflow 定向集成 | `go test -tags=integration ./test/integration -run 'TestIdentitySecurityAgainstRealPostgresAndRedis/workflow_department_tree,_custom_scope_and_role_boundaries' -count=1 -v` | PASS | 多级部门、树移动、custom、角色边界 |
| Go lint | `golangci-lint run` | PASS | `0 issues.` |
| 漏洞扫描 | `govulncheck ./...` | PASS | 无可达漏洞；仍有 1 个导入包和 8 个模块的非可达公告 |
| sqlc 生成 | `CGO_ENABLED=0 go tool sqlc generate` | PASS | 生成物稳定 |
| oapi-codegen 生成 | `go tool oapi-codegen -config api/oapi-codegen.yaml api/openapi/v1/openapi.yaml` | PASS | 生成物稳定 |
| TypeScript 严格检查 | `npm --prefix frontend-api run check` | PASS | OpenAPI 生成并 `tsc --noEmit` |
| 生成物一致性 | `git diff --check` | PASS | 无空白错误；生成文件纳入 diff 审查 |
| Linux Race | `docker run --rm ... go test -race ./...` | PASS | `golang:1.27.3-bookworm` |

## 7. 遗留问题

### 阻断多租户发布

- 当前没有 tenant 模型和 `tenant_id` 隔离；多租户产品化前必须补充租户边界测试和迁移方案。

### 其他产品决策

- 漏洞资产、漏洞和修复工单已被 Phase 4 明确排除，不是未实现项。
- AI assistant 和支付订单账本仍依赖产品/Provider 决策。
- 外部 Provider 只能声明至少一次投递，不能声明严格 exactly-once。
- 生产数据切换、Provider 凭证、对象存储真实集成和回调地址未验证。

### 非阻断后续

- Identity/Notification 的部分 Repository 仍偏大，属于后续可维护性债务。
- 动态业务 metadata、settings values 和 workflow variables 继续保留受控动态类型。
- React/Next.js 页面、组件、截图和 E2E 属于 Phase 2 之后的工作。

## 8. 最终建议

### 结论：`ACCEPTED WITH CONDITIONS`

Phase 1 当前单租户 Go 后端契约基线可以进入 Phase 2；条件如下：

1. Phase 2 只消费当前 canonical `/api/v1` 契约和生成类型。
2. 任何部署不得宣称已经具备跨租户隔离，除非先完成 tenant 模型和跨租户测试。
3. AI、支付、外部 Provider 和生产数据切换仍需单独的产品与安全批准。
4. Phase 2 前端必须先建立真实 API 接入、权限拒绝、Session/CSRF、错误分支和截图/E2E 基线，不能以静态页面替代。

如果产品要求多租户隔离或本轮要求把租户能力视为 Phase 1 完成条件，则结论应改为 `NOT ACCEPTED`，直到租户模型和跨租户测试完成。
