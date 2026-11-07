-- +goose Up
ALTER TABLE identity_sessions
    ADD COLUMN refresh_family_id UUID NOT NULL DEFAULT gen_random_uuid(),
    ADD COLUMN refresh_generation BIGINT NOT NULL DEFAULT 0,
    ADD CONSTRAINT identity_sessions_refresh_generation_check CHECK (refresh_generation >= 0);

CREATE TABLE identity_refresh_token_history (
    id UUID PRIMARY KEY,
    session_id UUID NOT NULL REFERENCES identity_sessions(id) ON DELETE CASCADE,
    token_hash BYTEA NOT NULL,
    generation BIGINT NOT NULL,
    used_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT identity_refresh_token_history_hash_unique UNIQUE (token_hash),
    CONSTRAINT identity_refresh_token_history_generation_check CHECK (generation >= 0)
);

CREATE INDEX identity_refresh_token_history_session_idx
    ON identity_refresh_token_history (session_id, generation DESC);

ALTER TABLE identity_one_time_codes
    ADD COLUMN max_attempts INTEGER NOT NULL DEFAULT 5,
    ADD CONSTRAINT identity_one_time_codes_max_attempts_check CHECK (max_attempts > 0 AND max_attempts <= 10);

CREATE TABLE access_cache_epoch (
    singleton BOOLEAN PRIMARY KEY DEFAULT true,
    version BIGINT NOT NULL DEFAULT 1,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT access_cache_epoch_singleton_check CHECK (singleton)
);

INSERT INTO access_cache_epoch (singleton, version) VALUES (true, 1);

-- +goose StatementBegin
CREATE FUNCTION bump_access_cache_epoch() RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    UPDATE access_cache_epoch
    SET version = version + 1, updated_at = now()
    WHERE singleton = true;
    RETURN NULL;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER access_roles_cache_epoch
    AFTER INSERT OR UPDATE OR DELETE ON access_roles
    FOR EACH STATEMENT EXECUTE FUNCTION bump_access_cache_epoch();
CREATE TRIGGER access_menus_cache_epoch
    AFTER INSERT OR UPDATE OR DELETE ON access_menus
    FOR EACH STATEMENT EXECUTE FUNCTION bump_access_cache_epoch();
CREATE TRIGGER access_user_roles_cache_epoch
    AFTER INSERT OR UPDATE OR DELETE ON access_user_roles
    FOR EACH STATEMENT EXECUTE FUNCTION bump_access_cache_epoch();
CREATE TRIGGER access_role_permissions_cache_epoch
    AFTER INSERT OR UPDATE OR DELETE ON access_role_permissions
    FOR EACH STATEMENT EXECUTE FUNCTION bump_access_cache_epoch();
CREATE TRIGGER access_role_menus_cache_epoch
    AFTER INSERT OR UPDATE OR DELETE ON access_role_menus
    FOR EACH STATEMENT EXECUTE FUNCTION bump_access_cache_epoch();
CREATE TRIGGER access_role_departments_cache_epoch
    AFTER INSERT OR UPDATE OR DELETE ON access_role_departments
    FOR EACH STATEMENT EXECUTE FUNCTION bump_access_cache_epoch();
CREATE TRIGGER organization_memberships_cache_epoch
    AFTER INSERT OR UPDATE OR DELETE ON organization_memberships
    FOR EACH STATEMENT EXECUTE FUNCTION bump_access_cache_epoch();
CREATE TRIGGER organization_departments_cache_epoch
    AFTER INSERT OR UPDATE OR DELETE ON organization_departments
    FOR EACH STATEMENT EXECUTE FUNCTION bump_access_cache_epoch();
CREATE TRIGGER identity_users_cache_epoch
    AFTER INSERT OR DELETE OR UPDATE OF status, deleted_at ON identity_users
    FOR EACH ROW EXECUTE FUNCTION bump_access_cache_epoch();

-- +goose Down
DROP TRIGGER IF EXISTS identity_users_cache_epoch ON identity_users;
DROP TRIGGER IF EXISTS organization_departments_cache_epoch ON organization_departments;
DROP TRIGGER IF EXISTS organization_memberships_cache_epoch ON organization_memberships;
DROP TRIGGER IF EXISTS access_role_departments_cache_epoch ON access_role_departments;
DROP TRIGGER IF EXISTS access_role_menus_cache_epoch ON access_role_menus;
DROP TRIGGER IF EXISTS access_role_permissions_cache_epoch ON access_role_permissions;
DROP TRIGGER IF EXISTS access_user_roles_cache_epoch ON access_user_roles;
DROP TRIGGER IF EXISTS access_menus_cache_epoch ON access_menus;
DROP TRIGGER IF EXISTS access_roles_cache_epoch ON access_roles;
DROP FUNCTION IF EXISTS bump_access_cache_epoch();
DROP TABLE IF EXISTS access_cache_epoch;
ALTER TABLE identity_one_time_codes
    DROP CONSTRAINT IF EXISTS identity_one_time_codes_max_attempts_check,
    DROP COLUMN IF EXISTS max_attempts;
DROP TABLE IF EXISTS identity_refresh_token_history;
ALTER TABLE identity_sessions
    DROP CONSTRAINT IF EXISTS identity_sessions_refresh_generation_check,
    DROP COLUMN IF EXISTS refresh_generation,
    DROP COLUMN IF EXISTS refresh_family_id;
