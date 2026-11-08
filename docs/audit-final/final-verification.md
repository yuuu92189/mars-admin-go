# 最终验证报告

验证日期：2026-11-08

## 必跑检查

| 检查 | 结果 |
| --- | --- |
| `gofmt -w .` | 通过 |
| `go vet ./...` | 通过 |
| `go test ./...` | 通过 |
| `go test -race ./...` | 通过（Linux `golang:1.27.3-bookworm` 容器，源代码 bind mount，`CGO_ENABLED=1`） |
| `go test -tags=integration ./... -count=1` | 通过，`test/integration` 42.100s |
| `golangci-lint run` | `0 issues.` |
| `govulncheck ./...` | 无可达漏洞；报告显示 1 个 import 包和 8 个依赖模块存在漏洞，但当前代码没有可达调用路径 |
| `go tool sqlc generate` | 通过（宿主机缺少 gcc，使用 `CGO_ENABLED=0` 执行） |
| `docker compose config --quiet` | 通过 |

## 真实服务验证

执行：

```text
docker compose up --build -d
```

结果：

- Goose migration `000021_outbox_lease_ownership.sql` 成功执行。
- 数据库当前 Goose version：`21`。
- `outbox_events` 已包含 `lease_token`、`lease_generation`、`lease_expires_at`、`execution_deadline`。
- Worker 启动成功，Scheduler、Notification、Outbox River client 均启动。
- API `GET /health/live`：`200`
- API `GET /health/ready`：`200`

## 真实 PostgreSQL/Redis 集成测试

测试容器：

- PostgreSQL `18.6-bookworm`
- Redis `8.10.1-alpine`

完整集成测试通过，覆盖：

- 业务事务回滚和 Outbox 原子性。
- 生产者去重。
- 多 Relay Worker 不重复领取。
- 失败退避、最大重试、dead-letter 和人工 requeue。
- 过期 processing 租约恢复。
- 旧 token 不能发布或失败。
- 续租 token/generation 校验。
- 并发过期租约重分配。
- 慢发布续租。
- 最大执行时间停止续租。
- 父 context 取消并停止后续状态更新。
- River 入队与 Outbox published 原子提交。
- Consumer 失败重试。
- Redis 稳定事件 ID 去重。
- 公告通知任务稳定业务键去重。
- published Outbox 清理保留期。

Outbox 核心集成测试另执行了 `-count=5` 连续运行，5 次均通过，用于确认事件默认 `available_at` 使用数据库时钟后不存在立即领取的时钟抖动。

## 未声明事项

- 没有宣称外部 SMS/Email/Push Provider exactly-once。
- 没有宣称 WebSocket 断线期间事件可回放。
- 没有宣称跨 7 天 Redis 去重窗口仍能抑制重复消息。
