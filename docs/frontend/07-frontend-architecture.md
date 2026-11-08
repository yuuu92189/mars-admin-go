# 07 Frontend Architecture

## Final Directory

```text
frontend/
├── e2e/
│   ├── authenticated.spec.ts
│   ├── login.spec.ts
│   ├── responsive.spec.ts
│   └── visual.spec.ts
├── scripts/capture-visual-comparison.mjs
├── visual-baselines/
├── visual-comparison/
├── src/
│   ├── app/
│   │   ├── (auth)/login/page.tsx
│   │   ├── (dashboard)/layout.tsx
│   │   ├── (dashboard)/dashboard/page.tsx
│   │   ├── (dashboard)/settings/profile/page.tsx
│   │   ├── api/[...path]/route.ts
│   │   ├── globals.css
│   │   ├── layout.tsx
│   │   └── providers.tsx
│   ├── components/
│   │   ├── auth/
│   │   ├── business/
│   │   ├── layout/
│   │   ├── providers/
│   │   ├── ui/
│   │   └── visual/
│   ├── features/
│   │   ├── auth/
│   │   └── dashboard/
│   ├── lib/
│   │   ├── api/
│   │   ├── navigation.ts
│   │   ├── permissions.ts
│   │   └── motion.ts
│   ├── styles/
│   │   ├── tokens.css
│   │   ├── base.css
│   │   ├── materials.css
│   │   ├── components.css
│   │   ├── instruments.css
│   │   ├── layout.css
│   │   └── pages.css
│   └── test/setup.ts
├── package.json
├── pnpm-lock.yaml
├── tsconfig.json
├── next.config.ts
├── postcss.config.mjs
├── eslint.config.mjs
└── playwright.config.ts
```

## Boundaries

- Server Components: root layout, protected dashboard layout and cookie-to-API session validation.
- Client Components: login form, shell interaction, theme/toast providers, dashboard fetch and overlays.
- Feature code owns API calls; UI primitives are presentation-only.
- `frontend-api` remains the contract owner and generation source.
- Known backend menu routes resolve to an explicit Phase 3 placeholder; unknown authenticated paths call `notFound()`.
- Tailwind handles layout/responsive utility composition; Pearl Glass remains CSS variables and preserved custom CSS.

## Dependency Policy

No Redux/MobX/Zustand, TanStack Query, SWR, Axios, Ant Design, MUI, Chakra, shadcn or second UI/CSS/animation framework is used. React context is sufficient for theme, toast and current user.

## Security Boundary

The Next layer never treats middleware or client visibility as authorization. Go authorizes every API request. The server layout only prevents an unauthenticated shell render; it is not a substitute for backend permission checks.
