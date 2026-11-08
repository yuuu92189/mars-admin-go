# 数据库、SQL 与事务审计

## 技术栈

- PostgreSQL 18.6
- pgx/v5
- sqlc
- Goose
- ltree、citext、GIN、部分索引

整体方向符合 PostgreSQL SQL-first，没有 ORM。

## 已确认问题

### DB-001 软删除与唯一约束冲突 — 已修复

位置：

- `db/migrations/000001_identity.sql:23`
- `db/migrations/000002_access.sql:14`
- `db/migrations/000003_organization.sql:18`
- `db/migrations/000003_organization.sql:19`
- `db/migrations/000003_organization.sql:38`

问题：

- 用户名、邮箱、手机号、角色 code、部门 code/path、岗位 code 使用全量唯一约束。
- 对应表存在 `deleted_at` 软删除字段。

影响：

- 删除后无法重新创建同名/同 code 记录。
- 旧数据迁移和账号重建会受到阻塞。

建议：

- 使用 `WHERE deleted_at IS NULL` 的部分唯一索引。
- 迁移时必须保留当前活动数据约束。

已通过 `db/migrations/000022_soft_delete_uniqueness.sql` 完成。

### DB-002 Codegen 表不存在时返回空结构 — 已修复

位置：

- `internal/modules/codegen/adapter/postgres/repository.go:45`

问题：

- 第一条表描述查询错误被忽略。
- 表不存在时仍返回空列列表，后续可保存无效导入记录。

建议：

- 检查 `pgx.ErrNoRows` 并返回明确错误。

### DB-003 系统设置批量更新不是单事务 — 已修复

位置：

- `internal/modules/system/application/settings.go:93`
- `internal/modules/system/adapter/postgres/repository.go:221`

问题：

- 多个 setting key 逐个调用 `UpsertSetting`。
- 每个 key 独立提交事务。

影响：

- 中途失败会留下部分设置已更新、部分未更新。

建议：

- 在 repository 增加 `UpsertSettings`，一次事务写入全部 key、revision 和缓存失效边界。

### DB-004 部分索引缺失 — 已修复

已确认的高频查询：

- Scheduler tick：`status = 'active' AND next_run_at <= now()`
- Storage cleanup：`status = 'deleted' AND deleted_at < ...`
- Notification send logs：按 `notice_id` 查询
- Setting revisions：按 `setting_id` 查询

影响：

- 数据量增长后会出现全表扫描。

建议：

- 已增加 scheduler 活动到期部分索引。
- 已增加 storage 软删除清理索引。
- 已增加 notification send log、setting revision、workflow role todo 和 audit login username 索引。

已通过 `db/migrations/000023_query_indexes.sql` 完成。

## 事务和并发

### 做得好的部分

- 用户创建、角色分配、组织关系写入在单事务内。
- 工作流决策锁定任务和实例。
- 公告发布、收件人和 Outbox 在单事务内。
- Outbox 与 River 插入在单事务内。
- Refresh Token Rotation 使用旧 hash CAS。
- Scheduler tick 更新和 River 插入在单事务内。

### 需要修复

- Workflow Terminate 没有 `FOR UPDATE`，状态更新没有 `status='running'` 条件。
- Settings 多 key 更新不是单事务。

## N+1 和重复查询

- Messaging legacy group list 对每个 group 再查 members，存在 N+1。
- Identity ListUsers 不加载角色/部门，避免 N+1，但详情查询会追加多次查询。
- Monitoring Dashboard 顺序执行 10 条 count 查询，可改为单条聚合。

## Migration 审计

- 迁移按 Goose 编号递增，未发现历史迁移被修改。
- 当前最新版本为 `23`。
- 迁移 Down 基本对称。
- 需要新增部分唯一索引迁移，不能直接修改历史迁移。

## 连接池

- Max/Min connections 有配置。
- 连接超时 10 秒，健康检查 2 秒。
- 需要增加对 0/负时长配置的显式拒绝。
