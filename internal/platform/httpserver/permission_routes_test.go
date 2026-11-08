package httpserver

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"

	"mars-admin-go/api"
	"mars-admin-go/internal/platform/requestctx"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type routeSpec struct {
	Method string
	Path   string
	Auth   bool
	Mode   string
	Codes  []string
	Source string
}

type routeState struct {
	auth  bool
	mode  string
	codes []string
}

type handlerSpec struct {
	File   string
	Func   string
	Prefix string
}

// canonicalHandlerSpecs intentionally excludes the legacy /api/* surface.
// Those routes have a separate compatibility contract and are not part of the
// canonical OpenAPI document checked here.
var canonicalHandlerSpecs = []handlerSpec{
	{File: "internal/modules/identity/adapter/http/handler.go", Func: "Routes", Prefix: "/api/v1/auth"},
	{File: "internal/modules/identity/adapter/http/handler.go", Func: "AdminRoutes", Prefix: "/api/v1/admin/users"},
	{File: "internal/modules/access/adapter/http/handler.go", Func: "Routes", Prefix: "/api/v1/admin/access"},
	{File: "internal/modules/organization/adapter/http/handler.go", Func: "Routes", Prefix: "/api/v1/admin/organization"},
	{File: "internal/modules/system/adapter/http/handler.go", Func: "Routes", Prefix: "/api/v1/admin"},
	{File: "internal/modules/audit/adapter/http/handler.go", Func: "Routes", Prefix: "/api/v1/admin/audit"},
	{File: "internal/modules/monitoring/http.go", Func: "Routes", Prefix: "/api/v1/admin/monitoring"},
	{File: "internal/modules/storage/adapter/http/handler.go", Func: "Routes", Prefix: "/api/v1/files"},
	{File: "internal/modules/messaging/adapter/http/handler.go", Func: "Routes", Prefix: "/api/v1"},
	{File: "internal/modules/notification/adapter/http/handler.go", Func: "Routes", Prefix: "/api/v1/notices"},
	{File: "internal/modules/scheduler/adapter/http/handler.go", Func: "Routes", Prefix: "/api/v1/admin/scheduler"},
	{File: "internal/modules/workflow/adapter/http/handler.go", Func: "Routes", Prefix: "/api/v1/workflows"},
	{File: "internal/modules/codegen/adapter/http/handler.go", Func: "Routes", Prefix: "/api/v1/admin/codegen"},
	{File: "internal/modules/integrations/http.go", Func: "Routes", Prefix: "/api/v1/admin/integrations"},
}

func TestCanonicalRoutePermissionsMatchOpenAPI(t *testing.T) {
	root := repositoryRoot(t)
	routes := collectCanonicalRoutes(t, root)
	routes = append(routes,
		routeSpec{Method: "GET", Path: "/metrics", Auth: true, Mode: "all", Codes: []string{"audit:read"}, Source: "internal/bootstrap/app.go"},
		routeSpec{Method: "GET", Path: "/health/live", Source: "internal/bootstrap/app.go"},
		routeSpec{Method: "GET", Path: "/health/ready", Source: "internal/bootstrap/app.go"},
		routeSpec{Method: "POST", Path: "/api/v1/payments/callbacks/{provider}", Source: "internal/bootstrap/app.go"},
	)
	require.NotEmpty(t, routes)

	document := loadOpenAPIContract(t)
	operations := make(map[string]*openapi3.Operation)
	for path, pathItem := range document.Paths.Map() {
		for method, operation := range pathItem.Operations() {
			operations[strings.ToUpper(method)+" "+path] = operation
		}
	}

	seeded := readSeedPermissions(t, root)
	catalog := readCatalogPermissions(t, root)
	seen := make(map[string]bool)
	stats := struct {
		public, authOnly, business int
	}{}

	for _, route := range routes {
		key := route.Method + " " + route.Path
		operation := operations[key]
		require.NotNil(t, operation, "%s missing from OpenAPI (%s)", key, route.Source)
		seen[key] = true
		if route.Auth {
			if route.Mode == "" {
				stats.authOnly++
				require.Equal(t, true, operation.Extensions["x-auth-required"], "%s: expected x-auth-required", key)
			} else {
				stats.business++
				require.NotEmpty(t, route.Codes, "%s: permission mode has no code", key)
				require.Equal(t, route.Codes[0], fmt.Sprint(operation.Extensions["x-required-permission"]), "%s: permission mismatch", key)
			}
		} else {
			stats.public++
			require.Empty(t, operation.Security, "%s: public route must have security: []", key)
			_, hasPermission := operation.Extensions["x-required-permission"]
			_, hasAuth := operation.Extensions["x-auth-required"]
			require.False(t, hasPermission || hasAuth, "%s: public route has a protected extension", key)
		}
		for _, code := range route.Codes {
			_, inSeed := seeded[code]
			_, inCatalog := catalog[code]
			require.True(t, inSeed, "%s uses %s not present in seed", key, code)
			require.True(t, inCatalog, "%s uses %s not present in catalog", key, code)
		}
	}
	unmatched := make([]string, 0)
	for key := range operations {
		if !seen[key] {
			unmatched = append(unmatched, key)
		}
	}
	sort.Strings(unmatched)
	require.Empty(t, unmatched, "OpenAPI operations without a matching actual route: %v", unmatched)

	for code := range seeded {
		_, inCatalog := catalog[code]
		require.True(t, inCatalog, "seed permission %s is not documented in permission catalog", code)
	}
	for code := range catalog {
		_, inSeed := seeded[code]
		require.True(t, inSeed, "catalog permission %s is not present in seed", code)
	}
	t.Logf("route permissions checked: total=%d public=%d authOnly=%d business=%d", len(routes), stats.public, stats.authOnly, stats.business)
}

