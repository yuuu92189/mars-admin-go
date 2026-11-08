# 依赖审计

## 直接依赖

| 依赖 | 用途 | 判断 |
| --- | --- | --- |
| `go-chi/chi/v5` | HTTP 路由 | 保留 |
| `pgx/v5` | PostgreSQL | 保留 |
| `sqlc-dev/sqlc` | SQL 代码生成 | 保留 |
| `pressly/goose` | Migration | 保留 |
| `riverqueue/river` | 持久化任务队列 | 保留 |
| `redis/go-redis` | 限流、缓存、实时 fan-out | 保留 |
| `prometheus/client_golang` | Metrics | 保留 |
| `otel` | Trace | 保留 |
| `google/uuid` | UUIDv7 | 保留 |
| `expr-lang/expr` | Workflow 条件 | 保留 |
| `pquerna/otp` | TOTP | 保留 |
| `robfig/cron` | Cron 解析 | 保留 |
| `minio-go` | S3 兼容存储 | 保留 |
| `go-sql-driver/mysql` | Legacy MySQL 导入 | 保留但仅限离线工具 |
| `testcontainers-go` | 真实依赖集成测试 | 保留 |
| `testify` | 测试断言 | 保留 |
| `x/crypto` | Argon2id/bcrypt | 保留 |
| `oapi-codegen` | OpenAPI 模型/服务生成 | 保留，但当前生成代码未参与运行 |
| `kin-openapi` | OpenAPI 校验 | 保留 |

## 功能重叠和大型依赖

- `sqlc` 和 `testcontainers` 会带入较多间接依赖，但属于开发/测试工具，不进入最终二进制。
- `kin-openapi` 与 `oapi-codegen` 都用于 API 契约，用途不同。
- `go-redis` 同时承担限流、缓存和实时发布，是合理复用。
- 未发现 ORM、第二个 SQL 工具、第二个 HTTP 框架或重复消息队列。

## 可删除或待确认

- `generated/api/api.gen.go` 当前未被运行代码导入。
- `FindRefreshHistorySession` 接口和实现未被业务调用。
- `SettingsService.Decrypt` 当前没有调用点。
- `getkin/kin-openapi` 只用于测试，若未来取消运行时契约校验，可转为测试工具依赖。

## 安全

`govulncheck` 在当前提交报告无可达漏洞；仍需持续在 CI 中运行。

## 结论

依赖选择与当前技术栈匹配，没有明显“为了框架而框架”的问题。最需要限制的是 legacy MySQL driver 和测试容器依赖的边界，避免进入主运行路径。
