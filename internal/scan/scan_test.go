package scan

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yehia2amer/BuildBloatBuster/internal/config"
)

// setupTestDir creates a temporary directory structure for testing.
func setupTestDir(t *testing.T) (string, func()) {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "BuildBloatBuster-test-*")
	require.NoError(t, err)

	// Create test structure
	// project1
	// |- node_modules
	// |- src
	// |- .git
	// |- deep
	//    |- nested
	//       |- target
	// project2
	// |- vendor
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "project1", "node_modules"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "project1", "src"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "project1", ".git"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "project1", "deep", "nested", "target"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "project2", "vendor"), 0755))

	// Create a file to give a directory size
	_, err = os.Create(filepath.Join(tmpDir, "project1", "node_modules", "file.tmp"))
	require.NoError(t, err)

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

func TestScanner_ScanPaths(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	t.Run("finds standard targets", func(t *testing.T) {
		cfg := config.GetDefaults()
		cfg.ScanPaths = []string{tmpDir}
		cfg.ExcludePaths = []string{} // Isolate test from global excludes
		scanner := NewScanner(cfg)

		candidates, err := scanner.ScanPaths()
		require.NoError(t, err)

		// Expect to find node_modules, target, and vendor
		assert.Len(t, candidates, 3)

		foundPaths := make(map[string]bool)
		for _, c := range candidates {
			foundPaths[filepath.Base(c.Path)] = true
		}
		assert.True(t, foundPaths["node_modules"])
		assert.True(t, foundPaths["target"])
		assert.True(t, foundPaths["vendor"])
	})

	t.Run("respects max depth", func(t *testing.T) {
		cfg := config.GetDefaults()
		cfg.ScanPaths = []string{tmpDir}
		cfg.MaxDepth = 2 // Should find node_modules and vendor, but not target
		cfg.ExcludePaths = []string{}
		scanner := NewScanner(cfg)

		candidates, err := scanner.ScanPaths()
		require.NoError(t, err)

		assert.Len(t, candidates, 2)
		foundPaths := make(map[string]bool)
		for _, c := range candidates {
			foundPaths[filepath.Base(c.Path)] = true
		}
		assert.True(t, foundPaths["node_modules"])
		assert.True(t, foundPaths["vendor"])
		assert.False(t, foundPaths["target"])
	})

	t.Run("respects exclude names", func(t *testing.T) {
		cfg := config.GetDefaults()
		cfg.ScanPaths = []string{tmpDir}
		cfg.ExcludeNames = append(cfg.ExcludeNames, "vendor")
		cfg.ExcludePaths = []string{}
		scanner := NewScanner(cfg)

		candidates, err := scanner.ScanPaths()
		require.NoError(t, err)

		assert.Len(t, candidates, 2) // node_modules and target
		foundPaths := make(map[string]bool)
		for _, c := range candidates {
			foundPaths[filepath.Base(c.Path)] = true
		}
		assert.True(t, foundPaths["node_modules"])
		assert.True(t, foundPaths["target"])
		assert.False(t, foundPaths["vendor"])
	})

	t.Run("respects exclude paths", func(t *testing.T) {
		cfg := config.GetDefaults()
		cfg.ScanPaths = []string{tmpDir}
		// Start with a clean slate for excludes to avoid side-effects from defaults
		cfg.ExcludePaths = []string{filepath.Join(tmpDir, "project1")}
		scanner := NewScanner(cfg)

		candidates, err := scanner.ScanPaths()
		require.NoError(t, err)

		assert.Len(t, candidates, 1) // only vendor in project2
		assert.Equal(t, "vendor", filepath.Base(candidates[0].Path))
	})

	t.Run("does not find excluded by default", func(t *testing.T) {
		cfg := config.GetDefaults()
		cfg.ScanPaths = []string{tmpDir}
		cfg.ExcludePaths = []string{}
		scanner := NewScanner(cfg)

		candidates, err := scanner.ScanPaths()
		require.NoError(t, err)

		foundSrc := false
		foundGit := false
		for _, c := range candidates {
			if filepath.Base(c.Path) == "src" {
				foundSrc = true
			}
			if filepath.Base(c.Path) == ".git" {
				foundGit = true
			}
		}
		assert.False(t, foundSrc, "should not find 'src' because it's a default exclude name")
		assert.False(t, foundGit, "should not find '.git' because it's a VCS folder")
	})
}
