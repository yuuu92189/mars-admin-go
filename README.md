# Mars Admin Go Edition

Mars Admin Go Edition is an independent Go 1.27.3 backend rebuilt from the audited Mars Admin Java project. It uses a modular monolith, PostgreSQL 18.6, Redis 8.10.1, pgx, sqlc, River, OpenAPI 3.1, WebSocket and Docker.

The original Java project is retained at `../mars-admin-master` as migration reference and is never imported or started by this service.

Current verified status and residual gaps are recorded in [`docs/status.md`](docs/status.md), [`docs/test-report.md`](docs/test-report.md) and [`docs/final-report.md`](docs/final-report.md).

The unified first-round frontend/backend architecture audit, Vue page/component inventory, design and animation analysis, API mapping, target architecture and implementation roadmap are in [`docs/architecture-audit/README.md`](docs/architecture-audit/README.md).

Phase 1 implementation results, API contract changes and verification evidence are in [`docs/architecture-audit/12-phase-1-implementation-report.md`](docs/architecture-audit/12-phase-1-implementation-report.md).

Phase 2 frontend foundation, design-system reconstruction, API/auth integration and acceptance evidence are in [`docs/frontend/README.md`](docs/frontend/README.md).

Phase 2.1 frontend hardening, automatic refresh, Vue-aligned appearance/Shell behavior, visual
comparison and final acceptance are in
[`docs/frontend/13-phase-2.1-implementation-report.md`](docs/frontend/13-phase-2.1-implementation-report.md)
and [`docs/frontend/14-phase-2.1-final-acceptance.md`](docs/frontend/14-phase-2.1-final-acceptance.md).

The latest Refresh concurrency and transactional Outbox audit is in [`docs/audit-round-3/audit-summary.md`](docs/audit-round-3/audit-summary.md).

The current Go-standard audit and remediation plan are in
[`docs/go-standard-audit/README.md`](docs/go-standard-audit/README.md).
The long-term project standard is [`docs/GO_ENGINEERING_STANDARD.md`](docs/GO_ENGINEERING_STANDARD.md).

## Quick Start

```bash
cp .env.example .env
docker compose up --build
```

The API starts at `http://localhost:8080`. Health endpoints:

- `GET /health/live`
- `GET /health/ready`
- `GET /metrics` (requires `audit:read`)
- `GET /docs`
- `GET /openapi.yaml`

To create a local bootstrap administrator, set a non-empty password before first start:

```bash
BOOTSTRAP_ADMIN_PASSWORD='choose-a-long-local-password' docker compose up --build
```

The bootstrap operation is idempotent and assigns the database-backed `admin` super-administrator role. It is not a hard-coded production credential.

## Local Development

```bash
go mod download
go tool sqlc generate
go tool oapi-codegen -config api/oapi-codegen.yaml api/openapi/v1/openapi.yaml
go run ./cmd/migrate up
go run ./cmd/api
go run ./cmd/worker
```

For the Phase 2 React frontend:

```bash
cd frontend
pnpm install --frozen-lockfile
pnpm dev
```

The Next.js app runs at `http://127.0.0.1:3000` and proxies canonical `/api/v1/*` through its same-origin Route Handler to the Go API. The Vue reference remains at `scaffold-master-main/scaffold-vue` for visual comparison only.

Set the single server-side API target when it is not the default:

```bash
cd frontend
API_BASE_URL=http://localhost:8081 pnpm dev
```

Required environment variables are documented in `.env.example`. `DATABASE_URL` must be a PostgreSQL URL. Production startup rejects unsafe cookie combinations and does not silently replace missing provider credentials.

## Architecture

- `cmd/api`, `cmd/worker`, `cmd/migrate`, `cmd/outbox`, `cmd/legacy-import`: separate process entry points.
- `internal/modules/*`: feature-owned domain, application, HTTP adapter and PostgreSQL adapter code.
- `internal/platform/*`: configuration, security primitives, HTTP middleware, database, Redis, telemetry and compatibility envelope.
- `db/migrations`: Goose schema migrations.
- `db/queries`: sqlc query sources, generated into module-local packages.
- `api/openapi/v1/openapi.yaml`: canonical API contract.

Authentication uses opaque server-side sessions and Argon2id passwords. Redis is limited to ephemeral coordination, rate limiting and realtime fan-out. PostgreSQL is the source of truth.

Cross-process domain events use a PostgreSQL transactional outbox. The Worker relay claims events with `SKIP LOCKED`, inserts River jobs and updates outbox state in one transaction, and retries consumers with exponential backoff. `go run ./cmd/outbox list|requeue|cleanup` provides operational recovery.

## Tests

```bash
gofmt -w .
go vet ./...
go test ./...
go test -race ./...
go tool sqlc generate
go tool oapi-codegen -config api/oapi-codegen.yaml api/openapi/v1/openapi.yaml
```

Integration tests requiring containers are documented in `docs/testing.md`.

## Legacy Compatibility

The Vue and UniApp clients can continue to call `/api/auth`, `/api/sys/*`, `/api/chat/*`, `/api/workflow/*` and `/ws/message`. A boundary adapter wraps only those legacy routes in `{code,message,data}`. New clients should use `/api/v1`.

The primary Vue pages were exercised against the Go API, including dynamic menus, CRUD lists, files, chat, notices, monitoring, workflow and codegen. See [`docs/api/frontend-compatibility.md`](docs/api/frontend-compatibility.md).

## Security

The legacy SQL dump contains secrets and personal data. Those values were not copied. Rotate every credential found in the dump before using any migrated data. See `docs/security/security-report.md`.
