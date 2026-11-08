# Java / RuoYi 残留审计

## 1. Java 语法和运行时残留

结论：未发现。

证据：

- 无 `pom.xml`、Gradle 文件、JAR、Spring Boot、MyBatis、Hibernate、JPA。
- `go.mod` 只包含 Go 依赖。
- 运行时为 Go 二进制，Docker 最终镜像不包含 JVM。

## 2. RuoYi 框架依赖残留

结论：未发现框架依赖。

项目没有导入 RuoYi Java 代码或运行时，但保留了部分旧 HTTP 路由、字段名和 `{code,message,data}` envelope，用于 Vue/UniApp 兼容。

兼容代码边界：

- `internal/platform/compat/envelope.go`
- 各模块 `LegacyRoutes` / `legacy*` Handler
- `internal/migration/legacy`

这些是迁移兼容层，不是 RuoYi 运行时。

## 3. Java 架构思想影子

### 已确认问题

- 大 Repository 接口和固定的 `domain/application/adapter` 目录有一定 Java 风格压力。
- 但 `domain` 不依赖数据库或 HTTP，`application` 直接使用 Go interface，adapter 明确，因此不能判定为机械照搬。
- 没有 `ServiceImpl`、`BaseService`、`BaseMapper`、Factory 容器或注解扫描。

### 兼容层中的 Java 风格

- `legacy*` Handler 中存在大量 `map[string]any` 响应拼装。
- 这是兼容旧前端的边界代码，应继续限制在 adapter 层。

## 4. 旧系统迁移兼容代码

### 允许保留

- `cmd/legacy-import`
- `internal/migration/legacy`
- `github.com/go-sql-driver/mysql`
- 旧字段映射、旧 ID 映射、旧文件复制和旧配置脱敏

### 风险

- 导入器按表分批事务，不是整库单事务；失败时会保留已成功表的数据。
- 读取表时一次性加载到内存，超大旧表需要改成分页/流式。
- 导入器必须继续作为离线迁移工具，不得挂载到 API/Worker 运行时。

## 结论

项目没有 Java/RuoYi 运行时残留。存在的是受控的历史兼容层和少量 Java 风格接口体量问题，不属于“Java 转 Go 失败”。
