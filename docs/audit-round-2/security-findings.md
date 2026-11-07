# 安全问题清单

## 已修复

| ID | 级别 | 状态 | 问题 |
| --- | --- | --- | --- |
| SEC-001 | P0 | 已验证修复 | Refresh Token 旋转没有以旧 Token Hash 做 CAS |
| SEC-002 | P0 | 已验证修复 | 短信错误尝试未逐次持久化，存在暴力破解窗口 |
| SEC-003 | P1 | 已验证修复 | 密码变更和 Session 撤销不在同一事务 |
| SEC-004 | P1 | 已验证修复 | 冻结、删除用户后 Session 撤销事务不完整 |
| SEC-005 | P1 | 已验证修复 | 可信代理仅用布尔配置，直接信任 X-Forwarded-For 首项 |
| SEC-006 | P1 | 已验证修复 | 权限缓存删除失败可能继续使用旧高权限缓存 |
| SEC-007 | P1 | 已验证修复 | 社交首登先建用户再绑定，可能产生孤儿用户 |
| SEC-008 | P1 | 已验证修复 | 缺少社交身份解绑和“最后认证方式”保护 |
| SEC-009 | P1 | 已验证修复 | Request ID 可被客户端注入格式异常值 |
| SEC-010 | P1 | 已验证修复 | CSRF 只覆盖 `/api/v1/`，Refresh Cookie 策略不完整 |
| SEC-011 | P1 | 已验证修复 | CORS 对不允许来源的普通请求只省略响应头，仍执行处理 |
| SEC-012 | P1 | 已验证修复 | WebSocket 受 45 秒 HTTP Timeout 影响，且 shutdown 未关闭连接 |
| SEC-013 | P1 | 已验证修复 | Panic Recover 在响应头已提交后仍尝试写 500 |
| SEC-014 | P1 | 已验证修复 | 用户详情、更新、删除等未统一执行 Data Scope |
| SEC-015 | P1 | 已验证修复 | 普通管理员可分配超级管理员角色或修改超级管理员角色 |
| SEC-016 | P1 | 已验证修复 | 文件详情、下载、列表、删除存在水平越权 |
| SEC-017 | P1 | 已验证修复 | 公告详情可绕过定向投递范围 |
| SEC-018 | P1 | 已验证修复 | 用户 Session 撤销接口未校验 Session 归属 |
| SEC-019 | P1 | 已验证修复 | 工作流 businessId 缺少唯一约束，并发启动可重复 |
| SEC-020 | P1 | 已验证修复 | 工作流并发审批错误被映射为 500 |
| SEC-021 | P2 | 已验证修复 | 调度器推进 `next_run_at` 与投递 River 之间存在丢失窗口 |
| SEC-022 | P2 | 已验证修复 | `/metrics` 匿名公开 |
| SEC-023 | P2 | 已验证修复 | Helm 只读根文件系统没有可写 `/tmp` |

## 关键修复说明

### SEC-001 Refresh CAS

`RotateSessionTokens` 现在同时匹配：

- Session ID
- 旧 `refresh_token_hash`
- `revoked_at IS NULL`
- `refresh_expires_at > now()`
- 用户仍为 `active` 且未删除

Access Token Hash、Refresh Token Hash、过期时间和 generation 在单条 UPDATE CTE 内一起变更。旧 hash 写入 `identity_refresh_token_history`。零更新行时检查历史并撤销 Family。

### SEC-002 SMS 原子尝试

验证码查询使用 `FOR UPDATE` 和 CTE。错误与正确输入都会在 SQL 内递增 attempts；正确输入或达到上限时原子消费。发送与验证限流使用不同 key：

- 发送：手机号 1/分钟、IP 5/10 分钟、设备 3/10 分钟
- 验证：手机号 10/10 分钟、IP 20/10 分钟、设备 20/10 分钟

Redis 故障时返回 503，且在数据库验证码尝试前失败。

### SEC-005 可信代理

`AUTH_TRUSTED_PROXY_CIDRS` 使用明确 CIDR。只有 TCP 对端命中可信 CIDR 时才解析 `X-Forwarded-For`，并从右向左跨过可信代理。畸形转发链整体忽略。默认值为空，即默认不信任任何代理头。

### SEC-006 RBAC epoch

`access_cache_epoch` 由数据库触发器递增。角色、角色权限、角色菜单、角色数据范围、用户角色、部门成员、部门树和用户状态变化都会推进 epoch。Redis 条目携带 epoch，epoch 不一致时重新读取 PostgreSQL。

### SEC-009 至 SEC-013 HTTP 加固

- 服务端 Request ID 始终由服务端生成，外部 ID 只保留在 `X-External-Request-ID`。
- CSRF 覆盖所有 `/api/` 写操作，并覆盖 Refresh Cookie。
- CORS 对不允许的 Origin 直接返回 403。
- Bearer 认证优先于 Cookie，避免同时携带时误用 Cookie。
- WebSocket Upgrade 请求跳过 HTTP Timeout。
- WebSocket Origin 从 CORS 白名单生成，并支持优雅关闭。
- Recover 检测响应头是否提交，已提交时不再追加错误响应。

### SEC-014 至 SEC-018 越权

用户管理现在执行以下校验：

- 用户详情、更新、删除、重置密码、撤销 Session 目标必须在 Data Scope 内。
- 普通管理员不能修改超级管理员。
- 普通管理员不能分配超级管理员角色。
- 修改部门集合时必须全部落在当前管理员可管理范围。

文件与公告查询在 SQL/服务层执行归属校验。当前用户撤销 Session 时使用 `id + user_id` 条件更新。

## 仍存在但已披露

详见 `remaining-risks.md`。主要残余风险包括：

- 外部通知 Provider 无法仅靠本地数据库实现严格 exactly-once。
- Refresh 历史表尚未配置定期归档。
- Helm 的反向代理 CIDR、密钥、S3、PostgreSQL 和 Redis 仍需部署方提供。
- 文件对象删除失败时会留下不可见对象，当前选择了安全优先而非强行恢复元数据。

