package application

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"go/format"
	"regexp"
	"sort"
	"strings"

	"mars-admin-go/internal/modules/codegen/domain"
	"mars-admin-go/internal/platform/apperror"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Service struct {
	repo domain.Repository
}

func New(repo domain.Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Tables(ctx context.Context, schema string) ([]domain.Table, error) {
	if schema == "" {
		schema = "public"
	}
	if !validIdentifier(schema) {
		return nil, apperror.Validation("schema", "Invalid schema identifier.")
	}
	return s.repo.Tables(ctx, schema)
}

func (s *Service) ImportTables(ctx context.Context, schema string, tableNames []string) ([]domain.ImportedTable, error) {
	if schema == "" {
		schema = "public"
	}
	if !validIdentifier(schema) {
		return nil, apperror.Validation("schema", "Invalid schema identifier.")
	}
	if len(tableNames) == 0 || len(tableNames) > 50 {
		return nil, apperror.Validation("tables", "Select between 1 and 50 tables.")
	}
	items := make([]domain.ImportedTable, 0, len(tableNames))
	for _, tableName := range tableNames {
		if !validIdentifier(tableName) {
			return nil, apperror.Validation("tables", "Invalid table identifier.")
		}
		table, err := s.repo.Table(ctx, schema, tableName)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, apperror.NotFound("Table")
			}
			return nil, err
		}
		className := exported(table.Name)
		moduleName := strings.ToLower(table.Name)
		functionName := table.Name
		if table.Comment != "" {
			functionName = table.Comment
		}
		items = append(items, domain.ImportedTable{
			ID: uuid.New(), Schema: schema, Name: table.Name, Comment: table.Comment,
			ClassName: className, PackageName: "generated", ModuleName: moduleName,
			BusinessName: lowerCamel(table.Name), FunctionName: functionName,
			Author: "", GenType: "zip", GenPath: "", FrontType: "vue3",
			FormLayout: "one-column", ParentMenuID: "", Remark: "", Columns: table.Columns,
		})
	}
	return s.repo.SaveImportedTables(ctx, items)
}

func (s *Service) ImportedTables(ctx context.Context, query string, page, pageSize int) ([]domain.ImportedTable, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.repo.ListImportedTables(ctx, query, page, pageSize)
}

func (s *Service) ImportedTable(ctx context.Context, id uuid.UUID) (domain.ImportedTable, error) {
	return s.repo.ImportedTable(ctx, id)
}

func (s *Service) UpdateImportedTable(ctx context.Context, table domain.ImportedTable) (domain.ImportedTable, error) {
	if table.ID == uuid.Nil {
		return domain.ImportedTable{}, apperror.Validation("id", "A table id is required.")
	}
	if !validIdentifier(table.PackageName) || !validIdentifier(table.ModuleName) {
		return domain.ImportedTable{}, apperror.Validation("module", "Module and package must be valid identifiers.")
	}
	return s.repo.UpdateImportedTable(ctx, table)
}

func (s *Service) DeleteImportedTables(ctx context.Context, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return apperror.Validation("ids", "At least one table id is required.")
	}
	return s.repo.DeleteImportedTables(ctx, ids)
}

func (s *Service) PreviewImported(ctx context.Context, id uuid.UUID) (domain.Output, error) {
	table, err := s.repo.ImportedTable(ctx, id)
	if err != nil {
		return domain.Output{}, err
	}
	return s.Preview(ctx, domain.GenerateInput{
		Schema: table.Schema, Tables: []string{table.Name},
		Module: table.ModuleName, Package: table.PackageName,
	})
}

func (s *Service) SyncImported(ctx context.Context, id uuid.UUID) (domain.ImportedTable, error) {
	table, err := s.repo.ImportedTable(ctx, id)
	if err != nil {
		return domain.ImportedTable{}, err
	}
	current, err := s.repo.Table(ctx, table.Schema, table.Name)
	if err != nil {
		return domain.ImportedTable{}, err
	}
	table.Comment = current.Comment
	table.Columns = current.Columns
	return s.repo.UpdateImportedTable(ctx, table)
}

func (s *Service) Preview(ctx context.Context, input domain.GenerateInput) (domain.Output, error) {
	if input.Schema == "" {
		input.Schema = "public"
	}
	if input.Module == "" {
		input.Module = "feature"
	}
	if input.Package == "" {
		input.Package = strings.ToLower(input.Module)
	}
	if !validIdentifier(input.Schema) || !validIdentifier(input.Module) || !validIdentifier(input.Package) {
		return domain.Output{}, apperror.Validation("input", "Schema, module and package must be valid identifiers.")
	}
	if len(input.Tables) == 0 || len(input.Tables) > 20 {
		return domain.Output{}, apperror.Validation("tables", "Select between 1 and 20 tables.")
	}
	files := make(map[string]string)
	warnings := make([]string, 0)
	for _, tableName := range input.Tables {
		if !validIdentifier(tableName) {
			return domain.Output{}, apperror.Validation("tables", "Invalid table identifier.")
		}
		table, err := s.repo.Table(ctx, input.Schema, tableName)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return domain.Output{}, apperror.NotFound("Table")
			}
			return domain.Output{}, err
		}
		model, err := renderModel(table, input.Package)
		if err != nil {
			return domain.Output{}, err
		}
		files[input.Module+"/"+table.Name+"/domain/"+table.Name+".go"] = model
		files[input.Module+"/"+table.Name+"/adapter/postgres/queries.sql"] = renderQueries(table)
		files[input.Module+"/"+table.Name+"/adapter/http/routes.go"] = renderRoutes(table, input)
		files["api/openapi/"+table.Name+".yaml"] = renderOpenAPI(table, input)
		if len(table.Columns) > 30 {
			warnings = append(warnings, table.Name+" has more than 30 columns; review generated request shapes.")
		}
	}
	output := domain.Output{Files: files, Warnings: warnings}
	if err := s.repo.RecordGeneration(ctx, input, output); err != nil {
		return domain.Output{}, err
	}
	return output, nil
}

