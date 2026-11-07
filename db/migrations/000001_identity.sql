-- +goose Up
CREATE EXTENSION IF NOT EXISTS citext;

CREATE TABLE identity_users (
    id UUID PRIMARY KEY,
    username CITEXT NOT NULL,
    password_hash TEXT NOT NULL,
    display_name VARCHAR(100) NOT NULL,
    email CITEXT,
    phone VARCHAR(32),
    avatar_url TEXT NOT NULL DEFAULT '',
    user_type VARCHAR(20) NOT NULL DEFAULT 'admin',
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    gender SMALLINT NOT NULL DEFAULT 0,
    failed_login_count INTEGER NOT NULL DEFAULT 0,
    locked_until TIMESTAMPTZ,
    mfa_secret_encrypted BYTEA,
    last_login_at TIMESTAMPTZ,
    version BIGINT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT identity_users_username_unique UNIQUE (username),
    CONSTRAINT identity_users_email_unique UNIQUE (email),
    CONSTRAINT identity_users_phone_unique UNIQUE (phone),
    CONSTRAINT identity_users_type_check CHECK (user_type IN ('admin', 'pc', 'app')),
    CONSTRAINT identity_users_status_check CHECK (status IN ('pending', 'active', 'suspended', 'rejected', 'left')),
    CONSTRAINT identity_users_gender_check CHECK (gender IN (0, 1, 2)),
    CONSTRAINT identity_users_failed_login_check CHECK (failed_login_count >= 0)
);

CREATE TABLE identity_sessions (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES identity_users(id) ON DELETE CASCADE,
    client_type VARCHAR(20) NOT NULL,
    token_hash BYTEA NOT NULL,
    refresh_token_hash BYTEA NOT NULL,
    device_name VARCHAR(200) NOT NULL DEFAULT '',
    ip_address INET,
    user_agent TEXT NOT NULL DEFAULT '',
    expires_at TIMESTAMPTZ NOT NULL,
    refresh_expires_at TIMESTAMPTZ NOT NULL,
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT identity_sessions_client_check CHECK (client_type IN ('admin', 'web', 'app')),
    CONSTRAINT identity_sessions_token_unique UNIQUE (token_hash),
    CONSTRAINT identity_sessions_refresh_unique UNIQUE (refresh_token_hash)
);

CREATE INDEX identity_sessions_user_idx ON identity_sessions (user_id, created_at DESC);
CREATE INDEX identity_sessions_active_idx ON identity_sessions (expires_at) WHERE revoked_at IS NULL;

CREATE TABLE identity_social_identities (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES identity_users(id) ON DELETE CASCADE,
    provider VARCHAR(40) NOT NULL,
    subject VARCHAR(255) NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT identity_social_provider_subject_unique UNIQUE (provider, subject)
);

CREATE TABLE identity_one_time_codes (
    id UUID PRIMARY KEY,
    purpose VARCHAR(40) NOT NULL,
    subject VARCHAR(255) NOT NULL,
    code_hash BYTEA NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ,
    attempts INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT identity_one_time_codes_attempts_check CHECK (attempts >= 0)
);

CREATE INDEX identity_one_time_codes_lookup_idx
    ON identity_one_time_codes (purpose, subject, expires_at DESC)
    WHERE consumed_at IS NULL;

-- +goose Down
DROP TABLE IF EXISTS identity_one_time_codes;
DROP TABLE IF EXISTS identity_social_identities;
DROP TABLE IF EXISTS identity_sessions;
DROP TABLE IF EXISTS identity_users;
