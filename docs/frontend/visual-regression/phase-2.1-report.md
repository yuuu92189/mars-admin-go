# Phase 2.1 Visual Regression Report

## Method

The comparison starts both the real Vue reference and the real React production application. The
same Playwright Chromium version, viewport, device scale, theme, density, reduced-motion setting and
settle delay are used for each pair. Captcha and Dashboard data are deterministic fixtures only for
screenshot isolation; they do not enter application runtime paths.

Each pair records:

- Full-page Vue screenshot.
- Full-page React screenshot.
- Pixel diff image.
- Viewport, frame, rail, topbar, content and radius metrics.
- Mismatch pixel count and ratio.

The script uses `pixelmatch` at the same threshold for every scenario. No large masks, screenshot
backgrounds, hidden content or lowered per-scenario thresholds are used.

## Reproduce

```bash
cd frontend
VUE_REFERENCE_URL=http://127.0.0.1:5173 \
REACT_URL=http://127.0.0.1:3000 \
VISUAL_API_URL=http://127.0.0.1:8081 \
E2E_ADMIN_USERNAME=<local-test-user> \
E2E_ADMIN_PASSWORD=<local-test-password> \
pnpm visual:compare:shell
```

Login comparison:

```bash
pnpm visual:compare
```

## Shell and Dashboard Results

| Scenario               | Mismatch | Vue rail | React rail | Vue topbar | React topbar | Result                              |
| ---------------------- | -------: | -------: | ---------: | ---------: | -----------: | ----------------------------------- |
| Expanded Light         |    5.79% |    232px |      232px |       64px |         64px | Geometry aligned                    |
| Collapsed Dark         |    5.16% |     76px |       76px |       64px |         64px | Geometry aligned                    |
| Compact Light          |    5.79% |    232px |      232px |       64px |         64px | Geometry aligned                    |
| Command Palette        |   14.04% |    232px |      232px |       64px |         64px | Shell aligned; content list differs |
| Appearance Dialog      |    7.23% |    232px |      232px |       64px |         64px | Shell aligned; copy/data differs    |
| User Menu              |    5.79% |    232px |      232px |       64px |         64px | Geometry aligned                    |
| Dashboard Mobile Light |    7.68% |    260px |      260px |       64px |         64px | Responsive geometry aligned         |
| Dashboard Mobile Dark  |    6.75% |    260px |      260px |       64px |         64px | Responsive geometry aligned         |

Desktop frame radius is `30px` in both applications. Mobile frame radius is `0px` in both. No
horizontal overflow was observed in the paired captures.

## Login Results

The login comparison covers `1440x900` and `390x844`, each in Light and Dark. Brand-panel and login
card geometry match exactly:

- Desktop brand panel: `795.81px`.
- Desktop login card: `420px`.
- Mobile brand panel: `390px`.
- Mobile login card: `354px`.
- Scroll width equals viewport width in every scenario.

Artifacts are in `frontend/visual-comparison/`.

## Difference Analysis

- Dashboard content differs because Vue renders trend, activity and security fixtures while React
  intentionally renders only the canonical Go counters available in Phase 2.1.
- The Command Palette difference is concentrated in result rows because React uses the current
  backend menu set while the Vue fixture contains a shorter route list.
- Text, locale rendering and non-deterministic dashboard values contribute to remaining small
  pixel differences.
- Sidebar widths, topbar height, frame geometry, responsive breakpoints, modal shell and material
  treatment are aligned.

## Artifacts

- [Shell metrics](artifacts/shell-metrics.json)
- [Vue expanded shell](artifacts/vue-shell-expanded-light-desktop.png)
- [React expanded shell](artifacts/react-shell-expanded-light-desktop.png)
- [Expanded diff](artifacts/diff-shell-expanded-light-desktop.png)
- [Vue command palette](artifacts/vue-shell-command-palette-light-desktop.png)
- [React command palette](artifacts/react-shell-command-palette-light-desktop.png)
- [Command Palette diff](artifacts/diff-shell-command-palette-light-desktop.png)
- [Vue mobile dashboard](artifacts/vue-dashboard-mobile-light.png)
- [React mobile dashboard](artifacts/react-dashboard-mobile-light.png)
- [Mobile dashboard diff](artifacts/diff-dashboard-mobile-light.png)

## Conclusion

The Shell and login foundation has direct Vue/React evidence and aligned geometry. Pixel parity is
not claimed for content that intentionally differs in Phase 2.1. Dashboard business-module parity
remains a Phase 3 item.
