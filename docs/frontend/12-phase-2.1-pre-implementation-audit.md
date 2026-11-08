# 12 Phase 2.1 Pre-Implementation Audit

## Repository Baseline

| Item                       | Value                   |
| -------------------------- | ----------------------- |
| Branch                     | `main`                  |
| HEAD                       | `045b206`               |
| Remote main                | `045b206`               |
| Worktree                   | Clean before Phase 2.1  |
| Phase 2 foundation         | `9e45794`               |
| Remote CI verification     | `4943378`               |
| Remote verification report | `045b206`               |
| Frontend CI                | PASS, run `42457068872` |
| Frontend E2E               | PASS, run `42457202146` |

The current worktree was clean and matched `origin/main` before this audit. The existing Phase 2 implementation and its reports were read before deciding what to change.

## Existing Implementation

- Next.js App Router frontend in `frontend/`.
- React strict TypeScript, Tailwind plus the seven copied Vue CSS layers.
- Login, protected shell, Dashboard, profile, 403, 404 and error pages.
- Cookie-based authentication through the Next same-origin proxy.
- RFC 9457 error parsing, CSRF header injection and 401 event handling.
- Theme/density providers, responsive Sidebar, Header, Command Palette and Appearance Dialog.
- Vitest/RTL, Playwright, visual baselines and GitHub Actions.

## Findings

### P1: API Target Configuration Can Diverge

Current state:

- `frontend/src/app/api/[...path]/route.ts` reads `API_PROXY_TARGET`.
- `frontend/next.config.ts` also reads `API_PROXY_TARGET` for health/OpenAPI rewrites.
- `frontend/src/lib/api/server-auth.ts` reads `API_INTERNAL_BASE_URL`.
- Both variables independently default to `http://localhost:8080`.
- There is no validation that the values match.

Impact:

- A deployment can send browser requests to one Go API and perform Server Component session validation against another.
- Login can succeed while a refresh of `/dashboard` appears unauthenticated.
- A typo can silently fall back to localhost in production.

Required change:

- Introduce one authoritative server-side `API_BASE_URL`.
- Keep temporary compatibility aliases with explicit conflict detection.
- Make proxy, rewrites and Server Auth use the same resolver.
- Reject missing, malformed or conflicting production configuration.
- Keep the internal URL out of client bundles.

### P1: Refresh Cookie Flow Is Not Implemented

Current state:

- Go exposes `POST /api/v1/auth/refresh`.
- The endpoint accepts the HttpOnly `mars_refresh` cookie and returns a rotated `LoginResult`.
- Successful refresh sets new `mars_session`, `mars_refresh` and `mars_csrf` cookies.
- CSRF middleware protects refresh POST requests.
- Current React API Client dispatches `mars:unauthorized` immediately on 401.
- No single-flight refresh, retry-once or refresh-specific failure classification exists.
- Server Auth treats any non-OK `/auth/me` response, including 5xx or network failure, as unauthenticated.

Additional backend issue:

- `mars_refresh` lives until refresh TTL.
- `mars_csrf` currently expires with the shorter access-session TTL.
- Refresh therefore can fail CSRF validation after the access session expires even though the refresh cookie remains valid.

Required change:

- Extend the CSRF cookie lifetime to the refresh-session lifetime without exposing the refresh token.
- Implement `refreshSession()` with single-flight concurrency.
- Retry an original browser request at most once.
- Never recursively refresh a failed refresh request.
- Distinguish 401, 403, 5xx and network failures.
- Disable refresh during explicit logout.
- Add a BFF refresh redirect for Server Component session expiry.

### P1: Server Component Session Expiry Cannot Set Cookies

Current state:

- Dashboard layout calls `/auth/me` with the incoming Cookie header.
- If that request returns 401, the layout redirects to login.
- A Server Component cannot safely write the rotated Set-Cookie response back to the browser.

Required change:

- Keep the same-origin path `/api/v1/auth/refresh`, which is required by the refresh cookie Path attribute.
- Add a Route Handler BFF that can receive the refresh cookie, forward the CSRF header, copy all rotated Set-Cookie headers and redirect back to a validated relative path.
- Avoid token values in URLs or browser storage.

### P1: Vue Appearance Behavior Is Not Restored

Vue behavior:

- Theme values: `light`, `dark`, `auto`.
- Default theme: `light`.
- Density values: `comfy`, `regular`, `compact`.
- Default density: `regular`.
- Sidebar collapsed default: `false`.
- Persistence key: `scaffold.appearance`.
- Rail collapsed state is part of the same appearance object.

Current React behavior:

- Theme values: `light`, `dark`, `system`.
- Default theme: `system`.
- Density values: `comfortable`, `compact`.
- Default density: `comfortable`.
- Rail collapsed state is component-local and is lost on refresh.
- `data-density` values do not match the Vue CSS selectors, so density is effectively a no-op.

Required change:

- Restore the Vue enums, defaults and storage semantics.
- Apply all three density values to the real CSS variables.
- Persist rail collapsed state.
- Use a readable appearance cookie as an SSR mirror to avoid hydration mismatch and first-paint rail/theme flash.
- Keep localStorage as the primary browser preference store.

### P1: Command Palette and Shell Interactions Are Incomplete

Vue behavior:

- Recent pages, page search, real user search, actions and local rule-based “ask system”.
- Arrow Up/Down selection, Enter activation, Escape close, focus trap and focus restoration.
- Appearance, password and logout actions.
- Logout confirmation.
- Password Dialog backed by `POST /api/v1/auth/password`.

