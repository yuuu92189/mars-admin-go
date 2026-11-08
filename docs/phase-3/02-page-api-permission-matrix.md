# Page API and Permission Matrix

> Historical Phase 3 record. Asset Management, Vulnerability Management and Remediation Work
> Orders are excluded by the Phase 4 business-scope decision, not blocked work.

| Page | React route | API operations | Required permission | Backend enforcement | Status |
| --- | --- | --- | --- | --- | --- |
| Dashboard | `/dashboard` | `monitoringDashboard`, `listAuditEvents`, `listLoginEvents` | `audit:read` | Route middleware | Implemented |
| Users | `/system/user` | `listUsers`, `getUser`, `createUser`, `updateUser`, `deleteUser`, `resetUserPassword`, `revokeUserSessions` | `user:read/create/update/delete` | Route middleware | Implemented |
| Roles | `/system/role` | `listRoles`, `createRole`, `updateRole`, `deleteRole`, `listMenus`, `listPermissions`, `getRoleMenus`, `setRoleMenus`, `setRolePermissions` | `role:read/create/assign` | Route middleware | Implemented |
| Menus | `/system/menu` | `listMenus`, `createMenu`, `updateMenu`, `deleteMenu` | `role:read/assign` | Route middleware | Implemented |
| Departments | `/system/dept` | `listDepartments`, `createDepartment`, `updateDepartment`, `deleteDepartment` | `department:read/manage` | Route middleware | Implemented |
| Positions | `/system/post` | `listPositions`, `createPosition`, `updatePosition`, `deletePosition` | `department:read/manage` | Route middleware | Implemented |
| Dictionaries | `/system/dict` | `listDictionaries`, `createDictionary`, `updateDictionary`, `deleteDictionary`, `listDictionaryItems`, `createDictionaryItem`, `updateDictionaryItem`, `deleteDictionaryItem` | `dictionary:read/manage` | Route middleware | Implemented |
| Settings | `/system/config` | `listSettingGroups`, `getSettingGroup`, `updateSettingGroup` | `settings:read/manage` | Route middleware | Implemented |
| Notices | `/system/notice` | `listNotices`, `listMyNotices`, `getNotice`, `createNotice`, `updateNotice`, `deleteNotice`, `publishNotice`, `withdrawNotice`, `readNotice`, `noticeUnreadCounts` | `notice:read/manage` | Route middleware | Implemented |
| Scheduled Jobs | `/monitor/job` | `listScheduledJobs`, `createScheduledJob`, `updateScheduledJob`, `deleteScheduledJob`, pause, resume, run, runs | `scheduler:read/manage` | Route middleware | Implemented |
| Operation Logs | `/monitor/operlog` | `listAuditEvents` | `audit:read` | Route middleware | Implemented |
| Login Logs | `/monitor/loginlog` | `listLoginEvents` | `audit:read` | Route middleware | Implemented |
| Codegen | `/tool/generator` | `listDatabaseTables`, `listImportedCodegenTables`, `importCodegenTables`, `getImportedCodegenTable`, `updateImportedCodegenTable`, `deleteImportedCodegenTable`, `previewImportedCodegenTable`, `syncImportedCodegenTable`, `downloadGeneratedCode` | `codegen:read/manage` | Route middleware | Implemented |
| Assets | Not created | No canonical API | Not applicable | Not applicable | EXCLUDED |
| Vulnerabilities | Not created | No canonical API | Not applicable | Not applicable | EXCLUDED |
| Repair Tasks | Not created | No canonical API | Not applicable | Not applicable | EXCLUDED |

## Permission Rules

- Server page wrappers call `requirePageAccess` before rendering client views.
- The server wrapper checks menu visibility and at least one page-level permission.
- Action buttons use `hasPermission` for UX only.
- Every API operation remains protected by Go middleware.
- `isSuperAdmin` bypasses frontend permission checks only; Go remains authoritative.

## Route Alignment

Migration `000024_phase3_business_routes.sql` aligns seeded menu paths with the Vue-derived target routes:

- `org/dept` to `system/dept`
- `org/post` to `system/post`
- `message/notice` to `system/notice`
- `log/loginlog` to `monitor/loginlog`
- `log/operlog` to `monitor/operlog`
- `tool/gen` to `tool/generator`
