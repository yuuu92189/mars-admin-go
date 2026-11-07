-- +goose Up
ALTER TABLE identity_users
    ADD COLUMN mfa_enabled_at TIMESTAMPTZ;

-- +goose Down
ALTER TABLE identity_users DROP COLUMN IF EXISTS mfa_enabled_at;
