# 测试体系审计

## 当前测试

### 单元测试

已有：

- Workflow state machine 与 fuzz test
- Messaging application validation
- WebSocket origin 与 heartbeat
- Password、Token、SecretBox
- HTTP middleware、rate limit、compat envelope
- Config validation
- Integration provider protocol tests
- Storage local/S3 protocol tests
- Notification timeout

### 集成测试

`test/integration` 使用真实 Testcontainers：

- PostgreSQL 18.6
- Redis 8.10.1
- MySQL 8.0.36

覆盖：

- API 登录、CRUD、上传、消息、公告、角色、字典、调度、工作流
- Identity Refresh 并发、重放、CSRF、Data Scope
- Outbox lease、多 Worker、dead-letter、续租、崩溃恢复
- Legacy import 幂等

## 覆盖率缺口

`go test -cover ./...` 显示大量核心模块单元覆盖为 `0.0%`，包括：

- Access application
- Audit application
- Codegen application
- Identity application
- Notification application
- Organization application
- Scheduler application
- Storage application
- System application
- Workflow application
- 多数 PostgreSQL/River adapter

集成测试能覆盖关键链路，但不能替代单元错误分支测试。

## 本轮新增测试

- 2 MiB multipart 上传。
- Legacy 非 JSON 流式响应。
- Monitoring 权限。
- 个人资料状态/类型保护。
- 群主移除/降级保护和解散群发送拒绝。
- 软删除 username 复用。
- Codegen 不存在表和写权限。
- Settings 敏感值加密及解密。
- Notification 跨页总数和 read-all。
- 固定窗口 RateLimit。
- Workflow Terminate/Decide 并发一致性。

## 已确认缺口

1. 没有 Settings 多 key 部分失败注入测试，只验证了成功批量事务。
2. 没有支付回调重放/订单状态测试。
3. 没有 Access 第 21 个 legacy role 的 HTTP 级回归测试。
4. 没有大文件下载的实际内存上限测试，只验证了流式路径。

## 测试原则

- 保留真实 PostgreSQL 集成测试。
- 并发修复必须有真实数据库测试。
- HTTP 修复必须有 Handler 或 middleware 测试。
- 迁移修复必须有升级和回滚验证。
- 不为覆盖率写无价值断言。
