# Relay 故障恢复

## 事务边界

业务状态写入和 Outbox 插入在同一个 PostgreSQL 事务内：

- 业务事务回滚：Outbox 事件不存在。
- 业务事务提交：Outbox 事件存在，等待 Relay 投递。

River 入队和 Outbox `published` 更新在同一个 PostgreSQL 事务内：

- River 入队失败：Outbox 状态更新回滚。
- Outbox 状态更新失败：River job 插入回滚。
- 两者都成功后才提交。

## 故障阶段

### 领取后、发布前崩溃

Outbox 保持 `processing`，租约到期后由新的 Worker 重新领取。旧 token 不能完成或失败。

测试：`stale processing lock is recovered after relay crash`

### 发布失败

Relay 使用当前租约调用 `MarkOutboxFailed`：

- 未达上限：回到 `failed` 并按指数退避。
- 达到上限：转为 `dead`。
- 旧租约 owner 的失败回写被拒绝。

### River 入队事务回滚

测试：`River insert and published status are atomic`

通过将事件状态模拟为非 processing，使状态更新失败，验证 River job 没有残留，Outbox 状态没有被错误发布。

### 领取租约被接管

旧 Worker 继续运行时：

- 续租失败会取消发布 context。
- `MarkPublished` 和 `MarkFailed` 都带 token/generation 条件。
- 新 Worker 的租约不会被旧 Worker 覆盖。

测试：

- `stale lease owner cannot publish or fail a reassigned event`
- `relay stops renewal at the execution deadline and leaves recovery to a new owner`

### River Consumer 失败

River job 仍按 River 自身重试策略执行。Outbox 已成功转为 `published` 后不会因为 Consumer 失败而重新领取。

测试：`consumer failure is retried by River`

### dead-letter 人工恢复

管理员 `requeue` 将 `failed`/`dead` 恢复为 `pending`，重置 attempts，保留 last_error。下一次领取生成新 token 和新 generation。

## 恢复语义

恢复保证事件不会因为崩溃而丢失，也不会让旧租约 owner 覆盖新租约 owner。它不等于消费者副作用 exactly-once：

- 外部 Provider 可能重复发送。
- WebSocket 断线期间的事件不可回放。
- 客户端需要用历史接口进行最终一致性同步。
