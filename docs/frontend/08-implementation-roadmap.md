# 08 Implementation Roadmap

| Stage | Status | Evidence |
| --- | --- | --- |
| Stage 0 audit | Completed | `docs/frontend/01`-`07` |
| Stage 1 initialization | Completed | `frontend/package.json`, config, lockfile |
| Stage 2 design system | Completed | copied seven CSS layers + React theme providers |
| Stage 3 core components | Completed foundation subset | `frontend/src/components/ui` |
| Stage 4 shell | Completed | AppShell/Sidebar/Header/responsive state |
| Stage 5 API/auth | Completed | API client, proxy, login, session, CSRF, permissions |
| Stage 6 basic pages | Completed | login, dashboard, profile, 403, 404, error |
| Stage 7 tests/visual | Completed for Phase 2 scope | 19 Vitest/RTL tests, 25 passed Playwright tests, automated baselines and direct Vue/React comparison |
| Stage 8 acceptance | Completed | `11-phase-2-acceptance-report.md` |

## Phase 3 Gate

Before migrating a business page:

1. Confirm its product scope and canonical Go API.
2. Map its Vue components and interactions to React.
3. Add real API loading/empty/error/403 states.
4. Add focused component and E2E tests.
5. Compare screenshots at the same viewport/theme/state.

Unimplemented routes intentionally render a Phase 3 placeholder; they must not be treated as migrated business pages.
