package legacy

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Options struct {
	MySQLDSN          string
	Postgres          *pgxpool.Pool
	SourceDescription string
	ReportPath        string
	LegacyUploadRoot  string
	TargetUploadRoot  string
}

type Report struct {
	RunID      uuid.UUID              `json:"runId"`
	Source     string                 `json:"source"`
	StartedAt  time.Time              `json:"startedAt"`
	FinishedAt *time.Time             `json:"finishedAt,omitempty"`
	Status     string                 `json:"status"`
	Tables     map[string]TableReport `json:"tables"`
	Validation map[string]int64       `json:"validation"`
	Warnings   []string               `json:"warnings"`
}

type TableReport struct {
	Read     int      `json:"read"`
	Inserted int      `json:"inserted"`
	Skipped  int      `json:"skipped"`
	Failed   int      `json:"failed"`
	Errors   []string `json:"errors,omitempty"`
}

type Importer struct {
	source *sql.DB
	target *pgxpool.Pool
	opts   Options
	report Report
}

type sourceRow map[string]any

type importStep struct {
	name string
	run  func(context.Context, pgx.Tx) error
}

var errLegacyReferenceMissing = errors.New("legacy reference is missing")

func Run(ctx context.Context, opts Options) (Report, error) {
	if opts.MySQLDSN == "" {
		return Report{}, errors.New("mysql DSN is required")
	}
	if opts.Postgres == nil {
		return Report{}, errors.New("PostgreSQL pool is required")
	}
	source, err := sql.Open("mysql", opts.MySQLDSN)
	if err != nil {
		return Report{}, fmt.Errorf("open MySQL source: %w", err)
	}
	defer func() { _ = source.Close() }()
	source.SetMaxOpenConns(4)
	source.SetMaxIdleConns(2)
	source.SetConnMaxLifetime(10 * time.Minute)
	if err := source.PingContext(ctx); err != nil {
		return Report{}, fmt.Errorf("ping MySQL source: %w", err)
	}

	now := time.Now().UTC()
	importer := &Importer{
		source: source,
		target: opts.Postgres,
		opts:   opts,
		report: Report{
			RunID: mustID(), Source: opts.SourceDescription, StartedAt: now, Status: "running",
			Tables: make(map[string]TableReport), Validation: make(map[string]int64),
			Warnings: []string{
				"Legacy file objects are not copied by this importer; file metadata import is skipped.",
				"Sensitive provider configuration values are redacted before settings import.",
			},
		},
	}
	if _, err := importer.target.Exec(ctx, `
		INSERT INTO legacy_import_runs (id, source_description, status, started_at, report)
		VALUES ($1, $2, 'running', $3, '{}'::jsonb)`,
		importer.report.RunID, importer.report.Source, now,
	); err != nil {
		return Report{}, fmt.Errorf("create import run: %w", err)
	}

	steps := []importStep{
		{"identity_users", importer.importUsers},
		{"organization_departments", importer.importDepartments},
		{"organization_positions", importer.importPositions},
		{"access_roles", importer.importRoles},
		{"access_menus", importer.importMenus},
		{"access_permissions", importer.importPermissions},
		{"access_role_permissions", importer.importRolePermissions},
		{"access_user_roles", importer.importUserRoles},
		{"access_role_menus", importer.importRoleMenus},
		{"access_role_departments", importer.importRoleDepartments},
		{"organization_memberships", importer.importUserDepartments},
		{"organization_user_positions", importer.importUserPositions},
		{"system_dictionary_types", importer.importDictionaryTypes},
		{"system_dictionary_items", importer.importDictionaryItems},
		{"system_settings", importer.importSettings},
		{"storage_file_groups", importer.importFileGroups},
		{"storage_files", importer.importFiles},
		{"messaging_groups", importer.importChatGroups},
		{"messaging_group_members", importer.importChatGroupMembers},
		{"messaging_messages", importer.importChatMessages},
		{"notification_notices", importer.importNotices},
		{"notification_recipients", importer.importNoticeRecipients},
		{"notification_send_logs", importer.importNoticeSendLogs},
	}
	for _, step := range steps {
		if ctx.Err() != nil {
			break
		}
		tx, err := importer.target.Begin(ctx)
		if err != nil {
			return importer.fail(ctx, err)
		}
		table := importer.report.Tables[step.name]
		if table.Errors == nil {
			table.Errors = make([]string, 0)
		}
		importer.report.Tables[step.name] = table
		err = step.run(ctx, tx)
		if err != nil {
			_ = tx.Rollback(ctx)
			table = importer.report.Tables[step.name]
			table.Failed++
			table.Errors = append(table.Errors, err.Error())
			importer.report.Tables[step.name] = table
			continue
		}
		if err := tx.Commit(ctx); err != nil {
			_ = tx.Rollback(ctx)
			table = importer.report.Tables[step.name]
			table.Failed++
			table.Errors = append(table.Errors, err.Error())
			importer.report.Tables[step.name] = table
		}
	}
	if err := importer.validate(ctx); err != nil {
		importer.report.Warnings = append(importer.report.Warnings, "validation query failed: "+err.Error())
	}
	finished := time.Now().UTC()
	importer.report.FinishedAt = &finished
	importer.report.Status = "completed"
	for _, table := range importer.report.Tables {
		if table.Failed > 0 {
			importer.report.Status = "failed"
			break
		}
	}
	encoded, err := json.MarshalIndent(importer.report, "", "  ")
	if err != nil {
		return importer.report, err
	}
	if err := importer.persistReport(ctx, encoded); err != nil {
		return importer.report, err
	}
	if opts.ReportPath != "" {
		if err := os.MkdirAll(filepath.Dir(opts.ReportPath), 0o750); err != nil {
			return importer.report, err
		}
		if err := os.WriteFile(opts.ReportPath, encoded, 0o600); err != nil {
			return importer.report, err
		}
	}
	if importer.report.Status != "completed" {
		return importer.report, fmt.Errorf("legacy import completed with row failures")
	}
	return importer.report, nil
}

func (i *Importer) fail(ctx context.Context, cause error) (Report, error) {
	finished := time.Now().UTC()
	i.report.FinishedAt = &finished
	i.report.Status = "failed"
	encoded, _ := json.Marshal(i.report)
	_, _ = i.target.Exec(ctx, `
		UPDATE legacy_import_runs SET status = 'failed', finished_at = $2, report = $3 WHERE id = $1`,
		i.report.RunID, finished, encoded,
	)
	return i.report, cause
}

func (i *Importer) persistReport(ctx context.Context, encoded []byte) error {
	_, err := i.target.Exec(ctx, `
		UPDATE legacy_import_runs
		SET status = $2, finished_at = $3, report = $4
		WHERE id = $1`,
		i.report.RunID, i.report.Status, i.report.FinishedAt, encoded,
	)
	return err
}

