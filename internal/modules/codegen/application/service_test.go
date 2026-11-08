package application

import (
	"strings"
	"testing"

	"mars-admin-go/internal/modules/codegen/domain"
)

func TestRenderModelOmitsUnusedImports(t *testing.T) {
	model, err := renderModel(domain.Table{
		Schema: "public",
		Name:   "plain_record",
		Columns: []domain.Column{
			{Name: "name", DataType: "text"},
			{Name: "amount", DataType: "numeric"},
		},
	}, "generated")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(model, "github.com/google/uuid") || strings.Contains(model, `"time"`) {
		t.Fatalf("plain model contains unused imports:\n%s", model)
	}
	if !strings.Contains(model, "type PlainRecord struct") {
		t.Fatalf("plain model missing type declaration:\n%s", model)
	}
}

func TestRenderModelImportsUUIDAndTimeWhenUsed(t *testing.T) {
	model, err := renderModel(domain.Table{
		Schema: "public",
		Name:   "timed_record",
		Columns: []domain.Column{
			{Name: "id", DataType: "uuid"},
			{Name: "created_at", DataType: "timestamp with time zone"},
		},
	}, "generated")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(model, "github.com/google/uuid") || !strings.Contains(model, `"time"`) {
		t.Fatalf("typed model missing imports:\n%s", model)
	}
}
