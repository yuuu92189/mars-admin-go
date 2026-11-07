-- +goose Up
INSERT INTO system_setting_groups (id, code, name, description, sort_order)
VALUES
    ('019b0000-0000-7000-8000-000000020001', 'system', 'System', 'Site identity, branding and watermark settings.', 1),
    ('019b0000-0000-7000-8000-000000020002', 'register', 'Registration', 'Registration and account approval settings.', 2),
    ('019b0000-0000-7000-8000-000000020003', 'login', 'Login', 'Login protection and session settings.', 3),
    ('019b0000-0000-7000-8000-000000020004', 'password', 'Password policy', 'Password complexity and expiration policy.', 4),
    ('019b0000-0000-7000-8000-000000020005', 'email', 'Email', 'SMTP delivery settings.', 5),
    ('019b0000-0000-7000-8000-000000020006', 'emailTemplate', 'Email templates', 'Transactional email templates.', 6),
    ('019b0000-0000-7000-8000-000000020007', 'sms', 'SMS', 'SMS provider settings.', 7),
    ('019b0000-0000-7000-8000-000000020008', 'smsTemplate', 'SMS templates', 'Transactional SMS templates.', 8),
    ('019b0000-0000-7000-8000-000000020009', 'storage', 'Storage', 'Local and S3-compatible storage settings.', 9),
    ('019b0000-0000-7000-8000-000000020010', 'push', 'Push', 'DingTalk, Feishu and WeCom webhook settings.', 10),
    ('019b0000-0000-7000-8000-000000020011', 'thirdParty', 'Third-party login', 'Social login provider settings.', 11),
    ('019b0000-0000-7000-8000-000000020012', 'wechatMiniProgram', 'WeChat Mini Program', 'WeChat Mini Program login settings.', 12),
    ('019b0000-0000-7000-8000-000000020013', 'wechatMp', 'WeChat Official Account', 'WeChat Official Account settings.', 13),
    ('019b0000-0000-7000-8000-000000020014', 'payment', 'Payment', 'WeChat Pay and Alipay settings.', 14),
    ('019b0000-0000-7000-8000-000000020016', 'other', 'Other', 'Additional application settings.', 16)
ON CONFLICT (code) DO NOTHING;

INSERT INTO system_settings (id, group_id, key, value_type, value_text)
VALUES
    ('019b0000-0000-7000-8000-000000021001', '019b0000-0000-7000-8000-000000020001', 'siteName', 'string', 'Mars Admin Go Edition'),
    ('019b0000-0000-7000-8000-000000021002', '019b0000-0000-7000-8000-000000020001', 'siteDescription', 'string', '现代化后台管理系统'),
    ('019b0000-0000-7000-8000-000000021003', '019b0000-0000-7000-8000-000000020001', 'watermarkEnabled', 'boolean', 'true'),
    ('019b0000-0000-7000-8000-000000021004', '019b0000-0000-7000-8000-000000020001', 'watermarkOpacity', 'json', '0.1')
ON CONFLICT (group_id, key) DO NOTHING;

-- +goose Down
DELETE FROM system_setting_groups
WHERE id IN (
    '019b0000-0000-7000-8000-000000020001',
    '019b0000-0000-7000-8000-000000020002',
    '019b0000-0000-7000-8000-000000020003',
    '019b0000-0000-7000-8000-000000020004',
    '019b0000-0000-7000-8000-000000020005',
    '019b0000-0000-7000-8000-000000020006',
    '019b0000-0000-7000-8000-000000020007',
    '019b0000-0000-7000-8000-000000020008',
    '019b0000-0000-7000-8000-000000020009',
    '019b0000-0000-7000-8000-000000020010',
    '019b0000-0000-7000-8000-000000020011',
    '019b0000-0000-7000-8000-000000020012',
    '019b0000-0000-7000-8000-000000020013',
    '019b0000-0000-7000-8000-000000020014',
    '019b0000-0000-7000-8000-000000020016'
);
