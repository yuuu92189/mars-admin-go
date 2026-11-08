# Phase 3 Readiness Audit

> Historical Phase 3 record. Phase 4 later made the business boundary explicit: Asset
> Management, Vulnerability Management and Remediation Work Orders are excluded from this
> project, not pending or blocked deliverables. References below are retained only as historical
> evidence of the reference-Vue inventory.

## Repository Baseline

| Item | Evidence |
| --- | --- |
| Repository | `https://github.com/yuuu92189/mars-admin-go` |
| Branch at audit start | `main` |
| HEAD at audit start | `834f1e2 docs(frontend): record final e2e verification` |
| Worktree at audit start | Clean and synchronized with `origin/main` |
| GitHub visibility | Public (`private: false` via GitHub REST API) |
| Frontend | Next.js App Router, React 19, TypeScript strict, Tailwind 4 |
| Backend | Go modular monolith, PostgreSQL, Redis, River |
| Contract | `api/openapi/v1/openapi.yaml` |
| Generated types | `frontend-api/src/schema.ts` |

## Vue Page Inventory

| Vue page | Target React route | Go domain | Initial status |
| --- | --- | --- | --- |
| Dashboard | `/dashboard` | Monitoring + Audit | READY |
| Users | `/system/user` | Identity + Access + Organization | READY |
| Roles | `/system/role` | Access | READY |
| Menus | `/system/menu` | Access | READY |
| Departments | `/system/dept` | Organization | READY |
| Positions | `/system/post` | Organization | READY |
| Dictionaries | `/system/dict` | System | READY |
| Settings | `/system/config` | System | READY |
| Notices | `/system/notice` | Notification | READY |
| Scheduled Jobs | `/monitor/job` | Scheduler | READY |
| Operation Logs | `/monitor/operlog` | Audit | READY |
| Login Logs | `/monitor/loginlog` | Audit | READY |
| Codegen | `/tool/generator` | Codegen | READY |
| Assets | Vue-only `/vuln/asset` | None | EXCLUDED |
| Vulnerabilities | Vue-only `/vuln/vuln` | None | EXCLUDED |
| Repair Tasks | Vue-only `/vuln/repair` | None | EXCLUDED |

The Vue reference has 17 business views including the three vulnerability views. The Go repository has complete business APIs for 13 of those views and no complete vulnerability domain.

## API Capability Matrix

| Capability | OpenAPI | Handler | Permission | Repository / SQL | Result |
| --- | --- | --- | --- | --- | --- |
| User list/detail/create/update/delete | Present | Present | `user:read/create/update/delete` | Present | READY |
| Password reset / revoke sessions | Present | Present | `user:update` | Present | READY |
| Department/position filtering | Added in Phase 3 | Present | `user:read` | Present | READY |
| Role CRUD | Present | Present | `role:read/create` | Present | READY |
| Role menus / permissions | Present | Present | `role:read/assign` | Present | READY |
| Menu CRUD | Present | Present | `role:read/assign` | Present | READY |
| Department tree / CRUD | Present | Present | `department:read/manage` | Present | READY |
| Position CRUD | Present | Present | `department:read/manage` | Present | READY |
| Dictionary types/items | Present | Present | `dictionary:read/manage` | Present | READY |
| Settings groups/update | Present | Present | `settings:read/manage` | Present | READY |
| Notices / read / publish / withdraw / delete | Present | Present | `notice:read/manage` | Present | READY |
| Scheduler jobs / runs | Present | Present | `scheduler:read/manage` | Present | READY |
| Audit events / login events | Present | Present | `audit:read` | Present | READY |
| Codegen import / preview / sync / ZIP | Present | Present | `codegen:read/manage` | Present | READY |
| Vulnerability assets/vulns/repairs | Not applicable | Not applicable | Not applicable | Not applicable | EXCLUDED |
| Vulnerability AI endpoints | Not applicable | Not applicable | Not applicable | Not applicable | EXCLUDED |

## Contract Gaps Corrected

The initial OpenAPI document did not describe several real request bodies or query parameters. Phase 3 adds:

- User `departmentId` / `positionId` query filters.
- Role list pagination and keyword filters.
- Role update and role authorization request bodies.
- Menu and position input request bodies.
- Dictionary item input request bodies.
- Notice input and canonical delete operation.
- Scheduler input and pagination.
- Audit event filters and login pagination.
- Codegen imported-table schemas, import input, preview output, and ZIP download.
- Real Dashboard counters and time-series schema.

## Data Model Differences

- Vue uses numeric IDs; Go uses UUID strings.
- Vue status codes are numeric; Go business statuses are strings such as `active`, `disabled`, `draft`, `published`.
- Vue Quartz fields do not exist in Go. The React scheduler uses River-compatible handlers, payload, retry count, timeout, and concurrency key.
- Vue role data scopes use labels; Go uses `all`, `custom`, `department`, `department_tree`, and `self`.
- Vue codegen describes Java/Vue output; Go returns Go-project files from the real codegen service.

## Risks

1. Audit trend and login heatmap currently aggregate the latest 100 returned events rather than dedicated statistics endpoints. The UI labels this sample scope.
2. The canonical user list now enriches role, department, and position IDs in batch queries. This adds three read queries per user page.
3. Sensitive settings are write-only in the browser. Empty sensitive values preserve the stored secret.
4. The three vulnerability pages are explicitly excluded by the Phase 4 business-scope decision.
5. Visual comparison and full Go-backed E2E remain required before final acceptance.

## Migration Order

1. Dashboard and real monitoring aggregation.
2. Users.
3. Roles and menus.
4. Departments and positions.
5. Dictionaries and settings.
6. Notices.
7. Scheduler.
8. Audit logs.
9. Codegen.
10. E2E, visual regression, CI, and final acceptance.

## Acceptance Criteria

Each migrated page must have:

- A real React page and server-side route visibility check.
- Real Go API calls through the shared client.
- Canonical permission codes.
- Loading, empty, error, and forbidden states.
- No static business data or mock runtime path.
- Unit or integration coverage for its adapters and critical interactions.
- Desktop and mobile visual verification before final acceptance.
