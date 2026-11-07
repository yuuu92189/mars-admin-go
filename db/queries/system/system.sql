-- name: ListDictionaryTypes :many
SELECT t.*,
       (SELECT count(*) FROM system_dictionary_items i WHERE i.type_id = t.id)::bigint AS item_count
FROM system_dictionary_types t
ORDER BY t.code;

-- name: GetDictionaryType :one
SELECT * FROM system_dictionary_types WHERE id = $1;

-- name: CreateDictionaryType :one
INSERT INTO system_dictionary_types (id, code, name, description, status)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: UpdateDictionaryType :one
UPDATE system_dictionary_types
SET code = $2, name = $3, description = $4, status = $5, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteDictionaryType :execrows
DELETE FROM system_dictionary_types WHERE id = $1;

-- name: ListDictionaryItems :many
SELECT * FROM system_dictionary_items
WHERE type_id = $1
ORDER BY sort_order, label;

-- name: CreateDictionaryItem :one
INSERT INTO system_dictionary_items (
    id, type_id, value, label, sort_order, tag_style, is_default, status
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: UpdateDictionaryItem :one
UPDATE system_dictionary_items
SET value = $2, label = $3, sort_order = $4, tag_style = $5,
    is_default = $6, status = $7, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteDictionaryItem :execrows
DELETE FROM system_dictionary_items WHERE id = $1;

-- name: ListSettingGroups :many
SELECT g.*,
       (SELECT count(*) FROM system_settings s WHERE s.group_id = g.id)::bigint AS setting_count
FROM system_setting_groups g
ORDER BY g.sort_order, g.code;

-- name: GetSettingGroupByCode :one
SELECT * FROM system_setting_groups WHERE code = $1;

-- name: ListSettingsByGroup :many
SELECT * FROM system_settings WHERE group_id = $1 ORDER BY key;

-- name: GetSetting :one
SELECT * FROM system_settings WHERE id = $1;

-- name: GetSettingByKey :one
SELECT s.*
FROM system_settings s
JOIN system_setting_groups g ON g.id = s.group_id
WHERE g.code = $1 AND s.key = $2;

-- name: UpsertSetting :one
INSERT INTO system_settings (
    id, group_id, key, value_type, value_text, is_sensitive, validation, version
) VALUES ($1, $2, $3, $4, $5, $6, $7, 1)
ON CONFLICT (group_id, key) DO UPDATE
SET value_type = excluded.value_type,
    value_text = excluded.value_text,
    is_sensitive = excluded.is_sensitive,
    validation = excluded.validation,
    version = system_settings.version + 1,
    updated_at = now()
RETURNING *;

-- name: RecordSettingRevision :exec
INSERT INTO system_setting_revisions (
    id, setting_id, old_value, new_value, changed_by
) VALUES ($1, $2, $3, $4, $5);

