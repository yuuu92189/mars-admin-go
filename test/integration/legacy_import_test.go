//go:build integration

package integration_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	legacyimport "mars-admin-go/internal/migration/legacy"
	identitypostgres "mars-admin-go/internal/modules/identity/adapter/postgres"
	identityapp "mars-admin-go/internal/modules/identity/application"
	identitydomain "mars-admin-go/internal/modules/identity/domain"
	"mars-admin-go/internal/platform/config"
	"mars-admin-go/internal/platform/database"

	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	mysqltest "github.com/testcontainers/testcontainers-go/modules/mysql"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"golang.org/x/crypto/bcrypt"
)

func TestLegacyCoreImportAndBCryptUpgrade(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	mysqlContainer, err := mysqltest.Run(ctx, "mysql:8.0.36",
		mysqltest.WithDatabase("mars_system"),
		mysqltest.WithUsername("mars"),
		mysqltest.WithPassword("mars-password"),
	)
	require.NoError(t, err)
	testcontainers.CleanupContainer(t, mysqlContainer)
	mysqlDSN, err := mysqlContainer.ConnectionString(ctx, "parseTime=true", "loc=UTC")
	require.NoError(t, err)
	source, err := sql.Open("mysql", mysqlDSN)
	require.NoError(t, err)
	defer func() { _ = source.Close() }()
	seedLegacyMySQL(t, ctx, source)

	postgresContainer, err := postgres.Run(ctx, "postgres:18.6-bookworm",
		postgres.WithDatabase("mars_admin"),
		postgres.WithUsername("mars"),
		postgres.WithPassword("mars-password"),
		postgres.BasicWaitStrategies(),
	)
	require.NoError(t, err)
	testcontainers.CleanupContainer(t, postgresContainer)
	targetURL, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	pool, err := database.Open(ctx, config.DatabaseConfig{
		URL: targetURL, MaxConnections: 5, MinConnections: 1, ConnectionTimeout: 10 * time.Second,
	})
	require.NoError(t, err)
	defer pool.Close()
	require.NoError(t, database.RunMigrations(ctx, pool, "up"))

	legacyUploadRoot := t.TempDir()
	targetUploadRoot := t.TempDir()
	legacyFile := filepath.Join(legacyUploadRoot, "2026", "01", "integration.txt")
	require.NoError(t, os.MkdirAll(filepath.Dir(legacyFile), 0o750))
	require.NoError(t, os.WriteFile(legacyFile, []byte("legacy file body"), 0o640))

	report, err := legacyimport.Run(ctx, legacyimport.Options{
		MySQLDSN: mysqlDSN, Postgres: pool, SourceDescription: "integration",
		LegacyUploadRoot: legacyUploadRoot, TargetUploadRoot: targetUploadRoot,
	})
	if err != nil {
		encoded, _ := json.MarshalIndent(report, "", "  ")
		t.Fatalf("legacy import failed: %v\n%s", err, encoded)
	}
	require.Equal(t, "completed", report.Status)
	require.Equal(t, 1, report.Tables["identity_users"].Inserted)
	require.Equal(t, 1, report.Tables["organization_departments"].Inserted)
	require.Equal(t, 1, report.Tables["access_roles"].Inserted)
	require.Equal(t, 2, report.Tables["messaging_messages"].Inserted)
	require.Equal(t, 1, report.Tables["notification_notices"].Inserted)
	require.Equal(t, 1, report.Tables["notification_recipients"].Inserted)
	require.Equal(t, 1, report.Tables["storage_files"].Inserted)

	targetFile := filepath.Join(targetUploadRoot, "2026", "01", "integration.txt")
	fileBody, err := os.ReadFile(targetFile)
	require.NoError(t, err)
	require.Equal(t, "legacy file body", string(fileBody))

	secondReport, err := legacyimport.Run(ctx, legacyimport.Options{
		MySQLDSN: mysqlDSN, Postgres: pool, SourceDescription: "integration-rerun",
		LegacyUploadRoot: legacyUploadRoot, TargetUploadRoot: targetUploadRoot,
	})
	require.NoError(t, err)
	require.Equal(t, "completed", secondReport.Status)
	require.Equal(t, 0, secondReport.Tables["identity_users"].Inserted)
	require.GreaterOrEqual(t, secondReport.Tables["identity_users"].Skipped, 1)

	var settingJSON string
	require.NoError(t, pool.QueryRow(ctx, `
		SELECT value_text FROM system_settings WHERE key = 'legacy.system'`).Scan(&settingJSON))
	var settings map[string]any
	require.NoError(t, json.Unmarshal([]byte(settingJSON), &settings))
	smtp, ok := settings["smtp"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "", smtp["password"])

	identityRepo := identitypostgres.New(pool)
	identityService := identityapp.New(identityRepo, nil, identityapp.Options{})
	login, err := identityService.Login(ctx, identitydomain.LoginInput{
		Username: "legacyadmin", Password: "legacy-password", ClientType: "admin",
	})
	require.NoError(t, err)
	require.NotEmpty(t, login.AccessToken)

	var passwordHash string
	require.NoError(t, pool.QueryRow(ctx, `
		SELECT password_hash FROM identity_users WHERE username = 'legacyadmin'`).Scan(&passwordHash))
	require.True(t, strings.HasPrefix(passwordHash, "$argon2id$"))
}

