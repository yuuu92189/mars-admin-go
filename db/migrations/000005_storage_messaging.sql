-- +goose Up
CREATE TABLE storage_file_groups (
    id UUID PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_by UUID REFERENCES identity_users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE storage_files (
    id UUID PRIMARY KEY,
    group_id UUID REFERENCES storage_file_groups(id) ON DELETE SET NULL,
    owner_user_id UUID REFERENCES identity_users(id) ON DELETE SET NULL,
    original_name VARCHAR(255) NOT NULL,
    storage_key TEXT NOT NULL,
    storage_provider VARCHAR(40) NOT NULL,
    content_type VARCHAR(200) NOT NULL,
    size_bytes BIGINT NOT NULL,
    sha256_hash BYTEA NOT NULL,
    visibility VARCHAR(20) NOT NULL DEFAULT 'private',
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT storage_files_storage_key_unique UNIQUE (storage_key),
    CONSTRAINT storage_files_size_check CHECK (size_bytes >= 0),
    CONSTRAINT storage_files_visibility_check CHECK (visibility IN ('private', 'internal', 'public')),
    CONSTRAINT storage_files_status_check CHECK (status IN ('active', 'quarantined', 'deleted'))
);

CREATE INDEX storage_files_owner_idx ON storage_files (owner_user_id, created_at DESC);
CREATE INDEX storage_files_group_idx ON storage_files (group_id, created_at DESC);

CREATE TABLE messaging_groups (
    id UUID PRIMARY KEY,
    name VARCHAR(150) NOT NULL,
    avatar_url TEXT NOT NULL DEFAULT '',
    owner_id UUID NOT NULL REFERENCES identity_users(id) ON DELETE RESTRICT,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    dissolved_at TIMESTAMPTZ,
    CONSTRAINT messaging_groups_status_check CHECK (status IN ('active', 'dissolved'))
);

CREATE TABLE messaging_group_members (
    group_id UUID NOT NULL REFERENCES messaging_groups(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES identity_users(id) ON DELETE CASCADE,
    member_role VARCHAR(20) NOT NULL DEFAULT 'member',
    muted_until TIMESTAMPTZ,
    joined_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    left_at TIMESTAMPTZ,
    PRIMARY KEY (group_id, user_id),
    CONSTRAINT messaging_group_members_role_check CHECK (member_role IN ('owner', 'admin', 'member'))
);

CREATE INDEX messaging_group_members_user_idx ON messaging_group_members (user_id) WHERE left_at IS NULL;

CREATE TABLE messaging_messages (
    id UUID PRIMARY KEY,
    client_message_id VARCHAR(120),
    sender_id UUID NOT NULL REFERENCES identity_users(id) ON DELETE RESTRICT,
    receiver_id UUID REFERENCES identity_users(id) ON DELETE RESTRICT,
    group_id UUID REFERENCES messaging_groups(id) ON DELETE CASCADE,
    message_type VARCHAR(20) NOT NULL DEFAULT 'text',
    content TEXT NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    sequence_number BIGINT GENERATED ALWAYS AS IDENTITY,
    edited_at TIMESTAMPTZ,
    recalled_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT messaging_messages_target_check CHECK (
        (receiver_id IS NOT NULL AND group_id IS NULL)
        OR (receiver_id IS NULL AND group_id IS NOT NULL)
    ),
    CONSTRAINT messaging_messages_type_check CHECK (message_type IN ('text', 'image', 'file', 'system')),
    CONSTRAINT messaging_messages_content_check CHECK (length(content) BETWEEN 1 AND 10000)
);

CREATE UNIQUE INDEX messaging_messages_idempotency_idx
    ON messaging_messages (sender_id, client_message_id)
    WHERE client_message_id IS NOT NULL;
CREATE INDEX messaging_messages_direct_idx ON messaging_messages (sender_id, receiver_id, sequence_number DESC);
CREATE INDEX messaging_messages_group_idx ON messaging_messages (group_id, sequence_number DESC);

CREATE TABLE messaging_read_receipts (
    message_id UUID NOT NULL REFERENCES messaging_messages(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES identity_users(id) ON DELETE CASCADE,
    read_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (message_id, user_id)
);

CREATE TABLE notification_notices (
    id UUID PRIMARY KEY,
    title VARCHAR(200) NOT NULL,
    content TEXT NOT NULL,
    notice_type SMALLINT NOT NULL DEFAULT 1,
    channels JSONB NOT NULL DEFAULT '["station"]'::jsonb,
    target_type VARCHAR(20) NOT NULL DEFAULT 'all',
    target_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    status VARCHAR(20) NOT NULL DEFAULT 'draft',
    created_by UUID REFERENCES identity_users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    published_at TIMESTAMPTZ,
    withdrawn_at TIMESTAMPTZ,
    CONSTRAINT notification_notices_type_check CHECK (notice_type IN (1, 2)),
    CONSTRAINT notification_notices_status_check CHECK (status IN ('draft', 'published', 'withdrawn')),
    CONSTRAINT notification_notices_target_check CHECK (target_type IN ('all', 'users', 'departments'))
);

CREATE TABLE notification_recipients (
    notice_id UUID NOT NULL REFERENCES notification_notices(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES identity_users(id) ON DELETE CASCADE,
    read_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (notice_id, user_id)
);

CREATE INDEX notification_recipients_user_idx ON notification_recipients (user_id, read_at);

CREATE TABLE notification_send_logs (
    id UUID PRIMARY KEY,
    notice_id UUID NOT NULL REFERENCES notification_notices(id) ON DELETE CASCADE,
    channel VARCHAR(40) NOT NULL,
    status VARCHAR(20) NOT NULL,
    target_count INTEGER NOT NULL DEFAULT 0,
    success_count INTEGER NOT NULL DEFAULT 0,
    error_message TEXT NOT NULL DEFAULT '',
    attempt INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT notification_send_logs_status_check CHECK (status IN ('pending', 'success', 'failed'))
);

-- +goose Down
DROP TABLE IF EXISTS notification_send_logs;
DROP TABLE IF EXISTS notification_recipients;
DROP TABLE IF EXISTS notification_notices;
DROP TABLE IF EXISTS messaging_read_receipts;
DROP TABLE IF EXISTS messaging_messages;
DROP TABLE IF EXISTS messaging_group_members;
DROP TABLE IF EXISTS messaging_groups;
DROP TABLE IF EXISTS storage_files;
DROP TABLE IF EXISTS storage_file_groups;
