# 10 Phase 2 Implementation Report

## Delivered

- Next.js App Router application in `frontend/`.
- React 19.3.1, Next 16.4.5, TypeScript 5.9.3, Tailwind 4.3.5, GSAP 3.15.1.
- Pinned pnpm lockfile.
- Pearl Glass design system copied from the Vue reference and adapted for React.
- Current Vue Kintsugi brand mark, login copy, four-stage journey and typed terminal ported to React.
- Theme provider: light/dark/system; density provider: comfortable/compact.
- Toast provider, focus-safe Dialog/Drawer, accessible core controls and typed data table.
- Protected shell with API-driven menu normalization, responsive sidebar, header, command palette and appearance dialog.
- Login page with real captcha endpoint, canonical fields and Cookie session flow.
- Same-origin API proxy preserving cookies and CSRF.
- Dashboard using real `/api/v1/admin/monitoring/dashboard` counters only.
- Explicit Phase 3 placeholder for unbuilt business routes.
- Unknown authenticated routes return the real 404 state instead of a placeholder.
- Vitest/RTL tests, Playwright tests, visual screenshots and GitHub Actions workflow.

## Known Environment Note

The local Docker service on 8080 was stale and rejected canonical captcha fields. Verification of the current source used a separate API on 8081 against the local development database and an isolated Redis test database. The local acceptance API used a test-only login-attempt allowance; production defaults were not changed. The committed application default remains `http://localhost:8080`; environments must deploy the current main image or set `API_PROXY_TARGET`.

## Verification Record

| Command | Result |
| --- | --- |
| `pnpm install --frozen-lockfile` | PASS |
| `pnpm typecheck` | PASS |
| `pnpm lint` | PASS |
| `pnpm format:check` | PASS |
| `pnpm test` | PASS, 9 files / 19 tests |
| `pnpm build` | PASS |
| `pnpm check:api-types` | PASS |
| `pnpm test:e2e` with current-source API on `localhost:8081` | PASS, 25 passed / 11 skipped (36 total) |
| `pnpm visual:compare` | PASS, Vue/React at 1440×900 Light and 390×844 Dark |
| Remote GitHub Actions Frontend CI | PASS, run `42457068872` |
| Remote GitHub Actions Frontend E2E | PASS, run `42457202146` |

The visual suite masks the random captcha image because its value is intentionally non-deterministic. The direct comparison script supplies the same deterministic captcha fixture to both applications. This is a screenshot fixture, not a mocked login flow.

The authenticated tests are serial because they share one local administrator account. Project-specific tests intentionally skip the irrelevant desktop/mobile half of the matrix.

## Deliberate Non-Goals

- No bulk Vue business-page migration.
- No fake dashboard trends, users, permissions or business statistics.
- No second OpenAPI generation pipeline.
- No production credentials in source, local storage or screenshots.
