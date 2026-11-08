# Phase 1 Implementation Report

## 1. Scope and Outcome

Phase 1 targeted the engineering baseline and API contract needed before React/Next.js page development. The implementation is based on the first-round audit and the current source at the start of this phase.

### Status

| Area | Status |
| --- | --- |
| Metrics security contract | Implemented and tested |
| Canonical login contract | Implemented and tested |
| Legacy login adapter | Implemented and tested |
| UUID/pagination/error baseline | Implemented |
| OpenAPI success response schemas | Completed for all 2xx responses |
| Group management OpenAPI | Completed for existing canonical routes |
| Permission catalog and route/seed checks | Implemented |
| Workflow instance Data Scope | Implemented and integration-tested |
| SQLSTATE error mapping | Implemented and unit-tested |
| Known cross-module sqlc imports | Removed |
| Repository interface governance | Access and Workflow split; Identity/Notification remain broad |
| Legacy stubs | Real statistics or explicit not-supported responses |
| TypeScript API generation | Implemented and strict-compiled |
| React/Next.js pages | Not started by design |

Phase 1 can enter Phase 2 for the implemented contract baseline, but the product-scoped items listed in section 9 remain open.

## 2. P1-01 Metrics Security

### Before

`api/openapi/v1/openapi.yaml` declared `/metrics` as `security: []`, while `internal/bootstrap/app.go` protected it with `RequirePermission("audit:read")`.

### After

- `/metrics` inherits the global bearer/cookie security scheme.
- The operation declares `x-required-permission: audit:read`.
- OpenAPI includes 401, 403 and 500 Problem responses.
- The integration test still verifies anonymous access is rejected and an authorized administrator can access metrics.

### Evidence

- `api/openapi/v1/openapi.yaml`
- `internal/bootstrap/app.go`
- `api/openapi_contract_test.go`
- `test/integration/api_test.go`

## 3. P1-02 Login Contract

### Canonical API

The canonical request uses:

- `captchaId`
- `captchaCode`
- `loginType`
- `phone` / `smsCode`
- `wxCode` / `authCode`
- `mfaCode`

The canonical response uses:

- `token`
- `refreshToken`
- `csrfToken`
- `sessionId`
- `expiresAt` as RFC 3339
- `user`
- `access`

The canonical API does not expose `expiresIn` as a second meaning for the absolute expiry.

### Legacy Adapter

The `/api/*` login adapter accepts the old `captcha` and `captchaUuid` aliases, maps them to the canonical fields, and adds `expiresIn` only in the legacy response. The new `/api/v1` route does not accept the old aliases as new contract fields.

### Files

- `internal/modules/identity/domain/identity.go`
- `internal/modules/identity/adapter/http/handler.go`
- `internal/modules/identity/adapter/http/handler_test.go`
- `api/openapi/v1/openapi.yaml`

## 4. P1-03 Permission Baseline

Added [`docs/api/permission-catalog.md`](../api/permission-catalog.md), including:

- Canonical permission codes.
- Legacy Vue-to-Go mapping.
- API/menu/route/button/data-scope separation.
- Migration rules that preserve existing role grants.

Added automated checks:

- Every `RequirePermission` code used by Go handlers exists in `db/migrations/000007_seed_authorization.sql`.
- Every `x-required-permission` in OpenAPI exists in the seed.
- Known cross-module sqlc imports are rejected by a source-boundary test.

No existing role or permission rows were deleted. No permission migration was required because the canonical seed already contained the codes used by the implemented routes.

## 5. P1-04 UUID, Pagination and Errors

### UUID

- Canonical IDs remain UUID strings.
- OpenAPI path IDs use `format: uuid`.
- The new TypeScript schema uses strings for UUID values.

### Pagination

`httpserver.PageResponse` now emits only:

```json
{
  "items": [],
  "total": 0,
  "page": 1,
  "pageSize": 20
}
```

Legacy handlers may continue to build `list`/`records`/`rows` aliases at the compatibility boundary.

### Errors

`httpserver.WriteError` now maps PostgreSQL SQLSTATE values at one boundary:

| SQLSTATE | HTTP | Problem code |
| --- | --- | --- |
| `23505` | 409 | `resource_conflict` |
| `23503` | 409 | `resource_in_use` |
| `23502`, `23514`, `22P02` | 422 | `validation_failed` |
| `40001`, `40P01` | 409 | `transaction_conflict` |
| `57014` | 503 | `request_cancelled` |

Database constraint names, table names and SQL text are not returned to clients. Wrapped PostgreSQL errors are also classified.

### Evidence

- `internal/platform/httpserver/json.go`
- `internal/platform/httpserver/json_test.go`
- `internal/platform/requestctx/actor.go`

## 6. P1-05 OpenAPI and P1-06 Groups

### OpenAPI

Current contract:

- 99 paths.
- 134 operations.
- 110 operation-level required-permission declarations.
- 12 authenticated-only operations using `x-auth-required`.
- All 2xx responses declare content.
- All operations have unique operationId values.
- `/metrics`, login, pagination, files, messages, groups, notices, scheduler, workflow, codegen, monitoring and integrations have typed success schemas.
- Dynamic maps remain only where the domain is genuinely dynamic, such as message metadata, setting values and workflow variables.

### Group Management

The following existing canonical routes now have OpenAPI operations:

- Remove group member.
- Update group member role.
- Mute group member.
- Transfer group owner.

Existing group member add/list, group messages, quit and dissolve operations also have permission metadata and response schemas.

### Evidence

- `api/openapi/v1/openapi.yaml`
- `api/openapi_contract_test.go`
- `internal/modules/messaging/adapter/http/handler.go`

## 7. Go Engineering Governance

### Workflow Data Scope

`workflow:read` is no longer sufficient to read every instance.

