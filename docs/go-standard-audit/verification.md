# 最终验证记录

验证日期：2026-11-08

## 命令结果

| 命令 | 结果 |
| --- | --- |
| `gofmt -w .` | 通过 |
| `go vet ./...` | 通过 |
| `go test ./...` | 通过 |
| `go test -race ./...` | 通过，在 `golang:1.27.3-bookworm` 容器中执行，源代码 bind mount，`CGO_ENABLED=1` |
| `go test -tags=integration ./... -count=1` | 通过，`test/integration` 47.725s |
| `go mod tidy` | 通过 |
| `golangci-lint run` | `0 issues.` |
| `govulncheck ./...` | 无可达漏洞；1 个 import 包和 8 个依赖模块有已知漏洞，但当前代码无调用路径 |
| `go tool sqlc generate` | 通过 |
| `go tool oapi-codegen -config api/oapi-codegen.yaml api/openapi/v1/openapi.yaml` | 通过 |
| `docker compose config --quiet` | 通过 |
| `docker compose up --build -d` | 通过 |

## 真实依赖测试

Testcontainers 实际启动：

- PostgreSQL `18.6-bookworm`
- Redis `8.10.1-alpine`
- MySQL `8.0.36`，用于 legacy import

新增 `TestGovernanceFixesAgainstRealDependencies` 覆盖：

- Monitoring 权限。
- Profile 安全字段保护。
- Group owner 保护和解散群发送拒绝。
- Soft delete 后用户名复用。
- Codegen 不存在表和写权限。
- Notification 跨页总数、read-all 和幂等 read。
- Settings 敏感值加密、base64 存储和解密。
- 固定窗口 RateLimit。
- Workflow Terminate/Decide 并发一致性。

## 运行态

- Goose version：`23`
- 迁移 `000022_soft_delete_uniqueness.sql`：成功
- 迁移 `000023_query_indexes.sql`：成功
- API `/health/live`：`200`
- API `/health/ready`：`200`
- Worker：Scheduler、Notification、Outbox River client 均启动

## Helm

本机未安装 `helm`，使用容器镜像 `alpine/helm:3.18.4` 执行：

- `helm lint`：通过，0 chart failed
- `helm template`：通过

## 覆盖率

`go test -cover ./...` 显示：

- Workflow domain：71.8%
- Storage local：64.1%
- Storage S3：64.7%
- Config：53.0%
- Security：47.1%
- Messaging realtime：38.6%
- HTTP server：36.3%
- Codegen application：28.1%
- 多个 application/adapter 仍为 0% 单元覆盖

因此覆盖率不是本轮的生产就绪证明。真实 PostgreSQL/Redis/MySQL 集成测试是主要证据，残余覆盖缺口记录在 `testing.md`。

## 环境限制

- 宿主机缺少可用 gcc，Windows 原生 `go test -race` 无法编译 race runtime；使用 Linux Go 容器执行并实际完成全部包的 race 测试。
- 本机没有 helm 二进制；使用固定容器镜像执行 lint/template。
