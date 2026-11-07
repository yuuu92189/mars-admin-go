# API Documentation

The canonical contract is `api/openapi/v1/openapi.yaml`, OpenAPI 3.1. Runtime documentation is available at `/docs`, and the raw contract is served at `/openapi.yaml`.

## Boundaries

- `/api/v1/*` is the canonical API. Successful responses use direct resource JSON; errors use RFC 9457 Problem Details.
- `/api/*` is the legacy Vue/UniApp compatibility surface. Responses are wrapped as `{code,message,msg,data}` by the HTTP boundary adapter.
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

The legacy raw `Authorization` header and WebSocket query token remain available only for existing clients.

## Compatibility

See `frontend-compatibility.md` for the old-to-new route mapping and browser validation status.
