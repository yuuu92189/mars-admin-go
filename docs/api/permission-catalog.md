# Permission Catalog

This catalog is the Phase 1 permission baseline for the canonical Go API.

## Layers

| Layer | Meaning | Enforcement |
| --- | --- | --- |
| API permission | Required for a protected HTTP operation | `RequirePermission` / `RequireAnyPermission` |
| Menu permission | Controls whether a menu is visible | Access menu query |
| Route permission | Controls page navigation in the frontend | Frontend route guard + API authorization |
| Button permission | Controls whether an action is displayed | Frontend permission component |
| Data scope | Restricts rows/records visible to an actor | Repository/SQL predicate |

Menu visibility and button visibility never replace API authorization.

## Canonical Codes

The source of truth for seeded codes is `db/migrations/000007_seed_authorization.sql`. The automated check in `internal/platform/httpserver/permissions_test.go` fails if a route uses a permission that is absent from the seed.

| Code | API responsibility |
| --- | --- |
| `user:read` | Read users and user sessions |
| `user:create` | Create users |
| `user:update` | Update users, reset passwords, revoke sessions |
| `user:delete` | Delete users |
| `role:read` | Read roles, permissions and menus |
| `role:create` | Create/update/delete non-super roles |
| `role:assign` | Assign menus/permissions and manage menus |
| `department:read` | Read departments and positions |
| `department:manage` | Create/update/delete departments and positions |
| `dictionary:read` | Read dictionary types/items |
| `dictionary:manage` | Manage dictionary types/items |
| `settings:read` | Read settings and integration status |
| `settings:manage` | Update settings and create payment sessions |
| `audit:read` | Read audit/login events, monitoring and metrics |
| `file:upload` | Upload files |
| `file:download` | List/read/download files |
| `file:delete` | Delete files |
| `message:send` | Send/block/group-manage messages |
| `message:read` | Read conversations, groups, receipts |
| `notice:read` | Read own/all notices and unread counts |
| `notice:manage` | Create/update/publish/withdraw notices |
| `scheduler:read` | Read jobs and run history |
| `scheduler:manage` | Create/update/pause/resume/run/delete jobs |
| `workflow:read` | Read definitions, visible instances, events and tasks |
| `workflow:manage` | Create/publish definitions, start/terminate instances |
| `workflow:approve` | Approve/reject assigned tasks |
| `codegen:read` | Read table metadata and imported configuration |
| `codegen:manage` | Import, sync, preview and mutate codegen configuration |

`role:create` is intentionally kept as the current coarse write permission for role mutations. A future migration may split it into `role:update` and `role:delete`, but Phase 1 does not silently change existing role grants.

## Automated Verification

Phase 1 final acceptance adds a route-level cross-check in `internal/platform/httpserver/permission_routes_test.go`. It parses the actual canonical handler route registrations, including group and route middleware, and compares:

- Actual route method/path to OpenAPI operations.
- Route permission codes to this catalog.
- Route permission codes to `db/migrations/000007_seed_authorization.sql`.
- Seed permissions back to this catalog.
- Public, authenticated-only and business-permission routes.

The current matrix is 137 canonical operations: 12 public, 12 authenticated-only and 113 business-permission routes. `RequireAnyPermission` OR semantics have a separate HTTP middleware test.

## Legacy Vue Mapping

The reference Vue application uses older permission strings. The mapping is documentation for migration; it is not an authorization bypass.

| Legacy code | Canonical code |
| --- | --- |
| `system:user:list` | `user:read` |
| `system:user:add` | `user:create` |
| `system:user:edit` | `user:update` |
| `system:user:remove` | `user:delete` |
| `system:user:resetPwd` | `user:update` |
| `system:role:list` | `role:read` |
| `system:role:add` | `role:create` |
| `system:role:edit` | `role:assign` |
| `system:role:remove` | `role:create` |
| `system:menu:add/edit/remove` | `role:assign` |
| `system:dept:*` | `department:read/manage` |
| `system:post:*` | `department:read/manage` |
| `system:dict:*` | `dictionary:read/manage` |
| `system:config:*` | `settings:read/manage` |
| `system:notice:add/remove` | `notice:manage` |
| `monitor:operlog:list/remove` | `audit:read` |
| `monitor:loginlog:list/remove` | `audit:read` |
| `monitor:job:*` | `scheduler:read/manage` |
| `tool:gen:list` | `codegen:read/manage` |
| `vuln:*` | Excluded domain; no canonical permission or API is planned in this project |

## Migration Rules

- Do not delete role grants while changing codes.
- Seed new codes with an idempotent migration.
- Copy grants from a legacy code to the canonical code only after the migration mapping is approved.
- Validate role count, permission count and representative allow/deny requests before and after migration.
- Do not restore removed compatibility routes to bridge permission-code differences; migrate the
  caller to canonical permissions.
