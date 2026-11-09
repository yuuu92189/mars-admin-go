# API Documentation

The canonical contract is `api/openapi/v1/openapi.yaml`, OpenAPI 3.1. Runtime documentation is available at `/docs`, and the raw contract is served at `/openapi.yaml`.

## Boundaries

- `/api/v1/*` is the canonical API. Successful responses use direct resource JSON; errors use RFC 9457 Problem Details.
- `/api/*` is not mounted. Legacy routes, field aliases and `{code,message,msg,data}` envelopes are removed and covered by `404` integration assertions.
- `/ws/message` is the authenticated WebSocket endpoint.

## Generation

```bash
go tool oapi-codegen -config api/oapi-codegen.yaml api/openapi/v1/openapi.yaml
```

The generated package is `generated/api`. The contract is validated by `api/openapi_test.go`, including unique operation IDs.

## Authentication

- Browser sessions: `mars_session` HttpOnly cookie.
- API clients: opaque Bearer token.
- Refresh: rotating `mars_refresh` cookie or refresh token request.
- Unsafe browser requests: `X-CSRF-Token` from `mars_csrf`.

The legacy raw `Authorization` header and WebSocket query token are rejected. API clients must use
`Authorization: Bearer <token>`; browser and WebSocket clients must use the session cookie.

## Compatibility

See `frontend-compatibility.md` for the canonical React contract, removed routes and offline
import boundary.
