-- +goose Up
CREATE TABLE system_dictionary_types (
    id UUID PRIMARY KEY,
    code CITEXT NOT NULL,
    name VARCHAR(100) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT system_dictionary_types_code_unique UNIQUE (code),
    CONSTRAINT system_dictionary_types_status_check CHECK (status IN ('active', 'disabled'))
);

CREATE TABLE system_dictionary_items (
    id UUID PRIMARY KEY,
    type_id UUID NOT NULL REFERENCES system_dictionary_types(id) ON DELETE CASCADE,
    value VARCHAR(200) NOT NULL,
    label VARCHAR(200) NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0,
    tag_style VARCHAR(50) NOT NULL DEFAULT 'default',
    is_default BOOLEAN NOT NULL DEFAULT false,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT system_dictionary_items_value_unique UNIQUE (type_id, value),
    CONSTRAINT system_dictionary_items_status_check CHECK (status IN ('active', 'disabled'))
);

CREATE TABLE system_setting_groups (
    id UUID PRIMARY KEY,
    code CITEXT NOT NULL,
    name VARCHAR(100) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    sort_order INTEGER NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT system_setting_groups_code_unique UNIQUE (code),
    CONSTRAINT system_setting_groups_status_check CHECK (status IN ('active', 'disabled'))
);

CREATE TABLE system_settings (
    id UUID PRIMARY KEY,
    group_id UUID NOT NULL REFERENCES system_setting_groups(id) ON DELETE CASCADE,
    key CITEXT NOT NULL,
    value_type VARCHAR(20) NOT NULL,
    value_text TEXT NOT NULL DEFAULT '',
    is_sensitive BOOLEAN NOT NULL DEFAULT false,
    validation JSONB NOT NULL DEFAULT '{}'::jsonb,
    version BIGINT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT system_settings_key_unique UNIQUE (group_id, key),
    CONSTRAINT system_settings_type_check CHECK (value_type IN ('string', 'integer', 'boolean', 'json', 'secret'))
);

CREATE TABLE system_setting_revisions (
    id UUID PRIMARY KEY,
    setting_id UUID NOT NULL REFERENCES system_settings(id) ON DELETE CASCADE,
    old_value TEXT,
    new_value TEXT,
    changed_by UUID REFERENCES identity_users(id) ON DELETE SET NULL,
    changed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE audit_events (
    id UUID PRIMARY KEY,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    actor_user_id UUID REFERENCES identity_users(id) ON DELETE SET NULL,
    actor_name VARCHAR(100) NOT NULL DEFAULT '',
    action VARCHAR(120) NOT NULL,
    resource_type VARCHAR(80) NOT NULL,
    resource_id VARCHAR(120) NOT NULL DEFAULT '',
    outcome VARCHAR(20) NOT NULL,
    request_id VARCHAR(80) NOT NULL DEFAULT '',
    trace_id VARCHAR(80) NOT NULL DEFAULT '',
    ip_address INET,
    user_agent TEXT NOT NULL DEFAULT '',
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    CONSTRAINT audit_events_outcome_check CHECK (outcome IN ('success', 'failure', 'denied'))
);

CREATE INDEX audit_events_occurred_idx ON audit_events (occurred_at DESC);
CREATE INDEX audit_events_actor_idx ON audit_events (actor_user_id, occurred_at DESC);
CREATE INDEX audit_events_action_idx ON audit_events (action, occurred_at DESC);
CREATE INDEX audit_events_metadata_gin_idx ON audit_events USING GIN (metadata);

CREATE TABLE audit_login_events (
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES identity_users(id) ON DELETE SET NULL,
    username VARCHAR(100) NOT NULL,
    client_type VARCHAR(20) NOT NULL,
    outcome VARCHAR(20) NOT NULL,
    failure_reason VARCHAR(200) NOT NULL DEFAULT '',
    ip_address INET,
    user_agent TEXT NOT NULL DEFAULT '',
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT audit_login_events_outcome_check CHECK (outcome IN ('success', 'failure'))
);

CREATE INDEX audit_login_events_occurred_idx ON audit_login_events (occurred_at DESC);

-- +goose Down
DROP TABLE IF EXISTS audit_login_events;
DROP TABLE IF EXISTS audit_events;
DROP TABLE IF EXISTS system_setting_revisions;
DROP TABLE IF EXISTS system_settings;
DROP TABLE IF EXISTS system_setting_groups;
DROP TABLE IF EXISTS system_dictionary_items;
DROP TABLE IF EXISTS system_dictionary_types;
