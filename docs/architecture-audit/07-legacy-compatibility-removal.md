# 07 Online Compatibility Removal

## Status

Completed on 2026-11-09.

The online compatibility layer has been removed:

- `internal/platform/compat` and its envelope tests are deleted.
- `/api/*` route mounts are removed from `internal/bootstrap/app.go`.
- `/api/crypto/config` and `/api/dashboard/*` are removed.
- `LegacyRoutes`, `legacy*` handlers, Java payload aliases and Java-shaped response builders are
  removed from the module HTTP adapters.
- Legacy login aliases are removed; login accepts only canonical `LoginInput` fields.
- Notice delete and codegen ZIP download now use canonical handler names.

Integration coverage verifies that representative removed paths return `404`:

- `/api/crypto/config`
- `/api/auth/info`
- `/api/sys/user/page`
- `/api/dashboard/stats`
- `/api/sys/chat/users`
- `/api/monitor/server/info`
- `/api/workflow/definition/list`
- `/api/tool/gen/page`

## Current API Contract

The runtime exposes `/api/v1/*` only:

- direct resource JSON
- RFC 9457 Problem errors with correct HTTP status
- `items/total/page/pageSize` pagination
- UUID string identifiers
- Bearer or session-cookie authentication
- CSRF enforcement for cookie-authenticated mutations

React and any rebuilt mobile client must target this contract. The old Vue/UniApp HTTP callers are
not supported by the online backend.

## Preserved Offline Boundary

The following remain intentionally:

- `cmd/legacy-import`
- `internal/migration/legacy`
- old-ID mapping and import evidence tables
- MySQL dump reading and bcrypt-to-Argon2id migration logic

These tools run only as explicit offline commands. They are not mounted into the API or Worker
runtime and must not be reused as an online compatibility adapter.

The Vue capture mocks under `frontend/scripts` are visual-regression fixtures and do not define a
supported runtime API.

## Verification

- `go test ./...`: PASS.
- `go test -tags=integration ./test/integration -count=1`: PASS.
- The integration suite exercises canonical social login, current-user/menu, users, dashboard,
  messaging and codegen behavior before checking removed routes return `404`.
- `rg` checks confirm no `platform/compat`, `LegacyRoutes` or `legacy*` handler declarations remain
  in the module HTTP adapters.

## Follow-Up

Compatibility removal does not complete the Java product migration. Remaining page and workflow
gaps stay tracked in
[`14-java-product-traceability.md`](14-java-product-traceability.md), and each future slice must
ship with Go domain/API work, OpenAPI, React, permissions, error/empty states and tests.
