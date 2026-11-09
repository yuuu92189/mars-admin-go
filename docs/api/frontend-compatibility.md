# Frontend API Contract

## Current Boundary

- The online runtime exposes canonical `/api/v1/*` routes only.
- Responses are direct JSON resources. Errors use RFC 9457 `application/problem+json` and the
  matching HTTP status.
- Pagination is `items`, `total`, `page`, `pageSize`.
- IDs are UUID strings. React code must not convert them to numbers.
- Authentication uses Bearer tokens or the server-side session cookie. Browser mutations also
  use the CSRF cookie/header pair.
- Legacy `/api/*` routes, Java field aliases and `{code,message,data}` envelopes are not mounted.
  The integration suite asserts representative removed routes return `404`.

## React Surface

The React/Next.js application uses the canonical contract through its same-origin Route Handler.
Implemented slices include login, registration, profile, dashboard, users, roles, menus,
departments, positions, dictionaries, settings, files, notices, direct chat, scheduler and logs,
codegen and the shared application shell.

The current product state remains **Go foundation + partial React prototype**. The authoritative
page-by-page Java gap list is
[`../architecture-audit/14-java-product-traceability.md`](../architecture-audit/14-java-product-traceability.md).

## Removed Compatibility Contract

The following are intentionally unsupported in the online runtime:

- `/api/auth/*`, `/api/app/auth/*`, `/api/web/auth/*`
- `/api/sys/*`, `/api/monitor/*`, `/api/workflow/*`, `/api/tool/gen/*`
- `/api/dashboard/*`, `/api/crypto/config`
- `captchaUuid`, `captcha`, `expiresIn`, `rows`, `records`, `list` and other legacy aliases
- HTTP 200 responses that wrap application errors

## Offline And Reference Boundaries

- `cmd/legacy-import` and `internal/migration/legacy` are offline migration tools. They are not
  mounted in the API or Worker runtime.
- `scaffold-master-main/scaffold-vue` is a business/design reference only. Vue capture mocks under
  `frontend/scripts` are test fixtures, not supported online routes.
- UniApp compatibility against a real WeChat toolchain is not certified and must be rebuilt on
  the canonical contract before being treated as a supported client.

## Contract Verification

- `api/openapi_test.go` validates the OpenAPI 3.1 document and operation IDs.
- Frontend generated API types are checked against the OpenAPI source.
- `test/integration/api_test.go` exercises canonical login, profile/menu, users, dashboard,
  messaging and codegen flows against PostgreSQL and Redis.
- The same integration test asserts representative removed `/api/*` paths return `404`.
