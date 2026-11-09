# Java / RuoYi 残留审计

Updated: 2026-11-09

## 1. Java 语法和运行时残留

结论：未发现。

证据：

- 无 `pom.xml`、Gradle 文件、JAR、Spring Boot、MyBatis、Hibernate 或 JPA。
- `go.mod` 只包含 Go 依赖。
- 运行时为 Go 二进制，Docker 最终镜像不包含 JVM。

## 2. 在线兼容层

结论：已删除。

- `internal/platform/compat` 已删除。
- `internal/bootstrap/app.go` 不再挂载 `/api/*` legacy 路由。
- 各模块 HTTP adapter 不再包含 `LegacyRoutes`、`legacy*` handler、Java payload alias 或
  `{code,message,data}` 响应拼装。
- React 只调用 canonical `/api/v1/*`，使用直接 JSON、UUID 字符串、canonical pagination 和
  RFC 9457 Problem。
- 集成测试对代表性旧路径断言 `404`。

## 3. 离线迁移边界

以下代码允许保留，因为它们不进入在线 API/Worker 运行时：

- `cmd/legacy-import`
- `internal/migration/legacy`
- 旧 ID 映射、旧字段映射、旧文件复制和旧配置脱敏逻辑
- 生成 sqlc 包中的 `LegacyIDMap` / `LegacyImportRun` 模型

风险：

- 导入器按表分批事务，失败时保留已成功表的数据。
- 读取表时一次性加载到内存，超大旧表需要分页或流式改造。
- 导入器必须继续作为显式离线命令运行，不得挂载为 HTTP API。

## 4. Java 架构思想影子

仍有少量 Go interface 体量偏大的 P2 问题，但没有 `ServiceImpl`、`BaseService`、`BaseMapper`、
Factory 容器或注解扫描，也没有 Java 包结构机械复制。

## 结论

项目不存在 Java/RuoYi 运行时残留。在线 Java HTTP 兼容层已删除，离线迁移工具保持隔离。
