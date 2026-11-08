package api

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/require"
)

func TestOpenAPISecurityAndPermissionMetadata(t *testing.T) {
	document := loadContract(t)
	publicOperations := map[string]bool{
		"healthLive": true, "healthReady": true, "login": true, "issueCaptcha": true,
		"sendLoginSMS": true, "loginSMS": true, "refreshSession": true, "registerUser": true,
		"loginTypes": true, "socialLogin": true, "publicSettings": true, "paymentCallback": true,
	}

	for path, pathItem := range document.Paths.Map() {
		for method, operation := range pathItem.Operations() {
			require.NotEmpty(t, operation.OperationID, "%s %s", method, path)
			if publicOperations[operation.OperationID] {
				require.Empty(t, operation.Security, "%s must remain public", operation.OperationID)
				_, hasPermission := operation.Extensions["x-required-permission"]
				_, authOnly := operation.Extensions["x-auth-required"]
				require.False(t, hasPermission || authOnly, "%s must not declare a protected operation extension", operation.OperationID)
				continue
			}
			_, hasPermission := operation.Extensions["x-required-permission"]
			_, authOnly := operation.Extensions["x-auth-required"]
			require.True(t, hasPermission || authOnly, "%s %s must declare x-required-permission or x-auth-required", method, path)
		}
	}
}

func TestMetricsContractRequiresAuditRead(t *testing.T) {
	document := loadContract(t)
	operation := document.Paths.Find("/metrics").Get
	require.NotNil(t, operation)
	require.Equal(t, "audit:read", fmt.Sprint(operation.Extensions["x-required-permission"]))
	require.Contains(t, operation.Responses.Map(), "401")
	require.Contains(t, operation.Responses.Map(), "403")
}

func TestLoginContractUsesCanonicalFieldsAndAbsoluteExpiry(t *testing.T) {
	document := loadContract(t)
	request := document.Components.Schemas["LoginRequest"].Value
	require.NotNil(t, request)
	require.Contains(t, request.Properties, "captchaId")
	require.Contains(t, request.Properties, "captchaCode")
	require.NotContains(t, request.Properties, "captchaUuid")
	require.NotContains(t, request.Properties, "uuid")

	result := document.Components.Schemas["LoginResult"].Value
	require.NotNil(t, result)
	require.Contains(t, result.Properties, "expiresAt")
	require.Contains(t, result.Properties, "sessionId")
	require.NotContains(t, result.Properties, "expiresIn")
	require.Equal(t, "date-time", result.Properties["expiresAt"].Value.Format)
}

func TestGroupManagementOperationsExist(t *testing.T) {
	document := loadContract(t)
	required := []string{
		"removeGroupMember",
		"updateGroupMemberRole",
		"muteGroupMember",
		"transferGroupOwner",
	}
	found := make(map[string]bool)
	for _, path := range document.Paths.Map() {
		for _, operation := range path.Operations() {
			found[operation.OperationID] = true
		}
	}
	for _, operationID := range required {
		require.True(t, found[operationID], "missing %s", operationID)
	}
}

func TestKeyOperationsHaveResponseSchemas(t *testing.T) {
	document := loadContract(t)
	required := []string{
		"listUsers", "listRoles", "listDepartments", "listDictionaries", "listAuditEvents",
		"listFiles", "listMessageUsers", "listGroups", "listNotices", "listMyNotices",
		"listScheduledJobs", "listWorkflowDefinitions", "listWorkflowInstances",
		"listTodoWorkflowTasks", "listDoneWorkflowTasks", "listDatabaseTables",
		"monitoringSnapshot",
	}
	operations := make(map[string]*openapi3.Operation)
	for _, path := range document.Paths.Map() {
		for _, operation := range path.Operations() {
			operations[operation.OperationID] = operation
		}
	}

	for _, operationID := range required {
		operation := operations[operationID]
		require.NotNil(t, operation, "missing %s", operationID)
		foundSuccessSchema := false
		for status, responseRef := range operation.Responses.Map() {
			if !strings.HasPrefix(status, "2") || status == "204" {
				continue
			}
			require.NotNil(t, responseRef.Value, "%s %s response missing value", operationID, status)
			media := responseRef.Value.Content.Get("application/json")
			require.NotNil(t, media, "%s %s missing application/json", operationID, status)
			require.NotNil(t, media.Schema, "%s %s missing response schema", operationID, status)
			foundSuccessSchema = true
		}
		require.True(t, foundSuccessSchema, "%s has no 2xx JSON response schema", operationID)
	}
}

type successSchemaStats struct {
	Operations             int
	Responses              int
	SuccessResponses       int
	NoBodySuccessResponses int
	BodySuccessResponses   int
	MediaTypes             int
	Schemas                int
}

func TestAllSuccessResponsesHaveValidSchemas(t *testing.T) {
	document := loadContract(t)
	stats, issues := inspectSuccessResponseSchemas(document)
	t.Logf("checked operations=%d responses=%d successResponses=%d noBodySuccessResponses=%d bodySuccessResponses=%d mediaTypes=%d schemas=%d",
		stats.Operations, stats.Responses, stats.SuccessResponses, stats.NoBodySuccessResponses,
		stats.BodySuccessResponses, stats.MediaTypes, stats.Schemas)
	require.NotZero(t, stats.Operations)
	require.NotZero(t, stats.SuccessResponses)
	require.NotZero(t, stats.BodySuccessResponses)
	require.NotZero(t, stats.Schemas)
	require.Empty(t, issues, "success response schema issues:\n%s", strings.Join(issues, "\n"))
}

