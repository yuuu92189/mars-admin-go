# Mars Admin Frontend

Phase 2 Next.js frontend foundation for Mars Admin Go Edition.

## Run

```bash
pnpm install --frozen-lockfile
pnpm dev
```

The default Go API target is `http://localhost:8080`. Override it with:

```bash
API_PROXY_TARGET=http://localhost:8081 pnpm dev
```

The browser talks to the same-origin Next Route Handler at `/api/v1/*`; the proxy preserves cookies and CSRF and strips the browser `Origin` before forwarding upstream.

## Verify

```bash
pnpm typecheck
pnpm lint
pnpm format:check
pnpm test
pnpm build
pnpm test:e2e
pnpm visual:compare
```

Authenticated Playwright tests are skipped unless `E2E_ADMIN_USERNAME` and `E2E_ADMIN_PASSWORD` are provided. Use local test credentials only.

`pnpm visual:compare` captures Vue and React login pages under the same viewport, theme, captcha and reduced-motion conditions. Start the Vue reference on port 5173 and the React app on port 3000 first, or set `VUE_REFERENCE_URL` and `REACT_URL`. Results are written to `visual-comparison/`.

## Scope

Phase 2 includes the design system, core UI primitives, shell, login, dashboard, API/auth foundation and test infrastructure. Business pages intentionally remain Phase 3 placeholders.
