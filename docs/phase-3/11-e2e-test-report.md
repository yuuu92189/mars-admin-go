# E2E Test Report

Status: PASS

## Full Go-backed Playwright Run

Command:

```powershell
cd frontend
$env:E2E_ADMIN_USERNAME="admin"
$env:E2E_ADMIN_PASSWORD="<local-test-password>"
$env:E2E_SESSION_TTL_SECONDS="45"
pnpm exec playwright test --reporter=list
```

Result:

- 35 passed
- 17 skipped by project or capability guards
- 0 failed
- Total runtime: 1.2 minutes

The 17 skips are conditional coverage guards, not failures or test-time bypasses:

- Desktop-only interactions are skipped in the mobile project.
- Mobile-only interactions are skipped in the desktop project.
- The short-TTL refresh test runs once in the desktop project.
- The password lifecycle and 404 checks run once.
- The Phase 3 page matrix runs once to avoid duplicate live-data mutation.

## Covered Flows

- Login page rendering, theme restoration, protected redirect, forbidden page, and API error.
- Admin login, Dashboard shell, real counters, theme/density persistence, logout, and protected-route redirect.
- Desktop rail collapse, command palette keyboard flow, user search, and password dialog validation.
- Mobile navigation scrim.
- Dashboard 403 and service-unavailable states.
- Dashboard overflow checks at 1920, 1440, 1024, 768, and 390 widths.
- All 13 Phase 3 pages with real Go API content.
- Users, roles, dictionaries, and scheduler primary interactions.
- Phase 3 overflow checks at 1440, 1024, 768, and 390 widths.
- Temporary-user creation, password change, old-password rejection, new-password login, and cleanup.
- Expired access session refresh through the Next BFF.
- Authenticated 404 state.
- Login visual snapshots in desktop and mobile projects.

## Supporting Test Results

| Gate | Result |
| --- | --- |
| `go test ./...` | PASS |
| `go test -tags=integration ./test/integration` | PASS (44.62s) |
| `pnpm test` | PASS (19 files, 57 tests) |
| `pnpm check:api-types` | PASS |
| `pnpm build` | PASS |
| `pnpm format:check` | PASS |
| `pnpm lint` | PASS |
| `pnpm typecheck` | PASS |

## Test Isolation Fixes

- Login rate limits are configurable and now apply consistently to username, IP, and device keys.
- CI uses `AUTH_LOGIN_RATE_LIMIT=100` and `AUTH_LOGIN_MAX_ATTEMPTS=100` for the Go-backed E2E environment.
- The wrong-password test uses a unique unknown username instead of locking the shared administrator.
- Login navigation is no longer blocked by the decorative success animation.
- Session refresh integration uses a 45-second access TTL to avoid unrelated mid-test expiry while still proving refresh behavior.
- Long page matrices and password lifecycle tests have bounded, explicit timeout budgets.

## Remaining CI Evidence

The local full-suite evidence is complete. Remote GitHub Actions evidence is added after the final push and is recorded in `13-phase-3-final-acceptance.md`.
