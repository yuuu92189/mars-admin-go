-- +goose Up
ALTER TABLE outbox_events
    ADD COLUMN lease_token UUID,
    ADD COLUMN lease_generation BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN lease_expires_at TIMESTAMPTZ,
    ADD COLUMN execution_deadline TIMESTAMPTZ;

ALTER TABLE outbox_events
    ADD CONSTRAINT outbox_events_lease_generation_check
        CHECK (lease_generation >= 0),
    ADD CONSTRAINT outbox_events_lease_deadline_check
        CHECK (
            lease_expires_at IS NULL
            OR execution_deadline IS NULL
            OR lease_expires_at <= execution_deadline
        ),
    ADD CONSTRAINT outbox_events_processing_lease_check
        CHECK (
            status <> 'processing'
            OR (
                lease_token IS NOT NULL
                AND lease_expires_at IS NOT NULL
                AND execution_deadline IS NOT NULL
            )
        );

-- In-flight rows from the pre-lease implementation are intentionally assigned
-- an unknown token. They remain unavailable until the short lease expires and
-- are then recovered through the normal claim path.
UPDATE outbox_events
SET lease_token = gen_random_uuid(),
    lease_generation = 1,
    lease_expires_at = COALESCE(locked_at, now()) + interval '30 seconds',
    execution_deadline = COALESCE(locked_at, now()) + interval '5 minutes'
WHERE status = 'processing';

CREATE INDEX outbox_events_lease_expiry_idx
    ON outbox_events (lease_expires_at, created_at)
    WHERE status = 'processing';

-- +goose Down
DROP INDEX IF EXISTS outbox_events_lease_expiry_idx;

ALTER TABLE outbox_events
    DROP CONSTRAINT IF EXISTS outbox_events_processing_lease_check,
    DROP CONSTRAINT IF EXISTS outbox_events_lease_deadline_check,
    DROP CONSTRAINT IF EXISTS outbox_events_lease_generation_check,
    DROP COLUMN IF EXISTS execution_deadline,
    DROP COLUMN IF EXISTS lease_expires_at,
    DROP COLUMN IF EXISTS lease_generation,
    DROP COLUMN IF EXISTS lease_token;
