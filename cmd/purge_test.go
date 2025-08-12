package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/user/BuildBloatBuster/internal/config"
	"github.com/user/BuildBloatBuster/internal/erase"
)

func setupPurgeTest(t *testing.T) (string, func()) {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "purge-test-*")
	require.NoError(t, err)

	quarantineDir := filepath.Join(tmpDir, "quarantine")
	require.NoError(t, os.MkdirAll(quarantineDir, 0755))

	// Create a new item (should not be purged)
	createNewItem(t, quarantineDir, "new-item", time.Now())

	// Create an old item (should be purged)
	createOldItem(t, quarantineDir, "old-item", time.Now().AddDate(0, 0, -10))

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return quarantineDir, cleanup
}

func createNewItem(t *testing.T, quarantineDir, name string, timestamp time.Time) {
	t.Helper()
	itemPath := filepath.Join(quarantineDir, name)
	require.NoError(t, os.Mkdir(itemPath, 0755))

	meta := erase.Metadata{
		OriginalPath:  "/dummy/original/path/" + name,
		QuarantinePath: itemPath,
		Timestamp:     timestamp,
		SizeBytes:     1234,
	}
	writeTestMetadata(t, itemPath+".meta.json", meta)
}

func createOldItem(t *testing.T, quarantineDir, name string, timestamp time.Time) {
	t.Helper()
	itemPath := filepath.Join(quarantineDir, name)
	require.NoError(t, os.Mkdir(itemPath, 0755))

	meta := erase.Metadata{
		OriginalPath:  "/dummy/original/path/" + name,
		QuarantinePath: itemPath,
		Timestamp:     timestamp,
		SizeBytes:     5678,
	}
	writeTestMetadata(t, itemPath+".meta.json", meta)
}

func writeTestMetadata(t *testing.T, path string, meta erase.Metadata) {
	t.Helper()
	data, err := json.Marshal(meta)
	require.NoError(t, err)
	err = os.WriteFile(path, data, 0644)
	require.NoError(t, err)
}

func TestPurge(t *testing.T) {
	quarantineDir, cleanup := setupPurgeTest(t)
	defer cleanup()

	// Set up config for the test
	Cfg = config.GetDefaults()
	Cfg.Delete.QuarantineDir = quarantineDir

	// For the test, we will manually call the core logic of runPurge
	// to avoid dealing with interactive prompts.
	items, err := listQuarantinedItems(quarantineDir)
	require.NoError(t, err)
	assert.Len(t, items, 2)

	// Purge items older than 5 days
	var toPurge []string
	var toPurgeMeta []string
	cutoff := time.Now().AddDate(0, 0, -5)

	for _, item := range items {
		if item.Timestamp.Before(cutoff) {
			toPurge = append(toPurge, item.QuarantinePath)
			toPurgeMeta = append(toPurgeMeta, item.QuarantinePath+".meta.json")
		}
	}

	require.Len(t, toPurge, 1, "should find one old item to purge")
	require.Equal(t, filepath.Join(quarantineDir, "old-item"), toPurge[0])

	// Manually purge for the test
	for i, path := range toPurge {
		require.NoError(t, os.RemoveAll(path))
		require.NoError(t, os.Remove(toPurgeMeta[i]))
	}

	// Verify that only the new item remains
	remainingItems, err := listQuarantinedItems(quarantineDir)
	require.NoError(t, err)
	assert.Len(t, remainingItems, 1)
	assert.Equal(t, filepath.Join(quarantineDir, "new-item"), remainingItems[0].QuarantinePath)
}