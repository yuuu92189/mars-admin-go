# Import Validation

## Generated Dry Run

The command `go run ./cmd/legacy-import -source ../mars-admin-master/sql/mars-system.sql -report docs/migration/import-report.json -dry-run=true` found:

- 35 populated tables
- 170,671 insert rows
- 1 intentional exclusion
- live import disabled

## Required Validation Before Import

- row counts by table match the reviewed report
- all legacy IDs have deterministic UUID mappings
- orphan user-role, role-menu, user-dept and group-member references are zero
- username/email/phone duplicates are resolved
- department graph has no cycles
- money values round-trip exactly
- workflow definitions pass the Go definition validator
- job handlers resolve to the explicit safe handler registry
- sensitive configuration values are removed and rotated
- every imported session/token is discarded

## After Import

- compare counts and sampled hashes
- run permission and department-scope integration tests
- verify browser login for a migrated administrator
- verify direct/group message history and read counts
- verify notices and workflow histories
- preserve an immutable migration report with rejected-row reasons

## Full Dump Execution

On 2026-11-07 the complete `mars-system.sql` dump (27,857,570 bytes) was loaded into an isolated MySQL 8.0.36 container. The reviewed importer then migrated it into a separate PostgreSQL 18.6 database, `mars_admin_import_validation`.

Result: **completed**, zero failed rows.

Imported counts include:

- 397 active legacy users
- 8 departments and 6 positions
- 2 roles, 397 user-role links and 187 role-menu links
- 119 legacy menus plus the 26 default Go menus
- 792 direct/group messages
- 18 chat groups and 34 group memberships
- 2 published notices with recipients and send logs
- 14 redacted legacy setting groups
- 1 copied local file object; 9 metadata rows were skipped because the source files were absent from the supplied upload directory

Validation after import:

- `identity_users`: 397
- `organization_departments`: 8
- `access_roles`: 2
- `access_menus`: 145
- `legacy_id_map`: 1,359
- orphan `access_user_roles`: 0
- orphan `access_role_menus`: 0

The full report is stored at `docs/migration/full-import-report.json`.

A second execution completed with zero inserted rows, zero failed rows and 2,605 skipped rows, demonstrating idempotent mapped imports. Its report is stored at `docs/migration/full-import-idempotency-report.json`.

Warnings retained in the report:

- 9 file records reference objects absent from the provided legacy upload tree.
- one orphan chat group references an owner absent from the imported user set.
- two positions referenced a missing parent and were imported as roots.
- provider configuration values were redacted before persistence.
