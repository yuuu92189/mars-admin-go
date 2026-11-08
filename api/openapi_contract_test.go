package api

import (
	"context"
	"fmt"
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

func TestCanonicalSuccessResponsesDeclareContent(t *testing.T) {
	document := loadContract(t)
	noContent := []string{}
	for path, pathItem := range document.Paths.Map() {
		for method, operation := range pathItem.Operations() {
			for status, responseRef := range operation.Responses.Map() {
				if !strings.HasPrefix(status, "2") || status == "204" {
					continue
				}
				if responseRef.Value == nil || len(responseRef.Value.Content) == 0 {
					noContent = append(noContent, method+" "+path+" ("+status+")")
				}
			}
		}
	}
	require.Empty(t, noContent, "2xx responses without content: %v", noContent)
}

func loadContract(t *testing.T) *openapi3.T {
	t.Helper()
	loader := openapi3.NewLoader()
	document, err := loader.LoadFromData(OpenAPISpec)
	require.NoError(t, err)
	require.NoError(t, document.Validate(context.Background()))
	return document
}
