# 并发与故障恢复审计

## Identity

### 已验证

- 20 路并发 Refresh 只有一个 CAS 成功。
- 旧 Refresh Token 在 grace period 内返回 401，不撤销赢家 Session。
- grace period 外重放撤销 Session Family。
- 登录失败计数和锁定在数据库内原子更新。

### 残余问题

- 登录锁定到期后 failed count 不自动归零，一次错误密码可能立即再次锁定。
- RateLimit 的 INCR + EXPIRE pipeline 每次请求都刷新 TTL，窗口不是固定窗口。
- 未知用户名不执行等价密码哈希，存在响应时间枚举风险。

## Outbox

### 已验证

- Claim 使用 `FOR UPDATE SKIP LOCKED`。
- lease token + generation 阻止旧 Worker 覆盖新 Worker。
- 租约过期恢复不绕过 max_attempts。
- 续租有最大执行截止时间。
- Renew/MarkPublished/MarkFailed 都校验所有权。
- River 插入与 published 状态同事务。

### 结论

Outbox 并发模型已经达到本轮目标的可靠性要求。剩余风险主要是外部 Provider 至少一次语义。

## Workflow

### 已验证

- Decide 使用 `FOR UPDATE OF t, i`。
- 重复决策返回 conflict。
- definition + business_id 唯一约束阻止重复实例。

### 已确认问题

#### CONC-001 Terminate 与 Decide 竞态 — 已修复

位置：

- `internal/modules/workflow/adapter/postgres/repository.go:462`

问题：

- Terminate 读取实例时没有 `FOR UPDATE`。
- 更新实例状态时没有 `WHERE status='running'`。
- 并发 Decide 可能先完成任务，随后 Terminate 把实例改为 terminated，或反向发生。

影响：

- task 状态、instance 状态和 outbox 事件可能语义不一致。

建议：

- 事务开始时锁定 instance。
- 使用 status CAS 并检查 affected rows。

## Messaging / WebSocket

- Hub 的 clients map 有 RWMutex。
- 连接计数使用 atomic。
- writer 有 30 秒 ping 和 10 秒写超时。
- 单连接发送队列满时关闭连接，避免无限堆积。
- Redis publish 使用稳定 event ID 和 Lua SET NX。

问题：

- Redis 发布失败时会本地 dispatch 并返回错误，重试可能重复投递。
- WebSocket query token 会进入代理和访问日志，旧前端仍在用。
- 没有单用户连接数限制。

## Scheduler

- Tick 使用 `FOR UPDATE SKIP LOCKED`。
- 任务 next_run_at 更新、run 创建和 River 插入在同一事务。
- River job 失败有重试记录。

问题：

- `RunNow` 的 run 状态更新错误被忽略。
- `legacyStatistics` 和 `legacyCleanLogs` 是静默 stub。

## Rate Limit

原实现：

```text
TxPipeline: INCR key; EXPIRE key window
```

已修复：

- 每次请求都会重置窗口。
- 持续请求可使计数长期不归零。

建议：

- 已使用 Lua：首次 INCR 后仅设置一次 TTL，并补充固定窗口真实 Redis 测试。

## Context 与关闭

- API/Worker 有明确取消和超时。
- WebSocket 读写有超时。
- Outbox 续租失败会取消当前发布 context。
- 少数 bootstrap 清理路径忽略错误，但不构成长期运行竞态。

## 并发结论

Outbox 和 Refresh 已具备较强并发保障。下一优先级是 Workflow Terminate、群组所有权和 RateLimit 窗口。
