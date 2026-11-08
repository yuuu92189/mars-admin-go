-- +goose Up
ALTER TABLE identity_refresh_token_history
    ADD COLUMN rotated_at TIMESTAMPTZ;

UPDATE identity_refresh_token_history
SET rotated_at = used_at
WHERE rotated_at IS NULL;

ALTER TABLE identity_refresh_token_history
    ALTER COLUMN rotated_at SET DEFAULT now(),
    ALTER COLUMN rotated_at SET NOT NULL;

CREATE INDEX identity_refresh_token_history_rotated_idx
    ON identity_refresh_token_history (rotated_at);

-- +goose Down
DROP INDEX IF EXISTS identity_refresh_token_history_rotated_idx;
ALTER TABLE identity_refresh_token_history
    DROP COLUMN IF EXISTS rotated_at;
