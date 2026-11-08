-- name: InsertOutboxEvent :exec
INSERT INTO outbox_events (
    id, aggregate_type, aggregate_id, event_type, deduplication_key,
    payload, max_attempts, available_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
)
ON CONFLICT (aggregate_type, aggregate_id, event_type, deduplication_key)
DO NOTHING;

-- name: ClaimOutboxEvents :many
WITH candidates AS (
    SELECT event.id
    FROM outbox_events AS event
    WHERE (
        event.status IN ('pending', 'failed')
        AND event.available_at <= now()
    )
    OR (
        event.status = 'processing'
        AND event.locked_at < now() - ($2::bigint * interval '1 millisecond')
    )
    ORDER BY event.available_at, event.created_at
    LIMIT $1
    FOR UPDATE SKIP LOCKED
)
UPDATE outbox_events AS event
SET status = 'processing',
    attempts = event.attempts + 1,
    locked_at = now(),
    last_error = ''
FROM candidates
WHERE event.id = candidates.id
RETURNING event.*;

-- name: MarkOutboxPublished :execrows
UPDATE outbox_events
SET status = 'published',
    published_at = now(),
    locked_at = NULL,
    last_error = ''
WHERE id = $1 AND status = 'processing';

-- name: MarkOutboxFailed :execrows
UPDATE outbox_events
SET status = CASE WHEN attempts >= max_attempts THEN 'dead' ELSE 'failed' END,
    available_at = CASE
        WHEN attempts >= max_attempts THEN available_at
        ELSE now() + make_interval(
            secs => LEAST($3::double precision * power(2, attempts), $4::double precision)
        )
    END,
    locked_at = NULL,
    last_error = left($2::text, 4000)
WHERE id = $1 AND status = 'processing';

-- name: RequeueOutboxEvent :execrows
UPDATE outbox_events
SET status = 'pending',
    attempts = 0,
    available_at = now(),
    locked_at = NULL,
    published_at = NULL,
    last_error = ''
WHERE id = $1 AND status IN ('failed', 'dead');

-- name: GetOutboxEvent :one
SELECT * FROM outbox_events WHERE id = $1;

-- name: ListOutboxEvents :many
SELECT *
FROM outbox_events
WHERE ($1::text = '' OR status = $1)
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: DeletePublishedOutboxEvents :execrows
DELETE FROM outbox_events
WHERE status = 'published'
  AND published_at < now() - ($1::bigint * interval '1 millisecond');
