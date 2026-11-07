# 权限与数据范围审计

## RBAC

API 权限仍由 `RequirePermission` 执行，菜单可见性不作为 API 授权依据。

本轮新增两类垂直权限保护：

1. 普通管理员不能修改超级管理员用户。
2. 普通管理员不能分配超级管理员角色，也不能修改/删除超级管理员角色、权限或菜单。

超级管理员仍可执行这些操作，数据库 epoch 会正常失效相关缓存。

## Data Scope

已覆盖：

- `all`
- `self`
- `department`
- `department_tree`
- `custom`

### 列表

`ListUsers` 把 actor 和 scope 下推到 PostgreSQL 查询。`custom` 使用角色允许部门关联，不依赖前端传值。

### 单条用户访问

新增 `GetUserByIDInDataScope`。管理员查看、更新、删除、重置密码、撤销 Session 时：

- 超级管理员：可访问全部。
- 普通管理员：目标必须落在其 Data Scope。
- 越界或不存在对普通管理员统一返回 403，减少目标存在性泄漏。

### 部门修改

`CountDepartmentsWithinScope` 在 SQL 中校验目标部门集合：

- `self` 不允许分配部门。
- `department` 只能分配 actor 所在部门。
- `department_tree` 只能分配 actor 部门树内部门。
- `custom` 只能分配角色显式允许部门。

请求中的 `departmentId` 不能直接绕过范围。

### 角色分配

普通管理员分配角色时，SQL 检查是否包含 `is_super_admin=true` 的角色。包含即拒绝。

## 权限缓存一致性

### 权威版本

数据库新增单例表 `access_cache_epoch`。权限相关表变化时由 statement/row trigger 原子递增 version。

覆盖：

- `access_roles`
- `access_menus`
- `access_user_roles`
- `access_role_permissions`
- `access_role_menus`
- `access_role_departments`
- `organization_memberships`
- `organization_departments`
- `identity_users.status/deleted_at`

### Redis 缓存

缓存值包含 epoch。读取流程：

1. 从 PostgreSQL 读取当前 epoch。
2. Redis hit 且 epoch 相同才使用。
3. epoch 不同、Redis 错误或反序列化失败时读取 PostgreSQL。
4. 新缓存值写入当前 epoch。

因此删除 Redis key 失败不会继续使用旧高权限数据。删除操作仍是性能优化，数据库 epoch 是安全权威。

### 验证

真实 PostgreSQL + Redis 测试：

- 用户获得 `user:read` 并写入缓存。
- 删除角色权限关联。
- 数据库 epoch 增加。
- Redis 中旧缓存仍存在。
- 再次认证时不再包含 `user:read`。

## 资源级授权

### 文件

- 文件列表在 SQL 中按 owner 过滤。
- 文件组计数按 owner 过滤。
- 详情、下载、预览、删除要求 owner 匹配或 actor 为超级管理员。
- 普通用户无法通过 UUID 读取其他用户文件。

### 公告

- 普通用户列表只返回 `notification_recipients` 中自己的已发布公告。
- 详情使用 `GetMyNotice` 校验 recipient。
- 拥有 `notice:manage` 或超级管理员才可查看管理范围、发送日志和任意公告详情。

### 聊天

- 单聊历史 SQL 同时限定双方用户 ID。
- 群组历史、成员、发送和权限操作执行成员角色检查。
- `clientMessageId` 唯一约束防止重复插入。

### 工作流

- Todo/Done 按 assignee 或 assignee role 查询。
- 审批前和事务内都校验 assignee。
- `(definition_id, business_id)` 唯一约束防止重复启动。
- 并发审批只允许一个从 `pending` 到 `completed`。

## 验证覆盖

真实数据库测试覆盖：

- Department scope 用户列表
- 越界用户详情 403
- 越权更新部门 403
- 普通管理员分配超级管理员角色 403
- 普通管理员修改超级管理员 403
- 普通管理员修改超级管理员角色 403
- 文件跨用户读取/删除 404
- 公告非 recipient 404
- Session 跨用户撤销 404
- 工作流并发启动/审批

## 残余边界

- Identity 用户管理、文件、公告、聊天和工作流已有资源级测试。
- 组织、存储分组、系统配置等管理接口主要依赖显式 RBAC 权限；其更细的部门级数据范围不是本轮新增范围。
- Data Scope 的 `custom` 语义以角色关联部门为准，不允许通过请求体覆盖。

