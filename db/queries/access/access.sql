-- name: ListMenusForUser :many
SELECT DISTINCT m.*
FROM access_menus m
JOIN access_role_menus rm ON rm.menu_id = m.id
JOIN access_user_roles ur ON ur.role_id = rm.role_id
WHERE ur.user_id = $1 AND m.status = 'active' AND m.visible = true
ORDER BY m.sort_order, m.name;

-- name: ListAllMenus :many
SELECT * FROM access_menus
WHERE status = 'active' AND visible = true
ORDER BY sort_order, name;

-- name: ListRoles :many
SELECT r.*,
       (SELECT count(*) FROM access_user_roles ur WHERE ur.role_id = r.id)::bigint AS user_count
FROM access_roles r
WHERE r.deleted_at IS NULL
  AND ($1::text = '' OR r.name ILIKE '%' || $1 || '%' OR r.code ILIKE '%' || $1 || '%')
ORDER BY r.is_super_admin DESC, r.code
LIMIT $2 OFFSET $3;

-- name: CountRoles :one
SELECT count(*) FROM access_roles r
WHERE r.deleted_at IS NULL
  AND ($1::text = '' OR r.name ILIKE '%' || $1 || '%' OR r.code ILIKE '%' || $1 || '%');

-- name: GetRole :one
SELECT * FROM access_roles WHERE id = $1 AND deleted_at IS NULL;

-- name: CreateRole :one
INSERT INTO access_roles (id, name, code, description, data_scope)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: UpdateRole :one
UPDATE access_roles
SET name = $2, description = $3, data_scope = $4, status = $5,
    version = version + 1, updated_at = now()
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteRole :execrows
UPDATE access_roles SET deleted_at = now(), updated_at = now()
WHERE id = $1 AND is_super_admin = false AND deleted_at IS NULL;

-- name: SetRolePermissions :exec
DELETE FROM access_role_permissions WHERE role_id = $1;

-- name: AddRolePermission :execrows
INSERT INTO access_role_permissions (role_id, permission_id)
SELECT $1, id FROM access_permissions WHERE code = $2
ON CONFLICT DO NOTHING;

-- name: ListPermissions :many
SELECT * FROM access_permissions ORDER BY module, code;

-- name: ListRoleMenus :many
SELECT menu_id FROM access_role_menus WHERE role_id = $1;

-- name: SetRoleMenus :exec
DELETE FROM access_role_menus WHERE role_id = $1;

-- name: AddRoleMenu :exec
INSERT INTO access_role_menus (role_id, menu_id) VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: CreateMenu :one
INSERT INTO access_menus (
    id, parent_id, name, menu_type, route_path, component, permission_code,
    icon, sort_order, visible, external_url, status
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
RETURNING *;

-- name: UpdateMenu :one
UPDATE access_menus
SET parent_id = $2, name = $3, menu_type = $4, route_path = $5,
    component = $6, permission_code = $7, icon = $8, sort_order = $9,
    visible = $10, external_url = $11, status = $12, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: SoftDeleteMenu :execrows
DELETE FROM access_menus parent WHERE parent.id = $1
AND NOT EXISTS (SELECT 1 FROM access_menus child WHERE child.parent_id = parent.id);
