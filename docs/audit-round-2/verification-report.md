# 验证报告

验证日期：2026-11-08

## Go 单元测试

命令：

```text
gofmt -w .
go test ./...
```

结果：通过。

## 真实集成测试

命令：

```text
go test -tags=integration ./test/integration -count=1
```

结果：通过，约 31 秒。

覆盖：

- 真实 PostgreSQL、Redis、MySQL Testcontainers
- 完整 API 登录、Refresh、CSRF/Cookie、WebSocket、消息、文件、公告、权限和兼容接口
- 20 路 Refresh 并发
- Refresh 重放
- 多设备改密/重置/冻结/删除
- Session 跨用户撤销
- 20 路社交首登
- 最后认证方式保护
- 权限缓存 epoch
- Department Data Scope
- 超级管理员角色保护
- 文件跨用户访问
- 公告定向投递
- 工作流启动与审批并发
- 组织树循环回滚
- SMS attempts、并发、过期、已消费、IP 限额、Redis 故障
- 旧库导入及 BCrypt 升级

## Race Detector

宿主机 Windows 缺少可用 GCC 路径，因此使用 Linux 容器执行。

命令：

```text
docker run --rm \
  -v C:/Users/admin/Desktop/admin/mars-admin-go:/src \
  -v mars-admin-go-mod:/go/pkg/mod \
  -v mars-admin-go-build:/root/.cache/go-build \
  -w /src golang:1.27.3-bookworm \
  sh -c "go test -race ./..."
```

结果：通过。

## 静态分析

### go vet

```text
go vet ./...
```

结果：通过。

### golangci-lint

```text
golangci-lint run
```

结果：`0 issues`。

### govulncheck

```text
govulncheck ./...
```

结果：

- 调用路径可达漏洞：0
- 模块中存在但当前代码不可达的漏洞：8
- 导入包中存在但不可达的漏洞：1

未把不可达漏洞误报为代码漏洞。

## 代码生成

命令：

```text
CGO_ENABLED=0 go tool sqlc generate
CGO_ENABLED=0 go tool oapi-codegen \
  -config api/oapi-codegen.yaml \
  api/openapi/v1/openapi.yaml
```

结果：通过。

新增社交解绑 API 后，重新生成 `generated/api/api.gen.go`。

## Compose

命令：

```text
docker compose config --quiet
docker compose up --build -d
docker compose ps
```

结果：

- Compose 配置有效
- PostgreSQL healthy
- Redis healthy
- API healthy
- Worker running
- migration version = 18

健康检查：

```text
GET /health/live  -> 200
GET /health/ready -> 200
```

Worker 日志：

- River client started
- River worker started
- Scheduler tick completed

## 生产 distroless 镜像

构建：

```text
docker build -t mars-admin-go:audit .
```

镜像配置：

```text
User=65532:65532
ReadonlyRootfs=true
```

运行测试：

- `--read-only`
- `--tmpfs /tmp`
- `--tmpfs /app/uploads`
- 连接现有 PostgreSQL/Redis
- `/health/ready` 返回 200

结果：通过。

## Helm

宿主机未安装 Helm CLI，因此使用固定版本容器：

```text
docker run --rm \
  -v .../deploy/helm/mars-admin-go:/chart \
  alpine/helm:3.18.6 lint /chart
```

结果：

```text
1 chart(s) linted, 0 chart(s) failed
```

`helm template`：

- API Deployment
- Worker Deployment
- Service

均成功渲染。

## 旧库导入证据

本轮没有使用固定数据或伪造对象。已有完整导入产物：

- 首次导入：397 用户，0 failed
- 二次导入：0 inserted，0 failed，2,605 skipped
- `legacy_id_map`: 1,359
- orphan user-role: 0
- orphan role-menu: 0

报告：

- `docs/migration/full-import-report.json`
- `docs/migration/full-import-idempotency-report.json`
- `docs/database/import-validation.md`

缺失的 9 个文件和 1 个孤立群组仍明确记录，没有生成假对象。

## 未执行项

- 当前仓库没有 Vue/UniApp 源码，未运行前端构建。
- 未重新执行完整 MySQL dump 导入，因为本轮工作区没有保留源 dump/上传目录；使用并核验了已有真实执行报告。
- 未执行 Scheduler 进程强杀故障注入。
- 未进行外部 SMS/Email/Push Provider 的 exactly-once 认证。

