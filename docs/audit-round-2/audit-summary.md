# Mars Admin Go 第二轮审计总结

审计日期：2026-11-08

## 范围

本轮审计基于现有 Go 模块化单体架构完成，没有更换技术栈，没有重建项目，也没有修改前端 UI。重点覆盖：

- Refresh Token 并发旋转、Token Family 与重放检测
- 短信验证码原子尝试、限流和账号枚举
- 密码变更、管理重置、冻结、删除后的 Session 撤销
- 可信代理、客户端 IP、CSRF、CORS、Request ID、Recovery、WebSocket
- RBAC 缓存一致性、超级管理员保护、Data Scope
- 文件、公告、聊天、工作流的数据归属和越权
- PostgreSQL 事务、唯一约束、组织树循环、调度幂等
- Docker、Helm、健康检查、非 Root、只读根文件系统和 graceful shutdown

## 结论

本轮记录 23 个 P0/P1/P2 问题，合并为 21 个修复交付组。最关键的认证并发缺陷已经由数据库原子 CAS 消除；权限缓存改为由 PostgreSQL epoch 作为权威版本；用户、角色、Session、文件、公告和工作流的高风险越权路径已增加服务端校验。

已完成的主要安全能力：

- 同一旧 Refresh Token 的 20 路并发请求恰好一个成功。
- 旧 Refresh Token 重放会撤销对应 Session Family。
- 短信错误次数逐次持久化，第 5 次错误后立即失效。
- SMS 并发正确验证码只允许一个成功。
- 发送限流与验证限流分离，并按手机号、IP、设备组合执行。
- Redis 故障时 SMS 安全失败，验证码不被消费。
- 密码变更、管理重置、冻结、删除与 Session 撤销在同一数据库事务完成。
- 普通管理员不能修改超级管理员用户或角色。
- 用户单条访问、更新、删除、重置密码和部门分配执行 Data Scope。
- 文件详情、下载、列表、删除执行本人或超级管理员校验。
- 公告详情和列表对普通用户只返回已发布且被定向投递的数据。
- 工作流业务键唯一，并发启动和审批只允许一个成功。
- 调度 tick 将推进下次时间、创建 run、写 River job 合并为同一事务。
- 权限缓存条目绑定数据库 epoch，Redis 中的旧权限不会被继续采用。
- `/metrics` 不再匿名公开。

## 严重程度统计

| 级别 | 已修复问题 |
| --- | ---: |
| P0 | 2 |
| P1 | 18 |
| P2 | 3 |

仍需部署配置或后续增强的事项单独列在 `remaining-risks.md`，不混入“已彻底修复”统计。

## 数据库变化

追加迁移，未修改历史迁移：

- `000016_identity_security_hardening.sql`
- `000017_social_login_safety.sql`
- `000018_workflow_business_idempotency.sql`

Compose 数据库已实际从 migration 15 升级到 18。

## 测试结论

- `go test ./...`：通过
- `go test -tags=integration ./test/integration -count=1`：通过，约 31 秒
- Linux `go test -race ./...`：通过
- `go vet ./...`：通过
- `golangci-lint run`：0 issues
- `govulncheck ./...`：无可达漏洞
- `sqlc generate`、`oapi-codegen`：通过
- `docker compose config`：通过
- `docker compose up --build -d`：通过
- API `/health/live`、`/health/ready`：200
- Worker River client 正常运行
- Helm lint/template：通过
- distroless 镜像非 Root + 只读根文件系统运行：ready 检查通过

完整命令和证据见 `verification-report.md`。

## 既有迁移证据

前序已完成的完整旧库导入证据保留在：

- `docs/database/import-validation.md`
- `docs/migration/full-import-report.json`
- `docs/migration/full-import-idempotency-report.json`

证据显示 397 名用户导入、0 失败；第二次执行 0 新增、0 失败。缺失的 9 个文件对象和 1 个孤立群组没有被伪造。

## 未宣称的事项

- 没有把外部短信、邮件、Push Provider 宣称为严格 exactly-once。
- 没有宣称 Helm 已包含生产 PostgreSQL、Redis、NetworkPolicy 或 PDB。
- 没有宣称所有历史业务表都已迁移到 Transactional Outbox。
- 没有宣称在未保留旧库源 dump 的工作区中重新执行过完整 MySQL 导入。

