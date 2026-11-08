# System Management Migration

## Users

- Department tree filtering backed by `departmentId`.
- Table and board views.
- Create/edit/delete, role/department/position assignment.
- Status changes, batch actions, session revocation.
- Password reset returns a one-time temporary password and never stores it.
- User detail drawer and permission trace link.

## Roles

- Role CRUD and data scope.
- Permission matrix with role-by-menu cells.
- Permission tree drawer and real menu/permission writes.
- Permission trace view using `?trace=user:<uuid>` or `?trace=role:<uuid>`.

## Menus

- Tree table and navigation star map.
- Parent, type, route, component metadata, permission code, icon, sort, visibility.
- Route values are data only and are never executed as arbitrary component paths.

## Organization

- Department tree table and pannable/zoomable organization canvas.
- Department CRUD, leader/contact fields, member counts and member drawer.
- Position table and post ladder, CRUD and member preview.

## Dictionaries and Settings

- Dictionary type swatchbook/table and item drawer.
- Default item, status, sort, and tone.
- Settings group navigation, inline controls, sensitive masking, diff bar, and reset.
- Empty sensitive values preserve the encrypted server-side value.

## Notices

- My notices and sent records.
- Unread count, read receipt, detail drawer.
- User/department targeting, publish, withdraw, and canonical delete.
