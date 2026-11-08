# Phase 4 Business Scope Audit

## Supported Business Surface

The Go backend supports Identity, Access, Organization, System, Notification, Scheduler, Audit,
Monitoring, Codegen, Storage, Messaging, Workflow, Integrations and Outbox capabilities.
Infrastructure modules do not require independent React pages.

The 13 migrated business pages are:

1. Dashboard
2. Users
3. Roles
4. Menus
5. Departments
6. Positions
7. Dictionaries
8. Settings
9. Notices
10. Scheduler
11. Operation logs
12. Login logs
13. Codegen

Each page has an explicit App Router route, server-side access check, typed API module and
loading/empty/error/permission handling. The current frontend uses the live Go API rather than
runtime fixtures for these pages.

## Excluded Domains

The following are intentionally excluded:

- Asset Management
- Vulnerability Management
- Remediation Work Orders

No tables, APIs, permissions, menus, mock data, scanners or fake pages were added for these
domains. `scaffold-vue` remains a visual/interaction reference only.

The login brand panel originally retained the reference project's asset/vulnerability/repair
wording. That user-facing copy was replaced with the actual Identity, Access, Workflow and Audit
capabilities; the four-stage visual treatment itself was retained.

Historical Phase 1 and Phase 3 documents now carry scope notices. Their old "blocked", "missing"
or "Phase 6" references are retained only as historical evidence and are not current backlog
items.

## Result

The business scope is internally consistent with the Go backend. No unsupported page is reported
as migrated, and no excluded domain is listed as a missing project deliverable.
