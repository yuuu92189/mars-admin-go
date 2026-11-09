# Phase 4 Issue Register

## Summary

| Severity | Found | Fixed | Remaining |
| --- | ---: | ---: | ---: |
| P0 | 0 | 0 | 0 |
| P1 | 10 | 10 | 0 |
| P2 | 17 | 17 | 0 |
| P3 | 4 | 4 | 0 |

## P1 Issues

| ID | Location | Problem | Fix | Verification |
| --- | --- | --- | --- | --- |
| P1-01 | `identity/application/service.go` | MFA setup could replace an enabled secret from an active session. | Reject setup while MFA is enabled. | Real PostgreSQL MFA regression test. |
| P1-02 | `identity/application/service.go`, `access/application/service.go` | Non-super role assignment could grant unheld permissions or broader data scope. | Permission subset and data-scope checks on assignment and role mutation. | Limited-admin integration test. |
| P1-03 | `workflow/application/service.go` | Terminate used an ID without the instance visibility predicate. | Visibility check before termination. | Outsider termination integration test. |
| P1-04 | `identity/application/service.go`, bootstrap policy | Self-registration ignored the public registration setting. | Server-side dynamic registration policy, fail closed. | Disabled registration API returned `403`. |
| P1-05 | `frontend/src/app/api/[...path]/route.ts` | Non-JSON request bodies were buffered in the Next process. | Forward `request.body` as a stream with `duplex`. | Route Vitest stream regression test. |
| P1-06 | `codegen/adapter/http/handler.go` | Codegen ZIP was assembled in an unbounded memory buffer. | Stream ZIP and cap table selection at 50. | Handler validation test and build. |
| P1-07 | `frontend/src/features/auth/login-brand-panel.tsx` | Login branding presented excluded asset/vulnerability/remediation domains as product capabilities. | Replace copy with Identity, Access, Workflow and Audit stages. | Login visual suite and viewport baselines pass. |
| P1-08 | `integrations/payment_*`, `db/migrations/000025_payment_ledger.sql` | Payment callbacks had no order ledger, amount binding or replay protection. | Persist orders and callbacks, validate amount/currency, deduplicate provider event IDs and transition paid state atomically. | Real PostgreSQL payment ledger integration test. |
| P1-09 | `httpserver/middleware.go`, `messaging/realtime/hub.go` | Legacy raw Authorization and WebSocket query tokens were still accepted. | Require Bearer or session cookie; reject raw/query forms. | Middleware and WebSocket rejection tests. |
| P1-10 | `scheduler/adapter/river/backend.go` | Scheduler concurrency keys were persisted but not enforced. | Acquire session advisory locks and snooze competing River jobs. | PostgreSQL same-key exclusion integration test. |

## P2 Issues

| ID | Location | Problem | Fix |
| --- | --- | --- | --- |
| P2-01 | access/identity/organization SQL | Disabled or deleted roles still contributed menu/custom scope. | Active-role predicates. |
| P2-02 | access menu service | Deep parent cycles could cause recursive tree failure. | Recursive descendant check and guarded tree builders. |
| P2-03 | messaging group repository | Owner transfer and group mutations could silently no-op or target non-members. | Membership checks and affected-row validation. |
| P2-04 | system settings | Existing sensitive settings could lose sensitivity on update. | Preserve existing sensitivity and encrypt. |
| P2-05 | storage upload | Multipart temporary files were not removed; Office ZIP types were rejected. | `RemoveAll`, bounded parse and ZIP-based MIME compatibility. |
| P2-06 | login screen | Protocol-relative post-login redirect was accepted. | Safe destination validation. |
| P2-07 | notification compatibility | Retry/send-log routes returned fake success/empty data. | Online compatibility routes removed; canonical channel logs/retry remain product gaps. |
| P2-08 | workflow compatibility | Definition compatibility lists silently truncated at 20 rows. | Online compatibility routes removed; canonical workflow list APIs remain. |
| P2-09 | visual E2E | Screenshot ran before the settings-ready button state. | Wait for enabled submit button. |
| P2-10 | realtime hub | Subscription context was not cancelled on application close. | Store/cancel subscription context. |
| P2-11 | current documentation | OpenAPI operation counts and migration/status statements were stale. | Phase 4 docs and current status/report updates. |
| P2-12 | `.dockerignore` | Backend image builds transferred frontend, dependency and test artifacts (600+ MB). | Exclude non-runtime paths; verified 1.82 MB context and successful image builds. |
| P2-13 | E2E login helpers | Parallel suite navigation could exceed the default 5-second assertion despite a successful login; brand-stage names also collided with the submit-button locator. | Use bounded 15-second navigation waits and exact submit-button names. |
| P2-14 | visual E2E baselines | Viewport screenshots included the Next.js development indicator. | Hide the development portal before every baseline screenshot and regenerate the baselines. |
| P2-15 | GitHub Actions | Only frontend workflows existed; full Go tests, generation drift and Race lacked a dedicated remote gate. | Add Backend CI with quality, integration and race jobs; validate workflows with actionlint. |
| P2-16 | historical Phase 1/3 and architecture docs | Excluded security domains were still described as blocked, missing, or Phase 6 work. | Add historical scope banners and mark the domains EXCLUDED throughout current documentation. |
| P2-17 | `scripts/cutover-rehearsal.sh`, migration runbook | Backup/restore and cutover had no isolated rehearsal evidence. | Add checksum-verified restore rehearsal and record migration version/row counts; local PostgreSQL report passes. |

## P3 Issues

- Tracing spans used `unmatched` before chi routing; span names are now updated after routing.
- `/docs` was blocked by the default CSP; docs-specific CSP now permits the Scalar CDN.
- Codegen multi-row deletion was non-transactional; it now uses one transaction.
- Application startup now cleans up telemetry/Redis/hub resources on post-setup errors.

## Conditions Not Counted As Fixed Defects

- Live provider credentials and real production payment settlement certification.
- Production legacy-data cutover approval and production execution.
- Existing legacy clients that have not moved to the canonical `/api/v1` contract and supported
  authentication.
- Historical-statistics APIs for compatibility charts.
