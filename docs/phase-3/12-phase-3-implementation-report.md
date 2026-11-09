# Phase 3 Implementation Report

> Historical Phase 3 record. The 13 routes below were a foundation/prototype slice, not completion
> of the 35-page Java product. Domains marked EXCLUDED are not pending deliverables.

## Implementation

- Implemented 13 prototype business pages in React App Router routes.
- Added typed API modules for Identity, Access, Organization, System, Notifications, Scheduler, Audit, Monitoring, and Codegen.
- Added server-side page access guards with menu visibility and permission checks.
- Added shared business UI primitives for page headers, filters, tables, bulk actions, drawers, dialogs, empty states, and status feedback.
- Added real Dashboard aggregations for login trend, previous-period login count, operation trend, success rate, role distribution, recent logins, and recent activity.
- Added user department and position filtering plus batched relation enrichment.
- Added canonical notice deletion and codegen ZIP download routes.
- Aligned seeded menu paths with the migrated routes through `000024_phase3_business_routes.sql`.
- Updated OpenAPI and regenerated `frontend-api/src/schema.ts`.
- Added configurable login rate limiting across username, IP, and device policies.
- Updated login UI to load public settings before enabling submit and to hide captcha when disabled.
- Added deterministic Phase 3 Vue/React visual capture and comparison scripts.

## Verification

| Gate | Result |
| --- | --- |
| `pnpm format:check` | PASS |
| `pnpm lint` | PASS |
| `pnpm typecheck` | PASS |
| `pnpm test` | PASS: 19 files, 57 tests |
| `pnpm check:api-types` | PASS |
| `pnpm build` | PASS |
| `go test ./...` | PASS |
| `go test -tags=integration ./test/integration` | PASS: 44.62s |
| Full Go-backed Playwright | PASS: 35 passed, 17 conditional skips, 0 failed |
| Vue visual capture | PASS: 78 screenshots |
| React visual capture | PASS: 78 screenshots |
| Visual diffs | PASS: 78 diffs, average ratio 0.0444, no console/page errors |

## Known Conditions

- Asset Management, Vulnerability Management and Remediation Work Orders are explicitly excluded
  by the Phase 4 business-scope decision. No canonical domain, schema, API, permission or page is
  planned for them.
- Visual comparisons use deterministic Vue fixtures against live React Go data. Content-level differences are expected and documented; the React side does not use fixtures in its runtime path.
- The historical audit charts currently aggregate the latest 100 returned events where the backend does not expose a dedicated historical statistics endpoint.

## Scope Discipline

No vulnerability page, scanner integration, AI service, fake API, or mock runtime data path was
added. Excluded domains are documented rather than fabricated.
