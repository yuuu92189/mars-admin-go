# 数据库与事务审计

## Migration

历史 migration 未修改。新增：

### `000016_identity_security_hardening.sql`

- Session refresh family 与 generation
- Refresh token history
- One-time code max_attempts
- Access cache epoch 与触发器

### `000017_social_login_safety.sql`

- `identity_users.password_login_enabled`

### `000018_workflow_business_idempotency.sql`

- 唯一索引 `(definition_id, business_id)`

当前 Compose 数据库已从 15 实际升级到 18。

升级前检查了已存在 workflow 业务键重复数，结果为 0，唯一索引可安全创建。

## 事务边界

### 密码

`ChangePasswordAndRevokeSessions`：

- 更新密码
- 重新启用密码登录
- 撤销全部 Session
- 同一事务提交

### 用户更新

`UpdateUser`：

- 乐观锁版本校验
- 用户字段
- 角色关联
- 部门关联
- 岗位关联
- 非 active 状态 Session 撤销
- 同一事务提交

### 用户删除

软删除与 Session 撤销在同一事务完成。

### 社交用户

创建用户和绑定 `(provider, subject)` 在同一事务完成。唯一约束冲突会回滚用户创建。

### 社交解绑

先 `SELECT ... FOR UPDATE` 锁定用户行，再检查剩余认证方式并删除目标身份。并发解绑不会同时移除最后两种认证方式。

## 工作流

### 启动

- Definition 必须 published。
- `businessId` 最大 120。
- 唯一索引保证相同 definition/businessId 只创建一个实例。
- 唯一冲突映射为 HTTP 409。

### 审批

- `SELECT ... FOR UPDATE OF t, i` 锁定 task 和 instance。
- task 必须 pending，instance 必须 running。
- 再次校验 assignee。
- 完成 task、更新 instance、取消剩余 task、创建下一 task、写 event 在同一事务。
- task 已完成时映射为 409，而不是 500。

### 验证

真实 PostgreSQL 测试同时发起：

- 2 个相同 businessId 的 Start：1 成功、1 冲突。
- 2 个相同 task 的 Approve：1 成功、1 冲突。

## 调度器

原 tick 流程在提交 `next_run_at` 后才创建 run 和 River job，进程退出会造成遗漏。

新流程在同一个 PostgreSQL 事务内完成：

1. `SELECT ... FOR UPDATE SKIP LOCKED` 选取 due jobs。
2. 推进 `last_enqueued_at` 和 `next_run_at`。
3. 插入 `scheduler_job_runs`。
4. 使用 River `InsertTx` 写入任务。
5. 回填 `river_job_id`。
6. 提交。

任一步失败都回滚，下一次 tick 仍可重新处理 due job。

## 组织树

部门更新在事务内：

- 锁定当前部门。
- 计算新 path。
- 校验新父路径不以当前旧 path 为祖先。
- 更新当前部门。
- 同步更新子树 path。

真实测试验证父部门移动到子部门会被拒绝，事务回滚后 parent 保持为空。

## 唯一约束与重复

- 用户名、邮箱、手机号：数据库唯一约束
- 社交 provider/subject：数据库唯一约束
- 会话 access/refresh hash：数据库唯一约束
- Refresh history hash：数据库唯一约束
- 消息 `clientMessageId`：唯一约束
- Workflow definition/businessId：新增唯一索引

## SQL 安全

- 稳定查询使用 sqlc 参数化。
- 列表排序使用固定 CASE 白名单。
- Data Scope 条件在 SQL 内执行。
- 动态 SQL 搜索仅使用参数绑定，没有字符串拼接用户输入。

## 旧库迁移证据

已有完整导入报告：

- 首次：397 用户，0 failed。
- 二次：0 inserted，0 failed，2,605 skipped。
- `legacy_id_map`: 1,359。
- 孤立 user-role: 0。
- 孤立 role-menu: 0。

缺失对象仍保持缺失状态：

- 9 个 legacy 文件对象不在提供的上传目录。
- 1 个群组 owner 在旧库用户集缺失。
- 2 个岗位父级缺失，按 root 导入。

## 残余风险

- Refreshtoken history 尚无归档/分区策略。
- 尚未全面引入 Transactional Outbox。
- 外部通知 provider 的副作用不能由本地 PostgreSQL 单独保证 exactly-once。
- 文件对象删除失败时，元数据已软删除但对象可能残留，属于安全优先的已知折衷。

