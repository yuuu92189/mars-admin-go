# Mars Admin Frontend

Phase 2.1 Next.js frontend foundation for Mars Admin Go Edition.

## Run

```bash
pnpm install --frozen-lockfile
pnpm dev
```

The single server-side Go API target defaults to `http://localhost:8080`. Override it with:

```bash
API_BASE_URL=http://localhost:8081 pnpm dev
```

The browser talks to the same-origin Next Route Handler at `/api/v1/*`; the proxy preserves cookies and CSRF and strips the browser `Origin` before forwarding upstream.

`API_PROXY_TARGET` and `API_INTERNAL_BASE_URL` remain deprecated compatibility aliases. They are
accepted only when they agree with `API_BASE_URL`; conflicting values fail startup.

## Verify

```bash
pnpm typecheck
pnpm lint
pnpm format:check
pnpm test
pnpm check:api-types
pnpm build
pnpm test:e2e
pnpm visual:compare
pnpm visual:compare:shell
```

Authenticated Playwright tests are skipped unless `E2E_ADMIN_USERNAME` and `E2E_ADMIN_PASSWORD` are provided. Use local test credentials only.

`pnpm visual:compare` captures Vue and React login pages under the same viewport, theme, captcha and reduced-motion conditions. Start the Vue reference on port 5173 and the React app on port 3000 first, or set `VUE_REFERENCE_URL` and `REACT_URL`. Results are written to `visual-comparison/`.

`pnpm visual:compare:shell` captures expanded/collapsed Light/Dark, density, Command Palette,
Appearance Dialog, user menu and mobile Dashboard pairs. It requires a local test API at
`VISUAL_API_URL` (default `http://127.0.0.1:8081`) plus local `E2E_ADMIN_USERNAME` and
`E2E_ADMIN_PASSWORD`. Results and diffs are written under
`../docs/frontend/visual-regression/artifacts/`.

## Scope

Phase 2.1 includes the design system, core UI primitives, shell, login, dashboard, unified API
target, Cookie refresh, real password/logout flows and test infrastructure. Business pages
intentionally remain Phase 3 placeholders.
