-- +goose Up
INSERT INTO access_menus (
    id, parent_id, name, menu_type, route_path, component, permission_code,
    icon, sort_order, visible, status
) VALUES
    ('019b0000-0000-7000-8000-000000010001', NULL, 'Dashboard', 2, 'dashboard', 'dashboard/index', '', 'HomeOutline', 1, true, 'active'),
    ('019b0000-0000-7000-8000-000000010100', NULL, 'System', 1, 'system', '', '', 'SettingsOutline', 10, true, 'active'),
    ('019b0000-0000-7000-8000-000000010101', '019b0000-0000-7000-8000-000000010100', 'Users', 2, 'system/user', 'system/user/index', 'user:read', 'PersonOutline', 1, true, 'active'),
    ('019b0000-0000-7000-8000-000000010102', '019b0000-0000-7000-8000-000000010100', 'Roles', 2, 'system/role', 'system/role/index', 'role:read', 'PeopleOutline', 2, true, 'active'),
    ('019b0000-0000-7000-8000-000000010103', '019b0000-0000-7000-8000-000000010100', 'Menus', 2, 'system/menu', 'system/menu/index', 'role:read', 'MenuOutline', 3, true, 'active'),
    ('019b0000-0000-7000-8000-000000010104', '019b0000-0000-7000-8000-000000010100', 'Dictionaries', 2, 'system/dict', 'system/dict/index', 'dictionary:read', 'BookOutline', 4, true, 'active'),
    ('019b0000-0000-7000-8000-000000010105', '019b0000-0000-7000-8000-000000010100', 'Settings', 2, 'system/config', 'system/config/index', 'settings:read', 'SettingsSharp', 5, true, 'active'),
    ('019b0000-0000-7000-8000-000000010106', '019b0000-0000-7000-8000-000000010100', 'Files', 2, 'system/file', 'system/file/index', 'file:download', 'FolderOutline', 6, true, 'active'),
    ('019b0000-0000-7000-8000-000000010200', NULL, 'Organization', 1, 'org', '', '', 'GitNetworkOutline', 20, true, 'active'),
    ('019b0000-0000-7000-8000-000000010201', '019b0000-0000-7000-8000-000000010200', 'Departments', 2, 'org/dept', 'org/dept/index', 'department:read', 'GitNetworkOutline', 1, true, 'active'),
    ('019b0000-0000-7000-8000-000000010202', '019b0000-0000-7000-8000-000000010200', 'Positions', 2, 'org/post', 'org/post/index', 'department:read', 'IdCardOutline', 2, true, 'active'),
    ('019b0000-0000-7000-8000-000000010300', NULL, 'Messages', 1, 'message', '', '', 'ChatbubbleOutline', 30, true, 'active'),
    ('019b0000-0000-7000-8000-000000010301', '019b0000-0000-7000-8000-000000010300', 'Notices', 2, 'message/notice', 'message/notice/index', 'notice:read', 'NotificationsOutline', 1, true, 'active'),
    ('019b0000-0000-7000-8000-000000010302', '019b0000-0000-7000-8000-000000010300', 'Chat', 2, 'message/chat', 'message/chat/index', 'message:read', 'ChatbubbleOutline', 2, true, 'active'),
    ('019b0000-0000-7000-8000-000000010400', NULL, 'Logs', 1, 'log', '', '', 'ListOutline', 40, true, 'active'),
    ('019b0000-0000-7000-8000-000000010401', '019b0000-0000-7000-8000-000000010400', 'Login Logs', 2, 'log/loginlog', 'log/loginlog/index', 'audit:read', 'LogInOutline', 1, true, 'active'),
    ('019b0000-0000-7000-8000-000000010402', '019b0000-0000-7000-8000-000000010400', 'Operation Logs', 2, 'log/operlog', 'log/operlog/index', 'audit:read', 'ListOutline', 2, true, 'active'),
    ('019b0000-0000-7000-8000-000000010500', NULL, 'Monitoring', 1, 'monitor', '', '', 'ServerOutline', 50, true, 'active'),
    ('019b0000-0000-7000-8000-000000010501', '019b0000-0000-7000-8000-000000010500', 'Scheduled Jobs', 2, 'monitor/job', 'monitor/job/index', 'scheduler:read', 'TimerOutline', 1, true, 'active'),
    ('019b0000-0000-7000-8000-000000010502', '019b0000-0000-7000-8000-000000010500', 'Server', 2, 'monitor/server', 'monitor/server/index', 'audit:read', 'DesktopOutline', 2, true, 'active'),
    ('019b0000-0000-7000-8000-000000010600', NULL, 'Workflow', 1, 'workflow', '', '', 'DocumentTextOutline', 60, true, 'active'),
    ('019b0000-0000-7000-8000-000000010601', '019b0000-0000-7000-8000-000000010600', 'Definitions', 2, 'workflow/definition', 'workflow/definition/index', 'workflow:read', 'DocumentTextOutline', 1, true, 'active'),
    ('019b0000-0000-7000-8000-000000010602', '019b0000-0000-7000-8000-000000010600', 'Todo', 2, 'workflow/task/todo', 'workflow/task/todo', 'workflow:read', 'CheckboxOutline', 2, true, 'active'),
    ('019b0000-0000-7000-8000-000000010603', '019b0000-0000-7000-8000-000000010600', 'Done', 2, 'workflow/task/done', 'workflow/task/done', 'workflow:read', 'CheckmarkDoneOutline', 3, true, 'active'),
    ('019b0000-0000-7000-8000-000000010604', '019b0000-0000-7000-8000-000000010600', 'Monitor', 2, 'workflow/monitor', 'workflow/monitor/index', 'workflow:read', 'AnalyticsOutline', 4, true, 'active'),
    ('019b0000-0000-7000-8000-000000010700', NULL, 'Codegen', 2, 'tool/gen', 'tool/gen/index', 'codegen:read', 'CodeSlashOutline', 70, true, 'active')
ON CONFLICT (id) DO NOTHING;

-- +goose Down
DELETE FROM access_menus WHERE id::text LIKE '019b0000-0000-7000-8000-00000001%';
