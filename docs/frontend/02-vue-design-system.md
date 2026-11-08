# 02 Vue Design System

The Phase 2 CSS baseline is a direct local copy of the seven Vue reference layers. Tailwind is used for layout utilities; the Pearl Glass material remains authored in CSS variables.

## Core Tokens

| Token group   | Actual source values                                                              |
| ------------- | --------------------------------------------------------------------------------- |
| Ink           | `#101418`, `#596368`, `#7e888c`, `#99a2a5`                                        |
| Semantic      | Blue `#236de9`, Lime `#82dd2d`, Red `#ff5a50`                                     |
| Radius        | `8px`, `12px`, `16px`, `20px`, `24px`, plus `999px` pills                         |
| Layout        | Rail `232px`, collapsed rail `76px`, topbar `64px`                                |
| Control       | `36px` default, `30px` small                                                      |
| Table density | `48px` regular default, `56px` comfy, `40px` compact                              |
| Easing        | `cubic-bezier(.16, 1, .3, 1)`                                                     |
| Motion        | fast/normal/slow transitions use the copied CSS layer; GSAP uses the premium ease |

## Materials

| Class        | Role                                | Source behavior                                      |
| ------------ | ----------------------------------- | ---------------------------------------------------- |
| `.m-rail`    | Navigation rail                     | near-solid translucent background, 22px blur         |
| `.m-pill`    | Active navigation, segmented thumb  | strongest white edge, inset highlight, 22px blur     |
| `.m-float`   | Popover/dialog/palette              | 28px blur, saturated glass, largest ambient shadow   |
| `.m-surface` | Cards/tables                        | near-solid reading surface with edge highlight       |
| `.m-groove`  | Inputs and inactive segmented track | inset groove, soft glass line                        |
| `.m-glow`    | Pointer material                    | radial white/lime glow at leaf container coordinates |

The React implementation preserves the classes and adds pointer glow only where the original component used it. The material is not reduced to a single `backdrop-filter` rule.

## Theme and Density

- Light and dark values live under `:root[data-theme="dark"]`.
- System theme resolves from `prefers-color-scheme`.
- Density uses `data-density="comfy"` and `data-density="compact"`.
- An inline root script applies stored theme/density before hydration to avoid a first-paint flash.
- `prefers-reduced-transparency` removes glass blur and falls back to solid surfaces.

## Typography

- Primary stack: `SF Pro Display`, `SF Pro Text`, `PingFang SC`, system fonts.
- Mono stack: `SF Mono`, `JetBrains Mono`, `Menlo`, `Consolas`.
- Display scales: `56px`, `40px`, `28px`; page titles use the Vue 28px desktop / 22px mobile scale.

## Verification

The direct Vue/React comparison is generated with `pnpm visual:compare` and stored in:

- `frontend/visual-comparison/vue-login-desktop-light.png`
- `frontend/visual-comparison/react-login-desktop-light.png`
- `frontend/visual-comparison/vue-login-mobile-dark.png`
- `frontend/visual-comparison/react-login-mobile-dark.png`
- `frontend/visual-comparison/metrics.json`

At 1440×900, the Vue and React login stage, brand panel, login side and card bounds are identical. At 390×844, the brand panel height differs by about 1.5px and the login card starts about 0.75px lower; neither causes overflow or overlap.

The automated screenshot baselines are in:

- `frontend/e2e/visual.spec.ts-snapshots/login-desktop-light-desktop-win32.png`
- `frontend/e2e/visual.spec.ts-snapshots/login-mobile-dark-desktop-win32.png`
- `frontend/e2e/visual.spec.ts-snapshots/login-desktop-light-mobile-win32.png`
- `frontend/e2e/visual.spec.ts-snapshots/login-mobile-dark-mobile-win32.png`
- `frontend/visual-baselines/dashboard-desktop-light.png`
- `frontend/visual-baselines/dashboard-desktop-dark.png`
- `frontend/visual-baselines/dashboard-mobile-light.png`
- `frontend/visual-baselines/dashboard-mobile-dark.png`

The current Go captcha contract returns one image rather than the Vue-era light/dark pair. React keeps the Vue CSS layer and adds a narrowly scoped single-image compatibility class so the captcha remains visible in dark mode.