func (i *Importer) importUsers(ctx context.Context, tx pgx.Tx) error {
	rows, err := i.readTable(ctx, `
		SELECT id, dept_id, username, password, nickname, avatar, email, phone, gender,
		       status, is_quit, remark, create_time, update_time, user_type, open_id, deleted
		FROM sys_user ORDER BY id`)
	if err != nil {
		return err
	}
	table := i.report.Tables["identity_users"]
	table.Read = len(rows)
	for _, row := range rows {
		if boolInt(row, "deleted") == 1 {
			table.Skipped++
			continue
		}
		err := i.withSavepoint(ctx, tx, func() error {
			oldID := text(row, "id")
			existing, err := i.findUserByUsername(ctx, tx, text(row, "username"))
			if err == nil {
				if err := i.createMapping(ctx, tx, "identity_users", oldID, existing); err != nil {
					return err
				}
				table.Skipped++
				return nil
			}
			if !errors.Is(err, pgx.ErrNoRows) {
				return err
			}
			userID, exists, err := i.getOrCreateMapping(ctx, tx, "identity_users", oldID)
			if err != nil {
				return err
			}
			if exists {
				table.Skipped++
				return nil
			}
			created := timestamp(row, "create_time")
			updated := timestamp(row, "update_time")
			status := legacyUserStatus(row)
			userType := text(row, "user_type")
			if userType != "admin" && userType != "pc" && userType != "app" {
				userType = "admin"
			}
			displayName := text(row, "nickname")
			if displayName == "" {
				displayName = text(row, "username")
			}
			_, err = tx.Exec(ctx, `
				INSERT INTO identity_users (
					id, username, password_hash, display_name, email, phone, avatar_url,
					user_type, status, gender, created_at, updated_at
				) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
				userID, text(row, "username"), text(row, "password"), displayName,
				optionalText(row, "email"), optionalText(row, "phone"), text(row, "avatar"),
				userType, status, int16Value(row, "gender"), created, updated,
			)
			if err != nil {
				return err
			}
			if openID := text(row, "open_id"); openID != "" {
				_, err = tx.Exec(ctx, `
					INSERT INTO identity_social_identities (id, user_id, provider, subject)
					VALUES ($1, $2, 'wechat_mp', $3)
					ON CONFLICT (provider, subject) DO NOTHING`,
					mustID(), userID, openID,
				)
			}
			if err == nil {
				table.Inserted++
			}
			return err
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				table.Skipped++
				continue
			}
			table.Failed++
			table.Errors = appendReportError(table.Errors, err)
		}
	}
	i.report.Tables["identity_users"] = table
	return nil
}

func (i *Importer) importDepartments(ctx context.Context, tx pgx.Tx) error {
	rows, err := i.readTable(ctx, `
		SELECT id, parent_id, ancestors, dept_name, sort, leader, phone, email, status,
		       create_time, update_time, deleted
		FROM sys_dept ORDER BY LENGTH(ancestors), id`)
	if err != nil {
		return err
	}
	table := i.report.Tables["organization_departments"]
	table.Read = len(rows)
	for _, row := range rows {
		if boolInt(row, "deleted") == 1 {
			table.Skipped++
			continue
		}
		err := i.withSavepoint(ctx, tx, func() error {
			oldID := text(row, "id")
			departmentID, exists, err := i.getOrCreateMapping(ctx, tx, "organization_departments", oldID)
			if err != nil {
				return err
			}
			if exists {
				table.Skipped++
				return nil
			}
			var parentID pgtype.UUID
			path := "dept_" + sanitizeLTREE(oldID)
			legacyParent := int64Value(row, "parent_id")
			if legacyParent > 0 {
				mappedParent, err := i.lookupMapping(ctx, tx, "organization_departments", strconv.FormatInt(legacyParent, 10))
				if err != nil {
					return fmt.Errorf("parent department %d: %w", legacyParent, err)
				}
				parentID = pgUUID(mappedParent)
				var parentPath string
				if err := tx.QueryRow(ctx, `SELECT path::text FROM organization_departments WHERE id = $1`, mappedParent).Scan(&parentPath); err != nil {
					return err
				}
				path = parentPath + "." + sanitizeLTREE(oldID)
			}
			_, err = tx.Exec(ctx, `
				INSERT INTO organization_departments (
					id, parent_id, name, code, path, leader_name, phone, email, sort_order,
					status, created_at, updated_at
				) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
				departmentID, parentID, text(row, "dept_name"), "legacy_dept_"+oldID, path,
				text(row, "leader"), text(row, "phone"), optionalText(row, "email"),
				int32Value(row, "sort"), statusValue(row), timestamp(row, "create_time"), timestamp(row, "update_time"),
			)
			if err == nil {
				table.Inserted++
			}
			return err
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				table.Skipped++
				continue
			}
			table.Failed++
			table.Errors = appendReportError(table.Errors, err)
		}
	}
	i.report.Tables["organization_departments"] = table
	return nil
}

func (i *Importer) importPositions(ctx context.Context, tx pgx.Tx) error {
	rows, err := i.readTable(ctx, `
		SELECT id, parent_id, post_code, post_name, sort, status, remark, create_time, update_time, deleted
		FROM sys_post ORDER BY id`)
	if err != nil {
		return err
	}
	table := i.report.Tables["organization_positions"]
	table.Read = len(rows)
	for _, row := range rows {
		if boolInt(row, "deleted") == 1 {
			table.Skipped++
			continue
		}
		err := i.withSavepoint(ctx, tx, func() error {
			oldID := text(row, "id")
			positionID, exists, err := i.getOrCreateMapping(ctx, tx, "organization_positions", oldID)
			if err != nil {
				return err
			}
			if exists {
				table.Skipped++
				return nil
			}
			var parentID pgtype.UUID
			if legacyParent := int64Value(row, "parent_id"); legacyParent > 0 {
				mapped, err := i.lookupMapping(ctx, tx, "organization_positions", strconv.FormatInt(legacyParent, 10))
				switch {
				case errors.Is(err, pgx.ErrNoRows):
					i.addWarning(fmt.Sprintf("position %s references missing parent %d; imported as root", text(row, "id"), legacyParent))
				case err != nil:
					return err
				default:
					parentID = pgUUID(mapped)
				}
			}
			_, err = tx.Exec(ctx, `
				INSERT INTO organization_positions (
					id, parent_id, code, name, sort_order, status, description, created_at, updated_at
				) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
				positionID, parentID, text(row, "post_code"), text(row, "post_name"),
				int32Value(row, "sort"), statusValue(row), text(row, "remark"),
				timestamp(row, "create_time"), timestamp(row, "update_time"),
			)
			if err == nil {
				table.Inserted++
			}
			return err
		})
		if err != nil {
			table.Failed++
			table.Errors = appendReportError(table.Errors, err)
		}
	}
	i.report.Tables["organization_positions"] = table
	return nil
}

func (i *Importer) importRoles(ctx context.Context, tx pgx.Tx) error {
	rows, err := i.readTable(ctx, `
		SELECT id, name, code, sort, status, remark, data_scope, create_time, update_time, deleted
		FROM sys_role ORDER BY id`)
	if err != nil {
		return err
	}
	table := i.report.Tables["access_roles"]
	table.Read = len(rows)
	for _, row := range rows {
		if boolInt(row, "deleted") == 1 {
			table.Skipped++
			continue
		}
		err := i.withSavepoint(ctx, tx, func() error {
			oldID := text(row, "id")
			var existingRoleID uuid.UUID
			lookupErr := tx.QueryRow(ctx, `
				SELECT id FROM access_roles WHERE code = $1 AND deleted_at IS NULL`,
				text(row, "code"),
			).Scan(&existingRoleID)
			if lookupErr == nil {
				if err := i.createMapping(ctx, tx, "access_roles", oldID, existingRoleID); err != nil {
					return err
				}
				table.Skipped++
				return nil
			}
			if !errors.Is(lookupErr, pgx.ErrNoRows) {
				return lookupErr
			}
			roleID, exists, err := i.getOrCreateMapping(ctx, tx, "access_roles", oldID)
			if err != nil {
				return err
			}
			if exists {
				table.Skipped++
				return nil
			}
			_, err = tx.Exec(ctx, `
				INSERT INTO access_roles (
					id, name, code, description, is_super_admin, data_scope, status,
					created_at, updated_at
				) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
				roleID, text(row, "name"), text(row, "code"), text(row, "remark"),
				text(row, "code") == "admin", legacyDataScope(int32Value(row, "data_scope")),
				statusValue(row), timestamp(row, "create_time"), timestamp(row, "update_time"),
			)
			if err == nil {
				table.Inserted++
			}
			return err
		})
		if err != nil {
			table.Failed++
			table.Errors = appendReportError(table.Errors, err)
		}
	}
	i.report.Tables["access_roles"] = table
	return nil
}

