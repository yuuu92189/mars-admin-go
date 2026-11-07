package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Table struct {
	Schema  string   `json:"schema"`
	Name    string   `json:"name"`
	Comment string   `json:"comment,omitempty"`
	Columns []Column `json:"columns,omitempty"`
}

type Column struct {
	Name         string `json:"name"`
	DataType     string `json:"dataType"`
	Nullable     bool   `json:"nullable"`
	DefaultValue string `json:"defaultValue,omitempty"`
	Comment      string `json:"comment,omitempty"`
	PrimaryKey   bool   `json:"primaryKey"`
}

type GenerateInput struct {
	Schema    string   `json:"schema"`
	Tables    []string `json:"tables"`
	Module    string   `json:"module"`
	Package   string   `json:"package"`
	APIPrefix string   `json:"apiPrefix"`
}

type Output struct {
	Files    map[string]string `json:"files"`
	Warnings []string          `json:"warnings"`
}

type ImportedTable struct {
	ID           uuid.UUID `json:"id"`
	Schema       string    `json:"schema"`
	Name         string    `json:"tableName"`
	Comment      string    `json:"tableComment"`
	ClassName    string    `json:"className"`
	PackageName  string    `json:"packageName"`
	ModuleName   string    `json:"moduleName"`
	BusinessName string    `json:"businessName"`
	FunctionName string    `json:"functionName"`
	Author       string    `json:"author"`
	GenType      string    `json:"genType"`
	GenPath      string    `json:"genPath"`
	FrontType    string    `json:"frontType"`
	FormLayout   string    `json:"formLayout"`
	ParentMenuID string    `json:"parentMenuId"`
	Remark       string    `json:"remark"`
	Columns      []Column  `json:"columns"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type Repository interface {
	Tables(ctx context.Context, schema string) ([]Table, error)
	Table(ctx context.Context, schema, table string) (Table, error)
	RecordGeneration(ctx context.Context, input GenerateInput, output Output) error
	SaveImportedTables(ctx context.Context, tables []ImportedTable) ([]ImportedTable, error)
	ListImportedTables(ctx context.Context, query string, page, pageSize int) ([]ImportedTable, int64, error)
	ImportedTable(ctx context.Context, id uuid.UUID) (ImportedTable, error)
	UpdateImportedTable(ctx context.Context, table ImportedTable) (ImportedTable, error)
	DeleteImportedTables(ctx context.Context, ids []uuid.UUID) error
}
