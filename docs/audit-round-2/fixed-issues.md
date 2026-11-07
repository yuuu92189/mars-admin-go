# 已修复问题明细

## FIX-001 — Refresh Token 并发旋转

- 问题编号：FIX-001 / SEC-001
- 严重程度：P0
- 文件路径：
  - `db/queries/identity/sessions.sql`
  - `internal/modules/identity/adapter/postgres/repository.go`
  - `internal/modules/identity/application/service.go`
  - `db/migrations/000016_identity_security_hardening.sql`
- 原始问题：UPDATE 未匹配旧 Refresh Hash，20 个并发请求可能产生多个成功。
- 修复方法：单条 UPDATE CTE 以旧 Hash 做 CAS，同时旋转 access/refresh、generation 和过期时间；旧 Hash 写历史；重放撤销 Family。
- 新增测试：`identity_security_test.go` 的 20 路 Refresh 并发、重放、撤销、过期测试。
- 测试结果：通过，恰好 1 个成功，最终 Hash 与成功响应一致。
- 是否彻底解决：数据库旋转竞态已彻底解决；历史表归档属于独立运维风险。

## FIX-002 — SMS 验证码尝试与限流

- 问题编号：FIX-002 / SEC-002
- 严重程度：P0
- 文件路径：
  - `db/queries/identity/sessions.sql`
  - `internal/modules/identity/adapter/postgres/repository.go`
  - `internal/modules/identity/application/service.go`
  - `internal/platform/httpserver/ratelimit.go`
- 原始问题：错误码未证明每次原子递增；正确码并发、发送限流和验证限流边界不完整。
- 修复方法：`FOR UPDATE` + UPDATE CTE；每次尝试原子递增；正确或达到上限时消费；手机号/IP/设备分离发送和验证限流；账号不存在统一错误。
- 新增测试：连续 5 次错误、达到上限后正确码、20 路正确码并发、过期、已消费、跨手机号 IP 绕过、Redis 故障。
- 测试结果：全部通过。
- 是否彻底解决：是，针对数据库和 Redis 路径均有真实验证。

## FIX-003 — 密码与 Session 事务

- 问题编号：FIX-003 / SEC-003
- 严重程度：P1
- 文件路径：
  - `internal/modules/identity/adapter/postgres/repository.go`
  - `internal/modules/identity/application/service.go`
- 原始问题：密码更新和 Session 撤销是两个独立操作，中间失败会留下可继续使用的旧 Session。
- 修复方法：新增 `ChangePasswordAndRevokeSessions`，同事务更新密码并撤销所有 Session。
- 新增测试：多设备改密、管理重置。
- 测试结果：通过，所有旧 Access Token 返回 401。
- 是否彻底解决：是。

## FIX-004 — 冻结与删除用户 Session

- 问题编号：FIX-004 / SEC-004
- 严重程度：P1
- 文件路径：
  - `internal/modules/identity/adapter/postgres/repository.go`
  - `internal/modules/identity/application/service.go`
- 原始问题：状态更新、软删除和 Session 撤销边界不一致。
- 修复方法：用户字段、角色、部门、岗位和状态更新同事务；非 active 自动撤销 Session；软删除与撤销同事务。
- 新增测试：冻结、恢复、删除和多个设备。
- 测试结果：通过。
- 是否彻底解决：是。

## FIX-005 — 可信代理 CIDR

- 问题编号：FIX-005 / SEC-005
- 严重程度：P1
- 文件路径：
  - `internal/platform/config/config.go`
  - `internal/platform/httpserver/ratelimit.go`
  - `internal/platform/httpserver/middleware.go`
  - `internal/modules/identity/adapter/http/handler.go`
- 原始问题：布尔开关会信任任意来源的 XFF 首项；HSTS 也信任任意 X-Forwarded-Proto。
- 修复方法：`AUTH_TRUSTED_PROXY_CIDRS`；从右向左遍历可信代理链；畸形链整体丢弃；HSTS 只信任可信代理的 proto。
- 新增测试：直接访问、伪造 XFF、单层/多层代理、非法 IP、IPv6、HSTS。
- 测试结果：通过。
- 是否彻底解决：代码路径已解决；生产必须正确配置代理 CIDR。

## FIX-006 — RBAC 缓存 epoch

- 问题编号：FIX-006 / SEC-006
- 严重程度：P1
- 文件路径：
  - `db/migrations/000016_identity_security_hardening.sql`
  - `internal/modules/identity/application/service.go`
  - `internal/modules/access/application/service.go`
