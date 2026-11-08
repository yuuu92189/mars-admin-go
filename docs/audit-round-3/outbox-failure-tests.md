# Outbox 故障测试

真实 PostgreSQL/Redis 测试位于 `test/integration/outbox_test.go`。

## 1. 业务事务回滚

操作：

- 同一事务插入 dictionary type 和 Outbox event。
- Rollback。

结果：

- dictionary type 数量 0。
- Outbox event 数量 0。

结论：业务数据与 Outbox 同步回滚。

## 2. 重复生产

操作：

- 同一 aggregate/event/dedup key 在同一事务插入两次。

结果：

- 数据库只有 1 条 Outbox event。

结论：生产端重试幂等。

## 3. 多 Relay Worker

操作：

- 40 个事件。
- 8 个 Relay 同时运行，每个批次 5。
- Publisher 延迟 2ms 增加竞争。

结果：

- 40 个事件全部 published。
- Publisher 观察 40 个唯一 ID。
- 无重复发布。

结论：`SKIP LOCKED` 多实例竞争正确。

## 4. Relay 失败与指数退避

操作：

- Failing Publisher。
- MaxAttempts=2。

结果：

- 第一次失败：`failed`，attempts=1，`available_at` 向后移动。
- 第二次失败：`dead`，attempts=2。
- CLI/Repo Requeue 后重新成功发布。

结论：退避、最大重试、dead letter 和重投有效。

## 5. Relay 崩溃恢复

操作：

- Claim 后不发布，模拟进程崩溃。
- lock timeout 20ms。
- 再次 Claim。

结果：

- 同一事件被重新 Claim。
- attempts 从 1 增至 2。

结论：stale processing lock 可恢复。

## 6. River 入队与状态原子性

操作：

- Claim event 后人为把状态改回 pending。
- River Backend `InsertTx` 成功写入事务，但 `Mark published` 影响 0 行。

结果：

- Publish 返回错误。
- River 事务回滚。
- `river_job` 中该 event ID 数量为 0。
- Outbox 仍为 pending。

结论：River job 与 Outbox published 状态不会部分提交。

## 7. 消费者失败

操作：

- Outbox consumer handler 始终返回错误。
- 启动真实 River Outbox Worker。

结果：

- Handler 被调用至少 2 次。
- River 自动重试。

结论：消费者失败由 River 重试处理。

## 8. Redis 实时投递

操作：

- 订阅 `mars:admin:realtime`。
- `RedisPublisher.Publish` 发送 realtime event。

结果：

- Redis subscriber 收到包含目标 user ID 的消息。
- 相同 Event ID 第二次 Publish 不产生第二条消息。
- 去重键存在。

## 9. 重复通知投递入队

操作：

- 对同一 notice/channel 连续 Enqueue 两次。

结果：

- 返回相同 River job ID。
- `river_job` 只有 1 条。

## 10. Provider timeout

单元测试：

- Fake SMS Provider 阻塞到 context deadline。

结果：

- 返回 `context.DeadlineExceeded`。
- success count 为 0。

结论：Provider timeout 不会被标记为成功，River delivery job 可重试。

## 11. 文件对象删除失败

单元测试：

- Fake storage backend `Delete` 返回错误。

结果：

- Outbox consumer 返回错误，交由 River 重试。

API 集成测试同时验证成功路径：

- 上传文件。
- 删除元数据。
- Outbox event published。
- 本地对象实际消失。

## 12. 清理

- 将 published 事件时间推进 2 天。
- Retention 设为 1 天。
- Cleanup 删除事件。

结论：published 清理有 retention 边界。

