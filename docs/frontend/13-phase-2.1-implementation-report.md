# 13 Phase 2.1 Implementation Report

## Objective

Phase 2.1 hardens the existing Next.js foundation without starting Phase 3 business-page
migration. The work focuses on one API target, real Cookie refresh, Vue-aligned appearance and
Shell behavior, evidence-backed visual comparison, and Go-backed quality gates.

## Delivered Changes

### API Target

- Added `frontend/src/lib/config/api-base-url.ts` as the single server-side resolver.
- `API_BASE_URL` is authoritative for the Next proxy, rewrites and Server Component auth.
- Legacy `API_PROXY_TARGET` and `API_INTERNAL_BASE_URL` values are accepted only when they agree
  with the authoritative value.
- Production and CI fail closed when configuration is absent or invalid; there is no localhost
  fallback in those environments.
- Credentials, paths, queries and fragments are rejected in the configured origin.
- Browser traffic remains same-origin through `/api/v1/*`; the internal target never enters the
  client bundle.

### Refresh and Authentication

- Added `POST /api/v1/auth/refresh` proxy handling with all rotated `Set-Cookie` headers preserved.
- Added a GET BFF refresh route for Server Component expiry. It forwards the refresh Cookie and
  CSRF token, then redirects to a validated same-origin path.
- Added `x-mars-pathname` in the Next 16 proxy so the BFF can return to the original page.
- Browser API requests use a single-flight refresh and retry an original request at most once.
- Refresh 401 is terminal and never recursive. 403, 5xx and network failures do not trigger refresh
  or clear authentication.
- Explicit logout disables refresh and duplicate unauthorized events are suppressed.
- The readable `mars_csrf` Cookie now lives for the refresh-session TTL, while the refresh token
  remains HttpOnly.

### Appearance

- Restored Vue values: `light`, `dark`, `auto`; `comfy`, `regular`, `compact`.
- Restored defaults: `light`, `regular`, rail expanded.
- Restored `scaffold.appearance` storage and the `mars_appearance` SSR mirror Cookie.
- Persisted Sidebar collapse state and added SSR attributes for theme, density and rail state.
- Restored all three density options and their real row spacing.
- Kept compatibility with the former `system` and `comfortable` values.

### Shell and Command Palette

- Restored recent-page history, page search, real permission-gated user search, appearance,
  password change, logout and local rule-based system help.
- Added Arrow Up/Down, Enter, Escape, Tab focus trap, focus restoration and combobox/listbox ARIA.
- Restored Header theme entry, user menu keyboard handling, password Dialog and logout confirmation.
- The super-administrator flag from `/auth/me` now drives frontend permission checks; backend
  authorization remains authoritative.
- The local `?` help mode never sends prompts to an external AI service.

### Minimal Go Changes

- Extended the CSRF Cookie lifetime to the refresh TTL.
- Returned field-level validation for an incorrect current password.
- Exposed `isSuperAdmin` in the current-user contract.
- Added the missing OpenAPI request body for `POST /api/v1/auth/password`.
- Added Cookie-lifetime tests and regenerated the TypeScript contract types.

## Verification

| Gate                                 | Result                                                                                                          |
| ------------------------------------ | --------------------------------------------------------------------------------------------------------------- |
| `pnpm install --frozen-lockfile`     | PASS                                                                                                            |
| `pnpm format:check`                  | PASS                                                                                                            |
| `pnpm lint`                          | PASS                                                                                                            |
| `pnpm typecheck`                     | PASS                                                                                                            |
| `pnpm test`                          | PASS, 17 files / 50 tests                                                                                       |
| `pnpm check:api-types`               | PASS                                                                                                            |
| `pnpm build`                         | PASS                                                                                                            |
| Go identity and HTTP tests           | PASS                                                                                                            |
| Critical Playwright suite            | PASS, 28 passed / 14 intentional project skips                                                                  |
| Login Vue/React comparison           | PASS, 8 screenshots, matching geometry                                                                          |
| Shell/Dashboard Vue/React comparison | PASS for evidence capture, 8 paired scenarios                                                                   |
| Real refresh BFF                     | PASS, expired session returned through refresh to Dashboard                                                     |
| Real password lifecycle              | PASS, temporary user created, changed password, old password rejected, new password accepted, test user deleted |
| GitHub Actions Frontend CI           | PASS, run `42466214627`, job `128686232041`                                                                     |
| GitHub Actions Frontend E2E          | PASS, run `42466214592`, job `128686231984`                                                                     |

The Playwright suite covers `390x844`, `768x1024`, `1024x768`, `1440x900`, `1920x1080` and the
existing mobile matrix. Authenticated tests use only local or isolated CI test credentials.

## Visual Evidence

- Login comparison: `frontend/visual-comparison/`
- Shell and Dashboard comparison: `docs/frontend/visual-regression/artifacts/`
- Visual report: `docs/frontend/visual-regression/phase-2.1-report.md`
- Vue/React behavior matrix: `docs/frontend/vue-react-parity-matrix.md`

## Residual Risks and Scope

- Dashboard content is still limited to real canonical counters. Vue trend, activity and security
  modules remain Phase 3 work.
- Shell pixel differences remain where Vue and React intentionally render different real data or
  different fixture menu depth. Geometry, viewport, material, typography and responsive behavior
  are compared directly.
- Remote GitHub Actions evidence is recorded for commit `d64f5ed`.
- No production database, OAuth provider, secret or deployment visibility setting was changed by
  this implementation.
