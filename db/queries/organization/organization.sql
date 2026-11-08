-- name: ListDepartments :many
WITH RECURSIVE tree AS (
    SELECT d.*, 0 AS depth
    FROM organization_departments d
    WHERE d.parent_id IS NULL AND d.deleted_at IS NULL
    UNION ALL
    SELECT child.*, tree.depth + 1
    FROM organization_departments child
    JOIN tree ON child.parent_id = tree.id
    WHERE child.deleted_at IS NULL AND tree.depth < 50
)
SELECT * FROM tree AS department
WHERE (
    $2::text = 'all'
    OR (
        $2::text = 'self'
        AND EXISTS (
            SELECT 1 FROM organization_memberships mine
            WHERE mine.user_id = $1 AND mine.department_id = department.id
        )
    )
    OR (
        $2::text = 'department'
        AND EXISTS (
            SELECT 1 FROM organization_memberships mine
            WHERE mine.user_id = $1 AND mine.department_id = department.id
        )
    )
    OR (
        $2::text = 'department_tree'
        AND EXISTS (
            SELECT 1
            FROM organization_memberships mine
            JOIN organization_departments mine_department ON mine_department.id = mine.department_id
            WHERE mine.user_id = $1 AND department.path <@ mine_department.path
        )
    )
    OR (
        $2::text = 'custom'
        AND EXISTS (
            SELECT 1
            FROM access_user_roles role_user
            JOIN access_role_departments allowed ON allowed.role_id = role_user.role_id
            WHERE role_user.user_id = $1 AND allowed.department_id = department.id
        )
    )
)
ORDER BY depth, sort_order, name;

-- name: GetDepartment :one
SELECT department.*
FROM organization_departments AS department
WHERE department.id = $1
  AND department.deleted_at IS NULL
  AND (
      $3::text = 'all'
      OR (
          $3::text = 'self'
          AND EXISTS (
              SELECT 1 FROM organization_memberships mine
              WHERE mine.user_id = $2 AND mine.department_id = department.id
          )
      )
      OR (
          $3::text = 'department'
          AND EXISTS (
              SELECT 1 FROM organization_memberships mine
              WHERE mine.user_id = $2 AND mine.department_id = department.id
          )
      )
      OR (
          $3::text = 'department_tree'
          AND EXISTS (
              SELECT 1
              FROM organization_memberships mine
              JOIN organization_departments mine_department ON mine_department.id = mine.department_id
              WHERE mine.user_id = $2 AND department.path <@ mine_department.path
          )
      )
      OR (
          $3::text = 'custom'
          AND EXISTS (
              SELECT 1
              FROM access_user_roles role_user
              JOIN access_role_departments allowed ON allowed.role_id = role_user.role_id
              WHERE role_user.user_id = $2 AND allowed.department_id = department.id
          )
      )
  );

-- name: CreateDepartment :one
INSERT INTO organization_departments (
    id, parent_id, name, code, path, leader_name, phone, email, sort_order, status
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: UpdateDepartment :one
UPDATE organization_departments
SET parent_id = $2, name = $3, code = $4, path = $5, leader_name = $6,
    phone = $7, email = $8, sort_order = $9, status = $10, updated_at = now()
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteDepartment :execrows
UPDATE organization_departments AS department
SET deleted_at = now(), updated_at = now()
WHERE department.id = $1 AND department.deleted_at IS NULL
  AND NOT EXISTS (
    SELECT 1 FROM organization_departments child
    WHERE child.parent_id = department.id AND child.deleted_at IS NULL
  );

-- name: ListPositions :many
SELECT * FROM organization_positions
WHERE deleted_at IS NULL
ORDER BY sort_order, name;

-- name: GetPosition :one
SELECT * FROM organization_positions WHERE id = $1 AND deleted_at IS NULL;

-- name: CreatePosition :one
INSERT INTO organization_positions (
    id, parent_id, code, name, sort_order, status, description
) VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: UpdatePosition :one
UPDATE organization_positions
SET parent_id = $2, code = $3, name = $4, sort_order = $5,
    status = $6, description = $7, updated_at = now()
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeletePosition :execrows
UPDATE organization_positions
SET deleted_at = now(), updated_at = now()
WHERE id = $1 AND deleted_at IS NULL;

-- name: SetUserDepartments :exec
DELETE FROM organization_memberships WHERE user_id = $1;

-- name: AddUserDepartment :exec
INSERT INTO organization_memberships (user_id, department_id, is_primary)
VALUES ($1, $2, $3)
ON CONFLICT (user_id, department_id) DO UPDATE SET is_primary = excluded.is_primary;

-- name: SetUserPositions :exec
DELETE FROM organization_user_positions WHERE user_id = $1;

-- name: AddUserPosition :exec
INSERT INTO organization_user_positions (user_id, position_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: ListUserDepartments :many
SELECT d.*
FROM organization_departments d
JOIN organization_memberships m ON m.department_id = d.id
WHERE m.user_id = $1 AND d.deleted_at IS NULL
ORDER BY m.is_primary DESC, d.name;

-- name: ListUserPositions :many
SELECT p.*
FROM organization_positions p
JOIN organization_user_positions up ON up.position_id = p.id
WHERE up.user_id = $1 AND p.deleted_at IS NULL
ORDER BY p.name;
