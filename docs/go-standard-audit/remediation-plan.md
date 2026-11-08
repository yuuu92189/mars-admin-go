# 渐进式修复计划

## P0

当前审计未确认可直接定级的 P0。没有发现无需认证即可绕过权限、无事务数据丢失或直接认证绕过。

## P1

### P1-001 文件上传上限失效 — 已修复

- 文件：`internal/platform/httpserver/middleware.go`、`internal/bootstrap/app.go`、storage Handler
- 问题：全局 1 MiB BodyLimit 覆盖 50 MiB 上传配置
- 修复：multipart 由 storage Handler 使用配置上限约束
- 验证：>1 MiB 上传测试

### P1-002 Legacy 下载内存放大 — 已修复

- 文件：`internal/platform/compat/envelope.go`
- 问题：所有旧 API 响应完整缓冲
- 修复：非 JSON 响应直接流式透传，JSON 响应才缓冲 envelope
- 验证：大文件下载流式测试

### P1-003 监控接口缺少权限 — 已修复

- 文件：`internal/modules/monitoring/http.go`
- 问题：只认证不鉴权
- 修复：`audit:read`
- 验证：无权限用户 403，有权限用户 200

### P1-004 个人资料可修改安全字段 — 已修复

- 文件：`internal/modules/identity/application/service.go`
- 问题：普通用户可修改 `userType`/`status`
- 修复：个人资料接口拒绝或忽略这两个字段
- 验证：普通用户 profile 更新后状态/类型不变

### P1-005 群主保护和解散群发送 — 已修复

- 文件：`db/queries/messaging/messaging.sql`、messaging repository/service
- 问题：管理员可移除群主；解散群仍可发送
- 修复：owner 行禁止移除/降级；MemberRole 要求 active group
- 验证：真实 PostgreSQL 群组测试

### P1-006 Workflow 终止竞态 — 已修复

- 文件：`internal/modules/workflow/adapter/postgres/repository.go`
- 问题：Terminate 未锁实例且无 status CAS
- 修复：`FOR UPDATE` + `WHERE status='running'`
- 验证：Terminate/Decide 并发集成测试

### P1-007 软删除唯一约束 — 已修复

- 文件：Goose 新迁移
- 问题：删除后无法复用 username/email/phone/role code/department code/path/position code
- 修复：部分唯一索引 `WHERE deleted_at IS NULL`
- 验证：删除后重建同名记录

### P1-008 Settings 敏感键和非原子更新 — 已修复

- 文件：system settings service/repository
- 问题：敏感键识别不一致；多 key 部分提交
- 修复：统一识别规则；批量 Upsert 单事务
- 验证：secret/token key 必须加密；中途失败全回滚

### P1-009 Codegen 正确性和权限 — 已修复

- 文件：codegen repository/application/handler
- 问题：不存在表返回空；生成模型有 unused imports；写接口只要求 read
- 修复：不存在表报错；按列条件生成 imports；写接口要求 `codegen:manage`
- 验证：生成代码 parse/compile；权限测试

### P1-010 通知分页和 read-all — 已修复

- 文件：notification repository/service/handler、messaging SQL
- 问题：My total 错误；legacy read-all 只处理首屏；重复 read 返回 404
- 修复：增加 My count；批量/全量 read；read 幂等
- 验证：>20 条公告分页和 read-all

### P1-011 Access legacy role 查询截断 — 已修复

- 文件：`internal/modules/access/adapter/http/handler.go`
- 问题：按 ID 查询先拉 1000，但 service 实际截断到 20
- 修复：增加按 ID 的 service/repository 查询
- 验证：第 21 个角色可读取

### P1-012 配置静默回退 — 已修复

- 文件：`internal/platform/config/config.go`
- 问题：非法整数、布尔和时长静默使用默认值
- 修复：解析错误显式返回
- 验证：非法环境变量启动失败

### P1-013 RateLimit 窗口被刷新 — 已修复

- 文件：`internal/platform/httpserver/ratelimit.go`
- 问题：INCR + EXPIRE 每次请求都刷新 TTL
- 修复：Lua 首次设置 TTL
- 验证：真实 Redis 固定窗口测试

## P2

- 缩小 Identity/Access/Messaging Repository 接口。
- 消除 Notification -> Messaging sqlc 跨模块依赖。
- 消除 Identity -> Access sqlc 跨模块依赖。
- 统一 PostgreSQL 错误映射为 SQLSTATE。
- 修复 Identity 关系加载错误被忽略。
- 修复 Audit 数组脱敏和写入错误可观测性。
- 减少 legacy Handler 中 `map[string]any` 拼装。
- 替换 `legacyFileConfigTest`、`legacyRefreshSettings`、`legacyEmptySMSLogs`、`legacyNoFiles` 等静默 stub。
- 改进 Legacy Import 的流式读取和内存上限。
- 增加 scheduler/storage/setting/notification 索引。
- 清理未使用的 generated API 或明确其契约用途。
- 修复 `scripts/smoke.sh` 对受保护 `/metrics` 的调用。

## 暂不扩大范围

- 不重写模块分层。
- 不替换 pgx、sqlc、Goose、River 或 Redis。
- 不删除 legacy 兼容路由。
- 不在没有订单模型的情况下虚构支付回调结算功能。
- 不将外部 Provider 宣称为 exactly-once。