func renderModel(table domain.Table, packageName string) (string, error) {
	var buffer bytes.Buffer
	fmt.Fprintf(&buffer, "package %s\n\n", packageName)
	needsTime := false
	needsUUID := false
	for _, column := range table.Columns {
		columnType := goType(column)
		needsTime = needsTime || strings.Contains(columnType, "time.")
		needsUUID = needsUUID || strings.Contains(columnType, "uuid.")
	}
	if needsTime || needsUUID {
		buffer.WriteString("import (\n")
		if needsTime {
			buffer.WriteString("\t\"time\"\n")
		}
		if needsUUID {
			buffer.WriteString("\n\t\"github.com/google/uuid\"\n")
		}
		buffer.WriteString(")\n\n")
	}
	fmt.Fprintf(&buffer, "// %s maps the %s.%s table.\n", exported(table.Name), table.Schema, table.Name)
	fmt.Fprintf(&buffer, "type %s struct {\n", exported(table.Name))
	for _, column := range table.Columns {
		jsonName := lowerCamel(column.Name)
		fmt.Fprintf(&buffer, "\t%s %s `json:\"%s\"`\n", exported(column.Name), goType(column), jsonName)
	}
	buffer.WriteString("}\n")
	formatted, err := format.Source(buffer.Bytes())
	if err != nil {
		return "", err
	}
	return string(formatted), nil
}

func renderQueries(table domain.Table) string {
	idColumn := "id"
	for _, column := range table.Columns {
		if column.PrimaryKey {
			idColumn = column.Name
			break
		}
	}
	return fmt.Sprintf(`-- name: Get%s :one
SELECT * FROM %s.%s WHERE %s = $1;

-- name: List%s :many
SELECT * FROM %s.%s ORDER BY %s LIMIT $1 OFFSET $2;

-- name: Count%s :one
SELECT count(*) FROM %s.%s;

-- name: Delete%s :exec
DELETE FROM %s.%s WHERE %s = $1;
`, exported(table.Name), table.Schema, table.Name, idColumn,
		exported(table.Name), table.Schema, table.Name, idColumn,
		exported(table.Name), table.Schema, table.Name,
		exported(table.Name), table.Schema, table.Name, idColumn)
}

func renderRoutes(table domain.Table, input domain.GenerateInput) string {
	base := input.APIPrefix
	if base == "" {
		base = "/api/v1/admin/" + strings.ReplaceAll(table.Name, "_", "-")
	}
	return fmt.Sprintf(`package http

import "net/http"

// Register%s mounts the generated resource routes.
// Base path: %s.
func Register%s(router interface {
	Get(pattern string, handlerFn http.HandlerFunc)
	Post(pattern string, handlerFn http.HandlerFunc)
	Put(pattern string, handlerFn http.HandlerFunc)
	Delete(pattern string, handlerFn http.HandlerFunc)
}) {
	// Handlers must be bound to generated application methods before registration.
	_ = router
}
	`, exported(table.Name), base, exported(table.Name))
}

func renderOpenAPI(table domain.Table, input domain.GenerateInput) string {
	base := input.APIPrefix
	if base == "" {
		base = "/api/v1/admin/" + strings.ReplaceAll(table.Name, "_", "-")
	}
	return fmt.Sprintf(`paths:
  %s:
    get:
      operationId: list%s
      responses:
        "200":
          description: Success
  %s/{id}:
    get:
      operationId: get%s
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
            format: uuid
      responses:
        "200":
          description: Success
`, base, exported(table.Name), base, exported(table.Name))
}

func goType(column domain.Column) string {
	value := strings.ToLower(column.DataType)
	nullable := column.Nullable
	switch {
	case strings.Contains(value, "uuid"):
		if nullable {
			return "*uuid.UUID"
		}
		return "uuid.UUID"
	case strings.Contains(value, "timestamp"):
		if nullable {
			return "*time.Time"
		}
		return "time.Time"
	case strings.Contains(value, "int"), strings.Contains(value, "serial"):
		if nullable {
			return "*int64"
		}
		return "int64"
	case strings.Contains(value, "numeric"), strings.Contains(value, "decimal"):
		if nullable {
			return "*float64"
		}
		return "float64"
	case strings.Contains(value, "bool"):
		if nullable {
			return "*bool"
		}
		return "bool"
	case strings.Contains(value, "json"):
		return "map[string]any"
	default:
		if nullable {
			return "*string"
		}
		return "string"
	}
}

func exported(value string) string {
	parts := regexp.MustCompile(`[^a-zA-Z0-9]+`).Split(value, -1)
	var builder strings.Builder
	for _, part := range parts {
		if part == "" {
			continue
		}
		builder.WriteString(strings.ToUpper(part[:1]))
		builder.WriteString(part[1:])
	}
	if builder.Len() == 0 {
		return "Generated"
	}
	return builder.String()
}

func lowerCamel(value string) string {
	name := exported(value)
	if len(name) > 1 {
		name = strings.ToLower(name[:1]) + name[1:]
	}
	return name
}

func validIdentifier(value string) bool {
	return regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`).MatchString(value)
}

func SortedTables(tables []domain.Table) []domain.Table {
	sort.Slice(tables, func(i, j int) bool { return tables[i].Name < tables[j].Name })
	return tables
}
