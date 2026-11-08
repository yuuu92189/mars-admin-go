# Legacy Data Migration

The old SQL dump is an audit source, not an executable migration.

## Mapping

`cmd/legacy-import` creates a row-count and table inventory report:

```bash
go run ./cmd/legacy-import \
  -source ../mars-admin-master/sql/mars-system.sql \
  -report docs/migration/import-report.json \
  -dry-run=true
```

## Production Import Gates

1. Run in a restricted environment with the dump encrypted at rest.
2. Create deterministic mappings from legacy integer IDs to UUIDv7 values.
3. Resolve duplicate usernames/emails/phones before insert.
4. Recalculate department paths instead of importing `ancestors` text.
5. Convert MySQL local `DATETIME` values using an explicitly reviewed source timezone.
6. Remove or rotate all credentials embedded in `sys_config_group`.
7. Reconcile user-role/menu/dept relationships after import.
8. Keep old and new row counts plus rejected-row reasons.
9. Do not mark migration successful until relationship and authorization checks pass.

Live import is intentionally blocked until those policies are reviewed. This prevents accidental ingestion of exposed secrets and personal data.

## Backup And Restore Rehearsal

`scripts/cutover-rehearsal.sh` creates a custom-format backup, restores it into an explicitly
separate rehearsal database, verifies the migration version and representative row counts, and
writes a JSON report. The Phase 4 local rehearsal passed against PostgreSQL 18.6; evidence is in
[`cutover-rehearsal.md`](cutover-rehearsal.md).

Production cutover remains a separately approved operation. The script refuses to run when the
source and rehearsal database URLs are identical.
