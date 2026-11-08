# Generated API Types

This directory is the Phase 1 TypeScript contract baseline for the future React/Next.js client. It does not contain application pages.

## Commands

```bash
cd frontend-api
npm ci
npm run generate
npm run check
```

`npm run generate` reads `../api/openapi/v1/openapi.yaml` and writes `src/schema.ts`. The generated file must not be edited manually.

The future Next.js project can import `src/schema.ts` directly or configure its generated-client path to consume the same OpenAPI document. The Go repository remains the contract owner until the frontend workspace is split or made a monorepo.

`npm run check` runs the generator followed by TypeScript strict compilation. CI runs the same command through `scripts/check.sh`.
