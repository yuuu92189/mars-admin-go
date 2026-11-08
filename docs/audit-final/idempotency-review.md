# Outbox 投递幂等性审查

## 总原则

本系统提供的是业务事务内的 Outbox 持久化和至少一次（at-least-once）投递，不宣称端到端 exactly-once。

本地 PostgreSQL 可以保证：

- 业务状态与 Outbox 事件同事务提交或回滚。
- 同一个 Outbox 事件只有一个当前租约 owner 可以标记完成。
- River 入队与 Outbox `published` 状态同事务提交或回滚。

外部 Provider、WebSocket 客户端断线重连和跨系统调用仍按至少一次语义处理。

## 用户状态事件

- Outbox deduplication key：`identity_user + userID + event type + version:<version>`。
- Redis 事件 ID：Outbox event ID。
- Redis 发布使用 Lua `SET NX + PUBLISH`，相同事件 ID 不会重复发布。
- 重复消费者调用同一事件 ID 时，实时发布不会产生第二个 Redis 消息。

## 公告通知

- Outbox deduplication key：`notification_notice + noticeID + publish`。
- 实时事件 ID：Outbox event ID。
- 外部投递 River job 的唯一业务键：`noticeId + channel`。
- 重复消费公告事件时，通知任务不会重复入队。
- SMS、Email、Push Provider 不支持本地事务幂等键时，实际投递仍为 at-least-once；`notification_send_logs` 记录每次尝试，可能产生多次投递记录。

## 聊天消息

- 直接消息和群消息由 `messaging_messages` 的 `(sender_id, client_message_id)` 部分唯一索引去重。
- 同一发送者重复提交同一个 `clientMessageId` 返回冲突，不生成第二条消息和第二个 Outbox 事件。
- Outbox deduplication key：`messaging_message + messageID + created`。
- 实时事件 ID 使用稳定 Outbox event ID；消息 payload 没有显式 event ID 时由消费者填入 Outbox ID。

## 工作流事件

- 工作流实例使用 `(definition_id, business_id)` 唯一索引。
- 任务决策在事务内锁定任务和实例，重复决策被拒绝。
- 决策 Outbox deduplication key 使用 task ID；终止事件使用固定 `terminated` key。
- 重复消费同一 Outbox event 不会改变工作流状态，只会在稳定事件 ID 下抑制重复实时发布。

## 文件删除

- 删除业务记录和删除 Outbox 事件在同一 PostgreSQL 事务内完成。
- `local` Backend 对不存在的对象返回成功。
- S3/MinIO `RemoveObject` 对同一 key 的重复删除是幂等操作。
- 已增加重复删除测试：
  - `internal/modules/storage/adapter/local/backend_test.go`
  - `internal/modules/storage/adapter/s3/backend_test.go`

## Redis 与 WebSocket 边界

- Redis 实时发布使用稳定事件 ID 和 7 天去重 key。
- WebSocket 客户端断线期间不会自动回放事件，必须通过历史接口重新同步。
- 7 天去重窗口之外不承诺重复抑制；这是有意的存储边界，不是 exactly-once 承诺。

## 外部 Provider 边界

- 外部 SMS、Email、Push 和 webhook 未获得 Provider 原生幂等键保证。
- 本轮没有把外部副作用宣称为 exactly-once。
- 需要严格端到端幂等时，应增加 Provider idempotency key 或按收件人/渠道维护业务投递账本。
