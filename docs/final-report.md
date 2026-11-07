# Final Engineering Report

```text
项目名称：
Mars Admin Go Edition

开发语言：
Go 1.27.3

数据库：
PostgreSQL 18.6

缓存：
Redis 8.10.1

架构：
Modular Monolith + Feature-oriented + Pragmatic DDD

API：
REST + OpenAPI 3.1

数据访问：
pgx v5.12.0 + sqlc v1.30.0

异步任务：
River v0.48.0

实时通信：
coder/websocket v1.8.15 + Redis Pub/Sub + PostgreSQL catch-up

构建：
Docker Multi-stage

测试：
Unit + Integration + Contract + Security + Race + Fuzz + Benchmark
```

## Implemented Modules

- Identity and session authentication
- Access control and dynamic menu tree
- Organization
- Dictionary
- Settings
- Audit
- Monitoring
- Storage
- Messaging and WebSocket
- Notification
- Scheduler and River Worker
- Workflow
- Codegen
- Integration provider contracts
- Legacy HTTP compatibility

## Passed Checks

- `gofmt`
- `go vet ./...`
- `go test ./...`
- `go test -race ./...` in the Go 1.27.3 Linux container
- `golangci-lint run`
- `govulncheck ./...`
- sqlc and oapi-codegen generation
- OpenAPI validation test
- Testcontainers PostgreSQL/Redis integration test
- 3-second workflow fuzz test
- Argon2id benchmark
- Docker multi-stage build
- Compose PostgreSQL/Redis/API/Worker smoke test
- Goose migrations through version 15
- complete legacy dump import into an isolated PostgreSQL validation database
- second full import run with zero inserted/failed rows
- Vue production build and page-by-page browser compatibility run

## Not Completed

- Live third-party SMS/payment/social/push/mail integrations
- Live S3/MinIO/OSS bucket certification
- Approved production legacy data cutover
- UniApp runtime E2E against a real WeChat toolchain

## Known Issues

- GCR is unreachable from this host; the default distroless runtime image could not be pulled here. Compose uses an Alpine runtime fallback for local verification while the production Dockerfile default remains distroless.
- Helm is provided as a chart but could not be linted because Helm is not installed on this host.
- Legacy token-in-header and WebSocket query-token compatibility remain until the frontends migrate to cookies.
- The legacy SQL dump contains exposed credentials and personal data. It has only been imported into an isolated validation database and must be rotated/minimized before production cutover.
- Nine legacy file records reference objects missing from the supplied upload tree; their metadata was skipped. One orphan chat group was skipped, and two positions with missing parents were imported as roots.

## Start

```bash
cd C:\Users\admin\Desktop\admin\mars-admin-go
docker compose up --build
```

API: `http://localhost:8080`  
Docs: `http://localhost:8080/docs`  
Readiness: `http://localhost:8080/health/ready`

Existing Vue admin UI (already running in this environment):

```bash
cd ..\mars-admin-master\mars-ui
npm run dev
```

UI: `http://127.0.0.1:3000`