func (i *Importer) importMenus(ctx context.Context, tx pgx.Tx) error {
	rows, err := i.readTable(ctx, `
		SELECT id, parent_id, name, type, path, component, permission, icon, sort,
		       visible, status, is_frame, create_time, update_time, deleted
		FROM sys_menu ORDER BY sort, id`)
	if err != nil {
		return err
	}
	table := i.report.Tables["access_menus"]
	table.Read = len(rows)
	pending := rows
	for pass := 0; pass <= len(rows) && len(pending) > 0; pass++ {
		next := make([]sourceRow, 0)
		for _, row := range pending {
			if boolInt(row, "deleted") == 1 {
				table.Skipped++
				continue
			}
			oldID := text(row, "id")
			_, exists, err := i.lookupOrCreateMappingExists(ctx, tx, "access_menus", oldID)
			if err != nil {
				table.Failed++
				table.Errors = appendReportError(table.Errors, err)
				continue
			}
			if exists {
				table.Skipped++
				continue
			}
			var parentID pgtype.UUID
			legacyParent := int64Value(row, "parent_id")
			if legacyParent > 0 {
				mapped, err := i.lookupMapping(ctx, tx, "access_menus", strconv.FormatInt(legacyParent, 10))
				if errors.Is(err, pgx.ErrNoRows) {
					next = append(next, row)
					continue
				}
				if err != nil {
					table.Failed++
					table.Errors = appendReportError(table.Errors, err)
					continue
				}
				parentID = pgUUID(mapped)
			}
			err = i.withSavepoint(ctx, tx, func() error {
				menuID, _, err := i.getOrCreateMapping(ctx, tx, "access_menus", oldID)
				if err != nil {
					return err
				}
				externalURL := ""
				if boolInt(row, "is_frame") == 1 {
					externalURL = text(row, "component")
				}
				_, err = tx.Exec(ctx, `
					INSERT INTO access_menus (
						id, parent_id, name, menu_type, route_path, component, permission_code,
						icon, sort_order, visible, external_url, status, created_at, updated_at
					) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`,
					menuID, parentID, text(row, "name"), int16Value(row, "type"),
					optionalText(row, "path"), optionalText(row, "component"), optionalText(row, "permission"),
					text(row, "icon"), int32Value(row, "sort"), boolInt(row, "visible") == 1,
					optionalTextValue(externalURL), activeStatus(int32Value(row, "status")),
					timestamp(row, "create_time"), timestamp(row, "update_time"),
				)
				return err
			})
			if err != nil {
				table.Failed++
				table.Errors = appendReportError(table.Errors, err)
				continue
			}
			table.Inserted++
		}
		if len(next) == len(pending) {
			for _, row := range next {
				table.Skipped++
				i.addWarning(fmt.Sprintf("menu %s references an unresolved parent and was skipped", text(row, "id")))
			}
			break
		}
		pending = next
	}
	i.report.Tables["access_menus"] = table
	return nil
}

func (i *Importer) importPermissions(ctx context.Context, tx pgx.Tx) error {
	rows, err := i.readTable(ctx, `
		SELECT DISTINCT permission FROM sys_menu
		WHERE permission IS NOT NULL AND permission <> '' AND deleted = 0`)
	if err != nil {
		return err
	}
	table := i.report.Tables["access_permissions"]
	table.Read = len(rows)
	for _, row := range rows {
		code := text(row, "permission")
		tag, err := tx.Exec(ctx, `
			INSERT INTO access_permissions (id, code, name, module, description)
			VALUES ($1, $2, $2, $3, 'Imported legacy permission')
			ON CONFLICT (code) DO NOTHING`,
			mustID(), code, permissionModule(code),
		)
		if err != nil {
			table.Failed++
			table.Errors = appendReportError(table.Errors, err)
			continue
		}
		if tag.RowsAffected() == 0 {
			table.Skipped++
		} else {
			table.Inserted++
		}
	}
	i.report.Tables["access_permissions"] = table
	return nil
}

func (i *Importer) importRolePermissions(ctx context.Context, tx pgx.Tx) error {
	rows, err := i.readTable(ctx, `
		SELECT DISTINCT rm.role_id, m.permission
		FROM sys_role_menu rm
		JOIN sys_menu m ON m.id = rm.menu_id
		WHERE m.permission IS NOT NULL AND m.permission <> '' AND m.deleted = 0
		ORDER BY rm.role_id, m.permission`)
	if err != nil {
		return err
	}
	return i.importMappings(ctx, tx, "access_role_permissions", rows, func(row sourceRow) (string, string, error) {
		roleID, err := i.lookupMapping(ctx, tx, "access_roles", text(row, "role_id"))
		if err != nil {
			return "", "", err
		}
		var permissionID uuid.UUID
		if err := tx.QueryRow(ctx, `SELECT id FROM access_permissions WHERE code = $1`, text(row, "permission")).Scan(&permissionID); err != nil {
			return "", "", err
		}
		return roleID.String(), permissionID.String(), nil
	}, `
		INSERT INTO access_role_permissions (role_id, permission_id) VALUES ($1, $2)
		ON CONFLICT DO NOTHING`)
}

func (i *Importer) importUserRoles(ctx context.Context, tx pgx.Tx) error {
	rows, err := i.readTable(ctx, `SELECT user_id, role_id FROM sys_user_role ORDER BY user_id, role_id`)
	if err != nil {
		return err
	}
	return i.importMappings(ctx, tx, "access_user_roles", rows, func(row sourceRow) (string, string, error) {
		userID, err := i.lookupMapping(ctx, tx, "identity_users", text(row, "user_id"))
		if err != nil {
			return "", "", err
		}
		roleID, err := i.lookupMapping(ctx, tx, "access_roles", text(row, "role_id"))
		if err != nil {
			return "", "", err
		}
		return userID.String(), roleID.String(), nil
	}, `
		INSERT INTO access_user_roles (user_id, role_id) VALUES ($1, $2)
		ON CONFLICT DO NOTHING`)
}

func (i *Importer) importRoleMenus(ctx context.Context, tx pgx.Tx) error {
	rows, err := i.readTable(ctx, `SELECT role_id, menu_id FROM sys_role_menu ORDER BY role_id, menu_id`)
	if err != nil {
		return err
	}
	return i.importMappings(ctx, tx, "access_role_menus", rows, func(row sourceRow) (string, string, error) {
		roleID, err := i.lookupMapping(ctx, tx, "access_roles", text(row, "role_id"))
		if err != nil {
			return "", "", err
		}
		menuID, err := i.lookupMapping(ctx, tx, "access_menus", text(row, "menu_id"))
		if err != nil {
			return "", "", err
		}
		return roleID.String(), menuID.String(), nil
	}, `
		INSERT INTO access_role_menus (role_id, menu_id) VALUES ($1, $2)
		ON CONFLICT DO NOTHING`)
}

func (i *Importer) importRoleDepartments(ctx context.Context, tx pgx.Tx) error {
	rows, err := i.readTable(ctx, `SELECT role_id, dept_id FROM sys_role_dept ORDER BY role_id, dept_id`)
	if err != nil {
		return err
	}
	return i.importMappings(ctx, tx, "access_role_departments", rows, func(row sourceRow) (string, string, error) {
		roleID, err := i.lookupMapping(ctx, tx, "access_roles", text(row, "role_id"))
		if err != nil {
			return "", "", err
		}
		departmentID, err := i.lookupMapping(ctx, tx, "organization_departments", text(row, "dept_id"))
		if err != nil {
			return "", "", err
		}
		return roleID.String(), departmentID.String(), nil
	}, `
		INSERT INTO access_role_departments (role_id, department_id) VALUES ($1, $2)
		ON CONFLICT DO NOTHING`)
}

