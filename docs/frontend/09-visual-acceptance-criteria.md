# 09 Visual Acceptance Criteria

## Reference

The visual reference is the local `scaffold-vue` source and its seven CSS layers, not a generic admin template. The React implementation preserves the original class names and material tokens.

## Required Viewports

| Viewport | Purpose | Status |
| --- | --- | --- |
| 1440px | wide desktop | PASS: Vue/React comparison, login/dashboard |
| 1280px | desktop screenshot | PASS: login/dashboard Light and Dark |
| 1024px | tablet landscape | PASS: login/dashboard, no horizontal overflow |
| 768px | tablet portrait | PASS: login/dashboard, no horizontal overflow |
| 390px | mobile | PASS: Vue/React comparison, login/dashboard and mobile rail |
| 360px | small mobile | PASS: login, no horizontal overflow |

## Screenshot Evidence

- `frontend/visual-comparison/vue-login-desktop-light.png`
- `frontend/visual-comparison/react-login-desktop-light.png`
- `frontend/visual-comparison/vue-login-mobile-dark.png`
- `frontend/visual-comparison/react-login-mobile-dark.png`
- `frontend/visual-comparison/metrics.json`
- `frontend/e2e/visual.spec.ts-snapshots/login-desktop-light-desktop-win32.png`
- `frontend/e2e/visual.spec.ts-snapshots/login-desktop-light-mobile-win32.png`
- `frontend/e2e/visual.spec.ts-snapshots/login-mobile-dark-desktop-win32.png`
- `frontend/e2e/visual.spec.ts-snapshots/login-mobile-dark-mobile-win32.png`
- `frontend/visual-baselines/dashboard-desktop-light.png`
- `frontend/visual-baselines/dashboard-desktop-dark.png`
- `frontend/visual-baselines/dashboard-mobile-light.png`
- `frontend/visual-baselines/dashboard-mobile-dark.png`
- `frontend/visual-baselines/dashboard-desktop-collapsed-light.png`
- `frontend/visual-baselines/dashboard-mobile-open-light.png`
- `frontend/visual-baselines/dashboard-1440-light.png`
- `frontend/visual-baselines/dashboard-1024-light.png`
- `frontend/visual-baselines/dashboard-768-light.png`
- `frontend/visual-baselines/dashboard-390-light.png`
- `frontend/visual-baselines/login-1440-light.png`
- `frontend/visual-baselines/login-1024-light.png`
- `frontend/visual-baselines/login-768-light.png`
- `frontend/visual-baselines/login-360-light.png`

The authenticated dashboard screenshots use a real current Go API and local development session. The Vue/React login captures use the same viewport, theme, deterministic captcha and reduced-motion setting. They are evidence of the Phase 2 shell/dashboard and login reconstruction, not a claim that Vue business pages are migrated.

## Checks

- Pearl Glass surfaces remain visible and readable.
- Sidebar width/collapse and mobile off-canvas behavior match source measurements; both states have E2E screenshots.
- Header command/theme/user actions remain reachable.
- Light and dark tokens do not introduce a different layout.
- Reduced motion disables deterministic visual noise.
- Text and controls do not overlap at 390px.
- The single-image canonical captcha remains visible in Light and Dark themes.
- No fake dashboard statistics are rendered.