func TestSuccessSchemaInspectorRejectsBrokenContracts(t *testing.T) {
	tests := []struct {
		name      string
		contract  string
		wantIssue string
	}{
		{
			name: "missing content",
			contract: `openapi: 3.1.0
info: {title: test, version: "1"}
paths:
  /missing-content:
    get:
      operationId: missingContent
      responses:
        "200": {description: no content}
`,
			wantIssue: "response 200: content is missing",
		},
		{
			name: "media type without schema",
			contract: `openapi: 3.1.0
info: {title: test, version: "1"}
paths:
  /missing-schema:
    get:
      operationId: missingSchema
      responses:
        "200":
          description: missing schema
          content:
            application/json: {}
`,
			wantIssue: "application/json: schema is missing",
		},
		{
			name: "dangling local ref",
			contract: `openapi: 3.1.0
info: {title: test, version: "1"}
paths:
  /dangling-ref:
    get:
      operationId: danglingRef
      responses:
        "200":
          description: dangling ref
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Missing"
`,
			wantIssue: "application/json: schema reference",
		},
		{
			name: "valid 204",
			contract: `openapi: 3.1.0
info: {title: test, version: "1"}
paths:
  /no-content:
    delete:
      operationId: noContent
      responses:
        "204": {description: deleted}
`,
		},
		{
			name: "valid composition",
			contract: `openapi: 3.1.0
info: {title: test, version: "1"}
paths:
  /composition:
    get:
      operationId: composition
      responses:
        "200":
          description: composed
          content:
            application/json:
              schema:
                oneOf:
                  - type: object
                    properties: {id: {type: string}}
                  - type: object
                    properties: {name: {type: string}}
`,
		},
		{
			name: "one valid media type among several",
			contract: `openapi: 3.1.0
info: {title: test, version: "1"}
paths:
  /multiple-media:
    get:
      operationId: multipleMedia
      responses:
        "200":
          description: multiple media
          content:
            application/xml: {}
            application/json:
              schema:
                type: object
`,
		},
		{
			name: "non-success response is ignored",
			contract: `openapi: 3.1.0
info: {title: test, version: "1"}
paths:
  /non-success:
    get:
      operationId: nonSuccess
      responses:
        "400": {description: no content required}
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "dangling local ref" {
				loader := openapi3.NewLoader()
				_, err := loader.LoadFromData([]byte(tt.contract))
				require.Error(t, err)
				return
			}
			document := loadContractYAML(t, tt.contract)
			stats, issues := inspectSuccessResponseSchemas(document)
			if tt.wantIssue == "" {
				require.Empty(t, issues, "%v", issues)
				return
			}
			require.NotEmpty(t, issues)
			require.Contains(t, strings.Join(issues, "\n"), tt.wantIssue)
			require.NotZero(t, stats.Responses)
		})
	}
}

func inspectSuccessResponseSchemas(document *openapi3.T) (successSchemaStats, []string) {
	stats := successSchemaStats{}
	issues := make([]string, 0)
	for path, pathItem := range document.Paths.Map() {
		for method, operation := range pathItem.Operations() {
			stats.Operations++
			for status, responseRef := range operation.Responses.Map() {
				stats.Responses++
				if !strings.HasPrefix(status, "2") {
					continue
				}
				stats.SuccessResponses++
				if status == "204" || status == "205" || method == "HEAD" {
					stats.NoBodySuccessResponses++
					continue
				}
				stats.BodySuccessResponses++
				location := fmt.Sprintf("%s %s [%s] response %s", method, path, operation.OperationID, status)
				if responseRef == nil || responseRef.Value == nil {
					issues = append(issues, location+": response definition is missing")
					continue
				}
				if len(responseRef.Value.Content) == 0 {
					issues = append(issues, location+": content is missing")
					continue
				}
				validSchema := false
				mediaIssues := make([]string, 0)
				mediaTypes := make([]string, 0, len(responseRef.Value.Content))
				for mediaType := range responseRef.Value.Content {
					mediaTypes = append(mediaTypes, mediaType)
				}
				sort.Strings(mediaTypes)
				for _, mediaType := range mediaTypes {
					stats.MediaTypes++
					media := responseRef.Value.Content[mediaType]
					if media == nil || media.Schema == nil {
						mediaIssues = append(mediaIssues, location+" "+mediaType+": schema is missing")
						continue
					}
					if media.Schema.Value == nil {
						mediaIssues = append(mediaIssues, location+" "+mediaType+": schema reference is unresolved: "+media.Schema.Ref)
						continue
					}
					stats.Schemas++
					if err := media.Schema.Value.Validate(context.Background()); err != nil {
						mediaIssues = append(mediaIssues, location+" "+mediaType+": invalid schema: "+err.Error())
						continue
					}
					validSchema = true
				}
				if !validSchema {
					issues = append(issues, mediaIssues...)
					issues = append(issues, location+": no valid schema")
				}
			}
		}
	}
	return stats, issues
}

func loadContract(t *testing.T) *openapi3.T {
	t.Helper()
	loader := openapi3.NewLoader()
	document, err := loader.LoadFromData(OpenAPISpec)
	require.NoError(t, err)
	require.NoError(t, document.Validate(context.Background()))
	return document
}

func loadContractYAML(t *testing.T, raw string) *openapi3.T {
	t.Helper()
	loader := openapi3.NewLoader()
	document, err := loader.LoadFromData([]byte(raw))
	require.NoError(t, err)
	return document
}
