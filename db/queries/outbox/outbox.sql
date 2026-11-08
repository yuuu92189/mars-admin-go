-- name: InsertOutboxEvent :exec
INSERT INTO outbox_events (
    id, aggregate_type, aggregate_id, event_type, deduplication_key,
    payload, max_attempts, available_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, COALESCE(sqlc.arg(available_at)::timestamptz, now())
)
ON CONFLICT (aggregate_type, aggregate_id, event_type, deduplication_key)
DO NOTHING;

-- name: ClaimOutboxEvents :many
WITH exhausted AS (
    UPDATE outbox_events AS event
    SET status = 'dead',
        locked_at = NULL,
        lease_token = NULL,
        lease_expires_at = NULL,
        execution_deadline = NULL,
        last_error = left(
            CASE
                WHEN event.last_error = '' THEN 'maximum attempts reached'
                ELSE event.last_error || E'\nmaximum attempts reached'
            END,
            4000
        )
    WHERE event.attempts >= event.max_attempts
      AND (
          (
              event.status IN ('pending', 'failed')
              AND event.available_at <= now()
          )
          OR (
              event.status = 'processing'
              AND COALESCE(
                  event.lease_expires_at,
                  event.locked_at + ($2::bigint * interval '1 millisecond')
              ) <= now()
          )
      )
    RETURNING event.id
),
candidates AS (
    SELECT event.id
    FROM outbox_events AS event
    WHERE event.attempts < event.max_attempts
      AND NOT EXISTS (
          SELECT 1
          FROM exhausted
          WHERE exhausted.id = event.id
      )
      AND (
          (
              event.status IN ('pending', 'failed')
              AND event.available_at <= now()
          )
          OR (
              event.status = 'processing'
              AND COALESCE(
                  event.lease_expires_at,
                  event.locked_at + ($2::bigint * interval '1 millisecond')
              ) <= now()
          )
      )
    ORDER BY event.available_at, event.created_at
    LIMIT $1
    FOR UPDATE SKIP LOCKED
)
UPDATE outbox_events AS event
SET status = 'processing',
    attempts = event.attempts + 1,
    locked_at = now(),
    lease_token = gen_random_uuid(),
    lease_generation = event.lease_generation + 1,
    lease_expires_at = LEAST(
        now() + ($2::bigint * interval '1 millisecond'),
        now() + ($3::bigint * interval '1 millisecond')
    ),
    execution_deadline = now() + ($3::bigint * interval '1 millisecond'),
    last_error = event.last_error
FROM candidates
WHERE event.id = candidates.id
RETURNING event.*;

-- name: MarkOutboxPublished :execrows
UPDATE outbox_events
SET status = 'published',
    published_at = now(),
    locked_at = NULL,
    lease_token = NULL,
    lease_expires_at = NULL,
    execution_deadline = NULL
WHERE id = $1
  AND lease_token = $2
  AND lease_generation = $3
  AND status = 'processing'
  AND lease_expires_at > now();

-- name: MarkOutboxFailed :execrows
UPDATE outbox_events
SET status = CASE WHEN attempts >= max_attempts THEN 'dead' ELSE 'failed' END,
    available_at = CASE
        WHEN attempts >= max_attempts THEN available_at
        ELSE now() + make_interval(
            secs => LEAST($5::double precision * power(2, attempts), $6::double precision)
        )
    END,
    locked_at = NULL,
    lease_token = NULL,
    lease_expires_at = NULL,
    execution_deadline = NULL,
    last_error = left($4::text, 4000)
WHERE id = $1
  AND lease_token = $2
  AND lease_generation = $3
  AND status = 'processing'
  AND lease_expires_at > now();

-- name: RenewOutboxLease :one
UPDATE outbox_events
SET locked_at = now(),
    lease_expires_at = LEAST(
        now() + ($4::bigint * interval '1 millisecond'),
        execution_deadline
    )
WHERE id = $1
  AND lease_token = $2
  AND lease_generation = $3
  AND status = 'processing'
  AND lease_expires_at > now()
  AND execution_deadline > now()
RETURNING lease_expires_at, execution_deadline;

-- name: RequeueOutboxEvent :execrows
UPDATE outbox_events
SET status = 'pending',
    attempts = 0,
    available_at = now(),
    locked_at = NULL,
    lease_token = NULL,
    lease_expires_at = NULL,
    execution_deadline = NULL,
    published_at = NULL,
    last_error = last_error
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
