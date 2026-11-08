# 11 Phase 2 Acceptance Report

## Acceptance Matrix

| ID | Result | Evidence |
| --- | --- | --- |
| P2-01 Next.js | PASS | `pnpm build` |
| P2-02 TypeScript strict | PASS | `pnpm typecheck` |
| P2-03 dependency governance | PASS | no redundant UI/state/data framework |
| P2-04 OpenAPI reuse | PASS | `@mars/api-types` -> `frontend-api/src/schema.ts` |
| P2-05 design tokens | PASS | copied Vue token layer and frontend docs |
| P2-06 Pearl Glass | PASS | materials CSS + light/dark screenshots |
| P2-07 themes | PASS | light/dark/system provider and E2E theme check |
| P2-08 core UI components | PASS | typed components and 9 Vitest/RTL files |
| P2-09 layout | PASS | AppShell/Sidebar/Header |
| P2-10 Sidebar | PASS | grouped API menus, desktop collapse E2E and mobile drawer/scrim E2E |
| P2-11 Header | PASS | breadcrumb, command palette shortcut/Escape, theme and user menu |
| P2-12 API client | PASS | typed client, RFC 9457, cookies, CSRF |
| P2-13 authentication | PASS | real current-source login and session cookies |
| P2-14 permissions | PASS with condition | frontend helpers + backend authority; superadmin flag is not exposed |
| P2-15 animation | PASS for foundation | GSAP lifecycle, CSS motion and reduced motion |
| P2-16 responsive | PASS for Phase 2 pages | login at 1440/1280/1024/768/390/360; dashboard at 1440/1024/768/390 plus desktop/mobile |
| P2-17 tests | PASS for Phase 2 scope | 19 Vitest/RTL tests; Playwright 25 passed / 11 intentionally skipped |
| P2-18 visual regression | PASS with scope | automated baselines plus direct Vue/React login comparison and geometry |
| P2-19 CI | NOT RUN | workflow definitions exist, but remote GitHub Actions execution is pending |
| P2-20 documentation | PASS | this directory |

## Conditions

1. Business routes remain Phase 3 placeholders; they are not migrated pages.
2. Authenticated E2E requires `E2E_ADMIN_USERNAME` and `E2E_ADMIN_PASSWORD`; without local test credentials those tests skip. In the recorded run they executed against the current-source API.
3. The superadmin UI cannot infer `isSuperAdmin` from `/auth/me`; backend 403 remains authoritative. No dedicated non-admin E2E account was available, so visible-menu filtering is unit-tested and real permission enforcement remains backend-only.
4. Current environment must use a current main API image. A stale 8080 image is not contract evidence.
5. Remote GitHub Actions execution is `NOT RUN`; only local workflow definitions and local command results exist.
6. Dashboard visual snapshots are evidence for the foundation view, not all Vue dashboard modules.

## Conclusion

`ACCEPTED WITH CONDITIONS`

Phase 2 frontend foundation can proceed to Phase 3 page-by-page migration under the conditions above.
