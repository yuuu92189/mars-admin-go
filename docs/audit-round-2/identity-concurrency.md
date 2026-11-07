# 身份与并发审计

## Refresh Token

### 原问题

旧实现先查询 Refresh Hash，再用 Session ID 和撤销/过期条件执行 UPDATE。没有把旧 Refresh Hash 放入更新条件，因此同一个旧 Token 可被多个并发请求同时用于生成不同新 Token。

### 新实现

- Session 创建时生成独立 `refresh_family_id`。
- `refresh_generation` 每次旋转原子加一。
- 旋转是单条 SQL CTE：
  - `UPDATE ... WHERE id = ? AND refresh_token_hash = ? AND revoked_at IS NULL AND refresh_expires_at > now()`
  - 同时更新 Access Hash、Refresh Hash、Access 过期、Refresh 过期和 generation。
  - 旧 Refresh Hash 插入历史表。
- 零更新行时查询历史表；历史命中即认定重放，并原子撤销该 Session Family。
- 错误响应统一为：

`The refresh token is invalid or expired.`

该消息不区分 Session 不存在、已撤销、已过期、用户不可用或 Token 重放。

### 验证

真实 PostgreSQL 并发测试：

- 20 路提交同一个旧 Refresh Token。
- 成功数恰好 1。
- 数据库最终 Refresh Hash 与成功响应中的新 Token 对应。
- generation 为 1。
- 重放使 Session Family 被撤销。
- 旧 Token 再次使用失败。
- 新 Token 在重放撤销后也失败。
- 已撤销 Session 不可刷新。
- 已过期 Refresh 不可刷新。

## Session 生命周期

### 密码

`ChangePasswordAndRevokeSessions` 在单个事务内：

1. 更新密码 hash。
2. 启用 `password_login_enabled`。
3. 撤销该用户全部 Session。
4. 提交。

密码校验失败不会撤销 Session。

### 管理重置

管理员重置密码使用同一事务。重置成功后，目标用户所有设备立即失效。

### 冻结、状态变更和删除

`UpdateUser` 现在在同一事务内更新：

- 用户资料
- 状态
- 角色关联
- 部门/岗位
- 非 active 状态下的 Session 撤销

`SoftDeleteUser` 在同一事务内软删除用户并撤销全部 Session。

### Session 归属

用户自助撤销 Session 使用：

```sql
UPDATE identity_sessions
SET revoked_at = now()
WHERE id = $1 AND user_id = $2 AND revoked_at IS NULL
```

知道其他用户 Session ID 也不能撤销其会话。

## 社交身份并发

首登不再使用“创建用户”和“绑定身份”两个独立事务。`CreateSocialUser` 在单事务内创建用户和 `(provider, subject)` 身份。

并发首登时，PostgreSQL 唯一约束决定唯一赢家；其他请求回滚整笔用户创建并读取已存在身份，不会留下孤儿用户。

社交用户 `password_login_enabled=false`。解绑在事务内锁定用户行，并检查：

- 是否仍有其他社交身份
- 是否启用了可用密码登录

最后一个认证方式不能被移除。

## 短信验证码

### 原子尝试

候选验证码使用 `FOR UPDATE`。每次提交都在数据库内更新 attempts：

- 错误：attempts + 1，保留未消费，直到达到上限。
- 正确：attempts + 1，立即消费。
- 达到 max_attempts：立即消费。
- 已消费、已过期或达到上限：后续正确码也失败。

上层不做“先读 attempts 再判断”。

### 限流

发送和验证使用独立策略。手机号、IP、设备 hash 均参与 key。不同手机号无法绕过 IP 总体验证配额。

### 枚举与故障

- 验证码正确但用户不存在：统一验证码错误。
- 用户非 active：统一验证码错误。
- Redis 不可用：返回 503，验证码 attempts 不变。
- 开发环境 SMS 日志仅记录手机号后缀和验证码长度，不记录明文验证码。

## 当前策略的显式取舍

当前重放策略较严格：即使两个请求是同一客户端在极短时间内重试，旧 Refresh Token 的第二次提交也会被判定为重放并撤销 Family。客户端应在旋转响应丢失时重新登录，而不是无限重试旧 Refresh Token。

