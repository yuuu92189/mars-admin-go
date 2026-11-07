-- +goose Up
CREATE TABLE identity_captcha_challenges (
    id UUID PRIMARY KEY,
    kind VARCHAR(20) NOT NULL,
    answer_hash BYTEA,
    target_x INTEGER,
    tolerance INTEGER NOT NULL DEFAULT 0,
    expires_at TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT identity_captcha_challenges_kind_check CHECK (kind IN ('image', 'slider')),
    CONSTRAINT identity_captcha_challenges_tolerance_check CHECK (tolerance >= 0 AND tolerance <= 20)
);

CREATE INDEX identity_captcha_challenges_active_idx
    ON identity_captcha_challenges (expires_at)
    WHERE consumed_at IS NULL;

-- +goose Down
DROP TABLE IF EXISTS identity_captcha_challenges;
