# Java Product Traceability

## Scope Correction

The Java backend and `mars-admin-master/mars-ui` are the product reference for page scope, fields,
operations, status transitions, permissions and acceptance scenarios. `scaffold-vue` is only a
design reference for the pearl-glass language, layout, components and motion. The previous
"13 Go-supported pages completed" statement is retained as historical implementation evidence,
not as product acceptance.

Evidence read for this audit:

- Java routes: `mars-admin-master/mars-ui/src/router/index.ts`.
- Java pages: `mars-admin-master/mars-ui/src/views/**/*.vue`.
- Java APIs: `mars-admin-master/mars-ui/src/api/*.ts`.
- Java controllers: `mars-admin-master/mars-api/mars-admin-api/src/main/java/com/mars/admin/controller/**/*.java`.
- Current Go modules: `mars-admin-go/internal/modules`.
- Current React routes: `mars-admin-go/frontend/src/app/**/page.tsx`.
- Design reference: `scaffold-master-main/scaffold-vue/src/styles/*.css`, `src/components/ui`,
  `src/components/layout`, `src/components/business` and `src/composables/motion.ts`.

## Page And Domain Matrix

Status meanings: `prototype` means a real API-backed page exists but has not been accepted against
the Java page; `missing` means no route or no matching vertical slice; `partial` means the backend
or page has only a subset of the Java operations.

| Java page | Java path | Java operations / data | Java backend capability | Go module and API status | Current React | Design/motion reference | Priority and acceptance |
| --- | --- | --- | --- | --- | --- | --- | --- |
| Login | `/login` | password/SMS login, captcha, remember account, error/loading | `AdminAuthController`, auth service | `identity`; `/api/v1/auth/login`, captcha, SMS exist | real login page | LoginBrandPanel, login card, GSAP entrance | P0: cookie session, CSRF, captcha, error states |
| Register | `/register` | username, password confirmation, nickname, email/phone policy, captcha, agreement | `AdminAuthController.register`, public config | `identity`; `/auth/register`, public settings and captcha consumption exist | implemented | login pearl card, field components | P0: policy-driven form, pending/audit result |
| Dashboard | `/dashboard` | greeting, clock, stats, shortcuts, changelog, system info | `DashboardController.stats` | `monitoring`; `/admin/monitoring/dashboard` partial | real but visually divergent prototype | Metric, Odometer, Panel, page entrance | P1: Java counters, real shortcuts and permission filtering |
| Profile | `/profile` | avatar upload, profile fields, save/reset | `AdminAuthController.profile`, file upload | `identity` and `storage`; `/auth/profile`, `/files` exist | implemented | Profile card, Field, Avatar, upload overlay | P0: editable profile, validation, avatar upload |
| System users | `/system/user` | search, pagination, add/edit/delete, batch, approve/reject, reset password, quit, import/export | `SysUserController`, role/post/dept APIs | `identity`/`access`/`organization`; CRUD and reset exist; import/export missing | real prototype | SearchForm, DataTable, FormDialog, bulk bar | P1: all operations and permission buttons |
| System roles | `/system/role` | search, CRUD, menu/data-scope assignment, matrix | `SysRoleController` | `access`; CRUD, menus, permissions partial | real prototype | PermissionMatrix, tree, drawer | P1: assignment semantics and effective permissions |
| System menus | `/system/menu` | tree/list, CRUD, type/path/component/icon/status | `SysMenuController` | `access`; list/create/update/delete exist | real prototype | tree/list, form dialog | P1: route and permission validation |
| System dictionaries | `/system/dict` | type CRUD, item CRUD, search, status | `SysDictTypeController`, `SysDictDataController` | `system`; CRUD exists | real prototype | DictSwatchbook, DictTag, list page | P1: type/item lifecycle and dictionary consumers |
| System config | `/system/config` | grouped settings, secrets, payment/email/SMS tests, key generation | `SysConfigGroupController` | `system`; groups/update/public exist; provider tests missing | real prototype | settings navigation, sensitive input, diff bar | P1: group coverage and explicit not-configured states |
| File list | `/system/file` | groups, upload, preview, text, rename, move, delete, download | `SysFileController`, `SysFileGroupController` | `storage`; upload/list/get/delete exist; groups/preview/rename/move partial | implemented core | file grid/table, drag upload, drawer preview | P1: add groups, preview, rename and move |
| File config | `/system/file-config` | storage providers, master config, test connection | `SysFileConfig` Java capability | `storage`; no equivalent API | missing | form cards, secret fields, test action | P2: provider settings with explicit unsupported states |
| Customers | `/system/customer` | search, CRUD, batch delete | Java `Customer` controller/API | no dedicated Go module | missing | SearchForm, DataTable | P2: decide product ownership before schema work |
| Students | `/system/student` | search, CRUD, import/export/template | Java `StudentController` | no dedicated Go module | missing | SearchForm, DataTable, upload | P2: decide product ownership before schema work |
| Notices | `/message/notice` | admin CRUD/publish/withdraw, my notices, read, channels, send logs/retry | `SysNoticeController` | `notification`; core CRUD/read/publish partial; channel logs missing | real prototype | Notice list, target picker, status tags | P1: channels and delivery records |
| Chat | `/message/chat` | private/group chat, history, unread, block, group members/roles | `SysChatController`, `ChatGroupController`, WebSocket | `messaging`; direct/group APIs and WebSocket exist | direct chat implemented | chat shell, composer, contact rail, motion | P1: add group tabs, blocklist, reconnect evidence |
| Departments | `/org/dept` | tree, CRUD, move, search/status | `SysDeptController` | `organization`; CRUD exists | real prototype | DeptTree, OrgCanvas | P1: move and scope checks |
| Positions | `/org/post` | list/tree, CRUD, move | `SysPostController` | `organization`; CRUD exists | real prototype | PostLadder, tree | P1: parent/move semantics |
| Login logs | `/log/loginlog` | page, filters, delete/clean | `SysLoginLogController` | `audit`; list exists; delete/clean partial | real prototype | audit table, risk tags | P1: retention and cleanup permissions |
| Operation logs | `/log/operlog` | page, filters, detail, delete/clean | `SysOperLogController` | `audit`; list exists | real prototype | audit table, detail drawer | P1: safe redaction and retention |
| Online users | `/monitor/online` | list, force logout | `MonitorController.online` | `identity`; session list/revoke exists under auth | missing | DataTable, status tags | P1: server-owned sessions only |
| Scheduled jobs | `/monitor/job` | CRUD, pause/resume, run, logs/statistics/clean | `SysJobController` | `scheduler`; CRUD/run/runs exist | real prototype | CronBuilder, run history, chart | P1: concurrency and run states |
| Cache | `/monitor/cache` | stats, keys, value, delete | `MonitorController.cache` | `monitoring`; no admin cache API | missing | instrument panels, key table | P2: safe read-only defaults |
| API access | `/monitor/api-access` | page, statistics, filters | `ApiAccessLogController` | `monitoring`; no API access read model | missing | trend chart, table | P1: privacy-safe request telemetry |
| Server | `/monitor/server` | CPU/memory/disk/runtime | `MonitorController.server` | `monitoring`; snapshot only | missing | gauges, server cards | P1: bounded runtime metrics |
| Server manager | `/monitor/server-manager` | server CRUD, test, connect, SSH | `SysServerController`, SSH WebSocket | no Go server registry or SSH boundary | missing | server cards, terminal drawer | P2: security review before SSH |
| Druid iframe | `/monitor/druid` | external frame | Java-only integration | not applicable | missing | iframe shell | P2: excluded unless replacement is decided |
| Workflow definitions | `/workflow/definition` | search, create, publish/unpublish, active, copy, delete | `FlowDefinitionController` | `workflow`; list/create/publish partial | missing | definition table, state tags | P1: version and idempotency |
| Workflow designer | `/workflow/designer` | visual definition editor | Java workflow model | `workflow` domain has state machine, no designer API | missing | designer canvas and inspector | P2: model contract before UI |
| Workflow todo | `/workflow/task/todo` | search, approve/reject, variables | `FlowTaskController` | `workflow`; todo/decide partial | missing | task table, approval modal | P1: assignee and data scope |
| Workflow done | `/workflow/task/done` | history, search | `FlowTaskController` | `workflow`; done partial | missing | task table | P1: visibility |
| Workflow monitor | `/workflow/monitor` | instances, start, terminate | `FlowInstanceController` | `workflow`; instances/events/terminate partial | missing | monitor table and event timeline | P1: operator visibility |
| Codegen | `/tool/gen` | DB tables, import, config, preview, generate/remove, sync | `GenController` | `codegen`; list/import/preview/download partial | real prototype | GeneratorPipeline, code viewer | P1: safe generation boundaries |
| Test menu | `/test/test` | static test route | none | excluded | missing | none | Not product scope |
| Common iframe | `/common/iframe` | external frame shell | none | excluded | missing | iframe shell | Not product scope |
| Redirect | `/redirect/:path` | tab refresh intermediary | frontend-only | not needed | missing | none | Not product scope |
| 404 | fallback | not found | none | route exists | real 404 | NotFound | P3: preserve shell behavior |

