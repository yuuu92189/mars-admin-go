# Phase 4 Test Acceptance

Final verification rerun: 2026-11-09.

## Executed Checks

| Check | Command / environment | Result |
| --- | --- | --- |
| Go formatting | `gofmt -l .` | PASS, no files |
| Go vet | `CGO_ENABLED=0 go vet ./...` | PASS |
| Go unit/package tests | `CGO_ENABLED=0 go test ./...` | PASS |
| Real dependency integration | `CGO_ENABLED=0 go test -tags=integration ./... -count=1` | PASS, 40.3s |
| Race detector | Linux `golang:1.27.3-bookworm`, `go test -race ./...` | PASS |
| Static analysis | `golangci-lint run` | PASS after remediation |
| Vulnerability symbols | `govulncheck ./...` | no reachable vulnerabilities |
| SQLC generation | `CGO_ENABLED=0 go tool sqlc generate` | PASS |
| Go OpenAPI generation | `go tool oapi-codegen -config api/oapi-codegen.yaml ...` | PASS |
| Frontend format | `pnpm format:check` | PASS |
| Frontend lint | `pnpm lint` | PASS |
| TypeScript strict | `pnpm typecheck` | PASS |
| Frontend unit/component | `pnpm test` | PASS, 20 files / 58 tests |
| OpenAPI TypeScript check | `pnpm check:api-types` | PASS |
| Next production build | `pnpm build` with explicit `API_BASE_URL` | PASS |
| Docker images | `docker compose build api worker migrate` | PASS, 1.82 MB build context |
| Helm chart | `helm lint` + `helm template` via `alpine/helm:3.18.4` | PASS |
| GitHub workflow syntax | `actionlint` via Docker | PASS |
| Local Markdown links | `README.md` and `docs/**/*.md` relative-link scan | PASS |
| Payment ledger | PostgreSQL integration: amount binding, callback dedupe and mismatch rejection | PASS |
| Scheduler concurrency | PostgreSQL advisory-lock same-key exclusion and release | PASS |
| Legacy token migration | Raw Authorization and WebSocket query-token rejection tests | PASS |
| Cutover rehearsal | isolated PostgreSQL restore, migration 25 and row-count verification | PASS |
| Go-backed Playwright | current source on `127.0.0.1:3000` + current API on `127.0.0.1:8082` | PASS, 34 passed / 18 skipped |
| Login visual baselines | `pnpm exec playwright test e2e/visual.spec.ts` | PASS, 4/4 |

The first full Playwright run during the final verification window hit one 5-second login
navigation timeout in the Phase 3 page matrix while the Linux race build was consuming host
resources. The login state had reached "登录成功，正在进入"; the isolated page-matrix rerun passed
3/3 and the final full rerun passed 34/18 with zero failures. No skip is counted as a PASS.
The authenticated, Phase 3 and session-refresh login helpers now allow 15 seconds for the bounded
post-login navigation; exact button names avoid collisions with the interactive brand-stage
buttons. All visual baseline captures hide the Next.js development portal first.

## Skipped Tests

- Session-refresh E2E is conditional on `E2E_SESSION_TTL_SECONDS`; this local run used `0`.
- Other skips are viewport/project conditions already declared by the Phase 3 E2E suite.
- No skip is counted as a PASS.

## API Smoke Evidence

- `GET /health/ready` returned `{"status":"ready"}`.
- Public settings exposed captcha disabled and registration disabled.
- `POST /api/v1/auth/register` returned `403` with `Self-registration is disabled.`.
- Admin password login returned `200` on the current source API.
- `/docs` returned `200` with a CSP that permits the documented Scalar CDN.

## Environment Limits

- The Windows host lacks a C compiler, so Race Detector was run inside Docker.
- `gcr.io` is unreachable; Compose uses the explicitly documented Alpine fallback.
- Real external provider credentials were unavailable; not-configured behavior is tested instead.
- The new Backend CI workflow has not run remotely because the worktree has not been pushed.

## Historical Visual Evidence

The retained Phase 3 visual matrix contains 78 Vue captures, 78 React captures and 78 diffs,
with an average mismatch of `0.0444`. Vue captures use deterministic fixtures; React captures use
live Go data. Those differences remain documented rather than treated as zero.
