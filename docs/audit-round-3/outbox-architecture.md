# Transactional Outbox 架构

## 数据流

```text
业务事务
  -> 业务数据
  -> outbox_events
  -> COMMIT

Outbox Relay Worker
  -> SKIP LOCKED claim
  -> River InsertTx
  -> Mark published
  -> COMMIT

River outbox consumer
  -> 通知投递入队 / Redis 实时事件 / 文件对象删除
```

## 表结构

`outbox_events`：

- `id`
- `aggregate_type`
- `aggregate_id`
- `event_type`
- `deduplication_key`
- `payload JSONB`
- `status`
- `attempts`
- `max_attempts`
- `available_at`
- `locked_at`
- `published_at`
- `last_error`
- `created_at`

唯一约束：

```text
(aggregate_type, aggregate_id, event_type, deduplication_key)
```

索引：

- pending/failed 可执行时间
- processing 锁定时间
- aggregate 查询
- published 清理时间

状态：

- `pending`
- `processing`
- `published`
- `failed`
- `dead`

## 事务生产

统一 helper：

- `internal/platform/outbox/outbox.go`
- `internal/platform/outbox/sqlc/*`

业务 Repository 使用现有 pgx 事务直接插入 Outbox。

### 用户

`internal/modules/identity/adapter/postgres/repository.go`：

- 状态变化
- 密码变更
- 软删除

### 公告

`internal/modules/notification/adapter/postgres/repository.go`：

- 发布状态、收件人、station 日志、Outbox 同一事务。

### 聊天

`internal/modules/messaging/adapter/postgres/repository.go`：

- 消息插入和实时事件 Outbox 同一事务。
- 群聊收件人在同一事务中读取当前成员。

### 工作流

`internal/modules/workflow/adapter/postgres/repository.go`：

- task 完成、instance 更新、下一 task、workflow event、Outbox 同一事务。
- 终止流程也写 Outbox。

### 文件

`internal/modules/storage/adapter/postgres/repository.go`：

- 文件软删除与 `storage.file.deleted` Outbox 同一事务。
- Worker consumer 异步删除对象存储对象。

## Relay

`internal/modules/outbox/application/relay.go`：

- `Claim` 使用 `FOR UPDATE SKIP LOCKED`
- 批大小默认 50
- processing lock 默认 2 分钟
- 失败指数退避：2 秒起，最大 10 分钟
- 达到 `max_attempts` 后进入 `dead`
- 每小时清理超过 7 天的 published 事件

## River 一致性

`internal/modules/outbox/adapter/river/backend.go`：

1. 开启 PostgreSQL 事务。
2. River `InsertTx` 插入 outbox consumer job。
3. 更新 Outbox 为 `published`。
4. 同一事务提交。

如果 River insert 或状态更新失败，整个事务回滚。不会出现“River job 已提交但 Outbox 仍 processing”的窗口。

## 消费

Outbox River 队列消费者：

- `notification.notice.published`：幂等入队 SMS/Email/Push delivery job，并发布实时公告事件。
- `messaging.*.created`：发布 Redis realtime 事件，事件携带稳定 Outbox ID。
- `workflow.task.decided`：发布实时工作流事件。
- `identity.user.status_changed`：发布用户状态事件。
- `storage.file.deleted`：调用对象存储删除。

通知 delivery enqueue 使用 River `ByArgs` 唯一约束。重复 Outbox 消费不会重复创建同一 notice/channel delivery job。

Realtime Redis 发布使用 Lua 脚本把 `SET NX EX` 去重键和 `PUBLISH` 放在同一 Redis 原子操作中。相同 Outbox Event ID 的 River 重试不会再次发布实时消息。

## 运维

`cmd/outbox` 支持：

```text
outbox list [status] [limit]
outbox requeue <event-id>
outbox cleanup <retention>
```

失败或 dead Outbox 事件可重新投递。

## 一致性边界

PostgreSQL Outbox 与 River 入队是原子的。

River 消费与外部 Provider 发送不是原子的：

- River consumer 失败会重试。
- Provider 已成功但本地超时会可能重试。
- 不宣称外部短信、邮件、Push 严格 exactly-once。

