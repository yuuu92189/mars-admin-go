# 多 Worker 竞争测试

## 测试环境

- 真实 PostgreSQL：`postgres:18.6-bookworm`
- 真实 Redis：`redis:8.10.1-alpine`
- Testcontainers 启动，迁移执行到 Goose version `21`
- 测试入口：`test/integration/outbox_test.go`
- 完整命令：`go test -tags=integration ./... -count=1`

## 覆盖场景

### 1. 旧租约失效后由新 Worker 接管

测试：`stale lease owner cannot publish or fail a reassigned event`

流程：

1. Worker A 领取事件，获得 token A 和 generation 1。
2. 测试将 A 的租约模拟为过期。
3. Worker B 领取同一事件，获得 token B 和 generation 2。
4. A 调用 `MarkPublished`，返回 `ErrLeaseLost`。
5. A 调用 `MarkFailed`，返回 `ErrLeaseLost`。
6. B 调用 `MarkPublished` 成功。

最终状态为 `published`，`attempts = 2`。

### 2. 续租所有权

测试：`lease renewal requires the current token and generation`

- A 可以成功续租。
- 使用错误 token 不能续租。
- 使用错误 generation 不能续租。
- 错误 owner 的 `MarkFailed` 和 `MarkPublished` 都被拒绝。
- 当前 owner 仍可完成发布。

### 3. 过期租约并发重分配

测试：`expired lease is reassigned exactly once under concurrent claims`

8 个 Worker 同时对同一过期事件调用 `Claim`。实际结果只有一个 Worker 获得有效租约，token 不同且 generation 从 1 增至 2。

### 4. 多 Worker 正常领取不重复

测试：`multiple relay workers claim disjoint batches`

8 个 Relay 并发处理 40 个事件。每个事件只被发布一次，最终 40 个事件全部为 `published`。

### 5. 最大重试和 dead-letter

测试：

- `crash recovery cannot bypass max attempts and requeue resets attempts`
- `claim atomically dead-letters a ready event already at max attempts`

验证耗尽次数的 ready/expired processing 事件自动转为 `dead`，dead 事件不会自动领取。

### 6. 慢发布期间持续续租

测试：`relay renews a lease while publication is slow`

使用真实 PostgreSQL 和计数 Repository：

- 发布耗时超过初始租约 TTL。
- Relay 至少完成 2 次续租。
- 当前 owner 最终成功发布。

### 7. 达到最大执行时间后停止旧 Worker

测试：`relay stops renewal at the execution deadline and leaves recovery to a new owner`

- 发布方阻塞等待 context。
- Relay 持续续租，直到固定 `execution_deadline`。
- 续租失败后 context 被取消，旧 Worker 不写失败状态。
- 新 Worker 可以重新领取并成功发布。

### 8. 父 context 取消

测试：`relay propagates parent context cancellation without writing failure`

- Relay 在发布阻塞时收到父 context 取消。
- `RunOnce` 返回 `context.Canceled`。
- 不写入 `failed` 或 `dead`。
- 事件保持 `processing`，租约到期后可由新 Worker 恢复。

## 结论

租约 token 和 generation 的原子校验在真实 PostgreSQL 并发下生效。旧 Worker 的完成、失败和续租更新均不能覆盖新 owner。
