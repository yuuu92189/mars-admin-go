# Engineering Status

Updated: 2026-11-09

## Current State

The current state is **Go foundation + partial React prototype**, not Java product acceptance.
Phase 4 records the completed engineering remediation; the corrected Java product traceability
matrix is in [`architecture-audit/14-java-product-traceability.md`](architecture-audit/14-java-product-traceability.md).

- Go module directories: 14 (`access`, `audit`, `codegen`, `identity`, `integrations`,
  `messaging`, `monitoring`, `notification`, `organization`, `outbox`, `scheduler`, `storage`,
  `system`, `workflow`).
- Canonical OpenAPI operations: 137 (12 public, 12 authenticated-only, 113 business-permission
  routes).
- Goose migrations: 25, latest `000025_payment_ledger.sql`.
- React App Router page files: 22, including the first P0/P1 slices for login, registration,
  profile, dashboard, users/roles/menus/departments/positions/dictionaries/settings, notices,
  direct chat, scheduler/logs, codegen and file upload/download. The remaining Java pages are explicitly
  tracked as missing or partial in the traceability matrix.
- Go test functions: 93 across unit, domain, security, contract and integration tests.
- Frontend tests: 20 files / 60 Vitest tests after the P0 additions.
- Full Go-backed Playwright regression: 36 passed, 18 conditional skips, 0 failed.
- GitHub Actions: the P0/P1 code commit `20c81ea6bfb890bb3581995e79a5524106dea2dd` passed
  Backend CI, Frontend CI and Frontend E2E remotely; the register locator follow-up
  `968f47118afac5bd38f4297343769e1ccce28a9d` passed Frontend CI and Frontend E2E.

## Implemented

- Identity users, Argon2id, durable sessions, refresh rotation, replay classification and
  configurable refresh history cleanup.
- Dynamic captcha and registration policy from public system settings, with server-side
  enforcement, including registration captcha consumption.
- Policy-aware React registration, editable profile, authenticated avatar upload, profile shell
  entry and Java-shaped nested menu flattening.
- TOTP MFA setup/enable/disable, including protection against replacing an enabled secret without
  a deliberate disable flow.
- RBAC roles, permissions, menus, data scopes and server-side route authorization. Non-super
  administrators cannot assign roles with unheld permissions or broader data scopes.
- Department tree with `ltree`, positions, memberships and scoped organization operations.
- Dictionaries, encrypted sensitive settings, revisions and audited updates.
- Local and S3-compatible storage with bounded uploads, content checks, private downloads and
  asynchronous object cleanup.
- React file management slice with upload, pagination, authenticated download and deletion.
- React direct-chat slice with real history, send/read state, same-origin WebSocket reconnect and
  polling fallback; group chat and blocklist UI remain tracked as gaps.
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
