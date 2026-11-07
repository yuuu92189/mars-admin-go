-- name: CreateFile :one
INSERT INTO storage_files (
    id, group_id, owner_user_id, original_name, storage_key, storage_provider,
    content_type, size_bytes, sha256_hash, visibility
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: GetFile :one
SELECT * FROM storage_files WHERE id = $1 AND deleted_at IS NULL;

-- name: ListFiles :many
SELECT * FROM storage_files
WHERE deleted_at IS NULL
  AND ($1::uuid IS NULL OR group_id = $1)
  AND ($2::text = '' OR original_name ILIKE '%' || $2 || '%')
  AND ($3::boolean OR owner_user_id = $4)
ORDER BY created_at DESC
LIMIT $5 OFFSET $6;

-- name: CountFiles :one
SELECT count(*) FROM storage_files
WHERE deleted_at IS NULL
  AND ($1::uuid IS NULL OR group_id = $1)
  AND ($2::text = '' OR original_name ILIKE '%' || $2 || '%')
  AND ($3::boolean OR owner_user_id = $4);

-- name: SoftDeleteFile :execrows
UPDATE storage_files SET status = 'deleted', deleted_at = now(), updated_at = now()
WHERE id = $1 AND deleted_at IS NULL;
