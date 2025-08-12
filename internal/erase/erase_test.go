package erase

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/user/BuildBloatBuster/internal/config"
	"github.com/user/BuildBloatBuster/internal/scan"
)

func setupEraseTest(t *testing.T) (string, string, func()) {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "erase-test-*")
	require.NoError(t, err)

	quarantineDir := filepath.Join(tmpDir, "quarantine")
	require.NoError(t, os.MkdirAll(quarantineDir, 0755))

	// Create a dummy directory to be quarantined
	dummyProject := filepath.Join(tmpDir, "my-project")
	dummyTarget := filepath.Join(dummyProject, "node_modules")
	require.NoError(t, os.MkdirAll(dummyTarget, 0755))
	_, err = os.Create(filepath.Join(dummyTarget, "some-file.js"))
	require.NoError(t, err)

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return dummyTarget, quarantineDir, cleanup
}

func TestEraser_Quarantine(t *testing.T) {
	dummyPath, quarantineDir, cleanup := setupEraseTest(t)
	defer cleanup()

	cfg := config.GetDefaults()
	cfg.Delete.QuarantineDir = quarantineDir
	cfg.Delete.Mode = "quarantine"

	eraser := NewEraser(cfg)

	candidates := []scan.Candidate{
		{Path: dummyPath, SizeBytes: 1024, Reason: "test"},
	}

	err := eraser.EraseCandidates(candidates)
	require.NoError(t, err)

	// 1. Check that original directory is gone
	_, err = os.Stat(dummyPath)
	assert.True(t, os.IsNotExist(err), "original directory should have been moved")

	// 2. Check that something exists in quarantine
	quarantineItems, err := os.ReadDir(quarantineDir)
	require.NoError(t, err)
	// Expecting one directory and one metadata file
	assert.Len(t, quarantineItems, 2)

	// 3. Find the metadata file and verify its content
	var metaPath string
	var quarantinedDir string
	for _, item := range quarantineItems {
		if filepath.Ext(item.Name()) == ".json" {
			metaPath = filepath.Join(quarantineDir, item.Name())
		} else {
			quarantinedDir = filepath.Join(quarantineDir, item.Name())
		}
	}
	require.NotEmpty(t, metaPath, "metadata file should exist")
	require.NotEmpty(t, quarantinedDir, "quarantined directory should exist")

	// 4. Verify metadata content
	metaData, err := os.ReadFile(metaPath)
	require.NoError(t, err)

	var meta Metadata
	err = json.Unmarshal(metaData, &meta)
	require.NoError(t, err)

	assert.Equal(t, dummyPath, meta.OriginalPath)
	assert.Equal(t, quarantinedDir, meta.QuarantinePath)
	assert.NotZero(t, meta.Timestamp)
	assert.Equal(t, int64(1024), meta.SizeBytes)
}