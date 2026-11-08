# Refresh 重放检测

## 历史记录

`identity_refresh_token_history` 现在记录：

- Session ID
- 旧 Refresh Token Hash
- generation
- `used_at`
- `rotated_at`

`token_hash` 唯一。旧 Token 明文不进入数据库或日志。

## 分类 SQL

`ClassifyRefreshReplay`：

1. 根据旧 Hash 查询历史。
2. 使用 `now() - gracePeriod` 判断是否在宽限期。
3. 只有超过宽限期的历史记录才进入撤销 CTE。
4. 返回 Session ID、`within_grace` 和 `revoked`。

撤销与分类在同一条 PostgreSQL 语句中，多个实例共享同一数据库时间。

## 真正重放

超过宽限期后：

- Session Family 被设置 `revoked_at=now()`。
- 后续 Refresh 无法重新激活 Session。
- Access Token 的查询条件也要求 `revoked_at IS NULL`。

## 历史清理

新增 `DeleteExpiredRefreshHistory`：

- 已撤销 Session：`revoked_at` 超过保留期后删除历史。
- 未撤销 Session：`refresh_expires_at` 超过保留期后删除历史。
- 活跃 Family 的历史不会被删除。

配置：

```text
AUTH_REFRESH_HISTORY_RETENTION=720h
```

Worker 启动时执行一次清理，此后每 6 小时执行。

## 验证

- 活跃 Family 旋转后历史保留。
- 已撤销 Session 的旧历史在 retention 后删除。
- 清理不影响活跃 Session 的重放分类能力。
- Migration 19 从旧 `used_at` 回填 `rotated_at`。

## 残余风险

- 当前采用删除策略，没有单独的历史归档表。
- 安全审计所需的登录/操作事件仍由 audit 表保留；Refresh Token Hash 历史只用于会话重放检测。
- Grace Period 内的重放被检测为失败，但不立即撤销；这是为消除并发误撤销而明确接受的窗口。