- 原始问题：Redis 删除失败被忽略，旧高权限缓存可能继续使用。
- 修复方法：数据库 epoch 作为权威版本；缓存值携带 epoch；触发器覆盖权限、角色、菜单、成员和用户状态变化。
- 新增测试：权限删除后旧 Redis 仍存在，但重新认证不再拥有权限。
- 测试结果：通过。
- 是否彻底解决：是。

## FIX-007 — 社交登录与解绑

- 问题编号：FIX-007 / SEC-007、SEC-008
- 严重程度：P1
- 文件路径：
  - `internal/modules/identity/adapter/postgres/repository.go`
  - `internal/modules/identity/application/service.go`
  - `db/migrations/000017_social_login_safety.sql`
- 原始问题：首登可能先创建孤儿用户；没有安全解绑。
- 修复方法：用户与身份同事务创建；唯一约束解决并发；新增 `password_login_enabled`；事务内锁定用户并检查剩余认证方式。
- 新增测试：20 路社交首登、唯一身份、最后认证方式解绑 409、第二认证方式存在时可解绑。
- 测试结果：通过。
- 是否彻底解决：是。

## FIX-008 — Request ID

- 问题编号：FIX-008 / SEC-009
- 严重程度：P1
- 文件路径：`internal/platform/httpserver/middleware.go`
- 原始问题：只检查长度，不检查字符集，并直接复用外部 ID。
- 修复方法：服务端始终生成内部 ID；外部 ID 校验长度和字符集，仅通过 `X-External-Request-ID` 保留。
- 新增测试：非法字符、合法外部 ID。
- 测试结果：通过。
- 是否彻底解决：是。

## FIX-009 — CSRF

- 问题编号：FIX-009 / SEC-010
- 严重程度：P1
- 文件路径：
  - `internal/platform/httpserver/middleware.go`
  - `internal/modules/identity/adapter/http/handler.go`
- 原始问题：只覆盖 `/api/v1/`，只检测 `mars_session`，Refresh Cookie 不完整。
- 修复方法：覆盖 `/api/` 写操作；检测 session/refresh Cookie；Bearer 优先且不依赖 Cookie；清除 Refresh Cookie 使用正确 Path。
- 新增测试：legacy API、Refresh Cookie、Bearer 流程。
- 测试结果：通过。
- 是否彻底解决：是。

## FIX-010 — CORS

- 问题编号：FIX-010 / SEC-011
- 严重程度：P1
- 文件路径：`internal/platform/httpserver/middleware.go`
- 原始问题：不允许的 Origin 在非 OPTIONS 请求中仍进入业务处理。
- 修复方法：任何带不允许 Origin 的请求直接 403。
- 新增测试：非预检 POST 不允许来源。
- 测试结果：通过。
- 是否彻底解决：是。

## FIX-011 — WebSocket

- 问题编号：FIX-011 / SEC-012
- 严重程度：P1
- 文件路径：
  - `internal/platform/httpserver/middleware.go`
  - `internal/modules/messaging/realtime/hub.go`
  - `internal/bootstrap/app.go`
- 原始问题：全局 HTTP Timeout 作用于长连接；Origin 硬编码；App.Close 不关闭连接。
- 修复方法：Upgrade 跳过 HTTP Timeout；Origin 从 CORS 配置生成；Hub 跟踪关闭状态、并行 close、等待注销；App.Close 调用 Hub.Close。
- 新增测试：Upgrade 无 deadline、Origin 配置生成、API 集成 WebSocket ping/pong。
- 测试结果：通过。
- 是否彻底解决：是。

## FIX-012 — Panic Recover

- 问题编号：FIX-012 / SEC-013
- 严重程度：P1
- 文件路径：`internal/platform/httpserver/middleware.go`
- 原始问题：响应头提交后 panic 仍尝试写 500。
- 修复方法：Recover 使用提交状态记录器，已提交时不追加问题响应。
- 新增测试：写入 204 后 panic。
- 测试结果：通过。
- 是否彻底解决：是。

## FIX-013 — 用户 Data Scope

- 问题编号：FIX-013 / SEC-014
- 严重程度：P1
- 文件路径：
  - `db/queries/identity/users.sql`
  - `internal/modules/identity/application/service.go`
  - `internal/modules/identity/adapter/http/handler.go`
- 原始问题：用户详情和变更接口只检查权限，不检查目标数据范围。
- 修复方法：SQL 单条范围查询；详情/更新/删除/重置/撤销统一执行；部门分配校验。
- 新增测试：department scope 列表、详情、更新、部门越界。
- 测试结果：通过。
- 是否彻底解决：用户管理路径已解决。

## FIX-014 — 超级管理员保护

- 问题编号：FIX-014 / SEC-015
- 严重程度：P1
- 文件路径：
  - `internal/modules/identity/application/service.go`
  - `internal/modules/access/application/service.go`