## Current Gap Summary

### P0 Foundation

- Registration, editable profile, avatar upload, profile shell entry and nested menu handling are
  now implemented and covered by Go/frontend tests.
- Add notifications to the shell user menu and external-frame handling to navigation.

### P1 Vertical Slices

- File list core and direct chat are implemented; group UI, file groups, preview/rename/move and
  explicit WebSocket reconnect evidence remain.
- Online users should be built on server-owned sessions; cache/API/server pages require new bounded
  monitoring read models, not Java `/monitor/*` routes.
- Workflow definition/todo/done/monitor need versioned OpenAPI schemas and visibility tests.
- User import/export/template, notice channel logs/retry and codegen write actions are not complete
  until their Go contracts and tests exist.

### Explicitly Not Target Scope

- Java `/api` envelope, Java DTO field names, Java route aliases, Java token branches and Java
  permission strings are not contracts for React.
- `scaffold-vue` asset/vulnerability/repair pages are design-only references.
- Druid, external iframe and test-menu pages are not automatically product requirements; record a
  decision before implementing them.

## Implementation Order

1. P0 authentication, registration, profile, shell navigation and permission-aware actions.
2. P1 files, notices, chat and monitoring read models.
3. P1 workflow definition/tasks/monitor and user/role/permission completion.
4. P2 file configuration, customers/students, cache/server manager and designer.
5. Remove online compatibility routes only after callers, OpenAPI and tests prove replacement.