func (i *Importer) importUserDepartments(ctx context.Context, tx pgx.Tx) error {
	rows, err := i.readTable(ctx, `SELECT id AS user_id, dept_id FROM sys_user WHERE dept_id IS NOT NULL AND deleted = 0`)
	if err != nil {
		return err
	}
	return i.importMappings(ctx, tx, "organization_memberships", rows, func(row sourceRow) (string, string, error) {
		userID, err := i.lookupMapping(ctx, tx, "identity_users", text(row, "user_id"))
		if err != nil {
			return "", "", err
		}
		departmentID, err := i.lookupMapping(ctx, tx, "organization_departments", text(row, "dept_id"))
		if err != nil {
			return "", "", err
		}
		return userID.String(), departmentID.String(), nil
	}, `
		INSERT INTO organization_memberships (user_id, department_id, is_primary)
		VALUES ($1, $2, true)
		ON CONFLICT (user_id, department_id) DO NOTHING`)
}

func (i *Importer) importUserPositions(ctx context.Context, tx pgx.Tx) error {
	rows, err := i.readTable(ctx, `SELECT user_id, post_id FROM sys_user_post ORDER BY user_id, post_id`)
	if err != nil {
		return err
	}
	return i.importMappings(ctx, tx, "organization_user_positions", rows, func(row sourceRow) (string, string, error) {
		userID, err := i.lookupMapping(ctx, tx, "identity_users", text(row, "user_id"))
		if err != nil {
			return "", "", err
		}
		positionID, err := i.lookupMapping(ctx, tx, "organization_positions", text(row, "post_id"))
		if err != nil {
			return "", "", err
		}
		return userID.String(), positionID.String(), nil
	}, `
		INSERT INTO organization_user_positions (user_id, position_id) VALUES ($1, $2)
		ON CONFLICT DO NOTHING`)
}

func (i *Importer) importDictionaryTypes(ctx context.Context, tx pgx.Tx) error {
	rows, err := i.readTable(ctx, `
		SELECT id, dict_name, dict_type, status, remark FROM sys_dict_type WHERE deleted = 0 ORDER BY id`)
	if err != nil {
		return err
	}
	table := i.report.Tables["system_dictionary_types"]
	table.Read = len(rows)
	for _, row := range rows {
		err := i.withSavepoint(ctx, tx, func() error {
			oldID := text(row, "id")
			id, exists, err := i.getOrCreateMapping(ctx, tx, "system_dictionary_types", oldID)
			if err != nil {
				return err
			}
			if exists {
				table.Skipped++
				return nil
			}
			_, err = tx.Exec(ctx, `
				INSERT INTO system_dictionary_types (id, code, name, description, status)
				VALUES ($1, $2, $3, $4, $5)`,
				id, text(row, "dict_type"), text(row, "dict_name"), text(row, "remark"), statusValue(row),
			)
			if err == nil {
				table.Inserted++
			}
			return err
		})
		if err != nil {
			table.Failed++
			table.Errors = appendReportError(table.Errors, err)
		}
	}
	i.report.Tables["system_dictionary_types"] = table
	return nil
}

