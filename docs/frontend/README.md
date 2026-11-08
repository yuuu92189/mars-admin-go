# Frontend Foundation

This directory records the Phase 2 frontend audit, architecture and acceptance evidence. The implementation lives in `frontend/`; `frontend-api/` remains the single OpenAPI TypeScript generation package.

## Status

- Phase 2 foundation: implemented.
- Phase 2.1 hardening: implemented; local acceptance passed, remote CI must be recorded for the
  final pushed commit.
- Reference design: current `scaffold-vue` source, including the Kintsugi login brand panel and four-stage terminal.
- Current backend contract: `api/openapi/v1/openapi.yaml`.
- Current frontend: Next.js App Router + React + TypeScript strict + Tailwind CSS + local Pearl Glass CSS.
- Business page migration: intentionally not part of Phase 2; known backend menu routes show an explicit Phase 3 placeholder, while unknown routes return 404.

## Documents

| Document                                                                             | Purpose                                                  |
| ------------------------------------------------------------------------------------ | -------------------------------------------------------- |
| [01-frontend-audit.md](01-frontend-audit.md)                                         | Current repository and reference source audit            |
| [02-vue-design-system.md](02-vue-design-system.md)                                   | Extracted Pearl Glass tokens and visual rules            |
| [03-vue-layout-analysis.md](03-vue-layout-analysis.md)                               | App shell, sidebar, header and responsive measurements   |
| [04-component-mapping.md](04-component-mapping.md)                                   | Vue component to React mapping and Phase 2 component set |
| [05-animation-mapping.md](05-animation-mapping.md)                                   | GSAP/CSS/WAAPI behavior and React lifecycle rules        |
| [06-api-integration-analysis.md](06-api-integration-analysis.md)                     | OpenAPI, auth, CSRF, permissions and page mapping        |
| [07-frontend-architecture.md](07-frontend-architecture.md)                           | Final directory structure and server/client boundaries   |
| [08-implementation-roadmap.md](08-implementation-roadmap.md)                         | Phase 2 delivery sequence and Phase 3 gates              |
| [09-visual-acceptance-criteria.md](09-visual-acceptance-criteria.md)                 | Viewports, screenshots and visual checks                 |
| [10-phase-2-implementation-report.md](10-phase-2-implementation-report.md)           | Actual implementation and file evidence                  |
| [11-phase-2-acceptance-report.md](11-phase-2-acceptance-report.md)                   | P2-01 through P2-20 acceptance results                   |
| [12-phase-2.1-pre-implementation-audit.md](12-phase-2.1-pre-implementation-audit.md) | Phase 2.1 baseline and gap audit                         |
| [13-phase-2.1-implementation-report.md](13-phase-2.1-implementation-report.md)       | Phase 2.1 implementation and local verification          |
| [14-phase-2.1-final-acceptance.md](14-phase-2.1-final-acceptance.md)                 | Phase 2.1 acceptance matrix                              |
| [vue-react-parity-matrix.md](vue-react-parity-matrix.md)                             | Vue behavior to React implementation and evidence        |
| [visual-regression/phase-2.1-report.md](visual-regression/phase-2.1-report.md)       | Vue/React Shell and Dashboard visual evidence            |

## Commands

```bash
cd frontend
pnpm install --frozen-lockfile
pnpm format:check
pnpm typecheck
pnpm lint
pnpm test
pnpm check:api-types
pnpm build
pnpm test:e2e
pnpm visual:compare
pnpm visual:compare:shell
```

`pnpm visual:compare` expects the Vue reference on `http://127.0.0.1:5173` and the React app on `http://127.0.0.1:3000`; both URLs can be overridden with `VUE_REFERENCE_URL` and `REACT_URL`.

`pnpm visual:compare:shell` also requires a local test API at `VISUAL_API_URL` (default
`http://127.0.0.1:8081`) and local `E2E_ADMIN_USERNAME` / `E2E_ADMIN_PASSWORD`.

Authenticated Playwright checks are skipped unless `E2E_ADMIN_USERNAME` and `E2E_ADMIN_PASSWORD` are supplied. They must never point at production credentials. The real password lifecycle creates and deletes an isolated temporary user.

The frontend uses one server-side `API_BASE_URL`. Browser traffic always stays on the Next
same-origin proxy. `API_PROXY_TARGET` and `API_INTERNAL_BASE_URL` are deprecated compatibility
aliases and are rejected when they conflict.
