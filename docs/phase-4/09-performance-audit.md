# Phase 4 Performance Audit

## Fixed Risks

- The Next catch-all proxy no longer buffers non-JSON request bodies. Uploads are streamed to the
  Go API, whose handler applies the configured byte limit.
- Storage uploads stage to a bounded temporary file and remove multipart temporary files.
- Codegen ZIP responses stream through `archive/zip` and reject more than 50 selected tables.
- API pagination is bounded for the main list endpoints.
- Prometheus HTTP metrics cap the in-memory recent-call ring at 1,000 records.

## Database Observations

- List queries use explicit limits and stable ordering.
- Custom data-scope predicates are bounded by actor roles and memberships.
- Existing query indexes cover scheduler due jobs, storage cleanup, notification send logs,
  setting revisions, active refresh sessions, workflow role todos and login-event usernames.
- No production-scale `EXPLAIN ANALYZE` dataset was available; no production performance claim is
  made.

## Residual Conditions

- External provider retries can duplicate delivery because provider-side idempotency is not
  universally available.
- Historical compatibility endpoints were removed. Any future history-statistics page must use a
  dedicated canonical query rather than reintroducing bounded in-memory compatibility pagination.
