# 05 Animation Mapping

## Source Findings

The Vue reference uses:

- GSAP 3.15 for shell/page entrances, login intro, card tilts, shared-element overlays and chart reveals.
- CSS transitions for hover, focus, material changes, segmented thumb and switch fill.
- CSS keyframes for spinner, toast progress and shimmer/skeleton.
- Canvas RAF loops for neural/background visualizations, relation graphs and organization canvas in the wider Vue component library.
- `prefers-reduced-motion` and reduced-transparency media queries.

## React Mapping

| Behavior | Implementation | Cleanup |
| --- | --- | --- |
| Shell entrance | `useShellEntrance` + `gsap.context` | `ctx.revert()` on unmount |
| Dashboard reveal | `useCardEntrance` + GSAP stagger | `ctx.revert()` on unmount |
| Login intro | `useLayoutEffect` GSAP timeline for brand, journey, footer and card | `timeline.revert()` and listener removal |
| Login success | GSAP button-to-orb, ring and check timeline | timeline completes before route replacement |
| Login card pointer tilt/glow | `gsap.quickTo` plus CSS custom coordinates | kill tweens and clear tilt functions on unmount |
| Login brand terminal | React intervals for typing, stage rotation and spinner | clear all timers on unmount/reduced motion |
| Navigation thumb | measured transform with CSS transition | `ResizeObserver.disconnect()` |
| Hover/focus/material | copied CSS transitions | CSS-only |
| Modal/drawer/toast | CSS transition plus portal lifecycle | body overflow and key listeners restored |

## Reduced Motion

- `prefersReducedMotion()` gates GSAP and canvas animation.
- Login terminal text is rendered in its completed state when reduced motion is requested.
- CSS base layer forces animation/transition durations to near-zero under reduced motion.
- Material glow is disabled for coarse pointers and reduced motion.
- E2E visual tests set `reducedMotion: "reduce"` so screenshots are deterministic.

## Not Implemented

Complex Vue chart timelines, Flip shared-element morphs, organization canvas pan/zoom and drag-board spring physics remain Phase 3+ because their owning business pages are not part of Phase 2.