func seedLegacyMySQL(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("legacy-password"), bcrypt.MinCost)
	require.NoError(t, err)

	schema := []string{
		`CREATE TABLE sys_dept (
			id BIGINT PRIMARY KEY, parent_id BIGINT, ancestors VARCHAR(500), dept_name VARCHAR(100),
			sort INT, leader VARCHAR(100), phone VARCHAR(30), email VARCHAR(100), status TINYINT,
			create_time DATETIME, update_time DATETIME, deleted TINYINT
		)`,
		`CREATE TABLE sys_post (
			id BIGINT PRIMARY KEY, parent_id BIGINT, post_code VARCHAR(100), post_name VARCHAR(100),
			sort INT, status TINYINT, remark VARCHAR(500), create_time DATETIME, update_time DATETIME, deleted TINYINT
		)`,
		`CREATE TABLE sys_role (
			id BIGINT PRIMARY KEY, name VARCHAR(100), code VARCHAR(100), sort INT, status TINYINT,
			remark VARCHAR(500), data_scope TINYINT, create_time DATETIME, update_time DATETIME, deleted TINYINT
		)`,
		`CREATE TABLE sys_user (
			id BIGINT PRIMARY KEY, dept_id BIGINT, username VARCHAR(100), password VARCHAR(200),
			nickname VARCHAR(100), avatar VARCHAR(500), email VARCHAR(100), phone VARCHAR(30),
			gender TINYINT, status TINYINT, is_quit TINYINT, remark VARCHAR(500),
			create_time DATETIME, update_time DATETIME, user_type VARCHAR(20), open_id VARCHAR(100), deleted TINYINT
		)`,
		`CREATE TABLE sys_menu (
			id BIGINT PRIMARY KEY, parent_id BIGINT, name VARCHAR(100), type TINYINT, path VARCHAR(500),
			component VARCHAR(500), permission VARCHAR(200), icon VARCHAR(100), sort INT,
			visible TINYINT, status TINYINT, is_frame TINYINT, create_time DATETIME, update_time DATETIME, deleted TINYINT
		)`,
		`CREATE TABLE sys_user_role (user_id BIGINT, role_id BIGINT)`,
		`CREATE TABLE sys_role_menu (role_id BIGINT, menu_id BIGINT)`,
		`CREATE TABLE sys_role_dept (role_id BIGINT, dept_id BIGINT)`,
		`CREATE TABLE sys_user_post (user_id BIGINT, post_id BIGINT)`,
		`CREATE TABLE sys_dict_type (
			id BIGINT PRIMARY KEY, dict_name VARCHAR(100), dict_type VARCHAR(100),
			status TINYINT, remark VARCHAR(500), deleted TINYINT
		)`,
		`CREATE TABLE sys_dict_data (
			id BIGINT PRIMARY KEY, sort INT, dict_label VARCHAR(100), dict_value VARCHAR(100),
			dict_type VARCHAR(100), css_class VARCHAR(100), list_class VARCHAR(100),
			is_default TINYINT, status TINYINT, deleted TINYINT
		)`,
		`CREATE TABLE sys_config_group (
			id BIGINT PRIMARY KEY, group_code VARCHAR(100), group_name VARCHAR(100),
			config_value TEXT, sort INT, status TINYINT
		)`,
		`CREATE TABLE sys_file_group (
			id BIGINT PRIMARY KEY, name VARCHAR(100), sort INT, create_time DATETIME
		)`,
		`CREATE TABLE sys_file (
			id BIGINT PRIMARY KEY, original_name VARCHAR(200), file_name VARCHAR(200),
			file_path VARCHAR(500), url VARCHAR(500), file_size BIGINT, file_type VARCHAR(100),
			file_suffix VARCHAR(20), storage_type VARCHAR(20), group_id BIGINT,
			create_by VARCHAR(64), create_time DATETIME
		)`,
		`CREATE TABLE sys_chat_group (
			id BIGINT PRIMARY KEY, name VARCHAR(100), avatar VARCHAR(500), owner_id BIGINT,
			announcement VARCHAR(500), max_members INT, status TINYINT,
			create_time DATETIME, update_time DATETIME
		)`,
		`CREATE TABLE sys_chat_group_member (
			id BIGINT PRIMARY KEY, group_id BIGINT, user_id BIGINT, nickname VARCHAR(100),
			role TINYINT, muted TINYINT, join_time DATETIME
		)`,
		`CREATE TABLE sys_chat_message (
			id BIGINT PRIMARY KEY, sender_id BIGINT, sender_name VARCHAR(100), sender_avatar VARCHAR(500),
			receiver_id BIGINT, content TEXT, msg_type TINYINT, is_read TINYINT, send_time DATETIME
		)`,
		`CREATE TABLE sys_chat_group_message (
			id BIGINT PRIMARY KEY, group_id BIGINT, sender_id BIGINT, sender_name VARCHAR(100),
			sender_avatar VARCHAR(500), content TEXT, msg_type TINYINT, send_time DATETIME
		)`,
		`CREATE TABLE sys_notice (
			id BIGINT PRIMARY KEY, title VARCHAR(200), content TEXT, notice_type TINYINT,
			channels VARCHAR(200), target_type TINYINT, target_ids VARCHAR(500), status TINYINT,
			create_by BIGINT, create_name VARCHAR(100), create_time DATETIME, update_time DATETIME, deleted TINYINT
		)`,
		`CREATE TABLE sys_user_notice (
			id BIGINT PRIMARY KEY, user_id BIGINT, notice_id BIGINT, is_read TINYINT, read_time DATETIME
		)`,
		`CREATE TABLE sys_notice_send_log (
			id BIGINT PRIMARY KEY, notice_id BIGINT, channel VARCHAR(50), status TINYINT,
			target_count INT, success_count INT, error_msg VARCHAR(500), send_time DATETIME
		)`,
	}
	for _, statement := range schema {
		_, err := db.ExecContext(ctx, statement)
		require.NoError(t, err)
	}

	now := time.Now().UTC()
	statements := []struct {
		query string
		args  []any
	}{
		{`INSERT INTO sys_dept VALUES (1, 0, '0', 'Legacy HQ', 1, 'Lead', '', 'hq@example.invalid', 1, ?, ?, 0)`, []any{now, now}},
		{`INSERT INTO sys_post VALUES (1, 0, 'engineer', 'Engineer', 1, 1, '', ?, ?, 0)`, []any{now, now}},
		{`INSERT INTO sys_role VALUES (1, 'Admin', 'admin', 1, 1, 'legacy', 1, ?, ?, 0)`, []any{now, now}},
		{`INSERT INTO sys_role VALUES (2, 'User', 'user', 2, 1, '', 5, ?, ?, 0)`, []any{now, now}},
		{`INSERT INTO sys_user VALUES (1, 1, 'legacyadmin', ?, 'Legacy Admin', '', 'legacy@example.invalid', '13800138000', 1, 1, 0, '', ?, ?, 'admin', '', 0)`, []any{string(passwordHash), now, now}},
		{`INSERT INTO sys_menu VALUES (1, 0, 'System', 1, '/system', 'Layout', '', 'Settings', 1, 1, 1, 0, ?, ?, 0)`, []any{now, now}},
		{`INSERT INTO sys_menu VALUES (2, 1, 'Users', 2, 'system/user', 'system/user/index', 'user:read', 'People', 1, 1, 1, 0, ?, ?, 0)`, []any{now, now}},
		{`INSERT INTO sys_menu VALUES (3, 2, 'Create user', 3, '', '', 'user:create', '', 1, 0, 1, 0, ?, ?, 0)`, []any{now, now}},
		{`INSERT INTO sys_user_role VALUES (1, 1)`, nil},
		{`INSERT INTO sys_role_menu VALUES (1, 1)`, nil},
		{`INSERT INTO sys_role_menu VALUES (1, 2)`, nil},
		{`INSERT INTO sys_role_menu VALUES (1, 3)`, nil},
		{`INSERT INTO sys_role_dept VALUES (1, 1)`, nil},
		{`INSERT INTO sys_user_post VALUES (1, 1)`, nil},
		{`INSERT INTO sys_dict_type VALUES (1, 'Status', 'sys_status', 1, '', 0)`, nil},
		{`INSERT INTO sys_dict_data VALUES (1, 1, 'Normal', '1', 'sys_status', '', 'success', 0, 1, 0)`, nil},
		{`INSERT INTO sys_config_group VALUES (1, 'system', 'System', '{"siteName":"Legacy","smtp":{"password":"exposed","host":"smtp.example.invalid"}}', 1, 1)`, nil},
		{`INSERT INTO sys_file_group VALUES (1, 'Legacy files', 1, ?)`, []any{now}},
		{`INSERT INTO sys_file VALUES (1, 'integration.txt', 'integration.txt', '2026/01/integration.txt',
			'/api/files/2026/01/integration.txt', 16, 'text/plain', '.txt', 'local', 1, '1', ?)`, []any{now}},
		{`INSERT INTO sys_chat_group VALUES (1, 'Legacy group', '', 1, '', 200, 1, ?, ?)`, []any{now, now}},
		{`INSERT INTO sys_chat_group_member VALUES (1, 1, 1, NULL, 2, 0, ?)`, []any{now}},
		{`INSERT INTO sys_chat_message VALUES (1, 1, 'Legacy Admin', '', 1, 'hello direct', 1, 1, ?)`, []any{now}},
		{`INSERT INTO sys_chat_group_message VALUES (1, 1, 1, 'Legacy Admin', '', 'hello group', 1, ?)`, []any{now}},
		{`INSERT INTO sys_notice VALUES (1, 'Legacy notice', 'Imported notice body', 1, '["station"]', 3, NULL, 1, 1, 'Legacy Admin', ?, ?, 0)`, []any{now, now}},
		{`INSERT INTO sys_user_notice VALUES (1, 1, 1, 1, ?)`, []any{now}},
		{`INSERT INTO sys_notice_send_log VALUES (1, 1, 'station', 1, 1, 1, '', ?)`, []any{now}},
	}
	for _, statement := range statements {
		_, err := db.ExecContext(ctx, statement.query, statement.args...)
		require.NoError(t, err)
	}
}
