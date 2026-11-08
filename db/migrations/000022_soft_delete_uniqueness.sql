-- +goose Up
ALTER TABLE identity_users
    DROP CONSTRAINT IF EXISTS identity_users_username_unique,
    DROP CONSTRAINT IF EXISTS identity_users_email_unique,
    DROP CONSTRAINT IF EXISTS identity_users_phone_unique;

CREATE UNIQUE INDEX identity_users_username_active_unique
    ON identity_users (username)
    WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX identity_users_email_active_unique
    ON identity_users (email)
    WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX identity_users_phone_active_unique
    ON identity_users (phone)
    WHERE deleted_at IS NULL;

ALTER TABLE access_roles
    DROP CONSTRAINT IF EXISTS access_roles_code_unique;
CREATE UNIQUE INDEX access_roles_code_active_unique
    ON access_roles (code)
    WHERE deleted_at IS NULL;

ALTER TABLE organization_departments
    DROP CONSTRAINT IF EXISTS organization_departments_code_unique,
    DROP CONSTRAINT IF EXISTS organization_departments_path_unique;
CREATE UNIQUE INDEX organization_departments_code_active_unique
    ON organization_departments (code)
    WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX organization_departments_path_active_unique
    ON organization_departments (path)
    WHERE deleted_at IS NULL;

ALTER TABLE organization_positions
    DROP CONSTRAINT IF EXISTS organization_positions_code_unique;
CREATE UNIQUE INDEX organization_positions_code_active_unique
    ON organization_positions (code)
    WHERE deleted_at IS NULL;

-- +goose Down
DROP INDEX IF EXISTS organization_positions_code_active_unique;
ALTER TABLE organization_positions
    ADD CONSTRAINT organization_positions_code_unique UNIQUE (code);

DROP INDEX IF EXISTS organization_departments_path_active_unique;
DROP INDEX IF EXISTS organization_departments_code_active_unique;
ALTER TABLE organization_departments
    ADD CONSTRAINT organization_departments_path_unique UNIQUE (path),
    ADD CONSTRAINT organization_departments_code_unique UNIQUE (code);

DROP INDEX IF EXISTS access_roles_code_active_unique;
ALTER TABLE access_roles
    ADD CONSTRAINT access_roles_code_unique UNIQUE (code);

DROP INDEX IF EXISTS identity_users_phone_active_unique;
DROP INDEX IF EXISTS identity_users_email_active_unique;
DROP INDEX IF EXISTS identity_users_username_active_unique;
ALTER TABLE identity_users
    ADD CONSTRAINT identity_users_username_unique UNIQUE (username),
    ADD CONSTRAINT identity_users_email_unique UNIQUE (email),
    ADD CONSTRAINT identity_users_phone_unique UNIQUE (phone);
