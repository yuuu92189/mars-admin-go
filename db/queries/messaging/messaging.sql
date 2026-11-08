-- name: GetOrCreateDirectConversation :one
SELECT id FROM messaging_groups WHERE id = $1;

-- name: InsertDirectMessage :one
INSERT INTO messaging_messages (
    id, client_message_id, sender_id, receiver_id, message_type, content, metadata
) VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: CountBlockedConversation :one
SELECT count(*) FROM messaging_user_blocks
WHERE (user_id = $1 AND blocked_user_id = $2)
   OR (user_id = $2 AND blocked_user_id = $1);

-- name: BlockUser :exec
INSERT INTO messaging_user_blocks (user_id, blocked_user_id)
VALUES ($1, $2) ON CONFLICT DO NOTHING;

-- name: UnblockUser :execrows
DELETE FROM messaging_user_blocks
WHERE user_id = $1 AND blocked_user_id = $2;

-- name: ListBlockedUsers :many
SELECT u.id, u.username, u.display_name, u.avatar_url
FROM messaging_user_blocks b
JOIN identity_users u ON u.id = b.blocked_user_id
WHERE b.user_id = $1 AND u.deleted_at IS NULL
ORDER BY u.display_name;

-- name: InsertGroupMessage :one
INSERT INTO messaging_messages (
    id, client_message_id, sender_id, group_id, message_type, content, metadata
) VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetMessage :one
SELECT * FROM messaging_messages WHERE id = $1;

-- name: ListDirectMessages :many
SELECT * FROM messaging_messages
WHERE (
    (sender_id = $1 AND receiver_id = $2)
    OR (sender_id = $2 AND receiver_id = $1)
)
AND ($3::bigint = 0 OR sequence_number < $3)
ORDER BY sequence_number DESC
LIMIT $4;

-- name: ListGroupMessages :many
SELECT * FROM messaging_messages
WHERE group_id = $1
  AND ($2::bigint = 0 OR sequence_number < $2)
ORDER BY sequence_number DESC
LIMIT $3;

-- name: MarkDirectMessagesRead :exec
INSERT INTO messaging_read_receipts (message_id, user_id)
SELECT id, $1
FROM messaging_messages
WHERE sender_id = $2 AND receiver_id = $1 AND created_at <= $3
ON CONFLICT DO NOTHING;

-- name: CountDirectUnread :one
SELECT count(*)
FROM messaging_messages m
WHERE m.receiver_id = $1
  AND NOT EXISTS (
    SELECT 1 FROM messaging_read_receipts r
    WHERE r.message_id = m.id AND r.user_id = $1
  );

-- name: CreateGroup :one
INSERT INTO messaging_groups (id, name, avatar_url, owner_id)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetGroup :one
SELECT * FROM messaging_groups WHERE id = $1 AND status = 'active';

-- name: ListGroupsForUser :many
SELECT g.*
FROM messaging_groups g
JOIN messaging_group_members m ON m.group_id = g.id
WHERE m.user_id = $1 AND m.left_at IS NULL AND g.status = 'active'
ORDER BY g.updated_at DESC;

-- name: UpdateGroup :one
UPDATE messaging_groups
SET name = $2, avatar_url = $3, updated_at = now()
WHERE id = $1 AND status = 'active'
RETURNING *;

-- name: AddGroupMember :exec
INSERT INTO messaging_group_members (group_id, user_id, member_role)
VALUES ($1, $2, $3)
ON CONFLICT (group_id, user_id) DO UPDATE
SET member_role = CASE
        WHEN messaging_group_members.member_role = 'owner' THEN 'owner'
        ELSE excluded.member_role
    END,
    left_at = NULL;

-- name: RemoveGroupMember :execrows
UPDATE messaging_group_members SET left_at = now()
WHERE group_id = $1 AND user_id = $2 AND member_role <> 'owner' AND left_at IS NULL;

-- name: ListGroupMembers :many
SELECT gm.*, u.username, u.display_name, u.avatar_url
FROM messaging_group_members gm
JOIN identity_users u ON u.id = gm.user_id
WHERE gm.group_id = $1 AND gm.left_at IS NULL
ORDER BY CASE gm.member_role WHEN 'owner' THEN 0 WHEN 'admin' THEN 1 ELSE 2 END, u.display_name;

-- name: UpdateGroupMemberRole :execrows
UPDATE messaging_group_members AS member
SET member_role = $3
WHERE member.group_id = $1 AND member.user_id = $2 AND member.left_at IS NULL
  AND member.member_role <> 'owner';

-- name: SetGroupMemberMuted :execrows
UPDATE messaging_group_members
SET muted_until = $3
WHERE group_id = $1 AND user_id = $2 AND left_at IS NULL;

-- name: QuitGroup :execrows
UPDATE messaging_group_members
SET left_at = now()
WHERE group_id = $1 AND user_id = $2 AND member_role <> 'owner' AND left_at IS NULL;

-- name: TransferGroupOwner :exec
UPDATE messaging_groups SET owner_id = $2, updated_at = now() WHERE id = $1 AND status = 'active';

-- name: TransferGroupMemberRoles :exec
UPDATE messaging_group_members
SET member_role = CASE
    WHEN user_id = $2 THEN 'owner'
    WHEN user_id = $3 THEN 'admin'
    ELSE member_role
END
WHERE group_id = $1 AND user_id IN ($2, $3);

-- name: IsGroupMember :one
SELECT EXISTS (
    SELECT 1 FROM messaging_group_members
    WHERE group_id = $1 AND user_id = $2 AND left_at IS NULL
);

-- name: DissolveGroup :execrows
UPDATE messaging_groups
SET status = 'dissolved', dissolved_at = now(), updated_at = now()
WHERE id = $1 AND owner_id = $2 AND status = 'active';

-- name: ListChatUsers :many
SELECT u.id, u.username, u.display_name, u.avatar_url
FROM identity_users u
WHERE u.deleted_at IS NULL AND u.status = 'active' AND u.id <> $1
  AND ($2::text = '' OR u.username ILIKE '%' || $2 || '%' OR u.display_name ILIKE '%' || $2 || '%')
ORDER BY u.display_name
LIMIT $3;
