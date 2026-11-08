# Refresh 并发与宽限期

## 原缺陷

旧流程中，并发请求 B 的 CAS 失败后会立即把历史记录识别为重放并撤销 Session。请求 A 虽然成功返回新 Token，但 Session 已被 B 撤销，导致新 Token 立即失效。

原集成测试把“并发后 Session 必须撤销”写成了正确行为，掩盖了业务错误。

## 新状态转换

同一旧 Refresh Token 的状态规则：

| 条件 | 结果 |
| --- | --- |
| Session 有效，旧 Hash 匹配 | 单条 CAS 成功，generation +1 |
| Session 已撤销 | 401，不产生新凭证 |
| Refresh 已过期 | 401，不产生新凭证 |
| 用户非 active/deleted | 401，不产生新凭证 |
| CAS 失败，历史存在且 `rotated_at` 在宽限期内 | 401，不撤销 Session |
| CAS 失败，历史存在且超过宽限期 | 原子撤销 Session Family，401 |
| CAS 失败，无历史 | 401，不撤销 |

## 数据库时间

`identity_refresh_token_history.rotated_at` 在旋转事务内由 PostgreSQL `now()` 写入。宽限判断和撤销在 `ClassifyRefreshReplay` 单条 SQL 中完成，不依赖 Go 进程内存计时器。

配置：

```text
AUTH_REFRESH_GRACE_PERIOD=15s
```

允许范围 `0..2m`。设为 0 表示不提供并发宽限。

## 为什么没有加密保存旋转结果

评估过把新 Access/Refresh/CSRF 凭证用独立 AEAD 密钥短期加密保存，使同一旧 Token 的并发重试返回完全相同结果。

本轮没有采用，原因：

- 会新增可解密新 Refresh Token 的数据库密文，扩大高价值凭证攻击面。
- 需要独立密钥生命周期、轮换和泄漏恢复流程。
- 浏览器共享 Cookie Jar 会接收赢家响应；移动端失败请求可重新登录或重新读取最新 Cookie。
- 当前非破坏性宽限期已解决核心业务问题：输掉 CAS 的请求不会撤销赢家 Session。

安全权衡：

- 宽限期内旧 Token 重试仍返回 401，但不会触发 Family 撤销。
- 超过宽限期的旧 Token 使用视为真实重放并撤销 Family。
- 宽限期越短越严格，越长越能覆盖多标签页/弱网并发。

## 并发测试

真实 PostgreSQL 测试：

- 两个 Service 实例交替处理 20 个并发 Refresh 请求。
- 只有 1 个成功。
- 其余 19 个返回 401。
- 数据库 `refresh_generation=1`。
- `revoked_at` 保持 `NULL`。
- 赢家 Access Token 可认证。
- 赢家 Refresh Token 可再次成功旋转。

## 宽限期测试

- 赢家旋转后，另一 Service 在宽限期内使用旧 Token：401，Session 不撤销。
- 新 Access Token 仍有效。
- 新 Refresh Token 可继续旋转。
- 测试将历史 `rotated_at` 推进到 1 分钟前，再次使用旧 Token：401，Session 被撤销。
- 撤销后新 Access/Refresh Token 均失效。

## 异常测试

- Session 已撤销：401。
- Refresh 已过期：401。
- 用户被冻结：401。
- 用户被删除：401。
- Canceled context 下数据库旋转：返回错误，旧 Hash 不变。
- Redis Access Cache 不可用：Refresh 仍通过 PostgreSQL 权威数据完成。
- 多实例：使用两个 Service 实例验证。

## Cookie 行为

Refresh 失败路径不调用 `clearSessionCookies`。API 集成测试断言失败响应没有 `Set-Cookie`，不会清除赢家请求刚下发的新 Cookie。

## 残余影响

并发中失败的客户端仍会收到 401。它不会破坏赢家 Session，但客户端可能需要重新读取共享 Cookie 或重新登录。该行为被明确记录，而不是把所有并发问题推给前端。

