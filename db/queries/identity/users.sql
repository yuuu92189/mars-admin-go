-- name: GetUserByUsername :one
SELECT *
FROM identity_users
WHERE username = $1 AND deleted_at IS NULL;

-- name: GetUserByPhone :one
SELECT *
FROM identity_users
WHERE phone = $1 AND deleted_at IS NULL;

-- name: GetSocialUser :one
SELECT u.*
FROM identity_social_identities social
JOIN identity_users u ON u.id = social.user_id
WHERE social.provider = $1 AND social.subject = $2 AND u.deleted_at IS NULL;

-- name: CreateSocialIdentity :exec
INSERT INTO identity_social_identities (id, user_id, provider, subject, metadata)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (provider, subject) DO NOTHING;

-- name: CreateSocialIdentityStrict :exec
INSERT INTO identity_social_identities (id, user_id, provider, subject, metadata)
VALUES ($1, $2, $3, $4, $5);

-- name: GetUserByID :one
SELECT *
FROM identity_users
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetUserByIDInDataScope :one
SELECT u.*
FROM identity_users u
WHERE u.id = $1
  AND u.deleted_at IS NULL
  AND (
    $3::text = 'all'
    OR ($3::text = 'self' AND u.id = $2)
    OR (
      $3::text = 'department'
      AND EXISTS (
        SELECT 1
        FROM organization_memberships mine
        JOIN organization_memberships target ON target.department_id = mine.department_id
        WHERE mine.user_id = $2 AND target.user_id = u.id
      )
    )
    OR (
      $3::text = 'department_tree'
      AND EXISTS (
        SELECT 1
        FROM organization_memberships mine
        JOIN organization_departments my_dept ON my_dept.id = mine.department_id
        JOIN organization_memberships target ON target.user_id = u.id
        JOIN organization_departments target_dept ON target_dept.id = target.department_id
        WHERE mine.user_id = $2
          AND target_dept.path <@ my_dept.path
      )
    )
    OR (
      $3::text = 'custom'
      AND EXISTS (
        SELECT 1
        FROM access_user_roles aur
        JOIN access_role_departments ard ON ard.role_id = aur.role_id
        JOIN organization_memberships target ON target.department_id = ard.department_id
        WHERE aur.user_id = $2 AND target.user_id = u.id
      )
    )
  );

-- name: CountDepartmentsWithinScope :one
SELECT count(DISTINCT department.id)::bigint
FROM organization_departments department
WHERE department.deleted_at IS NULL
  AND department.id = ANY($3::uuid[])
  AND (
    $2::text = 'all'
    OR (
      $2::text = 'department'
      AND EXISTS (
        SELECT 1
        FROM organization_memberships mine
        WHERE mine.user_id = $1 AND mine.department_id = department.id
      )
    )
    OR (
      $2::text = 'department_tree'
      AND EXISTS (
        SELECT 1
        FROM organization_memberships mine
        JOIN organization_departments my_dept ON my_dept.id = mine.department_id
        WHERE mine.user_id = $1 AND department.path <@ my_dept.path
      )
    )
    OR (
      $2::text = 'custom'
      AND EXISTS (
        SELECT 1
        FROM access_user_roles role
        JOIN access_role_departments allowed ON allowed.role_id = role.role_id
        WHERE role.user_id = $1 AND allowed.department_id = department.id
      )
    )
  );

-- name: CountPrivilegedRoles :one
SELECT count(*)::bigint
FROM access_roles
WHERE id = ANY($1::uuid[])
  AND is_super_admin = true
  AND deleted_at IS NULL;

-- name: ListUsers :many
SELECT u.*
FROM identity_users u
WHERE u.deleted_at IS NULL
  AND ($1::text = '' OR u.username ILIKE '%' || $1 || '%' OR u.display_name ILIKE '%' || $1 || '%')
  AND ($2::text = '' OR u.status = $2)
  AND ($3::text = '' OR u.user_type = $3)
  AND (
    $4::text = 'all'
    OR ($4::text = 'self' AND u.id = $5)
    OR (
      $4::text = 'department'
      AND EXISTS (
        SELECT 1 FROM organization_memberships mine
        JOIN organization_memberships target ON target.department_id = mine.department_id
        WHERE mine.user_id = $5 AND target.user_id = u.id
      )
    )
    OR (
      $4::text = 'department_tree'
      AND EXISTS (
        SELECT 1
        FROM organization_memberships mine
        JOIN organization_departments my_dept ON my_dept.id = mine.department_id
        JOIN organization_memberships target ON target.user_id = u.id
        JOIN organization_departments target_dept ON target_dept.id = target.department_id
        WHERE mine.user_id = $5
          AND target_dept.path <@ my_dept.path
      )
    )
    OR (
      $4::text = 'custom'
      AND EXISTS (
        SELECT 1
        FROM access_user_roles aur
        JOIN access_role_departments ard ON ard.role_id = aur.role_id
        JOIN organization_memberships target ON target.department_id = ard.department_id
        WHERE aur.user_id = $5 AND target.user_id = u.id
      )
    )
  )
