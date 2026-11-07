-- +goose Up
CREATE TABLE messaging_user_blocks (
    user_id UUID NOT NULL REFERENCES identity_users(id) ON DELETE CASCADE,
    blocked_user_id UUID NOT NULL REFERENCES identity_users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, blocked_user_id),
    CONSTRAINT messaging_user_blocks_self_check CHECK (user_id <> blocked_user_id)
);

CREATE INDEX messaging_user_blocks_blocked_idx ON messaging_user_blocks (blocked_user_id);

-- +goose Down
DROP TABLE IF EXISTS messaging_user_blocks;
