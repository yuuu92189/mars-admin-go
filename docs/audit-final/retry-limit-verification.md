# 最大重试次数验证

## 结论

最大重试次数在正常领取、失败回写、租约过期恢复和管理员重投四条路径上均被检查。崩溃恢复不能绕过 `max_attempts`。

## 行为矩阵

| 场景 | SQL 行为 | 结果 |
| --- | --- | --- |
| `pending`/`failed` 且 `attempts < max_attempts` | 领取并 `attempts + 1` | 可执行 |
| `pending`/`failed` 且 `attempts >= max_attempts` | 原子转 `dead` | 不领取 |
| `processing` 租约过期且 `attempts < max_attempts` | 生成新租约并 `attempts + 1` | 可恢复 |
| `processing` 租约过期且 `attempts >= max_attempts` | 原子转 `dead` | 不允许崩溃恢复绕过上限 |
| `dead` | 不满足领取条件 | 不自动领取 |
| 管理员 `requeue` | `attempts = 0`、清空租约、保留错误 | 可重新领取 |

## 失败原因保留

- `MarkOutboxFailed` 将错误写入 `last_error`，最多 4000 字符。
- 达到上限时状态直接转为 `dead`，错误仍保留。
- 领取重试不会清空 `last_error`。
- 过期且耗尽的领取路径会追加 `maximum attempts reached`。
- 手动重投不会清空 `last_error`。

## 实际测试

`test/integration/outbox_test.go` 中新增并通过：

- `failure backoff max attempts dead letter and requeue`
- `crash recovery cannot bypass max attempts and requeue resets attempts`
- `claim atomically dead-letters a ready event already at max attempts`

测试使用真实 PostgreSQL 18.6，验证：

- 失败后按退避时间回到 `failed`。
- 第二次失败达到 `max_attempts = 2` 时转 `dead`。
- 租约过期且 `max_attempts = 1` 时转 `dead`。
- 已经处于 ready 状态且达到上限的事件转 `dead`。
- `dead` 事件不会自动领取。
- 管理员 `Requeue` 后 `attempts` 为 `0`，错误仍保留，随后可重新领取并成功发布。

## CLI

`go run ./cmd/outbox requeue <id>` 或构建后的 `outbox requeue <id>` 输出明确包含：

- `status: pending`
- `attempts: 0`
- `attemptsReset: true`
- `lastErrorPreserved: true`
