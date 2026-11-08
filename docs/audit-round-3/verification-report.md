# 第三轮验证报告

验证日期：2026-11-08

## Go 单元测试

```text
gofmt -w .
go test ./...
```

结果：通过。

## 全量真实集成测试

```text
go test -tags=integration ./... -count=1
```

结果：通过。

包含：

- PostgreSQL 18.6
- Redis 8.10.1
- MySQL 8.0.36 legacy import
- API 兼容测试
- Refresh 并发/宽限/重放
- SMS 原子尝试
- RBAC/Data Scope
- 文件删除 Outbox
- 公告 Outbox
- 消息 Outbox
- 工作流 Outbox
- Outbox relay 故障矩阵
- Scheduler River 失败回滚

## Race

宿主机 Windows 缺少可用 GCC 路径，使用 Linux Go 容器：

```text
docker run --rm \
  -v C:/Users/admin/Desktop/admin/mars-admin-go:/src \
  -v mars-admin-go-mod:/go/pkg/mod \
  -v mars-admin-go-build:/root/.cache/go-build \
  -w /src golang:1.27.3-bookworm \
  sh -c "go test -race ./..."
```

结果：通过。

## 静态检查

```text
go vet ./...
golangci-lint run
govulncheck ./...
```

结果：

- go vet：通过
- golangci-lint：0 issues
- govulncheck：调用路径 0 漏洞

govulncheck 仍报告依赖模块中的不可达漏洞记录，未误报为调用路径漏洞。

## 代码生成

```text
CGO_ENABLED=0 go tool sqlc generate
CGO_ENABLED=0 go tool oapi-codegen \
  -config api/oapi-codegen.yaml \
  api/openapi/v1/openapi.yaml
```

结果：通过。

## Compose

```text
docker compose config --quiet
docker compose up --build -d
docker compose ps
```

结果：

- PostgreSQL healthy
- Redis healthy
- API healthy
- Worker running
- migration version = 20
- Worker 三个 River client 正常启动：
  - Scheduler
  - Notification
  - Outbox

健康检查：

```text
GET /health/ready -> 200
```

Outbox CLI：

```text
docker compose run --rm --no-deps \
  --entrypoint /usr/local/bin/outbox \
  api list pending 5
```

结果：命令成功执行。

## 生产 distroless 镜像

```text
docker build -t mars-admin-go:audit .
```

运行：

- `--read-only`
- `--tmpfs /tmp`
- `--tmpfs /app/uploads`
- UID/GID 65532
- 真实 PostgreSQL/Redis

结果：`/health/ready` 200。

镜像包含：

- api
- worker
- migrate
- outbox
- legacy-import
- CA 证书
- zoneinfo

## Helm

```text
helm lint deploy/helm/mars-admin-go
helm template ...
```

结果：

- 1 chart linted，0 failed
- API/Worker/Service 成功渲染
- Pod 非 Root、seccomp、只读根文件系统、`/tmp` emptyDir

## 前端

外部 Vue 工程：

```text
cd ../mars-admin-master/mars-ui
npm run build
```

结果：通过。

警告：

- Sass legacy JS API deprecation
- 多个 chunk 超过 500kB

没有 TypeScript/Vite 编译失败。

## 明确未验证/未宣称

- 外部 SMS/Email/Push Provider 严格 exactly-once。
- 浏览器多标签端到端自动化测试；服务端并发已覆盖。
- 多 Provider 文件删除路由。
- 字典/系统配置多租户范围。

