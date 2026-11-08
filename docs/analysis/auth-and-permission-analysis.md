# Authentication And Permission Analysis

> Historical reference analysis. Phase 4 removed raw Authorization and WebSocket query-token
> acceptance; current clients must use Bearer or session cookies.

## Legacy Flow

1. Frontend sends `Authorization` containing a Sa-Token value.
2. Password login uses BCrypt, an optional captcha and Redis retry counters.
3. Sa-Token stores login IDs and session-bound role/permission caches.
4. API checks use `@SaCheckPermission` annotations.
5. User list data scope is rewritten by `DataPermissionInterceptor`.
6. Admin role bypasses menu filtering and receives all enabled menus.

## Observed Weaknesses

- Redis is the account lock source, not a durable authentication policy.
- Token is passed in WebSocket query strings and stored in browser persistent state.
- SQL data-scope failure paths can return an empty condition and fail open.
- Permission caching invalidation is ad hoc and tied to SaSession.
- Login logs are not part of the login transaction.

## New Model

- Passwords use Argon2id with per-hash salt and explicit parameter limits.
- Browser auth uses an opaque token in an HttpOnly, SameSite cookie. App clients can use a bearer token.
- Only SHA-256 token hashes are stored. Raw tokens never leave the issuing response.
- Every login creates a durable session row. Refresh rotates both access and refresh hashes atomically.
- Permissions are loaded with the session and checked server-side by route middleware.
- Super-admin bypass is an explicit database property (`access_roles.is_super_admin`), not a username convention.
- Data scopes are converted into query predicates and parameters. There is no SQL string concatenation from user input.
- Login failures update user counters in PostgreSQL and write an audit event. Redis is used for rate limiting and short-lived checks only.
