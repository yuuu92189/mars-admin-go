# 生产部署与可观测性审计

## Docker

默认 runtime：

`gcr.io/distroless/static-debian13:nonroot`

已验证：

- 运行用户 `65532:65532`
- 非 Root
- 无 shell
- CA 证书存在
- 复制 `/usr/share/zoneinfo`
- `/app/uploads` 属于非 Root 用户
- 健康检查使用 API 二进制 `-healthcheck`

Compose 开发构建使用 Alpine runtime，但同样以 `65532:65532` 运行。

生产 distroless 镜像额外使用：

- `--read-only`
- `--tmpfs /tmp`
- `--tmpfs /app/uploads`

在该组合下 API 成功连接现有 PostgreSQL/Redis，`/health/ready` 返回 200。

## API 生命周期

- HTTP Server 有 ReadHeaderTimeout、ReadTimeout、WriteTimeout、IdleTimeout。
- WebSocket Upgrade 不再被 45 秒 HTTP Timeout 包裹。
- App.Close 先调用 Hub.Close。
- Hub.Close 向所有连接发送 Going Away close，并等待连接注销。
- HTTP server shutdown 先停止接入，应用关闭再释放 Hub、Telemetry、Redis、PostgreSQL。

## Worker

Compose 实际重建并运行 Worker：

- River client 启动
- Scheduler tick 完成
- PostgreSQL migration 版本为 18
- 无启动错误

Worker 的 Scheduler 和 Notification 都使用 River，并具有 stop context。

## Health

- `/health/live`：进程存活
- `/health/ready`：同时检查 PostgreSQL 与 Redis
- API Compose healthcheck：healthy

实际结果：

```text
/health/live  -> 200
/health/ready -> 200
```

## Compose

已执行：

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
- Migration 18

## Helm

Chart 增强：

- Pod `runAsNonRoot`
- UID/GID 65532
- `fsGroup`
- `seccompProfile: RuntimeDefault`
- container 禁止提权
- 丢弃全部 capabilities
- `readOnlyRootFilesystem`
- `/tmp` emptyDir
- readiness/liveness
- 生产配置默认 `AUTH_COOKIE_SECURE=true`
- 生产存储默认 `STORAGE_PROVIDER=s3`

校验：

- `helm lint`：0 failed
- `helm template`：成功渲染 API/Worker Deployment 和 Service

## Metrics 与内部接口

- `/metrics` 从匿名公开改为需要 `audit:read`。
- Prometheus 需要配置管理员 Bearer Token 或受控反向代理认证。
- HTTP metrics 使用 chi route pattern，未使用原始路径作为 label。
- 仓库未发现 pprof 路由。

## 依赖与漏洞

- `govulncheck ./...`：调用路径中 0 漏洞。
- 仍有依赖模块中的不可达漏洞记录，不应被解释成调用代码受影响。

## 部署方必须配置

Helm 不内置生产 PostgreSQL、Redis、S3、Ingress 控制器或 secret。部署方必须提供：

- `DATABASE_URL`
- `REDIS_URL`
- `BOOTSTRAP_ADMIN_PASSWORD`
- `SETTINGS_ENCRYPTION_KEY`
- S3 bucket/region/credentials
- 反向代理 CIDR
- Prometheus 抓取认证

反向代理 CIDR 留空是安全默认值，但会让 API 看到代理 IP，从而可能使限流按代理聚合。生产必须明确填写 Ingress Controller 或负载均衡器的可信网段。

## 未包含

- NetworkPolicy
- PodDisruptionBudget
- 自动数据库备份策略
- 自动 Redis HA
- 证书自动签发
- Chart 内置 migration Job

这些属于部署平台职责，当前代码没有伪称已完成。

