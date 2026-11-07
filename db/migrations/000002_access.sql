-- +goose Up
CREATE TABLE access_roles (
    id UUID PRIMARY KEY,
    name VARCHAR(80) NOT NULL,
    code CITEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    is_super_admin BOOLEAN NOT NULL DEFAULT false,
    data_scope VARCHAR(40) NOT NULL DEFAULT 'self',
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    version BIGINT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT access_roles_code_unique UNIQUE (code),
    CONSTRAINT access_roles_status_check CHECK (status IN ('active', 'disabled')),
    CONSTRAINT access_roles_data_scope_check CHECK (data_scope IN ('all', 'custom', 'department', 'department_tree', 'self'))
);

CREATE TABLE access_permissions (
    id UUID PRIMARY KEY,
    code VARCHAR(120) NOT NULL,
    name VARCHAR(120) NOT NULL,
    module VARCHAR(40) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT access_permissions_code_unique UNIQUE (code)
);

CREATE TABLE access_user_roles (
    user_id UUID NOT NULL REFERENCES identity_users(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES access_roles(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, role_id)
);

CREATE TABLE access_role_permissions (
    role_id UUID NOT NULL REFERENCES access_roles(id) ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES access_permissions(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (role_id, permission_id)
);

CREATE TABLE access_menus (
    id UUID PRIMARY KEY,
    parent_id UUID REFERENCES access_menus(id) ON DELETE RESTRICT,
    name VARCHAR(80) NOT NULL,
    menu_type SMALLINT NOT NULL,
    route_path VARCHAR(255),
    component VARCHAR(255),
    permission_code VARCHAR(120),
    icon VARCHAR(80) NOT NULL DEFAULT '',
    sort_order INTEGER NOT NULL DEFAULT 0,
    visible BOOLEAN NOT NULL DEFAULT true,
    external_url TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT access_menus_type_check CHECK (menu_type IN (1, 2, 3)),
    CONSTRAINT access_menus_status_check CHECK (status IN ('active', 'disabled')),
    CONSTRAINT access_menus_self_parent_check CHECK (parent_id IS NULL OR parent_id <> id)
);

CREATE INDEX access_menus_parent_idx ON access_menus (parent_id, sort_order);
CREATE INDEX access_menus_permission_idx ON access_menus (permission_code) WHERE permission_code IS NOT NULL;

CREATE TABLE access_role_menus (
    role_id UUID NOT NULL REFERENCES access_roles(id) ON DELETE CASCADE,
    menu_id UUID NOT NULL REFERENCES access_menus(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, menu_id)
);

-- +goose Down
DROP TABLE IF EXISTS access_role_menus;
DROP TABLE IF EXISTS access_menus;
DROP TABLE IF EXISTS access_role_permissions;
DROP TABLE IF EXISTS access_user_roles;
DROP TABLE IF EXISTS access_permissions;
DROP TABLE IF EXISTS access_roles;