ORDER BY
  CASE WHEN $6::text = 'username_asc' THEN u.username END ASC,
  CASE WHEN $6::text = 'username_desc' THEN u.username END DESC,
  CASE WHEN $6::text = 'created_desc' THEN u.created_at END DESC,
  CASE WHEN $6::text = 'created_asc' THEN u.created_at END ASC,
  u.id
LIMIT $7 OFFSET $8;

-- name: CountUsers :one
SELECT count(*)
FROM identity_users u
WHERE u.deleted_at IS NULL
  AND ($1::text = '' OR u.username ILIKE '%' || $1 || '%' OR u.display_name ILIKE '%' || $1 || '%')
  AND ($2::text = '' OR u.status = $2)
  AND ($3::text = '' OR u.user_type = $3)
  AND (
    $4::text = 'all'
    OR ($4::text = 'self' AND u.id = $5)
    OR (
      $4::text = 'department'
      AND EXISTS (
        SELECT 1 FROM organization_memberships mine
        JOIN organization_memberships target ON target.department_id = mine.department_id
        WHERE mine.user_id = $5 AND target.user_id = u.id
      )
    )
    OR (
      $4::text = 'department_tree'
      AND EXISTS (
        SELECT 1
        FROM organization_memberships mine
        JOIN organization_departments my_dept ON my_dept.id = mine.department_id
        JOIN organization_memberships target ON target.user_id = u.id
        JOIN organization_departments target_dept ON target_dept.id = target.department_id
        WHERE mine.user_id = $5
          AND target_dept.path <@ my_dept.path
      )
    )
    OR (
      $4::text = 'custom'
      AND EXISTS (
        SELECT 1
        FROM access_user_roles aur
        JOIN access_role_departments ard ON ard.role_id = aur.role_id
        JOIN organization_memberships target ON target.department_id = ard.department_id
        WHERE aur.user_id = $5 AND target.user_id = u.id
      )
    )
  );

-- name: CreateUser :one
INSERT INTO identity_users (
    id, username, password_hash, display_name, email, phone, avatar_url, user_type, status, gender
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
)
RETURNING *;

-- name: UpdateUser :one
UPDATE identity_users
SET display_name = $2,
    email = $3,
    phone = $4,
    avatar_url = $5,
    user_type = $6,
    status = $7,
    gender = $8,
    version = version + 1,
    updated_at = now()
WHERE id = $1 AND deleted_at IS NULL AND version = $9
RETURNING *;

-- name: SoftDeleteUser :execrows
UPDATE identity_users
SET deleted_at = now(), updated_at = now(), version = version + 1
WHERE id = $1 AND deleted_at IS NULL;

-- name: UpdateUserPassword :execrows
UPDATE identity_users
SET password_hash = $2, password_login_enabled = true, updated_at = now(), version = version + 1
WHERE id = $1 AND deleted_at IS NULL;

-- name: DisablePasswordLogin :exec
UPDATE identity_users
SET password_login_enabled = false, updated_at = now()
WHERE id = $1 AND deleted_at IS NULL;

-- name: RecordFailedLogin :one
UPDATE identity_users
SET failed_login_count = failed_login_count + 1,
    locked_until = CASE WHEN failed_login_count + 1 >= $2 THEN $3 ELSE locked_until END,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: ClearFailedLogin :exec
UPDATE identity_users
SET failed_login_count = 0, locked_until = NULL, last_login_at = now(), updated_at = now()
WHERE id = $1;

-- name: SetUserRoles :exec
DELETE FROM access_user_roles WHERE user_id = $1;

-- name: SetMFASecret :execrows
UPDATE identity_users
SET mfa_secret_encrypted = $2, mfa_enabled_at = NULL, updated_at = now()
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetMFASecret :one
SELECT mfa_secret_encrypted, mfa_enabled_at
FROM identity_users
WHERE id = $1 AND deleted_at IS NULL;

-- name: EnableMFA :execrows
UPDATE identity_users
SET mfa_enabled_at = now(), updated_at = now()
WHERE id = $1 AND deleted_at IS NULL AND mfa_secret_encrypted IS NOT NULL;

-- name: DisableMFA :execrows
UPDATE identity_users
SET mfa_secret_encrypted = NULL, mfa_enabled_at = NULL, updated_at = now()
WHERE id = $1 AND deleted_at IS NULL;

-- name: AddUserRole :exec
INSERT INTO access_user_roles (user_id, role_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: ListUserRoleCodes :many
SELECT r.code, r.is_super_admin, r.data_scope
FROM access_roles r
JOIN access_user_roles ur ON ur.role_id = r.id
WHERE ur.user_id = $1 AND r.status = 'active' AND r.deleted_at IS NULL
ORDER BY r.code;

-- name: GetUserAccess :many
SELECT
    r.code AS role_code,
    r.is_super_admin,
    r.data_scope,
    COALESCE(p.code, '') AS permission_code
FROM access_user_roles ur
JOIN access_roles r ON r.id = ur.role_id
LEFT JOIN access_role_permissions rp ON rp.role_id = r.id
LEFT JOIN access_permissions p ON p.id = rp.permission_id
WHERE ur.user_id = $1 AND r.status = 'active' AND r.deleted_at IS NULL
ORDER BY r.code, p.code;
