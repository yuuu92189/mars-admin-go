# Feature Inventory And Mapping

Status terms: `implemented` means a Go code path exists; `verified` additionally means automated checks passed in this environment; `external` requires third-party credentials; `gap` is explicitly not complete.

| Legacy capability | Go module | New API family | PostgreSQL source | Test coverage | Status |
|---|---|---|---|---|---|
| Username/password login | identity | `/api/v1/auth/login` | `identity_users`, `identity_sessions` | password and session unit tests | implemented |
| SMS login | identity + integrations | `/api/v1/auth/sms/*` | one-time codes, sessions | provider/unit tests | external |
| WeChat/MP/mini program/Apple/Alipay login | identity + integrations | `/api/v1/auth/social/*` | social identities, sessions | provider contract tests | external |
| Refresh rotation, logout, force logout | identity | `/api/v1/auth/refresh`, sessions | sessions | rotation and revocation tests | implemented |
| Users, status, reset password | identity | `/api/v1/admin/users` | users, roles | repository integration | implemented |
| Roles, permissions, menus | access | `/api/v1/admin/access/*` | access tables | RBAC/data-scope tests | implemented |
| Department tree and positions | organization | `/api/v1/admin/organization/*` | recursive CTE over departments | tree tests | implemented |
| Data scope filters | access + organization | enforced on user lists | role scope and role departments | unit/integration | implemented |
| Dictionaries | system | `/api/v1/admin/dictionaries` | dictionary tables | CRUD integration | implemented |
| Settings and public config | system | `/api/v1/admin/settings` | setting groups/items/revisions | validation/secret tests | implemented |
| Login and operation audit | audit | `/api/v1/admin/audit/events` | audit tables | middleware tests | implemented |
| Runtime/DB/Redis metrics | monitoring | `/metrics` | pgx/Redis/runtime | smoke test | implemented |
| Local file storage | storage | `/api/v1/files` | storage files | traversal/content tests | implemented |
| S3-compatible/MinIO/Aliyun OSS | storage + integrations | same file API | provider config | provider contract | external |
| Direct chat and history | messaging | `/api/v1/messages` | conversations/messages | cursor/idempotency tests | implemented |
| Group management and group chat | messaging | `/api/v1/groups` | groups/members/messages | membership tests | implemented |
| WebSocket push and Redis fan-out | messaging | `/ws/message` | messages + Redis Pub/Sub | websocket tests | implemented |
| Notices and recipients | notification | `/api/v1/notices` | notices/recipients/logs | publish/read tests | implemented |
| Cron/delayed/immediate jobs | scheduler + River | `/api/v1/admin/scheduler/jobs` | scheduler tables + River | worker tests | implemented |
| Workflow definition and approval | workflow | `/api/v1/workflows` | definitions/instances/tasks/events | state-machine tests | implemented |
| PostgreSQL schema introspection/codegen | codegen | `/api/v1/admin/codegen` | information_schema | generator tests | implemented |
| SMS/payment/push providers | integrations | provider registry | encrypted settings | contract tests | external |
| Legacy Vue/UniApp routes | compatibility adapter | `/api/*` | shared services | contract tests | implemented |
| Legacy student/customer samples | not a target domain | none | not migrated | migration decision | intentional exclusion |
| SSH web terminal | not retained | none | none | security review | intentional exclusion |

## Acceptance Rule

A row is not marked `verified` until its checks are observed in the execution report. External providers are not reported as successful without credentials; they must return a typed not-configured error.