Current React behavior:

- Page search and Appearance only.
- No recent pages.
- No active-result keyboard navigation.
- No focus trap or focus restoration.
- No password Dialog.
- No logout confirmation.
- Header directly logs out.
- Theme icon cycles the theme instead of opening Appearance as Vue does.

Required change:

- Restore recent pages with a five-item local-only record.
- Add page and action keyboard navigation.
- Add focus trap/restore and ARIA combobox/listbox semantics.
- Add real password change because the Go endpoint and OpenAPI operation exist.
- Add logout confirmation and prevent duplicate submission.
- Add permission-gated user search using the canonical `user:read` API.
- Add a small local “ask system” mode based on real current user/navigation data; do not fake an external AI service.

### P2: Visual Regression Only Covers Login

Current state:

- Direct Vue/React login screenshots and metrics exist.
- Dashboard screenshots are React-only.
- No Vue Shell/Dashboard screenshots, diff images or same-state metrics exist.

Required change:

- Capture real Vue and React login, shell and Dashboard states.
- Cover expanded/collapsed rail, Light/Dark, density, user menu, Appearance Dialog, Command Palette, desktop and mobile.
- Use deterministic fixture data for Dashboard visual comparison while keeping runtime behavior on the real API.
- Save screenshots, diff images and JSON metrics under `docs/frontend/visual-regression/`.

### P2: Remote E2E Coverage Is Too Narrow

Current state:

- Remote E2E only runs desktop login against a frontend-only server.
- Authenticated, refresh, density, persistence and shell flows run only locally.
- CI does not start PostgreSQL, Redis or the Go API.

Required change:

- Add a Go/PostgreSQL/Redis test service to the remote E2E workflow.
- Use isolated credentials and test databases only.
- Run login, protected route, refresh, logout, Sidebar persistence, density and Command Palette E2E.
- Upload Playwright reports, traces and screenshots only on failure.

## Already Correct and To Preserve

- Current Vue Kintsugi login brand panel, terminal and responsive geometry.
- Canonical `/api/v1` browser paths and same-origin proxy.
- HttpOnly session/refresh cookies and readable CSRF token.
- No token or password in localStorage.
- Dashboard uses real counters and has no fake statistics.
- Unknown authenticated routes return 404.
- Existing Playwright screenshot baselines and current-source API evidence.

## Files Expected To Change

- `frontend/next.config.ts`
- `frontend/.env.example`
- `frontend/src/app/api/[...path]/route.ts`
- `frontend/src/app/api/v1/auth/refresh/route.ts`
- `frontend/src/lib/api/client.ts`
- `frontend/src/lib/api/auth.ts`
- `frontend/src/lib/api/server-auth.ts`
- `frontend/src/components/auth/auth-provider.tsx`
- `frontend/src/components/providers/theme-provider.tsx`
- `frontend/src/components/layout/app-shell.tsx`
- `frontend/src/components/layout/header.tsx`
- `frontend/src/components/layout/command-palette.tsx`
- `frontend/src/components/layout/appearance-dialog.tsx`
- new password/confirm dialogs and appearance/recent-page helpers
- `frontend/src/app/(dashboard)/layout.tsx`
- `frontend/src/middleware.ts` or the Next 16 equivalent
- Go cookie lifetime and its tests
- Playwright, visual comparison scripts and CI workflows
- Phase 2.1 reports and parity matrix

## Risk and Dependency Notes

- Refresh rotation is stateful and replay-protected; tests must not reuse an old refresh token.
- The refresh cookie Path is `/api/v1/auth/refresh`, so the BFF must preserve that path.
- Browser refresh and Server Component refresh need different response handling.
- Server Components cannot set browser cookies directly.
- Appearance SSR state must be readable without exposing sensitive data.
- User search must remain permission-gated and must not expose all users without `user:read`.
- Dashboard visual comparison needs deterministic API fixtures; this is visual-test data only and must not enter runtime paths.
- Adding image diff tooling must remain dev-only and narrowly scoped.

## Test Plan

- API URL resolver: default, explicit, legacy, invalid, conflicting, missing production.
- Refresh Client: valid session, 401 refresh success, retry once, refresh 401, refresh 403, 5xx, network failure, concurrent single-flight, logout suppression.
- Server BFF: cookie forwarding, Set-Cookie copying, safe redirect, refresh failure.
- Appearance: defaults, all enums, localStorage, cookie SSR mirror, rail persistence, density CSS variables, hydration.
- Shell: recent pages, page search, user search permission gate, keyboard navigation, focus trap/restore, password Dialog, logout confirmation.
- Playwright: wrong password, real login, page refresh, access expiry and refresh, logout, rail persistence, density, Command Palette, 390/768/1440/1920.
- Visual: Vue/React login, shell and Dashboard; screenshots, diff images and metrics.
- CI: full quality gate plus Go-backed authenticated E2E and failure artifacts.

## Audit Conclusion

Phase 2.1 is required. The existing foundation is usable, but API configuration drift and missing refresh handling are P1 release risks. Appearance and Shell behavior are materially behind the Vue reference. Implementation should proceed in this order:

1. API URL unification and tests.
2. Refresh cookie/CSRF correction, client single-flight and Server BFF.
3. Appearance defaults, enums and persistence.
4. Command Palette, password and logout interactions.
5. Vue/React Shell and Dashboard visual comparison.
6. E2E and CI hardening.
7. Final reports and acceptance.
