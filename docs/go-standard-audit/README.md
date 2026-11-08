# Mars Admin Go 工程规范审计

## 审计范围

审计基于提交 `a90745a` 及当前工作树，覆盖：

- `go.mod`、`go.sum`、`tools.go`
- `cmd/` 的 API、Worker、迁移、Outbox 和 legacy-import 入口
- `internal/bootstrap`、`internal/platform`、全部业务模块
- `db/migrations`、`db/queries`、sqlc 配置和数据库约束
- `api/openapi`、Dockerfile、Compose、Helm、脚本和测试
- 真实 PostgreSQL、Redis、MySQL 集成测试
- legacy MySQL 导入器及兼容层

未逐行重复审计 sqlc/oapi-codegen 自动生成代码，但检查了生成来源、生成命令和调用方式。

## 审计结论

当前项目已经不是 Java 代码的直接移植，整体方向符合 Go modular monolith：

- 进程入口与业务装配分离。
- Domain 包不依赖 HTTP、pgx 或 Redis。
- PostgreSQL 是业务事实源，SQL 和迁移明确。
- API、Worker、Migration、Outbox CLI 独立运行。
- Identity、Outbox、Workflow 已有并发和真实数据库测试。
- Docker 默认使用非 root distroless 镜像。

审计时确认的主要问题不在“代码像不像 Java”，而在生产边界：

- 少数模块通过其他模块的 sqlc 包直接读取数据。
- 部分 HTTP 中间件会破坏上传或造成大文件内存放大。
- 若干兼容 Handler 存在静默成功、分页截断和部分更新默认值错误。
- 部分配置错误会被静默替换为默认值。
- 软删除与全量唯一约束冲突。
- Codegen 预览可能生成不可编译代码。
- 支付回调目前只验签，没有订单状态落库和重放控制。

## 本轮已修复

- multipart 上传不再被全局 1 MiB JSON BodyLimit 截断。
- legacy 非 JSON 响应改为流式透传，不再整文件缓冲。
- Monitoring 接口统一要求 `audit:read`。
- 个人资料接口禁止修改 `userType` 和 `status`。
- 群主不能被管理员移除或降级，解散群不能继续发送。
- Workflow Terminate 使用实例锁和 `status='running'` CAS，并统一 Decide 锁顺序。
- 软删除记录可以重新使用 username、email、phone、role code、department code/path 和 position code。
- Settings 批量写入改为单事务，敏感值使用 base64 文本保存 AES-GCM 密文。
- Codegen 拒绝不存在的表，按实际字段生成 imports，写接口要求 `codegen:manage`。
- 通知 My 分页返回真实总数，read-all 覆盖全部收件记录，重复 read 幂等。
- Legacy role 按 ID 查询不再受 20 条分页截断。
- 非法环境变量不再静默回退。
- Redis RateLimit 改为固定窗口 Lua。
- 增加关键查询索引和真实依赖治理测试。

## 基线质量评价

| 维度 | 评价 |
| --- | --- |
| Go 原生程度 | 较高 |
| 模块化程度 | 较高，但仍有跨模块 adapter 依赖 |
| 数据库一致性 | 较高，事务覆盖广；存在少数非原子批量写 |
| 并发可靠性 | Identity 与 Outbox 较强；Workflow 终止存在竞态 |
| HTTP 安全 | 基础中间件完整；监控和兼容接口仍有缺口 |
| 测试覆盖 | 集成测试价值高，但多个核心模块单元覆盖为 0% |
| Java/RuoYi 残留 | 无 Java 运行时和框架依赖，存在兼容接口和 Java 风格接口体量 |
| 生产就绪度 | 尚未达到无条件生产上线标准 |

## 治理顺序

1. 修复 P1 安全、上传、下载、群组权限和工作流竞态。
2. 修复数据库软删除唯一约束和 Codegen 正确性。
3. 修复个人资料、配置解析、通知分页和兼容接口行为。
4. 为已修复路径补测试。
5. 保持跨模块数据库依赖和支付回调等大范围问题为明确残余风险。