func (i *Importer) importDictionaryItems(ctx context.Context, tx pgx.Tx) error {
	rows, err := i.readTable(ctx, `
		SELECT id, sort, dict_label, dict_value, dict_type, css_class, list_class,
		       is_default, status FROM sys_dict_data WHERE deleted = 0 ORDER BY dict_type, sort, id`)
	if err != nil {
		return err
	}
	table := i.report.Tables["system_dictionary_items"]
	table.Read = len(rows)
	for _, row := range rows {
		err := i.withSavepoint(ctx, tx, func() error {
			oldID := text(row, "id")
			itemID, exists, err := i.getOrCreateMapping(ctx, tx, "system_dictionary_items", oldID)
			if err != nil {
				return err
			}
			if exists {
				table.Skipped++
				return nil
			}
			var typeID uuid.UUID
			if err := tx.QueryRow(ctx, `SELECT id FROM system_dictionary_types WHERE code = $1`, text(row, "dict_type")).Scan(&typeID); err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return errLegacyReferenceMissing
				}
				return err
			}
			tagStyle := text(row, "list_class")
			if tagStyle == "" {
				tagStyle = text(row, "css_class")
			}
			if tagStyle == "" {
				tagStyle = "default"
			}
			_, err = tx.Exec(ctx, `
				INSERT INTO system_dictionary_items (
					id, type_id, value, label, sort_order, tag_style, is_default, status
				) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
				itemID, typeID, text(row, "dict_value"), text(row, "dict_label"),
				int32Value(row, "sort"), tagStyle, boolInt(row, "is_default") == 1, statusValue(row),
			)
			if err == nil {
				table.Inserted++
			}
			return err
		})
		if err != nil {
			if errors.Is(err, errLegacyReferenceMissing) {
				table.Skipped++
			} else {
				table.Failed++
				table.Errors = appendReportError(table.Errors, err)
			}
		}
	}
	i.report.Tables["system_dictionary_items"] = table
	return nil
}

func (i *Importer) importSettings(ctx context.Context, tx pgx.Tx) error {
	rows, err := i.readTable(ctx, `
		SELECT id, group_code, group_name, config_value FROM sys_config_group WHERE status = 1 ORDER BY sort, id`)
	if err != nil {
		return err
	}
	table := i.report.Tables["system_settings"]
	table.Read = len(rows)
	for _, row := range rows {
		err := i.withSavepoint(ctx, tx, func() error {
			groupCode := text(row, "group_code")
			var groupID uuid.UUID
			groupErr := tx.QueryRow(ctx, `SELECT id FROM system_setting_groups WHERE code = $1`, groupCode).Scan(&groupID)
			if errors.Is(groupErr, pgx.ErrNoRows) {
				groupID = mustID()
				_, groupErr = tx.Exec(ctx, `
					INSERT INTO system_setting_groups (id, code, name, description, sort_order, status)
					VALUES ($1, $2, $3, 'Imported legacy configuration group', $4, 'active')`,
					groupID, groupCode, text(row, "group_name"), int32Value(row, "sort"),
				)
			}
			if groupErr != nil {
				return groupErr
			}
			var value any
			raw := []byte(text(row, "config_value"))
			if len(raw) > 0 {
				if err := json.Unmarshal(raw, &value); err != nil {
					value = map[string]any{"unparsed": string(raw)}
				}
			}
			value = sanitizeLegacyConfig(value)
			encoded, _ := json.Marshal(value)
			tag, err := tx.Exec(ctx, `
				INSERT INTO system_settings (
					id, group_id, key, value_type, value_text, is_sensitive, validation, version
				) VALUES ($1, $2, $3, 'json', $4, false, '{}'::jsonb, 1)
				ON CONFLICT (group_id, key) DO NOTHING`,
				mustID(), groupID, "legacy."+groupCode, string(encoded),
			)
			if err == nil {
				if tag.RowsAffected() == 0 {
					table.Skipped++
				} else {
					table.Inserted++
				}
			}
			return err
		})
		if err != nil {
			table.Failed++
			table.Errors = appendReportError(table.Errors, err)
		}
	}
	i.report.Tables["system_settings"] = table
	return nil
}

func (i *Importer) importFileGroups(ctx context.Context, tx pgx.Tx) error {
	rows, err := i.readTable(ctx, `
		SELECT id, name, sort, create_time FROM sys_file_group ORDER BY sort, id`)
	if err != nil {
		return err
	}
	table := i.report.Tables["storage_file_groups"]
	table.Read = len(rows)
	for _, row := range rows {
		err := i.withSavepoint(ctx, tx, func() error {
			oldID := text(row, "id")
			id, exists, err := i.getOrCreateMapping(ctx, tx, "storage_file_groups", oldID)
			if err != nil {
				return err
			}
			if exists {
				table.Skipped++
				return nil
			}
			_, err = tx.Exec(ctx, `
				INSERT INTO storage_file_groups (id, name, sort_order, created_at)
				VALUES ($1, $2, $3, $4)`,
				id, text(row, "name"), int32Value(row, "sort"), timestamp(row, "create_time"),
			)
			if err == nil {
				table.Inserted++
			}
			return err
		})
		if err != nil {
			table.Failed++
			table.Errors = appendReportError(table.Errors, err)
		}
	}
	i.report.Tables["storage_file_groups"] = table
	return nil
}

func (i *Importer) importFiles(ctx context.Context, tx pgx.Tx) error {
	rows, err := i.readTable(ctx, `
		SELECT id, original_name, file_name, file_path, url, file_size, file_type, file_suffix,
		       storage_type, group_id, create_by, create_time
		FROM sys_file ORDER BY id`)
	if err != nil {
		return err
	}
	table := i.report.Tables["storage_files"]
	table.Read = len(rows)
	if i.opts.LegacyUploadRoot == "" || i.opts.TargetUploadRoot == "" {
		table.Skipped = len(rows)
		table.Errors = append(table.Errors, "file metadata import skipped: legacy and target upload roots are not configured")
		i.report.Tables["storage_files"] = table
		return nil
	}
	for _, row := range rows {
		if text(row, "storage_type") != "" && text(row, "storage_type") != "local" {
			table.Skipped++
			continue
		}
		var copiedPath string
		err := i.withSavepoint(ctx, tx, func() error {
			oldID := text(row, "id")
			fileID, exists, err := i.getOrCreateMapping(ctx, tx, "storage_files", oldID)
			if err != nil {
				return err
			}
			if exists {
				table.Skipped++
				return nil
			}
			relativePath := filepath.Clean(filepath.FromSlash(text(row, "file_path")))
			sourcePath, err := secureJoin(i.opts.LegacyUploadRoot, relativePath)
			if err != nil {
				return err
			}
			targetPath, err := secureJoin(i.opts.TargetUploadRoot, relativePath)
			if err != nil {
				return err
			}
			hash, size, err := copyFile(sourcePath, targetPath)
			if err != nil {
				return err
			}
			copiedPath = targetPath
			var groupID pgtype.UUID
			if legacyGroup := int64Value(row, "group_id"); legacyGroup > 0 {
				mapped, err := i.lookupMapping(ctx, tx, "storage_file_groups", strconv.FormatInt(legacyGroup, 10))
				if err != nil {
					return err
				}
				groupID = pgUUID(mapped)
			}
			var ownerID pgtype.UUID
			if legacyOwner := text(row, "create_by"); legacyOwner != "" {
				if mapped, err := i.lookupMapping(ctx, tx, "identity_users", legacyOwner); err == nil {
					ownerID = pgUUID(mapped)
				}
			}
			_, err = tx.Exec(ctx, `
				INSERT INTO storage_files (
					id, group_id, owner_user_id, original_name, storage_key, storage_provider,
					content_type, size_bytes, sha256_hash, visibility, status, created_at, updated_at
				) VALUES ($1, $2, $3, $4, $5, 'local', $6, $7, $8, 'private', 'active', $9, $9)`,
				fileID, groupID, ownerID, text(row, "original_name"), filepath.ToSlash(relativePath),
				text(row, "file_type"), size, hash, timestamp(row, "create_time"),
			)
			if err == nil {
				table.Inserted++
			}
			return err
		})
		if err != nil {
			if copiedPath != "" {
				_ = os.Remove(copiedPath)
			}
			if errors.Is(err, os.ErrNotExist) {
				table.Skipped++
				i.addWarning(fmt.Sprintf("legacy file %s is missing from the upload directory and was skipped", text(row, "id")))
			} else {
				table.Failed++
				table.Errors = appendReportError(table.Errors, err)
			}
		}
	}
	i.report.Tables["storage_files"] = table
	return nil
}

func (i *Importer) importChatGroups(ctx context.Context, tx pgx.Tx) error {
	rows, err := i.readTable(ctx, `
		SELECT id, name, avatar, owner_id, status, create_time, update_time
		FROM sys_chat_group ORDER BY id`)
	if err != nil {
		return err
	}
	table := i.report.Tables["messaging_groups"]
	table.Read = len(rows)
	for _, row := range rows {
		err := i.withSavepoint(ctx, tx, func() error {
			oldID := text(row, "id")
			groupID, exists, err := i.getOrCreateMapping(ctx, tx, "messaging_groups", oldID)
			if err != nil {
				return err
			}
			if exists {
				table.Skipped++
				return nil
			}
			ownerID, err := i.lookupMapping(ctx, tx, "identity_users", text(row, "owner_id"))
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					i.addWarning(fmt.Sprintf("chat group %s references a missing owner and was skipped", text(row, "id")))
					return errLegacyReferenceMissing
				}
				return err
			}
			status := "dissolved"
			var dissolvedAt pgtype.Timestamptz
			if int64Value(row, "status") == 1 {
				status = "active"
			} else {
				dissolvedAt = pgtype.Timestamptz{Time: timestamp(row, "update_time"), Valid: true}
			}
			_, err = tx.Exec(ctx, `
				INSERT INTO messaging_groups (
					id, name, avatar_url, owner_id, status, created_at, updated_at, dissolved_at
				) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
				groupID, text(row, "name"), text(row, "avatar"), ownerID, status,
				timestamp(row, "create_time"), timestamp(row, "update_time"), dissolvedAt,
			)
			if err == nil {
				table.Inserted++
			}
			return err
		})
		if err != nil {
			if errors.Is(err, errLegacyReferenceMissing) {
				table.Skipped++
			} else {
				table.Failed++
				table.Errors = appendReportError(table.Errors, err)
			}
		}
	}
	i.report.Tables["messaging_groups"] = table
	return nil
}

func (i *Importer) importChatGroupMembers(ctx context.Context, tx pgx.Tx) error {
	rows, err := i.readTable(ctx, `
		SELECT id, group_id, user_id, role, muted, join_time
		FROM sys_chat_group_member ORDER BY group_id, id`)
	if err != nil {
		return err
	}
	table := i.report.Tables["messaging_group_members"]
	table.Read = len(rows)
	for _, row := range rows {
		err := i.withSavepoint(ctx, tx, func() error {
			groupID, err := i.lookupMapping(ctx, tx, "messaging_groups", text(row, "group_id"))
			if err != nil {
				return err
			}
			userID, err := i.lookupMapping(ctx, tx, "identity_users", text(row, "user_id"))
			if err != nil {
				return err
			}
			role := "member"
			switch int32Value(row, "role") {
			case 2:
				role = "owner"
			case 1:
				role = "admin"
			}
			var mutedUntil pgtype.Timestamptz
			if boolInt(row, "muted") == 1 {
				mutedUntil = pgtype.Timestamptz{Time: time.Now().UTC().Add(24 * time.Hour), Valid: true}
			}
			tag, err := tx.Exec(ctx, `
				INSERT INTO messaging_group_members (group_id, user_id, member_role, muted_until, joined_at)
				VALUES ($1, $2, $3, $4, $5)
				ON CONFLICT (group_id, user_id) DO NOTHING`,
				groupID, userID, role, mutedUntil, timestamp(row, "join_time"),
			)
			if err == nil {
				if tag.RowsAffected() == 0 {
					table.Skipped++
				} else {
					table.Inserted++
				}
			}
			return err
		})
		if err != nil {
			table.Failed++
			table.Errors = appendReportError(table.Errors, err)
		}
	}
	i.report.Tables["messaging_group_members"] = table
	return nil
}

