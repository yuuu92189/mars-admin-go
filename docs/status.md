# Engineering Status

## Implemented

- Independent Go module targeting Go 1.27.3.
- Explicit dependency injection and modular monolith boundaries.
- PostgreSQL schema, Goose migrations, sqlc-generated data access and UUIDv7 identifiers.
- Identity users, Argon2id, durable sessions, refresh rotation, revocation, login lock state.
- Image/slider captcha, SMS code flow, encrypted TOTP secret and MFA login.
- Roles, permissions, menus, super-admin policy and permission middleware.
- User, role, menu and current-user APIs.
- Department tree using `ltree`, positions and membership persistence.
- Dictionaries, encrypted settings, revisions, audit events and login events.
- Local storage and S3-compatible storage adapter, upload validation and private download.
- Direct/group chat, message idempotency, read receipts, Redis fan-out and WebSocket compatibility.
- Notices, target resolution, recipients, read state and send logs.
- River-backed scheduler jobs, cron scanning, run status and timeout.
- Versioned workflow definitions, safe condition evaluation, approvals, rejection/termination hooks and event history.
- PostgreSQL introspection and modern Go/OpenAPI preview generation.
- Persisted code-generation table import, editable generation metadata, schema sync and ZIP download.
- Prometheus runtime/HTTP/database metrics, OpenTelemetry spans, JSON `slog` logs and request IDs.
- Expanded OpenAPI 3.1 contract covering canonical auth, access, organization, system, audit, storage, messaging, notification, scheduler, workflow, codegen, monitoring and integration routes.
- Docker multi-stage build, Compose, optional Helm and legacy HTTP envelope adapter.
- Unit, benchmark, fuzz, Testcontainers integration, Race, lint and vulnerability checks.
- Full legacy MySQL dump import completed in an isolated PostgreSQL validation database with zero failed rows and a second idempotent run.
- Page-by-page Vue compatibility run covering all primary administration, monitoring, messaging, workflow and codegen pages.

## Not Complete Or Not Certifiable Here

- Live Aliyun/Tencent/Qiniu SMS, WeChat Pay, Alipay, social OAuth and push provider calls. Interfaces and not-configured behavior exist; real credentials and callback certification do not.
- A live S3/MinIO/OSS integration test. The MinIO-compatible adapter builds but was not exercised against a bucket.
- Production cutover of the legacy data. The full dump has been imported into an isolated validation database; applying it to a production database still requires data-owner approval, secret rotation and downtime/cutover planning.
- UniApp runtime E2E against a real WeChat toolchain.
- SSH server credential management and web terminal were intentionally excluded as unsafe/out of scope.
- JVM/JDBC console and server credential management are intentionally not reproduced. Go runtime and Prometheus replace JVM metrics.
- The compatibility browser UI still stores the access token in Pinia persistence and uses the legacy WebSocket query token. New clients should migrate to HttpOnly cookies and CSRF tokens.

## Production Gate

Do not announce production readiness until third-party credentials are rotated and live-tested, the validated import report is approved for cutover, UniApp runtime acceptance is complete, and the required distroless image is built in a network that can access GCR.
