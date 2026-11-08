# P2 数据范围与剩余风险复查

## 用户

已有：

- `all`
- `self`
- `department`
- `department_tree`
- `custom`

用户列表、详情、更新、删除、重置密码、撤销 Session 均在服务层和 SQL 层执行范围校验。

## 部门

本轮新增：

- 部门列表 SQL scope 过滤。
- 单部门 SQL scope 过滤。
- 根部门创建仅允许 `all`。
- 非 `all` 创建子部门时父级必须在范围内。
- 更新目标和新父级必须在范围内。
- 删除目标必须在范围内。

代码：

- `db/queries/organization/organization.sql`
- `internal/modules/organization/application/service.go`
- `internal/modules/organization/adapter/postgres/repository.go`
- `internal/modules/identity/application/service.go`

测试：

- department scope 用户只能看到所属部门。
- `all`、`self`、`department_tree`、`custom` 均有真实 PostgreSQL 测试。
- 其他部门详情返回 404。
- 创建 root 返回 403。
- 在范围外父部门下创建返回 403。
- 将部门移动到范围外父级返回 403。
- 组织树循环仍被拒绝且事务回滚。

## 字典与系统配置

审计结论：

- 字典和系统配置是全局单例资源，没有 department owner 字段。
- 资源级 Data Scope 不应被伪造为部门范围。
- 继续使用 `dictionary:*`、`settings:*` RBAC 权限。
- 菜单可见性不作为 API 授权。

## 文件

- 列表 SQL owner 过滤。
- 详情/下载/删除 owner 或超级管理员。
- 删除元数据写 Outbox。
- Worker 异步删除对象，失败自动重试。

## 公告

- 普通用户只读取自己的 recipient 记录。
- manager/super admin 可读取管理范围。
- 发布、recipient、station log 和 Outbox 同事务。

## 聊天

- 单聊历史限定双方 ID。
- 群聊历史/操作执行成员和角色校验。
- 消息插入与 Outbox 同事务。

## 工作流

- Todo/Done 和审批执行 assignee 校验。
- 并发审批使用 task/instance 行锁。
- businessId 唯一。
- 审批事件写 Outbox。

## Refresh History

- 新增 retention 配置。
- Worker 启动和每 6 小时清理已终止 Family 的历史。
- 活跃 Family 历史保留。

## Scheduler

- due job、next_run、run、River InsertTx 同一事务。
- 新增真实 PostgreSQL 失败测试：River insert 失败时 next_run 和 run 全部回滚。

## WebSocket

- 服务端每 30 秒发送 WebSocket Ping。
- 读超时保持 90 秒。
- 客户端库自动回 Pong 即可维持连接。
- 测试覆盖 Ping 周期后仍可收发业务消息。

## Metrics

- `/metrics` 需要 `audit:read`。
- 真实 API 测试：匿名 401，管理员 200。
- Prometheus 应使用只读监控账号的 Bearer Token。

## 前端 API

- Go 真实 API 集成测试通过。
- Vue 生产构建通过。
- 当前没有修改前端 UI。
- 构建只有 Sass legacy API 和大 chunk 警告，没有 TypeScript/Vite 编译错误。

## 仍需后续处理

- 字典/系统配置若要支持多租户，需要新增明确 owner/tenant 模型，不能仅靠 Data Scope 猜测。
- 岗位目前作为全局组织元数据，仅执行 RBAC。
- 外部通知 Provider 的严格 exactly-once 仍需 Provider 幂等键。

