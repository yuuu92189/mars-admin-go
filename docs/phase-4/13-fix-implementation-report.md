# Phase 4 Fix Implementation Report

## Backend

- `internal/modules/identity`: dynamic captcha/registration policies, MFA setup guard, role
  permission/data-scope assignment checks, disabled-role custom-scope SQL.
- `internal/modules/access`: role mutation scope rules, menu cycle detection, safe tree building,
  active-role menu filtering.
- `internal/modules/workflow`: visibility-aware termination and complete compatibility
  definition pagination.
- `internal/modules/messaging`: client system-message rejection and validated group ownership
  transitions.
- `internal/modules/storage`: multipart cleanup, bounded uploads and ZIP-based Office MIME
  compatibility.
- `internal/modules/codegen`: streamed/capped ZIP download and transactional imported-table
  deletion.
- `internal/modules/system`: sensitive-setting preservation and dynamic public-setting policy.
- `internal/modules/notification`: explicit compatibility not-configured responses.
- `internal/modules/organization`: guarded department tree construction.
- `internal/bootstrap`: captcha/registration policy wiring and startup cleanup.
- `internal/platform/httpserver`: docs CSP and routed trace names.
- `internal/modules/messaging/realtime`: subscription cancellation on close.
- `.dockerignore`: removed frontend, dependency, build-output and test artifacts from the backend
  Docker context; current build context is 1.82 MB.
- `internal/modules/integrations`: payment order ledger, provider callback parsing/decryption,
  amount/currency binding and replay-safe callback persistence.
- `internal/modules/scheduler`: PostgreSQL advisory-lock concurrency gate with River snooze.
- `internal/platform/httpserver` and `messaging/realtime`: raw Authorization/query-token removal.
- `scripts/`: checksum-aware backup/restore and isolated cutover rehearsal.

## SQL And Generated Code

Updated query sources:

- `db/queries/access/access.sql`
- `db/queries/identity/users.sql`
- `db/queries/organization/organization.sql`
- `db/queries/system/system.sql`

Regenerated module-local SQLC packages. No historical migration was edited.

## Frontend

- `frontend/src/app/api/[...path]/route.ts`: streamed proxy bodies.
- `frontend/src/app/api/[...path]/route.test.ts`: streaming regression test.
- `frontend/src/features/auth/login-brand-panel.tsx`: replaced excluded-domain branding with
  supported Identity, Access, Workflow and Audit stages.
- `frontend/src/features/auth/login-screen.tsx`: safe post-login redirect.
- `frontend/e2e/visual.spec.ts`: wait for settings-ready button state.
- `frontend/playwright.config.ts`: configurable base URL for isolated verification.

## Tests Added Or Extended

- MFA enabled-secret replacement guard.
- Limited administrator role/permission/data-scope assignment.
- Menu descendant-cycle and disabled-role menu visibility.
- Workflow outsider termination.
- Group owner transfer/dissolve/quit boundaries.
- Sensitive-setting preservation.
- Storage ZIP-based document MIME compatibility.
- Codegen download selection cap.
- BFF request-body streaming.
- Login branding business-scope alignment and visual baseline refresh.
- E2E login navigation waits and exact submit-button selectors for deterministic parallel runs.
- Development overlay removal before all login and dashboard visual baseline captures.
- Backend CI workflow covering Go format, vet, tests, generated-source drift, integration and Race.
- Historical documentation scope banners and EXCLUDED-domain wording corrections.
- Payment ledger/replay, scheduler concurrency, legacy token rejection and restore rehearsal.
- Historical documentation scope banners and EXCLUDED-domain wording corrections.
- Backend Docker context and image-build regression.

## Verification

See [`11-test-acceptance.md`](11-test-acceptance.md) for command-level evidence. The current
source API was also exercised on port 8082 for readiness, public settings, disabled registration
and password login.
