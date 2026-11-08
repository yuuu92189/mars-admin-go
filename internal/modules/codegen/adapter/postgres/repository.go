package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"mars-admin-go/internal/modules/codegen/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Tables(ctx context.Context, schema string) ([]domain.Table, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT c.relname, COALESCE(obj_description(c.oid, 'pg_class'), '')
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = $1 AND c.relkind = 'r'
		ORDER BY c.relname`, schema)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tables := make([]domain.Table, 0)
	for rows.Next() {
		var table domain.Table
		table.Schema = schema
		if err := rows.Scan(&table.Name, &table.Comment); err != nil {
			return nil, err
		}
		tables = append(tables, table)
	}
	return tables, rows.Err()
}

func (r *Repository) Table(ctx context.Context, schema, table string) (domain.Table, error) {
	result := domain.Table{Schema: schema, Name: table}
	err := r.pool.QueryRow(ctx, `
		SELECT COALESCE(obj_description(c.oid, 'pg_class'), '')
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = $1 AND c.relname = $2`, schema, table).Scan(&result.Comment)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Table{}, pgx.ErrNoRows
	}
	if err != nil {
		return domain.Table{}, err
	}
	rows, err := r.pool.Query(ctx, `
		SELECT
		    a.attname,
		    format_type(a.atttypid, a.atttypmod),
		    NOT a.attnotnull,
		    COALESCE(pg_get_expr(d.adbin, d.adrelid), ''),
		    COALESCE(col_description(a.attrelid, a.attnum), ''),
		    EXISTS (
		        SELECT 1 FROM pg_index i
		        WHERE i.indrelid = a.attrelid AND a.attnum = ANY(i.indkey) AND i.indisprimary
		    )
		FROM pg_attribute a
		JOIN pg_class c ON c.oid = a.attrelid
		JOIN pg_namespace n ON n.oid = c.relnamespace
		LEFT JOIN pg_attrdef d ON d.adrelid = a.attrelid AND d.adnum = a.attnum
		WHERE n.nspname = $1 AND c.relname = $2 AND a.attnum > 0 AND NOT a.attisdropped
		ORDER BY a.attnum`, schema, table)
	if err != nil {
		return domain.Table{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var column domain.Column
		if err := rows.Scan(&column.Name, &column.DataType, &column.Nullable, &column.DefaultValue, &column.Comment, &column.PrimaryKey); err != nil {
			return domain.Table{}, err
		}
		result.Columns = append(result.Columns, column)
	}
	return result, rows.Err()
}

func (r *Repository) RecordGeneration(ctx context.Context, input domain.GenerateInput, output domain.Output) error {
	options, _ := json.Marshal(input)
	summary, _ := json.Marshal(map[string]any{"files": len(output.Files), "warnings": len(output.Warnings)})
	_, err := r.pool.Exec(ctx, `
		INSERT INTO codegen_generations (
		    id, schema_name, table_names, options, status, output_summary, created_by
		) VALUES ($1, $2, $3, $4, 'previewed', $5, NULL)`,
		mustID(), input.Schema, input.Tables, options, summary,
	)
	return err
}

func (r *Repository) SaveImportedTables(ctx context.Context, tables []domain.ImportedTable) ([]domain.ImportedTable, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	result := make([]domain.ImportedTable, 0, len(tables))
	for _, table := range tables {
		if table.ID == uuid.Nil {
			table.ID = mustID()
		}
		columns, _ := json.Marshal(table.Columns)
		var saved domain.ImportedTable
		err := tx.QueryRow(ctx, `
			INSERT INTO codegen_imported_tables (
				id, schema_name, table_name, table_comment, class_name, package_name,
				module_name, business_name, function_name, author, gen_type, gen_path,
				front_type, form_layout, parent_menu_id, remark, columns
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17
			)
			ON CONFLICT (schema_name, table_name) DO UPDATE SET
				table_comment = excluded.table_comment,
				class_name = excluded.class_name,
				package_name = excluded.package_name,
				module_name = excluded.module_name,
				business_name = excluded.business_name,
				function_name = excluded.function_name,
				author = excluded.author,
				gen_type = excluded.gen_type,
				gen_path = excluded.gen_path,
				front_type = excluded.front_type,
				form_layout = excluded.form_layout,
				parent_menu_id = excluded.parent_menu_id,
				remark = excluded.remark,
				columns = excluded.columns,
				updated_at = now()
			RETURNING id, schema_name, table_name, table_comment, class_name, package_name,
				module_name, business_name, function_name, author, gen_type, gen_path,
				front_type, form_layout, parent_menu_id, remark, columns, created_at, updated_at`,
			table.ID, table.Schema, table.Name, table.Comment, table.ClassName, table.PackageName,
			table.ModuleName, table.BusinessName, table.FunctionName, table.Author,
			table.GenType, table.GenPath, table.FrontType, table.FormLayout,
			table.ParentMenuID, table.Remark, columns,
		).Scan(
			&saved.ID, &saved.Schema, &saved.Name, &saved.Comment, &saved.ClassName, &saved.PackageName,
			&saved.ModuleName, &saved.BusinessName, &saved.FunctionName, &saved.Author,
			&saved.GenType, &saved.GenPath, &saved.FrontType, &saved.FormLayout,
			&saved.ParentMenuID, &saved.Remark, &saved.Columns, &saved.CreatedAt, &saved.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		result = append(result, saved)
	}
	return result, tx.Commit(ctx)
}

func (r *Repository) ListImportedTables(ctx context.Context, query string, page, pageSize int) ([]domain.ImportedTable, int64, error) {
	pattern := "%" + strings.TrimSpace(query) + "%"
	var total int64
	if err := r.pool.QueryRow(ctx, `
		SELECT count(*) FROM codegen_imported_tables
		WHERE $1 = '%%' OR table_name ILIKE $1 OR table_comment ILIKE $1`, pattern).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, schema_name, table_name, table_comment, class_name, package_name,
			module_name, business_name, function_name, author, gen_type, gen_path,
			front_type, form_layout, parent_menu_id, remark, columns, created_at, updated_at
		FROM codegen_imported_tables
		WHERE $1 = '%%' OR table_name ILIKE $1 OR table_comment ILIKE $1
		ORDER BY updated_at DESC
		LIMIT $2 OFFSET $3`, pattern, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]domain.ImportedTable, 0)
	for rows.Next() {
		item, err := scanImportedTable(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (r *Repository) ImportedTable(ctx context.Context, id uuid.UUID) (domain.ImportedTable, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, schema_name, table_name, table_comment, class_name, package_name,
			module_name, business_name, function_name, author, gen_type, gen_path,
			front_type, form_layout, parent_menu_id, remark, columns, created_at, updated_at
		FROM codegen_imported_tables WHERE id = $1`, id)
	return scanImportedTable(row)
}

func (r *Repository) UpdateImportedTable(ctx context.Context, table domain.ImportedTable) (domain.ImportedTable, error) {
	columns, _ := json.Marshal(table.Columns)
	row := r.pool.QueryRow(ctx, `
		UPDATE codegen_imported_tables SET
			class_name = $2, package_name = $3, module_name = $4, business_name = $5,
			function_name = $6, author = $7, gen_type = $8, gen_path = $9,
			front_type = $10, form_layout = $11, parent_menu_id = $12, remark = $13,
			columns = $14, updated_at = now()
		WHERE id = $1
		RETURNING id, schema_name, table_name, table_comment, class_name, package_name,
			module_name, business_name, function_name, author, gen_type, gen_path,
			front_type, form_layout, parent_menu_id, remark, columns, created_at, updated_at`,
		table.ID, table.ClassName, table.PackageName, table.ModuleName, table.BusinessName,
		table.FunctionName, table.Author, table.GenType, table.GenPath, table.FrontType,
		table.FormLayout, table.ParentMenuID, table.Remark, columns,
	)
	return scanImportedTable(row)
}

func (r *Repository) DeleteImportedTables(ctx context.Context, ids []uuid.UUID) error {
	for _, id := range ids {
		if _, err := r.pool.Exec(ctx, `DELETE FROM codegen_imported_tables WHERE id = $1`, id); err != nil {
			return err
		}
	}
	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanImportedTable(row rowScanner) (domain.ImportedTable, error) {
	var table domain.ImportedTable
	var columns []byte
	err := row.Scan(
		&table.ID, &table.Schema, &table.Name, &table.Comment, &table.ClassName, &table.PackageName,
		&table.ModuleName, &table.BusinessName, &table.FunctionName, &table.Author,
		&table.GenType, &table.GenPath, &table.FrontType, &table.FormLayout,
		&table.ParentMenuID, &table.Remark, &columns, &table.CreatedAt, &table.UpdatedAt,
	)
	if err != nil {
		return domain.ImportedTable{}, err
	}
	_ = json.Unmarshal(columns, &table.Columns)
	return table, nil
}

func mustID() uuid.UUID {
	id, err := uuid.NewV7()
	if err != nil {
		return uuid.New()
	}
	return id
}
