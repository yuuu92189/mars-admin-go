-- +goose Up
INSERT INTO access_roles (id, name, code, description, is_super_admin, data_scope)
VALUES ('019b0000-0000-7000-8000-000000000001', 'Super Administrator', 'admin', 'Bootstrap super administrator role', true, 'all')
ON CONFLICT (code) DO NOTHING;

INSERT INTO access_permissions (id, code, name, module)
VALUES
    ('019b0000-0000-7000-8000-000000000101', 'user:read', 'Read users', 'identity'),
    ('019b0000-0000-7000-8000-000000000102', 'user:create', 'Create users', 'identity'),
    ('019b0000-0000-7000-8000-000000000103', 'user:update', 'Update users', 'identity'),
    ('019b0000-0000-7000-8000-000000000104', 'user:delete', 'Delete users', 'identity'),
    ('019b0000-0000-7000-8000-000000000201', 'role:read', 'Read roles', 'access'),
    ('019b0000-0000-7000-8000-000000000202', 'role:create', 'Create roles', 'access'),
    ('019b0000-0000-7000-8000-000000000203', 'role:assign', 'Assign roles', 'access'),
    ('019b0000-0000-7000-8000-000000000301', 'department:read', 'Read departments', 'organization'),
    ('019b0000-0000-7000-8000-000000000302', 'department:manage', 'Manage departments', 'organization'),
    ('019b0000-0000-7000-8000-000000000401', 'dictionary:read', 'Read dictionaries', 'dictionary'),
    ('019b0000-0000-7000-8000-000000000402', 'dictionary:manage', 'Manage dictionaries', 'dictionary'),
    ('019b0000-0000-7000-8000-000000000501', 'settings:read', 'Read settings', 'settings'),
    ('019b0000-0000-7000-8000-000000000502', 'settings:manage', 'Manage settings', 'settings'),
    ('019b0000-0000-7000-8000-000000000601', 'audit:read', 'Read audit events', 'audit'),
    ('019b0000-0000-7000-8000-000000000701', 'file:upload', 'Upload files', 'storage'),
    ('019b0000-0000-7000-8000-000000000702', 'file:download', 'Download files', 'storage'),
    ('019b0000-0000-7000-8000-000000000703', 'file:delete', 'Delete files', 'storage'),
    ('019b0000-0000-7000-8000-000000000801', 'message:send', 'Send messages', 'messaging'),
    ('019b0000-0000-7000-8000-000000000802', 'message:read', 'Read messages', 'messaging'),
    ('019b0000-0000-7000-8000-000000000901', 'notice:read', 'Read notices', 'notification'),
    ('019b0000-0000-7000-8000-000000000902', 'notice:manage', 'Manage notices', 'notification'),
    ('019b0000-0000-7000-8000-000000001001', 'scheduler:read', 'Read scheduler', 'scheduler'),
    ('019b0000-0000-7000-8000-000000001002', 'scheduler:manage', 'Manage scheduler', 'scheduler'),
    ('019b0000-0000-7000-8000-000000001101', 'workflow:read', 'Read workflows', 'workflow'),
    ('019b0000-0000-7000-8000-000000001102', 'workflow:manage', 'Manage workflows', 'workflow'),
    ('019b0000-0000-7000-8000-000000001103', 'workflow:approve', 'Approve workflow tasks', 'workflow'),
    ('019b0000-0000-7000-8000-000000001201', 'codegen:read', 'Read generated code', 'codegen'),
    ('019b0000-0000-7000-8000-000000001202', 'codegen:manage', 'Generate code', 'codegen')
ON CONFLICT (code) DO NOTHING;

INSERT INTO system_setting_groups (id, code, name, sort_order)
VALUES
    ('019b0000-0000-7000-8000-000000002001', 'security', 'Security', 10),
    ('019b0000-0000-7000-8000-000000002002', 'site', 'Site', 20),
    ('019b0000-0000-7000-8000-000000002003', 'integrations', 'Integrations', 30)
ON CONFLICT (code) DO NOTHING;

INSERT INTO system_settings (id, group_id, key, value_type, value_text, is_sensitive)
VALUES
    ('019b0000-0000-7000-8000-000000002101', '019b0000-0000-7000-8000-000000002001', 'auth.captcha_enabled', 'boolean', 'false', false),
    ('019b0000-0000-7000-8000-000000002102', '019b0000-0000-7000-8000-000000002001', 'auth.login_max_attempts', 'integer', '5', false),
    ('019b0000-0000-7000-8000-000000002201', '019b0000-0000-7000-8000-000000002002', 'site.name', 'string', 'Mars Admin Go Edition', false),
    ('019b0000-0000-7000-8000-000000002202', '019b0000-0000-7000-8000-000000002002', 'site.registration_enabled', 'boolean', 'false', false)
ON CONFLICT (group_id, key) DO NOTHING;

-- +goose Down
DELETE FROM system_settings WHERE id IN (
    '019b0000-0000-7000-8000-000000002101',
    '019b0000-0000-7000-8000-000000002102',
    '019b0000-0000-7000-8000-000000002201',
    '019b0000-0000-7000-8000-000000002202'
);
DELETE FROM system_setting_groups WHERE id IN (
    '019b0000-0000-7000-8000-000000002001',
    '019b0000-0000-7000-8000-000000002002',
    '019b0000-0000-7000-8000-000000002003'
);
DELETE FROM access_permissions WHERE code LIKE 'user:%' OR code LIKE 'role:%' OR code LIKE 'department:%'
    OR code LIKE 'dictionary:%' OR code LIKE 'settings:%' OR code LIKE 'audit:%' OR code LIKE 'file:%'
    OR code LIKE 'message:%' OR code LIKE 'notice:%' OR code LIKE 'scheduler:%' OR code LIKE 'workflow:%'
    OR code LIKE 'codegen:%';
DELETE FROM access_roles WHERE code = 'admin';
