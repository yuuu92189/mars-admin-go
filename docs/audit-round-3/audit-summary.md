# Mars Admin Go 第三轮审计总结

审计日期：2026-11-08

## 目标

本轮完成两项 P1 工程整改：

1. 修复 Refresh Token 并发刷新误撤销。
2. 引入 PostgreSQL Transactional Outbox，并通过 River 投递跨进程事件。

同时复查并处理 Refresh History 清理、部门 Data Scope、Scheduler 失败回滚、文件删除补偿、WebSocket 心跳、Metrics 权限和前端 API 兼容性。

## 结论

Refresh Token CAS 语义保持不变。并发竞争现在由数据库 `rotated_at` 与可配置 Grace Period 分类：

- CAS 仍只有一个请求成功。
- 宽限期内旧 Token 重试返回通用 401，但不会撤销刚成功的 Session。
- 超过宽限期后旧 Token 被识别为真实重放并撤销 Session Family。
- 成功响应的新 Access Token 和新 Refresh Token 在没有真实重放时保持有效。

Outbox 已覆盖：

- 用户关键状态、密码变更、删除
- 公告发布
- 单聊与群聊消息事件
- 工作流审批与终止
- 文件删除对象补偿

业务数据与 Outbox 事件在同一 PostgreSQL 事务提交。Relay 使用 `SKIP LOCKED`，并在同一事务内完成 River `InsertTx` 和 Outbox `published` 状态更新。

## 数据库变更

未修改历史 Migration，新增：

- `000019_refresh_rotation_grace.sql`
- `000020_transactional_outbox.sql`

Compose 数据库已实际升级到 migration `20`。

## 主要验证结果

- 20 路 Refresh 并发：1 成功，0 误撤销。
- 宽限期内旧 Token：失败但不撤销。
- 超过宽限期旧 Token：失败并撤销 Family。
- 新 Access Token 和新 Refresh Token 在无重放场景下有效。
- Outbox 业务事务回滚：Outbox 同步回滚。
- 8 个 Relay Worker 并发：事件无重复发布。
- Relay 崩溃后 stale lock：可恢复。
- River enqueue 与 Outbox 状态：同一事务提交或回滚。
- 消费者失败：River 实际重试。
- Provider timeout：错误向上传播，不标记成功。
- Scheduler River 入队失败：`next_run_at` 与 run 全部回滚。
- 文件删除：Outbox 异步删除对象。
- 部门 Data Scope：列表、详情、父级、更新均受 PostgreSQL 范围约束。

## 质量门禁

- `go test ./...`：通过
- `go test -tags=integration ./... -count=1`：通过
- Linux `go test -race ./...`：通过
- `go vet ./...`：通过
- `golangci-lint run`：0 issues
- `govulncheck ./...`：无可达漏洞
- `sqlc generate`、`oapi-codegen`：通过
- Compose：healthy，migration 20
- Worker：Scheduler、Notification、Outbox 三个 River client 正常启动
- Helm lint/template：通过
- distroless 非 Root + 只读根文件系统：ready 200
- Vue 生产构建：通过，仅有 Sass legacy API 和 chunk size 警告

完整证据见 `verification-report.md`。

## 明确未宣称

- 未宣称外部短信、邮件、Push Provider 严格 exactly-once。
- 未宣称 WebSocket 客户端在断线期间仍能 exactly-once 收到事件；Redis 发布按事件 ID 幂等，断线客户端需通过历史接口重新同步。
- 未保存旧 Refresh Token 对应的明文或加密新凭证；宽限期采用非破坏性失败策略。
- 字典和系统配置属于全局资源，继续使用 RBAC，不虚构部门级归属。

