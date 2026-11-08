# Frontend Compatibility

## Canonical And Legacy Boundaries

- Canonical: `/api/v1/*`, direct resource JSON and RFC 9457 `application/problem+json`.
- Legacy: `/api/*`, wrapped by `internal/platform/compat` as `{code,message,data}` with HTTP 200 for legacy application errors.

The wrapper is an HTTP adapter. No legacy response type is stored in domain or application layers.

## Vue Admin

| Legacy path | New implementation |
|---|---|
| `/api/auth/captcha` | same auth service, image/slider challenge |
| `/api/auth/login` | opaque session, access/refresh/CSRF cookies and token response |
| `/api/auth/info` | current user, roles, permissions and menu tree |
| `/api/auth/profile`, `/api/auth/password` | `identity` application service |
| `/api/sys/user/*` | `identity` admin handlers |
| `/api/sys/role/*` | `access` role handlers |
| `/api/sys/dept/*`, `/api/sys/post/*` | `organization` handlers |
| `/api/sys/dict/*`, `/api/sys/config-group/*` | `system` handlers |
| `/api/sys/file/*` | `storage` handlers |
| `/api/sys/file-group/*`, `/api/sys/file-config/*` | file groups and settings-backed storage configuration |
| `/api/sys/chat/*`, `/api/chat/group/*` | `messaging` handlers |
| `/api/sys/notice/*` | `notification` handlers |
| `/api/monitor/operlog`, `/loginlog` | `audit` handlers |
| `/api/monitor/job/*` | `scheduler` handlers |
| `/api/monitor/online/*` | identity session handlers |
| `/api/monitor/cache/*` | bounded Redis cache inspection |
| `/api/monitor/api-access/*` | in-memory HTTP access statistics |
| `/api/workflow/*` | `workflow` handlers |
| `/api/tool/gen/*` | PostgreSQL codegen handlers |
| `/ws/message?token=...` | authenticated WebSocket hub |

## UniApp

The `/api/app/auth`, `/api/sys/chat`, `/api/chat/group`, `/api/sys/notice` and upload aliases are mounted. The app continues to receive the legacy envelope.

## Token Migration

The compatibility adapter accepts the legacy raw `Authorization` value. New browser clients should use the `mars_session` HttpOnly cookie and send `X-CSRF-Token` for unsafe `/api/v1` requests. App clients may keep Bearer tokens.

The WebSocket query-token form is retained only for compatibility. New clients should rely on the session cookie.

## Phase 1 Canonical Contract

- Canonical login uses `captchaId` and `captchaCode`; legacy `captchaUuid` and `captcha` are mapped only in the `/api/*` adapter.
- Canonical login returns `expiresAt` as RFC 3339. The compatibility adapter additionally returns `expiresIn` for the old Vue client.
- Canonical pagination is `items`, `total`, `page`, `pageSize`; legacy adapters may expose `rows`, `list` and `records`.
- Canonical IDs are UUID strings. Frontend code must not convert them to numbers.
- Canonical errors are RFC 9457 `application/problem+json`; legacy `/api/*` may retain the old envelope.
- Protected OpenAPI operations declare `x-required-permission`; authenticated-only operations declare `x-auth-required`.
- `/metrics` requires `audit:read`. Prometheus must use a dedicated read-only monitoring identity.
- The permission mapping is maintained in [`permission-catalog.md`](permission-catalog.md).

## Compatibility Test Status

The existing Vue source was run against the Go API through the Vite proxy. The following pages loaded without unresolved API errors:

- Dashboard and dynamic menu tree
- user, role, menu, dictionary, settings and file administration
- departments, positions and organization member views
- login logs, operation logs, online sessions and API access statistics
- notices, direct chat contact loading and direct message sending
- scheduled jobs, server monitoring and Redis monitoring
- workflow definitions, todo/done tasks and instance monitoring
- code generation table import, persisted table configuration and source preview
- file storage configuration and file group listing

Only narrow frontend compatibility changes were made: UUID unions in legacy types, shortened UUID display, protocol field normalization, current Go/PostgreSQL labels, and fixes for invalid icons/props. The application layout and Naive UI design were not replaced.

UniApp runtime testing against a real WeChat toolchain remains external. Provider-specific social login still requires real credentials and callback configuration.