func (i *Importer) importChatMessages(ctx context.Context, tx pgx.Tx) error {
	directRows, err := i.readTable(ctx, `
		SELECT id, sender_id, receiver_id, content, msg_type, is_read, send_time
		FROM sys_chat_message ORDER BY id`)
	if err != nil {
		return err
	}
	table := i.report.Tables["messaging_messages"]
	table.Read += len(directRows)
	for _, row := range directRows {
		if int64Value(row, "receiver_id") <= 0 {
			table.Skipped++
			continue
		}
		err := i.withSavepoint(ctx, tx, func() error {
			oldID := text(row, "id")
			messageID, exists, err := i.getOrCreateMapping(ctx, tx, "messaging_messages", "direct_"+oldID)
			if err != nil {
				return err
			}
			if exists {
				table.Skipped++
				return nil
			}
			senderID, err := i.lookupMapping(ctx, tx, "identity_users", text(row, "sender_id"))
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return errLegacyReferenceMissing
				}
				return err
			}
			receiverID, err := i.lookupMapping(ctx, tx, "identity_users", text(row, "receiver_id"))
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return errLegacyReferenceMissing
				}
				return err
			}
			_, err = tx.Exec(ctx, `
				INSERT INTO messaging_messages (
					id, client_message_id, sender_id, receiver_id, message_type, content, created_at
				) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
				messageID, "legacy_direct_"+oldID, senderID, receiverID,
				legacyMessageType(int32Value(row, "msg_type")), text(row, "content"), timestamp(row, "send_time"),
			)
			if err != nil {
				return err
			}
			if boolInt(row, "is_read") == 1 {
				_, err = tx.Exec(ctx, `
					INSERT INTO messaging_read_receipts (message_id, user_id, read_at)
					VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`,
					messageID, receiverID, timestamp(row, "send_time"),
				)
			}
			if err == nil {
				table.Inserted++
			}
			return err
		})
		if err != nil {
			if errors.Is(err, errLegacyReferenceMissing) {
				table.Skipped++
			} else {
				table.Failed++
				table.Errors = appendReportError(table.Errors, err)
			}
		}
	}

	groupRows, err := i.readTable(ctx, `
		SELECT id, group_id, sender_id, content, msg_type, send_time
		FROM sys_chat_group_message ORDER BY id`)
	if err != nil {
		return err
	}
	table.Read += len(groupRows)
	for _, row := range groupRows {
		err := i.withSavepoint(ctx, tx, func() error {
			oldID := text(row, "id")
			messageID, exists, err := i.getOrCreateMapping(ctx, tx, "messaging_messages", "group_"+oldID)
			if err != nil {
				return err
			}
			if exists {
				table.Skipped++
				return nil
			}
			groupID, err := i.lookupMapping(ctx, tx, "messaging_groups", text(row, "group_id"))
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return errLegacyReferenceMissing
				}
				return err
			}
			var senderID uuid.UUID
			if int64Value(row, "sender_id") > 0 {
				senderID, err = i.lookupMapping(ctx, tx, "identity_users", text(row, "sender_id"))
				if err != nil {
					if errors.Is(err, pgx.ErrNoRows) {
						return errLegacyReferenceMissing
					}
					return err
				}
			} else {
				if err := tx.QueryRow(ctx, `SELECT owner_id FROM messaging_groups WHERE id = $1`, groupID).Scan(&senderID); err != nil {
					return err
				}
			}
			_, err = tx.Exec(ctx, `
				INSERT INTO messaging_messages (
					id, client_message_id, sender_id, group_id, message_type, content, metadata, created_at
				) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
				messageID, "legacy_group_"+oldID, senderID, groupID,
				legacyMessageType(int32Value(row, "msg_type")), text(row, "content"),
				[]byte(`{"legacySystem":`+strconv.FormatBool(int64Value(row, "sender_id") == 0)+`}`),
				timestamp(row, "send_time"),
			)
			if err == nil {
				table.Inserted++
			}
			return err
		})
		if err != nil {
			if errors.Is(err, errLegacyReferenceMissing) {
				table.Skipped++
			} else {
				table.Failed++
				table.Errors = appendReportError(table.Errors, err)
			}
		}
	}
	i.report.Tables["messaging_messages"] = table
	return nil
}

func (i *Importer) importNotices(ctx context.Context, tx pgx.Tx) error {
	rows, err := i.readTable(ctx, `
		SELECT id, title, content, notice_type, channels, target_type, target_ids, status,
		       create_by, create_time, update_time, deleted
		FROM sys_notice ORDER BY id`)
	if err != nil {
		return err
	}
	table := i.report.Tables["notification_notices"]
	table.Read = len(rows)
	for _, row := range rows {
		if boolInt(row, "deleted") == 1 {
			table.Skipped++
			continue
		}
		err := i.withSavepoint(ctx, tx, func() error {
			oldID := text(row, "id")
			noticeID, exists, err := i.getOrCreateMapping(ctx, tx, "notification_notices", oldID)
			if err != nil {
				return err
			}
			if exists {
				table.Skipped++
				return nil
			}
			channels := parseStringArray(text(row, "channels"))
			targetType, targetIDs := i.mapNoticeTarget(ctx, tx, int32Value(row, "target_type"), text(row, "target_ids"))
			channelsJSON, _ := json.Marshal(channels)
			targetsJSON, _ := json.Marshal(targetIDs)
			status := "draft"
			if int64Value(row, "status") == 1 {
				status = "published"
			}
			var createdBy pgtype.UUID
			if legacyCreator := text(row, "create_by"); legacyCreator != "" {
				if mapped, err := i.lookupMapping(ctx, tx, "identity_users", legacyCreator); err == nil {
					createdBy = pgUUID(mapped)
				}
			}
			_, err = tx.Exec(ctx, `
				INSERT INTO notification_notices (
					id, title, content, notice_type, channels, target_type, target_ids, status,
					created_by, created_at, updated_at, published_at
				) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
				noticeID, text(row, "title"), text(row, "content"), int16Value(row, "notice_type"),
				channelsJSON, targetType, targetsJSON, status, createdBy,
				timestamp(row, "create_time"), timestamp(row, "update_time"),
				pgtype.Timestamptz{Time: timestamp(row, "create_time"), Valid: status == "published"},
			)
			if err == nil {
				table.Inserted++
			}
			return err
		})
		if err != nil {
			table.Failed++
			table.Errors = appendReportError(table.Errors, err)
		}
	}
	i.report.Tables["notification_notices"] = table
	return nil
}

func (i *Importer) importNoticeRecipients(ctx context.Context, tx pgx.Tx) error {
	rows, err := i.readTable(ctx, `
		SELECT notice_id, user_id, is_read, read_time FROM sys_user_notice ORDER BY notice_id, user_id`)
	if err != nil {
		return err
	}
	table := i.report.Tables["notification_recipients"]
	table.Read = len(rows)
	for _, row := range rows {
		err := i.withSavepoint(ctx, tx, func() error {
			noticeID, err := i.lookupMapping(ctx, tx, "notification_notices", text(row, "notice_id"))
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return errLegacyReferenceMissing
				}
				return err
			}
			userID, err := i.lookupMapping(ctx, tx, "identity_users", text(row, "user_id"))
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return errLegacyReferenceMissing
				}
				return err
			}
			var readAt pgtype.Timestamptz
			if boolInt(row, "is_read") == 1 {
				readAt = pgtype.Timestamptz{Time: timestamp(row, "read_time"), Valid: true}
			}
			tag, err := tx.Exec(ctx, `
				INSERT INTO notification_recipients (notice_id, user_id, read_at)
				VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`,
				noticeID, userID, readAt,
			)
			if err == nil {
				if tag.RowsAffected() == 0 {
					table.Skipped++
				} else {
					table.Inserted++
				}
			}
			return err
		})
		if err != nil {
			if errors.Is(err, errLegacyReferenceMissing) {
				table.Skipped++
			} else {
				table.Failed++
				table.Errors = appendReportError(table.Errors, err)
			}
		}
	}
	i.report.Tables["notification_recipients"] = table
	return nil
}

func (i *Importer) importNoticeSendLogs(ctx context.Context, tx pgx.Tx) error {
	rows, err := i.readTable(ctx, `
		SELECT id, notice_id, channel, status, target_count, success_count, error_msg, send_time
		FROM sys_notice_send_log ORDER BY id`)
	if err != nil {
		return err
	}
	table := i.report.Tables["notification_send_logs"]
	table.Read = len(rows)
	for _, row := range rows {
		err := i.withSavepoint(ctx, tx, func() error {
			oldID := text(row, "id")
			logID, exists, err := i.getOrCreateMapping(ctx, tx, "notification_send_logs", oldID)
			if err != nil {
				return err
			}
			if exists {
				table.Skipped++
				return nil
			}
			noticeID, err := i.lookupMapping(ctx, tx, "notification_notices", text(row, "notice_id"))
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return errLegacyReferenceMissing
				}
				return err
			}
			status := "failed"
			if int64Value(row, "status") == 1 {
				status = "success"
			}
			_, err = tx.Exec(ctx, `
				INSERT INTO notification_send_logs (
					id, notice_id, channel, status, target_count, success_count, error_message, created_at
				) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
				logID, noticeID, text(row, "channel"), status, int32Value(row, "target_count"),
				int32Value(row, "success_count"), text(row, "error_msg"), timestamp(row, "send_time"),
			)
			if err == nil {
				table.Inserted++
			}
			return err
		})
		if err != nil {
			if errors.Is(err, errLegacyReferenceMissing) {
				table.Skipped++
			} else {
				table.Failed++
				table.Errors = appendReportError(table.Errors, err)
			}
		}
	}
	i.report.Tables["notification_send_logs"] = table
	return nil
}

