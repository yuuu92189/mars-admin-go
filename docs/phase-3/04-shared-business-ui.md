# Shared Business UI

## Components Added

`frontend/src/components/business/business-page.tsx` provides:

- `BusinessPage`: consistent page container and header.
- `SearchForm`: query/reset actions with submit handling.
- `DataState`: loading, error, permission, empty, and content states.
- `StatusTag`: canonical status labels and tones.
- `BulkActionBar`: fixed multi-select action surface.
- `TreeNav`: recursive department/menu navigation.
- `DetailGrid`: read-only detail metadata.
- `SectionBand`: stacked monitoring bands.
- `formatDateTime`: locale-safe timestamp formatting.

`frontend/src/components/charts/metric-charts.tsx` provides:

- SVG trend chart.
- SVG bar chart with failure series.
- Tick-ring success gauge.
- Login heat grid.
- Audit swimlane.

## Loading Policy

`useDeferredEffect` schedules initial loads after the current render. This avoids React's synchronous set-state-in-effect anti-pattern while preserving explicit refresh actions and cancellable timers.

## API Policy

- All business pages call modules under `frontend/src/lib/api/`.
- Query parameters are serialized with `URLSearchParams`.
- Request bodies use generated OpenAPI schema aliases.
- RFC 9457 errors flow through `ApiError`.
- CSRF, cookies, refresh, request IDs, and blob downloads remain in the shared client.

## Explicit Non-Goals

- No second UI framework.
- No global state store.
- No mock business data path.
- No generic mega-component for all business logic.