- 原始问题：普通管理员可通过用户或角色接口修改超级管理员。
- 修复方法：普通管理员不能修改 super admin 用户；不能分配 super admin 角色；不能修改/删除 super admin 角色。
- 新增测试：用户修改、角色分配、角色权限变更、角色删除。
- 测试结果：通过。
- 是否彻底解决：是。

## FIX-015 — 文件归属

- 问题编号：FIX-015 / SEC-016
- 严重程度：P1
- 文件路径：
  - `db/queries/storage/storage.sql`
  - `internal/modules/storage/application/service.go`
- 原始问题：列表、详情、下载、删除未校验 owner。
- 修复方法：列表 SQL owner 过滤；详情/Open/Delete 本人或超级管理员校验；文件组计数过滤。
- 新增测试：另一用户列表为空、详情/删除 404、owner 可访问、超级管理员可访问。
- 测试结果：通过。
- 是否彻底解决：是。

## FIX-016 — 公告收件范围

- 问题编号：FIX-016 / SEC-017
- 严重程度：P1
- 文件路径：
  - `db/queries/messaging/messaging.sql`
  - `internal/modules/notification/application/service.go`
  - `internal/modules/notification/adapter/http/handler.go`
- 原始问题：普通用户通过 ID 可读取不属于自己的公告详情。
- 修复方法：`GetMyNotice` 同时限定 notice、recipient 和 published；非 manager 统一使用 My。
- 新增测试：recipient 可读、非 recipient 404、manager 可读。
- 测试结果：通过。
- 是否彻底解决：是。

## FIX-017 — Session 撤销归属

- 问题编号：FIX-017 / SEC-018
- 严重程度：P1
- 文件路径：
  - `db/queries/identity/sessions.sql`
  - `internal/modules/identity/application/service.go`
- 原始问题：认证用户知道其他 Session ID 时可撤销他人会话。
- 修复方法：自助撤销使用 `id + user_id` 条件；跨用户返回 404。
- 新增测试：跨用户撤销失败且目标 Token 仍有效；本人其他 Session 可撤销。
- 测试结果：通过。
- 是否彻底解决：是。

## FIX-018 — 工作流幂等

- 问题编号：FIX-018 / SEC-019、SEC-020
- 严重程度：P1
- 文件路径：
  - `db/migrations/000018_workflow_business_idempotency.sql`
  - `internal/modules/workflow/adapter/postgres/repository.go`
  - `internal/modules/workflow/application/service.go`
- 原始问题：相同 businessId 可重复启动；并发审批第二次错误返回 500。
- 修复方法：唯一索引；唯一冲突 409；Decide 返回错误统一映射为 Conflict。
- 新增测试：并发启动、并发审批。
- 测试结果：通过，各恰好 1 成功。
- 是否彻底解决：是。

## FIX-019 — 调度事务

- 问题编号：FIX-019 / SEC-021
- 严重程度：P2
- 文件路径：`internal/modules/scheduler/adapter/river/backend.go`
- 原始问题：推进 next_run 后、投递 River 前的进程退出会丢任务。
- 修复方法：推进时间、创建 run、River InsertTx、回填 job ID 同一事务。
- 新增测试：编译和 Worker 启动集成；未构造进程崩溃注入测试。
- 测试结果：Worker 正常运行；事务代码路径通过静态和启动验证。
- 是否彻底解决：事务窗口已解决；未做故障注入，因此报告不标记为故障注入已验证。

## FIX-020 — Metrics 访问

- 问题编号：FIX-020 / SEC-022
- 严重程度：P2
- 文件路径：`internal/bootstrap/app.go`
- 原始问题：`/metrics` 匿名可访问。
- 修复方法：要求 `audit:read`。
- 新增测试：真实 API 匿名 401、管理员 200。
- 测试结果：通过。
- 是否彻底解决：是，Prometheus 需配置认证。

## FIX-021 — 生产镜像与 Helm 写权限

- 问题编号：FIX-021 / SEC-023
- 严重程度：P2
- 文件路径：
  - `Dockerfile`
  - `deploy/helm/mars-admin-go/templates/api.yaml`
  - `deploy/helm/mars-admin-go/templates/worker.yaml`
- 原始问题：只读根文件系统下 `/tmp` 不可写；timezone 数据缺失风险。
- 修复方法：镜像复制 zoneinfo；Pod 加 seccomp、UID/GID/fsGroup；API/Worker 挂载 `/tmp` emptyDir。
- 新增测试：distroless read-only + nonroot + tmpfs 运行 ready 200。
- 测试结果：通过。
- 是否彻底解决：是。

