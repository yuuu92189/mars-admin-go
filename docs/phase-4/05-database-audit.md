# Phase 4 Database Audit

## Baseline

- PostgreSQL is the source of truth.
- 25 Goose migrations, latest version `000025_payment_ledger.sql`.
- SQLC is the default generated data-access path; bounded raw SQL remains in adapters where
  dynamic read models are required.
- Historical migrations were not edited. Schema-affecting work in this phase did not require a
  new migration.

## Findings And Fixes

- Disabled or deleted custom-scope roles could still contribute department scope in identity,
  organization and menu queries. Active-role and `deleted_at` predicates were added to the SQL.
- Menu hierarchy cycle prevention was missing beyond self-parent checks. A recursive descendant
  lookup and service validation now reject cycles.
- Sensitive settings could be downgraded when an existing row was updated under a key that did not
  match the heuristic marker list. The repository now preserves the sensitive flag and the
  service encrypts based on existing sensitivity.
- Multi-table Codegen imported-table deletion now runs in one transaction.
- Payment orders and provider callbacks are persisted in PostgreSQL. Callback events have a
  provider/event unique constraint, amount/currency validation and an atomic paid-state update.

## Verification

SQLC regeneration is reproducible with:

```bash
$env:CGO_ENABLED='0'; go tool sqlc generate
```

Integration tests run against PostgreSQL 18.6 and Redis 8.10.1 through Testcontainers. Existing
custom-scope, soft-delete uniqueness, transaction, outbox and workflow concurrency tests remain
green.
