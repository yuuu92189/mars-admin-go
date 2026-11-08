# Visual Regression Report

Status: PASS WITH DOCUMENTED DATA DIFFERENCES

## Scope

The Phase 3 visual matrix covers all 13 Go-supported business pages:

- Dashboard
- Users
- Roles
- Menus
- Departments
- Positions
- Dictionaries
- Settings
- Notices
- Scheduler
- Operation logs
- Login logs
- Codegen

Each page was captured in both the Vue reference application and the React implementation in six scenarios:

| Scenario | Viewport | Appearance |
| --- | --- | --- |
| `desktop-light` | 1440 x 900 | Light, regular density |
| `desktop-dark` | 1440 x 900 | Dark, regular density |
| `desktop-compact` | 1440 x 900 | Light, compact density |
| `tablet-landscape` | 1024 x 768 | Light, regular density |
| `tablet-portrait` | 768 x 1024 | Light, regular density |
| `mobile` | 390 x 844 | Light, regular density |

## Evidence

Artifacts:

- Vue: `docs/phase-3/visual-regression/vue/`
- React: `docs/phase-3/visual-regression/react/`
- Diffs: `docs/phase-3/visual-regression/diff/`
- Aggregate metrics: `docs/phase-3/visual-regression/phase3-visual-metrics.json`

Results:

| Metric | Result |
| --- | --- |
| Vue screenshots | 78 |
| React screenshots | 78 |
| Diff images | 78 |
| Console errors | 0 |
| Vue page errors | 0 |
| Average mismatch ratio | 0.0444 |
| Minimum mismatch ratio | 0.0182 |
| Maximum mismatch ratio | 0.1029 |
| Horizontal overflow in captures | 0 |

The React capture records confirm that dark mode and compact density are actually applied:

- 13 records with `theme=dark, density=regular`
- 13 records with `theme=light, density=compact`
- 52 records with `theme=light, density=regular`

## Difference Analysis

The Vue reference uses deterministic visual fixtures. React uses the live Go-backed business data and therefore has different row counts, timestamps, names, chart values, and empty-state combinations. The largest differences are on:

- Dashboard mobile: live login volume and trend values differ from the fixture.
- Menus mobile: live menu tree content is larger than the fixture.
- Codegen mobile: live imported-table metadata differs from the fixture.
- Operation logs: React renders live audit summaries and event rows, while the Vue fixture is intentionally sparse.

The page shells, navigation structure, typography, materials, chart placement, table patterns, drawer/dialog surfaces, and responsive breakpoints remain aligned with the Vue reference. No missing page or fallback/mock runtime path was found.

## Commands

```powershell
cd frontend
$env:E2E_ADMIN_USERNAME="admin"
$env:E2E_ADMIN_PASSWORD="<local-test-password>"
node scripts/capture-phase3-pages.mjs
node scripts/capture-phase3-vue-pages.mjs
node scripts/compare-phase3-visuals.mjs
```

The visual scripts remove stale failure screenshots before capture and explicitly apply appearance cookies/local storage before navigation.