The PostgreSQL query now filters by:

- Super/global scope.
- Initiator identity.
- Current or historical task assignee.
- Candidate role.
- Department scope.
- Department-tree scope.
- Custom role-department scope.

Instance detail and event history use the same visibility predicate. The integration test covers department isolation, assignee visibility, candidate-role visibility, outsider denial and global scope.

### SQLSTATE

See section 5.

### Cross-Module sqlc

- Identity access lookup moved into the identity-owned query set.
- Notification notice queries moved into `db/queries/notification` and a notification-owned sqlc package.
- Notification River delivery code now uses the notification sqlc package.
- A source-boundary test rejects the two known forbidden imports.

### Repository Interfaces

- Access application now depends on separate `RoleRepository` and `MenuRepository` use-case interfaces.
- Workflow application now depends on separate `DefinitionRepository` and `RuntimeRepository` use-case interfaces.
- Identity and Notification repositories remain broad and are explicitly retained as Phase 2+ debt; they were not rewritten without a concrete use-case split.

### Legacy Stubs

- Legacy scheduler statistics now aggregate actual run records for a requested job.
- Legacy scheduler log cleanup returns an explicit 501 Problem instead of fake 204.
- Legacy generated-file inspection/deletion returns an explicit 501 Problem.
- Legacy SMS log endpoints return an explicit 501 Problem.
- Legacy file-config delete/master operations return explicit 501 Problems.
- Storage connectivity test already returned typed not-configured and remains explicit.

## 8. TypeScript Contract Baseline

Added `frontend-api/`:

- `openapi-typescript` 7.13.0.
- TypeScript 5.9.3.
- `npm run generate`.
- `npm run check`.
- `src/schema.ts` generated from OpenAPI.

Verification:

- Generated schema has 0 `any` tokens.
- `tsc --noEmit` passes with strict mode and `noUncheckedIndexedAccess`.
- `scripts/check.sh` runs the generation and compile check.

The directory is intentionally not a React application. It is the contract package that the future Next.js app can import or share.

## 9. Database and Product Boundaries

### Database

- No new Goose migration was added.
- Existing migration count remains 23.
- No seed permission rows were deleted.
- No production database was connected or modified.
- Integration tests used ephemeral Testcontainers PostgreSQL/Redis/MySQL.
- SQL query ownership moved, but the physical schema did not change.

### Product Decisions Still Required

1. Whether to implement the vulnerability asset/vulnerability/repair domain.
2. Whether AI assistant is in scope and which provider/knowledge base is used.
3. Whether payment/order is in scope and what settlement/replay rules apply.
4. Frontend repository vs monorepo layout.
5. Legacy Vue/UniApp compatibility window.
6. Production data cutover approval and rollback owner.
7. Kubernetes/Helm requirement.
8. Advanced workflow actions beyond current approval/rejection/termination.

These items are not silently invented or marked complete.

## 10. Verification Results

| Command | Result |
| --- | --- |
| `gofmt -w .` | Passed |
| `go test ./...` | Passed |
| `go vet ./...` | Passed |
| `go test -tags=integration ./... -count=1` | Passed; `test/integration` 40.535s |
| Linux `go test -race ./...` | Passed in `golang:1.27.3-bookworm` |
| `golangci-lint run` | `0 issues.` |
| `govulncheck ./...` | No reachable vulnerabilities; 1 imported package and 8 modules contain non-reachable advisories |
| `go tool sqlc generate` | Passed |
| `go tool oapi-codegen ...` | Passed |
| `npm run check` in `frontend-api` | Passed; generated schema has 0 `any` |
| OpenAPI contract tests | Passed: 99 paths, 134 operations, all 2xx content, permissions present |
| `git diff --check` | Passed after final formatting |

Not executed:

- Live external Provider certification.
- Production database migration/cutover.
- React/Next.js browser or screenshot tests, because React pages are not part of Phase 1.

## 11. File Change Categories

### Go Source

- Identity HTTP/domain/application adapter.
- Workflow domain/application/HTTP/PostgreSQL adapter.
- Access application interface split.
- Scheduler, codegen and system legacy handlers.
- HTTP error and pagination boundary.
- Request context data scope.
- Integrations status authorization.

### Queries and Generated Code

- `db/queries/identity/users.sql`.
- `db/queries/access/access.sql`.
- `db/queries/messaging/messaging.sql`.
- `db/queries/notification/notification.sql`.
- `sqlc.yaml`.
- Generated identity/access/messaging/notification sqlc packages.
- Generated `generated/api/api.gen.go`.

### OpenAPI and TypeScript

- `api/openapi/v1/openapi.yaml`.
- `api/openapi_contract_test.go`.
- `frontend-api/package.json`.
- `frontend-api/package-lock.json`.
- `frontend-api/tsconfig.json`.
- `frontend-api/src/schema.ts`.
- `frontend-api/README.md`.

### Tests

- Identity login adapter tests.
- HTTP pagination/SQLSTATE tests.
- OpenAPI security/schema tests.
- Permission seed/OpenAPI consistency test.
- Cross-module sqlc dependency test.
- Workflow data-scope integration test.

### Documentation

- This report.
- `docs/api/permission-catalog.md`.
- `docs/api/frontend-compatibility.md`.
- Architecture audit updates.

## 12. Next Phase

Phase 2 can start only for the implemented contract baseline. It should:

1. Create the Next.js App Router project.
2. Configure TypeScript strict and Tailwind CSS.
3. Consume `frontend-api/src/schema.ts`.
4. Move the seven CSS layers and design tokens.
5. Build the custom React UI primitives and focus/portal infrastructure.
6. Establish Vitest, Playwright and screenshot regression before page implementation.

Product-scoped modules and production data/provider work remain gated by explicit owner decisions.
