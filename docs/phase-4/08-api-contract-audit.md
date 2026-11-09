# Phase 4 API Contract Audit

## Current Contract

- OpenAPI 3.1 document: `api/openapi/v1/openapi.yaml`.
- Canonical route permission test: 137 operations.
- Public operations: 12.
- Authenticated-only operations: 12.
- Business-permission operations: 113.
- Generated Go types and frontend TypeScript schema are checked in.

## Verification

```text
$env:CGO_ENABLED='0'; go tool oapi-codegen -config api/oapi-codegen.yaml api/openapi/v1/openapi.yaml
go test ./api
go test ./internal/platform/httpserver -run TestCanonicalRoutePermissionsMatchOpenAPI -v
pnpm check:api-types
```

The permission route test reported:

```text
route permissions checked: total=137 public=12 authOnly=12 business=113
```

Payment creation now has a typed `PaymentRequest` body, and the callback response includes the
merchant `orderId` plus `duplicate` classification. Provider callback payloads are parsed into a
normalized ledger event rather than being accepted as a generic `{status}` success.

## Compatibility Boundary

The historical `/api/*` compatibility routes and envelope were removed after this audit. The
online runtime now exposes `/api/v1/*` only, with canonical JSON, RFC 9457 problems and
`{items,total,page,pageSize}` pagination. Integration tests assert representative old paths return
`404`.
