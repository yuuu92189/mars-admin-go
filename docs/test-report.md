# Test And Verification Report

Execution date: 2026-11-08 through 2026-11-09

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
| Docker build | multi-stage build, Alpine runtime fallback because GCR is blocked here; 1.82 MB backend context | pass |
| Docker Compose | PostgreSQL, Redis, migration, API, Worker | API healthy, Worker running |
| Helm chart | `helm lint` and `helm template` via Docker | pass |
| GitHub workflow syntax | `actionlint` via Docker | pass |
| Migrations | Goose through version 26 | pass |
| Payment ledger and replay | PostgreSQL integration: amount binding, event dedupe, rejected mismatch | pass |
| Scheduler concurrency | PostgreSQL advisory lock same-key exclusion and release | pass |
| Authentication boundary | Raw Authorization and WebSocket query token rejection tests | pass |
| Online compatibility removal | `go test -tags=integration ./test/integration -count=1` with canonical flows and old-route `404` assertions | pass |
| Cutover rehearsal | isolated PostgreSQL restore and row-count verification | pass |
| Readiness | `/health/ready` | 200 |
| Metrics/docs | `/metrics`, `/openapi.yaml`, `/docs` | served |
| Full legacy import | MySQL 8.0.36 dump to isolated PostgreSQL 18.6 | 397 users, 0 failed rows |
| Import idempotency | second full import run | 0 inserted, 0 failed, 2,605 skipped |
| Historical Vue reference | Pre-removal visual/reference exercise only | not online acceptance |
| Phase 4 frontend checks | `pnpm format:check`, `pnpm lint`, `pnpm typecheck`, `pnpm test`, `pnpm check:api-types`, `pnpm build` | pass |
| P1 compatibility regressions | registration key, realtime conversation isolation, WebSocket proxy and profile redirect | pass |
| Full concurrent Playwright | 4 workers, current production build and real Go API | 37 passed, 19 conditional skips, 0 failed |

## Runtime Scenarios Observed

- Bootstrap administrator created idempotently with Argon2id password.
- Login returned opaque access and refresh tokens; POST refresh rotated the credential.
- Reusing the pre-rotation refresh token returned `401`.
- Bearer-authenticated `/api/v1/auth/me` returned user, role, permission and menu data.
- Canonical social login through `/api/v1/auth/login` returned a direct `token` and `user` payload.
- Canonical `/api/v1/auth/info` returned menu objects with boolean `visible`.
- User creation and self-scope list query worked against PostgreSQL.
- `/api/v1/admin/users` returned canonical `items/total/page/pageSize` pagination.
- Department creation produced an `ltree` path.
- Image and slider captcha challenges were generated; image challenge length was non-empty.
- TOTP MFA setup, enable, required-code response (`428`), MFA login and disable completed successfully.
- File upload recorded SHA-256 metadata and download returned the same byte count.
- Direct message idempotency rejected the second use of a `clientMessageId` with `409`.
- WebSocket connection returned `connected`; ping returned `pong` using Bearer authentication.
- Notice publication created recipient rows and appeared in the current user's inbox.
- Workflow definition publication, instance start and approval completed with `approved`.
- Scheduler job execution persisted a River job and completed with `succeeded`.
- Audit middleware recorded a role creation as `post/admin/success`.
- Canonical `/api/v1/messages` created a direct message with `messageType=text`.
- Codegen imported `access_menus` into PostgreSQL, displayed it after reload and previewed four generated files.
- Representative removed routes including `/api/auth/info`, `/api/sys/user/page`,
  `/api/dashboard/stats`, `/api/sys/chat/users`, `/api/monitor/server/info`,
  `/api/workflow/definition/list` and `/api/tool/gen/page` returned `404`.
- Online sessions, bounded Redis cache inspection and API access statistics returned live data.
- The canonical menu tree rendered in the React shell; the Vue tree remains a historical design
  reference rather than an online client.

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
- canonical social login and current-user/menu responses
- canonical users, dashboard and direct-message flows
- codegen import, persisted list and preview
- explicit `404` assertions for representative removed `/api/*` routes
- payment order/callback idempotency and amount/currency mismatch
- scheduler same-key advisory lock exclusion
- raw Authorization and WebSocket query-token rejection

The full legacy dump validation is documented in `docs/database/import-validation.md`; machine-readable evidence is in `docs/migration/full-import-report.json` and `docs/migration/full-import-idempotency-report.json`.

## Environment Limits

- The host lacks a Windows C compiler, so Race Detector was executed inside the official Go Linux image.
- `gcr.io` is unreachable from this host. The Dockerfile default remains the required distroless image; the local Compose runtime fallback is `alpine:3.22`.
- Backend CI, Frontend CI and Frontend E2E passed on the remote commits recorded in the status
  and final reports.
- Real social, SMS, payment, push, mail and object-storage credentials were unavailable. Their interfaces return explicit not-configured errors instead of fake success.
