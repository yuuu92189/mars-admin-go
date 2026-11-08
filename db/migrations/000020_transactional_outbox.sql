-- +goose Up
CREATE TABLE outbox_events (
    id UUID PRIMARY KEY,
    aggregate_type VARCHAR(80) NOT NULL,
    aggregate_id VARCHAR(160) NOT NULL,
    event_type VARCHAR(120) NOT NULL,
    deduplication_key VARCHAR(160) NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    attempts INTEGER NOT NULL DEFAULT 0,
    max_attempts INTEGER NOT NULL DEFAULT 10,
    available_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    locked_at TIMESTAMPTZ,
    published_at TIMESTAMPTZ,
    last_error TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT outbox_events_status_check
        CHECK (status IN ('pending', 'processing', 'published', 'failed', 'dead')),
    CONSTRAINT outbox_events_attempts_check CHECK (attempts >= 0),
    CONSTRAINT outbox_events_max_attempts_check CHECK (max_attempts > 0),
    CONSTRAINT outbox_events_deduplication_unique
        UNIQUE (aggregate_type, aggregate_id, event_type, deduplication_key)
);

CREATE INDEX outbox_events_ready_idx
    ON outbox_events (available_at, created_at)
    WHERE status IN ('pending', 'failed');

CREATE INDEX outbox_events_locked_idx
    ON outbox_events (locked_at)
    WHERE status = 'processing';

CREATE INDEX outbox_events_aggregate_idx
    ON outbox_events (aggregate_type, aggregate_id, created_at DESC);

CREATE INDEX outbox_events_published_idx
    ON outbox_events (published_at)
    WHERE status = 'published';

-- +goose Down
DROP TABLE IF EXISTS outbox_events;
