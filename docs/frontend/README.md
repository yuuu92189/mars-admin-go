# Frontend Foundation

This directory records the Phase 2 frontend audit, architecture and acceptance evidence. The implementation lives in `frontend/`; `frontend-api/` remains the single OpenAPI TypeScript generation package.

## Status

- Phase 2 foundation: implemented.
- Reference design: current `scaffold-vue` source, including the Kintsugi login brand panel and four-stage terminal.
- Current backend contract: `api/openapi/v1/openapi.yaml`.
- Current frontend: Next.js App Router + React + TypeScript strict + Tailwind CSS + local Pearl Glass CSS.
- Business page migration: intentionally not part of Phase 2; known backend menu routes show an explicit Phase 3 placeholder, while unknown routes return 404.

## Documents

| Document | Purpose |
| --- | --- |
| [01-frontend-audit.md](01-frontend-audit.md) | Current repository and reference source audit |
| [02-vue-design-system.md](02-vue-design-system.md) | Extracted Pearl Glass tokens and visual rules |
| [03-vue-layout-analysis.md](03-vue-layout-analysis.md) | App shell, sidebar, header and responsive measurements |
| [04-component-mapping.md](04-component-mapping.md) | Vue component to React mapping and Phase 2 component set |
| [05-animation-mapping.md](05-animation-mapping.md) | GSAP/CSS/WAAPI behavior and React lifecycle rules |
| [06-api-integration-analysis.md](06-api-integration-analysis.md) | OpenAPI, auth, CSRF, permissions and page mapping |
| [07-frontend-architecture.md](07-frontend-architecture.md) | Final directory structure and server/client boundaries |
| [08-implementation-roadmap.md](08-implementation-roadmap.md) | Phase 2 delivery sequence and Phase 3 gates |
| [09-visual-acceptance-criteria.md](09-visual-acceptance-criteria.md) | Viewports, screenshots and visual checks |
| [10-phase-2-implementation-report.md](10-phase-2-implementation-report.md) | Actual implementation and file evidence |
| [11-phase-2-acceptance-report.md](11-phase-2-acceptance-report.md) | P2-01 through P2-20 acceptance results |

## Commands

```bash
cd frontend
pnpm install --frozen-lockfile
pnpm typecheck
pnpm lint
pnpm test
pnpm build
pnpm test:e2e
pnpm visual:compare
```

`pnpm visual:compare` expects the Vue reference on `http://127.0.0.1:5173` and the React app on `http://127.0.0.1:3000`; both URLs can be overridden with `VUE_REFERENCE_URL` and `REACT_URL`.

Authenticated Playwright checks are skipped unless `E2E_ADMIN_USERNAME` and `E2E_ADMIN_PASSWORD` are supplied. They must never point at production credentials. The authenticated file runs serially so a shared local account does not trip the backend login-attempt limit.
