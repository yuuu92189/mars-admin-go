# Engineering Status

Updated: 2026-11-09

## Current State

Phase 4 completed a repository-wide audit and remediation pass against the current Go source,
database migrations, OpenAPI contract, React pages and automated test suites.

- Go module directories: 14 (`access`, `audit`, `codegen`, `identity`, `integrations`,
  `messaging`, `monitoring`, `notification`, `organization`, `outbox`, `scheduler`, `storage`,
  `system`, `workflow`).
- Canonical OpenAPI operations: 137 (12 public, 12 authenticated-only, 113 business-permission
  routes).
- Goose migrations: 25, latest `000025_payment_ledger.sql`.
- React App Router pages: 18, including all 13 Go-supported business pages.
- Go test functions: 91 across unit, domain, security, contract and integration tests.
- Frontend tests: 20 files / 58 Vitest tests.
- Full Go-backed Playwright suite: 52 tests with the visual suite included; the conditional
  session-refresh test is skipped unless the test TTL environment is supplied.
- GitHub Actions: frontend CI, frontend E2E and the newly added Backend CI workflow. Backend CI
  remains pending an authorized push for its first remote run.

## Implemented

- Identity users, Argon2id, durable sessions, refresh rotation, replay classification and
  configurable refresh history cleanup.
- Dynamic captcha and registration policy from public system settings, with server-side
  enforcement.
- TOTP MFA setup/enable/disable, including protection against replacing an enabled secret without
  a deliberate disable flow.
- RBAC roles, permissions, menus, data scopes and server-side route authorization. Non-super
  administrators cannot assign roles with unheld permissions or broader data scopes.
- Department tree with `ltree`, positions, memberships and scoped organization operations.
- Dictionaries, encrypted sensitive settings, revisions and audited updates.
- Local and S3-compatible storage with bounded uploads, content checks, private downloads and
  asynchronous object cleanup.
- Direct/group messaging, read receipts, Redis fan-out, WebSocket heartbeat and compatibility
  routes.
- Notices, target resolution, recipient/read state and River delivery jobs.
- Scheduler jobs, run history, cron scanning and transactional River enqueue.
- Scheduler distributed concurrency keys use PostgreSQL advisory locks with bounded River snoozes.
- Versioned workflows with assignee checks, data-scope reads, row-locked decisions and
  visibility-checked termination.
- PostgreSQL introspection, imported table metadata, preview generation and streamed ZIP
  download.
- Payment orders, provider callback ledger, amount/currency binding and replay-safe callback
  processing.
- Prometheus metrics, OpenTelemetry spans, structured `slog` logs, request IDs and compatibility
  envelopes.
- Docker multi-stage build, Compose development stack, Helm templates and legacy HTTP adapter.
- Raw legacy Authorization headers and WebSocket query tokens are rejected; Bearer and session
  cookie authentication remain supported.

## Not Complete Or Not Certifiable Here

- Live Aliyun/Tencent/Qiniu SMS, WeChat Pay, Alipay, social OAuth, SMTP and push provider
  certification. Interfaces and explicit not-configured behavior exist; real credentials were not
  available.
- Live S3/MinIO/OSS bucket certification.
- Production legacy-data cutover. The supplied dump was validated in an isolated PostgreSQL
  database; applying it to production still requires owner approval, secret rotation and a
  cutover plan.
- Payment callbacks are signature-verified and locally ledger-tested, but real provider settlement
  credentials and production certification were unavailable.
- Existing Vue/UniApp clients must migrate to Bearer or Cookie authentication; the backend no
  longer accepts raw Authorization headers or WebSocket query tokens.
- Redis is treated as ephemeral coordination and fan-out, not a business source of truth.

## Production Gate

Do not announce production readiness until provider credentials are rotated and live-tested, the
validated import report and production cutover are approved, existing clients complete the
authentication migration, and the required distroless image is built in an environment that can
access the runtime registry.