func (i *Importer) validate(ctx context.Context) error {
	checks := map[string]string{
		"identity_users": `SELECT count(*) FROM identity_users`,
		"departments":    `SELECT count(*) FROM organization_departments WHERE deleted_at IS NULL`,
		"roles":          `SELECT count(*) FROM access_roles WHERE deleted_at IS NULL`,
		"menus":          `SELECT count(*) FROM access_menus`,
		"orphan_user_roles": `SELECT count(*) FROM access_user_roles relation
			LEFT JOIN identity_users u ON u.id = relation.user_id
			LEFT JOIN access_roles r ON r.id = relation.role_id
			WHERE u.id IS NULL OR r.id IS NULL`,
		"orphan_role_menus": `SELECT count(*) FROM access_role_menus relation
			LEFT JOIN access_roles r ON r.id = relation.role_id
			LEFT JOIN access_menus m ON m.id = relation.menu_id
			WHERE r.id IS NULL OR m.id IS NULL`,
		"legacy_mappings": `SELECT count(*) FROM legacy_id_map`,
	}
	for name, query := range checks {
		var count int64
		if err := i.target.QueryRow(ctx, query).Scan(&count); err != nil {
			return err
		}
		i.report.Validation[name] = count
	}
	return nil
}

func (i *Importer) importMappings(
	ctx context.Context,
	tx pgx.Tx,
	tableName string,
	rows []sourceRow,
	keys func(sourceRow) (string, string, error),
	query string,
) error {
	table := i.report.Tables[tableName]
	table.Read = len(rows)
	for _, row := range rows {
		err := i.withSavepoint(ctx, tx, func() error {
			first, second, err := keys(row)
			if err != nil {
				return err
			}
			tag, err := tx.Exec(ctx, query, first, second)
			if err != nil {
				return err
			}
			if tag.RowsAffected() == 0 {
				table.Skipped++
			} else {
				table.Inserted++
			}
			return nil
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				table.Skipped++
				continue
			}
			table.Failed++
			table.Errors = appendReportError(table.Errors, err)
		}
	}
	i.report.Tables[tableName] = table
	return nil
}

