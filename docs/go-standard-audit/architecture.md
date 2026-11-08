# 架构审计

## 当前结构

```text
cmd/* -> internal/bootstrap -> module application/domain
                         \-> module HTTP/PostgreSQL/Redis/River adapters
```

`cmd` 主要负责配置、信号和生命周期；`internal/bootstrap` 负责 API 依赖装配；Worker 在 `cmd/worker` 中完成运行期装配。模块总体按业务能力组织，目录结构与 Modular Monolith 目标一致。

## 依赖方向

### 符合方向的部分

- `domain` 包只依赖 context、time、UUID 和少量纯规则库。
- HTTP Handler 调用 application service，不直接执行 SQL。
- PostgreSQL/River/Redis 实现位于 adapter 层。
- 启动装配集中在 `internal/bootstrap/app.go`。
- 没有 Java Spring、MyBatis、Hibernate、Maven 或 Gradle 运行时依赖。

### 已确认问题

#### ARCH-001 跨模块直接使用 sqlc adapter

位置：

- `internal/modules/identity/adapter/postgres/repository.go:12`
- `internal/modules/identity/adapter/postgres/repository.go:437`
- `internal/modules/notification/adapter/postgres/repository.go:9`
- `internal/modules/notification/adapter/river/backend.go:11`

实际问题：

- Identity PostgreSQL adapter 直接导入 Access 模块的 sqlc 包读取 `GetUserAccess`。
- Notification PostgreSQL/River adapter 直接导入 Messaging 模块的 sqlc 包读取公告表和收件人表。
- 模块边界在 Go 包依赖层面被穿透。

影响：

- 修改 Access 或 Messaging SQLC 输出会影响其他模块。
- 难以单独提取或测试模块。
- 违反项目自身 ADR 中“module-local sqlc packages”的目标。

建议：

- 在 Identity 使用方定义小型 `AccessReader` 接口，由 Access adapter 实现。
- 将 notice 查询拆分为 `db/queries/notification` 和 notification-owned sqlc 包。

#### ARCH-002 Domain Repository 接口过大

位置：

- `internal/modules/identity/domain/identity.go`
- `internal/modules/access/domain/access.go`
- `internal/modules/messaging/domain/messaging.go`

实际问题：

- 一个 Repository 接口聚合数十个方法。
- 接口定义在 domain，而不是由具体 application service 按使用场景定义。
- 测试替身必须实现大量无关方法。

影响：

- 增加测试和演进成本。
- 接口没有体现最小使用面。

建议：

- 渐进式按用例拆分接口，例如 `UserReader`、`SessionStore`、`CredentialStore`。
- 不要求一次性拆完，也不机械删除已有接口。

#### ARCH-003 Monitoring 跨模块直接读取业务表

位置：

- `internal/modules/monitoring/service.go`

实际问题：

- 监控模块直接查询 Identity、Storage、Messaging、Notification、Access 和 Outbox 表。

影响：

- 只读跨模块耦合，修改表结构会影响监控。
- 对小型管理后台可以暂时接受，但需要在模块边界中明确为“read model”。

建议：

- 短期保留，标注为 cross-module read model。
- 长期可改为 SQL view 或独立 dashboard query package。

## 命令和生命周期

### 符合 Go 习惯

- 使用 `signal.NotifyContext`。
- API 使用 `http.Server.Shutdown`。
- Worker 有显式 stop 顺序。
- River、Redis、PostgreSQL 生命周期明确。

### 残余问题

- `cmd/worker/main.go` 的启动清理使用 `stopBackend` 时忽略错误。
- API bootstrap 在 Hub 启动后如果后续依赖失败，没有调用 Hub 的 Close。
- 进程退出最终会释放资源，但测试和嵌入式启动会暴露生命周期泄漏。

## 架构结论

不需要推倒重来。当前最合适的治理方式是保持 Modular Monolith，逐步消除跨模块 sqlc 依赖，并缩小 application 使用方接口，而不是增加更多层或依赖注入框架。
