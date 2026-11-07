-- name: ListAuditEvents :many
SELECT * FROM audit_events
WHERE ($1::text = '' OR action = $1)
  AND ($2::text = '' OR resource_type = $2)
  AND ($3::timestamptz IS NULL OR occurred_at >= $3)
  AND ($4::timestamptz IS NULL OR occurred_at < $4)
ORDER BY occurred_at DESC
LIMIT $5 OFFSET $6;

-- name: CountAuditEvents :one
SELECT count(*) FROM audit_events
WHERE ($1::text = '' OR action = $1)
  AND ($2::text = '' OR resource_type = $2)
  AND ($3::timestamptz IS NULL OR occurred_at >= $3)
  AND ($4::timestamptz IS NULL OR occurred_at < $4);

-- name: InsertAuditEvent :exec
INSERT INTO audit_events (
    id, occurred_at, actor_user_id, actor_name, action, resource_type,
    resource_id, outcome, request_id, trace_id, ip_address, user_agent, metadata
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13);

-- name: ListLoginEvents :many
SELECT * FROM audit_login_events
ORDER BY occurred_at DESC
LIMIT $1 OFFSET $2;

-- name: CountLoginEvents :one
SELECT count(*) FROM audit_login_events;
