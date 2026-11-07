-- +goose Up
CREATE EXTENSION IF NOT EXISTS ltree;

CREATE TABLE organization_departments (
    id UUID PRIMARY KEY,
    parent_id UUID REFERENCES organization_departments(id) ON DELETE RESTRICT,
    name VARCHAR(100) NOT NULL,
    code CITEXT NOT NULL,
    path LTREE NOT NULL,
    leader_name VARCHAR(100) NOT NULL DEFAULT '',
    phone VARCHAR(32) NOT NULL DEFAULT '',
    email CITEXT,
    sort_order INTEGER NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT organization_departments_code_unique UNIQUE (code),
    CONSTRAINT organization_departments_path_unique UNIQUE (path),
    CONSTRAINT organization_departments_status_check CHECK (status IN ('active', 'disabled')),
    CONSTRAINT organization_departments_self_parent_check CHECK (parent_id IS NULL OR parent_id <> id)
);

CREATE INDEX organization_departments_path_gist_idx ON organization_departments USING GIST (path);
CREATE INDEX organization_departments_parent_idx ON organization_departments (parent_id, sort_order);

CREATE TABLE organization_positions (
    id UUID PRIMARY KEY,
    parent_id UUID REFERENCES organization_positions(id) ON DELETE RESTRICT,
    code CITEXT NOT NULL,
    name VARCHAR(100) NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT organization_positions_code_unique UNIQUE (code),
    CONSTRAINT organization_positions_status_check CHECK (status IN ('active', 'disabled'))
);

CREATE TABLE organization_memberships (
    user_id UUID NOT NULL REFERENCES identity_users(id) ON DELETE CASCADE,
    department_id UUID NOT NULL REFERENCES organization_departments(id) ON DELETE CASCADE,
    is_primary BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, department_id)
);

CREATE UNIQUE INDEX organization_one_primary_department_idx
    ON organization_memberships (user_id)
    WHERE is_primary;

CREATE TABLE organization_user_positions (
    user_id UUID NOT NULL REFERENCES identity_users(id) ON DELETE CASCADE,
    position_id UUID NOT NULL REFERENCES organization_positions(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, position_id)
);

CREATE TABLE access_role_departments (
    role_id UUID NOT NULL REFERENCES access_roles(id) ON DELETE CASCADE,
    department_id UUID NOT NULL REFERENCES organization_departments(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, department_id)
);

-- +goose Down
DROP TABLE IF EXISTS access_role_departments;
DROP TABLE IF EXISTS organization_user_positions;
DROP TABLE IF EXISTS organization_memberships;
DROP TABLE IF EXISTS organization_positions;
DROP TABLE IF EXISTS organization_departments;