func TestRequireAnyPermissionPreservesORSemantics(t *testing.T) {
	handler := RequireAnyPermission("alpha:read", "beta:read")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	allowed := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	request = request.WithContext(requestctx.WithActor(request.Context(), requestctx.Actor{
		UserID: uuid.New(), Permissions: []string{"beta:read"},
	}))
	handler.ServeHTTP(allowed, request)
	require.Equal(t, http.StatusNoContent, allowed.Code)

	denied := httptest.NewRecorder()
	request = httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	request = request.WithContext(requestctx.WithActor(request.Context(), requestctx.Actor{
		UserID: uuid.New(), Permissions: []string{"other:read"},
	}))
	handler.ServeHTTP(denied, request)
	require.Equal(t, http.StatusForbidden, denied.Code)
}

func collectCanonicalRoutes(t *testing.T, root string) []routeSpec {
	t.Helper()
	routes := make([]routeSpec, 0)
	for _, spec := range canonicalHandlerSpecs {
		path := filepath.Join(root, filepath.FromSlash(spec.File))
		fileSet := token.NewFileSet()
		parsed, err := parser.ParseFile(fileSet, path, nil, 0)
		require.NoError(t, err, "parse %s", spec.File)
		var function *ast.FuncDecl
		for _, declaration := range parsed.Decls {
			fn, ok := declaration.(*ast.FuncDecl)
			if ok && fn.Name.Name == spec.Func {
				function = fn
				break
			}
		}
		require.NotNil(t, function, "%s not found in %s", spec.Func, spec.File)
		walkRouteStatements(function.Body.List, routeState{}, spec.Prefix, spec.File+":"+spec.Func, &routes)
	}
	sort.Slice(routes, func(i, j int) bool {
		if routes[i].Path == routes[j].Path {
			return routes[i].Method < routes[j].Method
		}
		return routes[i].Path < routes[j].Path
	})
	return routes
}

func walkRouteStatements(statements []ast.Stmt, inherited routeState, prefix, source string, routes *[]routeSpec) {
	state := inherited
	for _, statement := range statements {
		exprStatement, ok := statement.(*ast.ExprStmt)
		if !ok {
			if block, ok := statement.(*ast.BlockStmt); ok {
				walkRouteStatements(block.List, state, prefix, source, routes)
			}
			continue
		}
		call, ok := exprStatement.X.(*ast.CallExpr)
		if !ok {
			continue
		}
		name, ok := callName(call)
		if !ok {
			continue
		}
		switch name {
		case "Use":
			state = applyRouteState(state, call.Args)
		case "Group":
			if len(call.Args) > 0 {
				if function, ok := call.Args[0].(*ast.FuncLit); ok {
					walkRouteStatements(function.Body.List, state, prefix, source, routes)
				}
			}
		default:
			if !isRouteMethod(name) {
				continue
			}
			path, ok := stringArgument(call, 0)
			if !ok {
				continue
			}
			if name == "Handle" && path != "/metrics" {
				continue
			}
			method := strings.ToUpper(name)
			if name == "Handle" {
				method = "GET"
			}
			routeState := state
			if selector, ok := call.Fun.(*ast.SelectorExpr); ok {
				if withCall, ok := selector.X.(*ast.CallExpr); ok {
					if withName, ok := callName(withCall); ok && withName == "With" {
						routeState = applyRouteState(routeState, withCall.Args)
					}
				}
			}
			*routes = append(*routes, routeSpec{
				Method: method, Path: joinRoutePath(prefix, path), Auth: routeState.auth,
				Mode: routeState.mode, Codes: append([]string(nil), routeState.codes...), Source: source,
			})
		}
	}
}

