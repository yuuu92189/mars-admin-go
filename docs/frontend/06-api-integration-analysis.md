# 06 API Integration Analysis

## Contract Source

The frontend imports generated types from `frontend-api/src/schema.ts` through the `@mars/api-types` alias. No handwritten backend DTOs are used for auth or dashboard responses.

## Client

`frontend/src/lib/api/client.ts` provides:

- Same-origin base path by default.
- `credentials: "include"`.
- JSON body serialization.
- `X-CSRF-Token` from the readable `mars_csrf` cookie for unsafe methods.
- RFC 9457 Problem Details parsing into `ApiError`.
- 204/205 handling.
- 401 event for the client auth provider.
- AbortSignal support through `RequestInit`.

## Same-Origin Proxy

The browser calls `/api/v1/*` on the Next origin. `frontend/src/app/api/[...path]/route.ts` forwards to the single server-side `API_BASE_URL`, strips the browser `Origin` header, preserves cookies and returns all upstream cookies. This avoids weakening Go CORS and keeps the production same-origin contract.

## Authentication

| Flow         | Endpoint                              | Frontend behavior                                                         |
| ------------ | ------------------------------------- | ------------------------------------------------------------------------- |
| Captcha      | `GET /api/v1/auth/captcha?type=image` | Render the current single backend image in Light and Dark; refresh button |
| Login        | `POST /api/v1/auth/login`             | Send canonical fields; never store token/password                         |
| Session      | `GET /api/v1/auth/me`                 | Server layout validates before rendering shell                            |
| Logout       | `POST /api/v1/auth/logout`            | CSRF header, cookie clearing, redirect                                    |
| Unauthorized | any 401                               | Dispatch `mars:unauthorized`, clear client state, redirect                |

The browser never reads `mars_session` or `mars_refresh`; only the non-HttpOnly CSRF cookie is read for the required header.

## Permissions and Menus

- `hasPermission`, `hasAnyPermission`, `hasAllPermissions` are implemented in `frontend/src/lib/permissions.ts`.
- Frontend permission checks are UX only; the Go API remains authoritative.
- Menu paths from `/auth/me` are relative and normalized by `frontend/src/lib/navigation.ts`.
- Known visible menu paths render the Phase 3 placeholder; unknown authenticated paths render 404.
- `CurrentUser` does not expose `isSuperAdmin`; the UI does not infer superadmin from a username or role string. Dashboard requests are attempted and a backend 403 is displayed as permission denied.

## Dashboard

`GET /api/v1/admin/monitoring/dashboard` is the only Phase 2 dashboard data source. It returns real integer counters:

- users/userCount, roles, menus, permissions
- online sessions, files, messages, published notices
- outbox pending/dead

The current contract does not expose Vue's trend, swimlane or vulnerability statistics. Those areas are not filled with fake data.
