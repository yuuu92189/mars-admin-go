-- +goose Up
ALTER TABLE identity_users
    ADD COLUMN password_login_enabled BOOLEAN NOT NULL DEFAULT true;

-- +goose Down
ALTER TABLE identity_users
    DROP COLUMN IF EXISTS password_login_enabled;
