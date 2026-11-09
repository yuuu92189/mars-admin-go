# Go 原生习惯审计

## 命名

### 结论

- 未发现 `utils`、`common`、`helpers`、`base` 泛化包。
- 包名简洁：`domain`、`application`、`postgres`、`river`、`http`。
- 未发现 `ServiceImpl`、`ManagerImpl` 等 Java 命名。
- 在线 HTTP 兼容层及 `legacy*` 命名已删除；离线导入工具仍保留明确命名。

### 问题

- Java 形状的 legacy JSON 兼容层已删除；新 HTTP 边界继续使用 typed request/response。
- `bootstrapLogger()` 让 Handler 依赖全局 `slog.Default()`，不如显式注入 logger。

## 接口设计

已确认大接口问题见 `architecture.md` 的 ARCH-002。当前不建议大规模重构，只需在新代码中遵守：

- 接口由使用方定义。
- 接口尽量只包含当前用例需要的方法。
- 不为了 Mock 而给所有结构体创建接口。

## Struct 设计

- Domain 类型职责基本清晰。
- 没有 BaseStruct 或继承模拟。
- `application.Service` 字段数量可控。
- `config.Config` 偏大，但属于集中配置快照，可接受。

## 错误处理

### 做得好的部分

- 业务错误有 `apperror.Error` 和 HTTP 映射。
- 事务错误多数保留 `%w`。
- 关键并发错误使用 `errors.Is`。
- 未发现业务路径中的无理由 panic。

### 问题

#### GO-ERR-001 错误字符串匹配 — 待治理

位置示例：

- `internal/modules/identity/adapter/postgres/repository.go` 的 `mapPGError`
- `internal/modules/access/adapter/postgres/repository.go` 的 `mapConflict`
- `internal/modules/system/adapter/postgres/repository.go` 的 `mapConflict`
- `internal/modules/messaging/application/service.go` 的 `isUniqueViolation`

问题：

- 使用 `strings.Contains(err.Error(), "duplicate")` 判断 PostgreSQL 错误。
- 应使用 `*pgconn.PgError` 和 SQLSTATE `23505`。

影响：

- 本地化和驱动文案变化会影响映射。
- 可能误判普通错误。

#### GO-ERR-002 忽略关系加载错误 — 已修复

位置：

- `internal/modules/identity/adapter/postgres/repository.go:39`
- `internal/modules/identity/adapter/postgres/repository.go:54`

- 已修复：`RoleIDs`、`DepartmentIDs`、`PositionIDs` 加载失败现在直接返回错误，不再返回部分 User。

#### GO-ERR-003 静默忽略审计写入 — 已修复

位置：

- `internal/modules/audit/application/service.go`

- 现在写入失败会使用注入的 `slog.Logger` 记录错误。

## Context

- I/O 边界普遍传递 context。
- HTTP 请求使用 `r.Context()`。
- Worker 使用取消信号。
- WebSocket 读操作有 90 秒超时。
- Timeout 中间件明确跳过 WebSocket upgrade。

问题：

- 部分后台清理启动时使用 `context.Background()`，只用于关闭阶段是合理的。
- 测试中大量 `context.Background()` 不属于生产问题。

## 并发

- Identity Refresh 使用数据库 CAS。
- Outbox 使用 lease token、generation 和续租。
- WebSocket Hub 有互斥保护和连接计数。
- 没有发现共享 map 无锁访问。

残余问题：

- Workflow Terminate 与 Decide 的竞态见 `concurrency.md`。
- `RateLimit` 当前用 pipeline 每次重置 TTL，窗口会随请求延长。

## 全局状态与初始化

- `slog.SetDefault` 只在 API bootstrap 中使用，风险可控。
- `db/embed.go` 的 panic 来自 `embed.FS` 静态读取，属于启动期不可恢复错误。
- 未发现 `init()` 滥用。

## Go 原生化判断

当前代码整体是 Go 原生实现。主要改进点是缩小接口、停止错误字符串匹配、显式处理被忽略错误，而不是删除现有分层。
