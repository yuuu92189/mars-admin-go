package api

import (
	"context"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

func TestOpenAPISpecIsValidAndOperationIDsAreUnique(t *testing.T) {
	loader := openapi3.NewLoader()
	document, err := loader.LoadFromData(OpenAPISpec)
	if err != nil {
		t.Fatal(err)
	}
	if err := document.Validate(context.Background()); err != nil {
		t.Fatal(err)
	}
	operationIDs := make(map[string]struct{})
	for _, path := range document.Paths.Map() {
		for _, operation := range path.Operations() {
			if operation.OperationID == "" {
				t.Fatalf("operation without operationId: %s", operation.Summary)
			}
			if _, exists := operationIDs[operation.OperationID]; exists {
				t.Fatalf("duplicate operationId %q", operation.OperationID)
			}
			operationIDs[operation.OperationID] = struct{}{}
		}
	}
}