func (i *Importer) readTable(ctx context.Context, query string) ([]sourceRow, error) {
	rows, err := i.source.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	result := make([]sourceRow, 0)
	for rows.Next() {
		values := make([]any, len(columns))
		pointers := make([]any, len(columns))
		for index := range values {
			pointers[index] = &values[index]
		}
		if err := rows.Scan(pointers...); err != nil {
			return nil, err
		}
		row := make(sourceRow, len(columns))
		for index, column := range columns {
			row[strings.ToLower(column)] = normalizeSQLValue(values[index])
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

func (i *Importer) withSavepoint(ctx context.Context, tx pgx.Tx, fn func() error) error {
	if _, err := tx.Exec(ctx, "SAVEPOINT legacy_row"); err != nil {
		return err
	}
	if err := fn(); err != nil {
		_, _ = tx.Exec(ctx, "ROLLBACK TO SAVEPOINT legacy_row")
		_, _ = tx.Exec(ctx, "RELEASE SAVEPOINT legacy_row")
		return err
	}
	_, err := tx.Exec(ctx, "RELEASE SAVEPOINT legacy_row")
	return err
}

func (i *Importer) getOrCreateMapping(ctx context.Context, tx pgx.Tx, tableName, legacyID string) (uuid.UUID, bool, error) {
	id, err := i.lookupMapping(ctx, tx, tableName, legacyID)
	if err == nil {
		return id, true, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, false, err
	}
	id = mustID()
	_, err = tx.Exec(ctx, `
		INSERT INTO legacy_id_map (legacy_table, legacy_id, new_uuid)
		VALUES ($1, $2, $3)
		ON CONFLICT (legacy_table, legacy_id) DO NOTHING`,
		tableName, legacyID, id,
	)
	if err != nil {
		return uuid.Nil, false, err
	}
	return id, false, nil
}

func (i *Importer) lookupOrCreateMappingExists(ctx context.Context, tx pgx.Tx, tableName, legacyID string) (uuid.UUID, bool, error) {
	id, err := i.lookupMapping(ctx, tx, tableName, legacyID)
	if err == nil {
		return id, true, nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, false, nil
	}
	return uuid.Nil, false, err
}

func (i *Importer) lookupMapping(ctx context.Context, tx pgx.Tx, tableName, legacyID string) (uuid.UUID, error) {
	var id uuid.UUID
	err := tx.QueryRow(ctx, `
		SELECT new_uuid FROM legacy_id_map
		WHERE legacy_table = $1 AND legacy_id = $2`, tableName, legacyID).Scan(&id)
	return id, err
}

func (i *Importer) createMapping(ctx context.Context, tx pgx.Tx, tableName, legacyID string, id uuid.UUID) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO legacy_id_map (legacy_table, legacy_id, new_uuid)
		VALUES ($1, $2, $3)
		ON CONFLICT (legacy_table, legacy_id) DO UPDATE SET new_uuid = legacy_id_map.new_uuid`,
		tableName, legacyID, id,
	)
	return err
}

func (i *Importer) findUserByUsername(ctx context.Context, tx pgx.Tx, username string) (uuid.UUID, error) {
	var id uuid.UUID
	err := tx.QueryRow(ctx, `SELECT id FROM identity_users WHERE username = $1 AND deleted_at IS NULL`, username).Scan(&id)
	return id, err
}

func (i *Importer) mapNoticeTarget(ctx context.Context, tx pgx.Tx, legacyType int32, raw string) (string, []string) {
	ids := parseIntArray(raw)
	mapped := make([]string, 0, len(ids))
	switch legacyType {
	case 1:
		for _, id := range ids {
			if value, err := i.lookupMapping(ctx, tx, "identity_users", strconv.FormatInt(id, 10)); err == nil {
				mapped = append(mapped, value.String())
			}
		}
		return "users", mapped
	case 2:
		for _, id := range ids {
			if value, err := i.lookupMapping(ctx, tx, "organization_departments", strconv.FormatInt(id, 10)); err == nil {
				mapped = append(mapped, value.String())
			}
		}
		return "departments", mapped
	default:
		return "all", nil
	}
}

func secureJoin(root, relative string) (string, error) {
	if filepath.IsAbs(relative) {
		return "", errors.New("absolute legacy file path is not allowed")
	}
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	full := filepath.Join(absoluteRoot, filepath.Clean(relative))
	relativePath, err := filepath.Rel(absoluteRoot, full)
	if err != nil || relativePath == ".." || strings.HasPrefix(relativePath, ".."+string(filepath.Separator)) {
		return "", errors.New("legacy file path escapes the configured root")
	}
	return full, nil
}

func copyFile(sourcePath, targetPath string) ([]byte, int64, error) {
	source, err := os.Open(sourcePath) // #nosec G304 -- source path is constrained by secureJoin.
	if err != nil {
		return nil, 0, fmt.Errorf("open legacy file: %w", err)
	}
	defer func() { _ = source.Close() }()
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o750); err != nil {
		return nil, 0, err
	}
	target, err := os.CreateTemp(filepath.Dir(targetPath), ".legacy-import-*")
	if err != nil {
		return nil, 0, err
	}
	tempPath := target.Name()
	defer func() {
		_ = target.Close()
		_ = os.Remove(tempPath)
	}()
	hasher := sha256.New()
	size, err := io.Copy(io.MultiWriter(target, hasher), source)
	if closeErr := target.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return nil, 0, err
	}
	if err := os.Chmod(tempPath, 0o600); err != nil {
		return nil, 0, err
	}
	if err := os.Rename(tempPath, targetPath); err != nil {
		return nil, 0, err
	}
	return hasher.Sum(nil), size, nil
}

func legacyMessageType(value int32) string {
	switch value {
	case 2:
		return "image"
	case 3:
		return "file"
	case 4:
		return "system"
	default:
		return "text"
	}
}

func parseStringArray(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return []string{"station"}
	}
	var values []string
	if json.Unmarshal([]byte(raw), &values) == nil {
		return values
	}
	return []string{"station"}
}

func parseIntArray(raw string) []int64 {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var numbers []int64
	if json.Unmarshal([]byte(raw), &numbers) == nil {
		return numbers
	}
	var stringsList []string
	if json.Unmarshal([]byte(raw), &stringsList) == nil {
		numbers = make([]int64, 0, len(stringsList))
		for _, value := range stringsList {
			if parsed, err := strconv.ParseInt(value, 10, 64); err == nil {
				numbers = append(numbers, parsed)
			}
		}
	}
	return numbers
}

func normalizeSQLValue(value any) any {
	switch typed := value.(type) {
	case []byte:
		return string(typed)
	case time.Time:
		return typed.UTC()
	default:
		return value
	}
}

func sanitizeLegacyConfig(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, nested := range typed {
			lower := strings.ToLower(key)
			if strings.Contains(lower, "password") || strings.Contains(lower, "secret") ||
				strings.Contains(lower, "privatekey") || strings.Contains(lower, "private_key") ||
				strings.Contains(lower, "accesskey") || strings.Contains(lower, "access_key") ||
				strings.Contains(lower, "apikey") || strings.Contains(lower, "api_key") ||
				strings.Contains(lower, "token") {
				out[key] = ""
				continue
			}
			out[key] = sanitizeLegacyConfig(nested)
		}
		return out
	case []any:
		out := make([]any, len(typed))
		for index, nested := range typed {
			out[index] = sanitizeLegacyConfig(nested)
		}
		return out
	default:
		return value
	}
}

func text(row sourceRow, key string) string {
	value := row[key]
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case []byte:
		return strings.TrimSpace(string(typed))
	case nil:
		return ""
	default:
		return strings.TrimSpace(fmt.Sprint(typed))
	}
}

func int64Value(row sourceRow, key string) int64 {
	raw := text(row, key)
	if raw == "" {
		return 0
	}
	value, _ := strconv.ParseInt(raw, 10, 64)
	return value
}

func int32Value(row sourceRow, key string) int32 {
	return int32(int64Value(row, key)) // #nosec G115 -- legacy tinyint/int values are range checked by column constraints after import.
}

func int16Value(row sourceRow, key string) int16 {
	return int16(int64Value(row, key)) // #nosec G115 -- legacy tinyint values are range checked by target constraints.
}

func boolInt(row sourceRow, key string) int64 {
	return int64Value(row, key)
}

func timestamp(row sourceRow, key string) time.Time {
	if value, ok := row[key].(time.Time); ok {
		return value.UTC()
	}
	return time.Now().UTC()
}

func optionalText(row sourceRow, key string) pgtype.Text {
	value := text(row, key)
	return pgtype.Text{String: value, Valid: value != ""}
}

func optionalTextValue(value string) pgtype.Text {
	return pgtype.Text{String: value, Valid: value != ""}
}

func statusValue(row sourceRow) string {
	if int64Value(row, "status") == 1 {
		return "active"
	}
	return "disabled"
}

func activeStatus(value int32) string {
	if value == 1 {
		return "active"
	}
	return "disabled"
}

func legacyUserStatus(row sourceRow) string {
	if boolInt(row, "is_quit") == 1 {
		return "left"
	}
	switch int64Value(row, "status") {
	case 1:
		return "active"
	case 2:
		return "pending"
	case 3:
		return "rejected"
	default:
		return "suspended"
	}
}

func legacyDataScope(value int32) string {
	switch value {
	case 1:
		return "all"
	case 2:
		return "custom"
	case 3:
		return "department"
	case 4:
		return "department_tree"
	default:
		return "self"
	}
}

func permissionModule(code string) string {
	if index := strings.IndexByte(code, ':'); index > 0 {
		return code[:index]
	}
	return "legacy"
}

func sanitizeLTREE(value string) string {
	var builder strings.Builder
	for _, char := range strings.ToLower(value) {
		if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '_' {
			builder.WriteRune(char)
		} else {
			builder.WriteByte('_')
		}
	}
	return strings.Trim(builder.String(), "_")
}

func appendReportError(errorsList []string, err error) []string {
	if len(errorsList) >= 20 {
		return errorsList
	}
	message := err.Error()
	if len(message) > 500 {
		message = message[:500]
	}
	return append(errorsList, message)
}

func (i *Importer) addWarning(message string) {
	if len(i.report.Warnings) >= 100 {
		return
	}
	for _, existing := range i.report.Warnings {
		if existing == message {
			return
		}
	}
	i.report.Warnings = append(i.report.Warnings, message)
}

func pgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: [16]byte(id), Valid: id != uuid.Nil}
}

func mustID() uuid.UUID {
	id, err := uuid.NewV7()
	if err != nil {
		return uuid.New()
	}
	return id
}
