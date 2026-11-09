# Final Engineering Report

Updated: 2026-11-09

## Overall Conclusion

The repository remains a coherent Go modular monolith with a React/Next.js App Router frontend.
The architecture is suitable for continued development: domain boundaries are explicit, SQL is
parameterized, PostgreSQL is the source of truth, Redis is ephemeral coordination, and API
contracts are generated from OpenAPI.

Phase 4 found and fixed security and correctness defects in MFA setup, role/data-scope
assignment, workflow termination visibility, registration policy enforcement, menu hierarchy
cycles, sensitive-setting downgrade handling, group membership transitions, upload handling and
Codegen ZIP generation. No P0 issue remains in the audited surface.

Production deployment is not certified. External provider credentials, production payment
settlement, legacy cutover approval and a live S3-compatible bucket remain external conditions.
The appropriate conclusion is **PASS WITH CONDITIONS**, not an unqualified production PASS.

## Verified Statistics

- Go files: 176.
- Go module directories: 14.
- Canonical OpenAPI operations: 137.
- Goose migrations: 25.
- React App Router pages: 18.
- Migrated Go-supported business pages: 13.
- Go test functions: 91.
- Frontend test files: 20.
- Frontend Vitest tests: 58.

## Completed Remediation

- MFA setup now refuses to replace an already-enabled secret.
- Captcha and self-registration are enforced from the public setting source on the server.
- Role assignment checks requested permissions and data scope against the actor.
- Role mutation and menu hierarchy operations reject privilege and cycle escalation paths.
- Workflow termination requires visibility through the same data-scope predicate used for reads.
- Messaging rejects client-supplied system messages and validates group ownership transitions.
- Payment orders now have a PostgreSQL ledger, callback event deduplication, amount/currency
  binding and replay-safe paid-state transitions.
- Scheduler concurrency keys are enforced with PostgreSQL advisory locks and bounded River
  snoozes.
- Raw legacy Authorization headers and WebSocket query tokens are rejected; Bearer and session
  cookie authentication remain supported.
- Login branding now describes Identity, Access, Workflow and Audit instead of excluded reference
  domains.
- Historical Phase 1/3 and architecture documents now mark excluded security domains as
  EXCLUDED rather than blocked or pending work.
- E2E login navigation assertions are bounded and deterministic under the parallel suite.
- Multipart uploads are bounded and temporary files are removed after parsing.
- ZIP generation streams to the response and rejects unbounded table selections.
- Existing sensitive settings remain encrypted when updated through the settings API.
- Legacy notice retry/send-log routes return explicit not-configured errors instead of fake
  success.
- The backend Docker context now excludes frontend, dependency and test artifacts; the verified
  build context is 1.82 MB.

## Verification

The current evidence is recorded in [`docs/phase-4/11-test-acceptance.md`](phase-4/11-test-acceptance.md):

- `gofmt -l .`: clean.
- `go vet ./...`: PASS.
- `go test ./...`: PASS.
- `go test -tags=integration ./test/integration -count=1`: PASS.
- Linux container `go test -race ./...`: PASS.
- `golangci-lint run`: PASS after remediation.
- `govulncheck ./...`: no reachable vulnerabilities.
- `sqlc generate` and `oapi-codegen`: PASS.
- Frontend format, lint, typecheck, Vitest, API type check and production build: PASS.
- Docker Compose backend image build: PASS.
- Helm lint and template rendering: PASS.
- Backend CI, Frontend CI and Frontend E2E remote runs: PASS for commit
  `d0e23f1417a5884a347b926386c0887082974e7c`.
- Payment ledger, callback replay, scheduler concurrency and legacy token rejection tests: PASS.
- Isolated backup/restore rehearsal: PASS, migration version 25.
- Go-backed Playwright and visual baselines: see the Phase 4 acceptance report for the final
  run evidence.

## Remaining Conditions

- Rotate and live-test all legacy/provider credentials.
- Complete production cutover approval; the isolated backup/restore rehearsal passed locally.
- Certify a real S3/MinIO/OSS bucket and external notification/payment providers.
- Upgrade legacy Vue/UniApp clients to Bearer or Cookie authentication.
- Certify real payment provider settlement with production credentials.

## Final Status

**PASS WITH CONDITIONS**
