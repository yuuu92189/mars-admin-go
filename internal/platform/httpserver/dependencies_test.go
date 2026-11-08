package httpserver

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNoKnownCrossModuleSQLCImports(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
	forbidden := map[string][]string{
		filepath.Join(root, "internal", "modules", "identity"): {
			"internal/modules/access/adapter/postgres/sqlc",
		},
		filepath.Join(root, "internal", "modules", "notification"): {
			"internal/modules/messaging/adapter/postgres/sqlc",
		},
	}

	for directory, imports := range forbidden {
		err := filepath.Walk(directory, func(path string, info os.FileInfo, walkErr error) error {
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
			for _, forbiddenImport := range imports {
				require.NotContains(t, string(source), forbiddenImport, "%s imports %s", path, forbiddenImport)
			}
			return nil
		})
		require.NoError(t, err)
	}
}
