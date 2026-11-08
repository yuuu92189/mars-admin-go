# Cutover And Restore Rehearsal

The Phase 4 rehearsal was executed against the local Compose PostgreSQL instance, not a
production database. The source was backed up with `pg_dump --format=custom`, restored into an
isolated `mars_admin_rehearsal` database, checked for migration version and representative row
counts, then the isolated database was dropped.

## Result

- Source backup: custom PostgreSQL dump, SHA-256 recorded.
- Restore target: isolated `mars_admin_rehearsal`.
- Migration version after restore: `25`.
- Representative counts: users `32`, menus `26`, settings `8`, payment orders `0`.
- Result: PASS.

Machine-readable evidence:

- [`cutover-rehearsal-report.json`](cutover-rehearsal-report.json)

## Reproducible Procedure

Use the scripts in [`scripts/`](../../scripts) with explicit source and rehearsal URLs:

```bash
export DATABASE_URL='postgres://.../source'
export REHEARSAL_DATABASE_URL='postgres://.../rehearsal'
export REHEARSAL_CONFIRM=1
scripts/cutover-rehearsal.sh backup/rehearsal.dump backup/rehearsal-report.json
```

The rehearsal script refuses to run when source and target URLs are equal. Production cutover
still requires a separately approved maintenance window, target provisioning, secret rotation and
owner sign-off; this local rehearsal is not a production cutover.
