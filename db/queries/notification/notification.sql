-- name: CreateNotice :one
INSERT INTO notification_notices (
    id, title, content, notice_type, channels, target_type, target_ids, status, created_by
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: UpdateNotice :one
UPDATE notification_notices
SET title = $2, content = $3, notice_type = $4, channels = $5,
    target_type = $6, target_ids = $7, updated_at = now()
WHERE id = $1 AND status = 'draft'
RETURNING *;

-- name: GetNotice :one
SELECT * FROM notification_notices WHERE id = $1;

-- name: ListNotices :many
SELECT * FROM notification_notices
WHERE ($1::text = '' OR status = $1)
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountNotices :one
SELECT count(*) FROM notification_notices WHERE ($1::text = '' OR status = $1);

-- name: PublishNotice :execrows
UPDATE notification_notices
SET status = 'published', published_at = now(), updated_at = now()
WHERE id = $1 AND status = 'draft';

-- name: WithdrawNotice :execrows
UPDATE notification_notices
SET status = 'withdrawn', withdrawn_at = now(), updated_at = now()
WHERE id = $1 AND status = 'published';

-- name: AddNoticeRecipient :exec
INSERT INTO notification_recipients (notice_id, user_id)
SELECT $1, id FROM identity_users WHERE deleted_at IS NULL
ON CONFLICT DO NOTHING;

-- name: ListMyNotices :many
SELECT n.*, r.read_at
FROM notification_notices n
JOIN notification_recipients r ON r.notice_id = n.id
WHERE r.user_id = $1 AND n.status = 'published'
ORDER BY n.published_at DESC
LIMIT $2 OFFSET $3;

-- name: CountMyNotices :one
SELECT count(*)
FROM notification_notices n
JOIN notification_recipients r ON r.notice_id = n.id
WHERE r.user_id = $1 AND n.status = 'published';

-- name: GetMyNotice :one
SELECT n.*, r.read_at
FROM notification_notices n
JOIN notification_recipients r ON r.notice_id = n.id
WHERE n.id = $1 AND r.user_id = $2 AND n.status = 'published';

-- name: ListNoticeRecipientContacts :many
SELECT u.id, u.username, u.display_name, u.phone, u.email
FROM notification_recipients recipient
JOIN identity_users u ON u.id = recipient.user_id
WHERE recipient.notice_id = $1 AND u.deleted_at IS NULL AND u.status = 'active'
ORDER BY u.id;

-- name: MarkNoticeRead :execrows
UPDATE notification_recipients SET read_at = now()
WHERE notice_id = $1 AND user_id = $2 AND read_at IS NULL;

-- name: MarkAllNoticesRead :execrows
UPDATE notification_recipients SET read_at = now()
WHERE user_id = $1 AND read_at IS NULL;

-- name: CountUnreadNotices :one
SELECT count(*) FROM notification_recipients
WHERE user_id = $1 AND read_at IS NULL;

-- name: InsertNoticeSendLog :exec
INSERT INTO notification_send_logs (
    id, notice_id, channel, status, target_count, success_count, error_message
) VALUES ($1, $2, $3, $4, $5, $6, $7);
