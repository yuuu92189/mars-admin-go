package httpserver

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEveryRoutePermissionIsSeeded(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
	seed, err := os.ReadFile(filepath.Join(root, "db", "migrations", "000007_seed_authorization.sql"))
	require.NoError(t, err)
	seeded := permissionCodes(string(seed))
	require.NotEmpty(t, seeded)

	used := make(map[string]struct{})
	permissionPattern := regexp.MustCompile(`RequirePermission\("([^"]+)"\)`)
	err = filepath.Walk(filepath.Join(root, "internal", "modules"), func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		source, readErr := os.ReadFile(path) // #nosec G304,G122 -- path is rooted at the repository internal directory.
		if readErr != nil {
			return readErr
		}
		for _, match := range permissionPattern.FindAllStringSubmatch(string(source), -1) {
			used[match[1]] = struct{}{}
		}
		return nil
	})
	require.NoError(t, err)
	require.NotEmpty(t, used)

	var missing []string
	for code := range used {
		if _, exists := seeded[code]; !exists {
			missing = append(missing, code)
		}
	}
	contract, err := os.ReadFile(filepath.Join(root, "api", "openapi", "v1", "openapi.yaml"))
	require.NoError(t, err)
	contractPermissionPattern := regexp.MustCompile(`x-required-permission:\s*([^\s]+)`)
	for _, match := range contractPermissionPattern.FindAllStringSubmatch(string(contract), -1) {
		if _, exists := seeded[match[1]]; !exists {
			missing = append(missing, "openapi:"+match[1])
		}
	}
	sort.Strings(missing)
	require.Empty(t, missing, "route permissions missing from seed: %v", missing)
}

func permissionCodes(sql string) map[string]struct{} {
	pattern := regexp.MustCompile(`\(\s*'[^']+'\s*,\s*'([^']+:[^']+)'\s*,`)
	result := make(map[string]struct{})
	for _, match := range pattern.FindAllStringSubmatch(sql, -1) {
		result[match[1]] = struct{}{}
	}
	return result
}
