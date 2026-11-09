# 安全审计

## 认证

### 做得好的部分

- 密码使用 Argon2id，支持 bcrypt 迁移校验和 rehash。
- Access/Refresh Token 只保存 SHA-256 hash。
- Refresh Token 使用 CAS 轮换、历史记录、generation 和 grace period。
- 旧 Refresh Token 超出 grace period 重放会撤销 Session Family。
- Cookie 默认 HttpOnly；生产环境强制 Secure。
- 支持 TOTP MFA，密钥使用 AES-256-GCM。
- 登录失败有数据库锁定和 Redis 限流。

### 已确认问题

#### SEC-001 监控接口只校验登录 — 已修复

位置：

- `internal/modules/monitoring/http.go:33`
- `internal/modules/monitoring/http.go:47`

问题：

- `/api/v1/admin/monitoring/snapshot`、`dashboard` 和 legacy monitoring 接口只要求认证。
- 任何登录用户都可以看到数据库连接池、Go runtime、主机名、用户/消息/公告统计。

建议：

- 统一要求 `audit:read`，与菜单和 `/metrics` 权限保持一致。

#### SEC-002 个人资料允许修改 userType/status — 已修复

位置：

- `internal/modules/identity/application/service.go:650`

问题：

- 普通用户可通过 `PUT /api/v1/auth/profile` 提交不同的 `userType` 或 `status`。

影响：

- 用户可自行改变业务用户类型或状态。
- 权限仍由角色决定，但状态和类型是持久化安全属性，不能由普通资料接口修改。

建议：

- UpdateProfile 强制只允许 displayName、email、phone、avatar、gender。
- 状态和用户类型只能由受权限保护的管理接口修改。

#### SEC-003 群管理员可移除群主 — 已修复

位置：

- `db/queries/messaging/messaging.sql:99`
- `internal/modules/messaging/application/service.go` 的 `RemoveMember`

问题：

- SQL 只禁止 `member_id = actor_id`，没有禁止 `member_role = 'owner'`。
- 管理员移除群主时，actor 不是 owner，条件可通过。

建议：

- 群主成员行禁止被 RemoveMember 更新。
- AddGroupMember 不得把 owner 降级。

#### SEC-004 已解散群仍可发送消息 — 已修复

位置：

- `db/queries/messaging/messaging.sql:138`
- `internal/modules/messaging/application/service.go` 的 `SendGroup`

问题：

- MemberRole 只检查成员关系，没有检查 group.status。
- DissolveGroup 不会清空成员关系。

影响：

- 解散后的群成员仍可能通过旧成员关系发送消息。

#### SEC-005 Codegen 只使用 read 权限控制写操作 — 已修复

位置：

- `internal/modules/codegen/adapter/http/handler.go:30`

问题：

- import、update、delete、preview、sync、generate 全部使用 `codegen:read`。
- 数据库已有 `codegen:manage` 权限。

建议：

- GET 只读接口使用 `codegen:read`。
- 所有写/生成接口使用 `codegen:manage`。

#### SEC-006 Settings 敏感键识别不一致 — 已修复

位置：

- `internal/modules/system/application/settings.go:135`
- `internal/modules/system/adapter/http/handler.go:908`

问题：

- 两套 heuristic 不完全一致。
- `token`、`credential`、`access_key` 等形式可能漏判。

建议：

- 统一 `IsSensitiveSettingKey`。
- 敏感值必须要求 SecretBox，不允许明文降级。
- 已使用 base64 文本承载 AES-GCM 密文，避免二进制写入 PostgreSQL `TEXT` 字段失败。

## 授权与数据权限

- Access/Role/Menu 路由有服务端权限检查。
- Identity 用户列表将 Data Scope 下推到 PostgreSQL。
- Department Tree 支持 all/department/department_tree/custom/self。
- Storage Get 只允许 owner 或 super admin。
- Group 操作有成员角色校验，但 owner 保护需要修复。
- Workflow 读接口目前不做 Data Scope，任何 `workflow:read` 用户可读取全部实例。
- Codegen 写接口权限过宽。

## HTTP API

### 已确认问题

#### SEC-007 旧接口 envelope 缓冲大文件 — 已随兼容层删除

位置：

- historical: `internal/platform/compat/envelope.go:34`

历史问题：

- 删除前，所有 legacy `/api/*` 响应先完整写入 `bytes.Buffer`。
- 文件下载和 ZIP 下载会整文件进入内存。

历史影响：

- 大文件下载可能造成高内存占用或 OOM。

当前处理：兼容层已删除；canonical 文件/ZIP 下载直接流式写响应。

#### SEC-008 全局 1 MiB BodyLimit 阻止合法上传 — 已修复

位置：

- `internal/bootstrap/app.go` 的 `BodyLimit(1 << 20)`
- `internal/platform/httpserver/middleware.go:312`
- `internal/modules/storage/adapter/http/handler.go` 的 `receiveUpload`

问题：

- 存储配置允许 50 MiB，但外层 BodyLimit 先限制为 1 MiB。
- Handler 内部再设置上传上限也无法突破外层限制。

建议：

- JSON 请求保持 1 MiB。
- multipart 请求由 storage Handler 使用配置上限约束。

## 支付与外部 Provider

- 支付回调验签存在。
- 没有订单状态表、金额/订单号绑定、回调幂等账本或时间戳重放窗口。
- 当前支付回调只能证明签名有效，不能证明订单已结算。

## 日志与审计

- 日志普遍使用结构化字段。
- 未见直接记录密码或 Token。
- Audit metadata 会递归脱敏 map，但数组中的 map 未递归脱敏。
- 审计写入错误被忽略，数据库故障时没有告警。

## 安全结论

当前基础安全设计明显优于普通 CRUD 项目，但还不能宣称生产安全。P1 修复应优先处理监控鉴权、个人资料状态篡改、群主保护、上传/下载和 Codegen 权限。
