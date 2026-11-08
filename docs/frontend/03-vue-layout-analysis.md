# 03 Vue Layout Analysis

## Shell Geometry

The source `layout.css` establishes a stage with 8px outer padding and an inner `.app-frame` with:

- `height: calc(100dvh - 16px)` and a 560px minimum.
- `grid-template-columns: var(--rail-w) minmax(0, 1fr)`.
- `border-radius: 30px`.
- Static ambient radial lights over `--canvas`.

React maps these to `.stage` and `.app-frame` without changing the geometry.

## Rail

- Default width: 232px.
- Collapsed width: 76px.
- Brand height: 64px.
- Navigation item height: 38px, 12px horizontal padding, 11px icon-label gap.
- Active state uses the `.nav-thumb.m-pill` visual language.
- On mobile below 820px the rail becomes fixed at 260px and translates off-canvas.

React `Sidebar` keeps the same classes and uses a `navGroups` model derived from the canonical `/auth/me` menu tree.

## Header and Content

- Header height: 64px.
- Desktop padding: `0 20px 0 14px`.
- Search trigger: 240px, collapsing to icon at 1080px.
- Main content scrolls independently with overscroll containment.
- Page padding: `26px 28px 40px`; mobile: `18px 16px 32px`.
- Page title: 28px desktop, 22px mobile.

## Responsive Behavior

| Breakpoint | Behavior |
| --- | --- |
| 1180px | Expanded rail contracts to 208px; command trigger contracts |
| 1080px | Command trigger becomes icon; split views stack |
| 820px | Rail becomes mobile drawer; stage loses outer padding; breadcrumbs hide |
| 760px | Bento cards and page grids collapse to one column |
| 390/360px | Login brand panel condenses above the card; terminal/journey/footer hide; card remains 354px wide with no overflow |

The React shell uses `matchMedia('(max-width: 820px)')`, an explicit mobile scrim and an accessible rail toggle.

The current Vue login page is not the older hidden-canvas variant. React ports the current `brand-panel` layout: the brand copy stays visible on mobile, while only the terminal and journey strip collapse. Direct Vue/React screenshots and geometry are recorded under `frontend/visual-comparison/`.
