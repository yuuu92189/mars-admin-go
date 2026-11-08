# Outbox 租约设计

## 目标

本轮只为 Transactional Outbox 增加可验证的租约所有权和有限续租能力，不改变业务事件模型，不替换 River、PostgreSQL 或 Redis。

## 数据库模型

迁移 `db/migrations/000021_outbox_lease_ownership.sql` 新增：

- `lease_token UUID`
- `lease_generation BIGINT NOT NULL DEFAULT 0`
- `lease_expires_at TIMESTAMPTZ`
- `execution_deadline TIMESTAMPTZ`

约束保证：

- `lease_generation >= 0`
- `lease_expires_at <= execution_deadline`
- `status = 'processing'` 时必须有完整的租约信息

升级时，迁移前已经处于 `processing` 的事件会获得一个未知 token。旧租约到期后，事件通过正常领取路径恢复，不会直接假定旧 Worker 仍然拥有租约。

## 原子领取

`db/queries/outbox/outbox.sql` 的 `ClaimOutboxEvents` 使用单个 PostgreSQL 语句完成：

1. 将 `pending`/`failed` 且已达到 `max_attempts` 的事件原子转为 `dead`。
2. 将过期且已达到 `max_attempts` 的 `processing` 事件原子转为 `dead`。
3. 从可用或已过期的、未耗尽次数的事件中选择候选行。
4. 使用 `FOR UPDATE SKIP LOCKED` 避免多个 Worker 取得同一行。
5. 增加 `attempts`。
6. 生成新的 `lease_token`。
7. 递增 `lease_generation`。
8. 设置 `locked_at`、`lease_expires_at` 和 `execution_deadline`。

没有 Go 层 `SELECT` 后再 `UPDATE` 的状态转换。

默认 `available_at` 由 PostgreSQL `now()` 计算，避免应用主机和数据库存在微小时钟偏差时，新事件刚插入却暂时无法领取。显式指定未来 `available_at` 的调度语义仍然保留。

## 所有权校验

以下操作都要求：

- `id = event_id`
- `lease_token = current_lease_token`
- `lease_generation = current_lease_generation`
- `status = 'processing'`
- `lease_expires_at > now()`

覆盖：

- `MarkOutboxPublished`
- `MarkOutboxFailed`
- `RenewOutboxLease`

租约过期回收由领取语句的 `FOR UPDATE SKIP LOCKED` 完成。旧 Worker 即使继续执行，也不能用旧 token/generation 更新已经被新 Worker 领取的事件。

## 续租和最大执行时间

默认值：

- 租约 TTL：`30s`
- 续租间隔：`10s`
- 单次领取最大执行时间：`5m`

Relay 在发布期间启动续租 ticker。续租失败时会取消发布 context，旧 Worker 停止后续完成或失败状态更新。

`execution_deadline` 在领取时固定，续租只延长 `lease_expires_at`，且不会超过该 deadline。因此：

- 长任务可以续租。
- 过期租约不能被续租。
- 已被新 Worker 接管的事件不能被旧 Worker 续租。
- 单个 Worker 不能通过无限续租永久占有事件。

## 重投策略

管理员通过 `cmd/outbox` 的 `requeue <id>` 手动重投：

- 只允许 `failed` 或 `dead`。
- `attempts` 重置为 `0`。
- `available_at` 设为当前时间。
- 清空锁和租约。
- 保留 `last_error`，便于审计最近失败原因。

`dead` 事件不会被自动领取。下一次领取会重新生成 token 并递增 generation。

## 关键代码

- `db/queries/outbox/outbox.sql`
- `internal/modules/outbox/adapter/postgres/repository.go`
- `internal/modules/outbox/application/relay.go`
- `internal/modules/outbox/adapter/river/backend.go`
- `internal/platform/outbox/outbox.go`
- `cmd/outbox/main.go`
