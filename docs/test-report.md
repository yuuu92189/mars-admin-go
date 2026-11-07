# Test And Verification Report

Execution date: 2026-11-07 through 2026-11-08

## Results

| Check | Command / environment | Result |
|---|---|---|
| Formatting | `gofmt -w .` | pass |
| Unit/integration package compilation | `go test ./...` | pass |
| Vet | `go vet ./...` | pass |
| Static analysis | `golangci-lint run` with v2.14.0 | 0 issues |
| Vulnerability symbols | `govulncheck ./...` | no reachable vulnerabilities |
| Race detector | Linux `golang:1.27.3-bookworm`, `go test -race ./...`, mounted module cache | pass |
| Fuzz | `FuzzValidateDefinition`, 3 seconds, 12 workers | 296,551 executions, pass |
| Benchmark | Argon2id verify, `BenchmarkVerifyPassword-12` | ~121.8 ms/op on host |
| OpenAPI validation | `api/openapi_test.go` | valid 3.1 document, unique operation IDs |
| sqlc generation | `go tool sqlc generate` v1.30.0 | pass |
| oapi-codegen | `go tool oapi-codegen` v2.8.0 | pass |
| Testcontainers integration | PostgreSQL 18.6 + Redis 8.10.1 | pass |
| Docker build | multi-stage build, Alpine runtime fallback because GCR is blocked here | pass |
| Docker Compose | PostgreSQL, Redis, migration, API, Worker | API healthy, Worker running |
| Migrations | Goose through version 15 | pass |
| Readiness | `/health/ready` | 200 |
| Metrics/docs | `/metrics`, `/openapi.yaml`, `/docs` | served |
| Full legacy import | MySQL 8.0.36 dump to isolated PostgreSQL 18.6 | 397 users, 0 failed rows |
| Import idempotency | second full import run | 0 inserted, 0 failed, 2,605 skipped |
| Vue build | `npm run build` | pass |
| Vue compatibility | Vite app logged into Go API, primary pages exercised | pass |

## Runtime Scenarios Observed

- Bootstrap administrator created idempotently with Argon2id password.
- Login returned opaque access and refresh tokens; POST refresh rotated the credential.
- Reusing the pre-rotation refresh token returned `401`.
- Bearer-authenticated `/api/v1/auth/me` returned user, role, permission and menu data.
- Legacy `/api/auth/login` returned `{code:200,message,data}`.
- User creation and self-scope list query worked against PostgreSQL.
- Department creation produced an `ltree` path.
- Image and slider captcha challenges were generated; image challenge length was non-empty.
- TOTP MFA setup, enable, required-code response (`428`), MFA login and disable completed successfully.
- File upload recorded SHA-256 metadata and download returned the same byte count.
- Direct message idempotency rejected the second use of a `clientMessageId` with `409`.
- WebSocket connection returned `connected`; ping returned `pong`.
- Notice publication created recipient rows and appeared in the current user's inbox.
- Workflow definition publication, instance start and approval completed with `approved`.
- Scheduler job execution persisted a River job and completed with `succeeded`.
- Audit middleware recorded a role creation as `post/admin/success`.
- Legacy direct chat loaded copied contacts, selected a contact and sent a `msgType=1` message through the compatibility route.
- Codegen imported `access_menus` into PostgreSQL, displayed it after reload and previewed four generated files.
- Online sessions, bounded Redis cache inspection and API access statistics returned live data.
- The full Vue menu tree rendered from the Go API and primary pages loaded with the existing Naive UI layout.

## Integration Test Coverage

`test/integration/api_test.go` uses Testcontainers with the fixed PostgreSQL and Redis versions and covers:

- full Goose migration from an empty database
- bootstrap admin and password login
- refresh rotation and refresh replay rejection
- WebSocket authentication and heartbeat
- protected profile request
- user creation
- direct-message idempotency
- local file upload
- audit event creation
- legacy menu/user contract adaptation
- legacy numeric unread counts
- legacy direct-message send with `msgType`
- codegen import, persisted list and preview

The full legacy dump validation is documented in `docs/database/import-validation.md`; machine-readable evidence is in `docs/migration/full-import-report.json` and `docs/migration/full-import-idempotency-report.json`.

## Environment Limits

- The host lacks a Windows C compiler, so Race Detector was executed inside the official Go Linux image.
- `gcr.io` is unreachable from this host. The Dockerfile default remains the required distroless image; the local Compose runtime fallback is `alpine:3.22`.
- Real social, SMS, payment, push, mail and object-storage credentials were unavailable. Their interfaces return explicit not-configured errors instead of fake success.
