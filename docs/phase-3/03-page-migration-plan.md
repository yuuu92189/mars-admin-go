# Page Migration Plan

## Completed Implementation Batches

| Batch | Scope | Main files |
| --- | --- | --- |
| Foundation | Typed API modules, shared page states, deferred loading, chart primitives | `frontend/src/lib/api/*`, `frontend/src/components/business/*`, `frontend/src/components/charts/*` |
| A | Dashboard and monitoring aggregation | `internal/modules/monitoring/*`, `features/dashboard/*` |
| B | Users | `features/users/*`, user filters in Identity |
| C | Roles and menus | `features/roles/*`, `features/menus/*` |
| D | Departments and positions | `features/departments/*`, `features/positions/*` |
| E | Dictionaries and settings | `features/dictionaries/*`, `features/settings/*` |
| F | Notices | `features/notices/*` |
| G | Scheduler | `features/scheduler/*` |
| H | Audit logs | `features/audit/*` |
| I | Codegen | `features/codegen/*` |

## Remaining Work

1. Add focused Vitest/RTL tests for API adapters, status mapping, permission branches, query serialization, cron validation, and download handling.
2. Add Go-backed Playwright coverage for the migrated pages.
3. Capture Vue/React screenshots at 1440x900, 1024x768, 768x1024, and 390x844.
4. Validate light, dark, and compact density states.
5. Run all frontend, Go, integration, E2E, and visual gates.
6. Update reports 10-13 with actual evidence only.

## Deferred Scope

Vulnerability assets, vulnerabilities, repair work orders, vulnerability statistics, and AI knowledge-base operations remain out of implementation scope until product and domain decisions exist.