func applyRouteState(state routeState, args []ast.Expr) routeState {
	for _, arg := range args {
		if expressionHasAuth(arg) {
			state.auth = true
		}
		if call, ok := arg.(*ast.CallExpr); ok {
			name, ok := callName(call)
			if !ok {
				continue
			}
			switch name {
			case "RequirePermission", "RequireAllPermissions":
				state.mode = "all"
				state.codes = stringArguments(call)
			case "RequireAnyPermission":
				state.mode = "any"
				state.codes = stringArguments(call)
			}
		}
	}
	return state
}

func callName(call *ast.CallExpr) (string, bool) {
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return "", false
	}
	return selector.Sel.Name, true
}

func isRouteMethod(name string) bool {
	switch name {
	case "Get", "Post", "Put", "Patch", "Delete", "Handle":
		return true
	default:
		return false
	}
}

func stringArgument(call *ast.CallExpr, index int) (string, bool) {
	if index >= len(call.Args) {
		return "", false
	}
	literal, ok := call.Args[index].(*ast.BasicLit)
	if !ok || literal.Kind != token.STRING {
		return "", false
	}
	value, err := strconv.Unquote(literal.Value)
	return value, err == nil
}

func stringArguments(call *ast.CallExpr) []string {
	values := make([]string, 0, len(call.Args))
	for index := range call.Args {
		if value, ok := stringArgument(call, index); ok {
			values = append(values, value)
		}
	}
	return values
}

func expressionHasAuth(expression ast.Expr) bool {
	found := false
	ast.Inspect(expression, func(node ast.Node) bool {
		selector, ok := node.(*ast.SelectorExpr)
		if ok && selector.Sel.Name == "auth" {
			found = true
			return false
		}
		return true
	})
	return found
}

func joinRoutePath(prefix, path string) string {
	if path == "/" {
		return strings.TrimRight(prefix, "/")
	}
	return strings.TrimRight(prefix, "/") + "/" + strings.TrimLeft(path, "/")
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}

func readSeedPermissions(t *testing.T, root string) map[string]struct{} {
	t.Helper()
	source, err := os.ReadFile(filepath.Join(root, "db", "migrations", "000007_seed_authorization.sql")) // #nosec G304 -- path is rooted at the repository root.
	require.NoError(t, err)
	pattern := regexp.MustCompile(`\(\s*'[^']+'\s*,\s*'([^']+:[^']+)'\s*,`)
	result := make(map[string]struct{})
	for _, match := range pattern.FindAllStringSubmatch(string(source), -1) {
		result[match[1]] = struct{}{}
	}
	return result
}

func readCatalogPermissions(t *testing.T, root string) map[string]struct{} {
	t.Helper()
	source, err := os.ReadFile(filepath.Join(root, "docs", "api", "permission-catalog.md")) // #nosec G304 -- path is rooted at the repository root.
	require.NoError(t, err)
	text := string(source)
	start := strings.Index(text, "## Canonical Codes")
	end := strings.Index(text, "## Legacy Vue Mapping")
	require.NotEqual(t, -1, start)
	require.Greater(t, end, start)
	pattern := regexp.MustCompile("\\|\\s*`([a-z]+:[a-z]+)`\\s*\\|")
	result := make(map[string]struct{})
	for _, match := range pattern.FindAllStringSubmatch(text[start:end], -1) {
		result[match[1]] = struct{}{}
	}
	return result
}

func loadOpenAPIContract(t *testing.T) *openapi3.T {
	t.Helper()
	loader := openapi3.NewLoader()
	document, err := loader.LoadFromData(api.OpenAPISpec)
	require.NoError(t, err)
	require.NoError(t, document.Validate(t.Context()))
	return document
}
